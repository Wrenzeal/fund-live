package service

import (
	"testing"
	"time"

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
