package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	QuantUniversePilotV1        = "pilot-v1"
	QuantSignalModeFullV4       = "full_v4_forward"
	QuantSignalModeHistoryProxy = "historical_proxy"
	QuantStrategyTop5Weekly     = "top5_weekly_equal_weight"
)

type QuantResearchStore struct {
	db *gorm.DB
}

func NewQuantResearchStore(db *gorm.DB) *QuantResearchStore {
	if db == nil {
		return nil
	}
	return &QuantResearchStore{db: db}
}

type quantSignalInputManifest struct {
	AnalysisBasis       string                                `json:"analysis_basis"`
	LatestHoldingPeriod string                                `json:"latest_holding_period,omitempty"`
	ModuleScores        []domain.FundAnalysisModuleScore      `json:"module_scores"`
	PrimaryEvidence     []domain.FundAnalysisEvidenceItem     `json:"primary_evidence"`
	CounterEvidence     []domain.FundAnalysisEvidenceItem     `json:"counter_evidence"`
	EventIntelligence   *domain.FundAnalysisEventIntelligence `json:"event_intelligence,omitempty"`
}

func (s *QuantResearchStore) SaveForwardSignal(ctx context.Context, fundID string, analysis *domain.FundAnalysis, decisionAt time.Time) error {
	if s == nil || s.db == nil || strings.TrimSpace(fundID) == "" || analysis == nil {
		return nil
	}
	if decisionAt.IsZero() {
		decisionAt = analysis.AsOfTime
	}
	if decisionAt.IsZero() {
		decisionAt = time.Now()
	}
	loc := time.FixedZone("CST", 8*60*60)
	signalDate := time.Date(decisionAt.In(loc).Year(), decisionAt.In(loc).Month(), decisionAt.In(loc).Day(), 0, 0, 0, 0, loc)

	inputJSON, err := json.Marshal(quantSignalInputManifest{
		AnalysisBasis:       analysis.AnalysisBasis,
		LatestHoldingPeriod: analysis.LatestHoldingPeriod,
		ModuleScores:        analysis.ModuleScores,
		PrimaryEvidence:     analysis.PrimaryEvidence,
		CounterEvidence:     analysis.CounterEvidence,
		EventIntelligence:   analysis.EventIntelligence,
	})
	if err != nil {
		return fmt.Errorf("marshal signal input manifest: %w", err)
	}
	outputJSON, err := json.Marshal(analysis)
	if err != nil {
		return fmt.Errorf("marshal signal output: %w", err)
	}
	eventIDs := make([]string, 0)
	if analysis.EventIntelligence != nil {
		for _, event := range analysis.EventIntelligence.Timeline {
			if event.EventID != "" {
				eventIDs = append(eventIDs, event.EventID)
			}
		}
	}
	eventJSON, _ := json.Marshal(uniqueSortedStrings(eventIDs))
	shadowScore := decimal.Zero
	if analysis.EventIntelligence != nil {
		shadowScore = analysis.EventIntelligence.ShadowEventScore
	}
	record := database.QuantSignalHistory{
		FundID:           strings.TrimSpace(fundID),
		AnalysisVersion:  analysis.AnalysisVersion,
		SignalDate:       signalDate,
		Mode:             QuantSignalModeFullV4,
		DecisionAt:       decisionAt,
		DataCutoffAt:     analysis.AsOfTime,
		TotalScore:       analysis.TotalScore,
		ShadowEventScore: shadowScore,
		InputJSON:        inputJSON,
		OutputJSON:       outputJSON,
		EventIDs:         eventJSON,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&record).Error
}

func uniqueSortedStrings(values []string) []string {
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
	sort.Strings(result)
	return result
}

type QuantBacktestRequest struct {
	StartDate            string          `json:"start_date"`
	EndDate              string          `json:"end_date"`
	UniverseVersion      string          `json:"universe_version"`
	SignalMode           string          `json:"signal_mode"`
	InitialCash          decimal.Decimal `json:"initial_cash"`
	TopN                 int             `json:"top_n"`
	CommissionBPS        decimal.Decimal `json:"commission_bps"`
	MinimumCommissionCNY decimal.Decimal `json:"minimum_commission_cny"`
	SlippageBPS          decimal.Decimal `json:"slippage_bps"`
	MinimumListingDays   int             `json:"minimum_listing_days"`
	MinimumAverageAmount decimal.Decimal `json:"minimum_average_amount"`
}

func DefaultQuantBacktestRequest() QuantBacktestRequest {
	end := time.Now().In(time.FixedZone("CST", 8*60*60))
	return QuantBacktestRequest{
		StartDate:            end.AddDate(-5, 0, 0).Format("2006-01-02"),
		EndDate:              end.Format("2006-01-02"),
		UniverseVersion:      QuantUniversePilotV1,
		SignalMode:           QuantSignalModeHistoryProxy,
		InitialCash:          decimal.NewFromInt(1_000_000),
		TopN:                 5,
		CommissionBPS:        decimal.NewFromInt(3),
		MinimumCommissionCNY: decimal.NewFromInt(5),
		SlippageBPS:          decimal.NewFromInt(5),
		MinimumListingDays:   120,
		MinimumAverageAmount: decimal.NewFromInt(20_000_000),
	}
}

func NormalizeQuantBacktestRequest(input QuantBacktestRequest) (QuantBacktestRequest, error) {
	defaults := DefaultQuantBacktestRequest()
	if strings.TrimSpace(input.StartDate) == "" {
		input.StartDate = defaults.StartDate
	}
	if strings.TrimSpace(input.EndDate) == "" {
		input.EndDate = defaults.EndDate
	}
	start, startErr := time.Parse("2006-01-02", input.StartDate)
	end, endErr := time.Parse("2006-01-02", input.EndDate)
	if startErr != nil || endErr != nil || end.Before(start) {
		return input, fmt.Errorf("start_date and end_date must be valid YYYY-MM-DD values")
	}
	if end.Sub(start) > 10*366*24*time.Hour {
		return input, fmt.Errorf("backtest range cannot exceed 10 years")
	}
	if strings.TrimSpace(input.UniverseVersion) == "" {
		input.UniverseVersion = defaults.UniverseVersion
	}
	if strings.TrimSpace(input.SignalMode) == "" {
		input.SignalMode = defaults.SignalMode
	}
	if input.SignalMode != QuantSignalModeHistoryProxy && input.SignalMode != QuantSignalModeFullV4 {
		return input, fmt.Errorf("unsupported signal_mode")
	}
	if !input.InitialCash.IsPositive() {
		input.InitialCash = defaults.InitialCash
	}
	if input.TopN <= 0 || input.TopN > 20 {
		input.TopN = defaults.TopN
	}
	if input.CommissionBPS.IsNegative() {
		return input, fmt.Errorf("commission_bps cannot be negative")
	}
	if input.CommissionBPS.IsZero() {
		input.CommissionBPS = defaults.CommissionBPS
	}
	if input.MinimumCommissionCNY.IsNegative() {
		return input, fmt.Errorf("minimum_commission_cny cannot be negative")
	}
	if input.MinimumCommissionCNY.IsZero() {
		input.MinimumCommissionCNY = defaults.MinimumCommissionCNY
	}
	if input.SlippageBPS.IsNegative() {
		return input, fmt.Errorf("slippage_bps cannot be negative")
	}
	if input.SlippageBPS.IsZero() {
		input.SlippageBPS = defaults.SlippageBPS
	}
	if input.MinimumListingDays <= 0 {
		input.MinimumListingDays = defaults.MinimumListingDays
	}
	if !input.MinimumAverageAmount.IsPositive() {
		input.MinimumAverageAmount = defaults.MinimumAverageAmount
	}
	return input, nil
}

func (s *QuantResearchStore) CreateBacktestJob(ctx context.Context, request QuantBacktestRequest) (*database.QuantBacktestJob, bool, error) {
	if s == nil || s.db == nil {
		return nil, false, fmt.Errorf("quant research store is unavailable")
	}
	normalized, err := NormalizeQuantBacktestRequest(request)
	if err != nil {
		return nil, false, err
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return nil, false, err
	}
	sum := sha256.Sum256(append([]byte(QuantStrategyTop5Weekly+"|lean|"), payload...))
	idempotencyKey := hex.EncodeToString(sum[:])

	var existing database.QuantBacktestJob
	if err := s.db.WithContext(ctx).Where("idempotency_key = ?", idempotencyKey).First(&existing).Error; err == nil {
		return &existing, false, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, false, err
	}
	jobID, err := randomHex(16)
	if err != nil {
		return nil, false, err
	}
	job := database.QuantBacktestJob{
		ID:              jobID,
		IdempotencyKey:  idempotencyKey,
		Status:          "queued",
		Strategy:        QuantStrategyTop5Weekly,
		UniverseVersion: normalized.UniverseVersion,
		SignalMode:      normalized.SignalMode,
		Engine:          "lean",
		ParametersJSON:  payload,
	}
	if err := s.db.WithContext(ctx).Create(&job).Error; err != nil {
		return nil, false, err
	}
	return &job, true, nil
}

func randomHex(bytesCount int) (string, error) {
	buffer := make([]byte, bytesCount)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func (s *QuantResearchStore) GetBacktestJob(ctx context.Context, jobID string) (*database.QuantBacktestJob, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("quant research store is unavailable")
	}
	var job database.QuantBacktestJob
	if err := s.db.WithContext(ctx).First(&job, "id = ?", strings.TrimSpace(jobID)).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *QuantResearchStore) ListBacktestJobs(ctx context.Context, limit int) ([]database.QuantBacktestJob, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("quant research store is unavailable")
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	var jobs []database.QuantBacktestJob
	if err := s.db.WithContext(ctx).
		Select("id", "status", "strategy", "universe_version", "signal_mode", "engine", "engine_version", "error_message", "attempt_count", "created_at", "started_at", "completed_at", "updated_at").
		Order("created_at DESC").Limit(limit).Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

func (s *QuantResearchStore) MarkBacktestQueueFailed(ctx context.Context, jobID string, cause error) error {
	if s == nil || s.db == nil {
		return nil
	}
	message := "queue unavailable"
	if cause != nil {
		message = cause.Error()
	}
	return s.db.WithContext(ctx).Model(&database.QuantBacktestJob{}).Where("id = ? AND status = ?", jobID, "queued").Updates(map[string]interface{}{
		"status":        "queue_failed",
		"error_message": message,
	}).Error
}

func (s *QuantResearchStore) RetryBacktestJob(ctx context.Context, jobID string) (bool, error) {
	if s == nil || s.db == nil {
		return false, fmt.Errorf("quant research store is unavailable")
	}
	result := s.db.WithContext(ctx).Model(&database.QuantBacktestJob{}).
		Where("id = ? AND status IN ?", strings.TrimSpace(jobID), []string{"failed", "queue_failed"}).
		Updates(map[string]interface{}{
			"status":        "queued",
			"error_message": "",
			"started_at":    nil,
			"completed_at":  nil,
		})
	return result.RowsAffected == 1, result.Error
}

type PilotInstrument struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Bucket string `json:"bucket"`
}

var pilotV1Instruments = []PilotInstrument{
	{"510300", "沪深300ETF", "broad"}, {"510050", "上证50ETF", "broad"}, {"510500", "中证500ETF", "broad"},
	{"159915", "创业板ETF", "broad"}, {"588000", "科创50ETF", "broad"}, {"512100", "中证1000ETF", "broad"},
	{"510880", "红利ETF", "style"}, {"515180", "红利ETF易方达", "style"}, {"159949", "创业板50ETF", "style"},
	{"512800", "银行ETF", "industry"}, {"512480", "半导体ETF", "industry"}, {"512010", "医药ETF", "industry"},
	{"512660", "军工ETF", "industry"}, {"512690", "酒ETF", "industry"}, {"515030", "新能源车ETF", "industry"},
	{"512170", "医疗ETF", "industry"}, {"159869", "游戏ETF", "industry"}, {"159928", "消费ETF", "industry"},
	{"511010", "国债ETF", "defensive"}, {"511260", "十年国债ETF", "defensive"}, {"511880", "银华日利ETF", "defensive"},
	{"518880", "黄金ETF", "commodity"}, {"159920", "恒生ETF", "overseas"}, {"513100", "纳指ETF", "overseas"},
	{"513500", "标普500ETF", "overseas"},
}

func (s *QuantResearchStore) SeedPilotUniverse(ctx context.Context, now time.Time) error {
	if s == nil || s.db == nil {
		return nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		benchmark := database.QuantInstrument{Symbol: "000300", Name: "沪深300", Exchange: "CSI", AssetClass: "index", UniverseGroup: "benchmark", Source: "eastmoney"}
		if err := tx.Clauses(clause.OnConflict{DoUpdates: clause.AssignmentColumns([]string{"name", "exchange", "asset_class", "universe_group", "source", "updated_at"})}).Create(&benchmark).Error; err != nil {
			return err
		}
		for _, item := range pilotV1Instruments {
			exchange := "SZ"
			if strings.HasPrefix(item.Symbol, "5") {
				exchange = "SH"
			}
			instrument := database.QuantInstrument{Symbol: item.Symbol, Name: item.Name, Exchange: exchange, AssetClass: "etf", UniverseGroup: item.Bucket, Source: "eastmoney"}
			if err := tx.Clauses(clause.OnConflict{DoUpdates: clause.AssignmentColumns([]string{"name", "exchange", "asset_class", "universe_group", "source", "updated_at"})}).Create(&instrument).Error; err != nil {
				return err
			}
			member := database.QuantUniverseMember{UniverseVersion: QuantUniversePilotV1, Symbol: item.Symbol, Bucket: item.Bucket, IncludedAt: now}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&member).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func PilotV1Instruments() []PilotInstrument {
	return append([]PilotInstrument(nil), pilotV1Instruments...)
}
