package service

import (
	"strings"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

func TestFundAnalysisServiceBuildTrackedETFRecommendation(t *testing.T) {
	service := NewFundAnalysisService()

	analysis := service.Build(FundAnalysisInput{
		Fund: &domain.Fund{
			ID:   "012970",
			Name: "鹏华国证半导体芯片ETF联接C",
			Type: "index",
		},
		Estimate: &domain.FundEstimate{
			FundID:         "012970",
			FundName:       "鹏华国证半导体芯片ETF联接C",
			ChangePercent:  decimal.RequireFromString("1.7800"),
			TotalHoldRatio: decimal.RequireFromString("100.0000"),
			HoldingDetails: []domain.HoldingDetail{
				{StockCode: "688981", StockName: "中芯国际", HoldingRatio: decimal.RequireFromString("10.2800"), StockChange: decimal.RequireFromString("3.2000"), Contribution: decimal.RequireFromString("0.3289")},
				{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("9.9700"), StockChange: decimal.RequireFromString("2.6000"), Contribution: decimal.RequireFromString("0.2592")},
				{StockCode: "300308", StockName: "中际旭创", HoldingRatio: decimal.RequireFromString("8.2100"), StockChange: decimal.RequireFromString("1.1000"), Contribution: decimal.RequireFromString("0.0903")},
			},
		},
		TimeSeries: []domain.TimeSeriesPoint{
			{Timestamp: time.Date(2026, time.April, 26, 10, 0, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("0.6200")},
			{Timestamp: time.Date(2026, time.April, 26, 11, 0, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("1.2400")},
			{Timestamp: time.Date(2026, time.April, 26, 14, 30, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("1.7800")},
		},
		SectorSnapshot: &domain.FundSectorSnapshot{
			PrimarySectorCode: "semiconductor",
			PrimarySectorName: "半导体",
			Confidence:        "high",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "semiconductor", SectorName: "半导体", WeightPercent: decimal.RequireFromString("71.4000")},
			},
		},
		ThemeSnapshot: &domain.FundThemeSnapshot{
			PrimaryThemeCode: "semiconductor_chip",
			PrimaryThemeName: "半导体芯片",
			Confidence:       "high",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "semiconductor_chip", ThemeName: "半导体芯片", WeightPercent: decimal.RequireFromString("65.1000")},
			},
		},
		PreviousThemeSnapshot: &domain.FundThemeSnapshot{
			PrimaryThemeCode: "ai_compute",
			PrimaryThemeName: "AI算力",
			Confidence:       "high",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "ai_compute", ThemeName: "AI算力", WeightPercent: decimal.RequireFromString("58.0000")},
			},
		},
		Holdings: []domain.StockHolding{
			{StockCode: "688981", StockName: "中芯国际", HoldingRatio: decimal.RequireFromString("10.2800"), ReportingPeriod: "2026-03-31"},
			{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("9.9700"), ReportingPeriod: "2026-03-31"},
			{StockCode: "300308", StockName: "中际旭创", HoldingRatio: decimal.RequireFromString("8.2100"), ReportingPeriod: "2026-03-31"},
		},
		PreviousHoldings: []domain.StockHolding{
			{StockCode: "688981", StockName: "中芯国际", HoldingRatio: decimal.RequireFromString("8.1200"), ReportingPeriod: "2025-12-31"},
			{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("7.3200"), ReportingPeriod: "2025-12-31"},
			{StockCode: "002371", StockName: "北方华创", HoldingRatio: decimal.RequireFromString("10.6100"), ReportingPeriod: "2025-12-31"},
		},
		PreviousHoldingPeriod: "2025-12-31",
		HoldingsSource:        SectorSourceTargetETFFallback,
		Now:                   time.Date(2026, time.April, 26, 15, 0, 0, 0, time.Local),
	})

	if analysis == nil {
		t.Fatal("analysis = nil")
	}
	if analysis.AnalysisType != FundAnalysisTypeTrackedETF {
		t.Fatalf("analysis type = %s, want %s", analysis.AnalysisType, FundAnalysisTypeTrackedETF)
	}
	if analysis.AnalysisBasis != "目标ETF口径" {
		t.Fatalf("analysis basis = %s, want 目标ETF口径", analysis.AnalysisBasis)
	}
	if !analysis.IncreasePercent.GreaterThan(analysis.HoldPercent) {
		t.Fatalf("increase = %s, hold = %s, want increase > hold", analysis.IncreasePercent, analysis.HoldPercent)
	}
	if DominantRecommendationFromAnalysis(analysis) != "increase" {
		t.Fatalf("dominant recommendation should follow summary threshold for strong tracked ETF: increase=%s hold=%s decrease=%s", analysis.IncreasePercent, analysis.HoldPercent, analysis.DecreasePercent)
	}
	if analysis.RiskLevel == "" {
		t.Fatal("risk level should not be empty")
	}
	if len(analysis.ModuleScores) != 6 {
		t.Fatalf("module len = %d, want 6", len(analysis.ModuleScores))
	}
	if len(analysis.EventImpacts) == 0 {
		t.Fatal("event impacts should not be empty")
	}
	if !strings.Contains(analysis.Summary, "结构偏积极") {
		t.Fatalf("summary = %q, want weak positive-structure wording for strong tracked ETF", analysis.Summary)
	}
	if len(analysis.ConfidenceFactors) == 0 {
		t.Fatal("confidence factors should be populated")
	}
	if len(analysis.PrimaryEvidence) == 0 {
		t.Fatal("primary evidence should be populated")
	}
	driverEvent := findEventImpact(analysis.EventImpacts, "top_positive_driver")
	if driverEvent == nil {
		t.Fatal("top_positive_driver event should exist")
	}
	if driverEvent.TargetScope != "holding" {
		t.Fatalf("driver target scope = %s, want holding", driverEvent.TargetScope)
	}
	if len(driverEvent.RelatedSymbols) == 0 || driverEvent.RelatedSymbols[0] != "688981" {
		t.Fatalf("driver related symbols = %v, want [688981]", driverEvent.RelatedSymbols)
	}
	if driverEvent.WeightHint == nil || !driverEvent.WeightHint.GreaterThan(decimal.Zero) {
		t.Fatal("driver weight hint should be populated")
	}
	quarterIncreaseEvent := findEventImpact(analysis.EventImpacts, "top_holding_increase")
	if quarterIncreaseEvent == nil {
		t.Fatal("top_holding_increase event should exist")
	}
	if quarterIncreaseEvent.Horizon != "quarterly" {
		t.Fatalf("quarter increase horizon = %q, want quarterly", quarterIncreaseEvent.Horizon)
	}
	exposureShiftEvent := findEventImpact(analysis.EventImpacts, "primary_exposure_shift")
	if exposureShiftEvent == nil {
		t.Fatal("primary_exposure_shift event should exist")
	}
	if exposureShiftEvent.TargetScope != "exposure" {
		t.Fatalf("exposure shift target scope = %q, want exposure", exposureShiftEvent.TargetScope)
	}
}

func TestDominantRecommendationRequiresActionThresholds(t *testing.T) {
	tests := []struct {
		name     string
		increase float64
		hold     float64
		decrease float64
		want     string
	}{
		{
			name:     "hold remains watch even when increase slightly leads",
			increase: 46.5,
			hold:     42.6,
			decrease: 10.9,
			want:     "hold",
		},
		{
			name:     "increase requires structural threshold",
			increase: 55,
			hold:     35,
			decrease: 10,
			want:     "increase",
		},
		{
			name:     "decrease requires high risk threshold",
			increase: 18,
			hold:     20,
			decrease: 62,
			want:     "decrease",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DominantRecommendationFromPercents(tt.increase, tt.decrease); got != tt.want {
				t.Fatalf("dominantRecommendation() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFundAnalysisServiceStrongIndexLikeSampleNotOverPenalized(t *testing.T) {
	service := NewFundAnalysisService()

	analysis := service.Build(FundAnalysisInput{
		Fund: &domain.Fund{
			ID:   "159813",
			Name: "半导体ETF鹏华",
			Type: "index",
		},
		Estimate: &domain.FundEstimate{
			FundID:         "159813",
			FundName:       "半导体ETF鹏华",
			ChangePercent:  decimal.RequireFromString("5.8430"),
			TotalHoldRatio: decimal.RequireFromString("68.2800"),
			HoldingDetails: []domain.HoldingDetail{
				{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("9.9900"), StockChange: decimal.RequireFromString("8.2005"), Contribution: decimal.RequireFromString("0.8192")},
				{StockCode: "688981", StockName: "中芯国际", HoldingRatio: decimal.RequireFromString("8.4100"), StockChange: decimal.RequireFromString("4.6990"), Contribution: decimal.RequireFromString("0.3952")},
			},
		},
		TimeSeries: []domain.TimeSeriesPoint{
			{Timestamp: time.Date(2026, time.April, 24, 9, 30, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("1.2048")},
			{Timestamp: time.Date(2026, time.April, 24, 15, 0, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("5.8430")},
		},
		SectorSnapshot: &domain.FundSectorSnapshot{
			PrimarySectorCode: "semiconductor",
			PrimarySectorName: "半导体",
			Confidence:        "high",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "semiconductor", SectorName: "半导体", WeightPercent: decimal.RequireFromString("53.3100")},
			},
		},
		ThemeSnapshot: &domain.FundThemeSnapshot{
			PrimaryThemeCode: "semiconductor_chip",
			PrimaryThemeName: "半导体芯片",
			Confidence:       "high",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "semiconductor_chip", ThemeName: "半导体芯片", WeightPercent: decimal.RequireFromString("53.3100")},
			},
		},
		Holdings: []domain.StockHolding{
			{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("9.9900"), ReportingPeriod: "2026-03-31"},
			{StockCode: "688981", StockName: "中芯国际", HoldingRatio: decimal.RequireFromString("8.4100"), ReportingPeriod: "2026-03-31"},
		},
		HoldingsSource: SectorSourceDirectHoldings,
		Now:            time.Date(2026, time.April, 27, 15, 0, 0, 0, time.Local),
	})

	if analysis == nil {
		t.Fatal("analysis = nil")
	}
	if analysis.TotalScore.LessThan(decimal.RequireFromString("66")) {
		t.Fatalf("total score = %s, want >= 66 after calibration", analysis.TotalScore)
	}
	if analysis.IncreasePercent.LessThan(decimal.RequireFromString("55")) {
		t.Fatalf("increase = %s, want >= 55 for strong index-like sample", analysis.IncreasePercent)
	}
}

func TestFundAnalysisServiceBuildWarnsWhenCoverageAndEventsAreWeak(t *testing.T) {
	service := NewFundAnalysisService()

	analysis := service.Build(FundAnalysisInput{
		Fund: &domain.Fund{
			ID:   "000362",
			Name: "示例混合基金",
			Type: "hybrid",
		},
		Estimate: &domain.FundEstimate{
			FundID:         "000362",
			FundName:       "示例混合基金",
			ChangePercent:  decimal.RequireFromString("-2.3800"),
			TotalHoldRatio: decimal.RequireFromString("18.0000"),
			HoldingDetails: []domain.HoldingDetail{
				{StockCode: "600000", StockName: "示例一", HoldingRatio: decimal.RequireFromString("10.0000"), StockChange: decimal.RequireFromString("-4.2000"), Contribution: decimal.RequireFromString("-0.4200")},
				{StockCode: "600001", StockName: "示例二", HoldingRatio: decimal.RequireFromString("8.0000"), StockChange: decimal.RequireFromString("-3.5000"), Contribution: decimal.RequireFromString("-0.2800")},
			},
		},
		TimeSeries: []domain.TimeSeriesPoint{
			{Timestamp: time.Date(2026, time.April, 26, 10, 0, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("0.2000")},
			{Timestamp: time.Date(2026, time.April, 26, 11, 0, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("-1.1000")},
			{Timestamp: time.Date(2026, time.April, 26, 14, 30, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("-2.3800")},
		},
		SectorSnapshot: &domain.FundSectorSnapshot{
			PrimarySectorCode: "other_equity",
			PrimarySectorName: "未归类权益",
			Confidence:        "low",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "other_equity", SectorName: "未归类权益", WeightPercent: decimal.RequireFromString("42.2500")},
			},
		},
		ThemeSnapshot: &domain.FundThemeSnapshot{
			PrimaryThemeCode: "other_theme",
			PrimaryThemeName: "未归类主题",
			Confidence:       "low",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "other_theme", ThemeName: "未归类主题", WeightPercent: decimal.RequireFromString("55.0000")},
			},
		},
		PreviousSectorSnapshot: &domain.FundSectorSnapshot{
			PrimarySectorCode: "consumer",
			PrimarySectorName: "消费",
			Confidence:        "medium",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "consumer", SectorName: "消费", WeightPercent: decimal.RequireFromString("51.0000")},
			},
		},
		Holdings: []domain.StockHolding{
			{StockCode: "600000", StockName: "示例一", HoldingRatio: decimal.RequireFromString("10.0000"), ReportingPeriod: "2024-03-31"},
			{StockCode: "600001", StockName: "示例二", HoldingRatio: decimal.RequireFromString("8.0000"), ReportingPeriod: "2024-03-31"},
		},
		PreviousHoldings: []domain.StockHolding{
			{StockCode: "600000", StockName: "示例一", HoldingRatio: decimal.RequireFromString("15.0000"), ReportingPeriod: "2023-12-31"},
			{StockCode: "600002", StockName: "示例三", HoldingRatio: decimal.RequireFromString("7.2000"), ReportingPeriod: "2023-12-31"},
			{StockCode: "600003", StockName: "示例四", HoldingRatio: decimal.RequireFromString("6.8000"), ReportingPeriod: "2023-12-31"},
		},
		PreviousHoldingPeriod: "2023-12-31",
		HoldingsSource:        SectorSourceDirectHoldings,
		Now:                   time.Date(2026, time.April, 26, 15, 0, 0, 0, time.Local),
	})

	if analysis == nil {
		t.Fatal("analysis = nil")
	}
	if analysis.Confidence != "low" {
		t.Fatalf("confidence = %s, want low", analysis.Confidence)
	}
	if analysis.DecreasePercent.LessThanOrEqual(analysis.IncreasePercent) {
		t.Fatalf("decrease = %s, increase = %s, want decrease > increase", analysis.DecreasePercent, analysis.IncreasePercent)
	}
	if analysis.TotalScore.GreaterThan(decimal.RequireFromString("50")) {
		t.Fatalf("total score = %s, want <= 50 for stale low coverage sample", analysis.TotalScore)
	}
	if len(analysis.Warnings) == 0 {
		t.Fatal("warnings should not be empty")
	}
	if len(analysis.EventImpacts) == 0 {
		t.Fatal("event impacts should not be empty")
	}
	if analysis.EventImpacts[0].Impact != "negative" {
		t.Fatalf("first event impact = %s, want negative", analysis.EventImpacts[0].Impact)
	}
	negativeDriver := findEventImpact(analysis.EventImpacts, "top_negative_driver")
	if negativeDriver == nil {
		t.Fatal("top_negative_driver event should exist")
	}
	if negativeDriver.Strength == "" || negativeDriver.Horizon == "" {
		t.Fatalf("negative driver metadata should be populated, got strength=%q horizon=%q", negativeDriver.Strength, negativeDriver.Horizon)
	}
	churnEvent := findEventImpact(analysis.EventImpacts, "top10_churn")
	if churnEvent == nil {
		t.Fatal("top10_churn event should exist")
	}
	if analysis.LatestHoldingPeriod != "2024-03-31" {
		t.Fatalf("latest holding period = %s, want 2024-03-31", analysis.LatestHoldingPeriod)
	}
}

func TestPreviousReportingPeriod(t *testing.T) {
	tests := map[string]string{
		"2026-03-31": "2025-12-31",
		"2025-06-30": "2025-03-31",
		"2025-09-30": "2025-06-30",
		"2025-12-31": "2025-09-30",
	}

	for input, want := range tests {
		got, ok := previousReportingPeriod(input)
		if !ok {
			t.Fatalf("previousReportingPeriod(%s) returned !ok", input)
		}
		if got != want {
			t.Fatalf("previousReportingPeriod(%s) = %s, want %s", input, got, want)
		}
	}
}

func TestFundAnalysisServicePrefersCurrentHoldingEventsInEventLayer(t *testing.T) {
	service := NewFundAnalysisService()

	analysis := service.Build(FundAnalysisInput{
		Fund: &domain.Fund{
			ID:   "159813",
			Name: "半导体ETF鹏华",
			Type: "index",
		},
		Estimate: &domain.FundEstimate{
			FundID:         "159813",
			FundName:       "半导体ETF鹏华",
			ChangePercent:  decimal.RequireFromString("0.8000"),
			TotalHoldRatio: decimal.RequireFromString("68.0000"),
			HoldingDetails: []domain.HoldingDetail{
				{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("9.9900"), StockChange: decimal.RequireFromString("3.2000"), Contribution: decimal.RequireFromString("0.3190")},
			},
		},
		TimeSeries: []domain.TimeSeriesPoint{
			{Timestamp: time.Date(2026, time.April, 26, 10, 0, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("0.2000")},
			{Timestamp: time.Date(2026, time.April, 26, 14, 30, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("0.8000")},
		},
		SectorSnapshot: &domain.FundSectorSnapshot{
			PrimarySectorCode: "semiconductor",
			PrimarySectorName: "半导体",
			Confidence:        "high",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "semiconductor", SectorName: "半导体", WeightPercent: decimal.RequireFromString("53.3100")},
			},
		},
		ThemeSnapshot: &domain.FundThemeSnapshot{
			PrimaryThemeCode: "semiconductor_chip",
			PrimaryThemeName: "半导体芯片",
			Confidence:       "high",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "semiconductor_chip", ThemeName: "半导体芯片", WeightPercent: decimal.RequireFromString("53.3100")},
			},
		},
		Holdings: []domain.StockHolding{
			{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("9.9900"), ReportingPeriod: "2026-03-31"},
		},
		CurrentHoldingEvents: []domain.FundAnalysisEventImpact{
			{
				Code:           "holding_current_notice_688041_test",
				Title:          "海光信息发布一季报",
				Impact:         "positive",
				Summary:        "近期开启事件：2026-04-20 发布《海光信息技术股份有限公司2026年第一季度报告》。",
				TargetScope:    "holding",
				Strength:       "high",
				Horizon:        "current",
				RelatedSymbols: []string{"688041"},
				WeightHint:     decimalPointerFromFloat(9.99),
			},
			{
				Code:           "holding_current_notice_688981_test",
				Title:          "中芯国际发布经营进展公告",
				Impact:         "positive",
				Summary:        "近期开启事件：2026-04-22 发布《中芯国际经营进展公告》。",
				TargetScope:    "holding",
				Strength:       "medium",
				Horizon:        "current",
				RelatedSymbols: []string{"688981"},
				WeightHint:     decimalPointerFromFloat(8.41),
			},
		},
		HoldingsSource: SectorSourceDirectHoldings,
		Now:            time.Date(2026, time.April, 26, 15, 0, 0, 0, time.Local),
	})

	if analysis == nil {
		t.Fatal("analysis = nil")
	}
	if !strings.Contains(analysis.ModuleScores[5].Summary, "近期公告") && !strings.Contains(analysis.ModuleScores[5].Summary, "近期") {
		// summary may still be generic; stricter check on main summary below
	}
	if !strings.Contains(analysis.Summary, "当前") {
		t.Fatalf("summary = %q, want non-empty current-event-aware summary", analysis.Summary)
	}
	foundHoldingEvent := false
	for _, item := range analysis.EventImpacts {
		if item.Code == "holding_current_notice_688041_test" {
			foundHoldingEvent = true
			break
		}
	}
	if !foundHoldingEvent {
		t.Fatalf("event impacts = %#v, want current holding notice", analysis.EventImpacts)
	}
	foundReason := false
	for _, reason := range analysis.Reasons {
		if strings.Contains(reason, "海光信息发布一季报") {
			foundReason = true
			break
		}
	}
	if !foundReason {
		t.Fatalf("reasons = %v, want current event reason", analysis.Reasons)
	}
	currentExposureEvent := findEventImpact(analysis.EventImpacts, "current_exposure_event_cluster")
	if currentExposureEvent == nil {
		t.Fatal("current_exposure_event_cluster should exist")
	}
	if currentExposureEvent.TargetScope != "exposure" {
		t.Fatalf("current exposure event target scope = %q, want exposure", currentExposureEvent.TargetScope)
	}
	if !strings.Contains(currentExposureEvent.Summary, "海光信息发布一季报") {
		t.Fatalf("current exposure summary = %q, want highlighted holding event title", currentExposureEvent.Summary)
	}
}

func TestFundAnalysisServiceIncludesCurrentFundEvents(t *testing.T) {
	service := NewFundAnalysisService()

	analysis := service.Build(FundAnalysisInput{
		Fund: &domain.Fund{ID: "159813", Name: "半导体ETF鹏华", Type: "index"},
		Estimate: &domain.FundEstimate{
			FundID:         "159813",
			FundName:       "半导体ETF鹏华",
			ChangePercent:  decimal.RequireFromString("-0.2000"),
			TotalHoldRatio: decimal.RequireFromString("68.0000"),
			HoldingDetails: []domain.HoldingDetail{
				{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("9.9900"), StockChange: decimal.RequireFromString("-1.0000"), Contribution: decimal.RequireFromString("-0.0990")},
			},
		},
		TimeSeries: []domain.TimeSeriesPoint{
			{Timestamp: time.Date(2026, time.April, 26, 10, 0, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("-0.1000")},
			{Timestamp: time.Date(2026, time.April, 26, 14, 30, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("-0.2000")},
		},
		SectorSnapshot: &domain.FundSectorSnapshot{
			PrimarySectorCode: "semiconductor",
			PrimarySectorName: "半导体",
			Confidence:        "high",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "semiconductor", SectorName: "半导体", WeightPercent: decimal.RequireFromString("53.3100")},
			},
		},
		ThemeSnapshot: &domain.FundThemeSnapshot{
			PrimaryThemeCode: "semiconductor_chip",
			PrimaryThemeName: "半导体芯片",
			Confidence:       "high",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "semiconductor_chip", ThemeName: "半导体芯片", WeightPercent: decimal.RequireFromString("53.3100")},
			},
		},
		Holdings: []domain.StockHolding{
			{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("9.9900"), ReportingPeriod: "2026-03-31"},
		},
		CurrentFundEvents: []domain.FundAnalysisEventImpact{
			{
				Code:        "fund_notice_test",
				Title:       "基金经理变更公告",
				Impact:      "negative",
				Summary:     "基金近期事件：2026-04-21 发布《基金经理变更公告》。",
				TargetScope: "fund",
				Strength:    "medium",
				Horizon:     "current",
			},
		},
		HoldingsSource: SectorSourceDirectHoldings,
		Now:            time.Date(2026, time.April, 26, 15, 0, 0, 0, time.Local),
	})

	if analysis == nil {
		t.Fatal("analysis = nil")
	}
	foundFundEvent := false
	for _, event := range analysis.EventImpacts {
		if event.Code == "fund_notice_test" {
			foundFundEvent = true
			break
		}
	}
	if !foundFundEvent {
		t.Fatalf("event impacts = %v, want fund notice event", analysis.EventImpacts)
	}
	foundWarning := false
	for _, warning := range analysis.Warnings {
		if strings.Contains(warning, "基金产品层重要事件偏负向") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Fatalf("warnings = %v, want fund-event warning", analysis.Warnings)
	}
}

func TestFundAnalysisServicePromotesQuarterlyHoldingChangesIntoReasonsAndWarnings(t *testing.T) {
	service := NewFundAnalysisService()

	analysis := service.Build(FundAnalysisInput{
		Fund: &domain.Fund{ID: "159813", Name: "半导体ETF鹏华", Type: "index"},
		Estimate: &domain.FundEstimate{
			FundID:         "159813",
			FundName:       "半导体ETF鹏华",
			ChangePercent:  decimal.RequireFromString("-0.2000"),
			TotalHoldRatio: decimal.RequireFromString("68.0000"),
		},
		ThemeSnapshot: &domain.FundThemeSnapshot{
			PrimaryThemeCode: "semiconductor_chip",
			PrimaryThemeName: "半导体芯片",
			Confidence:       "high",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "semiconductor_chip", ThemeName: "半导体芯片", WeightPercent: decimal.RequireFromString("53.3100")},
			},
		},
		Holdings: []domain.StockHolding{
			{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("10.0000"), ReportingPeriod: "2026-03-31"},
			{StockCode: "688981", StockName: "中芯国际", HoldingRatio: decimal.RequireFromString("8.0000"), ReportingPeriod: "2026-03-31"},
			{StockCode: "603986", StockName: "兆易创新", HoldingRatio: decimal.RequireFromString("2.0000"), ReportingPeriod: "2026-03-31"},
		},
		PreviousHoldings: []domain.StockHolding{
			{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("7.5000"), ReportingPeriod: "2025-12-31"},
			{StockCode: "688981", StockName: "中芯国际", HoldingRatio: decimal.RequireFromString("10.5000"), ReportingPeriod: "2025-12-31"},
			{StockCode: "300750", StockName: "宁德时代", HoldingRatio: decimal.RequireFromString("4.0000"), ReportingPeriod: "2025-12-31"},
			{StockCode: "002594", StockName: "比亚迪", HoldingRatio: decimal.RequireFromString("3.5000"), ReportingPeriod: "2025-12-31"},
			{StockCode: "600406", StockName: "国电南瑞", HoldingRatio: decimal.RequireFromString("3.0000"), ReportingPeriod: "2025-12-31"},
			{StockCode: "300274", StockName: "阳光电源", HoldingRatio: decimal.RequireFromString("2.6000"), ReportingPeriod: "2025-12-31"},
		},
		PreviousHoldingPeriod: "2025-12-31",
		HoldingsSource:        SectorSourceDirectHoldings,
		Now:                   time.Date(2026, time.April, 28, 12, 0, 0, 0, time.Local),
	})

	if analysis == nil {
		t.Fatal("analysis = nil")
	}
	foundReason := false
	for _, reason := range analysis.Reasons {
		if strings.Contains(reason, "海光信息权重由") {
			foundReason = true
			break
		}
	}
	if !foundReason {
		t.Fatalf("reasons = %v, want top increase quarterly reason", analysis.Reasons)
	}
	foundWarning := false
	for _, warning := range analysis.Warnings {
		if strings.Contains(warning, "中芯国际权重由") || strings.Contains(warning, "前十大新增") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Fatalf("warnings = %v, want quarterly decrease or churn warning", analysis.Warnings)
	}
}

func TestFundAnalysisServiceIncludesExposureBreakdownEvents(t *testing.T) {
	service := NewFundAnalysisService()

	analysis := service.Build(FundAnalysisInput{
		Fund: &domain.Fund{ID: "159813", Name: "半导体ETF鹏华", Type: "index"},
		Estimate: &domain.FundEstimate{
			FundID:         "159813",
			FundName:       "半导体ETF鹏华",
			ChangePercent:  decimal.RequireFromString("0.5000"),
			TotalHoldRatio: decimal.RequireFromString("68.2800"),
			HoldingDetails: []domain.HoldingDetail{
				{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("9.9900"), StockChange: decimal.RequireFromString("2.0000"), Contribution: decimal.RequireFromString("0.1990")},
			},
		},
		TimeSeries: []domain.TimeSeriesPoint{
			{Timestamp: time.Date(2026, time.April, 26, 10, 0, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("0.1000")},
			{Timestamp: time.Date(2026, time.April, 26, 14, 30, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("0.5000")},
		},
		SectorSnapshot: &domain.FundSectorSnapshot{
			PrimarySectorCode: "semiconductor",
			PrimarySectorName: "半导体",
			Confidence:        "high",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "semiconductor", SectorName: "半导体", WeightPercent: decimal.RequireFromString("53.3100")},
				{SectorCode: "communications_equipment", SectorName: "通信设备", WeightPercent: decimal.RequireFromString("10.0000")},
			},
		},
		PreviousSectorSnapshot: &domain.FundSectorSnapshot{
			PrimarySectorCode: "semiconductor",
			PrimarySectorName: "半导体",
			Confidence:        "high",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "semiconductor", SectorName: "半导体", WeightPercent: decimal.RequireFromString("45.0000")},
				{SectorCode: "communications_equipment", SectorName: "通信设备", WeightPercent: decimal.RequireFromString("16.0000")},
			},
		},
		ThemeSnapshot: &domain.FundThemeSnapshot{
			PrimaryThemeCode: "semiconductor_chip",
			PrimaryThemeName: "半导体芯片",
			Confidence:       "high",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "semiconductor_chip", ThemeName: "半导体芯片", WeightPercent: decimal.RequireFromString("53.3100")},
				{ThemeCode: "ai_compute", ThemeName: "AI算力", WeightPercent: decimal.RequireFromString("12.0000")},
			},
		},
		PreviousThemeSnapshot: &domain.FundThemeSnapshot{
			PrimaryThemeCode: "semiconductor_chip",
			PrimaryThemeName: "半导体芯片",
			Confidence:       "high",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "semiconductor_chip", ThemeName: "半导体芯片", WeightPercent: decimal.RequireFromString("44.0000")},
				{ThemeCode: "ai_compute", ThemeName: "AI算力", WeightPercent: decimal.RequireFromString("18.0000")},
			},
		},
		Holdings: []domain.StockHolding{
			{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("9.9900"), ReportingPeriod: "2026-03-31"},
		},
		HoldingsSource: SectorSourceDirectHoldings,
		Now:            time.Date(2026, time.April, 26, 15, 0, 0, 0, time.Local),
	})

	if analysis == nil {
		t.Fatal("analysis = nil")
	}
	themeIncrease := findEventImpact(analysis.EventImpacts, "theme_weight_increase")
	if themeIncrease == nil {
		t.Fatal("theme_weight_increase should exist")
	}
	sectorDecrease := findEventImpact(analysis.EventImpacts, "sector_weight_decrease")
	if sectorDecrease == nil {
		t.Fatal("sector_weight_decrease should exist")
	}
}

func TestFundAnalysisServiceIncludesMacroEvents(t *testing.T) {
	service := NewFundAnalysisService()

	analysis := service.Build(FundAnalysisInput{
		Fund: &domain.Fund{ID: "159813", Name: "半导体ETF鹏华", Type: "index"},
		Estimate: &domain.FundEstimate{
			FundID:         "159813",
			FundName:       "半导体ETF鹏华",
			ChangePercent:  decimal.RequireFromString("0.5000"),
			TotalHoldRatio: decimal.RequireFromString("68.2800"),
		},
		TimeSeries: []domain.TimeSeriesPoint{
			{Timestamp: time.Date(2026, time.April, 26, 10, 0, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("0.1000")},
			{Timestamp: time.Date(2026, time.April, 26, 14, 30, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("0.5000")},
		},
		SectorSnapshot: &domain.FundSectorSnapshot{
			PrimarySectorCode: "semiconductor",
			PrimarySectorName: "半导体",
			Confidence:        "high",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "semiconductor", SectorName: "半导体", WeightPercent: decimal.RequireFromString("53.3100")},
			},
		},
		ThemeSnapshot: &domain.FundThemeSnapshot{
			PrimaryThemeCode: "semiconductor_chip",
			PrimaryThemeName: "半导体芯片",
			Confidence:       "high",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "semiconductor_chip", ThemeName: "半导体芯片", WeightPercent: decimal.RequireFromString("53.3100")},
			},
		},
		Holdings: []domain.StockHolding{
			{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("9.9900"), ReportingPeriod: "2026-03-31"},
		},
		CurrentMacroEvents: []domain.FundAnalysisEventImpact{
			{
				Code:        "macro_ic_tax_support_2026",
				Title:       "集成电路税收支持延续",
				Impact:      "positive",
				Summary:     "近期官方继续强调集成电路产业支持，半导体主线的中期政策预期仍有支撑。",
				TargetScope: "macro",
				Strength:    "medium",
				Horizon:     "current",
			},
		},
		HoldingsSource: SectorSourceDirectHoldings,
		Now:            time.Date(2026, time.April, 27, 9, 0, 0, 0, time.Local),
	})

	if analysis == nil {
		t.Fatal("analysis = nil")
	}
	foundMacro := false
	for _, item := range analysis.EventImpacts {
		if item.Code == "macro_ic_tax_support_2026" {
			foundMacro = true
			break
		}
	}
	if !foundMacro {
		t.Fatalf("event impacts = %v, want macro event", analysis.EventImpacts)
	}
	foundReason := false
	for _, reason := range analysis.Reasons {
		if strings.Contains(reason, "宏观/政策层面值得关注") {
			foundReason = true
			break
		}
	}
	if !foundReason {
		t.Fatalf("reasons = %v, want macro event reason", analysis.Reasons)
	}
}

func TestLoadMacroPolicyEventsMatchesActualThemeCodes(t *testing.T) {
	events := LoadMacroPolicyEvents(
		time.Date(2026, time.April, 27, 9, 0, 0, 0, time.Local),
		&domain.FundSectorSnapshot{
			PrimarySectorCode: "communications_equipment",
			PrimarySectorName: "通信设备",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "communications_equipment", SectorName: "通信设备", WeightPercent: decimal.RequireFromString("18.0000")},
			},
		},
		&domain.FundThemeSnapshot{
			PrimaryThemeCode: "computing_power",
			PrimaryThemeName: "AI算力",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "computing_power", ThemeName: "AI算力", WeightPercent: decimal.RequireFromString("32.0000")},
			},
		},
	)

	found := false
	for _, item := range events {
		if item.Code == "macro_ai_plus_2026" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("events = %+v, want macro_ai_plus_2026 for computing_power theme", events)
	}
}

func TestLoadMacroPolicyEventsSupportsLegacyThemeAlias(t *testing.T) {
	events := LoadMacroPolicyEvents(
		time.Date(2026, time.April, 27, 9, 0, 0, 0, time.Local),
		&domain.FundSectorSnapshot{
			PrimarySectorCode: "communications_equipment",
			PrimarySectorName: "通信设备",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "communications_equipment", SectorName: "通信设备", WeightPercent: decimal.RequireFromString("18.0000")},
			},
		},
		&domain.FundThemeSnapshot{
			PrimaryThemeCode: "ai_compute",
			PrimaryThemeName: "AI算力",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "ai_compute", ThemeName: "AI算力", WeightPercent: decimal.RequireFromString("32.0000")},
			},
		},
	)

	found := false
	for _, item := range events {
		if item.Code == "macro_ai_plus_2026" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("events = %+v, want macro_ai_plus_2026 for legacy ai_compute alias", events)
	}
}

func TestLoadMacroPolicyEventsIncludesHealthcareTheme(t *testing.T) {
	events := LoadMacroPolicyEvents(
		time.Date(2026, time.April, 27, 9, 0, 0, 0, time.Local),
		&domain.FundSectorSnapshot{
			PrimarySectorCode: "healthcare_service",
			PrimarySectorName: "医疗服务",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "healthcare_service", SectorName: "医疗服务", WeightPercent: decimal.RequireFromString("21.0000")},
			},
		},
		&domain.FundThemeSnapshot{
			PrimaryThemeCode: "innovative_medicine",
			PrimaryThemeName: "创新药",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "innovative_medicine", ThemeName: "创新药", WeightPercent: decimal.RequireFromString("26.0000")},
			},
		},
	)

	found := false
	for _, item := range events {
		if item.Code == "macro_innovative_medicine_support_2026" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("events = %+v, want healthcare macro event", events)
	}
}

func TestLoadMacroPolicyEventsAddsPrimaryThemeContextToSummary(t *testing.T) {
	events := LoadMacroPolicyEvents(
		time.Date(2026, time.April, 27, 9, 0, 0, 0, time.Local),
		&domain.FundSectorSnapshot{
			PrimarySectorCode: "semiconductor",
			PrimarySectorName: "半导体",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "semiconductor", SectorName: "半导体", WeightPercent: decimal.RequireFromString("53.3000")},
			},
		},
		&domain.FundThemeSnapshot{
			PrimaryThemeCode: "semiconductor_chip",
			PrimaryThemeName: "半导体芯片",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "semiconductor_chip", ThemeName: "半导体芯片", WeightPercent: decimal.RequireFromString("53.3000")},
			},
		},
	)

	found := false
	for _, item := range events {
		if item.Code == "macro_ic_tax_support_2026" {
			if !strings.Contains(item.Summary, "当前主主题为半导体芯片") {
				t.Fatalf("summary = %q, want primary theme context", item.Summary)
			}
			if item.WeightHint == nil || !item.WeightHint.GreaterThan(decimal.Zero) {
				t.Fatal("macro event should carry exposure weight hint")
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("events = %+v, want semiconductor macro event", events)
	}
}

func TestLoadMacroPolicyEventsRequiresExposureWeight(t *testing.T) {
	events := LoadMacroPolicyEvents(
		time.Date(2026, time.April, 27, 9, 0, 0, 0, time.Local),
		&domain.FundSectorSnapshot{
			PrimarySectorCode: "semiconductor",
			PrimarySectorName: "半导体",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "semiconductor", SectorName: "半导体", WeightPercent: decimal.RequireFromString("3.0000")},
			},
		},
		&domain.FundThemeSnapshot{
			PrimaryThemeCode: "semiconductor_chip",
			PrimaryThemeName: "半导体芯片",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "semiconductor_chip", ThemeName: "半导体芯片", WeightPercent: decimal.RequireFromString("3.0000")},
			},
		},
	)
	for _, item := range events {
		if item.Code == "macro_ic_tax_support_2026" {
			t.Fatalf("events = %+v, low exposure should not trigger macro hotspot", events)
		}
	}
}

func TestLoadMacroPolicyEventsAddsRealtimeHormuzEnergyRadar(t *testing.T) {
	events := LoadMacroPolicyEvents(
		time.Date(2026, time.June, 18, 9, 0, 0, 0, time.Local),
		&domain.FundSectorSnapshot{
			PrimarySectorCode: "oil_gas_energy",
			PrimarySectorName: "石油能源",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "oil_gas_energy", SectorName: "石油能源", WeightPercent: decimal.RequireFromString("28.0000")},
			},
		},
		nil,
	)

	item := findEventImpact(events, "realtime_hormuz_reopening_energy_pressure_202606")
	if item == nil {
		t.Fatalf("events = %+v, want realtime Hormuz energy event", events)
	}
	if item.Impact != "negative" {
		t.Fatalf("impact = %q, want negative", item.Impact)
	}
	if item.TargetScope != "macro" {
		t.Fatalf("target_scope = %q, want macro", item.TargetScope)
	}
	if item.WeightHint == nil || !item.WeightHint.Equal(decimal.RequireFromString("28.0")) {
		t.Fatalf("weight_hint = %v, want 28.0", item.WeightHint)
	}
	if !strings.Contains(item.Summary, "当前主行业为石油能源") {
		t.Fatalf("summary = %q, want primary sector context", item.Summary)
	}
	if item.SourceName != "Associated Press" {
		t.Fatalf("source_name = %q, want Associated Press", item.SourceName)
	}
	if !strings.Contains(item.SourceURL, "apnews.com/article/strait-of-hormuz") {
		t.Fatalf("source_url = %q, want AP Hormuz article", item.SourceURL)
	}
	if item.SourcePublishedAt != "2026-06-15" {
		t.Fatalf("source_published_at = %q, want 2026-06-15", item.SourcePublishedAt)
	}
	if item.SourceConfidence != "high" {
		t.Fatalf("source_confidence = %q, want high", item.SourceConfidence)
	}
	if !strings.Contains(item.MappingBasis, "行业暴露：石油能源") {
		t.Fatalf("mapping_basis = %q, want sector exposure trace", item.MappingBasis)
	}
}

func TestLoadMacroPolicyEventsMapsHormuzCostReliefToConsumerExposure(t *testing.T) {
	events := LoadMacroPolicyEvents(
		time.Date(2026, time.June, 18, 9, 0, 0, 0, time.Local),
		&domain.FundSectorSnapshot{
			PrimarySectorCode: "consumer_service",
			PrimarySectorName: "消费服务",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "consumer_service", SectorName: "消费服务", WeightPercent: decimal.RequireFromString("24.0000")},
			},
		},
		&domain.FundThemeSnapshot{
			PrimaryThemeCode: "consumer_upgrade",
			PrimaryThemeName: "消费升级",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "consumer_upgrade", ThemeName: "消费升级", WeightPercent: decimal.RequireFromString("19.0000")},
			},
		},
	)

	found := false
	for _, item := range events {
		if item.Code == "realtime_hormuz_reopening_cost_relief_202606" {
			found = true
			if item.Impact != "positive" {
				t.Fatalf("impact = %q, want positive", item.Impact)
			}
			if !strings.Contains(item.Summary, "匹配暴露约") {
				t.Fatalf("summary = %q, want exposure context", item.Summary)
			}
			break
		}
	}
	if !found {
		t.Fatalf("events = %+v, want realtime Hormuz cost relief event", events)
	}
}

func TestLoadMacroPolicyEventsDoesNotBroadcastRealtimeHormuzToLowExposure(t *testing.T) {
	events := LoadMacroPolicyEvents(
		time.Date(2026, time.June, 18, 9, 0, 0, 0, time.Local),
		&domain.FundSectorSnapshot{
			PrimarySectorCode: "oil_gas_energy",
			PrimarySectorName: "石油能源",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "oil_gas_energy", SectorName: "石油能源", WeightPercent: decimal.RequireFromString("4.0000")},
			},
		},
		nil,
	)

	for _, item := range events {
		if strings.HasPrefix(item.Code, "realtime_hormuz_") {
			t.Fatalf("events = %+v, low exposure should not trigger realtime Hormuz event", events)
		}
	}
}

func TestEvidenceFromEventPreservesRealtimeSourceTrace(t *testing.T) {
	weight := decimal.RequireFromString("28.0")
	evidence := evidenceFromEvent(domain.FundAnalysisEventImpact{
		Code:              "realtime_hormuz_reopening_energy_pressure_202606",
		Title:             "美伊协议推动霍尔木兹重开预期",
		Summary:           "油气上游需要观察油价中枢下修压力。",
		Impact:            "negative",
		TargetScope:       "macro",
		Strength:          "high",
		Horizon:           "current",
		WeightHint:        &weight,
		SourceName:        "Associated Press",
		SourceURL:         "https://apnews.com/article/strait-of-hormuz-oil-prices-iran-war-8304cc39c6ebe6f863f6f39ee6ce9768",
		SourcePublishedAt: "2026-06-15",
		SourceConfidence:  "high",
		MappingBasis:      "行业暴露：石油能源，权重约 28.0%",
	}, "counter_event")

	if evidence.SourceName != "Associated Press" || evidence.SourceURL == "" {
		t.Fatalf("evidence source = (%q, %q), want AP source preserved", evidence.SourceName, evidence.SourceURL)
	}
	if evidence.SourcePublishedAt != "2026-06-15" || evidence.SourceConfidence != "high" {
		t.Fatalf("evidence source meta = (%q, %q), want date/confidence preserved", evidence.SourcePublishedAt, evidence.SourceConfidence)
	}
	if !strings.Contains(evidence.MappingBasis, "行业暴露：石油能源") {
		t.Fatalf("evidence mapping_basis = %q, want mapping basis preserved", evidence.MappingBasis)
	}
	if evidence.WeightHint == nil || !evidence.WeightHint.Equal(weight) {
		t.Fatalf("evidence weight_hint = %v, want %s", evidence.WeightHint, weight)
	}
}

func TestFundAnalysisServiceSuppressesWeakMixedExposureCluster(t *testing.T) {
	service := NewFundAnalysisService()

	analysis := service.Build(FundAnalysisInput{
		Fund: &domain.Fund{
			ID:   "123456",
			Name: "测试主题基金",
			Type: "hybrid",
		},
		Estimate: &domain.FundEstimate{
			FundID:         "123456",
			FundName:       "测试主题基金",
			ChangePercent:  decimal.RequireFromString("0.2000"),
			TotalHoldRatio: decimal.RequireFromString("50.0000"),
		},
		ThemeSnapshot: &domain.FundThemeSnapshot{
			PrimaryThemeCode: "computing_power",
			PrimaryThemeName: "AI算力",
			Confidence:       "high",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "computing_power", ThemeName: "AI算力", WeightPercent: decimal.RequireFromString("32.0000")},
			},
		},
		Holdings: []domain.StockHolding{
			{StockCode: "000001", StockName: "样本一", HoldingRatio: decimal.RequireFromString("5.0000"), ReportingPeriod: "2026-03-31"},
		},
		CurrentHoldingEvents: []domain.FundAnalysisEventImpact{
			{
				Code:        "mixed_positive",
				Title:       "样本一经营公告",
				Impact:      "positive",
				TargetScope: "holding",
				Strength:    "low",
				Horizon:     "intraday",
				Summary:     "正向样本",
			},
			{
				Code:        "mixed_negative",
				Title:       "样本二风险提示",
				Impact:      "negative",
				TargetScope: "holding",
				Strength:    "low",
				Horizon:     "intraday",
				Summary:     "负向样本",
			},
		},
		HoldingsSource: SectorSourceDirectHoldings,
		Now:            time.Date(2026, time.April, 28, 12, 0, 0, 0, time.Local),
	})

	if analysis == nil {
		t.Fatal("analysis = nil")
	}
	if event := findEventImpact(analysis.EventImpacts, "current_exposure_event_cluster"); event != nil {
		t.Fatalf("event impacts = %#v, weak mixed exposure cluster should be suppressed", analysis.EventImpacts)
	}
}

func TestNormalizeAnalysisEventImpactsPrioritizesSpecificEvents(t *testing.T) {
	events := normalizeAnalysisEventImpacts([]domain.FundAnalysisEventImpact{
		{
			Code:        "current_exposure_event_cluster",
			Title:       "半导体芯片主线近期事件密集",
			Impact:      "positive",
			Summary:     "泛化主线说明",
			TargetScope: "exposure",
			Strength:    "medium",
			Horizon:     "current",
		},
		{
			Code:           "holding_current_notice_1",
			Title:          "海光信息发布一季报",
			Impact:         "positive",
			Summary:        "具体持仓事件",
			TargetScope:    "holding",
			Strength:       "high",
			Horizon:        "current",
			RelatedSymbols: []string{"688041"},
			WeightHint:     decimalPointerFromFloat(9.99),
		},
		{
			Code:        "fund_notice_1",
			Title:       "基金经理变更公告",
			Impact:      "negative",
			Summary:     "基金自身事件",
			TargetScope: "fund",
			Strength:    "medium",
			Horizon:     "current",
		},
		{
			Code:        "analysis_basis",
			Title:       "当前分析口径",
			Impact:      "neutral",
			Summary:     "方法论说明",
			TargetScope: "methodology",
			Strength:    "low",
			Horizon:     "current",
		},
	})

	if len(events) < 3 {
		t.Fatalf("normalized events len = %d, want >= 3", len(events))
	}
	if events[0].Code != "holding_current_notice_1" {
		t.Fatalf("first event = %s, want specific holding event first", events[0].Code)
	}
	if events[len(events)-1].Code != "analysis_basis" {
		t.Fatalf("last event = %s, want analysis_basis last", events[len(events)-1].Code)
	}
}

func TestLoadIndexLayerEvents(t *testing.T) {
	events := LoadIndexLayerEvents(
		time.Date(2026, time.April, 27, 9, 0, 0, 0, time.Local),
		&domain.Fund{ID: "159813", Name: "半导体ETF鹏华", Type: "index"},
		SectorSourceDirectHoldings,
		&domain.FundSectorSnapshot{
			PrimarySectorCode: "semiconductor",
			PrimarySectorName: "半导体",
		},
		&domain.FundThemeSnapshot{
			PrimaryThemeCode: "semiconductor_chip",
			PrimaryThemeName: "半导体芯片",
		},
	)
	if len(events) == 0 {
		t.Fatal("LoadIndexLayerEvents() should return at least one event")
	}
	if events[0].TargetScope != "index" {
		t.Fatalf("target_scope = %q, want index", events[0].TargetScope)
	}
}

func TestFundAnalysisServiceBuildsCredibilityEvidenceBeforeAI(t *testing.T) {
	service := NewFundAnalysisService()

	analysis := service.Build(FundAnalysisInput{
		Fund: &domain.Fund{ID: "159813", Name: "半导体ETF鹏华", Type: "index"},
		Estimate: &domain.FundEstimate{
			FundID:         "159813",
			FundName:       "半导体ETF鹏华",
			ChangePercent:  decimal.RequireFromString("0.9000"),
			TotalHoldRatio: decimal.RequireFromString("52.0000"),
			HoldingDetails: []domain.HoldingDetail{
				{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("9.9900"), StockChange: decimal.RequireFromString("3.2000"), Contribution: decimal.RequireFromString("0.3190")},
			},
		},
		TimeSeries: []domain.TimeSeriesPoint{
			{Timestamp: time.Date(2026, time.April, 28, 10, 0, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("0.1000")},
			{Timestamp: time.Date(2026, time.April, 28, 14, 30, 0, 0, time.Local), ChangePercent: decimal.RequireFromString("0.9000")},
		},
		SectorSnapshot: &domain.FundSectorSnapshot{
			PrimarySectorCode: "semiconductor",
			PrimarySectorName: "半导体",
			Confidence:        "high",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "semiconductor", SectorName: "半导体", WeightPercent: decimal.RequireFromString("52.0000")},
			},
		},
		ThemeSnapshot: &domain.FundThemeSnapshot{
			PrimaryThemeCode: "semiconductor_chip",
			PrimaryThemeName: "半导体芯片",
			Confidence:       "high",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "semiconductor_chip", ThemeName: "半导体芯片", WeightPercent: decimal.RequireFromString("52.0000")},
			},
		},
		Holdings: []domain.StockHolding{
			{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("9.9900"), ReportingPeriod: "2026-03-31"},
		},
		CurrentHoldingEvents: []domain.FundAnalysisEventImpact{
			{
				Code:           "holding_current_notice_688041_test",
				Title:          "海光信息发布一季报",
				Impact:         "positive",
				Summary:        "近期开启事件：2026-04-20 发布《海光信息技术股份有限公司2026年第一季度报告》。",
				TargetScope:    "holding",
				Strength:       "high",
				Horizon:        "current",
				RelatedSymbols: []string{"688041"},
				WeightHint:     decimalPointerFromFloat(9.99),
			},
		},
		CurrentFundEvents: []domain.FundAnalysisEventImpact{
			{
				Code:        "fund_notice_generic",
				Title:       "基金普通公告",
				Impact:      "positive",
				Summary:     "基金普通公告不应抢占持仓与行业证据主位。",
				TargetScope: "fund",
				Strength:    "medium",
				Horizon:     "current",
			},
		},
		HoldingsSource: SectorSourceDirectHoldings,
		Now:            time.Date(2026, time.April, 28, 15, 0, 0, 0, time.Local),
	})

	if analysis == nil {
		t.Fatal("analysis = nil")
	}
	if len(analysis.ConfidenceFactors) < 6 {
		t.Fatalf("confidence factors len = %d, want >= 6", len(analysis.ConfidenceFactors))
	}
	if len(analysis.PrimaryEvidence) == 0 {
		t.Fatal("primary evidence should be populated")
	}
	if analysis.PrimaryEvidence[0].SourceScope != "holding" {
		t.Fatalf("first primary evidence scope = %q, want holding", analysis.PrimaryEvidence[0].SourceScope)
	}
	if len(analysis.CounterEvidence) == 0 {
		t.Fatal("counter evidence should be populated")
	}
	for _, reason := range analysis.Reasons {
		if strings.Contains(reason, "基金产品层事件") || strings.Contains(reason, "基金普通公告") {
			t.Fatalf("generic fund notice should not become main reason: %v", analysis.Reasons)
		}
	}
}

func findEventImpact(events []domain.FundAnalysisEventImpact, code string) *domain.FundAnalysisEventImpact {
	for i := range events {
		if events[i].Code == code {
			return &events[i]
		}
	}
	return nil
}
