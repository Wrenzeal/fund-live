package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/service"
)

type leanChart struct {
	Series map[string]leanSeries `json:"series"`
}

type leanSeries struct {
	Values [][]interface{} `json:"values"`
}

func normalizeLeanResult(root map[string]json.RawMessage, manifest *service.LeanJobManifest, tradingDates map[string]struct{}) (json.RawMessage, error) {
	if manifest == nil {
		return nil, fmt.Errorf("job manifest is required")
	}
	if len(tradingDates) == 0 {
		return nil, fmt.Errorf("trading dates are required")
	}
	var statistics map[string]string
	if err := json.Unmarshal(firstRaw(root, "Statistics", "statistics"), &statistics); err != nil {
		return nil, fmt.Errorf("decode statistics: %w", err)
	}
	var charts map[string]leanChart
	if err := json.Unmarshal(firstRaw(root, "Charts", "charts"), &charts); err != nil {
		return nil, fmt.Errorf("decode charts: %w", err)
	}
	start, err := time.Parse("2006-01-02", manifest.Parameters.StartDate)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse("2006-01-02", manifest.Parameters.EndDate)
	if err != nil {
		return nil, err
	}
	loc := time.FixedZone("CST", 8*60*60)
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, loc)

	strategy := normalizedDailySeries(seriesValues(charts, "Strategy Equity", "Equity"), start, end, true, loc, tradingDates)
	csi300 := normalizedDailySeries(seriesValues(charts, "Benchmarks", "沪深300"), start, end, true, loc, tradingDates)
	pilot := normalizedDailySeries(seriesValues(charts, "Benchmarks", "试点池等权"), start, end, true, loc, tradingDates)
	cash := normalizedDailySeries(seriesValues(charts, "Benchmarks", "现金"), start, end, true, loc, tradingDates)
	drawdown := normalizedDailySeries(seriesValues(charts, "Drawdown", "Equity Drawdown"), start, end, false, loc, tradingDates)

	result := service.QuantNormalizedResult{
		SchemaVersion: "quant-backtest.v1",
		Summary: service.QuantBacktestMetricSummary{
			TotalReturnPct:       metricNumber(statistics, "Net Profit"),
			CAGRPct:              metricNumber(statistics, "Compounding Annual Return"),
			MaxDrawdownPct:       metricNumber(statistics, "Drawdown"),
			Sharpe:               metricNumber(statistics, "Sharpe Ratio"),
			Sortino:              metricNumber(statistics, "Sortino Ratio"),
			InformationRatio:     metricNumber(statistics, "Information Ratio"),
			PortfolioTurnoverPct: metricNumber(statistics, "Portfolio Turnover"),
			TotalFeesCNY:         metricNumber(statistics, "Total Fees"),
			TotalOrders:          int(metricNumber(statistics, "Total Orders")),
			StartEquityCNY:       metricNumber(statistics, "Start Equity"),
			EndEquityCNY:         metricNumber(statistics, "End Equity"),
			CSI300ReturnPct:      totalSeriesReturn(csi300),
			PilotEqualReturnPct:  totalSeriesReturn(pilot),
			CashReturnPct:        totalSeriesReturn(cash),
		},
		Series: service.QuantBacktestSeries{
			Strategy:         strategy,
			CSI300:           csi300,
			PilotEqualWeight: pilot,
			Cash:             cash,
			Drawdown:         drawdown,
		},
	}
	result.Summary.ExcessReturnPct = result.Summary.TotalReturnPct - result.Summary.CSI300ReturnPct
	result.AnnualReturns = annualReturns(strategy, csi300, pilot, loc)
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func seriesValues(charts map[string]leanChart, chartName, seriesName string) [][]interface{} {
	chart, ok := charts[chartName]
	if !ok {
		return nil
	}
	return chart.Series[seriesName].Values
}

func normalizedDailySeries(values [][]interface{}, start, end time.Time, normalize bool, loc *time.Location, tradingDates map[string]struct{}) []service.QuantSeriesPoint {
	byDate := make(map[string]service.QuantSeriesPoint)
	for _, raw := range values {
		if len(raw) < 2 {
			continue
		}
		timestamp, ok := interfaceFloat(raw[0])
		if !ok {
			continue
		}
		value, ok := interfaceFloat(raw[1])
		if !ok {
			continue
		}
		pointTime := time.Unix(int64(timestamp), 0).In(loc)
		if pointTime.Before(start) || pointTime.After(end) {
			continue
		}
		key := pointTime.Format("2006-01-02")
		if _, ok := tradingDates[key]; !ok {
			continue
		}
		point := service.QuantSeriesPoint{Time: int64(timestamp), Value: value}
		if existing, exists := byDate[key]; !exists || point.Time >= existing.Time {
			byDate[key] = point
		}
	}
	points := make([]service.QuantSeriesPoint, 0, len(byDate))
	for _, point := range byDate {
		points = append(points, point)
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Time < points[j].Time })
	if normalize && len(points) > 0 && points[0].Value != 0 {
		initial := points[0].Value
		for index := range points {
			points[index].Value = points[index].Value / initial * 100
		}
	}
	return points
}

func readLeanTradingDates(path string) (map[string]struct{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	dates := make(map[string]struct{})
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
		if len(record) == 0 {
			continue
		}
		date := strings.TrimSpace(record[0])
		if _, parseErr := time.Parse("2006-01-02", date); parseErr != nil {
			continue
		}
		dates[date] = struct{}{}
	}
	if len(dates) == 0 {
		return nil, fmt.Errorf("benchmark market file has no trading dates")
	}
	return dates, nil
}

func interfaceFloat(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func metricNumber(statistics map[string]string, key string) float64 {
	value := strings.TrimSpace(statistics[key])
	replacer := strings.NewReplacer("%", "", "¥", "", "$", "", ",", "", " ", "")
	value = replacer.Replace(value)
	parsed, _ := strconv.ParseFloat(value, 64)
	return parsed
}

func totalSeriesReturn(points []service.QuantSeriesPoint) float64 {
	if len(points) < 2 || points[0].Value == 0 {
		return 0
	}
	return (points[len(points)-1].Value/points[0].Value - 1) * 100
}

func annualReturns(strategy, csi300, pilot []service.QuantSeriesPoint, loc *time.Location) []service.QuantBacktestAnnualReturn {
	strategyByYear := groupSeriesByYear(strategy, loc)
	years := make([]int, 0, len(strategyByYear))
	for year := range strategyByYear {
		years = append(years, year)
	}
	sort.Ints(years)
	result := make([]service.QuantBacktestAnnualReturn, 0, len(years))
	for _, year := range years {
		strategyPoints := strategyByYear[year]
		if len(strategyPoints) < 2 {
			continue
		}
		strategyReturn := calendarYearReturn(strategy, year, loc)
		csiReturn := calendarYearReturn(csi300, year, loc)
		result = append(result, service.QuantBacktestAnnualReturn{
			Year:                year,
			TradingDays:         len(strategyPoints),
			StrategyReturnPct:   strategyReturn,
			CSI300ReturnPct:     csiReturn,
			PilotEqualReturnPct: calendarYearReturn(pilot, year, loc),
			ExcessOverCSI300Pct: strategyReturn - csiReturn,
		})
	}
	return result
}

func calendarYearReturn(points []service.QuantSeriesPoint, year int, loc *time.Location) float64 {
	var previous *service.QuantSeriesPoint
	current := make([]service.QuantSeriesPoint, 0)
	for index := range points {
		pointYear := time.Unix(points[index].Time, 0).In(loc).Year()
		if pointYear < year {
			candidate := points[index]
			previous = &candidate
			continue
		}
		if pointYear == year {
			current = append(current, points[index])
		}
	}
	if len(current) < 2 {
		return 0
	}
	base := current[0].Value
	if previous != nil {
		base = previous.Value
	}
	if base == 0 {
		return 0
	}
	return (current[len(current)-1].Value/base - 1) * 100
}

func groupSeriesByYear(points []service.QuantSeriesPoint, loc *time.Location) map[int][]service.QuantSeriesPoint {
	result := make(map[int][]service.QuantSeriesPoint)
	for _, point := range points {
		year := time.Unix(point.Time, 0).In(loc).Year()
		result[year] = append(result[year], point)
	}
	return result
}
