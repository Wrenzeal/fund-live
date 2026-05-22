package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FundAnalysisSnapshotStore struct {
	db *gorm.DB
}

func NewFundAnalysisSnapshotStore(db *gorm.DB) *FundAnalysisSnapshotStore {
	if db == nil {
		return nil
	}
	return &FundAnalysisSnapshotStore{db: db}
}

func (s *FundAnalysisSnapshotStore) Save(ctx context.Context, fundID string, analysis *domain.FundAnalysis, generatedAt time.Time) error {
	if s == nil || s.db == nil || strings.TrimSpace(fundID) == "" || analysis == nil {
		return nil
	}
	if generatedAt.IsZero() {
		generatedAt = time.Now()
	}

	payload, err := json.Marshal(analysis)
	if err != nil {
		return fmt.Errorf("marshal analysis snapshot: %w", err)
	}

	record := &database.FundAnalysisSnapshot{
		FundID:              strings.TrimSpace(fundID),
		AnalysisType:        analysis.AnalysisType,
		AnalysisBasis:       analysis.AnalysisBasis,
		TotalScore:          analysis.TotalScore,
		Confidence:          analysis.Confidence,
		RiskLevel:           analysis.RiskLevel,
		IncreasePercent:     analysis.IncreasePercent,
		HoldPercent:         analysis.HoldPercent,
		DecreasePercent:     analysis.DecreasePercent,
		LatestHoldingPeriod: analysis.LatestHoldingPeriod,
		Summary:             analysis.Summary,
		GeneratedAt:         generatedAt,
		AnalysisJSON:        payload,
	}

	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "fund_id"}},
		UpdateAll: true,
	}).Create(record).Error
}

func (s *FundAnalysisSnapshotStore) Get(ctx context.Context, fundID string) (*domain.FundAnalysis, time.Time, error) {
	if s == nil || s.db == nil || strings.TrimSpace(fundID) == "" {
		return nil, time.Time{}, nil
	}
	var record database.FundAnalysisSnapshot
	if err := s.db.WithContext(ctx).First(&record, "fund_id = ?", strings.TrimSpace(fundID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, time.Time{}, nil
		}
		return nil, time.Time{}, err
	}
	analysis, err := decodeAnalysisSnapshot(record.AnalysisJSON)
	if err != nil {
		return nil, time.Time{}, err
	}
	return analysis, record.GeneratedAt, nil
}

func (s *FundAnalysisSnapshotStore) GetByFundIDs(ctx context.Context, fundIDs []string, now time.Time) (map[string]*domain.FundAnalysis, error) {
	result := make(map[string]*domain.FundAnalysis)
	if s == nil || s.db == nil || len(fundIDs) == 0 {
		return result, nil
	}

	uniqueIDs := uniqueSnapshotFundIDs(fundIDs)
	if len(uniqueIDs) == 0 {
		return result, nil
	}

	var records []database.FundAnalysisSnapshot
	if err := s.db.WithContext(ctx).Where("fund_id IN ?", uniqueIDs).Find(&records).Error; err != nil {
		return nil, err
	}
	for _, record := range records {
		analysis, err := decodeAnalysisSnapshot(record.AnalysisJSON)
		if err != nil {
			continue
		}
		if !IsFundAnalysisSnapshotFresh(analysis, record.GeneratedAt, now) {
			continue
		}
		analysis.AIExplanation = MarkAIExplanationSnapshotHit(analysis.AIExplanation)
		result[record.FundID] = analysis
	}
	return result, nil
}

func (s *FundAnalysisSnapshotStore) ListRankings(ctx context.Context, limit int, now time.Time) (increase, watch, risk []database.FundAnalysisSnapshot, err error) {
	if s == nil || s.db == nil {
		return nil, nil, nil, nil
	}
	if limit <= 0 {
		limit = 12
	}
	if increase, err = s.queryRankings(ctx, "increase_percent DESC, total_score DESC", limit, now); err != nil {
		return nil, nil, nil, err
	}
	if watch, err = s.queryRankings(ctx, "hold_percent DESC, total_score DESC", limit, now); err != nil {
		return nil, nil, nil, err
	}
	if risk, err = s.queryRankings(ctx, "decrease_percent DESC, total_score ASC", limit, now); err != nil {
		return nil, nil, nil, err
	}
	return increase, watch, risk, nil
}

func (s *FundAnalysisSnapshotStore) queryRankings(ctx context.Context, order string, limit int, now time.Time) ([]database.FundAnalysisSnapshot, error) {
	queryLimit := limit * 4
	if queryLimit < limit {
		queryLimit = limit
	}
	if queryLimit < 24 {
		queryLimit = 24
	}
	records := make([]database.FundAnalysisSnapshot, 0, queryLimit)
	if err := s.db.WithContext(ctx).
		Order(order).
		Order("generated_at DESC").
		Limit(queryLimit).
		Find(&records).Error; err != nil {
		return nil, err
	}
	filtered := make([]database.FundAnalysisSnapshot, 0, limit)
	for _, record := range records {
		analysis, err := decodeAnalysisSnapshot(record.AnalysisJSON)
		if err != nil || !IsFundAnalysisSnapshotFresh(analysis, record.GeneratedAt, now) {
			continue
		}
		filtered = append(filtered, record)
		if len(filtered) >= limit {
			break
		}
	}
	return filtered, nil
}

func decodeAnalysisSnapshot(payload []byte) (*domain.FundAnalysis, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	var analysis domain.FundAnalysis
	if err := json.Unmarshal(payload, &analysis); err != nil {
		return nil, fmt.Errorf("decode analysis snapshot: %w", err)
	}
	return &analysis, nil
}

func uniqueSnapshotFundIDs(fundIDs []string) []string {
	seen := make(map[string]struct{}, len(fundIDs))
	result := make([]string, 0, len(fundIDs))
	for _, fundID := range fundIDs {
		fundID = strings.TrimSpace(fundID)
		if fundID == "" {
			continue
		}
		if _, ok := seen[fundID]; ok {
			continue
		}
		seen[fundID] = struct{}{}
		result = append(result, fundID)
	}
	sort.Strings(result)
	return result
}
