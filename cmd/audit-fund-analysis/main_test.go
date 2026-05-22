package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveSamplesDefaultsToCorePool(t *testing.T) {
	samples := resolveSamples("")
	if len(samples) != len(coreSamples) {
		t.Fatalf("resolveSamples default len = %d, want %d", len(samples), len(coreSamples))
	}
}

func TestFetchDashboardAudit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/fund/012970/dashboard" {
			t.Fatalf("path = %s, want /api/v1/fund/012970/dashboard", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": {
				"fund": {
					"id": "012970",
					"name": "鹏华国证半导体芯片ETF联接C",
					"category_code": "feeder",
					"category_name": "ETF联接基金"
				},
				"estimate": {
					"change_percent": "1.7800",
					"data_source": "追踪目标ETF(sina): 芯片"
				},
				"analysis": {
					"analysis_type": "tracked_etf",
					"analysis_basis": "目标ETF口径",
					"total_score": "71.5",
					"confidence": "high",
					"risk_level": "medium",
					"increase_percent": "63.2",
					"hold_percent": "24.6",
					"decrease_percent": "12.2",
					"latest_holding_period": "2026-03-31",
					"summary": "当前结构偏积极，但更适合分批观察。",
					"reasons": ["示例理由"],
					"warnings": ["示例警告"],
					"event_impacts": [{"code":"holding_disclosure_fresh","title":"持仓披露较新","impact":"positive","summary":"示例事件","target_scope":"disclosure","strength":"medium","horizon":"quarterly"}],
					"module_scores": [{"code":"trend","name":"趋势","score":"74.1","summary":"示例模块"}],
					"confidence_factors": [{"code":"holding_coverage","name":"持仓覆盖率","level":"high","score":"90.0","summary":"覆盖较高"}],
					"primary_evidence": [{"code":"evidence_1","title":"主证据","summary":"示例主证据","evidence_type":"event","source_scope":"holding","impact":"positive","strength":"high","horizon":"current"}],
					"counter_evidence": [{"code":"counter_1","title":"反方证据","summary":"示例反方证据","evidence_type":"confidence","source_scope":"methodology","impact":"negative","strength":"medium","horizon":"current"}],
					"confidence_deductions": ["示例扣分"],
					"ai_explanation": {
						"status": "disabled",
						"provider": "disabled",
						"cache_status": "generated",
						"expires_at": "2026-05-08T23:59:59+08:00",
						"invalidation_basis": ["analysis_version:baseline_v3","trading_day:2026-05-08"],
						"rule_recommendation": "increase",
						"boundary_notice": "AI解释层只负责解释，不得改写评分。",
						"summary": "AI解释层未启用；当前展示规则证据降级摘要。",
						"attribution": [{"title":"主证据","summary":"示例主证据","citation_codes":["evidence_1"]}],
						"risk_notes": [{"title":"反方证据","summary":"示例反方证据","citation_codes":["counter_1"]}],
						"citations": [{"code":"evidence_1","source_type":"primary_evidence","source_scope":"holding","title":"主证据","summary":"示例主证据"}],
						"limitations": ["未配置真实 AI provider"]
					}
				}
			}
		}`))
	}))
	defer server.Close()

	result, err := fetchDashboardAudit(server.Client(), server.URL, auditSample{
		Code:  "012970",
		Label: "ETF联接样本",
	})
	if err != nil {
		t.Fatalf("fetchDashboardAudit() error = %v", err)
	}
	if result.Fund.ID != "012970" {
		t.Fatalf("fund id = %s, want 012970", result.Fund.ID)
	}
	if result.Analysis == nil {
		t.Fatal("analysis = nil")
	}
	if result.Analysis.AnalysisType != "tracked_etf" {
		t.Fatalf("analysis type = %s, want tracked_etf", result.Analysis.AnalysisType)
	}
	if len(result.Analysis.EventImpacts) != 1 {
		t.Fatalf("event impacts len = %d, want 1", len(result.Analysis.EventImpacts))
	}
	if len(result.Analysis.ConfidenceFactors) != 1 {
		t.Fatalf("confidence factors len = %d, want 1", len(result.Analysis.ConfidenceFactors))
	}
	if len(result.Analysis.PrimaryEvidence) != 1 {
		t.Fatalf("primary evidence len = %d, want 1", len(result.Analysis.PrimaryEvidence))
	}
	if result.Analysis.AIExplanation == nil {
		t.Fatal("AI explanation should not be nil")
	}
	if result.Analysis.AIExplanation.Status != "disabled" {
		t.Fatalf("AI explanation status = %s, want disabled", result.Analysis.AIExplanation.Status)
	}
	if verdict := sampleHeuristicVerdict(result); verdict == "" {
		t.Fatal("sampleHeuristicVerdict() should not be empty")
	}
}

func TestSampleHeuristicVerdictFlagsLowConfidenceFactors(t *testing.T) {
	verdict := sampleHeuristicVerdict(auditResult{
		Analysis: &analysisInfo{
			Confidence: "medium",
			PrimaryEvidence: []analysisEvidenceItem{
				{Code: "primary", Title: "主证据"},
			},
			CounterEvidence: []analysisEvidenceItem{
				{Code: "counter", Title: "反方证据"},
			},
			ConfidenceFactors: []confidenceFactor{
				{Code: "event_source_strength", Name: "事件来源强度", Level: "low"},
			},
		},
	})

	if !strings.Contains(verdict, "低可信度因子") || !strings.Contains(verdict, "事件来源强度") {
		t.Fatalf("verdict = %q, want low factor warning", verdict)
	}
}
