package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

type stubAIExplanationProvider struct {
	name   string
	output *domain.FundAnalysisAIExplanation
	err    error
	seen   *AIExplanationRequest
}

func (p *stubAIExplanationProvider) Name() string {
	return p.name
}

func (p *stubAIExplanationProvider) Generate(ctx context.Context, request AIExplanationRequest) (*domain.FundAnalysisAIExplanation, error) {
	requestCopy := request
	p.seen = &requestCopy
	if p.err != nil {
		return nil, p.err
	}
	return p.output, nil
}

func TestAIExplanationServiceDisabledFallbackUsesEvidenceOnly(t *testing.T) {
	analysis := testAIExplanationAnalysis()
	originalTotal := analysis.TotalScore
	originalRisk := analysis.RiskLevel
	originalIncrease := analysis.IncreasePercent

	explanation, err := NewAIExplanationService(nil).Explain(context.Background(), AIExplanationInput{
		Fund:     &domain.Fund{ID: "159813", Name: "半导体ETF鹏华", Type: "index"},
		Analysis: analysis,
		Holdings: []domain.StockHolding{
			{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("9.9900")},
		},
		SectorSnapshot: &domain.FundSectorSnapshot{
			PrimarySectorCode: "semiconductor",
			PrimarySectorName: "半导体",
			Breakdown: []domain.FundSectorBreakdown{
				{SectorCode: "semiconductor", SectorName: "半导体", WeightPercent: decimal.RequireFromString("52.0000")},
			},
		},
		ThemeSnapshot: &domain.FundThemeSnapshot{
			PrimaryThemeCode: "semiconductor_chip",
			PrimaryThemeName: "半导体芯片",
			Breakdown: []domain.FundThemeBreakdown{
				{ThemeCode: "semiconductor_chip", ThemeName: "半导体芯片", WeightPercent: decimal.RequireFromString("52.0000")},
			},
		},
		Now: time.Date(2026, time.May, 8, 15, 0, 0, 0, time.Local),
	})
	if err != nil {
		t.Fatalf("Explain() error = %v", err)
	}
	if explanation == nil {
		t.Fatal("explanation = nil")
	}
	if explanation.Status != AIExplanationStatusDisabled {
		t.Fatalf("status = %q, want disabled", explanation.Status)
	}
	if explanation.Provider != AIExplanationProviderDisabled {
		t.Fatalf("provider = %q, want disabled", explanation.Provider)
	}
	if explanation.RuleRecommendation != "increase" {
		t.Fatalf("rule recommendation = %q, want increase", explanation.RuleRecommendation)
	}
	if !strings.Contains(explanation.BoundaryNotice, "不得改写") {
		t.Fatalf("boundary notice = %q, want scoring boundary", explanation.BoundaryNotice)
	}
	if len(explanation.Attribution) == 0 || len(explanation.Citations) == 0 {
		t.Fatalf("fallback explanation should expose cited attribution: %+v", explanation)
	}
	if explanation.CacheKey == "" || explanation.CacheStatus != AIExplanationCacheStatusGenerated {
		t.Fatalf("cache metadata = key:%q status:%q, want generated cache metadata", explanation.CacheKey, explanation.CacheStatus)
	}
	if explanation.ExpiresAt.IsZero() || len(explanation.InvalidationBasis) == 0 {
		t.Fatalf("cache expiry/basis should be populated: %+v", explanation)
	}
	if !analysis.TotalScore.Equal(originalTotal) || analysis.RiskLevel != originalRisk || !analysis.IncreasePercent.Equal(originalIncrease) {
		t.Fatalf("AI explanation must not mutate rule scores: %+v", analysis)
	}
}

func TestAIExplanationServiceRejectsWhenEvidenceBundleMissing(t *testing.T) {
	analysis := &domain.FundAnalysis{
		TotalScore:      decimal.RequireFromString("50.0000"),
		RiskLevel:       "medium",
		IncreasePercent: decimal.RequireFromString("30.0000"),
		HoldPercent:     decimal.RequireFromString("40.0000"),
		DecreasePercent: decimal.RequireFromString("30.0000"),
	}

	explanation, err := NewAIExplanationService(nil).Explain(context.Background(), AIExplanationInput{
		Analysis: analysis,
		Now:      time.Date(2026, time.May, 8, 15, 0, 0, 0, time.Local),
	})
	if err != nil {
		t.Fatalf("Explain() error = %v", err)
	}
	if explanation == nil {
		t.Fatal("explanation = nil")
	}
	if explanation.Status != AIExplanationStatusRejected {
		t.Fatalf("status = %q, want rejected", explanation.Status)
	}
	if explanation.CacheStatus != AIExplanationCacheStatusNotCacheable {
		t.Fatalf("cache status = %q, want not_cacheable", explanation.CacheStatus)
	}
	if !strings.Contains(explanation.Summary, "证据包不足") {
		t.Fatalf("summary = %q, want evidence warning", explanation.Summary)
	}
	if len(explanation.Citations) != 0 {
		t.Fatalf("citations = %+v, want empty without evidence", explanation.Citations)
	}
}

func TestCanReuseAIExplanationRequiresMatchingCacheKeyAndExpiry(t *testing.T) {
	analysis := testAIExplanationAnalysis()
	input := AIExplanationInput{
		Analysis: analysis,
		Now:      time.Date(2026, time.May, 8, 15, 0, 0, 0, time.Local),
	}
	explanation, err := NewAIExplanationService(nil).Explain(context.Background(), input)
	if err != nil {
		t.Fatalf("Explain() error = %v", err)
	}
	cacheKey := BuildAIExplanationCacheKey(input)
	if !CanReuseAIExplanation(explanation, cacheKey, time.Date(2026, time.May, 8, 16, 0, 0, 0, time.Local)) {
		t.Fatalf("expected explanation to be reusable before expiry")
	}
	if CanReuseAIExplanation(explanation, "different-key", time.Date(2026, time.May, 8, 16, 0, 0, 0, time.Local)) {
		t.Fatalf("different cache key should not be reusable")
	}
	if CanReuseAIExplanation(explanation, cacheKey, time.Date(2026, time.May, 9, 0, 30, 0, 0, time.Local)) {
		t.Fatalf("expired explanation should not be reusable")
	}
	hit := MarkAIExplanationSnapshotHit(explanation)
	if hit == nil || hit.CacheStatus != AIExplanationCacheStatusSnapshotHit {
		t.Fatalf("snapshot hit metadata = %+v", hit)
	}
	if explanation.CacheStatus == AIExplanationCacheStatusSnapshotHit {
		t.Fatalf("MarkAIExplanationSnapshotHit should not mutate original explanation")
	}
}

func TestIsFundAnalysisSnapshotFreshRequiresCurrentVersionAndExplanationCache(t *testing.T) {
	now := time.Date(2026, time.May, 8, 15, 0, 0, 0, time.Local)
	analysis := testAIExplanationAnalysis()
	analysis.AnalysisVersion = CurrentFundAnalysisVersion
	explanation, err := NewAIExplanationService(nil).Explain(context.Background(), AIExplanationInput{Analysis: analysis, Now: now})
	if err != nil {
		t.Fatalf("Explain() error = %v", err)
	}
	analysis.AIExplanation = explanation

	if !IsFundAnalysisSnapshotFresh(analysis, now.Add(-time.Hour), now) {
		t.Fatalf("fresh snapshot should be reusable")
	}
	analysis.AnalysisVersion = FundAnalysisVersionV1
	if IsFundAnalysisSnapshotFresh(analysis, now.Add(-time.Hour), now) {
		t.Fatalf("old analysis version should be stale")
	}
	analysis.AnalysisVersion = CurrentFundAnalysisVersion
	analysis.AIExplanation = nil
	if IsFundAnalysisSnapshotFresh(analysis, now.Add(-time.Hour), now) {
		t.Fatalf("snapshot without AI explanation metadata should be stale")
	}
}

func TestAIExplanationServiceSanitizesProviderOutputCitations(t *testing.T) {
	provider := &stubAIExplanationProvider{
		name: "mock-provider",
		output: &domain.FundAnalysisAIExplanation{
			Summary: "基于已引用证据生成解释。",
			Attribution: []domain.FundAnalysisAIExplanationSection{
				{
					Title:         "有效归因",
					Summary:       "持仓事件支撑当前结构偏积极观察。",
					CitationCodes: []string{"holding_current_notice_688041_test"},
				},
				{
					Title:         "无来源热点",
					Summary:       "没有证据的泛热点不应展示。",
					CitationCodes: []string{"hallucinated_hotspot"},
				},
			},
		},
	}

	explanation, err := NewAIExplanationService(provider).Explain(context.Background(), AIExplanationInput{
		Analysis: testAIExplanationAnalysis(),
		Now:      time.Date(2026, time.May, 8, 15, 0, 0, 0, time.Local),
	})
	if err != nil {
		t.Fatalf("Explain() error = %v", err)
	}
	if provider.seen == nil {
		t.Fatal("provider did not receive request")
	}
	if len(provider.seen.Evidence) == 0 {
		t.Fatalf("provider request evidence should not be empty")
	}
	if explanation.Status != AIExplanationStatusReady {
		t.Fatalf("status = %q, want ready", explanation.Status)
	}
	if len(explanation.Attribution) != 1 {
		t.Fatalf("attribution = %+v, want only valid cited section", explanation.Attribution)
	}
	if got := explanation.Attribution[0].CitationCodes[0]; got != "holding_current_notice_688041_test" {
		t.Fatalf("citation = %q, want valid evidence code", got)
	}
	for _, citation := range explanation.Citations {
		if citation.Code == "hallucinated_hotspot" {
			t.Fatalf("unsupported citation leaked: %+v", explanation.Citations)
		}
	}
	if len(explanation.Limitations) == 0 || !strings.Contains(strings.Join(explanation.Limitations, "；"), "移除") {
		t.Fatalf("limitations = %+v, want unsupported citation note", explanation.Limitations)
	}
}

func TestAIExplanationServiceProviderFailureFallsBackWithoutBlocking(t *testing.T) {
	provider := &stubAIExplanationProvider{name: "mock-provider", err: errors.New("model unavailable")}
	explanation, err := NewAIExplanationService(provider).Explain(context.Background(), AIExplanationInput{
		Analysis: testAIExplanationAnalysis(),
		Now:      time.Date(2026, time.May, 8, 15, 0, 0, 0, time.Local),
	})
	if err != nil {
		t.Fatalf("Explain() error = %v", err)
	}
	if explanation.Status != AIExplanationStatusFallback {
		t.Fatalf("status = %q, want fallback", explanation.Status)
	}
	if explanation.Provider != "mock-provider" {
		t.Fatalf("provider = %q, want mock-provider", explanation.Provider)
	}
	if len(explanation.Citations) == 0 {
		t.Fatalf("fallback should keep rule citations: %+v", explanation)
	}
}

func testAIExplanationAnalysis() *domain.FundAnalysis {
	return &domain.FundAnalysis{
		TotalScore:      decimal.RequireFromString("68.1000"),
		Confidence:      "high",
		RiskLevel:       "medium",
		IncreasePercent: decimal.RequireFromString("62.0000"),
		HoldPercent:     decimal.RequireFromString("28.0000"),
		DecreasePercent: decimal.RequireFromString("10.0000"),
		Summary:         "当前结构偏积极，但仍需结合风险暴露观察。",
		PrimaryEvidence: []domain.FundAnalysisEvidenceItem{
			{
				Code:         "holding_current_notice_688041_test",
				Title:        "海光信息发布一季报",
				Summary:      "近期开启事件：2026-04-20 发布《海光信息技术股份有限公司2026年第一季度报告》。",
				EvidenceType: "event",
				SourceScope:  "holding",
				Impact:       "positive",
				Strength:     "high",
				Horizon:      "current",
			},
		},
		CounterEvidence: []domain.FundAnalysisEvidenceItem{
			{
				Code:         "counter_risk_score",
				Title:        "风险模块压制结论",
				Summary:      "风险模块仍提示波动压力。",
				EvidenceType: "risk",
				SourceScope:  "exposure",
				Impact:       "negative",
				Strength:     "medium",
				Horizon:      "current",
			},
		},
		EventImpacts: []domain.FundAnalysisEventImpact{
			{
				Code:        "holding_current_notice_688041_test",
				Title:       "海光信息发布一季报",
				Impact:      "positive",
				Summary:     "重仓股事件与当前结构偏积极观察相关。",
				TargetScope: "holding",
				Strength:    "high",
				Horizon:     "current",
			},
		},
		ConfidenceFactors: []domain.FundAnalysisConfidenceFactor{
			{
				Code:    "holding_coverage",
				Name:    "持仓覆盖",
				Level:   "high",
				Score:   decimal.RequireFromString("82.0000"),
				Summary: "持仓覆盖较充分。",
			},
		},
	}
}
