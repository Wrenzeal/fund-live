package service

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/trading"
	"github.com/shopspring/decimal"
)

type analysisFundRepository interface {
	domain.FundRepository
}

type analysisHoldingsResolver interface {
	GetHoldingsWithFallback(ctx context.Context, fundID string, fundName string) ([]domain.StockHolding, string, error)
	ResolveDisplayHoldings(ctx context.Context, fundID string, fundName string) (*domain.FundHoldingsDisplay, error)
}

type analysisSectorStore interface {
	UpsertFromHoldings(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundSectorSnapshot, error)
	UpsertThemeFromHoldings(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundThemeSnapshot, error)
	BuildSnapshot(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundSectorSnapshot, error)
	BuildThemeSnapshot(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundThemeSnapshot, error)
	ResolveFundCategory(ctx context.Context, fund *domain.Fund, snapshot *domain.FundSectorSnapshot) (*domain.FundCategory, error)
}

type FundAnalysisCoordinator struct {
	valuationService   domain.ValuationService
	fundRepo           analysisFundRepository
	holdingsResolver   analysisHoldingsResolver
	sectorStore        analysisSectorStore
	analysisService    *FundAnalysisService
	explanationService *AIExplanationService
	eventStore         *QuantEventStore
}

func (c *FundAnalysisCoordinator) SetQuantEventStore(store *QuantEventStore) {
	if c != nil {
		c.eventStore = store
	}
}

func NewFundAnalysisCoordinator(
	valuationService domain.ValuationService,
	fundRepo analysisFundRepository,
	holdingsResolver analysisHoldingsResolver,
	sectorStore analysisSectorStore,
) *FundAnalysisCoordinator {
	return &FundAnalysisCoordinator{
		valuationService:   valuationService,
		fundRepo:           fundRepo,
		holdingsResolver:   holdingsResolver,
		sectorStore:        sectorStore,
		analysisService:    NewFundAnalysisService(),
		explanationService: NewAIExplanationService(nil),
	}
}

func (c *FundAnalysisCoordinator) SetAIExplanationService(service *AIExplanationService) {
	if c != nil && service != nil {
		c.explanationService = service
	}
}

func (c *FundAnalysisCoordinator) BuildForFund(ctx context.Context, fundID string, now time.Time) (*domain.Fund, *domain.FundAnalysis, error) {
	if c == nil || c.fundRepo == nil || c.valuationService == nil {
		return nil, nil, nil
	}
	if now.IsZero() {
		now = time.Now()
	}

	fund, err := c.fundRepo.GetFundByID(ctx, fundID)
	if err != nil || fund == nil {
		return fund, nil, err
	}
	estimate, err := c.valuationService.CalculateEstimate(ctx, fundID)
	if err != nil {
		return fund, nil, err
	}
	timeSeries, err := c.valuationService.GetIntradayTimeSeries(ctx, fundID)
	if err != nil {
		return fund, nil, err
	}

	marketStatus := trading.GetMarketStatus(now)
	displayDate, _ := deriveTimeSeriesDisplayContextForAnalysis(timeSeries, marketStatus)
	timeSeries = alignTimeSeriesWithEstimateSnapshotForAnalysis(timeSeries, estimate, displayDate, marketStatus)

	sectorSnapshot, themeSnapshot, input, err := c.buildInput(ctx, fundID, fund, estimate, timeSeries, now)
	if err != nil {
		return fund, nil, err
	}

	if sectorSnapshot != nil && c.sectorStore != nil {
		category, categoryErr := c.sectorStore.ResolveFundCategory(ctx, fund, sectorSnapshot)
		if categoryErr != nil {
			log.Printf("⚠️ Failed to resolve fund category for %s: %v", fundID, categoryErr)
		} else if category != nil {
			fund.CategoryCode = category.Code
			fund.CategoryName = category.Name
		}
	}
	input.SectorSnapshot = sectorSnapshot
	input.ThemeSnapshot = themeSnapshot

	analysis := c.analysisService.Build(input)
	return fund, AttachAIExplanation(ctx, c.explanationService, AIExplanationInput{
		Fund:           fund,
		Analysis:       analysis,
		Holdings:       input.Holdings,
		SectorSnapshot: sectorSnapshot,
		ThemeSnapshot:  themeSnapshot,
		Now:            now,
	}), nil
}

func (c *FundAnalysisCoordinator) buildInput(
	ctx context.Context,
	fundID string,
	fund *domain.Fund,
	estimate *domain.FundEstimate,
	timeSeries []domain.TimeSeriesPoint,
	now time.Time,
) (*domain.FundSectorSnapshot, *domain.FundThemeSnapshot, FundAnalysisInput, error) {
	input := FundAnalysisInput{
		Fund:       fund,
		Estimate:   estimate,
		TimeSeries: timeSeries,
		Now:        now,
	}
	if fund == nil {
		return nil, nil, input, nil
	}

	holdings, source, err := c.resolveClassificationHoldings(ctx, fundID, fund)
	if err != nil {
		return nil, nil, input, err
	}
	input.Holdings = holdings
	input.HoldingsSource = source

	var sectorSnapshot *domain.FundSectorSnapshot
	var themeSnapshot *domain.FundThemeSnapshot
	if c.sectorStore != nil && hasEffectiveAnalysisHoldings(holdings) {
		sectorSnapshot, err = c.sectorStore.UpsertFromHoldings(ctx, fundID, holdings, source)
		if err != nil {
			log.Printf("⚠️ Failed to build fund sector snapshot for %s: %v", fundID, err)
		}
		themeSnapshot, err = c.sectorStore.UpsertThemeFromHoldings(ctx, fundID, holdings, source)
		if err != nil {
			log.Printf("⚠️ Failed to load fund theme snapshot for %s: %v", fundID, err)
		}
	}

	historySourceCode := fundID
	targetCode := ""
	if source == SectorSourceTargetETFFallback {
		targetCode = c.resolveTrackedETFCode(ctx, fundID, fund)
		if targetCode != "" {
			historySourceCode = targetCode
		}
	}

	input.CurrentHoldingEvents = LoadCurrentHoldingNewsEvents(ctx, holdings, now)
	input.CurrentFundEvents = LoadCurrentFundNoticeEvents(ctx, fundID, now)
	input.CurrentMacroEvents = LoadMacroPolicyEvents(now, sectorSnapshot, themeSnapshot)
	input.CurrentIndexEvents = LoadIndexLayerEvents(now, fund, source, sectorSnapshot, themeSnapshot)
	if source == SectorSourceTargetETFFallback && targetCode != "" {
		input.CurrentTargetEvents = LoadCurrentFundNoticeEvents(ctx, targetCode, now)
	}
	if c.eventStore != nil {
		currentEvents := mergeCurrentEventImpacts(input.CurrentHoldingEvents, input.CurrentFundEvents, input.CurrentTargetEvents, input.CurrentMacroEvents, input.CurrentIndexEvents)
		if saveErr := c.eventStore.SaveImpacts(ctx, fundID, currentEvents, now); saveErr != nil {
			log.Printf("⚠️ Failed to persist quant events for %s: %v", fundID, saveErr)
		} else if persisted, listErr := c.eventStore.ListAsOf(ctx, fundID, now, "", 50); listErr == nil && len(persisted) > 0 {
			input.CurrentHoldingEvents = filterEventsByTargetScope(persisted, "holding")
			input.CurrentFundEvents = filterEventsByTargetScope(persisted, "fund")
			input.CurrentTargetEvents = nil
			input.CurrentMacroEvents = filterEventsByTargetScope(persisted, "macro")
			input.CurrentIndexEvents = filterEventsByTargetScope(persisted, "index")
		}
	}
	previousHoldings, previousHoldingPeriod, previousErr := LoadPreviousQuarterHoldings(ctx, historySourceCode, holdings)
	if previousErr != nil {
		log.Printf("⚠️ Failed to load previous quarter holdings for %s: %v", fundID, previousErr)
	}
	input.PreviousHoldings = previousHoldings
	input.PreviousHoldingPeriod = previousHoldingPeriod
	if c.sectorStore != nil && hasEffectiveAnalysisHoldings(previousHoldings) {
		input.PreviousSectorSnapshot, _ = c.sectorStore.BuildSnapshot(ctx, fundID, previousHoldings, source)
		input.PreviousThemeSnapshot, _ = c.sectorStore.BuildThemeSnapshot(ctx, fundID, previousHoldings, source)
	}

	return sectorSnapshot, themeSnapshot, input, nil
}

func filterEventsByTargetScope(events []domain.FundAnalysisEventImpact, scope string) []domain.FundAnalysisEventImpact {
	result := make([]domain.FundAnalysisEventImpact, 0, len(events))
	for _, event := range events {
		if strings.TrimSpace(event.TargetScope) == scope {
			result = append(result, event)
		}
	}
	return result
}

func (c *FundAnalysisCoordinator) resolveClassificationHoldings(ctx context.Context, fundID string, fund *domain.Fund) ([]domain.StockHolding, string, error) {
	if c == nil || c.fundRepo == nil || fund == nil {
		return nil, "", nil
	}
	holdings, err := c.fundRepo.GetFundHoldings(ctx, fundID)
	if err != nil {
		return nil, "", err
	}
	source := SectorSourceDirectHoldings
	if c.holdingsResolver != nil {
		displayHoldings, displayErr := c.holdingsResolver.ResolveDisplayHoldings(ctx, fundID, fund.Name)
		if displayErr != nil {
			return nil, "", displayErr
		}
		if targetItem, ok := domain.PrimaryTrackedETF(displayHoldings); ok {
			targetCode := strings.TrimSpace(targetItem.Code)
			if targetCode != "" {
				targetHoldings, _ := c.resolveTrackedETFHoldings(ctx, targetCode)
				if hasEffectiveAnalysisHoldings(targetHoldings) {
					holdings = targetHoldings
					source = SectorSourceTargetETFFallback
				}
			}
		}
	}
	if !hasEffectiveAnalysisHoldings(holdings) && c.holdingsResolver != nil {
		resolvedHoldings, holdingsSource, resolveErr := c.holdingsResolver.GetHoldingsWithFallback(ctx, fundID, fund.Name)
		if resolveErr != nil {
			return nil, "", resolveErr
		}
		if len(resolvedHoldings) > 0 {
			holdings = resolvedHoldings
			if holdingsSource != "" && holdingsSource != fundID {
				source = SectorSourceTargetETFFallback
			}
		}
	}
	if strings.Contains(strings.ToLower(fund.Type), "qdii") || strings.Contains(strings.ToLower(fund.Name), "qdii") {
		source = SectorSourceQDIIHoldings
	}
	return holdings, source, nil
}

func (c *FundAnalysisCoordinator) resolveTrackedETFCode(ctx context.Context, fundID string, fund *domain.Fund) string {
	if c == nil || c.holdingsResolver == nil || fund == nil {
		return ""
	}
	displayHoldings, err := c.holdingsResolver.ResolveDisplayHoldings(ctx, fundID, fund.Name)
	if err != nil {
		return ""
	}
	if targetItem, ok := domain.PrimaryTrackedETF(displayHoldings); ok {
		return strings.TrimSpace(targetItem.Code)
	}
	return ""
}

func (c *FundAnalysisCoordinator) resolveTrackedETFHoldings(ctx context.Context, targetCode string) ([]domain.StockHolding, string) {
	if c == nil || c.fundRepo == nil {
		return nil, ""
	}
	holdings, err := c.fundRepo.GetFundHoldings(ctx, strings.TrimSpace(targetCode))
	if err != nil {
		return nil, ""
	}
	return holdings, ""
}

func hasEffectiveAnalysisHoldings(holdings []domain.StockHolding) bool {
	for _, holding := range holdings {
		if holding.HoldingRatio.GreaterThan(decimal.Zero) {
			return true
		}
	}
	return false
}

func deriveTimeSeriesDisplayContextForAnalysis(points []domain.TimeSeriesPoint, marketStatus trading.MarketStatus) (string, bool) {
	if len(points) == 0 {
		return marketStatus.DisplayDate, false
	}
	firstPointDate := points[0].Timestamp.In(trading.TradingLocation()).Format("2006-01-02")
	isHistorical := firstPointDate != marketStatus.CurrentDate
	if isHistorical {
		return firstPointDate, true
	}
	return marketStatus.DisplayDate, false
}

func alignTimeSeriesWithEstimateSnapshotForAnalysis(points []domain.TimeSeriesPoint, estimate *domain.FundEstimate, displayDate string, marketStatus trading.MarketStatus) []domain.TimeSeriesPoint {
	if estimate == nil || displayDate == "" {
		return points
	}
	loc := trading.TradingLocation()
	calculatedAt := estimate.CalculatedAt.In(loc)
	displayDay, err := time.ParseInLocation("2006-01-02", displayDate, loc)
	if err != nil {
		return points
	}
	targetTimestamp := time.Date(displayDay.Year(), displayDay.Month(), displayDay.Day(), 15, 0, 0, 0, loc)
	if displayDate == marketStatus.CurrentDate && marketStatus.IsTradingDay {
		targetTimestamp = time.Date(calculatedAt.Year(), calculatedAt.Month(), calculatedAt.Day(), calculatedAt.Hour(), (calculatedAt.Minute()/5)*5, 0, 0, loc)
	}
	alignedPoint := domain.TimeSeriesPoint{Timestamp: targetTimestamp, ChangePercent: estimate.ChangePercent, EstimateNav: estimate.EstimateNav}
	if len(points) == 0 {
		return []domain.TimeSeriesPoint{alignedPoint}
	}
	aligned := make([]domain.TimeSeriesPoint, len(points))
	copy(aligned, points)
	for i := range aligned {
		if aligned[i].Timestamp.In(loc).Equal(targetTimestamp) {
			aligned[i].ChangePercent = alignedPoint.ChangePercent
			aligned[i].EstimateNav = alignedPoint.EstimateNav
			return aligned
		}
	}
	aligned = append(aligned, alignedPoint)
	sort.Slice(aligned, func(i, j int) bool { return aligned[i].Timestamp.Before(aligned[j].Timestamp) })
	return aligned
}
