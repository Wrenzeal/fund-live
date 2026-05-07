package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	EstimateCapabilityStatusSupported   = "supported"
	EstimateCapabilityStatusDegraded    = "degraded"
	EstimateCapabilityStatusUnsupported = "unsupported"

	EstimateCapabilityTypeDirectHoldings       = "direct_holdings"
	EstimateCapabilityTypeQDIIHoldings         = "qdii_holdings"
	EstimateCapabilityTypeFeederTargetHoldings = "feeder_target_holdings"
	EstimateCapabilityTypeFeederTargetQuote    = "feeder_target_quote"
	EstimateCapabilityTypeValuationProfile     = "valuation_profile"
	EstimateCapabilityTypeUnknown              = "unknown"

	EstimateQuoteModeDomestic = "domestic"
	EstimateQuoteModeOverseas = "overseas_fixed"
	EstimateQuoteModeMixed    = "mixed"
)

type EstimateCapabilityService struct {
	db  *gorm.DB
	now func() time.Time
}

type estimateHoldingsAggregate struct {
	FundID        string
	HoldingsCount int64
	TotalRatio    decimal.Decimal
	USCount       int64
	DomesticCount int64
}

func (s *EstimateCapabilityService) ListRankingCandidateFundIDs(ctx context.Context, limit int) ([]string, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 80
	}

	var records []database.FundEstimateCapability
	if err := s.db.WithContext(ctx).
		Where("capability_status IN ?", []string{EstimateCapabilityStatusSupported, EstimateCapabilityStatusDegraded}).
		Order("CASE capability_status WHEN 'supported' THEN 0 ELSE 1 END ASC").
		Order("checked_at DESC").
		Order("fund_id ASC").
		Limit(limit).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list ranking candidate fund ids: %w", err)
	}

	result := make([]string, 0, len(records))
	for _, record := range records {
		fundID := strings.TrimSpace(record.FundID)
		if fundID == "" {
			continue
		}
		result = append(result, fundID)
	}
	return result, nil
}

func NewEstimateCapabilityService(db *gorm.DB) *EstimateCapabilityService {
	return &EstimateCapabilityService{
		db:  db,
		now: time.Now,
	}
}

func (s *EstimateCapabilityService) RefreshAll(ctx context.Context, batchSize int) (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	if batchSize <= 0 {
		batchSize = 2000
	}

	totalProcessed := 0
	cursor := ""
	for {
		var batch []database.Fund
		query := s.db.WithContext(ctx).
			Model(&database.Fund{}).
			Select("id", "name", "type").
			Order("id ASC").
			Limit(batchSize)
		if cursor != "" {
			query = query.Where("id > ?", cursor)
		}
		if err := query.Find(&batch).Error; err != nil {
			return totalProcessed, fmt.Errorf("load fund capability batch: %w", err)
		}
		if len(batch) == 0 {
			break
		}

		if err := s.refreshBatch(ctx, batch); err != nil {
			return totalProcessed, err
		}

		totalProcessed += len(batch)
		cursor = batch[len(batch)-1].ID
	}

	return totalProcessed, nil
}

func (s *EstimateCapabilityService) refreshBatch(ctx context.Context, funds []database.Fund) error {
	if len(funds) == 0 {
		return nil
	}

	fundIDs := make([]string, 0, len(funds))
	for _, fund := range funds {
		fundIDs = append(fundIDs, fund.ID)
	}

	directAggByFund, err := s.loadHoldingsAggregates(ctx, fundIDs)
	if err != nil {
		return err
	}
	profileSet, err := s.loadValuationProfiles(ctx, fundIDs)
	if err != nil {
		return err
	}
	mappingsByFund, err := s.loadResolvedMappings(ctx, fundIDs)
	if err != nil {
		return err
	}

	targetIDs := make([]string, 0, len(mappingsByFund))
	for _, mapping := range mappingsByFund {
		targetCode := strings.TrimSpace(mapping.TargetCode)
		if targetCode != "" {
			targetIDs = append(targetIDs, targetCode)
		}
	}
	targetAggByFund, err := s.loadHoldingsAggregates(ctx, estimateCapabilityUniqueStrings(targetIDs))
	if err != nil {
		return err
	}

	now := time.Now()
	if s.now != nil {
		now = s.now()
	}

	capabilities := make([]database.FundEstimateCapability, 0, len(funds))
	for _, fund := range funds {
		directAgg := directAggByFund[fund.ID]
		mapping := mappingsByFund[fund.ID]
		targetAgg := targetAggByFund[strings.TrimSpace(mapping.TargetCode)]
		capability := buildEstimateCapabilityRecord(fund, directAgg, targetAgg, profileSet[fund.ID], mapping, now)
		capabilities = append(capabilities, capability)
	}

	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "fund_id"}},
			UpdateAll: true,
		}).
		Create(&capabilities).Error
}

func (s *EstimateCapabilityService) BuildCollectorTargets(ctx context.Context, defaultSource domain.QuoteSource) ([]trackedFundTarget, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}

	var records []database.FundEstimateCapability
	if err := s.db.WithContext(ctx).
		Where("capability_status IN ?", []string{EstimateCapabilityStatusSupported, EstimateCapabilityStatusDegraded}).
		Order("fund_id ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("load estimate capabilities: %w", err)
	}

	targets := make([]trackedFundTarget, 0, len(records))
	for _, record := range records {
		if strings.TrimSpace(record.FundID) == "" {
			continue
		}
		refreshInterval := 3 * time.Minute
		if strings.TrimSpace(record.CapabilityStatus) == EstimateCapabilityStatusDegraded {
			refreshInterval = 5 * time.Minute
		}
		quoteMode := normalizeEstimateQuoteMode(record.QuoteSourceMode)
		targets = append(targets, trackedFundTarget{
			FundID:          record.FundID,
			Source:          domain.ResolveQuoteSource(defaultSource, domain.QuoteSourceSina),
			LastTrackedAt:   time.Time{},
			LastCollectedAt: time.Time{},
			RefreshInterval: refreshInterval,
			QuoteMode:       quoteMode,
			Persistent:      true,
		})
	}
	return targets, nil
}

func (s *EstimateCapabilityService) loadResolvedMappings(ctx context.Context, fundIDs []string) (map[string]database.FundMapping, error) {
	result := make(map[string]database.FundMapping)
	if len(fundIDs) == 0 {
		return result, nil
	}

	var mappings []database.FundMapping
	if err := s.db.WithContext(ctx).
		Where("feeder_code IN ? AND is_resolved = ? AND target_code <> ''", fundIDs, true).
		Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("load resolved fund mappings: %w", err)
	}
	for _, mapping := range mappings {
		result[mapping.FeederCode] = mapping
	}
	return result, nil
}

func (s *EstimateCapabilityService) loadValuationProfiles(ctx context.Context, fundIDs []string) (map[string]bool, error) {
	result := make(map[string]bool)
	if len(fundIDs) == 0 {
		return result, nil
	}

	var rows []struct {
		FundID string
	}
	if err := s.db.WithContext(ctx).
		Model(&database.FundValuationProfile{}).
		Select("fund_id").
		Where("fund_id IN ?", fundIDs).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load valuation profiles: %w", err)
	}
	for _, row := range rows {
		result[row.FundID] = true
	}
	return result, nil
}

func (s *EstimateCapabilityService) loadHoldingsAggregates(ctx context.Context, fundIDs []string) (map[string]estimateHoldingsAggregate, error) {
	result := make(map[string]estimateHoldingsAggregate)
	if len(fundIDs) == 0 {
		return result, nil
	}

	var rows []estimateHoldingsAggregate
	if err := s.db.WithContext(ctx).
		Model(&database.StockHolding{}).
		Select(`
			fund_id,
			COUNT(*) AS holdings_count,
			COALESCE(SUM(CASE WHEN holding_ratio > 0 THEN holding_ratio ELSE 0 END), 0) AS total_ratio,
			COALESCE(SUM(CASE WHEN exchange = 'US' THEN 1 ELSE 0 END), 0) AS us_count,
			COALESCE(SUM(CASE WHEN exchange <> 'US' THEN 1 ELSE 0 END), 0) AS domestic_count
		`).
		Where("fund_id IN ?", fundIDs).
		Group("fund_id").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load holdings aggregates: %w", err)
	}

	for _, row := range rows {
		result[row.FundID] = row
	}
	return result, nil
}

func buildEstimateCapabilityRecord(
	fund database.Fund,
	directAgg estimateHoldingsAggregate,
	targetAgg estimateHoldingsAggregate,
	hasValuationProfile bool,
	mapping database.FundMapping,
	now time.Time,
) database.FundEstimateCapability {
	record := database.FundEstimateCapability{
		FundID:               fund.ID,
		CapabilityStatus:     EstimateCapabilityStatusUnsupported,
		CapabilityType:       EstimateCapabilityTypeUnknown,
		QuoteSourceMode:      EstimateQuoteModeDomestic,
		TargetCode:           strings.TrimSpace(mapping.TargetCode),
		HoldingsCount:        directAgg.HoldingsCount,
		TotalHoldRatio:       directAgg.TotalRatio,
		HasEffectiveHoldings: directAgg.TotalRatio.GreaterThan(decimal.Zero),
		HasValuationProfile:  hasValuationProfile,
		HasTargetMapping:     strings.TrimSpace(mapping.TargetCode) != "",
		CheckedAt:            now,
	}

	switch {
	case directAgg.TotalRatio.GreaterThan(decimal.Zero):
		record.CapabilityStatus = EstimateCapabilityStatusSupported
		record.CapabilityType = EstimateCapabilityTypeDirectHoldings
		if isDatabaseQDIIFund(fund) {
			record.CapabilityType = EstimateCapabilityTypeQDIIHoldings
		}
		record.QuoteSourceMode = inferEstimateQuoteMode(directAgg)
	case hasValuationProfile:
		record.CapabilityStatus = EstimateCapabilityStatusSupported
		record.CapabilityType = EstimateCapabilityTypeValuationProfile
		record.QuoteSourceMode = EstimateQuoteModeDomestic
	case strings.TrimSpace(mapping.TargetCode) != "" && targetAgg.TotalRatio.GreaterThan(decimal.Zero):
		record.CapabilityStatus = EstimateCapabilityStatusSupported
		record.CapabilityType = EstimateCapabilityTypeFeederTargetHoldings
		record.HoldingsCount = targetAgg.HoldingsCount
		record.TotalHoldRatio = targetAgg.TotalRatio
		record.QuoteSourceMode = inferEstimateQuoteMode(targetAgg)
	case strings.TrimSpace(mapping.TargetCode) != "":
		record.CapabilityStatus = EstimateCapabilityStatusDegraded
		record.CapabilityType = EstimateCapabilityTypeFeederTargetQuote
		record.QuoteSourceMode = EstimateQuoteModeDomestic
	}

	return record
}

func inferEstimateQuoteMode(agg estimateHoldingsAggregate) string {
	switch {
	case agg.USCount > 0 && agg.DomesticCount > 0:
		return EstimateQuoteModeMixed
	case agg.USCount > 0:
		return EstimateQuoteModeOverseas
	default:
		return EstimateQuoteModeDomestic
	}
}

func normalizeEstimateQuoteMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case EstimateQuoteModeOverseas:
		return EstimateQuoteModeOverseas
	case EstimateQuoteModeMixed:
		return EstimateQuoteModeMixed
	default:
		return EstimateQuoteModeDomestic
	}
}

func isDatabaseQDIIFund(fund database.Fund) bool {
	fundType := strings.ToLower(strings.TrimSpace(fund.Type))
	name := strings.ToLower(strings.TrimSpace(fund.Name))
	return strings.Contains(fundType, "qdii") || strings.Contains(name, "qdii")
}

func estimateCapabilityUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

type EstimateCoverageScheduler struct {
	capabilityService *EstimateCapabilityService
	valuationService  *ValuationServiceImpl
	defaultSource     domain.QuoteSource
	scanBatchSize     int
	refreshInterval   time.Duration
}

func NewEstimateCoverageScheduler(
	capabilityService *EstimateCapabilityService,
	valuationService *ValuationServiceImpl,
	defaultSource domain.QuoteSource,
) *EstimateCoverageScheduler {
	return &EstimateCoverageScheduler{
		capabilityService: capabilityService,
		valuationService:  valuationService,
		defaultSource:     domain.ResolveQuoteSource(defaultSource, domain.QuoteSourceSina),
		scanBatchSize:     2000,
		refreshInterval:   30 * time.Minute,
	}
}

func (s *EstimateCoverageScheduler) Start(ctx context.Context) {
	if s == nil || s.capabilityService == nil || s.valuationService == nil {
		return
	}

	go s.run(ctx)
}

func (s *EstimateCoverageScheduler) run(ctx context.Context) {
	s.refreshAndSync(ctx)

	ticker := time.NewTicker(s.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.refreshAndSync(ctx)
		}
	}
}

func (s *EstimateCoverageScheduler) refreshAndSync(ctx context.Context) {
	processed, err := s.capabilityService.RefreshAll(ctx, s.scanBatchSize)
	if err != nil {
		log.Printf("⚠️ Estimate capability refresh failed: %v", err)
		return
	}

	targets, err := s.capabilityService.BuildCollectorTargets(ctx, s.defaultSource)
	if err != nil {
		log.Printf("⚠️ Estimate capability target build failed: %v", err)
		return
	}

	s.valuationService.SetManagedTargets(targets)
	log.Printf("📡 Estimate capability refresh complete: scanned %d funds, collector managed targets=%d", processed, len(targets))
}
