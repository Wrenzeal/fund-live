// Package repository contains data persistence implementations.
package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PostgresFundRepository is a PostgreSQL implementation of FundRepository.
type PostgresFundRepository struct {
	db *gorm.DB
}

// NewPostgresFundRepository creates a new PostgreSQL fund repository.
func NewPostgresFundRepository(db *gorm.DB) *PostgresFundRepository {
	return &PostgresFundRepository{db: db}
}

// GetFundByID retrieves a fund by its ID.
func (r *PostgresFundRepository) GetFundByID(ctx context.Context, fundID string) (*domain.Fund, error) {
	var dbFund database.Fund
	result := r.db.WithContext(ctx).First(&dbFund, "id = ?", fundID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get fund: %w", result.Error)
	}

	return r.toDomainFund(&dbFund), nil
}

// GetFundsByIDs retrieves multiple funds keyed by fund ID.
func (r *PostgresFundRepository) GetFundsByIDs(ctx context.Context, fundIDs []string) (map[string]*domain.Fund, error) {
	resultMap := make(map[string]*domain.Fund)
	if len(fundIDs) == 0 {
		return resultMap, nil
	}

	uniqueIDs := uniqueStrings(fundIDs)
	var dbFunds []database.Fund
	result := r.db.WithContext(ctx).
		Where("id IN ?", uniqueIDs).
		Find(&dbFunds)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get funds by ids: %w", result.Error)
	}

	for i := range dbFunds {
		fund := r.toDomainFund(&dbFunds[i])
		resultMap[fund.ID] = fund
	}

	return resultMap, nil
}

// SearchFunds searches for funds by name or code.
func (r *PostgresFundRepository) SearchFunds(ctx context.Context, query string, limit int) ([]*domain.Fund, error) {
	trimmedQuery := strings.TrimSpace(query)
	normalizedQuery := normalizeSearchQuery(trimmedQuery)
	if normalizedQuery == "" || limit <= 0 {
		return []*domain.Fund{}, nil
	}

	candidateLimit := searchCandidateLimit(limit)
	candidates := make([]*domain.Fund, 0, candidateLimit)
	seen := make(map[string]struct{}, candidateLimit)
	appendCandidates := func(records []database.Fund) {
		for i := range records {
			fund := r.toDomainFund(&records[i])
			if _, exists := seen[fund.ID]; exists {
				continue
			}
			seen[fund.ID] = struct{}{}
			candidates = append(candidates, fund)
		}
	}

	var highPriority []database.Fund
	prefixPattern := trimmedQuery + "%"
	result := r.db.WithContext(ctx).
		Where("id = ? OR id LIKE ? OR name ILIKE ? OR manager ILIKE ?",
			trimmedQuery, prefixPattern, prefixPattern, prefixPattern).
		Limit(candidateLimit).
		Find(&highPriority)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to search funds with prefix strategy: %w", result.Error)
	}
	appendCandidates(highPriority)

	if len(candidates) < limit {
		var fuzzy []database.Fund
		fuzzyPattern := "%" + trimmedQuery + "%"
		result = r.db.WithContext(ctx).
			Where("id LIKE ? OR name ILIKE ? OR manager ILIKE ?",
				fuzzyPattern, fuzzyPattern, fuzzyPattern).
			Limit(candidateLimit).
			Find(&fuzzy)
		if result.Error != nil {
			return nil, fmt.Errorf("failed to search funds with fuzzy strategy: %w", result.Error)
		}
		appendCandidates(fuzzy)
	}

	return rankAndLimitFunds(candidates, normalizedQuery, limit), nil
}

// GetFundHoldings retrieves the top holdings for a fund.
func (r *PostgresFundRepository) GetFundHoldings(ctx context.Context, fundID string) ([]domain.StockHolding, error) {
	var dbHoldings []database.StockHolding
	result := r.db.WithContext(ctx).
		Where("fund_id = ?", fundID).
		Order("holding_ratio DESC").
		Find(&dbHoldings)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get holdings: %w", result.Error)
	}

	holdings := make([]domain.StockHolding, len(dbHoldings))
	for i, h := range dbHoldings {
		holdings[i] = r.toDomainHolding(&h)
	}

	return holdings, nil
}

// ListFundIDsWithHoldings returns fund IDs that currently have persisted holdings.
func (r *PostgresFundRepository) ListFundIDsWithHoldings(ctx context.Context) ([]string, error) {
	var fundIDs []string
	if err := r.db.WithContext(ctx).
		Model(&database.StockHolding{}).
		Distinct("fund_id").
		Order("fund_id ASC").
		Pluck("fund_id", &fundIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to list fund ids with holdings: %w", err)
	}
	return fundIDs, nil
}

// SaveFund saves or updates a fund (upsert).
func (r *PostgresFundRepository) SaveFund(ctx context.Context, fund *domain.Fund) error {
	dbFund := r.toDBFund(fund)

	// Upsert: insert or update on conflict
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(&dbFund)

	if result.Error != nil {
		return fmt.Errorf("failed to save fund: %w", result.Error)
	}

	return nil
}

// SaveHoldings saves the holdings for a fund.
// This replaces all existing holdings for the fund.
func (r *PostgresFundRepository) SaveHoldings(ctx context.Context, fundID string, holdings []domain.StockHolding) error {
	if holdings == nil {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing holdings for this fund
		if err := tx.Where("fund_id = ?", fundID).Delete(&database.StockHolding{}).Error; err != nil {
			return fmt.Errorf("failed to delete old holdings: %w", err)
		}

		// Insert new holdings
		if len(holdings) > 0 {
			dbHoldings := make([]database.StockHolding, len(holdings))
			for i, h := range holdings {
				dbHoldings[i] = r.toDBHolding(fundID, &h)
			}

			if err := tx.Create(&dbHoldings).Error; err != nil {
				return fmt.Errorf("failed to insert holdings: %w", err)
			}
		}

		return nil
	})
}

// SaveTimeSeriesPoint saves a single time series data point to the database.
func (r *PostgresFundRepository) SaveTimeSeriesPoint(ctx context.Context, point *domain.TimeSeriesPoint, fundID string) error {
	dbPoint := database.FundTimeSeries{
		FundID:        fundID,
		Date:          point.Timestamp.Truncate(24 * time.Hour),
		Time:          point.Timestamp,
		ChangePercent: point.ChangePercent,
		EstimateNav:   point.EstimateNav,
	}

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "fund_id"},
			{Name: "time"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"date",
			"change_percent",
			"estimate_nav",
		}),
	}).Create(&dbPoint)
	if result.Error != nil {
		return fmt.Errorf("failed to save time series point: %w", result.Error)
	}
	return nil
}

// ReplaceTimeSeriesByDate replaces all time series points for a fund on a specific date.
func (r *PostgresFundRepository) ReplaceTimeSeriesByDate(ctx context.Context, fundID string, date time.Time, points []domain.TimeSeriesPoint) error {
	startOfDay := date.Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("fund_id = ? AND time >= ? AND time < ?", fundID, startOfDay, endOfDay).Delete(&database.FundTimeSeries{}).Error; err != nil {
			return fmt.Errorf("failed to delete old time series: %w", err)
		}

		if len(points) == 0 {
			return nil
		}

		dbPoints := make([]database.FundTimeSeries, 0, len(points))
		for _, point := range points {
			dbPoints = append(dbPoints, database.FundTimeSeries{
				FundID:        fundID,
				Date:          point.Timestamp.Truncate(24 * time.Hour),
				Time:          point.Timestamp,
				ChangePercent: point.ChangePercent,
				EstimateNav:   point.EstimateNav,
			})
		}

		if err := tx.Create(&dbPoints).Error; err != nil {
			return fmt.Errorf("failed to insert replaced time series: %w", err)
		}
		return nil
	})
}

// GetTimeSeriesByDate retrieves all time series points for a fund on a specific date.
func (r *PostgresFundRepository) GetTimeSeriesByDate(ctx context.Context, fundID string, date time.Time) ([]domain.TimeSeriesPoint, error) {
	var dbPoints []database.FundTimeSeries
	startOfDay := date.Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)

	result := r.db.WithContext(ctx).
		Where("fund_id = ? AND time >= ? AND time < ?", fundID, startOfDay, endOfDay).
		Order("time ASC").
		Find(&dbPoints)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get time series: %w", result.Error)
	}

	points := make([]domain.TimeSeriesPoint, len(dbPoints))
	for i, p := range dbPoints {
		points[i] = domain.TimeSeriesPoint{
			Timestamp:     p.Time,
			ChangePercent: p.ChangePercent,
			EstimateNav:   p.EstimateNav,
		}
	}
	return points, nil
}

// SaveFundHistory saves or updates a daily official NAV snapshot.
func (r *PostgresFundRepository) SaveFundHistory(ctx context.Context, history *domain.FundHistory) error {
	historyDate, err := time.Parse("2006-01-02", history.Date)
	if err != nil {
		return fmt.Errorf("failed to parse fund history date: %w", err)
	}

	record := &database.FundHistory{
		FundID:      history.FundID,
		Date:        historyDate,
		NetAssetVal: history.NetAssetVal,
		AccumVal:    history.AccumVal,
		DailyReturn: history.DailyReturn,
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "fund_id"},
			{Name: "date"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"net_asset_val",
			"accum_val",
			"daily_return",
		}),
	}).Create(record).Error; err != nil {
		return fmt.Errorf("failed to upsert fund history: %w", err)
	}
	return nil
}

// GetLatestFundHistory retrieves the latest official NAV snapshot for a fund.
func (r *PostgresFundRepository) GetLatestFundHistory(ctx context.Context, fundID string) (*domain.FundHistory, error) {
	var record database.FundHistory
	result := r.db.WithContext(ctx).
		Where("fund_id = ?", fundID).
		Order("date DESC").
		First(&record)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest fund history: %w", result.Error)
	}

	return &domain.FundHistory{
		FundID:      record.FundID,
		Date:        record.Date.Format("2006-01-02"),
		NetAssetVal: record.NetAssetVal,
		AccumVal:    record.AccumVal,
		DailyReturn: record.DailyReturn,
		CreatedAt:   record.CreatedAt,
	}, nil
}

// GetLatestFundHistoriesByFundIDs retrieves the latest official NAV snapshots for multiple funds.
func (r *PostgresFundRepository) GetLatestFundHistoriesByFundIDs(ctx context.Context, fundIDs []string) (map[string]*domain.FundHistory, error) {
	resultMap := make(map[string]*domain.FundHistory)
	if len(fundIDs) == 0 {
		return resultMap, nil
	}

	uniqueIDs := uniqueStrings(fundIDs)
	var records []database.FundHistory

	latestDateSubquery := r.db.WithContext(ctx).
		Model(&database.FundHistory{}).
		Select("fund_id, MAX(date) AS max_date").
		Where("fund_id IN ?", uniqueIDs).
		Group("fund_id")

	result := r.db.WithContext(ctx).
		Model(&database.FundHistory{}).
		Joins(
			"JOIN (?) AS latest ON latest.fund_id = fund_history.fund_id AND latest.max_date = fund_history.date",
			latestDateSubquery,
		).
		Find(&records)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get latest fund histories by ids: %w", result.Error)
	}

	for _, record := range records {
		resultMap[record.FundID] = &domain.FundHistory{
			FundID:      record.FundID,
			Date:        record.Date.Format("2006-01-02"),
			NetAssetVal: record.NetAssetVal,
			AccumVal:    record.AccumVal,
			DailyReturn: record.DailyReturn,
			CreatedAt:   record.CreatedAt,
		}
	}

	return resultMap, nil
}

// --- Conversion helpers ---

func (r *PostgresFundRepository) toDomainFund(dbFund *database.Fund) *domain.Fund {
	return &domain.Fund{
		ID:          dbFund.ID,
		Name:        dbFund.Name,
		Type:        dbFund.Type,
		Manager:     dbFund.Manager,
		Company:     dbFund.Company,
		NetAssetVal: dbFund.NetAssetVal,
		TotalScale:  dbFund.TotalScale,
		UpdatedAt:   dbFund.UpdatedAt,
	}
}

func (r *PostgresFundRepository) toDBFund(fund *domain.Fund) *database.Fund {
	return &database.Fund{
		ID:          fund.ID,
		Name:        fund.Name,
		Type:        fund.Type,
		Manager:     fund.Manager,
		Company:     fund.Company,
		NetAssetVal: fund.NetAssetVal,
		TotalScale:  fund.TotalScale,
	}
}

func (r *PostgresFundRepository) toDomainHolding(dbHolding *database.StockHolding) domain.StockHolding {
	return domain.StockHolding{
		StockCode:       dbHolding.StockCode,
		StockName:       dbHolding.StockName,
		Exchange:        domain.Exchange(dbHolding.Exchange),
		HoldingRatio:    dbHolding.HoldingRatio,
		HoldingShares:   dbHolding.HoldingShares,
		MarketValue:     dbHolding.MarketValue,
		ReportingPeriod: dbHolding.ReportingPeriod,
	}
}

func (r *PostgresFundRepository) toDBHolding(fundID string, h *domain.StockHolding) database.StockHolding {
	return database.StockHolding{
		FundID:          fundID,
		StockCode:       h.StockCode,
		StockName:       h.StockName,
		Exchange:        string(h.Exchange),
		HoldingRatio:    h.HoldingRatio,
		HoldingShares:   h.HoldingShares,
		MarketValue:     h.MarketValue,
		ReportingPeriod: h.ReportingPeriod,
	}
}

func uniqueStrings(values []string) []string {
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
