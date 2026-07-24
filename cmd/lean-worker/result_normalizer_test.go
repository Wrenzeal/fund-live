package main

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/service"
)

func TestNormalizeLeanResultBuildsStableComparisonPayload(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	ts := func(year int, month time.Month, day int) int64 {
		return time.Date(year, month, day, 15, 0, 0, 0, loc).Unix()
	}
	statistics, _ := json.Marshal(map[string]string{
		"Net Profit": "12.500%", "Compounding Annual Return": "6.100%", "Drawdown": "8.000%",
		"Sharpe Ratio": "0.75", "Sortino Ratio": "1.10", "Information Ratio": "0.32",
		"Portfolio Turnover": "18.20%", "Total Fees": "¥1234.50", "Total Orders": "42",
		"Start Equity": "1000000", "End Equity": "1125000",
	})
	charts, _ := json.Marshal(map[string]interface{}{
		"Strategy Equity": map[string]interface{}{"series": map[string]interface{}{
			"Equity": map[string]interface{}{"values": [][]interface{}{{ts(2025, 1, 2), 1_000_000}, {ts(2025, 12, 31), 1_100_000}, {ts(2026, 1, 2), 1_080_000}, {ts(2026, 7, 23), 1_125_000}}},
		}},
		"Benchmarks": map[string]interface{}{"series": map[string]interface{}{
			"沪深300": map[string]interface{}{"values": [][]interface{}{{ts(2024, 12, 31), 80}, {ts(2025, 1, 2), 100}, {ts(2025, 12, 31), 105}, {ts(2026, 1, 2), 106}, {ts(2026, 7, 23), 110}}},
			"试点池等权": map[string]interface{}{"values": [][]interface{}{{ts(2025, 1, 2), 100}, {ts(2025, 12, 31), 108}, {ts(2026, 1, 2), 109}, {ts(2026, 7, 23), 115}}},
			"现金":    map[string]interface{}{"values": [][]interface{}{{ts(2025, 1, 2), 100}, {ts(2026, 1, 2), 100}, {ts(2026, 7, 23), 100}}},
		}},
		"Drawdown": map[string]interface{}{"series": map[string]interface{}{
			"Equity Drawdown": map[string]interface{}{"values": [][]interface{}{{ts(2025, 1, 2), 0}, {ts(2025, 12, 31), -3}, {ts(2026, 7, 23), -1}}},
		}},
	})
	tradingDates := map[string]struct{}{
		"2025-01-02": {}, "2025-12-31": {}, "2026-01-02": {}, "2026-07-23": {},
	}
	payload, err := normalizeLeanResult(map[string]json.RawMessage{"statistics": statistics, "charts": charts}, &service.LeanJobManifest{
		Parameters: service.QuantBacktestRequest{StartDate: "2025-01-01", EndDate: "2026-07-23"},
	}, tradingDates)
	if err != nil {
		t.Fatalf("normalizeLeanResult() error = %v", err)
	}
	var result service.QuantNormalizedResult
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatal(err)
	}
	if result.SchemaVersion != "quant-backtest.v1" || result.Summary.TotalOrders != 42 || result.Summary.TotalFeesCNY != 1234.5 {
		t.Fatalf("unexpected summary: %#v", result.Summary)
	}
	if len(result.Series.CSI300) != 4 || result.Series.CSI300[0].Value != 100 {
		t.Fatalf("benchmark was not trimmed and normalized: %#v", result.Series.CSI300)
	}
	if math.Abs(result.Summary.CSI300ReturnPct-10) > 0.0001 || math.Abs(result.Summary.ExcessReturnPct-2.5) > 0.0001 {
		t.Fatalf("unexpected benchmark comparison: %#v", result.Summary)
	}
	if len(result.AnnualReturns) != 2 || result.AnnualReturns[0].Year != 2025 {
		t.Fatalf("unexpected annual returns: %#v", result.AnnualReturns)
	}
	if math.Abs(result.AnnualReturns[0].StrategyReturnPct-10) > 0.0001 {
		t.Fatalf("unexpected first-year return: %#v", result.AnnualReturns[0])
	}
	if math.Abs(result.AnnualReturns[1].StrategyReturnPct-2.272727) > 0.0001 || math.Abs(result.AnnualReturns[1].CSI300ReturnPct-4.761905) > 0.0001 {
		t.Fatalf("later annual return did not use the prior year close: %#v", result.AnnualReturns[1])
	}
}
