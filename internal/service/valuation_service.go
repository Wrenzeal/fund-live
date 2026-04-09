// Package service contains the core business logic implementations.
package service

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/trading"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

// ValuationServiceImpl implements the ValuationService interface.
type ValuationServiceImpl struct {
	fundRepo           domain.FundRepository
	quoteProvider      domain.QuoteProvider
	quoteProviders     map[domain.QuoteSource]domain.QuoteProvider
	defaultQuoteSource domain.QuoteSource
	cache              domain.CacheRepository
	dataLoader         *FundDataLoader
	profileStore       *ValuationProfileStore

	// FundResolver handles feeder fund to ETF resolution
	fundResolver *FundResolver

	// Time series storage with date-indexed keys: "fundID:YYYY-MM-DD"
	// This allows fallback to previous trading day data
	timeSeriesMu sync.RWMutex
	timeSeries   map[string][]domain.TimeSeriesPoint // key format: "fundID:2006-01-02"

	// Background collector control
	stopCollector chan struct{}
	collectorOnce sync.Once

	// Fund IDs to track (set by StartBackgroundCollector)
	trackedFunds         []trackedFundTarget
	trackedFundsMu       sync.RWMutex
	trackedFundTTL       time.Duration
	collectorConcurrency int
	now                  func() time.Time
}

type trackedFundTarget struct {
	FundID        string
	Source        domain.QuoteSource
	LastTrackedAt time.Time
}

// NewValuationService creates a new ValuationService instance.
func NewValuationService(
	fundRepo domain.FundRepository,
	quoteProvider domain.QuoteProvider,
	cache domain.CacheRepository,
) *ValuationServiceImpl {
	return &ValuationServiceImpl{
		fundRepo:             fundRepo,
		quoteProvider:        quoteProvider,
		quoteProviders:       map[domain.QuoteSource]domain.QuoteProvider{domain.QuoteSourceSina: quoteProvider},
		defaultQuoteSource:   domain.QuoteSourceSina,
		cache:                cache,
		dataLoader:           NewFundDataLoader(fundRepo),
		timeSeries:           make(map[string][]domain.TimeSeriesPoint),
		stopCollector:        make(chan struct{}),
		trackedFunds:         []trackedFundTarget{},
		trackedFundTTL:       6 * time.Hour,
		collectorConcurrency: 4,
		now:                  time.Now,
	}
}

// SetValuationProfileStore sets the valuation profile store for non-stock funds.
func (s *ValuationServiceImpl) SetValuationProfileStore(store *ValuationProfileStore) {
	s.profileStore = store
}

// SetFundDataLoader overrides the transient fund data loader used by read paths.
func (s *ValuationServiceImpl) SetFundDataLoader(loader *FundDataLoader) {
	if loader != nil {
		s.dataLoader = loader
	}
}

// SetQuoteProvider registers a quote provider for a specific source.
func (s *ValuationServiceImpl) SetQuoteProvider(source domain.QuoteSource, provider domain.QuoteProvider) {
	if s == nil || provider == nil {
		return
	}

	source = domain.ResolveQuoteSource(source, s.defaultQuoteSource)
	if s.quoteProviders == nil {
		s.quoteProviders = make(map[domain.QuoteSource]domain.QuoteProvider)
	}
	s.quoteProviders[source] = provider
	if source == s.defaultQuoteSource || s.quoteProvider == nil {
		s.quoteProvider = provider
	}
}

// SetDefaultQuoteSource overrides the fallback source used when the request has no user-specific preference.
func (s *ValuationServiceImpl) SetDefaultQuoteSource(source domain.QuoteSource) {
	if s == nil {
		return
	}

	s.defaultQuoteSource = domain.ResolveQuoteSource(source, domain.QuoteSourceSina)
	if provider, ok := s.quoteProviders[s.defaultQuoteSource]; ok {
		s.quoteProvider = provider
	}
}

// SetFundResolver sets the fund resolver for handling feeder fund resolution.
// This enables transparent access to target ETF holdings for feeder funds.
func (s *ValuationServiceImpl) SetFundResolver(resolver *FundResolver) {
	s.fundResolver = resolver
}

// StartBackgroundCollector starts a background goroutine that automatically
// collects time series data for tracked funds during trading hours.
// This ensures complete data from market open (09:30) regardless of frontend activity.
// If fundIDs is empty, the collector starts idle and waits for explicit tracking updates.
func (s *ValuationServiceImpl) StartBackgroundCollector(ctx context.Context, fundIDs []string, interval time.Duration) {
	s.collectorOnce.Do(func() {
		s.TrackFundIDs(fundIDs...)

		go s.runBackgroundCollector(ctx, interval)
		trackedCount := len(s.snapshotTrackedFunds())
		if trackedCount == 0 {
			log.Printf("🔄 Background data collector started idle (interval: %s, tracked funds: 0)", interval)
			return
		}
		log.Printf("🔄 Background data collector started (interval: %s, tracked targets: %d)", interval, trackedCount)
	})
}

// runBackgroundCollector is the main loop for the background data collector.
func (s *ValuationServiceImpl) runBackgroundCollector(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Do an initial collection immediately
	s.collectDataForAllFunds(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("⏹️ Background collector stopped (context cancelled)")
			return
		case <-s.stopCollector:
			log.Println("⏹️ Background collector stopped")
			return
		case <-ticker.C:
			// Only collect during trading hours
			if trading.IsTradingHours(time.Now()) {
				s.collectDataForAllFunds(ctx)
			}
		}
	}
}

// collectDataForAllFunds fetches estimates for all tracked funds.
func (s *ValuationServiceImpl) collectDataForAllFunds(ctx context.Context) {
	funds := s.snapshotTrackedFunds()

	if len(funds) == 0 {
		return
	}

	group, groupCtx := errgroup.WithContext(ctx)
	if s.collectorConcurrency > 0 {
		group.SetLimit(s.collectorConcurrency)
	}

	for _, target := range funds {
		target := target
		group.Go(func() error {
			targetCtx := domain.WithQuoteSource(groupCtx, target.Source)
			_, err := s.CalculateEstimate(targetCtx, target.FundID)
			if err != nil {
				// Log error but continue with other funds
				log.Printf("⚠️ Background collector: failed to collect data for %s[%s]: %v", target.FundID, target.Source, err)
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		log.Printf("⚠️ Background collector group wait failed: %v", err)
	}
}

// CalculateEstimate computes the real-time fund valuation estimate.
// This is the core algorithm:
// 1. Fetch top 10 holdings for the fund (with feeder fund resolution)
// 2. Concurrently fetch real-time prices using errgroup
// 3. Calculate weighted average change percent
// 4. Return the estimate with detailed breakdown
func (s *ValuationServiceImpl) CalculateEstimate(ctx context.Context, fundID string) (*domain.FundEstimate, error) {
	// Step 1: Get fund information
	fund, err := s.fundRepo.GetFundByID(ctx, fundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fund: %w", err)
	}
	if fund == nil {
		return nil, fmt.Errorf("fund not found: %s", fundID)
	}

	// Step 2: Get holdings (with feeder fund resolution)
	var holdings []domain.StockHolding
	var holdingsSource string = fundID // Track which fund's holdings we're using

	// First try direct holdings
	holdings, err = s.fundRepo.GetFundHoldings(ctx, fundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get holdings: %w", err)
	}

	fund, holdings, warmupScheduled := useCachedFundDataOrScheduleWarmup(s.dataLoader, fundID, fund, holdings)

	// If no holdings and we have a fund resolver, try feeder fund resolution
	if len(holdings) == 0 && s.fundResolver != nil {
		holdings, holdingsSource, err = s.fundResolver.GetHoldingsWithFallback(ctx, fundID, fund.Name)
		if err != nil {
			log.Printf("⚠️ Feeder fund resolution failed for %s: %v", fundID, err)
			// Continue with empty holdings - will fail below
		}
	}

	if len(holdings) == 0 {
		// 特殊情况：如果是联接基金，且已解析出目标 ETF，但该 ETF 无持仓（如黄金ETF、QDII ETF）
		// 此时直接使用目标 ETF 的实时行情作为预估值
		quoteSource, _ := s.resolveQuoteProvider(ctx)
		if holdingsSource != fundID && holdingsSource != "" {
			log.Printf("📊 Fund %s has no holdings, but tracks ETF %s. Using ETF quote directly.", fundID, holdingsSource)
			estimate, targetErr := s.calculateEstimateFromTargetETF(ctx, fund, holdingsSource, quoteSource)
			if targetErr != nil {
				return nil, targetErr
			}
			s.recordTimeSeriesPoint(fundID, quoteSource, estimate)
			return estimate, nil
		}

		if estimate, handled, profileErr := s.calculateEstimateFromValuationProfile(ctx, fund); handled {
			if profileErr != nil {
				return nil, profileErr
			}
			s.recordTimeSeriesPoint(fundID, domain.QuoteSourceFromContext(ctx), estimate)
			return estimate, nil
		}

		if IsFeederFund(fund.Name) {
			if warmupScheduled {
				return nil, ErrFundDataWarmupInProgress
			}
			return nil, fmt.Errorf("no holdings found for feeder fund %s (target ETF resolution may have failed)", fundID)
		}
		if warmupScheduled {
			return nil, ErrFundDataWarmupInProgress
		}
		return nil, fmt.Errorf("no holdings found for fund: %s", fundID)
	}

	if fund.NetAssetVal.IsZero() && warmupScheduled {
		return nil, ErrFundDataWarmupInProgress
	}

	// Log if using fallback holdings
	if holdingsSource != fundID {
		log.Printf("📊 Using holdings from target ETF %s for feeder fund %s", holdingsSource, fundID)
	}

	// Step 3: Get stock codes for quote fetching
	stockCodes := make([]string, len(holdings))
	for i, h := range holdings {
		stockCodes[i] = h.StockCode
	}

	// Step 4: Fetch real-time quotes (with caching)
	quotes, err := s.fetchQuotesWithCache(ctx, stockCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quotes: %w", err)
	}

	quoteSource, _ := s.resolveQuoteProvider(ctx)

	// Step 5: Calculate the estimate using precise decimal arithmetic
	estimate := s.calculateWeightedEstimate(fund, holdings, quotes, quoteSource)

	// Step 6: Store time series point for intraday chart
	s.recordTimeSeriesPoint(fundID, quoteSource, estimate)

	return estimate, nil
}

// fetchQuotesWithCache fetches quotes with caching support.
func (s *ValuationServiceImpl) fetchQuotesWithCache(ctx context.Context, stockCodes []string) (map[string]domain.StockQuote, error) {
	const cacheTTL = 60 // 60 seconds
	source, provider := s.resolveQuoteProvider(ctx)
	if provider == nil {
		return nil, fmt.Errorf("quote provider not configured for source %s", source)
	}
	cacheKeyPrefix := fmt.Sprintf("quote:%s:", source)

	result := make(map[string]domain.StockQuote)
	var uncachedCodes []string

	// Check cache first
	for _, code := range stockCodes {
		if cached, found := s.cache.Get(ctx, cacheKeyPrefix+code); found {
			if quote, ok := cached.(domain.StockQuote); ok {
				result[code] = quote
				continue
			}
		}
		uncachedCodes = append(uncachedCodes, code)
	}

	// If all quotes are cached, return early
	if len(uncachedCodes) == 0 {
		return result, nil
	}

	// Fetch uncached quotes
	freshQuotes, err := provider.GetRealTimeQuotes(ctx, uncachedCodes)
	if err != nil {
		return nil, err
	}

	// Cache the fresh quotes
	for code, quote := range freshQuotes {
		result[code] = quote
		_ = s.cache.Set(ctx, cacheKeyPrefix+code, quote, cacheTTL)
	}

	return result, nil
}

// calculateEstimateFromTargetETF estimates fund value using the target ETF's direct quote.
// This is used for feeder funds tracking ETFs that don't have stock holdings (e.g. Gold ETFs).
func (s *ValuationServiceImpl) calculateEstimateFromTargetETF(ctx context.Context, fund *domain.Fund, targetCode string, source domain.QuoteSource) (*domain.FundEstimate, error) {
	// Fetch quote for the target ETF
	quotes, err := s.fetchQuotesWithCache(ctx, []string{targetCode})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quote for target ETF %s: %w", targetCode, err)
	}

	quote, ok := quotes[targetCode]
	if !ok || quote.CurrentPrice.IsZero() {
		return nil, fmt.Errorf("no quote data for target ETF %s", targetCode)
	}

	// Calculate estimated NAV based on ETF change
	nav := fund.NetAssetVal
	estimatedNav := nav
	changePercent := quote.ChangePercent

	if !changePercent.IsZero() {
		changeFactor := decimal.NewFromFloat(1).Add(changePercent.Div(decimal.NewFromFloat(100)))
		estimatedNav = nav.Mul(changeFactor)
	}

	// Create a virtual holding detail for the ETF
	details := []domain.HoldingDetail{
		{
			StockCode:    targetCode,
			StockName:    quote.StockName,
			HoldingRatio: decimal.NewFromFloat(100.00), // Assume 100% tracking
			StockChange:  changePercent,
			Contribution: changePercent, // For 100% holding, contribution equals change
			CurrentPrice: quote.CurrentPrice,
			PrevClose:    quote.PrevClose,
		},
	}

	now := time.Now()
	return &domain.FundEstimate{
		FundID:         fund.ID,
		FundName:       fund.Name,
		EstimateNav:    estimatedNav,
		PrevNav:        nav,
		ChangePercent:  changePercent,
		ChangeAmount:   estimatedNav.Sub(nav),
		CalculatedAt:   now,
		HoldingDetails: details,
		TotalHoldRatio: decimal.NewFromFloat(100.00),
		DataSource:     fmt.Sprintf("追踪目标ETF: %s", quote.StockName),
	}, nil
}

// calculateWeightedEstimate performs the core weighted average calculation.
// Formula: EstimatedChange = Σ(StockChange × HoldingRatio) / Σ(HoldingRatio)
// All calculations use decimal.Decimal for precision.
func (s *ValuationServiceImpl) calculateWeightedEstimate(
	fund *domain.Fund,
	holdings []domain.StockHolding,
	quotes map[string]domain.StockQuote,
	source domain.QuoteSource,
) *domain.FundEstimate {
	return buildEstimateSnapshotFromQuotes(fund, holdings, quotes, source, time.Now())
}

// makeTimeSeriesKey creates a composite key for time series storage.
// Format: "fundID:YYYY-MM-DD"
func makeTimeSeriesKey(source domain.QuoteSource, fundID string, date time.Time) string {
	return fmt.Sprintf("%s:%s:%s", source, fundID, date.Format("2006-01-02"))
}

// recordTimeSeriesPoint stores an aligned in-memory time series point for intraday charting.
// Estimate requests can be much more frequent than the 5-minute chart granularity, so
// we collapse multiple updates within the same bucket instead of appending raw points.
func (s *ValuationServiceImpl) recordTimeSeriesPoint(fundID string, source domain.QuoteSource, estimate *domain.FundEstimate) {
	s.TrackFundsForSource(source, fundID)

	alignedTimestamp := alignTimeSeriesTimestamp(estimate.CalculatedAt)
	point := domain.TimeSeriesPoint{
		Timestamp:     alignedTimestamp,
		ChangePercent: estimate.ChangePercent,
		EstimateNav:   estimate.EstimateNav,
	}

	s.timeSeriesMu.Lock()
	key := makeTimeSeriesKey(source, fundID, alignedTimestamp)
	if s.timeSeries[key] == nil {
		s.timeSeries[key] = make([]domain.TimeSeriesPoint, 0, 72) // ~6 hours of 5-minute buckets
	}

	points := s.timeSeries[key]
	if len(points) > 0 && points[len(points)-1].Timestamp.Equal(alignedTimestamp) {
		points[len(points)-1] = point
		s.timeSeries[key] = points
	} else {
		s.timeSeries[key] = append(points, point)
	}
	s.cleanupOldDates()
	s.timeSeriesMu.Unlock()
}

// cleanupOldDates removes time series data older than 7 days.
func (s *ValuationServiceImpl) cleanupOldDates() {
	now := time.Now()
	cutoff := now.AddDate(0, 0, -7)

	for key := range s.timeSeries {
		// Parse date from key (format: fundID:2006-01-02)
		parts := key[len(key)-10:] // Last 10 chars = date
		if dateVal, err := time.Parse("2006-01-02", parts); err == nil {
			if dateVal.Before(cutoff) {
				delete(s.timeSeries, key)
			}
		}
	}
}

// GetIntradayTimeSeries returns the intraday time series for a fund.
// If persisted points are missing or start too late, it backfills the requested session
// from external intraday kline data.
func (s *ValuationServiceImpl) GetIntradayTimeSeries(ctx context.Context, fundID string) ([]domain.TimeSeriesPoint, error) {
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	targetDate := s.preferredTimeSeriesDate(now)
	quoteSource, _ := s.resolveQuoteProvider(ctx)
	targetKey := makeTimeSeriesKey(quoteSource, fundID, targetDate)

	if points := s.getTimeSeriesForKey(targetKey); len(points) > 0 && !needsTimeSeriesBackfill(points, targetDate, now) {
		return s.finalizeIntradayTimeSeries(ctx, fundID, targetDate, now, points), nil
	}

	if points, err := s.fundRepo.GetTimeSeriesByDate(ctx, fundID, targetDate); err == nil && len(points) > 0 {
		if !needsTimeSeriesBackfill(points, targetDate, now) {
			s.cacheTimeSeriesPoints(fundID, quoteSource, points)
			return s.finalizeIntradayTimeSeries(ctx, fundID, targetDate, now, points), nil
		}
	}

	if points, err := s.backfillTimeSeries(ctx, fundID, targetDate); err == nil && len(points) > 0 {
		s.cacheTimeSeriesPoints(fundID, quoteSource, points)
		if err := s.fundRepo.ReplaceTimeSeriesByDate(ctx, fundID, targetDate, points); err != nil {
			log.Printf("⚠️ Failed to persist backfilled time series for %s: %v", fundID, err)
		}
		return s.finalizeIntradayTimeSeries(ctx, fundID, targetDate, now, points), nil
	}

	if !shouldAllowPreviousTradingDayTimeSeriesFallback(now) {
		return []domain.TimeSeriesPoint{}, nil
	}

	// Fallback: search backwards for up to 7 days to find valid data
	for i := 1; i <= 7; i++ {
		prevDate := targetDate.AddDate(0, 0, -i)
		prevKey := makeTimeSeriesKey(quoteSource, fundID, prevDate)
		if points := s.getTimeSeriesForKey(prevKey); len(points) > 0 {
			return s.finalizeIntradayTimeSeries(ctx, fundID, prevDate, now, points), nil
		}
		if points, err := s.fundRepo.GetTimeSeriesByDate(ctx, fundID, prevDate); err == nil && len(points) > 0 {
			s.cacheTimeSeriesPoints(fundID, quoteSource, points)
			return s.finalizeIntradayTimeSeries(ctx, fundID, prevDate, now, points), nil
		}
	}

	return []domain.TimeSeriesPoint{}, nil
}

func (s *ValuationServiceImpl) finalizeIntradayTimeSeries(ctx context.Context, fundID string, targetDate, now time.Time, points []domain.TimeSeriesPoint) []domain.TimeSeriesPoint {
	points = ensureLunchBreakResumePoint(points, now)
	return s.alignAfterHoursTimeSeriesWithEstimate(ctx, fundID, targetDate, now, points)
}

func (s *ValuationServiceImpl) alignAfterHoursTimeSeriesWithEstimate(ctx context.Context, fundID string, targetDate, now time.Time, points []domain.TimeSeriesPoint) []domain.TimeSeriesPoint {
	if !shouldAlignAfterHoursTimeSeriesWithEstimate(now, targetDate) {
		return points
	}

	estimate, err := s.CalculateEstimate(ctx, fundID)
	if err != nil || estimate == nil {
		return points
	}

	closingPoint := domain.TimeSeriesPoint{
		Timestamp:     time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 15, 0, 0, 0, trading.TradingLocation()),
		ChangePercent: estimate.ChangePercent,
		EstimateNav:   estimate.EstimateNav,
	}

	updated := false
	for i := range points {
		localTs := points[i].Timestamp.In(trading.TradingLocation())
		if localTs.Year() == closingPoint.Timestamp.Year() &&
			localTs.Month() == closingPoint.Timestamp.Month() &&
			localTs.Day() == closingPoint.Timestamp.Day() &&
			localTs.Hour() == closingPoint.Timestamp.Hour() &&
			localTs.Minute() == closingPoint.Timestamp.Minute() {
			points[i].ChangePercent = closingPoint.ChangePercent
			points[i].EstimateNav = closingPoint.EstimateNav
			updated = true
			break
		}
	}

	if !updated {
		points = append(points, closingPoint)
		sort.Slice(points, func(i, j int) bool {
			return points[i].Timestamp.Before(points[j].Timestamp)
		})
	}

	return points
}

func shouldAlignAfterHoursTimeSeriesWithEstimate(now, targetDate time.Time) bool {
	localNow := now.In(trading.TradingLocation())
	if !trading.IsTradingDay(localNow) {
		return false
	}
	if trading.GetCurrentSession(localNow) != trading.SessionAfterHours {
		return false
	}
	return targetDate.In(trading.TradingLocation()).Format("2006-01-02") == localNow.Format("2006-01-02")
}

// cacheTimeSeriesPoints caches time series points in memory for fast access.
func (s *ValuationServiceImpl) cacheTimeSeriesPoints(fundID string, source domain.QuoteSource, points []domain.TimeSeriesPoint) {
	if len(points) == 0 {
		return
	}

	s.timeSeriesMu.Lock()
	defer s.timeSeriesMu.Unlock()

	key := makeTimeSeriesKey(source, fundID, points[0].Timestamp)
	s.timeSeries[key] = points
}

// getTimeSeriesForKey retrieves points for a specific key (thread-safe).
func (s *ValuationServiceImpl) getTimeSeriesForKey(key string) []domain.TimeSeriesPoint {
	s.timeSeriesMu.RLock()
	defer s.timeSeriesMu.RUnlock()

	if points, ok := s.timeSeries[key]; ok && len(points) > 0 {
		// Return a copy to avoid race conditions
		result := make([]domain.TimeSeriesPoint, len(points))
		copy(result, points)
		return result
	}
	return nil
}

func alignTimeSeriesTimestamp(ts time.Time) time.Time {
	local := ts.In(trading.TradingLocation())
	flooredMinute := (local.Minute() / 5) * 5
	return time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), flooredMinute, 0, 0, trading.TradingLocation())
}

// TrackFundIDs adds funds to the background collector tracking set.
func (s *ValuationServiceImpl) TrackFundIDs(fundIDs ...string) {
	s.TrackFundsForSource(s.defaultQuoteSource, fundIDs...)
}

// TrackFundsForSource adds funds for a specific quote source to the background collector tracking set.
func (s *ValuationServiceImpl) TrackFundsForSource(source domain.QuoteSource, fundIDs ...string) {
	if s == nil {
		return
	}

	source = domain.ResolveQuoteSource(source, s.defaultQuoteSource)
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	s.trackedFundsMu.Lock()
	defer s.trackedFundsMu.Unlock()

	seen := make(map[string]int, len(s.trackedFunds))
	activeTargets := s.trackedFunds[:0]
	for _, target := range s.trackedFunds {
		if s.isTrackedFundExpired(target, now) {
			continue
		}
		seen[string(target.Source)+"|"+target.FundID] = len(activeTargets)
		activeTargets = append(activeTargets, target)
	}
	s.trackedFunds = activeTargets

	for _, fundID := range fundIDs {
		fundID = strings.TrimSpace(fundID)
		if fundID == "" {
			continue
		}
		seenKey := string(source) + "|" + fundID
		if idx, ok := seen[seenKey]; ok {
			s.trackedFunds[idx].LastTrackedAt = now
			continue
		}
		seen[seenKey] = len(s.trackedFunds)
		s.trackedFunds = append(s.trackedFunds, trackedFundTarget{
			FundID:        fundID,
			Source:        source,
			LastTrackedAt: now,
		})
	}
}

func (s *ValuationServiceImpl) snapshotTrackedFunds() []trackedFundTarget {
	s.cleanupExpiredTrackedFunds()

	s.trackedFundsMu.RLock()
	defer s.trackedFundsMu.RUnlock()

	if len(s.trackedFunds) == 0 {
		return nil
	}

	now := time.Now()
	if s.now != nil {
		now = s.now()
	}

	funds := make([]trackedFundTarget, 0, len(s.trackedFunds))
	for _, target := range s.trackedFunds {
		if s.isTrackedFundExpired(target, now) {
			continue
		}
		funds = append(funds, target)
	}
	return funds
}

func (s *ValuationServiceImpl) isTrackedFundExpired(target trackedFundTarget, now time.Time) bool {
	if s == nil || s.trackedFundTTL <= 0 {
		return false
	}
	if target.LastTrackedAt.IsZero() {
		return false
	}
	return now.After(target.LastTrackedAt.Add(s.trackedFundTTL))
}

func (s *ValuationServiceImpl) cleanupExpiredTrackedFunds() {
	if s == nil {
		return
	}

	now := time.Now()
	if s.now != nil {
		now = s.now()
	}

	s.trackedFundsMu.Lock()
	defer s.trackedFundsMu.Unlock()

	if len(s.trackedFunds) == 0 {
		return
	}

	activeTargets := s.trackedFunds[:0]
	for _, target := range s.trackedFunds {
		if s.isTrackedFundExpired(target, now) {
			continue
		}
		activeTargets = append(activeTargets, target)
	}
	s.trackedFunds = activeTargets
}

func (s *ValuationServiceImpl) resolveQuoteProvider(ctx context.Context) (domain.QuoteSource, domain.QuoteProvider) {
	source := domain.ResolveQuoteSource(domain.QuoteSourceFromContext(ctx), s.defaultQuoteSource)
	if provider, ok := s.quoteProviders[source]; ok && provider != nil {
		return source, provider
	}
	if s.quoteProvider != nil {
		return source, s.quoteProvider
	}
	return source, nil
}
