package main

import (
	"net/http"
	"net/http/httptest"
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
					"summary": "当前偏向加仓，但更适合分批而不是追涨。",
					"reasons": ["示例理由"],
					"warnings": ["示例警告"],
					"event_impacts": [{"code":"holding_disclosure_fresh","title":"持仓披露较新","impact":"positive","summary":"示例事件"}],
					"module_scores": [{"code":"trend","name":"趋势","score":"74.1","summary":"示例模块"}]
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
}
