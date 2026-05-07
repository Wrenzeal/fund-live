package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type auditSample struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

var coreSamples = []auditSample{
	{Code: "159813", Label: "ETF｜半导体ETF鹏华"},
	{Code: "012970", Label: "ETF联接｜鹏华国证半导体芯片ETF联接C"},
	{Code: "023408", Label: "ETF联接｜华宝创业板人工智能ETF发起式联接C"},
	{Code: "005827", Label: "主动权益｜易方达蓝筹精选混合"},
	{Code: "000362", Label: "主动权益｜国泰聚信价值优势混合A"},
	{Code: "000370", Label: "QDII｜广发全球医疗保健美元现汇A"},
}

type apiEnvelope struct {
	Success bool             `json:"success"`
	Data    dashboardPayload `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type dashboardPayload struct {
	Fund     fundInfo      `json:"fund"`
	Estimate estimateInfo  `json:"estimate"`
	Analysis *analysisInfo `json:"analysis"`
}

type fundInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CategoryCode string `json:"category_code"`
	CategoryName string `json:"category_name"`
}

type estimateInfo struct {
	ChangePercent string `json:"change_percent"`
	DataSource    string `json:"data_source"`
}

type analysisInfo struct {
	AnalysisType        string           `json:"analysis_type"`
	AnalysisBasis       string           `json:"analysis_basis"`
	TotalScore          string           `json:"total_score"`
	Confidence          string           `json:"confidence"`
	RiskLevel           string           `json:"risk_level"`
	IncreasePercent     string           `json:"increase_percent"`
	HoldPercent         string           `json:"hold_percent"`
	DecreasePercent     string           `json:"decrease_percent"`
	LatestHoldingPeriod string           `json:"latest_holding_period"`
	Summary             string           `json:"summary"`
	Reasons             []string         `json:"reasons"`
	Warnings            []string         `json:"warnings"`
	EventImpacts        []analysisEvent  `json:"event_impacts"`
	ModuleScores        []analysisModule `json:"module_scores"`
}

type analysisEvent struct {
	Code    string `json:"code"`
	Title   string `json:"title"`
	Impact  string `json:"impact"`
	Summary string `json:"summary"`
}

type analysisModule struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Score   string `json:"score"`
	Summary string `json:"summary"`
}

type auditResult struct {
	Sample   auditSample   `json:"sample"`
	Fund     fundInfo      `json:"fund"`
	Estimate estimateInfo  `json:"estimate"`
	Analysis *analysisInfo `json:"analysis"`
}

func main() {
	baseURLDefault := strings.TrimSpace(os.Getenv("FUND_ANALYSIS_AUDIT_BASE_URL"))
	if baseURLDefault == "" {
		baseURLDefault = "http://127.0.0.1:13896"
	}

	baseURL := flag.String("base-url", baseURLDefault, "FundLive API base URL")
	fundsArg := flag.String("funds", "", "comma separated fund ids to audit; defaults to curated core sample pool")
	jsonOutput := flag.Bool("json", false, "print JSON instead of text summary")
	timeout := flag.Duration("timeout", 10*time.Second, "HTTP timeout")
	flag.Parse()

	samples := resolveSamples(*fundsArg)
	client := &http.Client{Timeout: *timeout}

	results := make([]auditResult, 0, len(samples))
	failed := false
	for _, sample := range samples {
		result, err := fetchDashboardAudit(client, strings.TrimRight(*baseURL, "/"), sample)
		if err != nil {
			failed = true
			fmt.Fprintf(os.Stderr, "❌ %s (%s): %v\n", sample.Code, sample.Label, err)
			continue
		}
		results = append(results, result)
	}

	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(results)
	} else {
		printTextReport(results, strings.TrimRight(*baseURL, "/"))
	}

	if failed {
		os.Exit(1)
	}
}

func resolveSamples(fundsArg string) []auditSample {
	fundsArg = strings.TrimSpace(fundsArg)
	if fundsArg == "" {
		return coreSamples
	}

	parts := strings.Split(fundsArg, ",")
	samples := make([]auditSample, 0, len(parts))
	for _, part := range parts {
		code := strings.TrimSpace(part)
		if code == "" {
			continue
		}
		samples = append(samples, auditSample{Code: code, Label: "手动指定样本"})
	}
	return samples
}

func fetchDashboardAudit(client *http.Client, baseURL string, sample auditSample) (auditResult, error) {
	url := fmt.Sprintf("%s/api/v1/fund/%s/dashboard", baseURL, sample.Code)
	resp, err := client.Get(url)
	if err != nil {
		return auditResult{}, err
	}
	defer resp.Body.Close()

	var envelope apiEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return auditResult{}, err
	}

	if !envelope.Success {
		if envelope.Error != nil {
			return auditResult{}, fmt.Errorf("%s: %s", envelope.Error.Code, envelope.Error.Message)
		}
		return auditResult{}, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	if envelope.Data.Analysis == nil {
		return auditResult{}, fmt.Errorf("dashboard.analysis is nil")
	}

	return auditResult{
		Sample:   sample,
		Fund:     envelope.Data.Fund,
		Estimate: envelope.Data.Estimate,
		Analysis: envelope.Data.Analysis,
	}, nil
}

func printTextReport(results []auditResult, baseURL string) {
	fmt.Printf("Fund analysis audit | base_url=%s | samples=%d\n", baseURL, len(results))
	fmt.Println(strings.Repeat("=", 96))
	for _, item := range results {
		fmt.Printf("[%s] %s (%s)\n", item.Sample.Label, item.Fund.Name, item.Fund.ID)
		fmt.Printf("  分类: %s | 预估涨跌: %s | 数据源: %s\n", valueOrDash(item.Fund.CategoryName), valueOrDash(item.Estimate.ChangePercent), valueOrDash(item.Estimate.DataSource))
		fmt.Printf("  评分: total=%s | 加=%s 平=%s 减=%s | 风险=%s | 覆盖=%s\n",
			valueOrDash(item.Analysis.TotalScore),
			valueOrDash(item.Analysis.IncreasePercent),
			valueOrDash(item.Analysis.HoldPercent),
			valueOrDash(item.Analysis.DecreasePercent),
			valueOrDash(item.Analysis.RiskLevel),
			valueOrDash(item.Analysis.Confidence),
		)
		fmt.Printf("  口径: %s | 类型: %s | 最新披露: %s\n",
			valueOrDash(item.Analysis.AnalysisBasis),
			valueOrDash(item.Analysis.AnalysisType),
			valueOrDash(item.Analysis.LatestHoldingPeriod),
		)
		fmt.Printf("  结论: %s\n", valueOrDash(item.Analysis.Summary))
		if len(item.Analysis.EventImpacts) > 0 {
			fmt.Println("  事件:")
			for _, event := range item.Analysis.EventImpacts {
				fmt.Printf("    - [%s] %s：%s\n", valueOrDash(event.Impact), valueOrDash(event.Title), valueOrDash(event.Summary))
			}
		}
		if len(item.Analysis.Warnings) > 0 {
			fmt.Println("  警告:")
			for _, warning := range item.Analysis.Warnings {
				fmt.Printf("    - %s\n", warning)
			}
		}
		fmt.Println(strings.Repeat("-", 96))
	}
}

func valueOrDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "--"
	}
	return value
}
