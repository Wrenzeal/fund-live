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
	overseasProvider   domain.QuoteProvider
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
	trackedFunds                 []trackedFundTarget
	trackedFundsMu               sync.RWMutex
	managedTargets               []trackedFundTarget
	managedTargetsMu             sync.RWMutex
	trackedFundTTL               time.Duration
	collectorConcurrency         int
	domesticCollectorConcurrency int
	overseasCollectorConcurrency int
	domesticBatchSize            int
	overseasBatchSize            int
	now                          func() time.Time
}

type trackedFundTarget struct {
	FundID          string
	Source          domain.QuoteSource
	LastTrackedAt   time.Time
	LastCollectedAt time.Time
	RefreshInterval time.Duration
	QuoteMode       string
	Persistent      bool
}

// NewValuationService creates a new ValuationService instance.
func NewValuationService(
	fundRepo domain.FundRepository,
	quoteProvider domain.QuoteProvider,
	cache domain.CacheRepository,
) *ValuationServiceImpl {
	return &ValuationServiceImpl{
		fundRepo:                     fundRepo,
		quoteProvider:                quoteProvider,
		quoteProviders:               map[domain.QuoteSource]domain.QuoteProvider{domain.QuoteSourceSina: quoteProvider},
		defaultQuoteSource:           domain.QuoteSourceSina,
		cache:                        cache,
		dataLoader:                   NewFundDataLoader(fundRepo),
		timeSeries:                   make(map[string][]domain.TimeSeriesPoint),
		stopCollector:                make(chan struct{}),
		trackedFunds:                 []trackedFundTarget{},
		managedTargets:               []trackedFundTarget{},
		trackedFundTTL:               6 * time.Hour,
		collectorConcurrency:         4,
		domesticCollectorConcurrency: 8,
		overseasCollectorConcurrency: 3,
		domesticBatchSize:            300,
		overseasBatchSize:            100,
		now:                          time.Now,
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

// SetOverseasQuoteProvider overrides the fixed quote provider used for overseas holdings.
func (s *ValuationServiceImpl) SetOverseasQuoteProvider(provider domain.QuoteProvider) {
	if s == nil || provider == nil {
		return
	}
	s.overseasProvider = provider
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
	s.collectDataForDueFunds(ctx)

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
				s.collectDataForDueFunds(ctx)
			}
		}
	}
}

// collectDataForDueFunds fetches estimates only for due tracked funds and applies batch limits.
func (s *ValuationServiceImpl) collectDataForDueFunds(ctx context.Context) {
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	domesticTargets, overseasTargets := s.snapshotDueTrackedFunds(now)
	if len(domesticTargets) == 0 && len(overseasTargets) == 0 {
		return
	}

	s.markTargetsCollected(append(domesticTargets, overseasTargets...), now)

	group, groupCtx := errgroup.WithContext(ctx)
	if len(domesticTargets) > 0 {
		targets := append([]trackedFundTarget(nil), domesticTargets...)
		group.Go(func() error {
			s.collectTargets(groupCtx, targets, s.domesticCollectorConcurrency)
			return nil
		})
	}
	if len(overseasTargets) > 0 {
		targets := append([]trackedFundTarget(nil), overseasTargets...)
		group.Go(func() error {
			s.collectTargets(groupCtx, targets, s.overseasCollectorConcurrency)
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		log.Printf("⚠️ Background collector group wait failed: %v", err)
	}
}

func (s *ValuationServiceImpl) collectTargets(ctx context.Context, targets []trackedFundTarget, concurrency int) {
	if len(targets) == 0 {
		return
	}

	group, groupCtx := errgroup.WithContext(ctx)
	if concurrency <= 0 {
		concurrency = s.collectorConcurrency
	}
	if concurrency > 0 {
		group.SetLimit(concurrency)
	}

	for _, target := range targets {
		target := target
		group.Go(func() error {
			targetCtx := domain.WithQuoteSource(groupCtx, target.Source)
			_, err := s.CalculateEstimate(targetCtx, target.FundID)
			if err != nil {
				log.Printf("⚠️ Background collector: failed to collect data for %s[%s|%s]: %v", target.FundID, target.Source, target.QuoteMode, err)
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		log.Printf("⚠️ Background collector target group wait failed: %v", err)
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

	trackedETFCode := ""
	if s.fundResolver != nil {
		if display, displayErr := s.fundResolver.ResolveDisplayHoldings(ctx, fundID, fund.Name); displayErr != nil {
			log.Printf("⚠️ Tracked ETF resolution failed for %s: %v", fundID, displayErr)
		} else if targetItem, ok := domain.PrimaryTrackedETF(display); ok {
			trackedETFCode = strings.TrimSpace(targetItem.Code)
		}
	}

	if trackedETFCode != "" {
		quoteSource, _ := s.resolveQuoteProvider(ctx)
		targetEstimate, targetErr := s.calculateEstimateFromTargetETF(ctx, fund, trackedETFCode, quoteSource)
		if targetErr == nil {
			s.recordTimeSeriesPoint(fundID, quoteSource, targetEstimate)
			return targetEstimate, nil
		}

		log.Printf("⚠️ Target ETF quote estimate failed for feeder fund %s via %s: %v; falling back to target ETF holdings", fundID, trackedETFCode, targetErr)
		targetHoldings, holdingsErr := s.fundResolver.loadTargetHoldingsWithFallback(ctx, trackedETFCode)
		if holdingsErr == nil && hasEffectiveHoldings(targetHoldings) {
			holdings = targetHoldings
			holdingsSource = trackedETFCode
		} else {
			if warmupScheduled {
				return nil, ErrFundDataWarmupInProgress
			}
			return nil, targetErr
		}
	}

	// If no effective holdings and we have a fund resolver, try feeder fund resolution
	if trackedETFCode == "" && !hasEffectiveHoldings(holdings) && s.fundResolver != nil {
		holdings, holdingsSource, err = s.fundResolver.GetHoldingsWithFallback(ctx, fundID, fund.Name)
		if err != nil {
			log.Printf("⚠️ Feeder fund resolution failed for %s: %v", fundID, err)
			// Continue with empty holdings - will fail below
		}
	}

	if !hasEffectiveHoldings(holdings) {
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
	if holdingsSource != fundID && trackedETFCode == "" {
		log.Printf("📊 Using holdings from target ETF %s for feeder fund %s", holdingsSource, fundID)

		quoteSource, _ := s.resolveQuoteProvider(ctx)
		targetEstimate, targetErr := s.calculateEstimateFromTargetETF(ctx, fund, holdingsSource, quoteSource)
		if targetErr == nil {
			s.recordTimeSeriesPoint(fundID, quoteSource, targetEstimate)
			return targetEstimate, nil
		}

		log.Printf("⚠️ Target ETF quote estimate failed for feeder fund %s via %s: %v; falling back to holdings-based estimate", fundID, holdingsSource, targetErr)
	}

	// Step 3: Get stock codes for quote fetching
	stockCodes := make([]string, len(holdings))
	for i, h := range holdings {
		stockCodes[i] = h.StockCode
	}

	// Step 4: Fetch real-time quotes (with caching)
	quotes, dataSourceLabel, err := s.fetchQuotesForFund(ctx, fund, holdings, stockCodes)
	if err != nil {
		if shouldUseQDIIHoldingDetailsFallback(fund, holdings) {
			estimate := s.buildQDIIHoldingDetailsEstimate(fund, holdings)
			s.recordTimeSeriesPoint(fundID, domain.QuoteSourceFromContext(ctx), estimate)
			return estimate, nil
		}
		return nil, fmt.Errorf("failed to fetch quotes: %w", err)
	}

	quoteSource, _ := s.resolveQuoteProvider(ctx)

	// Step 5: Calculate the estimate using precise decimal arithmetic
	estimate := s.calculateWeightedEstimate(fund, holdings, quotes, quoteSource)
	if dataSourceLabel != "" {
		estimate.DataSource = dataSourceLabel
	}

	// Step 6: Store time series point for intraday chart
	if !shouldUseFixedOverseasQuoteSource(fund, holdings) {
		s.recordTimeSeriesPoint(fundID, quoteSource, estimate)
	}

	if estimate.TotalHoldRatio.IsZero() && shouldUseQDIIHoldingDetailsFallback(fund, holdings) {
		fallback := s.buildQDIIHoldingDetailsEstimate(fund, holdings)
		s.recordTimeSeriesPoint(fundID, domain.QuoteSourceFromContext(ctx), fallback)
		return fallback, nil
	}

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

func (s *ValuationServiceImpl) fetchOverseasQuotesWithCache(ctx context.Context, stockCodes []string) (map[string]domain.StockQuote, error) {
	const cacheTTL = 60
	if s.overseasProvider == nil {
		return nil, fmt.Errorf("fixed overseas quote provider not configured")
	}
	cacheKeyPrefix := "quote:overseas_fixed:"

	result := make(map[string]domain.StockQuote)
	var uncachedCodes []string
	for _, code := range stockCodes {
		normalizedCode := strings.ToUpper(strings.TrimSpace(code))
		if cached, found := s.cache.Get(ctx, cacheKeyPrefix+normalizedCode); found {
			if quote, ok := cached.(domain.StockQuote); ok {
				result[normalizedCode] = quote
				continue
			}
		}
		uncachedCodes = append(uncachedCodes, normalizedCode)
	}

	if len(uncachedCodes) == 0 {
		return result, nil
	}

	freshQuotes, err := s.overseasProvider.GetRealTimeQuotes(ctx, uncachedCodes)
	if err != nil {
		return nil, err
	}

	for code, quote := range freshQuotes {
		normalizedCode := strings.ToUpper(strings.TrimSpace(code))
		result[normalizedCode] = quote
		_ = s.cache.Set(ctx, cacheKeyPrefix+normalizedCode, quote, cacheTTL)
	}

	return result, nil
}

func (s *ValuationServiceImpl) fetchQuotesForFund(
	ctx context.Context,
	fund *domain.Fund,
	holdings []domain.StockHolding,
	stockCodes []string,
) (map[string]domain.StockQuote, string, error) {
	if shouldUseFixedOverseasQuoteSource(fund, holdings) {
		overseasCodes, domesticCodes := splitHoldingCodesByExchange(holdings)
		quotes := make(map[string]domain.StockQuote)

		if len(overseasCodes) > 0 {
			fixedQuotes, err := s.fetchOverseasQuotesWithCache(ctx, overseasCodes)
			if err != nil {
				return nil, "", err
			}
			for code, quote := range fixedQuotes {
				quotes[code] = quote
			}
		}

		if len(domesticCodes) > 0 {
			domesticQuotes, err := s.fetchQuotesWithCache(ctx, domesticCodes)
			if err != nil {
				return nil, "", err
			}
			for code, quote := range domesticQuotes {
				quotes[code] = quote
			}
		}

		return quotes, "overseas_fixed", nil
	}

	quotes, err := s.fetchQuotesWithCache(ctx, stockCodes)
	if err != nil {
		return nil, "", err
	}
	return quotes, "", nil
}

// calculateEstimateFromTargetETF estimates fund value using the target ETF's direct quote.
// This is used for feeder funds tracking ETFs that don't have stock holdings (e.g. Gold ETFs).
func (s *ValuationServiceImpl) calculateEstimateFromTargetETF(ctx context.Context, fund *domain.Fund, targetCode string, source domain.QuoteSource) (*domain.FundEstimate, error) {
	quote, resolvedSource, err := s.fetchTargetETFQuoteWithFallback(ctx, targetCode, source)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quote for target ETF %s: %w", targetCode, err)
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
		DataSource:     fmt.Sprintf("追踪目标ETF(%s): %s", resolvedSource, quote.StockName),
	}, nil
}

func (s *ValuationServiceImpl) fetchTargetETFQuoteWithFallback(
	ctx context.Context,
	targetCode string,
	preferredSource domain.QuoteSource,
) (domain.StockQuote, domain.QuoteSource, error) {
	sources := s.targetETFFallbackSources(preferredSource)
	var lastErr error

	for _, source := range sources {
		quotes, err := s.fetchQuotesWithCache(domain.WithQuoteSource(ctx, source), []string{targetCode})
		if err != nil {
			lastErr = err
			continue
		}

		quote, ok := quotes[targetCode]
		if !ok || quote.CurrentPrice.IsZero() {
			lastErr = fmt.Errorf("no quote data for target ETF %s via %s", targetCode, source)
			continue
		}
		return quote, source, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no quote data for target ETF %s", targetCode)
	}
	return domain.StockQuote{}, "", lastErr
}

func (s *ValuationServiceImpl) targetETFFallbackSources(preferredSource domain.QuoteSource) []domain.QuoteSource {
	preferredSource = domain.ResolveQuoteSource(preferredSource, s.defaultQuoteSource)
	candidates := []domain.QuoteSource{preferredSource}

	for _, source := range []domain.QuoteSource{domain.QuoteSourceSina, domain.QuoteSourceTencent} {
		if source == preferredSource {
			continue
		}
		if provider, ok := s.quoteProviders[source]; ok && provider != nil {
			candidates = append(candidates, source)
		}
	}

	deduped := make([]domain.QuoteSource, 0, len(candidates))
	seen := make(map[domain.QuoteSource]struct{}, len(candidates))
	for _, source := range candidates {
		if _, exists := seen[source]; exists {
			continue
		}
		if provider, ok := s.quoteProviders[source]; !ok || provider == nil {
			continue
		}
		seen[source] = struct{}{}
		deduped = append(deduped, source)
	}
	return deduped
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

func hasEffectiveHoldings(holdings []domain.StockHolding) bool {
	if len(holdings) == 0 {
		return false
	}

	totalRatio := decimal.Zero
	for _, holding := range holdings {
		if holding.HoldingRatio.GreaterThan(decimal.Zero) {
			totalRatio = totalRatio.Add(holding.HoldingRatio)
		}
	}
	return totalRatio.GreaterThan(decimal.Zero)
}

func shouldUseQDIIHoldingDetailsFallback(fund *domain.Fund, holdings []domain.StockHolding) bool {
	if fund == nil || !hasEffectiveHoldings(holdings) {
		return false
	}
	if !isQDIIFundType(fund) {
		return false
	}

	for _, holding := range holdings {
		if holding.Exchange == domain.ExchangeUS {
			return true
		}
	}
	return false
}

func shouldUseFixedOverseasQuoteSource(fund *domain.Fund, holdings []domain.StockHolding) bool {
	if fund == nil || !isQDIIFundType(fund) {
		return false
	}
	for _, holding := range holdings {
		if holding.Exchange == domain.ExchangeUS {
			return true
		}
	}
	return false
}

func isQDIIFundType(fund *domain.Fund) bool {
	if fund == nil {
		return false
	}

	fundType := strings.ToLower(strings.TrimSpace(fund.Type))
	name := strings.ToLower(strings.TrimSpace(fund.Name))
	return strings.Contains(fundType, "qdii") || strings.Contains(name, "qdii")
}

func (s *ValuationServiceImpl) buildQDIIHoldingDetailsEstimate(fund *domain.Fund, holdings []domain.StockHolding) *domain.FundEstimate {
	details := make([]domain.HoldingDetail, 0, len(holdings))
	totalHoldRatio := decimal.Zero
	for _, holding := range holdings {
		totalHoldRatio = totalHoldRatio.Add(holding.HoldingRatio)
		details = append(details, domain.HoldingDetail{
			StockCode:    holding.StockCode,
			StockName:    holding.StockName,
			HoldingRatio: holding.HoldingRatio,
			StockChange:  decimal.Zero,
			Contribution: decimal.Zero,
			CurrentPrice: decimal.Zero,
			PrevClose:    decimal.Zero,
		})
	}

	return &domain.FundEstimate{
		FundID:         fund.ID,
		FundName:       fund.Name,
		EstimateNav:    fund.NetAssetVal,
		PrevNav:        fund.NetAssetVal,
		ChangePercent:  decimal.Zero,
		ChangeAmount:   decimal.Zero,
		TotalHoldRatio: totalHoldRatio,
		HoldingDetails: details,
		CalculatedAt:   time.Now(),
		DataSource:     "QDII持仓详情（盘中估值暂不支持）",
	}
}

func splitHoldingCodesByExchange(holdings []domain.StockHolding) ([]string, []string) {
	overseasCodes := make([]string, 0, len(holdings))
	domesticCodes := make([]string, 0, len(holdings))
	for _, holding := range holdings {
		code := strings.TrimSpace(holding.StockCode)
		if code == "" {
			continue
		}
		if holding.Exchange == domain.ExchangeUS {
			overseasCodes = append(overseasCodes, strings.ToUpper(code))
			continue
		}
		domesticCodes = append(domesticCodes, code)
	}
	return overseasCodes, domesticCodes
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
			if s.trackedFunds[idx].RefreshInterval <= 0 {
				s.trackedFunds[idx].RefreshInterval = time.Minute
			}
			if strings.TrimSpace(s.trackedFunds[idx].QuoteMode) == "" {
				s.trackedFunds[idx].QuoteMode = EstimateQuoteModeDomestic
			}
			continue
		}
		seen[seenKey] = len(s.trackedFunds)
		s.trackedFunds = append(s.trackedFunds, trackedFundTarget{
			FundID:          fundID,
			Source:          source,
			LastTrackedAt:   now,
			RefreshInterval: time.Minute,
			QuoteMode:       EstimateQuoteModeDomestic,
			Persistent:      false,
		})
	}
}

func (s *ValuationServiceImpl) snapshotTrackedFunds() []trackedFundTarget {
	s.cleanupExpiredTrackedFunds()

	s.trackedFundsMu.RLock()
	ephemeral := append([]trackedFundTarget(nil), s.trackedFunds...)
	s.trackedFundsMu.RUnlock()
	s.managedTargetsMu.RLock()
	managed := append([]trackedFundTarget(nil), s.managedTargets...)
	s.managedTargetsMu.RUnlock()

	if len(ephemeral) == 0 && len(managed) == 0 {
		return nil
	}

	now := time.Now()
	if s.now != nil {
		now = s.now()
	}

	funds := make([]trackedFundTarget, 0, len(ephemeral)+len(managed))
	seen := make(map[string]struct{}, len(ephemeral)+len(managed))
	for _, target := range ephemeral {
		if s.isTrackedFundExpired(target, now) {
			continue
		}
		key := makeTrackedTargetKey(target)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		funds = append(funds, target)
	}
	for _, target := range managed {
		key := makeTrackedTargetKey(target)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
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

func (s *ValuationServiceImpl) SetManagedTargets(targets []trackedFundTarget) {
	if s == nil {
		return
	}

	s.managedTargetsMu.Lock()
	defer s.managedTargetsMu.Unlock()

	existing := make(map[string]trackedFundTarget, len(s.managedTargets))
	for _, target := range s.managedTargets {
		existing[makeTrackedTargetKey(target)] = target
	}

	nextTargets := make([]trackedFundTarget, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		target.FundID = strings.TrimSpace(target.FundID)
		if target.FundID == "" {
			continue
		}
		target.Source = domain.ResolveQuoteSource(target.Source, s.defaultQuoteSource)
		if target.RefreshInterval <= 0 {
			target.RefreshInterval = 3 * time.Minute
		}
		target.QuoteMode = normalizeEstimateQuoteMode(target.QuoteMode)
		target.Persistent = true
		key := makeTrackedTargetKey(target)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if previous, ok := existing[key]; ok {
			target.LastCollectedAt = previous.LastCollectedAt
		}
		nextTargets = append(nextTargets, target)
	}

	s.managedTargets = nextTargets
}

func (s *ValuationServiceImpl) snapshotDueTrackedFunds(now time.Time) ([]trackedFundTarget, []trackedFundTarget) {
	targets := s.snapshotTrackedFunds()
	if len(targets) == 0 {
		return nil, nil
	}

	domesticDue := make([]trackedFundTarget, 0, len(targets))
	overseasDue := make([]trackedFundTarget, 0, len(targets))
	for _, target := range targets {
		interval := target.RefreshInterval
		if interval <= 0 {
			interval = time.Minute
		}
		if !target.LastCollectedAt.IsZero() && now.Before(target.LastCollectedAt.Add(interval)) {
			continue
		}
		if target.QuoteMode == EstimateQuoteModeOverseas || target.QuoteMode == EstimateQuoteModeMixed {
			overseasDue = append(overseasDue, target)
			continue
		}
		domesticDue = append(domesticDue, target)
	}

	sortTrackedTargetsByLastCollected(domesticDue)
	sortTrackedTargetsByLastCollected(overseasDue)

	return limitTrackedTargets(domesticDue, s.domesticBatchSize), limitTrackedTargets(overseasDue, s.overseasBatchSize)
}

func (s *ValuationServiceImpl) markTargetsCollected(targets []trackedFundTarget, collectedAt time.Time) {
	if len(targets) == 0 {
		return
	}

	keys := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		keys[makeTrackedTargetKey(target)] = struct{}{}
	}

	s.trackedFundsMu.Lock()
	for i := range s.trackedFunds {
		if _, ok := keys[makeTrackedTargetKey(s.trackedFunds[i])]; ok {
			s.trackedFunds[i].LastCollectedAt = collectedAt
		}
	}
	s.trackedFundsMu.Unlock()

	s.managedTargetsMu.Lock()
	for i := range s.managedTargets {
		if _, ok := keys[makeTrackedTargetKey(s.managedTargets[i])]; ok {
			s.managedTargets[i].LastCollectedAt = collectedAt
		}
	}
	s.managedTargetsMu.Unlock()
}

func makeTrackedTargetKey(target trackedFundTarget) string {
	return string(target.Source) + "|" + strings.TrimSpace(target.FundID)
}

func sortTrackedTargetsByLastCollected(targets []trackedFundTarget) {
	sort.Slice(targets, func(i, j int) bool {
		left := targets[i]
		right := targets[j]
		switch {
		case left.LastCollectedAt.IsZero() && right.LastCollectedAt.IsZero():
			return left.FundID < right.FundID
		case left.LastCollectedAt.IsZero():
			return true
		case right.LastCollectedAt.IsZero():
			return false
		case !left.LastCollectedAt.Equal(right.LastCollectedAt):
			return left.LastCollectedAt.Before(right.LastCollectedAt)
		default:
			return left.FundID < right.FundID
		}
	})
}

func limitTrackedTargets(targets []trackedFundTarget, limit int) []trackedFundTarget {
	if limit <= 0 || len(targets) <= limit {
		return targets
	}
	return targets[:limit]
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
