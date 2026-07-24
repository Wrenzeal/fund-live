package service

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/shopspring/decimal"
	"gorm.io/gorm/clause"
)

type HistoricalProxyFactors struct {
	Momentum20          decimal.Decimal `json:"momentum_20"`
	Momentum60          decimal.Decimal `json:"momentum_60"`
	Volatility20        decimal.Decimal `json:"volatility_20"`
	Drawdown60          decimal.Decimal `json:"drawdown_60"`
	AverageAmount20     decimal.Decimal `json:"average_amount_20"`
	PointInTimeBoundary string          `json:"point_in_time_boundary"`
}

func (s *QuantResearchStore) BuildHistoricalProxySignals(ctx context.Context, start, end time.Time) (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	if end.IsZero() {
		end = time.Now()
	}
	if start.IsZero() {
		start = end.AddDate(-5, 0, 0)
	}
	inserted := 0
	loc := time.FixedZone("CST", 8*60*60)
	for _, instrument := range PilotV1Instruments() {
		var bars []database.QuantMarketBar
		if err := s.db.WithContext(ctx).Where("symbol = ? AND date <= ?", instrument.Symbol, end).Order("date").Find(&bars).Error; err != nil {
			return inserted, err
		}
		records := make([]database.QuantSignalHistory, 0, len(bars))
		for index := 120; index < len(bars); index++ {
			bar := bars[index]
			if bar.Date.Before(start) || bar.Date.After(end) {
				continue
			}
			factors, score := historicalProxyScore(bars, index)
			inputJSON, _ := json.Marshal(factors)
			outputJSON, _ := json.Marshal(map[string]interface{}{
				"analysis_version": "historical_proxy_v1",
				"total_score":      score,
				"factors":          factors,
			})
			decisionAt := time.Date(bar.Date.In(loc).Year(), bar.Date.In(loc).Month(), bar.Date.In(loc).Day(), 15, 30, 0, 0, loc)
			records = append(records, database.QuantSignalHistory{
				FundID:           instrument.Symbol,
				AnalysisVersion:  "historical_proxy_v1",
				SignalDate:       bar.Date,
				Mode:             QuantSignalModeHistoryProxy,
				DecisionAt:       decisionAt,
				DataCutoffAt:     decisionAt,
				TotalScore:       score,
				ShadowEventScore: decimal.NewFromInt(50),
				InputJSON:        inputJSON,
				OutputJSON:       outputJSON,
				EventIDs:         json.RawMessage(`[]`),
			})
		}
		if len(records) == 0 {
			continue
		}
		result := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(records, 500)
		if result.Error != nil {
			return inserted, result.Error
		}
		inserted += int(result.RowsAffected)
	}
	return inserted, nil
}

func historicalProxyScore(bars []database.QuantMarketBar, index int) (HistoricalProxyFactors, decimal.Decimal) {
	closeAt := func(offset int) float64 {
		value, _ := bars[index-offset].AdjustedClose.Float64()
		return value
	}
	current := closeAt(0)
	momentum20 := percentChange(current, closeAt(20))
	momentum60 := percentChange(current, closeAt(60))
	returns := make([]float64, 0, 20)
	amountTotal := decimal.Zero
	peak := 0.0
	for cursor := index - 59; cursor <= index; cursor++ {
		closeValue, _ := bars[cursor].AdjustedClose.Float64()
		if closeValue > peak {
			peak = closeValue
		}
		if cursor >= index-19 {
			previous, _ := bars[cursor-1].AdjustedClose.Float64()
			returns = append(returns, percentChange(closeValue, previous)/100)
			amountTotal = amountTotal.Add(bars[cursor].Amount)
		}
	}
	volatility20 := standardDeviation(returns) * math.Sqrt(252) * 100
	drawdown60 := 0.0
	if peak > 0 {
		drawdown60 = (current/peak - 1) * 100
	}
	averageAmount := amountTotal.Div(decimal.NewFromInt(20))
	liquidityBonus := 0.0
	if averageAmount.GreaterThanOrEqual(decimal.NewFromInt(100_000_000)) {
		liquidityBonus = 4
	} else if averageAmount.GreaterThanOrEqual(decimal.NewFromInt(20_000_000)) {
		liquidityBonus = 2
	} else {
		liquidityBonus = -6
	}
	score := clamp(50+0.9*momentum20+0.35*momentum60-0.30*volatility20+0.35*drawdown60+liquidityBonus, 5, 95)
	return HistoricalProxyFactors{
		Momentum20:          decimal.NewFromFloat(momentum20).Round(4),
		Momentum60:          decimal.NewFromFloat(momentum60).Round(4),
		Volatility20:        decimal.NewFromFloat(volatility20).Round(4),
		Drawdown60:          decimal.NewFromFloat(drawdown60).Round(4),
		AverageAmount20:     averageAmount.Round(2),
		PointInTimeBoundary: "仅使用决策日收盘前已形成的价格与成交数据；下一交易日开盘成交。",
	}, decimal.NewFromFloat(score).Round(4)
}

func percentChange(current, previous float64) float64 {
	if previous == 0 {
		return 0
	}
	return (current/previous - 1) * 100
}

func standardDeviation(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := 0.0
	for _, value := range values {
		mean += value
	}
	mean /= float64(len(values))
	variance := 0.0
	for _, value := range values {
		delta := value - mean
		variance += delta * delta
	}
	return math.Sqrt(variance / float64(len(values)-1))
}

type QuantForwardHorizonSummary struct {
	HorizonDays         int             `json:"horizon_days"`
	SampleCount         int             `json:"sample_count"`
	AverageReturn       decimal.Decimal `json:"average_return"`
	AverageExcessReturn decimal.Decimal `json:"average_excess_return"`
	PositiveRate        decimal.Decimal `json:"positive_rate"`
	AverageRankIC       decimal.Decimal `json:"average_rank_ic"`
}

type QuantValidationSummary struct {
	Mode              string                       `json:"mode"`
	UniverseVersion   string                       `json:"universe_version"`
	SignalCount       int                          `json:"signal_count"`
	FirstSignalDate   *time.Time                   `json:"first_signal_date,omitempty"`
	LastSignalDate    *time.Time                   `json:"last_signal_date,omitempty"`
	Horizons          []QuantForwardHorizonSummary `json:"horizons"`
	LookaheadBoundary string                       `json:"lookahead_boundary"`
	Coverage          QuantResearchCoverage        `json:"coverage"`
}

type QuantResearchCoverage struct {
	InstrumentCount    int64      `json:"instrument_count"`
	MarketBarCount     int64      `json:"market_bar_count"`
	EventCount         int64      `json:"event_count"`
	EventVersionCount  int64      `json:"event_version_count"`
	MappedFundCount    int64      `json:"mapped_fund_count"`
	ExperimentCount    int64      `json:"experiment_count"`
	LatestMarketDate   *time.Time `json:"latest_market_date,omitempty"`
	LatestEventKnownAt *time.Time `json:"latest_event_known_at,omitempty"`
}

type quantSignalOutcome struct {
	Date   time.Time
	Score  float64
	Return float64
	Excess float64
}

func (s *QuantResearchStore) ValidationSummary(ctx context.Context, mode string) (*QuantValidationSummary, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if mode == "" {
		mode = QuantSignalModeHistoryProxy
	}
	var signals []database.QuantSignalHistory
	if err := s.db.WithContext(ctx).Where("mode = ?", mode).Order("signal_date, fund_id").Find(&signals).Error; err != nil {
		return nil, err
	}
	summary := &QuantValidationSummary{
		Mode:              mode,
		UniverseVersion:   QuantUniversePilotV1,
		SignalCount:       len(signals),
		LookaheadBoundary: "信号使用当日收盘数据，下一个交易日开盘建仓；事件必须满足 known_at <= decision_at。",
	}
	coverage, err := s.quantResearchCoverage(ctx)
	if err != nil {
		return nil, err
	}
	summary.Coverage = coverage
	if len(signals) == 0 {
		return summary, nil
	}
	first := signals[0].SignalDate
	last := signals[len(signals)-1].SignalDate
	summary.FirstSignalDate = &first
	summary.LastSignalDate = &last

	barsBySymbol := make(map[string][]database.QuantMarketBar)
	symbols := []string{"000300"}
	for _, item := range PilotV1Instruments() {
		symbols = append(symbols, item.Symbol)
	}
	var bars []database.QuantMarketBar
	if err := s.db.WithContext(ctx).Where("symbol IN ?", symbols).Order("symbol, date").Find(&bars).Error; err != nil {
		return nil, err
	}
	for _, bar := range bars {
		barsBySymbol[bar.Symbol] = append(barsBySymbol[bar.Symbol], bar)
	}

	for _, horizon := range []int{1, 5, 20} {
		outcomes := make([]quantSignalOutcome, 0, len(signals))
		positive := 0
		returnSum := 0.0
		excessSum := 0.0
		for _, signal := range signals {
			assetReturn, ok := forwardOpenToCloseReturn(barsBySymbol[signal.FundID], signal.SignalDate, horizon)
			if !ok {
				continue
			}
			benchmarkReturn, benchmarkOK := forwardOpenToCloseReturn(barsBySymbol["000300"], signal.SignalDate, horizon)
			if !benchmarkOK {
				benchmarkReturn = 0
			}
			score, _ := signal.TotalScore.Float64()
			excess := assetReturn - benchmarkReturn
			outcomes = append(outcomes, quantSignalOutcome{Date: signal.SignalDate, Score: score, Return: assetReturn, Excess: excess})
			returnSum += assetReturn
			excessSum += excess
			if assetReturn > 0 {
				positive++
			}
		}
		count := len(outcomes)
		horizonSummary := QuantForwardHorizonSummary{HorizonDays: horizon, SampleCount: count}
		if count > 0 {
			horizonSummary.AverageReturn = decimal.NewFromFloat(returnSum / float64(count) * 100).Round(4)
			horizonSummary.AverageExcessReturn = decimal.NewFromFloat(excessSum / float64(count) * 100).Round(4)
			horizonSummary.PositiveRate = decimal.NewFromFloat(float64(positive) / float64(count) * 100).Round(2)
			horizonSummary.AverageRankIC = decimal.NewFromFloat(averageDailyRankIC(outcomes)).Round(4)
		}
		summary.Horizons = append(summary.Horizons, horizonSummary)
	}
	return summary, nil
}

func (s *QuantResearchStore) quantResearchCoverage(ctx context.Context) (QuantResearchCoverage, error) {
	coverage := QuantResearchCoverage{}
	counts := []struct {
		model  interface{}
		target *int64
	}{
		{&database.QuantInstrument{}, &coverage.InstrumentCount},
		{&database.QuantMarketBar{}, &coverage.MarketBarCount},
		{&database.QuantEvent{}, &coverage.EventCount},
		{&database.QuantEventVersion{}, &coverage.EventVersionCount},
		{&database.QuantBacktestExperiment{}, &coverage.ExperimentCount},
	}
	for _, item := range counts {
		if err := s.db.WithContext(ctx).Model(item.model).Count(item.target).Error; err != nil {
			return coverage, err
		}
	}
	if err := s.db.WithContext(ctx).Model(&database.QuantEventAsset{}).
		Where("asset_type = ?", "fund").Distinct("asset_code").Count(&coverage.MappedFundCount).Error; err != nil {
		return coverage, err
	}
	var latest struct {
		MarketDate   *time.Time `gorm:"column:market_date"`
		EventKnownAt *time.Time `gorm:"column:event_known_at"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT MAX(date) FROM quant_market_bars) AS market_date,
			(SELECT MAX(known_at) FROM quant_event_versions) AS event_known_at
	`).Scan(&latest).Error; err != nil {
		return coverage, err
	}
	coverage.LatestMarketDate = latest.MarketDate
	coverage.LatestEventKnownAt = latest.EventKnownAt
	return coverage, nil
}

func forwardOpenToCloseReturn(bars []database.QuantMarketBar, signalDate time.Time, horizon int) (float64, bool) {
	start := sort.Search(len(bars), func(index int) bool { return bars[index].Date.After(signalDate) })
	end := start + horizon - 1
	if start >= len(bars) || end >= len(bars) {
		return 0, false
	}
	open, _ := bars[start].Open.Float64()
	closeValue, _ := bars[end].Close.Float64()
	if open <= 0 {
		return 0, false
	}
	return closeValue/open - 1, true
}

func averageDailyRankIC(outcomes []quantSignalOutcome) float64 {
	byDate := make(map[string][]quantSignalOutcome)
	for _, outcome := range outcomes {
		key := outcome.Date.Format("2006-01-02")
		byDate[key] = append(byDate[key], outcome)
	}
	total := 0.0
	count := 0
	for _, values := range byDate {
		if len(values) < 3 {
			continue
		}
		scores := make([]float64, len(values))
		returns := make([]float64, len(values))
		for index, value := range values {
			scores[index] = value.Score
			returns[index] = value.Return
		}
		ic := pearson(rankValues(scores), rankValues(returns))
		if !math.IsNaN(ic) {
			total += ic
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func rankValues(values []float64) []float64 {
	type indexedValue struct {
		index int
		value float64
	}
	indexed := make([]indexedValue, len(values))
	for index, value := range values {
		indexed[index] = indexedValue{index: index, value: value}
	}
	sort.SliceStable(indexed, func(i, j int) bool { return indexed[i].value < indexed[j].value })
	ranks := make([]float64, len(values))
	for rank, value := range indexed {
		ranks[value.index] = float64(rank + 1)
	}
	return ranks
}

func pearson(left, right []float64) float64 {
	if len(left) != len(right) || len(left) < 2 {
		return math.NaN()
	}
	leftMean, rightMean := 0.0, 0.0
	for index := range left {
		leftMean += left[index]
		rightMean += right[index]
	}
	leftMean /= float64(len(left))
	rightMean /= float64(len(right))
	numerator, leftDenominator, rightDenominator := 0.0, 0.0, 0.0
	for index := range left {
		leftDelta := left[index] - leftMean
		rightDelta := right[index] - rightMean
		numerator += leftDelta * rightDelta
		leftDenominator += leftDelta * leftDelta
		rightDenominator += rightDelta * rightDelta
	}
	if leftDenominator == 0 || rightDenominator == 0 {
		return math.NaN()
	}
	return numerator / math.Sqrt(leftDenominator*rightDenominator)
}
