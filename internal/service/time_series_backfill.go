package service

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/crawler"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/trading"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

type weightedSeriesSample struct {
	Timestamp time.Time
	Price     decimal.Decimal
}

type weightedSeriesResult struct {
	Holding     domain.StockHolding
	SessionOpen *weightedSeriesSample
	Samples     []weightedSeriesSample
	PrevClose   decimal.Decimal
}

func (s *ValuationServiceImpl) preferredTimeSeriesDate(now time.Time) time.Time {
	if shouldUseCurrentTradingDayTimeSeries(now) {
		return now
	}
	return previousTradingDay(now)
}

func previousTradingDay(now time.Time) time.Time {
	return trading.GetPreviousTradingDay(now)
}

func shouldUseCurrentTradingDayTimeSeries(now time.Time) bool {
	local := now.In(trading.TradingLocation())
	if !trading.IsTradingDay(local) {
		return false
	}

	switch trading.GetCurrentSession(local) {
	case trading.SessionMorning, trading.SessionLunchBreak, trading.SessionAfternoon, trading.SessionAfterHours:
		return true
	default:
		return false
	}
}

func shouldAllowPreviousTradingDayTimeSeriesFallback(now time.Time) bool {
	local := now.In(trading.TradingLocation())
	return !(trading.IsTradingDay(local) && trading.GetCurrentSession(local) == trading.SessionAfterHours)
}

func needsTimeSeriesBackfill(points []domain.TimeSeriesPoint, targetDate, now time.Time) bool {
	if len(points) == 0 {
		return true
	}

	sorted := make([]domain.TimeSeriesPoint, len(points))
	copy(sorted, points)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	loc := sorted[0].Timestamp.Location()
	if loc == nil {
		loc = time.Local
	}

	expectedDate := targetDate.In(loc).Format("2006-01-02")
	firstPoint := sorted[0].Timestamp.In(loc)
	if firstPoint.Format("2006-01-02") != expectedDate {
		return true
	}

	expectedOpen := time.Date(firstPoint.Year(), firstPoint.Month(), firstPoint.Day(), 9, 30, 0, 0, loc)
	if firstPoint.After(expectedOpen) {
		return true
	}

	for _, point := range sorted {
		localTs := point.Timestamp.In(loc)
		if localTs.Format("2006-01-02") != expectedDate {
			return true
		}
		if !isAlignedTimeSeriesPoint(localTs) {
			return true
		}
	}

	for i := 1; i < len(sorted); i++ {
		prev := sorted[i-1].Timestamp.In(loc)
		curr := sorted[i].Timestamp.In(loc)
		gap := curr.Sub(prev)
		if gap <= 10*time.Minute {
			continue
		}
		if isLunchBreakGap(prev, curr) {
			continue
		}
		return true
	}

	expectedLast := expectedLatestTimePoint(now.In(loc), targetDate.In(loc))
	lastPoint := sorted[len(sorted)-1].Timestamp.In(loc)
	if lastPoint.Before(expectedLast) {
		return true
	}

	return false
}

func isAlignedTimeSeriesPoint(ts time.Time) bool {
	if ts.Second() != 0 || ts.Nanosecond() != 0 {
		return false
	}
	if ts.Hour() == 9 && ts.Minute() == 30 {
		return true
	}
	return ts.Minute()%5 == 0
}

func isLunchBreakGap(prev, curr time.Time) bool {
	return prev.Hour() == 11 && prev.Minute() == 30 && curr.Hour() == 13 && curr.Minute() == 0
}

func expectedLatestTimePoint(now, targetDate time.Time) time.Time {
	dayStart := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	session := trading.GetCurrentSession(now)

	if targetDate.Format("2006-01-02") != now.Format("2006-01-02") {
		return dayStart.Add(15 * time.Hour)
	}

	switch session {
	case trading.SessionMorning:
		return floorToFiveMinute(now, dayStart.Add(9*time.Hour+30*time.Minute))
	case trading.SessionLunchBreak:
		return dayStart.Add(11*time.Hour + 30*time.Minute)
	case trading.SessionAfternoon:
		return floorToFiveMinute(now, dayStart.Add(13*time.Hour))
	default:
		return dayStart.Add(15 * time.Hour)
	}
}

func floorToFiveMinute(now, minimum time.Time) time.Time {
	if now.Before(minimum) {
		return minimum
	}
	flooredMinute := (now.Minute() / 5) * 5
	floored := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), flooredMinute, 0, 0, now.Location())
	if floored.Before(minimum) {
		return minimum
	}
	return floored
}

func (s *ValuationServiceImpl) loadTimeSeriesInputs(ctx context.Context, fundID string) (*domain.Fund, []domain.StockHolding, string, bool, error) {
	fund, err := s.fundRepo.GetFundByID(ctx, fundID)
	if err != nil {
		return nil, nil, fundID, false, fmt.Errorf("failed to get fund: %w", err)
	}
	if fund == nil {
		return nil, nil, fundID, false, fmt.Errorf("fund not found: %s", fundID)
	}

	holdingsSource := fundID
	holdings, err := s.fundRepo.GetFundHoldings(ctx, fundID)
	if err != nil {
		return nil, nil, fundID, false, fmt.Errorf("failed to get holdings: %w", err)
	}

	fund, holdings, warmupScheduled := useCachedFundDataOrScheduleWarmup(s.dataLoader, fundID, fund, holdings)

	if !hasEffectiveHoldings(holdings) && s.fundResolver != nil {
		holdings, holdingsSource, err = s.fundResolver.GetHoldingsWithFallback(ctx, fundID, fund.Name)
		if err != nil {
			log.Printf("⚠️ Feeder fund resolution for time series failed for %s: %v", fundID, err)
		}
	}

	return fund, holdings, holdingsSource, warmupScheduled, nil
}

func (s *ValuationServiceImpl) backfillTimeSeries(ctx context.Context, fundID string, targetDate time.Time) ([]domain.TimeSeriesPoint, error) {
	fund, holdings, holdingsSource, warmupScheduled, err := s.loadTimeSeriesInputs(ctx, fundID)
	if err != nil {
		return nil, err
	}

	if fund.NetAssetVal.IsZero() && warmupScheduled {
		return nil, ErrFundDataWarmupInProgress
	}

	if !hasEffectiveHoldings(holdings) {
		if holdingsSource != fundID && holdingsSource != "" {
			return s.buildDirectInstrumentTimeSeries(ctx, fund, holdingsSource, targetDate)
		}
		if warmupScheduled {
			return nil, ErrFundDataWarmupInProgress
		}
		return nil, fmt.Errorf("no holdings available to build intraday time series for %s", fundID)
	}

	return s.buildWeightedTimeSeries(ctx, fund, holdings, targetDate)
}

func (s *ValuationServiceImpl) buildWeightedTimeSeries(ctx context.Context, fund *domain.Fund, holdings []domain.StockHolding, targetDate time.Time) ([]domain.TimeSeriesPoint, error) {
	fetcher := crawler.NewSinaKLineFetcher()
	minuteFetcher := crawler.NewTencentMinuteFetcher()
	stockCodes := make([]string, 0, len(holdings))
	for _, holding := range holdings {
		stockCodes = append(stockCodes, holding.StockCode)
	}
	quotes, _ := s.fetchQuotesWithCache(ctx, stockCodes)
	source, _ := s.resolveQuoteProvider(ctx)
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(5)
	resultsCh := make(chan weightedSeriesResult, len(holdings))

	for _, holding := range holdings {
		holding := holding
		g.Go(func() error {
			result, err := buildHoldingWeightedSeries(ctx, fetcher, minuteFetcher, holding, targetDate, quotes[holding.StockCode].PrevClose)
			if err != nil || result == nil {
				return nil
			}
			resultsCh <- *result
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	close(resultsCh)

	seriesResults := make([]weightedSeriesResult, 0, len(holdings))
	for result := range resultsCh {
		seriesResults = append(seriesResults, result)
	}

	if len(seriesResults) == 0 {
		return nil, fmt.Errorf("no holding intraday series built")
	}

	slots := buildIntradayTimeSeriesSlots(targetDate, now)
	return buildWeightedSeriesPoints(fund, holdings, seriesResults, source, slots)
}

func (s *ValuationServiceImpl) buildDirectInstrumentTimeSeries(ctx context.Context, fund *domain.Fund, instrumentCode string, targetDate time.Time) ([]domain.TimeSeriesPoint, error) {
	if isHongKongCode(instrumentCode) {
		minuteFetcher := crawler.NewTencentMinuteFetcher()
		quotes, _ := s.fetchQuotesWithCache(ctx, []string{instrumentCode})
		prevClose := quotes[instrumentCode].PrevClose
		minutePoints, err := minuteFetcher.FetchIntradayMinutes(ctx, instrumentCode)
		if err != nil {
			return nil, err
		}

		sessionOpen, samples := selectFiveMinuteMinuteSamples(minutePoints, targetDate)
		if sessionOpen == nil || prevClose.IsZero() {
			return nil, fmt.Errorf("no intraday minute data for %s on %s", instrumentCode, targetDate.Format("2006-01-02"))
		}

		points := make([]domain.TimeSeriesPoint, 0, len(samples)+1)
		points = append(points, buildInstrumentSeriesPoint(sessionOpen.Timestamp, sessionOpen.Price, prevClose, fund.NetAssetVal))
		for _, sample := range samples {
			points = append(points, buildInstrumentSeriesPoint(sample.Timestamp, sample.Price, prevClose, fund.NetAssetVal))
		}
		return points, nil
	}

	fetcher := crawler.NewSinaKLineFetcher()
	bars, err := fetcher.FetchFiveMinuteKLines(ctx, instrumentCode, 300)
	if err != nil {
		return nil, err
	}

	dayBars, prevClose := selectBarsForDate(bars, targetDate)
	if len(dayBars) == 0 || prevClose.IsZero() {
		return nil, fmt.Errorf("no intraday kline data for %s on %s", instrumentCode, targetDate.Format("2006-01-02"))
	}

	points := make([]domain.TimeSeriesPoint, 0, len(dayBars)+1)
	points = append(points, buildInstrumentSeriesPoint(tradingDayOpen(dayBars[0].Timestamp), dayBars[0].Open, prevClose, fund.NetAssetVal))
	for _, bar := range dayBars {
		points = append(points, buildInstrumentSeriesPoint(bar.Timestamp, bar.Close, prevClose, fund.NetAssetVal))
	}
	return points, nil
}

func buildInstrumentSeriesPoint(ts time.Time, price, prevClose, nav decimal.Decimal) domain.TimeSeriesPoint {
	hundred := decimal.NewFromInt(100)
	changePercent := decimal.Zero
	if !prevClose.IsZero() {
		changePercent = price.Sub(prevClose).Div(prevClose).Mul(hundred).Round(4)
	}

	estimateNav := decimal.Zero
	if !nav.IsZero() {
		estimateNav = nav.Mul(decimal.NewFromInt(1).Add(changePercent.Div(hundred))).Round(4)
	}

	return domain.TimeSeriesPoint{
		Timestamp:     ts,
		ChangePercent: changePercent,
		EstimateNav:   estimateNav,
	}
}

func buildHoldingWeightedSeries(
	ctx context.Context,
	klineFetcher *crawler.SinaKLineFetcher,
	minuteFetcher *crawler.TencentMinuteFetcher,
	holding domain.StockHolding,
	targetDate time.Time,
	quotePrevClose decimal.Decimal,
) (*weightedSeriesResult, error) {
	if isHongKongHolding(holding) {
		minutePoints, err := minuteFetcher.FetchIntradayMinutes(ctx, holding.StockCode)
		if err != nil {
			return nil, err
		}

		sessionOpen, samples := selectFiveMinuteMinuteSamples(minutePoints, targetDate)
		if sessionOpen == nil || quotePrevClose.IsZero() {
			return nil, fmt.Errorf("no hong kong minute data for %s on %s", holding.StockCode, targetDate.Format("2006-01-02"))
		}

		return &weightedSeriesResult{
			Holding:     holding,
			SessionOpen: sessionOpen,
			Samples:     samples,
			PrevClose:   quotePrevClose,
		}, nil
	}

	bars, err := klineFetcher.FetchFiveMinuteKLines(ctx, holding.StockCode, 300)
	if err != nil {
		return nil, err
	}

	dayBars, prevClose := selectBarsForDate(bars, targetDate)
	if len(dayBars) == 0 {
		return nil, fmt.Errorf("no kline data for %s on %s", holding.StockCode, targetDate.Format("2006-01-02"))
	}
	if prevClose.IsZero() {
		prevClose = quotePrevClose
	}
	if prevClose.IsZero() {
		return nil, fmt.Errorf("no previous close for %s", holding.StockCode)
	}

	samples := make([]weightedSeriesSample, 0, len(dayBars))
	for _, bar := range dayBars {
		samples = append(samples, weightedSeriesSample{
			Timestamp: bar.Timestamp,
			Price:     bar.Close,
		})
	}
	sort.Slice(samples, func(i, j int) bool {
		return samples[i].Timestamp.Before(samples[j].Timestamp)
	})

	sessionOpen := weightedSeriesSample{
		Timestamp: tradingDayOpen(dayBars[0].Timestamp),
		Price:     dayBars[0].Open,
	}

	return &weightedSeriesResult{
		Holding:     holding,
		SessionOpen: &sessionOpen,
		Samples:     samples,
		PrevClose:   prevClose,
	}, nil
}

func buildWeightedSeriesPoints(
	fund *domain.Fund,
	holdings []domain.StockHolding,
	seriesResults []weightedSeriesResult,
	source domain.QuoteSource,
	slots []time.Time,
) ([]domain.TimeSeriesPoint, error) {
	if len(seriesResults) == 0 {
		return nil, fmt.Errorf("no aggregated intraday points built")
	}

	points := make([]domain.TimeSeriesPoint, 0, len(slots))
	for _, slot := range slots {
		quotes := make(map[string]domain.StockQuote, len(seriesResults))
		for _, series := range seriesResults {
			quote, ok := buildQuoteSnapshotAtSlot(series, slot)
			if !ok {
				continue
			}
			quotes[series.Holding.StockCode] = quote
		}

		estimate := buildEstimateSnapshotFromQuotes(fund, holdings, quotes, source, slot)
		if estimate.TotalHoldRatio.IsZero() {
			continue
		}

		points = append(points, domain.TimeSeriesPoint{
			Timestamp:     slot,
			ChangePercent: estimate.ChangePercent,
			EstimateNav:   estimate.EstimateNav,
		})
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("no valid weighted intraday points built")
	}
	return points, nil
}

func buildQuoteSnapshotAtSlot(series weightedSeriesResult, slot time.Time) (domain.StockQuote, bool) {
	price, ok := latestSeriesPriceAtOrBefore(series, slot)
	if !ok || series.PrevClose.IsZero() {
		return domain.StockQuote{}, false
	}

	changePercent := price.Sub(series.PrevClose).Div(series.PrevClose).Mul(decimal.NewFromInt(100)).Round(4)
	return domain.StockQuote{
		StockCode:     series.Holding.StockCode,
		StockName:     series.Holding.StockName,
		CurrentPrice:  price,
		PrevClose:     series.PrevClose,
		ChangePercent: changePercent,
		UpdatedAt:     slot,
	}, true
}

func latestSeriesPriceAtOrBefore(series weightedSeriesResult, slot time.Time) (decimal.Decimal, bool) {
	slot = slot.In(trading.TradingLocation())
	if series.SessionOpen == nil {
		return decimal.Zero, false
	}

	baseTs := series.SessionOpen.Timestamp.In(trading.TradingLocation())
	if slot.Before(baseTs) {
		return decimal.Zero, false
	}

	price := series.SessionOpen.Price
	if !slot.After(baseTs) {
		return price, true
	}

	for _, sample := range series.Samples {
		sampleTs := sample.Timestamp.In(trading.TradingLocation())
		if sampleTs.After(slot) {
			break
		}
		price = sample.Price
	}
	return price, true
}

func buildIntradayTimeSeriesSlots(targetDate, now time.Time) []time.Time {
	loc := trading.TradingLocation()
	targetLocal := targetDate.In(loc)
	nowLocal := now.In(loc)
	end := expectedLatestTimePoint(nowLocal, targetLocal)
	day := time.Date(targetLocal.Year(), targetLocal.Month(), targetLocal.Day(), 0, 0, 0, 0, loc)

	slots := make([]time.Time, 0, 64)
	appendRange := func(startHour, startMinute, endHour, endMinute int) {
		start := time.Date(day.Year(), day.Month(), day.Day(), startHour, startMinute, 0, 0, loc)
		finish := time.Date(day.Year(), day.Month(), day.Day(), endHour, endMinute, 0, 0, loc)
		for ts := start; !ts.After(finish) && !ts.After(end); ts = ts.Add(5 * time.Minute) {
			slots = append(slots, ts)
		}
	}

	appendRange(9, 30, 11, 30)

	if shouldIncludeLunchResumeSlot(targetLocal, nowLocal) {
		resume := time.Date(day.Year(), day.Month(), day.Day(), 13, 0, 0, 0, loc)
		if !resume.After(end) {
			slots = append(slots, resume)
		}
	}

	appendRange(13, 5, 15, 0)
	return slots
}

func shouldIncludeLunchResumeSlot(targetDate, now time.Time) bool {
	if targetDate.Format("2006-01-02") != now.Format("2006-01-02") {
		return true
	}

	switch trading.GetCurrentSession(now) {
	case trading.SessionAfternoon, trading.SessionAfterHours:
		return true
	default:
		return false
	}
}

func ensureLunchBreakResumePoint(points []domain.TimeSeriesPoint, now time.Time) []domain.TimeSeriesPoint {
	if len(points) == 0 {
		return points
	}

	loc := points[0].Timestamp.Location()
	if loc == nil {
		loc = trading.TradingLocation()
	}

	pointDate := points[0].Timestamp.In(trading.TradingLocation()).Format("2006-01-02")
	if !now.IsZero() {
		localNow := now.In(trading.TradingLocation())
		if pointDate == localNow.Format("2006-01-02") {
			switch trading.GetCurrentSession(localNow) {
			case trading.SessionAfternoon, trading.SessionAfterHours:
				// allow synthetic 13:00 continuity point
			default:
				return removeLunchBreakResumePoint(points, loc)
			}
		}
	}

	var has1300 bool
	var elevenThirty *domain.TimeSeriesPoint

	for i := range points {
		localTs := points[i].Timestamp.In(loc)
		if localTs.Hour() == 13 && localTs.Minute() == 0 {
			has1300 = true
			break
		}
		if localTs.Hour() == 11 && localTs.Minute() == 30 {
			copyPoint := points[i]
			elevenThirty = &copyPoint
		}
	}

	if elevenThirty == nil {
		return points
	}

	resumePoint := *elevenThirty
	baseTs := elevenThirty.Timestamp.In(loc)
	resumePoint.Timestamp = time.Date(baseTs.Year(), baseTs.Month(), baseTs.Day(), 13, 0, 0, 0, loc)

	if has1300 {
		for i := range points {
			localTs := points[i].Timestamp.In(loc)
			if localTs.Hour() == 13 && localTs.Minute() == 0 {
				points[i].ChangePercent = resumePoint.ChangePercent
				points[i].EstimateNav = resumePoint.EstimateNav
				points[i].Timestamp = resumePoint.Timestamp
			}
		}
	} else {
		points = append(points, resumePoint)
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp.Before(points[j].Timestamp)
	})

	return points
}

func removeLunchBreakResumePoint(points []domain.TimeSeriesPoint, loc *time.Location) []domain.TimeSeriesPoint {
	filtered := make([]domain.TimeSeriesPoint, 0, len(points))
	for _, point := range points {
		localTs := point.Timestamp.In(loc)
		if localTs.Hour() == 13 && localTs.Minute() == 0 {
			continue
		}
		filtered = append(filtered, point)
	}
	return filtered
}

func isHongKongHolding(holding domain.StockHolding) bool {
	return holding.Exchange == domain.ExchangeHK || isHongKongCode(holding.StockCode)
}

func isHongKongCode(code string) bool {
	return len(strings.TrimSpace(code)) == 5
}

func selectFiveMinuteMinuteSamples(points []crawler.TencentMinutePoint, targetDate time.Time) (*weightedSeriesSample, []weightedSeriesSample) {
	targetKey := targetDate.In(trading.TradingLocation()).Format("2006-01-02")
	filtered := make([]weightedSeriesSample, 0, len(points))

	for _, point := range points {
		localTs := point.Timestamp.In(trading.TradingLocation())
		if localTs.Format("2006-01-02") != targetKey {
			continue
		}

		totalMinutes := localTs.Hour()*60 + localTs.Minute()
		inMorning := totalMinutes >= 9*60+30 && totalMinutes <= 11*60+30
		inAfternoon := totalMinutes >= 13*60 && totalMinutes <= 15*60
		if !inMorning && !inAfternoon {
			continue
		}
		if localTs.Hour() == 13 && localTs.Minute() == 0 {
			continue
		}
		if localTs.Minute()%5 != 0 {
			continue
		}

		filtered = append(filtered, weightedSeriesSample{
			Timestamp: time.Date(localTs.Year(), localTs.Month(), localTs.Day(), localTs.Hour(), localTs.Minute(), 0, 0, trading.TradingLocation()),
			Price:     point.Price,
		})
	}

	if len(filtered) == 0 {
		return nil, nil
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})

	sessionOpen := filtered[0]
	if sessionOpen.Timestamp.Hour() != 9 || sessionOpen.Timestamp.Minute() != 30 {
		return &sessionOpen, filtered[1:]
	}

	samples := make([]weightedSeriesSample, 0, len(filtered)-1)
	for _, sample := range filtered[1:] {
		samples = append(samples, sample)
	}

	return &sessionOpen, samples
}

func selectBarsForDate(bars []crawler.SinaKLinePoint, targetDate time.Time) ([]crawler.SinaKLinePoint, decimal.Decimal) {
	if len(bars) == 0 {
		return nil, decimal.Zero
	}

	targetKey := targetDate.Format("2006-01-02")
	filtered := make([]crawler.SinaKLinePoint, 0)
	prevClose := decimal.Zero

	for _, bar := range bars {
		barDate := bar.Timestamp.Format("2006-01-02")
		if barDate < targetKey {
			prevClose = bar.Close
			continue
		}
		if barDate == targetKey {
			filtered = append(filtered, bar)
		}
	}

	if prevClose.IsZero() && len(filtered) > 0 {
		prevClose = filtered[0].Open
	}

	return filtered, prevClose
}

func tradingDayOpen(ts time.Time) time.Time {
	return time.Date(ts.Year(), ts.Month(), ts.Day(), 9, 30, 0, 0, ts.Location())
}
