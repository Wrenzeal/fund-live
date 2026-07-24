package service

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
)

func TestNormalizeQuantBacktestRequestAppliesPointInTimeDefaults(t *testing.T) {
	request, err := NormalizeQuantBacktestRequest(QuantBacktestRequest{StartDate: "2021-01-01", EndDate: "2026-01-01"})
	if err != nil {
		t.Fatalf("NormalizeQuantBacktestRequest() error = %v", err)
	}
	if request.TopN != 5 || request.UniverseVersion != QuantUniversePilotV1 || request.SignalMode != QuantSignalModeHistoryProxy {
		t.Fatalf("unexpected defaults: %#v", request)
	}
	if request.MinimumListingDays != 120 || request.MinimumAverageAmount.String() != "20000000" {
		t.Fatalf("unexpected eligibility defaults: %#v", request)
	}
	if request.RebalanceFrequency != QuantRebalanceWeekly || request.ExitRank != 5 || request.WeightingMethod != QuantWeightEqual || request.WeakMarketExposureBPS != 10_000 {
		t.Fatalf("unexpected strategy defaults: %#v", request)
	}
}

func TestNormalizeQuantBacktestRequestRejectsInvalidRange(t *testing.T) {
	_, err := NormalizeQuantBacktestRequest(QuantBacktestRequest{StartDate: "2026-01-02", EndDate: "2026-01-01"})
	if err == nil {
		t.Fatal("expected invalid date range error")
	}
}

func TestNormalizeCurrentEventMetadataSetsPointInTimeFields(t *testing.T) {
	now := time.Date(2026, 7, 23, 15, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	events := normalizeCurrentEventMetadata([]domain.FundAnalysisEventImpact{{
		Code: "notice-1", Title: "业绩预告", TargetScope: "holding", Impact: "positive", SourceName: "巨潮资讯", SourcePublishedAt: "2026-07-22",
	}}, now)
	if len(events) != 1 {
		t.Fatalf("event count = %d", len(events))
	}
	event := events[0]
	if event.EventID == "" || event.EventType != "earnings_forecast" || event.EventStatus != "disclosed" {
		t.Fatalf("unexpected normalized event: %#v", event)
	}
	if event.KnownAt == nil || !event.KnownAt.Equal(now) || event.KnownAtBasis != "first_seen" {
		t.Fatalf("unexpected known-at fields: %#v", event)
	}
	if event.SourceTier != "official" || event.AnnouncedAt == nil {
		t.Fatalf("unexpected source metadata: %#v", event)
	}
}

func TestShadowEventIntelligenceDoesNotMutateProductionScore(t *testing.T) {
	now := time.Date(2026, 7, 23, 15, 30, 0, 0, time.UTC)
	events := normalizeCurrentEventMetadata([]domain.FundAnalysisEventImpact{{
		Code: "positive", Title: "回购进展", TargetScope: "holding", Impact: "positive", Strength: "high", SourceName: "巨潮资讯",
	}}, now)
	productionScore := 61.2
	intelligence := buildShadowEventIntelligence(now, events, productionScore, 52)
	if productionScore != 61.2 {
		t.Fatalf("production score mutated: %f", productionScore)
	}
	if intelligence.Mode != "shadow" || !intelligence.ShadowDelta.IsPositive() {
		t.Fatalf("unexpected intelligence: %#v", intelligence)
	}
}

func TestSameISOWeek(t *testing.T) {
	friday := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)
	if !sameISOWeek(friday, friday.AddDate(0, 0, -1)) {
		t.Fatal("Thursday and Friday should share ISO week")
	}
	if sameISOWeek(friday, friday.AddDate(0, 0, 3)) {
		t.Fatal("Friday and next Monday should not share ISO week")
	}
}

func TestQuantInstrumentUpsertTargetsSymbolPrimaryKey(t *testing.T) {
	conflict := quantInstrumentUpsert()
	if len(conflict.Columns) != 1 || conflict.Columns[0].Name != "symbol" {
		t.Fatalf("unexpected conflict target: %#v", conflict.Columns)
	}
	if conflict.DoNothing || len(conflict.DoUpdates) == 0 {
		t.Fatalf("unexpected conflict action: %#v", conflict)
	}
}

func TestRiskV1RequestsAreFixedAndIncremental(t *testing.T) {
	variants, err := RiskV1Requests(QuantBacktestRequest{StartDate: "2022-01-19", EndDate: "2026-07-23"})
	if err != nil {
		t.Fatalf("RiskV1Requests() error = %v", err)
	}
	if len(variants) != 4 {
		t.Fatalf("len(variants) = %d, want 4", len(variants))
	}
	if got := quantBacktestStrategy(variants[0].Request); got != QuantStrategyTop5Weekly {
		t.Fatalf("variant A strategy = %s", got)
	}
	if got := quantBacktestStrategy(variants[1].Request); got != QuantStrategyTop5Buffer {
		t.Fatalf("variant B strategy = %s", got)
	}
	if got := quantBacktestStrategy(variants[2].Request); got != QuantStrategyTop5LowTurnover {
		t.Fatalf("variant C strategy = %s", got)
	}
	if got := quantBacktestStrategy(variants[3].Request); got != QuantStrategyTop5RiskControl {
		t.Fatalf("variant D strategy = %s", got)
	}
	if QuantBacktestImplementation != "risk-v1.2" {
		t.Fatalf("unexpected strategy implementation version: %s", QuantBacktestImplementation)
	}
	d := variants[3].Request
	if d.RebalanceFrequency != QuantRebalanceMonthly || d.ExitRank != 8 || d.MinimumWeightChangeBPS != 250 || d.WeightingMethod != QuantWeightInverseVolatility || d.TrendFilterDays != 120 || d.WeakMarketExposureBPS != 5_000 {
		t.Fatalf("unexpected D parameters: %#v", d)
	}
}

func TestNormalizeQuantBacktestRequestRejectsInvalidStrategyParameters(t *testing.T) {
	_, err := NormalizeQuantBacktestRequest(QuantBacktestRequest{
		StartDate: "2022-01-19", EndDate: "2026-07-23", TopN: 5, ExitRank: 4,
	})
	if err == nil {
		t.Fatal("expected exit_rank validation error")
	}
	_, err = NormalizeQuantBacktestRequest(QuantBacktestRequest{
		StartDate: "2022-01-19", EndDate: "2026-07-23", WeightingMethod: "black_box",
	})
	if err == nil {
		t.Fatal("expected weighting_method validation error")
	}
}

func TestWriteLeanSignalCSVUsesMonthlyPeriodEnd(t *testing.T) {
	directory := t.TempDir()
	path := directory + "/signals.csv"
	signals := []database.QuantSignalHistory{
		{SignalDate: time.Date(2026, 1, 30, 0, 0, 0, 0, time.UTC)},
		{SignalDate: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)},
	}
	if err := writeLeanSignalCSV(path, signals, QuantRebalanceMonthly); err != nil {
		t.Fatalf("writeLeanSignalCSV() error = %v", err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
	if len(lines) != 2 || !strings.HasSuffix(lines[0], ",1") || !strings.HasSuffix(lines[1], ",1") {
		t.Fatalf("unexpected monthly signals: %q", string(payload))
	}
}
