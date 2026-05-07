package service

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

const (
	FundAnalysisVersionV1 = "baseline_v1"

	FundAnalysisTypeDirectHoldings = "direct_holdings"
	FundAnalysisTypeTrackedETF     = "tracked_etf"
	FundAnalysisTypeQDIIHoldings   = "qdii_holdings"
)

type FundAnalysisInput struct {
	Fund                   *domain.Fund
	Estimate               *domain.FundEstimate
	TimeSeries             []domain.TimeSeriesPoint
	SectorSnapshot         *domain.FundSectorSnapshot
	ThemeSnapshot          *domain.FundThemeSnapshot
	PreviousSectorSnapshot *domain.FundSectorSnapshot
	PreviousThemeSnapshot  *domain.FundThemeSnapshot
	CurrentHoldingEvents   []domain.FundAnalysisEventImpact
	CurrentFundEvents      []domain.FundAnalysisEventImpact
	CurrentTargetEvents    []domain.FundAnalysisEventImpact
	CurrentMacroEvents     []domain.FundAnalysisEventImpact
	CurrentIndexEvents     []domain.FundAnalysisEventImpact
	Holdings               []domain.StockHolding
	PreviousHoldings       []domain.StockHolding
	PreviousHoldingPeriod  string
	HoldingsSource         string
	Now                    time.Time
}

type FundAnalysisService struct{}

type holdingDelta struct {
	StockCode      string
	StockName      string
	CurrentWeight  float64
	PreviousWeight float64
	Delta          float64
}

type holdingChangeInsights struct {
	PreviousPeriod string
	OverlapCount   int
	NewEntries     int
	RemovedEntries int
	TopIncrease    *holdingDelta
	TopDecrease    *holdingDelta
	Top3Delta      float64
}

type exposureShiftInsight struct {
	Scope          string
	PreviousCode   string
	PreviousName   string
	PreviousWeight float64
	CurrentCode    string
	CurrentName    string
	CurrentWeight  float64
	Changed        bool
	WeightDelta    float64
}

type currentExposureEventInsight struct {
	Name     string
	Scope    string
	Count    int
	Impact   string
	Strength string
	Summary  string
	Highlights []string
}

type exposureBreakdownDelta struct {
	Code           string
	Name           string
	CurrentWeight  float64
	PreviousWeight float64
	Delta          float64
}

type exposureBreakdownInsights struct {
	SectorIncrease *exposureBreakdownDelta
	SectorDecrease *exposureBreakdownDelta
	ThemeIncrease  *exposureBreakdownDelta
	ThemeDecrease  *exposureBreakdownDelta
	StyleDrift     bool
}

func NewFundAnalysisService() *FundAnalysisService {
	return &FundAnalysisService{}
}

func (s *FundAnalysisService) Build(input FundAnalysisInput) *domain.FundAnalysis {
	if input.Fund == nil || input.Estimate == nil {
		return nil
	}

	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}

	analysisType, analysisBasis := normalizeAnalysisBasis(input.HoldingsSource)
	coveragePercent := resolveCoveragePercent(input.Estimate, input.Holdings)
	holdingCoverage := clamp(coveragePercent/100, 0, 1)
	changePercent := decimalToFloat(input.Estimate.ChangePercent)
	intradayMomentum := computeIntradayMomentum(input.TimeSeries)
	intradayRange := computeIntradayRange(input.TimeSeries)
	positiveBreadth := computePositiveBreadth(input.Estimate.HoldingDetails)
	avgAbsMove := computeAverageAbsoluteMove(input.Estimate.HoldingDetails)
	sectorConfidenceScore := confidenceScore(sectorConfidence(input.SectorSnapshot))
	themeConfidenceScore := confidenceScore(themeConfidence(input.ThemeSnapshot))
	structureClarity := primaryExposureWeight(input.SectorSnapshot, input.ThemeSnapshot)
	latestPeriod, latestPeriodTime := latestHoldingPeriod(input.Holdings)
	disclosureAgeDays := holdingDisclosureAgeDays(now, latestPeriodTime)
	topHoldingWeight := topHoldingWeight(input.Holdings)
	concentrationWeight := math.Max(structureClarity, topHoldingWeight)
	classificationSignal := recognizedClassificationSignal(input.SectorSnapshot, input.ThemeSnapshot)
	holdingChanges := analyzeHoldingChanges(input.Holdings, input.PreviousHoldings, input.PreviousHoldingPeriod)
	exposureShift := analyzeExposureShift(input.SectorSnapshot, input.ThemeSnapshot, input.PreviousSectorSnapshot, input.PreviousThemeSnapshot)
	currentFocusEvents := mergeCurrentEventImpacts(input.CurrentMacroEvents, input.CurrentIndexEvents, input.CurrentHoldingEvents, input.CurrentFundEvents, input.CurrentTargetEvents)
	currentExposureEvent := analyzeCurrentExposureEvent(input.CurrentHoldingEvents, input.SectorSnapshot, input.ThemeSnapshot)
	exposureBreakdowns := analyzeExposureBreakdownChanges(input.SectorSnapshot, input.ThemeSnapshot, input.PreviousSectorSnapshot, input.PreviousThemeSnapshot)
	coveragePenalty := lowCoveragePenalty(coveragePercent)
	stalePenalty := staleDisclosurePenalty(disclosureAgeDays)
	classificationPenalty := lowClassificationPenalty(input.SectorSnapshot, input.ThemeSnapshot)

	trendScore := clamp(
		0.65*centeredMarketScore(changePercent, 4.0)+
			0.35*centeredMarketScore(intradayMomentum, 2.5),
		12,
		94,
	)

	structureScore := clamp(
		0.45*(holdingCoverage*100)+
			0.35*((sectorConfidenceScore+themeConfidenceScore)/2)+
			0.20*(40+structureClarity*0.9),
		15,
		96,
	)

	heatScore := clamp(
		0.40*clamp(30+avgAbsMove*10, 28, 94)+
			0.35*(35+positiveBreadth*50)+
			0.25*classificationSignal,
		18,
		95,
	)
	heatScore = clamp(heatScore-coveragePenalty*0.35-classificationPenalty*0.4, 15, 95)

	riskPenalty := clamp(
		math.Abs(changePercent)*10+
			intradayRange*8+
			math.Max(concentrationWeight-35, 0)*0.9+
			math.Max(topHoldingWeight-12, 0)*1.2+
			coveragePenalty*0.8+
			stalePenalty*0.5,
		0,
		85,
	)
	riskScore := clamp(82-riskPenalty+holdingCoverage*10, 8, 92)

	allocationScore := clamp(
		60-
			math.Abs(changePercent)*9-
			intradayRange*4+
			holdingCoverage*12-
			math.Max(changePercent-2.2, 0)*5,
		10,
		88,
	)
	if changePercent >= -1.5 && changePercent <= 0.8 {
		allocationScore = clamp(allocationScore+8, 10, 90)
	}
	if riskScore < 45 {
		allocationScore = clamp(allocationScore-8, 8, 84)
	}
	allocationScore = clamp(allocationScore-coveragePenalty*0.3-stalePenalty*0.25, 8, 88)

	if isMomentumFriendlyIndexLike(input.Fund, analysisType, trendScore, changePercent) {
		riskScore = clamp(riskScore+12, 8, 92)
		allocationScore = clamp(allocationScore+22, 8, 88)
	} else if isIndexLikeAnalysis(input.Fund, analysisType) && trendScore >= 70 && changePercent >= 1.0 {
		allocationScore = clamp(allocationScore+10, 8, 88)
	}

	eventFreshness := eventFreshnessScore(now, latestPeriodTime)
	traceabilityScore := clamp(55+holdingCoverage*20, 35, 85)
	if analysisType == FundAnalysisTypeTrackedETF {
		traceabilityScore = clamp(traceabilityScore+10, 35, 92)
	}
	if classificationSignal >= 65 {
		traceabilityScore = clamp(traceabilityScore+4, 35, 94)
	}
	eventScore := clamp(0.70*eventFreshness+0.30*traceabilityScore, 20, 90)
	eventScore = clamp(eventScore+holdingChangeEventAdjustment(holdingChanges), 18, 92)
	eventScore = clamp(eventScore+exposureShiftEventAdjustment(exposureShift), 18, 92)
	eventScore = clamp(eventScore+currentHoldingEventAdjustment(currentFocusEvents), 18, 92)
	eventScore = clamp(eventScore+currentExposureEventAdjustment(currentExposureEvent), 18, 92)
	eventScore = clamp(eventScore+exposureBreakdownEventAdjustment(exposureBreakdowns), 18, 92)
	eventScore = clamp(eventScore-stalePenalty*0.4, 18, 90)

	decisionBias := 0.32*(trendScore-50) +
		0.18*(structureScore-50) +
		0.16*(heatScore-50) +
		0.18*(allocationScore-50) +
		0.10*(eventScore-50) +
		0.16*(riskScore-50)
	decisionBias -= coveragePenalty*0.28 + stalePenalty*0.22 + classificationPenalty*0.20
	totalScore := clamp(50+decisionBias, 5, 95)
	increasePercent, holdPercent, decreasePercent := recommendationDistribution(decisionBias)

	confidence := deriveAnalysisConfidence(coveragePercent, sectorConfidence(input.SectorSnapshot), themeConfidence(input.ThemeSnapshot), analysisType)
	riskLevel := deriveRiskLevel(riskScore)

	moduleScores := []domain.FundAnalysisModuleScore{
		{
			Code:    "trend",
			Name:    "趋势",
			Score:   scoreDecimal(trendScore),
			Summary: buildTrendSummary(changePercent, intradayMomentum),
		},
		{
			Code:    "structure",
			Name:    "结构",
			Score:   scoreDecimal(structureScore),
			Summary: buildStructureSummary(coveragePercent, input.SectorSnapshot, input.ThemeSnapshot),
		},
		{
			Code:    "heat",
			Name:    "热度",
			Score:   scoreDecimal(heatScore),
			Summary: buildHeatSummary(avgAbsMove, positiveBreadth, input.ThemeSnapshot),
		},
		{
			Code:    "risk",
			Name:    "风险",
			Score:   scoreDecimal(riskScore),
			Summary: buildRiskSummary(intradayRange, concentrationWeight, riskLevel),
		},
		{
			Code:    "allocation",
			Name:    "性价比",
			Score:   scoreDecimal(allocationScore),
			Summary: buildAllocationSummary(changePercent, allocationScore),
		},
		{
			Code:    "event",
			Name:    "事件",
			Score:   scoreDecimal(eventScore),
			Summary: buildEventSummary(latestPeriod, analysisBasis, currentFocusEvents, currentExposureEvent, exposureBreakdowns, holdingChanges, exposureShift),
		},
	}

	reasons := buildAnalysisReasons(trendScore, structureScore, heatScore, allocationScore, eventScore, input.SectorSnapshot, input.ThemeSnapshot, analysisBasis, latestPeriod, currentFocusEvents, currentExposureEvent, exposureBreakdowns, holdingChanges, exposureShift)
	warnings := buildAnalysisWarnings(riskScore, confidence, eventScore, latestPeriod, disclosureAgeDays, changePercent, concentrationWeight, currentFocusEvents, currentExposureEvent, exposureBreakdowns, holdingChanges, exposureShift)
	eventImpacts := buildEventImpacts(analysisBasis, latestPeriod, disclosureAgeDays, confidence, concentrationWeight, currentFocusEvents, currentExposureEvent, exposureBreakdowns, holdingChanges, exposureShift, input.Estimate.HoldingDetails, input.SectorSnapshot, input.ThemeSnapshot)

	return &domain.FundAnalysis{
		AnalysisVersion:     FundAnalysisVersionV1,
		AnalysisType:        analysisType,
		AnalysisBasis:       analysisBasis,
		AsOfTime:            now,
		TotalScore:          scoreDecimal(totalScore),
		Confidence:          confidence,
		RiskLevel:           riskLevel,
		IncreasePercent:     scoreDecimal(increasePercent),
		HoldPercent:         scoreDecimal(holdPercent),
		DecreasePercent:     scoreDecimal(decreasePercent),
		LatestHoldingPeriod: latestPeriod,
		Summary:             buildAnalysisSummary(increasePercent, holdPercent, decreasePercent, riskLevel),
		Reasons:             reasons,
		Warnings:            warnings,
		EventImpacts:        eventImpacts,
		ModuleScores:        moduleScores,
	}
}

func normalizeAnalysisBasis(source string) (string, string) {
	switch strings.TrimSpace(source) {
	case SectorSourceTargetETFFallback:
		return FundAnalysisTypeTrackedETF, "目标ETF口径"
	case SectorSourceQDIIHoldings:
		return FundAnalysisTypeQDIIHoldings, "QDII持仓口径"
	default:
		return FundAnalysisTypeDirectHoldings, "基金持仓口径"
	}
}

func isIndexLikeAnalysis(fund *domain.Fund, analysisType string) bool {
	if analysisType == FundAnalysisTypeTrackedETF {
		return true
	}
	if fund == nil {
		return false
	}
	fundType := strings.ToLower(strings.TrimSpace(fund.Type))
	fundName := strings.ToLower(strings.TrimSpace(fund.Name))
	return strings.Contains(fundType, "index") ||
		strings.Contains(fundType, "指数") ||
		strings.Contains(fundName, "etf") ||
		strings.Contains(fundName, "联接")
}

func isMomentumFriendlyIndexLike(fund *domain.Fund, analysisType string, trendScore, changePercent float64) bool {
	return isIndexLikeAnalysis(fund, analysisType) && trendScore >= 78 && changePercent >= 2.0
}

func resolveCoveragePercent(estimate *domain.FundEstimate, holdings []domain.StockHolding) float64 {
	if estimate != nil && estimate.TotalHoldRatio.GreaterThan(decimal.Zero) {
		return clamp(decimalToFloat(estimate.TotalHoldRatio), 0, 100)
	}

	total := 0.0
	for _, holding := range holdings {
		total += decimalToFloat(holding.HoldingRatio)
	}
	return clamp(total, 0, 100)
}

func decimalToFloat(value decimal.Decimal) float64 {
	parsed, _ := value.Float64()
	return parsed
}

func centeredMarketScore(value, scale float64) float64 {
	if scale <= 0 {
		return 50
	}
	return clamp(50+(clamp(value/scale, -1, 1)*40), 5, 95)
}

func clamp(value, minValue, maxValue float64) float64 {
	switch {
	case value < minValue:
		return minValue
	case value > maxValue:
		return maxValue
	default:
		return value
	}
}

func computeIntradayMomentum(points []domain.TimeSeriesPoint) float64 {
	if len(points) < 2 {
		return 0
	}
	first := decimalToFloat(points[0].ChangePercent)
	last := decimalToFloat(points[len(points)-1].ChangePercent)
	return last - first
}

func computeIntradayRange(points []domain.TimeSeriesPoint) float64 {
	if len(points) == 0 {
		return 0
	}
	maxValue := decimalToFloat(points[0].ChangePercent)
	minValue := maxValue
	for _, point := range points[1:] {
		value := decimalToFloat(point.ChangePercent)
		if value > maxValue {
			maxValue = value
		}
		if value < minValue {
			minValue = value
		}
	}
	return maxValue - minValue
}

func computePositiveBreadth(details []domain.HoldingDetail) float64 {
	if len(details) == 0 {
		return 0.5
	}
	positiveCount := 0.0
	for _, detail := range details {
		if detail.StockChange.GreaterThanOrEqual(decimal.Zero) {
			positiveCount++
		}
	}
	return positiveCount / float64(len(details))
}

func computeAverageAbsoluteMove(details []domain.HoldingDetail) float64 {
	if len(details) == 0 {
		return 0
	}
	total := 0.0
	for _, detail := range details {
		total += math.Abs(decimalToFloat(detail.StockChange))
	}
	return total / float64(len(details))
}

func sectorConfidence(snapshot *domain.FundSectorSnapshot) string {
	if snapshot == nil {
		return "low"
	}
	return strings.TrimSpace(snapshot.Confidence)
}

func themeConfidence(snapshot *domain.FundThemeSnapshot) string {
	if snapshot == nil {
		return "low"
	}
	return strings.TrimSpace(snapshot.Confidence)
}

func sectorPrimaryName(snapshot *domain.FundSectorSnapshot) string {
	if snapshot == nil {
		return ""
	}
	return strings.TrimSpace(snapshot.PrimarySectorName)
}

func themePrimaryName(snapshot *domain.FundThemeSnapshot) string {
	if snapshot == nil {
		return ""
	}
	return strings.TrimSpace(snapshot.PrimaryThemeName)
}

func confidenceScore(confidence string) float64 {
	switch strings.TrimSpace(confidence) {
	case "high":
		return 85
	case "medium":
		return 66
	default:
		return 42
	}
}

func primaryExposureWeight(sectorSnapshot *domain.FundSectorSnapshot, themeSnapshot *domain.FundThemeSnapshot) float64 {
	weight := 0.0
	if sectorSnapshot != nil && len(sectorSnapshot.Breakdown) > 0 {
		weight = math.Max(weight, decimalToFloat(sectorSnapshot.Breakdown[0].WeightPercent))
	}
	if themeSnapshot != nil && len(themeSnapshot.Breakdown) > 0 {
		weight = math.Max(weight, decimalToFloat(themeSnapshot.Breakdown[0].WeightPercent))
	}
	return weight
}

func topBreakdownWeightFromSector(snapshot *domain.FundSectorSnapshot) float64 {
	if snapshot == nil || len(snapshot.Breakdown) == 0 {
		return 0
	}
	return decimalToFloat(snapshot.Breakdown[0].WeightPercent)
}

func topBreakdownWeightFromTheme(snapshot *domain.FundThemeSnapshot) float64 {
	if snapshot == nil || len(snapshot.Breakdown) == 0 {
		return 0
	}
	return decimalToFloat(snapshot.Breakdown[0].WeightPercent)
}

func topHoldingWeight(holdings []domain.StockHolding) float64 {
	weight := 0.0
	for _, holding := range holdings {
		weight = math.Max(weight, decimalToFloat(holding.HoldingRatio))
	}
	return weight
}

func topNWeight(holdings []domain.StockHolding, limit int) float64 {
	if len(holdings) == 0 || limit <= 0 {
		return 0
	}
	weights := make([]float64, 0, len(holdings))
	for _, holding := range holdings {
		weights = append(weights, decimalToFloat(holding.HoldingRatio))
	}
	sort.Slice(weights, func(i, j int) bool {
		return weights[i] > weights[j]
	})
	total := 0.0
	for i := 0; i < minAnalysisInt(len(weights), limit); i++ {
		total += weights[i]
	}
	return total
}

func recognizedClassificationSignal(sectorSnapshot *domain.FundSectorSnapshot, themeSnapshot *domain.FundThemeSnapshot) float64 {
	if hasRecognizedTheme(themeSnapshot) {
		return 78
	}
	if hasRecognizedSector(sectorSnapshot) {
		return 68
	}
	return 42
}

func hasRecognizedSector(snapshot *domain.FundSectorSnapshot) bool {
	return snapshot != nil && snapshot.PrimarySectorCode != "" && snapshot.PrimarySectorCode != "other_equity"
}

func hasRecognizedTheme(snapshot *domain.FundThemeSnapshot) bool {
	return snapshot != nil && snapshot.PrimaryThemeCode != "" && snapshot.PrimaryThemeCode != "other_theme"
}

func latestHoldingPeriod(holdings []domain.StockHolding) (string, time.Time) {
	type reportingCandidate struct {
		label string
		time  time.Time
	}

	candidates := make([]reportingCandidate, 0, len(holdings))
	seen := map[string]struct{}{}
	for _, holding := range holdings {
		label := strings.TrimSpace(holding.ReportingPeriod)
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		parsed, ok := parseReportingPeriod(label)
		if !ok {
			continue
		}
		candidates = append(candidates, reportingCandidate{label: label, time: parsed})
	}
	if len(candidates) == 0 {
		return "", time.Time{}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].time.After(candidates[j].time)
	})
	return candidates[0].label, candidates[0].time
}

func analyzeHoldingChanges(current, previous []domain.StockHolding, previousPeriod string) *holdingChangeInsights {
	if len(current) == 0 || len(previous) == 0 || strings.TrimSpace(previousPeriod) == "" {
		return nil
	}

	currentByCode := make(map[string]domain.StockHolding, len(current))
	previousByCode := make(map[string]domain.StockHolding, len(previous))
	for _, holding := range current {
		currentByCode[strings.TrimSpace(holding.StockCode)] = holding
	}
	for _, holding := range previous {
		previousByCode[strings.TrimSpace(holding.StockCode)] = holding
	}

	changes := &holdingChangeInsights{PreviousPeriod: strings.TrimSpace(previousPeriod)}
	for code, currentHolding := range currentByCode {
		previousHolding, existed := previousByCode[code]
		currentWeight := decimalToFloat(currentHolding.HoldingRatio)
		previousWeight := 0.0
		if existed {
			previousWeight = decimalToFloat(previousHolding.HoldingRatio)
			changes.OverlapCount++
		} else {
			changes.NewEntries++
		}

		delta := currentWeight - previousWeight
		name := strings.TrimSpace(currentHolding.StockName)
		if name == "" && existed {
			name = strings.TrimSpace(previousHolding.StockName)
		}
		candidate := &holdingDelta{
			StockCode:      code,
			StockName:      name,
			CurrentWeight:  currentWeight,
			PreviousWeight: previousWeight,
			Delta:          delta,
		}
		if delta > 0.01 && (changes.TopIncrease == nil || delta > changes.TopIncrease.Delta) {
			changes.TopIncrease = candidate
		}
		if delta < -0.01 && (changes.TopDecrease == nil || delta < changes.TopDecrease.Delta) {
			changes.TopDecrease = candidate
		}
	}

	for code, previousHolding := range previousByCode {
		if _, ok := currentByCode[code]; ok {
			continue
		}
		changes.RemovedEntries++
		delta := -decimalToFloat(previousHolding.HoldingRatio)
		candidate := &holdingDelta{
			StockCode:      code,
			StockName:      strings.TrimSpace(previousHolding.StockName),
			CurrentWeight:  0,
			PreviousWeight: decimalToFloat(previousHolding.HoldingRatio),
			Delta:          delta,
		}
		if changes.TopDecrease == nil || delta < changes.TopDecrease.Delta {
			changes.TopDecrease = candidate
		}
	}

	changes.Top3Delta = topNWeight(current, 3) - topNWeight(previous, 3)
	if changes.OverlapCount == 0 && changes.NewEntries == 0 && changes.RemovedEntries == 0 && changes.TopIncrease == nil && changes.TopDecrease == nil {
		return nil
	}
	return changes
}

func analyzeExposureShift(
	currentSector *domain.FundSectorSnapshot,
	currentTheme *domain.FundThemeSnapshot,
	previousSector *domain.FundSectorSnapshot,
	previousTheme *domain.FundThemeSnapshot,
) *exposureShiftInsight {
	if hasRecognizedTheme(currentTheme) && hasRecognizedTheme(previousTheme) {
		currentWeight := topBreakdownWeightFromTheme(currentTheme)
		previousWeight := topBreakdownWeightFromTheme(previousTheme)
		return &exposureShiftInsight{
			Scope:          "theme",
			PreviousCode:   strings.TrimSpace(previousTheme.PrimaryThemeCode),
			PreviousName:   strings.TrimSpace(previousTheme.PrimaryThemeName),
			PreviousWeight: previousWeight,
			CurrentCode:    strings.TrimSpace(currentTheme.PrimaryThemeCode),
			CurrentName:    strings.TrimSpace(currentTheme.PrimaryThemeName),
			CurrentWeight:  currentWeight,
			Changed:        strings.TrimSpace(previousTheme.PrimaryThemeCode) != strings.TrimSpace(currentTheme.PrimaryThemeCode),
			WeightDelta:    currentWeight - previousWeight,
		}
	}
	if hasRecognizedSector(currentSector) && hasRecognizedSector(previousSector) {
		currentWeight := topBreakdownWeightFromSector(currentSector)
		previousWeight := topBreakdownWeightFromSector(previousSector)
		return &exposureShiftInsight{
			Scope:          "sector",
			PreviousCode:   strings.TrimSpace(previousSector.PrimarySectorCode),
			PreviousName:   strings.TrimSpace(previousSector.PrimarySectorName),
			PreviousWeight: previousWeight,
			CurrentCode:    strings.TrimSpace(currentSector.PrimarySectorCode),
			CurrentName:    strings.TrimSpace(currentSector.PrimarySectorName),
			CurrentWeight:  currentWeight,
			Changed:        strings.TrimSpace(previousSector.PrimarySectorCode) != strings.TrimSpace(currentSector.PrimarySectorCode),
			WeightDelta:    currentWeight - previousWeight,
		}
	}
	return nil
}

func analyzeCurrentExposureEvent(
	currentHoldingEvents []domain.FundAnalysisEventImpact,
	sectorSnapshot *domain.FundSectorSnapshot,
	themeSnapshot *domain.FundThemeSnapshot,
) *currentExposureEventInsight {
	filtered := make([]domain.FundAnalysisEventImpact, 0, len(currentHoldingEvents))
	positiveCount := 0
	negativeCount := 0
	weightedSignals := 0.0
	type weightedEvent struct {
		title  string
		weight float64
	}
	highlights := make([]weightedEvent, 0, len(currentHoldingEvents))
	for _, event := range currentHoldingEvents {
		if strings.TrimSpace(event.TargetScope) != "holding" {
			continue
		}
		horizon := strings.TrimSpace(event.Horizon)
		if horizon != "current" && horizon != "intraday" {
			continue
		}
		filtered = append(filtered, event)
		weight := eventSignalWeight(event)
		weightedSignals += weight
		if title := strings.TrimSpace(event.Title); title != "" {
			highlights = append(highlights, weightedEvent{title: title, weight: weight})
		}
		switch strings.TrimSpace(event.Impact) {
		case "positive":
			positiveCount++
		case "negative":
			negativeCount++
		}
	}
	if len(filtered) < 2 && weightedSignals < 2.4 {
		return nil
	}
	if positiveCount > 0 && negativeCount > 0 && math.Abs(float64(positiveCount-negativeCount)) < 1 && weightedSignals < 4.2 {
		return nil
	}

	name := strings.TrimSpace(themePrimaryName(themeSnapshot))
	scope := "theme"
	if name == "" {
		name = strings.TrimSpace(sectorPrimaryName(sectorSnapshot))
		scope = "sector"
	}
	if name == "" {
		return nil
	}

	impact := "neutral"
	switch {
	case negativeCount > positiveCount:
		impact = "negative"
	case positiveCount > negativeCount:
		impact = "positive"
	}
	strength := "medium"
	if weightedSignals >= 4 || len(filtered) >= 3 {
		strength = "high"
	} else if weightedSignals < 2.8 {
		strength = "low"
	}
	sort.SliceStable(highlights, func(i, j int) bool {
		return highlights[i].weight > highlights[j].weight
	})
	highlightTitles := make([]string, 0, minAnalysisInt(len(highlights), 2))
	seenTitles := make(map[string]struct{}, minAnalysisInt(len(highlights), 2))
	for _, item := range highlights {
		if _, ok := seenTitles[item.title]; ok {
			continue
		}
		seenTitles[item.title] = struct{}{}
		highlightTitles = append(highlightTitles, item.title)
		if len(highlightTitles) >= 2 {
			break
		}
	}

	summary := "围绕 " + name + " 主线，近阶段重仓股事件逐步增多，适合结合事件与持仓权重一起观察。"
	if impact == "positive" {
		summary = "围绕 " + name + " 主线，近阶段重仓股事件偏正向，当前赛道情绪有一定支撑。"
	}
	if impact == "negative" {
		summary = "围绕 " + name + " 主线，近阶段重仓股事件偏负向，需警惕主线情绪转弱。"
	}
	if len(highlightTitles) > 0 {
		summary += " 当前较值得关注的线索包括：" + strings.Join(highlightTitles, "、") + "。"
	}

	return &currentExposureEventInsight{
		Name:     name,
		Scope:    scope,
		Count:    len(filtered),
		Impact:   impact,
		Strength: strength,
		Summary:  summary,
		Highlights: highlightTitles,
	}
}

func eventSignalWeight(event domain.FundAnalysisEventImpact) float64 {
	weight := 1.0
	switch strings.TrimSpace(event.Strength) {
	case "high":
		weight += 0.8
	case "medium":
		weight += 0.4
	}
	if strings.TrimSpace(event.Horizon) == "current" {
		weight += 0.4
	}
	if event.WeightHint != nil {
		weight += clamp(decimalToFloat(*event.WeightHint)/25, 0, 0.8)
	}
	return weight
}

func analyzeExposureBreakdownChanges(
	currentSector *domain.FundSectorSnapshot,
	currentTheme *domain.FundThemeSnapshot,
	previousSector *domain.FundSectorSnapshot,
	previousTheme *domain.FundThemeSnapshot,
) *exposureBreakdownInsights {
	insights := &exposureBreakdownInsights{}
	sectorDeltas := compareSectorBreakdowns(currentSector, previousSector)
	themeDeltas := compareThemeBreakdowns(currentTheme, previousTheme)

	for i := range sectorDeltas {
		item := sectorDeltas[i]
		if item.Delta >= 3 && (insights.SectorIncrease == nil || item.Delta > insights.SectorIncrease.Delta) {
			copyItem := item
			insights.SectorIncrease = &copyItem
		}
		if item.Delta <= -3 && (insights.SectorDecrease == nil || item.Delta < insights.SectorDecrease.Delta) {
			copyItem := item
			insights.SectorDecrease = &copyItem
		}
	}
	for i := range themeDeltas {
		item := themeDeltas[i]
		if item.Delta >= 3 && (insights.ThemeIncrease == nil || item.Delta > insights.ThemeIncrease.Delta) {
			copyItem := item
			insights.ThemeIncrease = &copyItem
		}
		if item.Delta <= -3 && (insights.ThemeDecrease == nil || item.Delta < insights.ThemeDecrease.Delta) {
			copyItem := item
			insights.ThemeDecrease = &copyItem
		}
	}

	if insights.ThemeIncrease != nil && insights.ThemeDecrease != nil {
		insights.StyleDrift = true
	}
	if insights.SectorIncrease != nil && insights.SectorDecrease != nil {
		insights.StyleDrift = true
	}
	if insights.SectorIncrease == nil && insights.SectorDecrease == nil && insights.ThemeIncrease == nil && insights.ThemeDecrease == nil && !insights.StyleDrift {
		return nil
	}
	return insights
}

func compareSectorBreakdowns(current, previous *domain.FundSectorSnapshot) []exposureBreakdownDelta {
	if current == nil || previous == nil {
		return nil
	}
	currentMap := make(map[string]domain.FundSectorBreakdown, len(current.Breakdown))
	previousMap := make(map[string]domain.FundSectorBreakdown, len(previous.Breakdown))
	for _, item := range current.Breakdown {
		currentMap[strings.TrimSpace(item.SectorCode)] = item
	}
	for _, item := range previous.Breakdown {
		previousMap[strings.TrimSpace(item.SectorCode)] = item
	}
	keys := make(map[string]struct{}, len(currentMap)+len(previousMap))
	for key := range currentMap {
		keys[key] = struct{}{}
	}
	for key := range previousMap {
		keys[key] = struct{}{}
	}
	result := make([]exposureBreakdownDelta, 0, len(keys))
	for key := range keys {
		currentItem := currentMap[key]
		previousItem := previousMap[key]
		name := strings.TrimSpace(currentItem.SectorName)
		if name == "" {
			name = strings.TrimSpace(previousItem.SectorName)
		}
		result = append(result, exposureBreakdownDelta{
			Code:           key,
			Name:           name,
			CurrentWeight:  decimalToFloat(currentItem.WeightPercent),
			PreviousWeight: decimalToFloat(previousItem.WeightPercent),
			Delta:          decimalToFloat(currentItem.WeightPercent) - decimalToFloat(previousItem.WeightPercent),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return math.Abs(result[i].Delta) > math.Abs(result[j].Delta)
	})
	return result
}

func compareThemeBreakdowns(current, previous *domain.FundThemeSnapshot) []exposureBreakdownDelta {
	if current == nil || previous == nil {
		return nil
	}
	currentMap := make(map[string]domain.FundThemeBreakdown, len(current.Breakdown))
	previousMap := make(map[string]domain.FundThemeBreakdown, len(previous.Breakdown))
	for _, item := range current.Breakdown {
		currentMap[strings.TrimSpace(item.ThemeCode)] = item
	}
	for _, item := range previous.Breakdown {
		previousMap[strings.TrimSpace(item.ThemeCode)] = item
	}
	keys := make(map[string]struct{}, len(currentMap)+len(previousMap))
	for key := range currentMap {
		keys[key] = struct{}{}
	}
	for key := range previousMap {
		keys[key] = struct{}{}
	}
	result := make([]exposureBreakdownDelta, 0, len(keys))
	for key := range keys {
		currentItem := currentMap[key]
		previousItem := previousMap[key]
		name := strings.TrimSpace(currentItem.ThemeName)
		if name == "" {
			name = strings.TrimSpace(previousItem.ThemeName)
		}
		result = append(result, exposureBreakdownDelta{
			Code:           key,
			Name:           name,
			CurrentWeight:  decimalToFloat(currentItem.WeightPercent),
			PreviousWeight: decimalToFloat(previousItem.WeightPercent),
			Delta:          decimalToFloat(currentItem.WeightPercent) - decimalToFloat(previousItem.WeightPercent),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return math.Abs(result[i].Delta) > math.Abs(result[j].Delta)
	})
	return result
}

func parseReportingPeriod(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if parsed, err := time.ParseInLocation("2006-01-02", raw, time.Local); err == nil {
		return parsed, true
	}

	cleaned := strings.ToUpper(strings.ReplaceAll(raw, "-", ""))
	if len(cleaned) == 6 && strings.Contains(cleaned, "Q") {
		yearPart := cleaned[:4]
		quarterPart := cleaned[5:]
		year, err := strconv.Atoi(yearPart)
		if err != nil {
			return time.Time{}, false
		}
		switch quarterPart {
		case "1":
			return time.Date(year, time.March, 31, 0, 0, 0, 0, time.Local), true
		case "2":
			return time.Date(year, time.June, 30, 0, 0, 0, 0, time.Local), true
		case "3":
			return time.Date(year, time.September, 30, 0, 0, 0, 0, time.Local), true
		case "4":
			return time.Date(year, time.December, 31, 0, 0, 0, 0, time.Local), true
		}
	}

	return time.Time{}, false
}

func eventFreshnessScore(now, latest time.Time) float64 {
	if latest.IsZero() {
		return 38
	}
	days := holdingDisclosureAgeDays(now, latest)
	switch {
	case days <= 120:
		return 84
	case days <= 240:
		return 72
	case days <= 370:
		return 60
	default:
		return 44
	}
}

func holdingDisclosureAgeDays(now, latest time.Time) float64 {
	if latest.IsZero() {
		return 9999
	}
	return now.Sub(latest).Hours() / 24
}

func lowCoveragePenalty(coveragePercent float64) float64 {
	switch {
	case coveragePercent >= 65:
		return 0
	case coveragePercent >= 45:
		return (65 - coveragePercent) * 0.55
	case coveragePercent >= 25:
		return 12 + (45-coveragePercent)*0.45
	default:
		return 22 + (25-coveragePercent)*0.3
	}
}

func staleDisclosurePenalty(disclosureAgeDays float64) float64 {
	switch {
	case disclosureAgeDays <= 120:
		return 0
	case disclosureAgeDays <= 240:
		return 6
	case disclosureAgeDays <= 370:
		return 12
	case disclosureAgeDays < 9999:
		return 18
	default:
		return 22
	}
}

func lowClassificationPenalty(sectorSnapshot *domain.FundSectorSnapshot, themeSnapshot *domain.FundThemeSnapshot) float64 {
	switch {
	case hasRecognizedSector(sectorSnapshot) && hasRecognizedTheme(themeSnapshot):
		return 0
	case hasRecognizedSector(sectorSnapshot) || hasRecognizedTheme(themeSnapshot):
		return 6
	default:
		return 12
	}
}

func deriveAnalysisConfidence(coveragePercent float64, sectorConfidence string, themeConfidence string, analysisType string) string {
	baseScore := 0.50*coveragePercent + 0.25*confidenceScore(sectorConfidence) + 0.25*confidenceScore(themeConfidence)
	if analysisType == FundAnalysisTypeTrackedETF && coveragePercent >= 60 {
		baseScore += 6
	}
	switch {
	case baseScore >= 74:
		return "high"
	case baseScore >= 56:
		return "medium"
	default:
		return "low"
	}
}

func deriveRiskLevel(riskScore float64) string {
	switch {
	case riskScore >= 70:
		return "low"
	case riskScore >= 48:
		return "medium"
	default:
		return "high"
	}
}

func recommendationDistribution(decisionBias float64) (float64, float64, float64) {
	increaseLogit := decisionBias / 12
	holdLogit := 1 - math.Abs(decisionBias)/24
	decreaseLogit := -decisionBias / 12

	maxLogit := math.Max(increaseLogit, math.Max(holdLogit, decreaseLogit))
	increaseExp := math.Exp(increaseLogit - maxLogit)
	holdExp := math.Exp(holdLogit - maxLogit)
	decreaseExp := math.Exp(decreaseLogit - maxLogit)

	total := increaseExp + holdExp + decreaseExp
	if total == 0 {
		return 33.33, 33.34, 33.33
	}

	increase := math.Round((increaseExp/total)*1000) / 10
	hold := math.Round((holdExp/total)*1000) / 10
	decrease := math.Round((decreaseExp/total)*1000) / 10
	sum := increase + hold + decrease
	if diff := math.Round((100-sum)*10) / 10; diff != 0 {
		hold += diff
	}
	return increase, hold, decrease
}

func scoreDecimal(value float64) decimal.Decimal {
	return decimal.NewFromFloat(value).Round(1)
}

func buildTrendSummary(changePercent, momentum float64) string {
	switch {
	case changePercent >= 1.2 || momentum >= 0.8:
		return "实时走势偏强，分时尾段仍有上行动量"
	case changePercent <= -1.2 || momentum <= -0.8:
		return "实时走势偏弱，分时修复动能仍不足"
	default:
		return "实时走势中性，分时动量暂未形成明显单边"
	}
}

func buildStructureSummary(coveragePercent float64, sectorSnapshot *domain.FundSectorSnapshot, themeSnapshot *domain.FundThemeSnapshot) string {
	primaryParts := make([]string, 0, 2)
	if hasRecognizedSector(sectorSnapshot) && sectorSnapshot.PrimarySectorName != "" {
		primaryParts = append(primaryParts, sectorSnapshot.PrimarySectorName)
	}
	if hasRecognizedTheme(themeSnapshot) && themeSnapshot.PrimaryThemeName != "" {
		primaryParts = append(primaryParts, themeSnapshot.PrimaryThemeName)
	}
	if len(primaryParts) == 0 {
		return "持仓结构可解释性一般，当前更多依赖基础覆盖率"
	}
	return "持仓覆盖率 " + decimal.NewFromFloat(coveragePercent).Round(1).StringFixed(1) + "%，主暴露集中在 " + strings.Join(primaryParts[:minAnalysisInt(len(primaryParts), 2)], " / ")
}

func buildHeatSummary(avgAbsMove float64, positiveBreadth float64, themeSnapshot *domain.FundThemeSnapshot) string {
	breadthText := decimal.NewFromFloat(positiveBreadth*100).Round(0).StringFixed(0) + "%"
	if hasRecognizedTheme(themeSnapshot) {
		return "盘中活跃度 " + decimal.NewFromFloat(avgAbsMove).Round(1).StringFixed(1) + "，正贡献广度约 " + breadthText + "，热点主题可识别"
	}
	return "盘中活跃度 " + decimal.NewFromFloat(avgAbsMove).Round(1).StringFixed(1) + "，正贡献广度约 " + breadthText
}

func buildRiskSummary(intradayRange float64, concentrationWeight float64, riskLevel string) string {
	return "日内振幅 " + decimal.NewFromFloat(intradayRange).Round(1).StringFixed(1) + "%，主暴露集中度 " + decimal.NewFromFloat(concentrationWeight).Round(1).StringFixed(1) + "%，当前判定为" + mapRiskLevelLabel(riskLevel)
}

func buildAllocationSummary(changePercent, allocationScore float64) string {
	switch {
	case allocationScore >= 65 && changePercent <= 1:
		return "当前位置更适合分批观察或中等力度加仓"
	case allocationScore <= 42 && changePercent >= 1.8:
		return "短线抬升较快，当前位置不宜追高"
	default:
		return "当前位置偏中性，更适合结合仓位控制节奏"
	}
}

func holdingChangeEventAdjustment(changes *holdingChangeInsights) float64 {
	return 0
}

func exposureShiftEventAdjustment(shift *exposureShiftInsight) float64 {
	return 0
}

func mergeCurrentEventImpacts(eventGroups ...[]domain.FundAnalysisEventImpact) []domain.FundAnalysisEventImpact {
	total := 0
	for _, group := range eventGroups {
		total += len(group)
	}
	if total == 0 {
		return nil
	}
	merged := make([]domain.FundAnalysisEventImpact, 0, total)
	for _, group := range eventGroups {
		merged = append(merged, group...)
	}
	return merged
}

func currentHoldingEventAdjustment(events []domain.FundAnalysisEventImpact) float64 {
	adjustment := 0.0
	for _, event := range events {
		scope := strings.TrimSpace(event.TargetScope)
		if scope != "holding" && scope != "fund" && scope != "macro" && scope != "index" {
			continue
		}
		delta := 0.0
		switch strings.TrimSpace(event.Impact) {
		case "positive":
			delta = 1.4
		case "negative":
			delta = -1.8
		default:
			delta = 0
		}
		if scope == "fund" {
			delta *= 0.7
		}
		if scope == "macro" {
			delta *= 0.9
		}
		if scope == "index" {
			delta *= 0.8
		}
		switch strings.TrimSpace(event.Strength) {
		case "high":
			delta *= 1.4
		case "medium":
			delta *= 1.0
		default:
			delta *= 0.6
		}
		adjustment += delta
	}
	return clamp(adjustment, -4, 4)
}

func currentExposureEventAdjustment(insight *currentExposureEventInsight) float64 {
	if insight == nil {
		return 0
	}
	switch insight.Impact {
	case "positive":
		return 1.2
	case "negative":
		return -1.5
	default:
		return 0
	}
}

func exposureBreakdownEventAdjustment(insights *exposureBreakdownInsights) float64 {
	if insights == nil {
		return 0
	}
	adjustment := 0.0
	if insights.StyleDrift {
		adjustment -= 0.8
	}
	if insights.ThemeIncrease != nil && insights.ThemeIncrease.Delta >= 5 {
		adjustment -= 0.5
	}
	if insights.ThemeDecrease != nil && math.Abs(insights.ThemeDecrease.Delta) >= 5 {
		adjustment += 0.5
	}
	return clamp(adjustment, -2, 2)
}

func buildEventSummary(latestPeriod, analysisBasis string, currentEvents []domain.FundAnalysisEventImpact, currentExposureEvent *currentExposureEventInsight, exposureBreakdowns *exposureBreakdownInsights, changes *holdingChangeInsights, shift *exposureShiftInsight) string {
	if latestPeriod == "" {
		return "基础事件层仍缺少有效披露，只能提供口径级参考"
	}
	if exposureBreakdowns != nil && exposureBreakdowns.StyleDrift {
		return "当前已纳入重仓股与主线近期事件，并额外识别行业/主题权重变化与风格漂移。"
	}
	if currentExposureEvent != nil {
		return "当前已纳入重仓股与主线近期事件，并以最新披露持仓作为解释底座。"
	}
	if len(currentEvents) > 0 {
		return "当前已纳入重仓股近期公告/业绩/经营事件，并以最新披露持仓作为解释底座。"
	}
	if changes != nil && changes.PreviousPeriod != "" {
		return "当前事件层以最新披露持仓、盘中主驱动和当前分析口径为主；相较 " + changes.PreviousPeriod + " 的上一季变化仅作辅助参考。"
	}
	return "当前仅纳入披露新鲜度与口径可解释性，最新持仓报告期为 " + latestPeriod + "（" + analysisBasis + "）"
}

func buildAnalysisReasons(trendScore, structureScore, heatScore, allocationScore, eventScore float64, sectorSnapshot *domain.FundSectorSnapshot, themeSnapshot *domain.FundThemeSnapshot, analysisBasis string, latestPeriod string, currentEvents []domain.FundAnalysisEventImpact, currentExposureEvent *currentExposureEventInsight, exposureBreakdowns *exposureBreakdownInsights, changes *holdingChangeInsights, shift *exposureShiftInsight) []string {
	reasons := make([]string, 0, 4)
	if trendScore >= 60 {
		reasons = append(reasons, "当前实时走势与分时动量偏正向，趋势模块给出较高评分。")
	}
	if structureScore >= 60 {
		if hasRecognizedTheme(themeSnapshot) && themeSnapshot.PrimaryThemeName != "" {
			reasons = append(reasons, "持仓结构主线较清晰，当前主题暴露集中在 "+themeSnapshot.PrimaryThemeName+"。")
		} else if hasRecognizedSector(sectorSnapshot) && sectorSnapshot.PrimarySectorName != "" {
			reasons = append(reasons, "持仓结构主线较清晰，当前行业暴露集中在 "+sectorSnapshot.PrimarySectorName+"。")
		}
	}
	for _, event := range currentEvents {
		if strings.TrimSpace(event.Impact) != "positive" {
			continue
		}
		scope := strings.TrimSpace(event.TargetScope)
		if scope != "holding" && scope != "fund" && scope != "macro" {
			continue
		}
		if scope == "fund" {
			reasons = append(reasons, "基金自身近期事件值得关注："+event.Title+"。")
		} else if scope == "macro" {
			reasons = append(reasons, "宏观/政策层面值得关注："+event.Title+"。")
		} else {
			reasons = append(reasons, "当前重仓股事件偏正向："+event.Title+"。")
		}
		break
	}
	if currentExposureEvent != nil && currentExposureEvent.Impact == "positive" {
		reasons = append(reasons, currentExposureEvent.Summary)
	}
	if changes != nil && changes.TopIncrease != nil && changes.TopIncrease.Delta >= 2 {
		reasons = append(reasons, buildHoldingDeltaSummary(changes.TopIncrease, changes.PreviousPeriod, "increase"))
	}
	if exposureBreakdowns != nil && exposureBreakdowns.ThemeDecrease != nil && math.Abs(exposureBreakdowns.ThemeDecrease.Delta) >= 5 {
		reasons = append(reasons, "相较上一季，主题暴露更分散："+exposureBreakdowns.ThemeDecrease.Name+" 权重回落 "+deltaPercentValue(math.Abs(exposureBreakdowns.ThemeDecrease.Delta))+"。")
	}
	if heatScore >= 60 {
		reasons = append(reasons, "盘中活跃度与正贡献广度尚可，热度模块对短线关注度给出正向判断。")
	}
	if allocationScore >= 60 {
		reasons = append(reasons, "当前位置并未明显透支性价比，分批观察或加仓的容错更高。")
	}
	if eventScore >= 60 && latestPeriod != "" {
		reasons = append(reasons, "最新持仓披露相对较新，当前"+analysisBasis+"的解释稳定性更好。")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "当前各模块没有形成明确单边优势，基础版建议更偏向中性观察。")
	}
	return reasons[:minAnalysisInt(len(reasons), 3)]
}

func buildAnalysisWarnings(riskScore float64, confidence string, eventScore float64, latestPeriod string, disclosureAgeDays float64, changePercent float64, concentrationWeight float64, currentEvents []domain.FundAnalysisEventImpact, currentExposureEvent *currentExposureEventInsight, exposureBreakdowns *exposureBreakdownInsights, changes *holdingChangeInsights, shift *exposureShiftInsight) []string {
	warnings := make([]string, 0, 4)
	if riskScore < 50 {
		warnings = append(warnings, "波动与集中度偏高，当前不适合激进提高仓位。")
	}
	if confidence == "low" {
		warnings = append(warnings, "持仓识别覆盖有限，当前量化结果更适合作为参考而不是绝对结论。")
	}
	for _, event := range currentEvents {
		if strings.TrimSpace(event.Impact) != "negative" {
			continue
		}
		scope := strings.TrimSpace(event.TargetScope)
		if scope != "holding" && scope != "fund" && scope != "macro" {
			continue
		}
		if scope == "fund" {
			warnings = append(warnings, "基金自身近期事件偏负向："+event.Title+"。")
		} else if scope == "macro" {
			warnings = append(warnings, "宏观/政策层面需关注："+event.Title+"。")
		} else {
			warnings = append(warnings, "当前重仓股事件偏负向："+event.Title+"。")
		}
		break
	}
	if currentExposureEvent != nil && currentExposureEvent.Impact == "negative" {
		warnings = append(warnings, currentExposureEvent.Summary)
	}
	if changes != nil && changes.TopDecrease != nil && math.Abs(changes.TopDecrease.Delta) >= 2 {
		warnings = append(warnings, buildHoldingDeltaSummary(changes.TopDecrease, changes.PreviousPeriod, "decrease"))
	}
	if changes != nil && changes.NewEntries+changes.RemovedEntries >= 4 {
		warnings = append(warnings, fmt.Sprintf("相较 %s，前十大新增 %d 只、退出 %d 只，当前组合主线切换幅度偏大。", changes.PreviousPeriod, changes.NewEntries, changes.RemovedEntries))
	}
	if exposureBreakdowns != nil && exposureBreakdowns.StyleDrift {
		warnings = append(warnings, "相较上一季，行业/主题权重出现双向变化，当前组合存在风格漂移迹象。")
	}
	if eventScore < 55 || latestPeriod == "" {
		warnings = append(warnings, "事件模块目前只覆盖披露新鲜度与口径可解释性，政策/财报/新闻事件仍待后续接入。")
	}
	if changePercent >= 2.5 {
		warnings = append(warnings, "短线涨幅已经较快，当前位置更需要避免追高。")
	}
	if concentrationWeight >= 45 {
		warnings = append(warnings, "主暴露过于集中，单一赛道波动会显著放大净值弹性。")
	}
	if latestPeriod != "" && disclosureAgeDays > 240 {
		warnings = append(warnings, "当前持仓披露已经偏旧，部分结构判断可能落后于基金经理最新调仓。")
	}
	if len(warnings) == 0 {
		warnings = append(warnings, "当前没有触发明显的高风险警报，但仍建议结合自身仓位做分批决策。")
	}
	return warnings[:minAnalysisInt(len(warnings), 3)]
}

func buildEventImpacts(
	analysisBasis, latestPeriod string,
	disclosureAgeDays float64,
	confidence string,
	concentrationWeight float64,
	currentEvents []domain.FundAnalysisEventImpact,
	currentExposureEvent *currentExposureEventInsight,
	exposureBreakdowns *exposureBreakdownInsights,
	changes *holdingChangeInsights,
	shift *exposureShiftInsight,
	holdingDetails []domain.HoldingDetail,
	sectorSnapshot *domain.FundSectorSnapshot,
	themeSnapshot *domain.FundThemeSnapshot,
) []domain.FundAnalysisEventImpact {
	events := make([]domain.FundAnalysisEventImpact, 0, 10)

	if currentExposureEvent != nil {
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "current_exposure_event_cluster",
			Title:       currentExposureEvent.Name + "主线近期事件密集",
			Impact:      currentExposureEvent.Impact,
			Summary:     currentExposureEvent.Summary,
			TargetScope: "exposure",
			Strength:    currentExposureEvent.Strength,
			Horizon:     "current",
		})
	}
	for _, event := range currentEvents {
		events = append(events, event)
	}
	if exposureBreakdowns != nil && exposureBreakdowns.ThemeIncrease != nil && exposureBreakdowns.ThemeIncrease.Delta >= 4 {
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "theme_weight_increase",
			Title:       exposureBreakdowns.ThemeIncrease.Name + "主题权重提升",
			Impact:      "negative",
			Summary:     "相较上一季，该主题权重提升 " + deltaPercentValue(exposureBreakdowns.ThemeIncrease.Delta) + "，当前主题暴露更集中。",
			TargetScope: "exposure",
			Strength:    driftStrength(exposureBreakdowns.ThemeIncrease.Delta),
			Horizon:     "quarterly",
			WeightHint:  decimalPointerFromFloat(exposureBreakdowns.ThemeIncrease.CurrentWeight),
		})
	}
	if exposureBreakdowns != nil && exposureBreakdowns.ThemeDecrease != nil && math.Abs(exposureBreakdowns.ThemeDecrease.Delta) >= 4 {
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "theme_weight_decrease",
			Title:       exposureBreakdowns.ThemeDecrease.Name + "主题权重回落",
			Impact:      "positive",
			Summary:     "相较上一季，该主题权重回落 " + deltaPercentValue(math.Abs(exposureBreakdowns.ThemeDecrease.Delta)) + "，组合主题分布更均衡。",
			TargetScope: "exposure",
			Strength:    driftStrength(math.Abs(exposureBreakdowns.ThemeDecrease.Delta)),
			Horizon:     "quarterly",
			WeightHint:  decimalPointerFromFloat(exposureBreakdowns.ThemeDecrease.PreviousWeight),
		})
	}
	if exposureBreakdowns != nil && exposureBreakdowns.SectorIncrease != nil && exposureBreakdowns.SectorIncrease.Delta >= 4 {
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "sector_weight_increase",
			Title:       exposureBreakdowns.SectorIncrease.Name + "行业权重提升",
			Impact:      "negative",
			Summary:     "相较上一季，该行业权重提升 " + deltaPercentValue(exposureBreakdowns.SectorIncrease.Delta) + "，行业暴露进一步集中。",
			TargetScope: "exposure",
			Strength:    driftStrength(exposureBreakdowns.SectorIncrease.Delta),
			Horizon:     "quarterly",
			WeightHint:  decimalPointerFromFloat(exposureBreakdowns.SectorIncrease.CurrentWeight),
		})
	}
	if exposureBreakdowns != nil && exposureBreakdowns.SectorDecrease != nil && math.Abs(exposureBreakdowns.SectorDecrease.Delta) >= 4 {
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "sector_weight_decrease",
			Title:       exposureBreakdowns.SectorDecrease.Name + "行业权重回落",
			Impact:      "positive",
			Summary:     "相较上一季，该行业权重回落 " + deltaPercentValue(math.Abs(exposureBreakdowns.SectorDecrease.Delta)) + "，行业分布比之前更分散。",
			TargetScope: "exposure",
			Strength:    driftStrength(math.Abs(exposureBreakdowns.SectorDecrease.Delta)),
			Horizon:     "quarterly",
			WeightHint:  decimalPointerFromFloat(exposureBreakdowns.SectorDecrease.PreviousWeight),
		})
	}

	switch {
	case latestPeriod == "":
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "holding_disclosure_missing",
			Title:       "持仓披露缺失",
			Impact:      "negative",
			Summary:     "当前缺少可靠的最新持仓报告期，事件层只能提供较弱的口径解释。",
			TargetScope: "disclosure",
			Strength:    "high",
			Horizon:     "quarterly",
		})
	case disclosureAgeDays <= 120:
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "holding_disclosure_fresh",
			Title:       "持仓披露较新",
			Impact:      "positive",
			Summary:     "最新持仓报告期为 " + latestPeriod + "，当前结构与主题判断相对更可信。",
			TargetScope: "disclosure",
			Strength:    "medium",
			Horizon:     "quarterly",
		})
	case disclosureAgeDays <= 240:
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "holding_disclosure_moderate",
			Title:       "持仓披露有一定滞后",
			Impact:      "neutral",
			Summary:     "最新持仓报告期为 " + latestPeriod + "，结构判断可用，但需警惕近一季潜在调仓。",
			TargetScope: "disclosure",
			Strength:    "medium",
			Horizon:     "quarterly",
		})
	default:
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "holding_disclosure_stale",
			Title:       "持仓披露偏旧",
			Impact:      "negative",
			Summary:     "最新持仓报告期为 " + latestPeriod + "，当前结构与主题判断可能落后于最新仓位。",
			TargetScope: "disclosure",
			Strength:    "high",
			Horizon:     "quarterly",
		})
	}

	if changes != nil && math.Abs(changes.Top3Delta) >= 3 {
		impact := "neutral"
		title := "前三大集中度变化不大"
		summary := ""
		if changes.Top3Delta > 0 {
			impact = "negative"
			title = "前三大集中度继续抬升"
			summary = fmt.Sprintf("相较 %s，当前前三大重仓合计权重提升 %s，组合主线更集中。", changes.PreviousPeriod, deltaPercentValue(changes.Top3Delta))
		} else {
			impact = "positive"
			title = "前三大集中度有所回落"
			summary = fmt.Sprintf("相较 %s，当前前三大重仓合计权重回落 %s，组合结构比上一季更分散。", changes.PreviousPeriod, deltaPercentValue(math.Abs(changes.Top3Delta)))
		}
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "top3_concentration_shift",
			Title:       title,
			Impact:      impact,
			Summary:     summary,
			TargetScope: "exposure",
			Strength:    driftStrength(math.Abs(changes.Top3Delta)),
			Horizon:     "quarterly",
			WeightHint:  decimalPointerFromFloat(math.Abs(changes.Top3Delta)),
		})
	}

	if changes != nil && changes.NewEntries+changes.RemovedEntries >= 2 {
		churnCount := changes.NewEntries + changes.RemovedEntries
		impact := "neutral"
		if churnCount >= 4 {
			impact = "negative"
		}
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "top10_churn",
			Title:       "前十大换手出现变化",
			Impact:      impact,
			Summary:     fmt.Sprintf("相较 %s，前十大新增 %d 只、退出 %d 只，组合主线可能正在切换。", changes.PreviousPeriod, changes.NewEntries, changes.RemovedEntries),
			TargetScope: "holding",
			Strength:    driftStrength(float64(churnCount)),
			Horizon:     "quarterly",
		})
	}

	if changes != nil && changes.TopIncrease != nil && changes.TopIncrease.Delta >= 1.2 {
		events = append(events, domain.FundAnalysisEventImpact{
			Code:           "top_holding_increase",
			Title:          displayHoldingName(changes.TopIncrease) + "季度内权重提升",
			Impact:         "neutral",
			Summary:        buildHoldingDeltaSummary(changes.TopIncrease, changes.PreviousPeriod, "increase"),
			TargetScope:    "holding",
			Strength:       driftStrength(math.Abs(changes.TopIncrease.Delta)),
			Horizon:        "quarterly",
			RelatedSymbols: compactSymbols(changes.TopIncrease.StockCode),
			WeightHint:     decimalPointerFromFloat(changes.TopIncrease.CurrentWeight),
		})
	}

	if changes != nil && changes.TopDecrease != nil && math.Abs(changes.TopDecrease.Delta) >= 1.2 {
		events = append(events, domain.FundAnalysisEventImpact{
			Code:           "top_holding_decrease",
			Title:          displayHoldingName(changes.TopDecrease) + "季度内权重回落",
			Impact:         "neutral",
			Summary:        buildHoldingDeltaSummary(changes.TopDecrease, changes.PreviousPeriod, "decrease"),
			TargetScope:    "holding",
			Strength:       driftStrength(math.Abs(changes.TopDecrease.Delta)),
			Horizon:        "quarterly",
			RelatedSymbols: compactSymbols(changes.TopDecrease.StockCode),
			WeightHint:     decimalPointerFromFloat(changes.TopDecrease.PreviousWeight),
		})
	}

	if shift != nil && (shift.Changed || math.Abs(shift.WeightDelta) >= 4) {
		impact := "neutral"
		if shift.Changed {
			impact = "negative"
		} else if shift.WeightDelta <= -4 {
			impact = "positive"
		} else if shift.WeightDelta >= 4 {
			impact = "negative"
		}
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "primary_exposure_shift",
			Title:       "主" + mapExposureScopeLabel(shift.Scope) + "发生变化",
			Impact:      impact,
			Summary:     buildExposureShiftSummary(shift),
			TargetScope: "exposure",
			Strength:    driftStrength(math.Max(math.Abs(shift.WeightDelta), 3)),
			Horizon:     "quarterly",
			WeightHint:  decimalPointerFromFloat(math.Abs(shift.WeightDelta)),
		})
	}

	if topPositive := strongestContributor(holdingDetails, true); topPositive != nil && math.Abs(decimalToFloat(topPositive.Contribution)) >= 0.08 {
		events = append(events, domain.FundAnalysisEventImpact{
			Code:           "top_positive_driver",
			Title:          strings.TrimSpace(topPositive.StockName) + "盘中拉动明显",
			Impact:         "positive",
			Summary:        buildHoldingDriverSummary(*topPositive, "positive"),
			TargetScope:    "holding",
			Strength:       contributionStrength(decimalToFloat(topPositive.Contribution)),
			Horizon:        "intraday",
			RelatedSymbols: compactSymbols(topPositive.StockCode),
			WeightHint:     decimalPointerFromValue(topPositive.HoldingRatio),
		})
	}

	if topNegative := strongestContributor(holdingDetails, false); topNegative != nil && math.Abs(decimalToFloat(topNegative.Contribution)) >= 0.08 {
		events = append(events, domain.FundAnalysisEventImpact{
			Code:           "top_negative_driver",
			Title:          strings.TrimSpace(topNegative.StockName) + "盘中拖累明显",
			Impact:         "negative",
			Summary:        buildHoldingDriverSummary(*topNegative, "negative"),
			TargetScope:    "holding",
			Strength:       contributionStrength(decimalToFloat(topNegative.Contribution)),
			Horizon:        "intraday",
			RelatedSymbols: compactSymbols(topNegative.StockCode),
			WeightHint:     decimalPointerFromValue(topNegative.HoldingRatio),
		})
	}

	events = append(events, domain.FundAnalysisEventImpact{
		Code:        "analysis_basis",
		Title:       "当前分析口径",
		Impact:      "neutral",
		Summary:     "本次量化分析基于" + analysisBasis + "计算，后续排行榜与持仓页也应复用同一口径。",
		TargetScope: "methodology",
		Strength:    "low",
		Horizon:     "current",
	})

	switch {
	case concentrationWeight >= 50:
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "exposure_concentrated",
			Title:       "主线暴露过于集中",
			Impact:      "negative",
			Summary:     "当前主暴露权重已超过 50%，一旦赛道退潮，净值波动会被显著放大。",
			TargetScope: "exposure",
			Strength:    "high",
			Horizon:     "medium_term",
			WeightHint:  decimalPointerFromFloat(concentrationWeight),
		})
	case concentrationWeight >= 30:
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "exposure_clear",
			Title:       "主线暴露较清晰",
			Impact:      "positive",
			Summary:     "当前行业/主题主暴露已经形成清晰主线，更适合做结构化判断。",
			TargetScope: "exposure",
			Strength:    "medium",
			Horizon:     "medium_term",
			WeightHint:  decimalPointerFromFloat(concentrationWeight),
		})
	}

	if confidence == "low" {
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "coverage_limited",
			Title:       "识别覆盖有限",
			Impact:      "negative",
			Summary:     "未归类权益或主题占比较高，当前结果更适合作为参考而不是绝对结论。",
			TargetScope: "exposure",
			Strength:    "medium",
			Horizon:     "medium_term",
		})
	} else if hasRecognizedTheme(themeSnapshot) || hasRecognizedSector(sectorSnapshot) {
		events = append(events, domain.FundAnalysisEventImpact{
			Code:        "classification_recognized",
			Title:       "主暴露可识别",
			Impact:      "positive",
			Summary:     "行业/主题主线已经被识别出来，结构解释和后续事件层承接会更自然。",
			TargetScope: "exposure",
			Strength:    "medium",
			Horizon:     "quarterly",
		})
	}

	events = normalizeAnalysisEventImpacts(events)
	return events[:minAnalysisInt(len(events), 10)]
}

func normalizeAnalysisEventImpacts(events []domain.FundAnalysisEventImpact) []domain.FundAnalysisEventImpact {
	if len(events) == 0 {
		return nil
	}

	filtered := make([]domain.FundAnalysisEventImpact, 0, len(events))
	hasStrongSpecificCurrentEvent := false
	for _, event := range events {
		if strings.TrimSpace(event.Code) == "current_exposure_event_cluster" && hasStrongSpecificCurrentEvent {
			continue
		}
		if isSpecificCurrentEvent(event) {
			hasStrongSpecificCurrentEvent = true
		}
		filtered = append(filtered, event)
	}

	deduped := make([]domain.FundAnalysisEventImpact, 0, len(filtered))
	seen := make(map[string]struct{}, len(filtered))
	for _, event := range filtered {
		key := analysisEventDedupKey(event)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, event)
	}

	sort.SliceStable(deduped, func(i, j int) bool {
		return analysisEventPriority(deduped[i]) > analysisEventPriority(deduped[j])
	})
	return deduped
}

func isSpecificCurrentEvent(event domain.FundAnalysisEventImpact) bool {
	scope := strings.TrimSpace(event.TargetScope)
	horizon := strings.TrimSpace(event.Horizon)
	if scope != "holding" && scope != "fund" {
		return false
	}
	if horizon != "current" && horizon != "intraday" {
		return false
	}
	return strings.TrimSpace(event.Title) != ""
}

func analysisEventDedupKey(event domain.FundAnalysisEventImpact) string {
	scope := strings.TrimSpace(event.TargetScope)
	horizon := strings.TrimSpace(event.Horizon)
	title := strings.TrimSpace(event.Title)
	switch strings.TrimSpace(event.Code) {
	case "current_exposure_event_cluster":
		return "cluster:" + scope + ":" + horizon
	case "analysis_basis":
		return "methodology"
	}
	if title != "" {
		return scope + ":" + horizon + ":" + title
	}
	return strings.TrimSpace(event.Code)
}

func analysisEventPriority(event domain.FundAnalysisEventImpact) int {
	score := 0
	switch strings.TrimSpace(event.TargetScope) {
	case "holding":
		score += 50
	case "fund":
		score += 42
	case "exposure":
		score += 34
	case "macro":
		score += 26
	case "index":
		score += 10
	case "disclosure":
		score += 18
	case "methodology":
		score += 4
	}

	switch strings.TrimSpace(event.Horizon) {
	case "current":
		score += 24
	case "intraday":
		score += 20
	case "quarterly":
		score += 16
	case "medium_term":
		score += 8
	}

	switch strings.TrimSpace(event.Impact) {
	case "negative":
		score += 7
	case "positive":
		score += 6
	default:
		score += 3
	}

	switch strings.TrimSpace(event.Strength) {
	case "high":
		score += 7
	case "medium":
		score += 4
	default:
		score += 1
	}

	if event.WeightHint != nil {
		score += minAnalysisInt(int(decimalToFloat(*event.WeightHint)/5), 6)
	}
	if strings.TrimSpace(event.Code) == "analysis_basis" {
		score -= 20
	}
	if strings.TrimSpace(event.Code) == "current_exposure_event_cluster" {
		score -= 6
	}
	return score
}

func strongestContributor(details []domain.HoldingDetail, positive bool) *domain.HoldingDetail {
	var best *domain.HoldingDetail
	for i := range details {
		contribution := decimalToFloat(details[i].Contribution)
		if positive && contribution <= 0 {
			continue
		}
		if !positive && contribution >= 0 {
			continue
		}
		if best == nil {
			best = &details[i]
			continue
		}
		bestContribution := decimalToFloat(best.Contribution)
		if positive && contribution > bestContribution {
			best = &details[i]
		}
		if !positive && contribution < bestContribution {
			best = &details[i]
		}
	}
	return best
}

func contributionStrength(contribution float64) string {
	absContribution := math.Abs(contribution)
	switch {
	case absContribution >= 0.45:
		return "high"
	case absContribution >= 0.18:
		return "medium"
	default:
		return "low"
	}
}

func buildHoldingDriverSummary(detail domain.HoldingDetail, direction string) string {
	stockName := strings.TrimSpace(detail.StockName)
	if stockName == "" {
		stockName = strings.TrimSpace(detail.StockCode)
	}
	contribution := decimalToFloat(detail.Contribution)
	weightHint := ""
	if detail.HoldingRatio.GreaterThan(decimal.Zero) {
		weightHint = "，持仓占比约 " + detail.HoldingRatio.Round(2).StringFixed(2) + "%"
	}
	switch direction {
	case "negative":
		return fmt.Sprintf("%s当前贡献约 %s%s，是净值的主要拖累项之一。", stockName, signedPercentValue(contribution, 4), weightHint)
	default:
		return fmt.Sprintf("%s当前贡献约 %s%s，是净值的主要拉动项之一。", stockName, signedPercentValue(contribution, 4), weightHint)
	}
}

func signedPercentValue(value float64, digits int32) string {
	sign := "+"
	if value < 0 {
		sign = "-"
	}
	return sign + decimal.NewFromFloat(math.Abs(value)).Round(digits).StringFixed(digits) + "%"
}

func deltaPercentValue(value float64) string {
	return decimal.NewFromFloat(value).Round(1).StringFixed(1) + "pct"
}

func driftStrength(value float64) string {
	switch {
	case value >= 5:
		return "high"
	case value >= 2:
		return "medium"
	default:
		return "low"
	}
}

func displayHoldingName(delta *holdingDelta) string {
	if delta == nil {
		return "重仓股"
	}
	if strings.TrimSpace(delta.StockName) != "" {
		return strings.TrimSpace(delta.StockName)
	}
	if strings.TrimSpace(delta.StockCode) != "" {
		return strings.TrimSpace(delta.StockCode)
	}
	return "重仓股"
}

func buildHoldingDeltaSummary(delta *holdingDelta, previousPeriod, direction string) string {
	if delta == nil {
		return ""
	}
	name := displayHoldingName(delta)
	switch direction {
	case "decrease":
		if delta.CurrentWeight <= 0 {
			return fmt.Sprintf("相较 %s，%s已退出当前前十大，上一季权重约 %.2f%%。", previousPeriod, name, delta.PreviousWeight)
		}
		return fmt.Sprintf("相较 %s，%s权重由 %.2f%% 降至 %.2f%%，回落 %s。", previousPeriod, name, delta.PreviousWeight, delta.CurrentWeight, deltaPercentValue(math.Abs(delta.Delta)))
	default:
		if delta.PreviousWeight <= 0 {
			return fmt.Sprintf("相较 %s，%s新进入当前前十大，当前权重约 %.2f%%。", previousPeriod, name, delta.CurrentWeight)
		}
		return fmt.Sprintf("相较 %s，%s权重由 %.2f%% 升至 %.2f%%，提升 %s。", previousPeriod, name, delta.PreviousWeight, delta.CurrentWeight, deltaPercentValue(delta.Delta))
	}
}

func mapExposureScopeLabel(scope string) string {
	switch scope {
	case "theme":
		return "主题"
	case "sector":
		return "行业"
	default:
		return "主线"
	}
}

func buildExposureShiftSummary(shift *exposureShiftInsight) string {
	if shift == nil {
		return ""
	}
	scopeLabel := mapExposureScopeLabel(shift.Scope)
	if shift.Changed {
		return fmt.Sprintf("相较上一季，主%s由 %s 切换到 %s，组合风格出现明显偏移。", scopeLabel, shift.PreviousName, shift.CurrentName)
	}
	if shift.WeightDelta > 0 {
		return fmt.Sprintf("相较上一季，主%s %s 的权重提升 %s，当前主线更集中。", scopeLabel, shift.CurrentName, deltaPercentValue(shift.WeightDelta))
	}
	return fmt.Sprintf("相较上一季，主%s %s 的权重回落 %s，当前主线比之前更分散。", scopeLabel, shift.CurrentName, deltaPercentValue(math.Abs(shift.WeightDelta)))
}

func compactSymbols(symbols ...string) []string {
	result := make([]string, 0, len(symbols))
	seen := make(map[string]struct{}, len(symbols))
	for _, symbol := range symbols {
		symbol = strings.TrimSpace(symbol)
		if symbol == "" {
			continue
		}
		if _, ok := seen[symbol]; ok {
			continue
		}
		seen[symbol] = struct{}{}
		result = append(result, symbol)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func decimalPointerFromValue(value decimal.Decimal) *decimal.Decimal {
	if !value.GreaterThan(decimal.Zero) {
		return nil
	}
	copied := value
	return &copied
}

func decimalPointerFromFloat(value float64) *decimal.Decimal {
	if value <= 0 {
		return nil
	}
	converted := decimal.NewFromFloat(value).Round(1)
	return &converted
}

func buildAnalysisSummary(increasePercent, holdPercent, decreasePercent float64, riskLevel string) string {
	switch {
	case increasePercent >= 55:
		if riskLevel == "high" {
			return "当前偏向加仓，但波动偏高，更适合小步分批而不是激进追涨。"
		}
		return "当前偏向加仓，趋势与结构端更占优，但仍建议分批而不是一次性重仓。"
	case decreasePercent >= 60:
		return "当前偏向减仓或控制仓位，先等风险与趋势信号重新收敛更稳妥。"
	default:
		return "当前更适合持有观察，等待趋势、事件和风险信号进一步拉开差距。"
	}
}

func mapRiskLevelLabel(level string) string {
	switch level {
	case "low":
		return "低风险"
	case "medium":
		return "中风险"
	default:
		return "高风险"
	}
}

func minAnalysisInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
