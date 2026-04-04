package service

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/RomaticDOG/fund/internal/crawler"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/trading"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

type weightedTrendPoint struct {
	weightedSum decimal.Decimal
	totalWeight decimal.Decimal
}

func (s *ValuationServiceImpl) preferredTimeSeriesDate(now time.Time) time.Time {
	switch trading.GetCurrentSession(now) {
	case trading.SessionMorning, trading.SessionAfternoon, trading.SessionLunchBreak:
		return now
	default:
		return previousTradingDay(now)
	}
}

func previousTradingDay(now time.Time) time.Time {
	return trading.GetPreviousTradingDay(now)
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

	if len(holdings) == 0 && s.fundResolver != nil {
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

	if len(holdings) == 0 {
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
	aggregates := make(map[time.Time]*weightedTrendPoint)
	hundred := decimal.NewFromInt(100)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(5)

	type seriesResult struct {
		Holding   domain.StockHolding
		Bars      []crawler.SinaKLinePoint
		PrevClose decimal.Decimal
	}

	resultsCh := make(chan seriesResult, len(holdings))

	for _, holding := range holdings {
		holding := holding
		g.Go(func() error {
			bars, err := fetcher.FetchFiveMinuteKLines(ctx, holding.StockCode, 300)
			if err != nil {
				return nil
			}
			dayBars, prevClose := selectBarsForDate(bars, targetDate)
			if len(dayBars) == 0 || prevClose.IsZero() {
				return nil
			}
			resultsCh <- seriesResult{Holding: holding, Bars: dayBars, PrevClose: prevClose}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	close(resultsCh)

	for result := range resultsCh {
		sessionOpen := tradingDayOpen(result.Bars[0].Timestamp)
		addWeightedAggregate(aggregates, sessionOpen, result.Bars[0].Open, result.PrevClose, result.Holding.HoldingRatio)
		for _, bar := range result.Bars {
			addWeightedAggregate(aggregates, bar.Timestamp, bar.Close, result.PrevClose, result.Holding.HoldingRatio)
		}
	}

	return buildWeightedSeriesPoints(aggregates, fund.NetAssetVal, hundred)
}

func (s *ValuationServiceImpl) buildDirectInstrumentTimeSeries(ctx context.Context, fund *domain.Fund, instrumentCode string, targetDate time.Time) ([]domain.TimeSeriesPoint, error) {
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

func addWeightedAggregate(aggregates map[time.Time]*weightedTrendPoint, ts time.Time, price, prevClose, holdingRatio decimal.Decimal) {
	if prevClose.IsZero() || holdingRatio.IsZero() {
		return
	}

	changePercent := price.Sub(prevClose).Div(prevClose).Mul(decimal.NewFromInt(100))
	if _, ok := aggregates[ts]; !ok {
		aggregates[ts] = &weightedTrendPoint{}
	}
	aggregates[ts].weightedSum = aggregates[ts].weightedSum.Add(changePercent.Mul(holdingRatio))
	aggregates[ts].totalWeight = aggregates[ts].totalWeight.Add(holdingRatio)
}

func buildWeightedSeriesPoints(aggregates map[time.Time]*weightedTrendPoint, nav, hundred decimal.Decimal) ([]domain.TimeSeriesPoint, error) {
	if len(aggregates) == 0 {
		return nil, fmt.Errorf("no aggregated intraday points built")
	}

	timestamps := make([]time.Time, 0, len(aggregates))
	for ts := range aggregates {
		timestamps = append(timestamps, ts)
	}
	sort.Slice(timestamps, func(i, j int) bool { return timestamps[i].Before(timestamps[j]) })

	points := make([]domain.TimeSeriesPoint, 0, len(timestamps))
	for _, ts := range timestamps {
		aggregate := aggregates[ts]
		if aggregate.totalWeight.IsZero() {
			continue
		}
		changePercent := aggregate.weightedSum.Div(aggregate.totalWeight).Round(4)
		estimateNav := decimal.Zero
		if !nav.IsZero() {
			estimateNav = nav.Mul(decimal.NewFromInt(1).Add(changePercent.Div(hundred))).Round(4)
		}
		points = append(points, domain.TimeSeriesPoint{
			Timestamp:     ts,
			ChangePercent: changePercent,
			EstimateNav:   estimateNav,
		})
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("no valid weighted intraday points built")
	}
	return points, nil
}

func ensureLunchBreakResumePoint(points []domain.TimeSeriesPoint) []domain.TimeSeriesPoint {
	if len(points) == 0 {
		return points
	}

	loc := points[0].Timestamp.Location()
	if loc == nil {
		loc = time.Local
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

	if has1300 || elevenThirty == nil {
		return points
	}

	resumePoint := *elevenThirty
	baseTs := elevenThirty.Timestamp.In(loc)
	resumePoint.Timestamp = time.Date(baseTs.Year(), baseTs.Month(), baseTs.Day(), 13, 0, 0, 0, loc)
	points = append(points, resumePoint)

	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp.Before(points[j].Timestamp)
	})

	return points
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
