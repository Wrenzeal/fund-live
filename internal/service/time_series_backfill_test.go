package service

import (
	"context"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/crawler"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

func mustShanghaiTime(t *testing.T, value string) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, loc)
	if err != nil {
		t.Fatalf("parse time: %v", err)
	}
	return parsed
}

func TestPreferredTimeSeriesDateLunchBreakUsesToday(t *testing.T) {
	svc := &ValuationServiceImpl{}
	now := mustShanghaiTime(t, "2026-03-25 12:02:00")

	got := svc.preferredTimeSeriesDate(now)
	if got.Format("2006-01-02") != "2026-03-25" {
		t.Fatalf("expected today during lunch break, got %s", got.Format("2006-01-02"))
	}
}

func TestPreferredTimeSeriesDateAfterHoursUsesToday(t *testing.T) {
	svc := &ValuationServiceImpl{}
	now := mustShanghaiTime(t, "2026-03-25 15:30:00")

	got := svc.preferredTimeSeriesDate(now)
	if got.Format("2006-01-02") != "2026-03-25" {
		t.Fatalf("expected same trading day after close, got %s", got.Format("2006-01-02"))
	}
}

func TestPreferredTimeSeriesDateCallAuctionUsesPreviousTradingDay(t *testing.T) {
	svc := &ValuationServiceImpl{}
	now := mustShanghaiTime(t, "2026-03-25 09:05:00")

	got := svc.preferredTimeSeriesDate(now)
	if got.Format("2006-01-02") != "2026-03-24" {
		t.Fatalf("expected previous trading day during call auction, got %s", got.Format("2006-01-02"))
	}
}

type historicalFallbackFundRepository struct {
	countingCollectorFundRepository
	pointsByDate map[string][]domain.TimeSeriesPoint
}

func (r *historicalFallbackFundRepository) GetFundByID(ctx context.Context, fundID string) (*domain.Fund, error) {
	return &domain.Fund{
		ID:          fundID,
		NetAssetVal: decimal.RequireFromString("1.0000"),
	}, nil
}

func (r *historicalFallbackFundRepository) GetTimeSeriesByDate(ctx context.Context, fundID string, date time.Time) ([]domain.TimeSeriesPoint, error) {
	key := date.In(tradingLocation()).Format("2006-01-02")
	points := r.pointsByDate[key]
	if len(points) == 0 {
		return nil, nil
	}

	cloned := make([]domain.TimeSeriesPoint, len(points))
	copy(cloned, points)
	return cloned, nil
}

func TestGetIntradayTimeSeriesAfterHoursDoesNotFallbackToPreviousTradingDay(t *testing.T) {
	repo := &historicalFallbackFundRepository{
		pointsByDate: map[string][]domain.TimeSeriesPoint{
			"2026-03-24": {
				{
					Timestamp:     mustShanghaiTime(t, "2026-03-24 09:30:00"),
					ChangePercent: decimal.RequireFromString("1.2300"),
					EstimateNav:   decimal.RequireFromString("1.0123"),
				},
			},
		},
	}

	service := NewValuationService(repo, noopQuoteProvider{}, noopCacheRepository{})
	service.now = func() time.Time {
		return mustShanghaiTime(t, "2026-03-25 15:30:00")
	}

	points, err := service.GetIntradayTimeSeries(context.Background(), "005827")
	if err != nil {
		t.Fatalf("GetIntradayTimeSeries() error = %v", err)
	}
	if len(points) != 0 {
		t.Fatalf("points len = %d, want 0 to avoid previous-day fallback after close", len(points))
	}
}

func TestSelectFiveMinuteMinuteSamplesKeepsHongKongTimeline(t *testing.T) {
	targetDate := mustShanghaiTime(t, "2026-04-08 19:30:00")
	points := []crawler.TencentMinutePoint{
		{Timestamp: mustShanghaiTime(t, "2026-04-08 09:30:00"), Price: decimal.RequireFromString("504.5")},
		{Timestamp: mustShanghaiTime(t, "2026-04-08 09:31:00"), Price: decimal.RequireFromString("505.0")},
		{Timestamp: mustShanghaiTime(t, "2026-04-08 11:30:00"), Price: decimal.RequireFromString("503.0")},
		{Timestamp: mustShanghaiTime(t, "2026-04-08 13:00:00"), Price: decimal.RequireFromString("504.5")},
		{Timestamp: mustShanghaiTime(t, "2026-04-08 09:35:00"), Price: decimal.RequireFromString("504.5")},
		{Timestamp: mustShanghaiTime(t, "2026-04-08 14:55:00"), Price: decimal.RequireFromString("507.8")},
		{Timestamp: mustShanghaiTime(t, "2026-04-08 13:05:00"), Price: decimal.RequireFromString("505.0")},
		{Timestamp: mustShanghaiTime(t, "2026-04-08 15:00:00"), Price: decimal.RequireFromString("508.0")},
		{Timestamp: mustShanghaiTime(t, "2026-04-08 15:01:00"), Price: decimal.RequireFromString("508.2")},
	}

	sessionOpen, samples := selectFiveMinuteMinuteSamples(points, targetDate)
	if sessionOpen == nil {
		t.Fatalf("sessionOpen = nil, want 09:30 sample")
	}
	if got := sessionOpen.Timestamp.Format("15:04"); got != "09:30" {
		t.Fatalf("sessionOpen = %s, want 09:30", got)
	}
	if len(samples) != 5 {
		t.Fatalf("samples len = %d, want 5", len(samples))
	}
	if got := samples[0].Timestamp.Format("15:04"); got != "09:35" {
		t.Fatalf("samples[0] = %s, want 09:35", got)
	}
	if got := samples[1].Timestamp.Format("15:04"); got != "11:30" {
		t.Fatalf("samples[1] = %s, want 11:30", got)
	}
	if got := samples[2].Timestamp.Format("15:04"); got != "13:05" {
		t.Fatalf("samples[2] = %s, want 13:05", got)
	}
	if got := samples[3].Timestamp.Format("15:04"); got != "14:55" {
		t.Fatalf("samples[3] = %s, want 14:55", got)
	}
	if got := samples[4].Timestamp.Format("15:04"); got != "15:00" {
		t.Fatalf("samples[4] = %s, want 15:00", got)
	}
	for _, sample := range samples {
		if got := sample.Timestamp.Format("15:04"); got == "13:00" {
			t.Fatalf("real 13:00 sample should be filtered out")
		}
	}
}

func TestEnsureLunchBreakResumePointDoesNotAdd1300BeforeAfternoonOpen(t *testing.T) {
	points := []domain.TimeSeriesPoint{
		{
			Timestamp:     mustShanghaiTime(t, "2026-04-08 11:25:00"),
			ChangePercent: decimal.RequireFromString("1.5592"),
			EstimateNav:   decimal.RequireFromString("1.7800"),
		},
		{
			Timestamp:     mustShanghaiTime(t, "2026-04-08 11:30:00"),
			ChangePercent: decimal.RequireFromString("1.5009"),
			EstimateNav:   decimal.RequireFromString("1.7790"),
		},
		{
			Timestamp:     mustShanghaiTime(t, "2026-04-08 13:00:00"),
			ChangePercent: decimal.RequireFromString("2.2257"),
			EstimateNav:   decimal.RequireFromString("1.7900"),
		},
	}

	got := ensureLunchBreakResumePoint(points, mustShanghaiTime(t, "2026-04-08 12:15:00"))
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 before 13:00", len(got))
	}
	for _, point := range got {
		if point.Timestamp.Format("15:04") == "13:00" {
			t.Fatalf("13:00 point should be hidden before afternoon open")
		}
	}
}

func TestEnsureLunchBreakResumePointAddsSynthetic1300AtAfternoonOpen(t *testing.T) {
	points := []domain.TimeSeriesPoint{
		{
			Timestamp:     mustShanghaiTime(t, "2026-04-08 11:25:00"),
			ChangePercent: decimal.RequireFromString("1.5592"),
			EstimateNav:   decimal.RequireFromString("1.7800"),
		},
		{
			Timestamp:     mustShanghaiTime(t, "2026-04-08 11:30:00"),
			ChangePercent: decimal.RequireFromString("1.5009"),
			EstimateNav:   decimal.RequireFromString("1.7790"),
		},
	}

	got := ensureLunchBreakResumePoint(points, mustShanghaiTime(t, "2026-04-08 13:00:00"))
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 at 13:00", len(got))
	}
	last := got[2]
	if gotTime := last.Timestamp.Format("15:04"); gotTime != "13:00" {
		t.Fatalf("synthetic point time = %s, want 13:00", gotTime)
	}
	if last.ChangePercent.StringFixed(4) != "1.5009" {
		t.Fatalf("synthetic point change = %s, want 1.5009", last.ChangePercent.StringFixed(4))
	}
}

func TestEnsureLunchBreakResumePointOverwritesExisting1300With1130Value(t *testing.T) {
	points := []domain.TimeSeriesPoint{
		{
			Timestamp:     mustShanghaiTime(t, "2026-04-08 11:30:00"),
			ChangePercent: decimal.RequireFromString("1.5009"),
			EstimateNav:   decimal.RequireFromString("1.7790"),
		},
		{
			Timestamp:     mustShanghaiTime(t, "2026-04-08 13:00:00"),
			ChangePercent: decimal.RequireFromString("2.2257"),
			EstimateNav:   decimal.RequireFromString("1.7900"),
		},
		{
			Timestamp:     mustShanghaiTime(t, "2026-04-08 13:05:00"),
			ChangePercent: decimal.RequireFromString("2.0089"),
			EstimateNav:   decimal.RequireFromString("1.7862"),
		},
	}

	got := ensureLunchBreakResumePoint(points, mustShanghaiTime(t, "2026-04-08 13:10:00"))
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[1].Timestamp.Format("15:04") != "13:00" {
		t.Fatalf("point[1] time = %s, want 13:00", got[1].Timestamp.Format("15:04"))
	}
	if got[1].ChangePercent.StringFixed(4) != "1.5009" {
		t.Fatalf("13:00 change = %s, want 1.5009", got[1].ChangePercent.StringFixed(4))
	}
	if got[2].ChangePercent.StringFixed(4) != "2.0089" {
		t.Fatalf("13:05 change = %s, want 2.0089", got[2].ChangePercent.StringFixed(4))
	}
}

type afterHoursAlignmentFundRepository struct {
	countingCollectorFundRepository
}

func (r *afterHoursAlignmentFundRepository) GetFundByID(ctx context.Context, fundID string) (*domain.Fund, error) {
	return &domain.Fund{
		ID:          fundID,
		Name:        "易方达蓝筹精选混合",
		NetAssetVal: decimal.RequireFromString("1.7526"),
	}, nil
}

func (r *afterHoursAlignmentFundRepository) GetFundHoldings(ctx context.Context, fundID string) ([]domain.StockHolding, error) {
	return []domain.StockHolding{
		{
			StockCode:    "00700",
			StockName:    "腾讯控股",
			Exchange:     domain.ExchangeHK,
			HoldingRatio: decimal.RequireFromString("9.98"),
		},
		{
			StockCode:    "600519",
			StockName:    "贵州茅台",
			Exchange:     domain.ExchangeSH,
			HoldingRatio: decimal.RequireFromString("9.90"),
		},
	}, nil
}

type fixedQuoteProvider struct {
	quotes map[string]domain.StockQuote
}

func (p fixedQuoteProvider) GetRealTimeQuotes(ctx context.Context, stockCodes []string) (map[string]domain.StockQuote, error) {
	result := make(map[string]domain.StockQuote, len(stockCodes))
	for _, code := range stockCodes {
		if quote, ok := p.quotes[code]; ok {
			result[code] = quote
		}
	}
	return result, nil
}

func (p fixedQuoteProvider) GetName() string { return "fixed" }

func TestAlignAfterHoursTimeSeriesWithEstimateReplacesClosingPoint(t *testing.T) {
	repo := &afterHoursAlignmentFundRepository{}
	provider := fixedQuoteProvider{
		quotes: map[string]domain.StockQuote{
			"00700": {
				StockCode:     "00700",
				StockName:     "腾讯控股",
				CurrentPrice:  decimal.RequireFromString("509.5"),
				PrevClose:     decimal.RequireFromString("489.2"),
				ChangePercent: decimal.RequireFromString("4.1496"),
			},
			"600519": {
				StockCode:     "600519",
				StockName:     "贵州茅台",
				CurrentPrice:  decimal.RequireFromString("1465.02"),
				PrevClose:     decimal.RequireFromString("1440.02"),
				ChangePercent: decimal.RequireFromString("1.7361"),
			},
		},
	}
	service := NewValuationService(repo, provider, noopCacheRepository{})

	now := mustShanghaiTime(t, "2026-04-08 19:30:00")
	points := []domain.TimeSeriesPoint{
		{
			Timestamp:     mustShanghaiTime(t, "2026-04-08 14:55:00"),
			ChangePercent: decimal.RequireFromString("1.5946"),
			EstimateNav:   decimal.RequireFromString("1.7805"),
		},
		{
			Timestamp:     mustShanghaiTime(t, "2026-04-08 15:00:00"),
			ChangePercent: decimal.RequireFromString("1.6594"),
			EstimateNav:   decimal.RequireFromString("1.7817"),
		},
	}

	aligned := service.alignAfterHoursTimeSeriesWithEstimate(context.Background(), "005827", now, now, points)
	if len(aligned) != 2 {
		t.Fatalf("aligned len = %d, want 2", len(aligned))
	}

	got := aligned[1].ChangePercent.StringFixed(4)
	if got == "1.6594" {
		t.Fatalf("closing point was not updated, still %s", got)
	}
	if got != "2.9477" {
		t.Fatalf("closing point = %s, want 2.9477 based on current estimate chain", got)
	}
}
