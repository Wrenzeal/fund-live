package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const QuantExperimentPresetRiskV1 = "risk-v1"

type QuantExperimentVariant struct {
	Key         string                    `json:"key"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Job         database.QuantBacktestJob `json:"job"`
	RiskGate    *QuantRiskGateAssessment  `json:"risk_gate,omitempty"`
}

type QuantRiskGateAssessment struct {
	Qualified        bool `json:"qualified"`
	SharpeImproved   bool `json:"sharpe_improved"`
	DrawdownImproved bool `json:"drawdown_improved"`
	CAGRPreserved    bool `json:"cagr_preserved"`
	AnnualWins       int  `json:"annual_wins"`
	ComparableYears  int  `json:"comparable_years"`
}

type QuantExperimentView struct {
	ID              string                   `json:"id"`
	Preset          string                   `json:"preset"`
	Status          string                   `json:"status"`
	UniverseVersion string                   `json:"universe_version"`
	SignalMode      string                   `json:"signal_mode"`
	StartDate       string                   `json:"start_date"`
	EndDate         string                   `json:"end_date"`
	BaseParameters  QuantBacktestRequest     `json:"base_parameters"`
	Variants        []QuantExperimentVariant `json:"variants"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

type quantRiskVariantDefinition struct {
	Key         string
	Name        string
	Description string
	Request     QuantBacktestRequest
}

func RiskV1Requests(base QuantBacktestRequest) ([]quantRiskVariantDefinition, error) {
	normalized, err := NormalizeQuantBacktestRequest(base)
	if err != nil {
		return nil, err
	}
	normalized.TopN = 5
	normalized.ExitRank = 5
	normalized.RebalanceFrequency = QuantRebalanceWeekly
	normalized.MinimumWeightChangeBPS = 0
	normalized.WeightingMethod = QuantWeightEqual
	normalized.VolatilityLookbackDays = 20
	normalized.TrendFilterDays = 0
	normalized.WeakMarketExposureBPS = 10_000

	baseline := normalized
	buffer := normalized
	buffer.ExitRank = 8
	lowTurnover := buffer
	lowTurnover.RebalanceFrequency = QuantRebalanceMonthly
	lowTurnover.MinimumWeightChangeBPS = 250
	riskControlled := lowTurnover
	riskControlled.WeightingMethod = QuantWeightInverseVolatility
	riskControlled.VolatilityLookbackDays = 20
	riskControlled.TrendFilterDays = 120
	riskControlled.WeakMarketExposureBPS = 5_000

	return []quantRiskVariantDefinition{
		{Key: "A", Name: "周度基线", Description: "周度 Top 5 等权，跌出 Top 5 后退出。", Request: baseline},
		{Key: "B", Name: "排名缓冲", Description: "周度 Top 5 进入，跌出 Top 8 后退出。", Request: buffer},
		{Key: "C", Name: "低换手", Description: "月度调仓并保留排名缓冲，权重偏差达到 2.5% 才交易。", Request: lowTurnover},
		{Key: "D", Name: "风险控制", Description: "低换手基础上按波动率倒数加权，弱势市场将风险敞口降至 50%。", Request: riskControlled},
	}, nil
}

func (s *QuantResearchStore) CreateRiskV1Experiment(ctx context.Context, base QuantBacktestRequest) (*QuantExperimentView, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("quant research store is unavailable")
	}
	definitions, err := RiskV1Requests(base)
	if err != nil {
		return nil, err
	}
	base = definitions[0].Request
	payload, err := json.Marshal(base)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(append([]byte(QuantExperimentPresetRiskV1+"|"+QuantBacktestImplementation+"|"), payload...))
	idempotencyKey := hex.EncodeToString(sum[:])

	var experiment database.QuantBacktestExperiment
	if err := s.db.WithContext(ctx).Where("idempotency_key = ?", idempotencyKey).First(&experiment).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		start, _ := time.Parse("2006-01-02", base.StartDate)
		end, _ := time.Parse("2006-01-02", base.EndDate)
		id, idErr := randomHex(16)
		if idErr != nil {
			return nil, idErr
		}
		experiment = database.QuantBacktestExperiment{
			ID:                 id,
			IdempotencyKey:     idempotencyKey,
			Preset:             QuantExperimentPresetRiskV1,
			UniverseVersion:    base.UniverseVersion,
			SignalMode:         base.SignalMode,
			StartDate:          start,
			EndDate:            end,
			BaseParametersJSON: payload,
		}
		if err := s.db.WithContext(ctx).Create(&experiment).Error; err != nil {
			return nil, err
		}
	}

	for _, definition := range definitions {
		job, _, createErr := s.CreateBacktestJob(ctx, definition.Request)
		if createErr != nil {
			return nil, createErr
		}
		link := database.QuantBacktestExperimentJob{ExperimentID: experiment.ID, JobID: job.ID, VariantKey: definition.Key}
		if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&link).Error; err != nil {
			return nil, err
		}
	}
	return s.GetBacktestExperiment(ctx, experiment.ID)
}

func (s *QuantResearchStore) ResolveBacktestDateRange(ctx context.Context, mode string) (string, string, error) {
	if s == nil || s.db == nil {
		return "", "", fmt.Errorf("quant research store is unavailable")
	}
	if strings.TrimSpace(mode) == "" {
		mode = QuantSignalModeHistoryProxy
	}
	type dateRange struct {
		First *time.Time
		Last  *time.Time
	}
	var signals dateRange
	if err := s.db.WithContext(ctx).Raw(`SELECT MIN(signal_date) AS first, MAX(signal_date) AS last FROM quant_signal_history WHERE mode = ?`, mode).Scan(&signals).Error; err != nil {
		return "", "", err
	}
	var latestMarket *time.Time
	if err := s.db.WithContext(ctx).Model(&database.QuantMarketBar{}).Select("MAX(date)").Scan(&latestMarket).Error; err != nil {
		return "", "", err
	}
	if signals.First == nil || signals.Last == nil || latestMarket == nil {
		return "", "", fmt.Errorf("quant signal or market history is empty")
	}
	end := *signals.Last
	if latestMarket.Before(end) {
		end = *latestMarket
	}
	return signals.First.Format("2006-01-02"), end.Format("2006-01-02"), nil
}

func (s *QuantResearchStore) GetBacktestExperiment(ctx context.Context, experimentID string) (*QuantExperimentView, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("quant research store is unavailable")
	}
	var experiment database.QuantBacktestExperiment
	if err := s.db.WithContext(ctx).First(&experiment, "id = ?", strings.TrimSpace(experimentID)).Error; err != nil {
		return nil, err
	}
	var links []database.QuantBacktestExperimentJob
	if err := s.db.WithContext(ctx).Where("experiment_id = ?", experiment.ID).Order("variant_key").Find(&links).Error; err != nil {
		return nil, err
	}
	definitions, _ := RiskV1Requests(decodeBacktestRequest(experiment.BaseParametersJSON))
	definitionByKey := make(map[string]quantRiskVariantDefinition, len(definitions))
	for _, definition := range definitions {
		definitionByKey[definition.Key] = definition
	}
	variants := make([]QuantExperimentVariant, 0, len(links))
	for _, link := range links {
		var job database.QuantBacktestJob
		if err := s.db.WithContext(ctx).First(&job, "id = ?", link.JobID).Error; err != nil {
			return nil, err
		}
		definition := definitionByKey[link.VariantKey]
		variants = append(variants, QuantExperimentVariant{Key: link.VariantKey, Name: definition.Name, Description: definition.Description, Job: job})
	}
	sort.Slice(variants, func(i, j int) bool { return variants[i].Key < variants[j].Key })
	applyRiskGates(variants)
	return &QuantExperimentView{
		ID:              experiment.ID,
		Preset:          experiment.Preset,
		Status:          experimentStatus(variants),
		UniverseVersion: experiment.UniverseVersion,
		SignalMode:      experiment.SignalMode,
		StartDate:       experiment.StartDate.Format("2006-01-02"),
		EndDate:         experiment.EndDate.Format("2006-01-02"),
		BaseParameters:  decodeBacktestRequest(experiment.BaseParametersJSON),
		Variants:        variants,
		CreatedAt:       experiment.CreatedAt,
		UpdatedAt:       experiment.UpdatedAt,
	}, nil
}

func (s *QuantResearchStore) ListBacktestExperiments(ctx context.Context, limit int) ([]QuantExperimentView, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("quant research store is unavailable")
	}
	if limit <= 0 || limit > 20 {
		limit = 6
	}
	var experiments []database.QuantBacktestExperiment
	if err := s.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Find(&experiments).Error; err != nil {
		return nil, err
	}
	result := make([]QuantExperimentView, 0, len(experiments))
	for _, experiment := range experiments {
		view, err := s.GetBacktestExperiment(ctx, experiment.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, *view)
	}
	return result, nil
}

func decodeBacktestRequest(payload json.RawMessage) QuantBacktestRequest {
	var request QuantBacktestRequest
	_ = json.Unmarshal(payload, &request)
	return request
}

func experimentStatus(variants []QuantExperimentVariant) string {
	if len(variants) == 0 {
		return "empty"
	}
	completed := 0
	hasRunning := false
	hasFailed := false
	for _, variant := range variants {
		switch variant.Job.Status {
		case "completed":
			completed++
		case "queued", "running":
			hasRunning = true
		case "failed", "queue_failed":
			hasFailed = true
		}
	}
	if completed == len(variants) {
		return "completed"
	}
	if hasRunning {
		return "running"
	}
	if hasFailed {
		return "failed"
	}
	return "pending"
}

func applyRiskGates(variants []QuantExperimentVariant) {
	if len(variants) == 0 || variants[0].Key != "A" || variants[0].Job.Status != "completed" {
		return
	}
	baseline, ok := decodeNormalizedResult(variants[0].Job.NormalizedJSON)
	if !ok {
		return
	}
	for index := 1; index < len(variants); index++ {
		candidate, candidateOK := decodeNormalizedResult(variants[index].Job.NormalizedJSON)
		if !candidateOK || variants[index].Job.Status != "completed" {
			continue
		}
		assessment := QuantRiskGateAssessment{
			SharpeImproved:   candidate.Summary.Sharpe > baseline.Summary.Sharpe,
			DrawdownImproved: candidate.Summary.MaxDrawdownPct < baseline.Summary.MaxDrawdownPct,
			CAGRPreserved:    candidate.Summary.CAGRPct >= baseline.Summary.CAGRPct-1,
		}
		baselineYears := make(map[int]QuantBacktestAnnualReturn, len(baseline.AnnualReturns))
		for _, annual := range baseline.AnnualReturns {
			if annual.TradingDays >= 120 {
				baselineYears[annual.Year] = annual
			}
		}
		for _, annual := range candidate.AnnualReturns {
			baselineAnnual, exists := baselineYears[annual.Year]
			if !exists || annual.TradingDays < 120 {
				continue
			}
			assessment.ComparableYears++
			if annual.StrategyReturnPct > baselineAnnual.StrategyReturnPct {
				assessment.AnnualWins++
			}
		}
		assessment.Qualified = assessment.SharpeImproved && assessment.DrawdownImproved && assessment.CAGRPreserved && assessment.AnnualWins >= 3
		variants[index].RiskGate = &assessment
	}
}

func decodeNormalizedResult(payload json.RawMessage) (QuantNormalizedResult, bool) {
	var result QuantNormalizedResult
	if len(payload) == 0 || json.Unmarshal(payload, &result) != nil || result.SchemaVersion == "" {
		return QuantNormalizedResult{}, false
	}
	return result, true
}
