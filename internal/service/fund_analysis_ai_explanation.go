package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

const (
	AIExplanationStatusReady    = "ready"
	AIExplanationStatusDisabled = "disabled"
	AIExplanationStatusFallback = "fallback"
	AIExplanationStatusRejected = "rejected"
	AIExplanationStatusFailed   = "failed"

	AIExplanationProviderDisabled = "disabled"

	AIExplanationCacheStatusGenerated    = "generated"
	AIExplanationCacheStatusSnapshotHit  = "snapshot_hit"
	AIExplanationCacheStatusNotCacheable = "not_cacheable"

	defaultAIExplanationTimeout = 3 * time.Second
)

const aiExplanationBoundaryNotice = "AI解释层只负责解释、归因、摘要和风险提示；不得改写 total_score、risk_level、increase/hold/decrease 分布或事件评分。"

// AIExplanationInput is the bounded context that may be passed to an AI provider.
//
// It intentionally contains the completed rule-based analysis plus structured
// evidence only. Providers must not receive authority to recalculate scores.
type AIExplanationInput struct {
	Fund           *domain.Fund
	Analysis       *domain.FundAnalysis
	Holdings       []domain.StockHolding
	SectorSnapshot *domain.FundSectorSnapshot
	ThemeSnapshot  *domain.FundThemeSnapshot
	Now            time.Time
}

// AIExplanationRequest is the provider-facing prompt payload.
type AIExplanationRequest struct {
	Fund               *domain.Fund
	Analysis           *domain.FundAnalysis
	Evidence           []domain.FundAnalysisAIExplanationCitation
	RuleRecommendation string
	BoundaryNotice     string
	GeneratedAt        time.Time
}

// AIExplanationProvider is the narrow boundary for a future LLM adapter.
//
// Implementations may produce natural-language explanations, but the output
// structure does not include score fields and every paragraph must cite an
// evidence code from AIExplanationRequest.Evidence.
type AIExplanationProvider interface {
	Name() string
	Generate(ctx context.Context, request AIExplanationRequest) (*domain.FundAnalysisAIExplanation, error)
}

// AIExplanationService validates provider output and supplies non-blocking fallback text.
type AIExplanationService struct {
	provider AIExplanationProvider
	timeout  time.Duration
}

type aiExplanationCacheMetadata struct {
	key               string
	expiresAt         time.Time
	invalidationBasis []string
}

func NewAIExplanationService(provider AIExplanationProvider) *AIExplanationService {
	return &AIExplanationService{
		provider: provider,
		timeout:  defaultAIExplanationTimeout,
	}
}

func (s *AIExplanationService) SetTimeout(timeout time.Duration) {
	if s == nil || timeout <= 0 {
		return
	}
	s.timeout = timeout
}

func (s *AIExplanationService) Explain(ctx context.Context, input AIExplanationInput) (*domain.FundAnalysisAIExplanation, error) {
	if input.Analysis == nil {
		return nil, nil
	}
	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}

	request := AIExplanationRequest{
		Fund:               input.Fund,
		Analysis:           input.Analysis,
		Evidence:           buildAIExplanationCitations(input),
		RuleRecommendation: analysisRuleRecommendation(input.Analysis),
		BoundaryNotice:     aiExplanationBoundaryNotice,
		GeneratedAt:        now,
	}
	cacheMetadata := buildAIExplanationCacheMetadata(input, request.Evidence, request.RuleRecommendation, now)
	if len(request.Evidence) == 0 {
		explanation := buildRejectedAIExplanation(request, "证据包不足，AI解释层无法确认可引用来源；请先补齐持仓、事件或行业/主题证据。")
		applyAIExplanationCacheMetadata(explanation, cacheMetadata, AIExplanationCacheStatusNotCacheable)
		return explanation, nil
	}

	provider := s.provider
	if provider == nil {
		explanation := buildFallbackAIExplanation(
			request,
			AIExplanationStatusDisabled,
			AIExplanationProviderDisabled,
			"AI解释层未启用；当前展示规则证据降级摘要，结论仍以结构化评分和证据包为准。",
			[]string{
				"未配置真实 AI provider，未发生外部模型调用。",
				"降级摘要只复述已有证据，不新增热点、行业判断或投资结论。",
			},
		)
		applyAIExplanationCacheMetadata(explanation, cacheMetadata, AIExplanationCacheStatusGenerated)
		return explanation, nil
	}

	providerName := strings.TrimSpace(provider.Name())
	if providerName == "" {
		providerName = "custom"
	}

	callCtx := ctx
	cancel := func() {}
	if s != nil && s.timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, s.timeout)
	}
	defer cancel()

	output, err := provider.Generate(callCtx, request)
	if err != nil {
		limitations := []string{"AI解释生成失败，已降级为规则证据摘要。"}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(callCtx.Err(), context.DeadlineExceeded) {
			limitations = []string{"AI解释生成超时，已降级为规则证据摘要。"}
		}
		explanation := buildFallbackAIExplanation(
			request,
			AIExplanationStatusFallback,
			providerName,
			"AI解释层暂不可用；页面继续展示规则型评分、结构化事件和证据包，不阻塞核心分析链路。",
			limitations,
		)
		applyAIExplanationCacheMetadata(explanation, cacheMetadata, AIExplanationCacheStatusGenerated)
		return explanation, nil
	}

	explanation := sanitizeAIExplanation(output, request, providerName)
	applyAIExplanationCacheMetadata(explanation, cacheMetadata, AIExplanationCacheStatusGenerated)
	return explanation, nil
}

func AttachAIExplanation(ctx context.Context, service *AIExplanationService, input AIExplanationInput) *domain.FundAnalysis {
	if input.Analysis == nil {
		return nil
	}
	if service == nil {
		service = NewAIExplanationService(nil)
	}
	explanation, err := service.Explain(ctx, input)
	if err == nil && explanation != nil {
		input.Analysis.AIExplanation = explanation
	}
	return input.Analysis
}

func BuildAIExplanationCacheKey(input AIExplanationInput) string {
	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	citations := buildAIExplanationCitations(input)
	ruleRecommendation := analysisRuleRecommendation(input.Analysis)
	return buildAIExplanationCacheMetadata(input, citations, ruleRecommendation, now).key
}

func CanReuseAIExplanation(explanation *domain.FundAnalysisAIExplanation, cacheKey string, now time.Time) bool {
	if explanation == nil {
		return false
	}
	cacheKey = strings.TrimSpace(cacheKey)
	if cacheKey == "" || strings.TrimSpace(explanation.CacheKey) != cacheKey {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}
	if !explanation.ExpiresAt.IsZero() && now.After(explanation.ExpiresAt) {
		return false
	}
	status := strings.TrimSpace(explanation.Status)
	return status == AIExplanationStatusReady ||
		status == AIExplanationStatusDisabled ||
		status == AIExplanationStatusFallback ||
		status == AIExplanationStatusRejected
}

func MarkAIExplanationSnapshotHit(explanation *domain.FundAnalysisAIExplanation) *domain.FundAnalysisAIExplanation {
	if explanation == nil {
		return nil
	}
	copyExplanation := *explanation
	copyExplanation.CacheStatus = AIExplanationCacheStatusSnapshotHit
	return &copyExplanation
}

func buildAIExplanationCacheMetadata(
	input AIExplanationInput,
	citations []domain.FundAnalysisAIExplanationCitation,
	ruleRecommendation string,
	now time.Time,
) aiExplanationCacheMetadata {
	if now.IsZero() {
		now = time.Now()
	}
	localNow := now.In(analysisCacheLocation())
	tradingDay := localNow.Format("2006-01-02")

	analysisVersion := ""
	analysisType := ""
	latestHoldingPeriod := ""
	if input.Analysis != nil {
		analysisVersion = strings.TrimSpace(input.Analysis.AnalysisVersion)
		analysisType = strings.TrimSpace(input.Analysis.AnalysisType)
		latestHoldingPeriod = strings.TrimSpace(input.Analysis.LatestHoldingPeriod)
	}
	fundID := ""
	if input.Fund != nil {
		fundID = strings.TrimSpace(input.Fund.ID)
	}

	evidenceSignature := aiExplanationEvidenceSignature(citations)
	rawKey := strings.Join([]string{
		"ai_explanation",
		fundID,
		analysisVersion,
		analysisType,
		tradingDay,
		latestHoldingPeriod,
		strings.TrimSpace(ruleRecommendation),
		evidenceSignature,
	}, "|")
	digest := sha256.Sum256([]byte(rawKey))

	expiresAt := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 23, 59, 59, 0, localNow.Location())
	return aiExplanationCacheMetadata{
		key:       hex.EncodeToString(digest[:]),
		expiresAt: expiresAt,
		invalidationBasis: uniqueNonEmptyStrings([]string{
			"analysis_version:" + analysisVersion,
			"trading_day:" + tradingDay,
			"holding_period:" + latestHoldingPeriod,
			"rule_recommendation:" + strings.TrimSpace(ruleRecommendation),
			"evidence_signature:" + evidenceSignature,
		}),
	}
}

func applyAIExplanationCacheMetadata(explanation *domain.FundAnalysisAIExplanation, metadata aiExplanationCacheMetadata, cacheStatus string) {
	if explanation == nil {
		return
	}
	explanation.CacheKey = metadata.key
	explanation.CacheStatus = strings.TrimSpace(cacheStatus)
	explanation.ExpiresAt = metadata.expiresAt
	explanation.InvalidationBasis = append([]string(nil), metadata.invalidationBasis...)
}

func aiExplanationEvidenceSignature(citations []domain.FundAnalysisAIExplanationCitation) string {
	if len(citations) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(citations))
	for _, citation := range citations {
		parts = append(parts, strings.Join([]string{
			strings.TrimSpace(citation.Code),
			strings.TrimSpace(citation.SourceType),
			strings.TrimSpace(citation.SourceScope),
			strings.TrimSpace(citation.Title),
			strings.TrimSpace(citation.Summary),
		}, "|"))
	}
	sort.Strings(parts)
	digest := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(digest[:])
}

func analysisCacheLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

func buildRejectedAIExplanation(request AIExplanationRequest, summary string) *domain.FundAnalysisAIExplanation {
	return &domain.FundAnalysisAIExplanation{
		Status:             AIExplanationStatusRejected,
		Provider:           AIExplanationProviderDisabled,
		GeneratedAt:        request.GeneratedAt,
		RuleRecommendation: request.RuleRecommendation,
		BoundaryNotice:     request.BoundaryNotice,
		Summary:            summary,
		Limitations: []string{
			"AI解释必须基于可引用证据；没有来源支撑时不生成热点、行业判断或投资结论。",
			"规则型评分仍可展示，但解释层应按“无法确认”处理。",
		},
	}
}

func buildFallbackAIExplanation(
	request AIExplanationRequest,
	status string,
	provider string,
	summary string,
	limitations []string,
) *domain.FundAnalysisAIExplanation {
	explanation := &domain.FundAnalysisAIExplanation{
		Status:             status,
		Provider:           provider,
		GeneratedAt:        request.GeneratedAt,
		RuleRecommendation: request.RuleRecommendation,
		BoundaryNotice:     request.BoundaryNotice,
		Summary:            summary,
		Attribution:        fallbackAttributionSections(request),
		RiskNotes:          fallbackRiskSections(request),
		Limitations: append([]string{
			"AI解释输出不参与评分，也不会改变规则建议分布。",
		}, limitations...),
	}
	explanation.Citations = citationsForSections(request.Evidence, explanation.Attribution, explanation.RiskNotes)
	if len(explanation.Citations) == 0 {
		explanation.Citations = firstNCitations(request.Evidence, 4)
	}
	return explanation
}

func fallbackAttributionSections(request AIExplanationRequest) []domain.FundAnalysisAIExplanationSection {
	if request.Analysis == nil {
		return nil
	}
	sections := make([]domain.FundAnalysisAIExplanationSection, 0, 2)
	for _, item := range request.Analysis.PrimaryEvidence {
		code := strings.TrimSpace(item.Code)
		if code == "" || !citationExists(request.Evidence, code) {
			continue
		}
		sections = append(sections, domain.FundAnalysisAIExplanationSection{
			Title:         item.Title,
			Summary:       item.Summary,
			CitationCodes: []string{code},
		})
		if len(sections) >= 2 {
			return sections
		}
	}
	for _, citation := range request.Evidence {
		if strings.TrimSpace(citation.Code) == "" {
			continue
		}
		if citation.SourceType == "counter_evidence" || citation.SourceType == "confidence" {
			continue
		}
		sections = append(sections, domain.FundAnalysisAIExplanationSection{
			Title:         citation.Title,
			Summary:       citation.Summary,
			CitationCodes: []string{citation.Code},
		})
		if len(sections) >= 2 {
			return sections
		}
	}
	return sections
}

func fallbackRiskSections(request AIExplanationRequest) []domain.FundAnalysisAIExplanationSection {
	if request.Analysis == nil {
		return nil
	}
	sections := make([]domain.FundAnalysisAIExplanationSection, 0, 2)
	for _, item := range request.Analysis.CounterEvidence {
		code := strings.TrimSpace(item.Code)
		if code == "" || !citationExists(request.Evidence, code) {
			continue
		}
		sections = append(sections, domain.FundAnalysisAIExplanationSection{
			Title:         item.Title,
			Summary:       item.Summary,
			CitationCodes: []string{code},
		})
		if len(sections) >= 2 {
			return sections
		}
	}
	for _, citation := range request.Evidence {
		if citation.SourceType != "counter_evidence" && citation.SourceType != "confidence" {
			continue
		}
		sections = append(sections, domain.FundAnalysisAIExplanationSection{
			Title:         citation.Title,
			Summary:       citation.Summary,
			CitationCodes: []string{citation.Code},
		})
		if len(sections) >= 2 {
			return sections
		}
	}
	return sections
}

func sanitizeAIExplanation(output *domain.FundAnalysisAIExplanation, request AIExplanationRequest, providerName string) *domain.FundAnalysisAIExplanation {
	if output == nil || strings.TrimSpace(output.Summary) == "" {
		return buildFallbackAIExplanation(
			request,
			AIExplanationStatusFallback,
			providerName,
			"AI解释层没有返回可展示内容，已降级为规则证据摘要。",
			[]string{"AI provider 返回空内容。"},
		)
	}

	validCodes := make(map[string]domain.FundAnalysisAIExplanationCitation, len(request.Evidence))
	for _, citation := range request.Evidence {
		code := strings.TrimSpace(citation.Code)
		if code != "" {
			validCodes[code] = citation
		}
	}

	removedUnsupported := 0
	attribution, attributionRemoved := sanitizeAIExplanationSections(output.Attribution, validCodes)
	riskNotes, riskRemoved := sanitizeAIExplanationSections(output.RiskNotes, validCodes)
	removedUnsupported += attributionRemoved + riskRemoved

	if len(attribution) == 0 && len(riskNotes) == 0 {
		return buildFallbackAIExplanation(
			request,
			AIExplanationStatusFallback,
			providerName,
			"AI解释层未提供有效引用，已降级为规则证据摘要；无来源支撑的段落不会展示。",
			[]string{"AI provider 输出缺少有效 citation_codes。"},
		)
	}

	status := strings.TrimSpace(output.Status)
	if status == "" {
		status = AIExplanationStatusReady
	}
	provider := strings.TrimSpace(output.Provider)
	if provider == "" {
		provider = providerName
	}
	generatedAt := output.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = request.GeneratedAt
	}
	limitations := append([]string(nil), output.Limitations...)
	if removedUnsupported > 0 {
		limitations = append(limitations, fmt.Sprintf("已移除 %d 段无有效来源引用的 AI 输出。", removedUnsupported))
	}
	limitations = append(limitations, "AI解释输出不参与评分，也不会改变规则建议分布。")

	explanation := &domain.FundAnalysisAIExplanation{
		Status:             status,
		Provider:           provider,
		Model:              strings.TrimSpace(output.Model),
		GeneratedAt:        generatedAt,
		RuleRecommendation: request.RuleRecommendation,
		BoundaryNotice:     request.BoundaryNotice,
		Summary:            strings.TrimSpace(output.Summary),
		Attribution:        attribution,
		RiskNotes:          riskNotes,
		Limitations:        uniqueNonEmptyStrings(limitations),
	}
	explanation.Citations = citationsForSections(request.Evidence, explanation.Attribution, explanation.RiskNotes)
	return explanation
}

func sanitizeAIExplanationSections(
	sections []domain.FundAnalysisAIExplanationSection,
	validCodes map[string]domain.FundAnalysisAIExplanationCitation,
) ([]domain.FundAnalysisAIExplanationSection, int) {
	result := make([]domain.FundAnalysisAIExplanationSection, 0, len(sections))
	removed := 0
	for _, section := range sections {
		summary := strings.TrimSpace(section.Summary)
		title := strings.TrimSpace(section.Title)
		if summary == "" || title == "" {
			removed++
			continue
		}
		validCitationCodes := make([]string, 0, len(section.CitationCodes))
		for _, code := range section.CitationCodes {
			code = strings.TrimSpace(code)
			if code == "" {
				continue
			}
			if _, ok := validCodes[code]; ok {
				validCitationCodes = append(validCitationCodes, code)
			}
		}
		validCitationCodes = uniqueNonEmptyStrings(validCitationCodes)
		if len(validCitationCodes) == 0 {
			removed++
			continue
		}
		result = append(result, domain.FundAnalysisAIExplanationSection{
			Title:         title,
			Summary:       summary,
			CitationCodes: validCitationCodes,
		})
	}
	return result, removed
}

func citationsForSections(
	available []domain.FundAnalysisAIExplanationCitation,
	sectionGroups ...[]domain.FundAnalysisAIExplanationSection,
) []domain.FundAnalysisAIExplanationCitation {
	availableByCode := make(map[string]domain.FundAnalysisAIExplanationCitation, len(available))
	for _, citation := range available {
		code := strings.TrimSpace(citation.Code)
		if code != "" {
			availableByCode[code] = citation
		}
	}

	seen := make(map[string]struct{})
	result := make([]domain.FundAnalysisAIExplanationCitation, 0)
	for _, sections := range sectionGroups {
		for _, section := range sections {
			for _, code := range section.CitationCodes {
				code = strings.TrimSpace(code)
				if code == "" {
					continue
				}
				if _, ok := seen[code]; ok {
					continue
				}
				citation, ok := availableByCode[code]
				if !ok {
					continue
				}
				seen[code] = struct{}{}
				result = append(result, citation)
			}
		}
	}
	return result
}

func firstNCitations(citations []domain.FundAnalysisAIExplanationCitation, limit int) []domain.FundAnalysisAIExplanationCitation {
	if limit <= 0 || len(citations) == 0 {
		return nil
	}
	if len(citations) < limit {
		limit = len(citations)
	}
	return append([]domain.FundAnalysisAIExplanationCitation(nil), citations[:limit]...)
}

func citationExists(citations []domain.FundAnalysisAIExplanationCitation, code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	for _, citation := range citations {
		if strings.TrimSpace(citation.Code) == code {
			return true
		}
	}
	return false
}

func buildAIExplanationCitations(input AIExplanationInput) []domain.FundAnalysisAIExplanationCitation {
	if input.Analysis == nil {
		return nil
	}
	citationsByCode := make(map[string]domain.FundAnalysisAIExplanationCitation)
	add := func(citation domain.FundAnalysisAIExplanationCitation) {
		citation.Code = strings.TrimSpace(citation.Code)
		citation.Title = strings.TrimSpace(citation.Title)
		citation.Summary = strings.TrimSpace(citation.Summary)
		citation.SourceType = strings.TrimSpace(citation.SourceType)
		citation.SourceScope = strings.TrimSpace(citation.SourceScope)
		if citation.Code == "" || citation.Title == "" {
			return
		}
		if citation.SourceType == "" {
			citation.SourceType = "evidence"
		}
		if _, exists := citationsByCode[citation.Code]; exists {
			return
		}
		citationsByCode[citation.Code] = citation
	}

	for _, item := range input.Analysis.PrimaryEvidence {
		add(domain.FundAnalysisAIExplanationCitation{
			Code:        item.Code,
			SourceType:  "primary_evidence",
			SourceScope: item.SourceScope,
			Title:       item.Title,
			Summary:     item.Summary,
		})
	}
	for _, item := range input.Analysis.CounterEvidence {
		add(domain.FundAnalysisAIExplanationCitation{
			Code:        item.Code,
			SourceType:  "counter_evidence",
			SourceScope: item.SourceScope,
			Title:       item.Title,
			Summary:     item.Summary,
		})
	}
	for _, event := range input.Analysis.EventImpacts {
		add(domain.FundAnalysisAIExplanationCitation{
			Code:        event.Code,
			SourceType:  "event",
			SourceScope: event.TargetScope,
			Title:       event.Title,
			Summary:     event.Summary,
		})
	}
	for _, factor := range input.Analysis.ConfidenceFactors {
		add(domain.FundAnalysisAIExplanationCitation{
			Code:        "confidence:" + factor.Code,
			SourceType:  "confidence",
			SourceScope: "methodology",
			Title:       factor.Name,
			Summary:     factor.Summary,
		})
	}

	if input.SectorSnapshot != nil && strings.TrimSpace(input.SectorSnapshot.PrimarySectorCode) != "" {
		add(domain.FundAnalysisAIExplanationCitation{
			Code:        "sector:" + input.SectorSnapshot.PrimarySectorCode,
			SourceType:  "sector_snapshot",
			SourceScope: "exposure",
			Title:       "主行业：" + strings.TrimSpace(input.SectorSnapshot.PrimarySectorName),
			Summary:     primarySectorCitationSummary(input.SectorSnapshot),
		})
	}
	if input.ThemeSnapshot != nil && strings.TrimSpace(input.ThemeSnapshot.PrimaryThemeCode) != "" {
		add(domain.FundAnalysisAIExplanationCitation{
			Code:        "theme:" + input.ThemeSnapshot.PrimaryThemeCode,
			SourceType:  "theme_snapshot",
			SourceScope: "exposure",
			Title:       "主主题：" + strings.TrimSpace(input.ThemeSnapshot.PrimaryThemeName),
			Summary:     primaryThemeCitationSummary(input.ThemeSnapshot),
		})
	}

	holdings := append([]domain.StockHolding(nil), input.Holdings...)
	sort.SliceStable(holdings, func(i, j int) bool {
		return holdings[i].HoldingRatio.GreaterThan(holdings[j].HoldingRatio)
	})
	for _, holding := range holdings {
		code := strings.TrimSpace(holding.StockCode)
		if code == "" || !holding.HoldingRatio.GreaterThan(decimal.Zero) {
			continue
		}
		add(domain.FundAnalysisAIExplanationCitation{
			Code:        "holding:" + code,
			SourceType:  "holding",
			SourceScope: "holding",
			Title:       "持仓：" + strings.TrimSpace(holding.StockName) + "（" + code + "）",
			Summary:     "当前持仓权重约 " + holding.HoldingRatio.Round(2).StringFixed(2) + "%。",
		})
		if countCitationsBySource(citationsByCode, "holding") >= 5 {
			break
		}
	}

	result := make([]domain.FundAnalysisAIExplanationCitation, 0, len(citationsByCode))
	for _, citation := range citationsByCode {
		result = append(result, citation)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return citationRank(result[i]) < citationRank(result[j])
	})
	return result
}

func primarySectorCitationSummary(snapshot *domain.FundSectorSnapshot) string {
	if snapshot == nil {
		return ""
	}
	weight := ""
	for _, item := range snapshot.Breakdown {
		if item.SectorCode == snapshot.PrimarySectorCode && item.WeightPercent.GreaterThan(decimal.Zero) {
			weight = "，暴露权重约 " + item.WeightPercent.Round(2).StringFixed(2) + "%"
			break
		}
	}
	return "当前主行业为 " + strings.TrimSpace(snapshot.PrimarySectorName) + weight + "。"
}

func primaryThemeCitationSummary(snapshot *domain.FundThemeSnapshot) string {
	if snapshot == nil {
		return ""
	}
	weight := ""
	for _, item := range snapshot.Breakdown {
		if item.ThemeCode == snapshot.PrimaryThemeCode && item.WeightPercent.GreaterThan(decimal.Zero) {
			weight = "，暴露权重约 " + item.WeightPercent.Round(2).StringFixed(2) + "%"
			break
		}
	}
	return "当前主主题为 " + strings.TrimSpace(snapshot.PrimaryThemeName) + weight + "。"
}

func citationRank(citation domain.FundAnalysisAIExplanationCitation) int {
	switch citation.SourceType {
	case "primary_evidence":
		return 10
	case "counter_evidence":
		return 20
	case "event":
		switch citation.SourceScope {
		case "holding":
			return 30
		case "exposure", "macro":
			return 35
		case "fund":
			return 40
		default:
			return 45
		}
	case "sector_snapshot", "theme_snapshot":
		return 50
	case "holding":
		return 60
	case "confidence":
		return 70
	default:
		return 90
	}
}

func countCitationsBySource(citations map[string]domain.FundAnalysisAIExplanationCitation, sourceType string) int {
	count := 0
	for _, citation := range citations {
		if citation.SourceType == sourceType {
			count++
		}
	}
	return count
}

func analysisRuleRecommendation(analysis *domain.FundAnalysis) string {
	return DominantRecommendationFromAnalysis(analysis)
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
