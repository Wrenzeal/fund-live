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
	AnalysisType         string                 `json:"analysis_type"`
	AnalysisBasis        string                 `json:"analysis_basis"`
	TotalScore           string                 `json:"total_score"`
	Confidence           string                 `json:"confidence"`
	RiskLevel            string                 `json:"risk_level"`
	IncreasePercent      string                 `json:"increase_percent"`
	HoldPercent          string                 `json:"hold_percent"`
	DecreasePercent      string                 `json:"decrease_percent"`
	LatestHoldingPeriod  string                 `json:"latest_holding_period"`
	Summary              string                 `json:"summary"`
	Reasons              []string               `json:"reasons"`
	Warnings             []string               `json:"warnings"`
	EventImpacts         []analysisEvent        `json:"event_impacts"`
	ModuleScores         []analysisModule       `json:"module_scores"`
	ConfidenceFactors    []confidenceFactor     `json:"confidence_factors"`
	PrimaryEvidence      []analysisEvidenceItem `json:"primary_evidence"`
	CounterEvidence      []analysisEvidenceItem `json:"counter_evidence"`
	ConfidenceDeductions []string               `json:"confidence_deductions"`
	AIExplanation        *aiExplanationInfo     `json:"ai_explanation"`
}

type analysisEvent struct {
	Code           string   `json:"code"`
	Title          string   `json:"title"`
	Impact         string   `json:"impact"`
	Summary        string   `json:"summary"`
	TargetScope    string   `json:"target_scope"`
	Strength       string   `json:"strength"`
	Horizon        string   `json:"horizon"`
	RelatedSymbols []string `json:"related_symbols"`
	WeightHint     string   `json:"weight_hint"`
}

type confidenceFactor struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Level   string `json:"level"`
	Score   string `json:"score"`
	Summary string `json:"summary"`
}

type analysisEvidenceItem struct {
	Code           string   `json:"code"`
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	EvidenceType   string   `json:"evidence_type"`
	SourceScope    string   `json:"source_scope"`
	Impact         string   `json:"impact"`
	Strength       string   `json:"strength"`
	Horizon        string   `json:"horizon"`
	RelatedSymbols []string `json:"related_symbols"`
	WeightHint     string   `json:"weight_hint"`
}

type analysisModule struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Score   string `json:"score"`
	Summary string `json:"summary"`
}

type aiExplanationInfo struct {
	Status             string                  `json:"status"`
	Provider           string                  `json:"provider"`
	Model              string                  `json:"model"`
	CacheKey           string                  `json:"cache_key"`
	CacheStatus        string                  `json:"cache_status"`
	ExpiresAt          string                  `json:"expires_at"`
	InvalidationBasis  []string                `json:"invalidation_basis"`
	RuleRecommendation string                  `json:"rule_recommendation"`
	BoundaryNotice     string                  `json:"boundary_notice"`
	Summary            string                  `json:"summary"`
	Attribution        []aiExplanationSection  `json:"attribution"`
	RiskNotes          []aiExplanationSection  `json:"risk_notes"`
	Citations          []aiExplanationCitation `json:"citations"`
	Limitations        []string                `json:"limitations"`
}

type aiExplanationSection struct {
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	CitationCodes []string `json:"citation_codes"`
}

type aiExplanationCitation struct {
	Code        string `json:"code"`
	SourceType  string `json:"source_type"`
	SourceScope string `json:"source_scope"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
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
		fmt.Printf("  直觉复核: %s\n", sampleHeuristicVerdict(item))
		if item.Analysis.AIExplanation != nil {
			fmt.Printf("  AI解释层: status=%s | provider=%s | cache=%s | rule=%s\n",
				valueOrDash(item.Analysis.AIExplanation.Status),
				valueOrDash(item.Analysis.AIExplanation.Provider),
				valueOrDash(item.Analysis.AIExplanation.CacheStatus),
				valueOrDash(item.Analysis.AIExplanation.RuleRecommendation),
			)
			if item.Analysis.AIExplanation.ExpiresAt != "" {
				fmt.Printf("    缓存失效: %s\n", item.Analysis.AIExplanation.ExpiresAt)
			}
			if len(item.Analysis.AIExplanation.InvalidationBasis) > 0 {
				fmt.Printf("    失效依据: %s\n", strings.Join(item.Analysis.AIExplanation.InvalidationBasis, "；"))
			}
			fmt.Printf("    边界: %s\n", valueOrDash(item.Analysis.AIExplanation.BoundaryNotice))
			fmt.Printf("    摘要: %s\n", valueOrDash(item.Analysis.AIExplanation.Summary))
			if len(item.Analysis.AIExplanation.Attribution) > 0 {
				fmt.Println("    归因段落:")
				for _, section := range item.Analysis.AIExplanation.Attribution {
					fmt.Printf("      - %s：%s（引用：%s）\n", valueOrDash(section.Title), valueOrDash(section.Summary), valueOrDash(strings.Join(section.CitationCodes, ",")))
				}
			}
			if len(item.Analysis.AIExplanation.RiskNotes) > 0 {
				fmt.Println("    风险段落:")
				for _, section := range item.Analysis.AIExplanation.RiskNotes {
					fmt.Printf("      - %s：%s（引用：%s）\n", valueOrDash(section.Title), valueOrDash(section.Summary), valueOrDash(strings.Join(section.CitationCodes, ",")))
				}
			}
			if len(item.Analysis.AIExplanation.Limitations) > 0 {
				fmt.Println("    降级 / 限制:")
				for _, limitation := range item.Analysis.AIExplanation.Limitations {
					fmt.Printf("      - %s\n", limitation)
				}
			}
		}
		if len(item.Analysis.ConfidenceFactors) > 0 {
			fmt.Println("  可信度拆解:")
			for _, factor := range item.Analysis.ConfidenceFactors {
				fmt.Printf("    - [%s/%s] %s：%s\n", valueOrDash(factor.Level), valueOrDash(factor.Score), valueOrDash(factor.Name), valueOrDash(factor.Summary))
			}
		}
		if gaps := dataGapSummaries(item.Analysis); len(gaps) > 0 {
			fmt.Println("  数据缺口 / 可信度扣分:")
			for _, gap := range gaps {
				fmt.Printf("    - %s\n", gap)
			}
		}
		if len(item.Analysis.PrimaryEvidence) > 0 {
			fmt.Println("  主证据:")
			for _, evidence := range item.Analysis.PrimaryEvidence {
				fmt.Printf("    - [%s/%s/%s] %s：%s\n", valueOrDash(evidence.SourceScope), valueOrDash(evidence.Impact), valueOrDash(evidence.Strength), valueOrDash(evidence.Title), valueOrDash(evidence.Summary))
			}
		}
		if len(item.Analysis.CounterEvidence) > 0 {
			fmt.Println("  反方证据 / 限制:")
			for _, evidence := range item.Analysis.CounterEvidence {
				fmt.Printf("    - [%s/%s/%s] %s：%s\n", valueOrDash(evidence.SourceScope), valueOrDash(evidence.Impact), valueOrDash(evidence.Strength), valueOrDash(evidence.Title), valueOrDash(evidence.Summary))
			}
		}
		if len(item.Analysis.EventImpacts) > 0 {
			fmt.Println("  事件:")
			for _, event := range item.Analysis.EventImpacts {
				fmt.Printf("    - [%s/%s/%s] %s：%s\n", valueOrDash(event.TargetScope), valueOrDash(event.Impact), valueOrDash(event.Strength), valueOrDash(event.Title), valueOrDash(event.Summary))
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

func sampleHeuristicVerdict(item auditResult) string {
	if item.Analysis == nil {
		return "需复核：analysis 为空"
	}
	if weakFactors := weakConfidenceFactors(item.Analysis); len(weakFactors) > 0 {
		return "需人工复核：存在低可信度因子（" + strings.Join(weakFactors, "、") + "），不能形成强结论"
	}
	if item.Analysis.Confidence == "low" {
		return "需人工复核：可信度偏低，不能形成强结论"
	}
	if len(item.Analysis.PrimaryEvidence) == 0 {
		return "需人工复核：缺少主证据"
	}
	if len(item.Analysis.CounterEvidence) == 0 && len(item.Analysis.Warnings) == 0 {
		return "基本符合：但缺少反方证据，建议继续人工抽查"
	}
	return "初步符合：已有主证据、反方限制和可信度拆解"
}

func weakConfidenceFactors(analysis *analysisInfo) []string {
	if analysis == nil {
		return nil
	}
	result := make([]string, 0, 3)
	for _, factor := range analysis.ConfidenceFactors {
		if strings.TrimSpace(factor.Level) != "low" {
			continue
		}
		name := strings.TrimSpace(factor.Name)
		if name == "" {
			name = strings.TrimSpace(factor.Code)
		}
		if name == "" {
			name = "未知因子"
		}
		result = append(result, name)
		if len(result) >= 3 {
			break
		}
	}
	return result
}

func dataGapSummaries(analysis *analysisInfo) []string {
	if analysis == nil {
		return nil
	}
	gaps := append([]string(nil), analysis.ConfidenceDeductions...)
	for _, factor := range analysis.ConfidenceFactors {
		if factor.Level == "low" {
			gaps = append(gaps, factor.Name+"偏弱："+factor.Summary)
		}
	}
	if len(gaps) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(gaps))
	result := make([]string, 0, len(gaps))
	for _, gap := range gaps {
		gap = strings.TrimSpace(gap)
		if gap == "" {
			continue
		}
		if _, ok := seen[gap]; ok {
			continue
		}
		seen[gap] = struct{}{}
		result = append(result, gap)
		if len(result) >= 4 {
			break
		}
	}
	return result
}

func valueOrDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "--"
	}
	return value
}
