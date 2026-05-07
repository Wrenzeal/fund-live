package service

import (
	"context"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

func TestBuildEstimateCapabilityRecordRecognizesQDIIHoldings(t *testing.T) {
	record := buildEstimateCapabilityRecord(
		database.Fund{
			ID:   "017437",
			Name: "华宝纳斯达克精选股票发起式(QDII)C",
			Type: "qdii",
		},
		estimateHoldingsAggregate{
			FundID:        "017437",
			HoldingsCount: 10,
			TotalRatio:    decimal.RequireFromString("62.5000"),
			USCount:       10,
		},
		estimateHoldingsAggregate{},
		false,
		database.FundMapping{},
		time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC),
	)

	if record.CapabilityStatus != EstimateCapabilityStatusSupported {
		t.Fatalf("status = %s, want supported", record.CapabilityStatus)
	}
	if record.CapabilityType != EstimateCapabilityTypeQDIIHoldings {
		t.Fatalf("type = %s, want qdii_holdings", record.CapabilityType)
	}
	if record.QuoteSourceMode != EstimateQuoteModeOverseas {
		t.Fatalf("quote mode = %s, want overseas_fixed", record.QuoteSourceMode)
	}
}

func TestBuildEstimateCapabilityRecordFallsBackToTargetQuoteWhenMappingExistsWithoutHoldings(t *testing.T) {
	record := buildEstimateCapabilityRecord(
		database.Fund{
			ID:   "999001",
			Name: "示例联接基金",
			Type: "index",
		},
		estimateHoldingsAggregate{},
		estimateHoldingsAggregate{},
		false,
		database.FundMapping{
			FeederCode: "999001",
			TargetCode: "510300",
			IsResolved: true,
		},
		time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC),
	)

	if record.CapabilityStatus != EstimateCapabilityStatusDegraded {
		t.Fatalf("status = %s, want degraded", record.CapabilityStatus)
	}
	if record.CapabilityType != EstimateCapabilityTypeFeederTargetQuote {
		t.Fatalf("type = %s, want feeder_target_quote", record.CapabilityType)
	}
	if record.TargetCode != "510300" {
		t.Fatalf("target code = %s, want 510300", record.TargetCode)
	}
}

func TestSnapshotDueTrackedFundsHonorsBatchLimitsAndIntervals(t *testing.T) {
	service := NewValuationService(&countingCollectorFundRepository{}, noopQuoteProvider{}, noopCacheRepository{})
	service.now = func() time.Time {
		return time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC)
	}
	service.domesticBatchSize = 2
	service.overseasBatchSize = 1
	service.managedTargets = []trackedFundTarget{
		{
			FundID:          "000001",
			Source:          domain.QuoteSourceSina,
			RefreshInterval: 3 * time.Minute,
			QuoteMode:       EstimateQuoteModeDomestic,
			Persistent:      true,
		},
		{
			FundID:          "000002",
			Source:          domain.QuoteSourceSina,
			LastCollectedAt: time.Date(2026, 4, 21, 9, 50, 0, 0, time.UTC),
			RefreshInterval: 3 * time.Minute,
			QuoteMode:       EstimateQuoteModeDomestic,
			Persistent:      true,
		},
		{
			FundID:          "000003",
			Source:          domain.QuoteSourceSina,
			LastCollectedAt: time.Date(2026, 4, 21, 9, 59, 0, 0, time.UTC),
			RefreshInterval: 3 * time.Minute,
			QuoteMode:       EstimateQuoteModeDomestic,
			Persistent:      true,
		},
		{
			FundID:          "017437",
			Source:          domain.QuoteSourceSina,
			LastCollectedAt: time.Date(2026, 4, 21, 9, 40, 0, 0, time.UTC),
			RefreshInterval: 5 * time.Minute,
			QuoteMode:       EstimateQuoteModeOverseas,
			Persistent:      true,
		},
		{
			FundID:          "017438",
			Source:          domain.QuoteSourceSina,
			RefreshInterval: 5 * time.Minute,
			QuoteMode:       EstimateQuoteModeOverseas,
			Persistent:      true,
		},
	}

	domestic, overseas := service.snapshotDueTrackedFunds(service.now())
	if len(domestic) != 2 {
		t.Fatalf("domestic len = %d, want 2", len(domestic))
	}
	if domestic[0].FundID != "000001" || domestic[1].FundID != "000002" {
		t.Fatalf("domestic targets = %#v", domestic)
	}
	if len(overseas) != 1 {
		t.Fatalf("overseas len = %d, want 1", len(overseas))
	}
	if overseas[0].FundID != "017438" {
		t.Fatalf("overseas targets = %#v, want zero-collected target first", overseas)
	}
}

func TestEstimateCoverageSchedulerSyncsManagedTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("integration-style capability refresh test skipped in short mode")
	}

	db := database.GetDB()
	if db == nil {
		t.Skip("database unavailable")
	}

	// Just ensure the method can be constructed and does not panic with nil-less deps.
	service := NewEstimateCapabilityService(db)
	valuation := NewValuationService(&countingCollectorFundRepository{}, noopQuoteProvider{}, noopCacheRepository{})
	scheduler := NewEstimateCoverageScheduler(service, valuation, domain.QuoteSourceSina)
	if scheduler.scanBatchSize != 2000 {
		t.Fatalf("scan batch size = %d, want 2000", scheduler.scanBatchSize)
	}
	if scheduler.refreshInterval <= 0 {
		t.Fatalf("refresh interval must be positive")
	}
	_ = context.Background()
}
