package service

import (
	"context"
	"fmt"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"gorm.io/gorm"
)

type historicalHoldingsStore interface {
	LoadByPeriod(ctx context.Context, fundCode string, reportingPeriod string) ([]domain.StockHolding, error)
	SaveHistoryPeriods(ctx context.Context, fundCode string, holdingsByPeriod map[string][]domain.StockHolding, source string) error
}

type HistoricalHoldingsSnapshotStore struct {
	db *gorm.DB
}

func NewHistoricalHoldingsSnapshotStore(db *gorm.DB) *HistoricalHoldingsSnapshotStore {
	if db == nil {
		return nil
	}
	return &HistoricalHoldingsSnapshotStore{db: db}
}

func (s *HistoricalHoldingsSnapshotStore) LoadByPeriod(ctx context.Context, fundCode string, reportingPeriod string) ([]domain.StockHolding, error) {
	if s == nil || s.db == nil || fundCode == "" || reportingPeriod == "" {
		return nil, nil
	}

	var records []database.StockHoldingHistory
	if err := s.db.WithContext(ctx).
		Where("fund_id = ? AND reporting_period = ?", fundCode, reportingPeriod).
		Order("holding_ratio DESC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("load stock holding history: %w", err)
	}

	result := make([]domain.StockHolding, 0, len(records))
	for _, item := range records {
		result = append(result, domain.StockHolding{
			StockCode:       item.StockCode,
			StockName:       item.StockName,
			Exchange:        domain.Exchange(item.Exchange),
			HoldingRatio:    item.HoldingRatio,
			HoldingShares:   item.HoldingShares,
			MarketValue:     item.MarketValue,
			ReportingPeriod: item.ReportingPeriod,
		})
	}
	return result, nil
}

func (s *HistoricalHoldingsSnapshotStore) SaveHistoryPeriods(ctx context.Context, fundCode string, holdingsByPeriod map[string][]domain.StockHolding, source string) error {
	if s == nil || s.db == nil || fundCode == "" || len(holdingsByPeriod) == 0 {
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for period, holdings := range holdingsByPeriod {
			if period == "" {
				continue
			}
			if err := tx.Where("fund_id = ? AND reporting_period = ?", fundCode, period).Delete(&database.StockHoldingHistory{}).Error; err != nil {
				return fmt.Errorf("delete stock holding history period %s: %w", period, err)
			}
			if len(holdings) == 0 {
				continue
			}
			records := make([]database.StockHoldingHistory, 0, len(holdings))
			for _, holding := range holdings {
				records = append(records, database.StockHoldingHistory{
					FundID:          fundCode,
					StockCode:       holding.StockCode,
					StockName:       holding.StockName,
					Exchange:        string(holding.Exchange),
					HoldingRatio:    holding.HoldingRatio,
					HoldingShares:   holding.HoldingShares,
					MarketValue:     holding.MarketValue,
					ReportingPeriod: period,
					Source:          source,
				})
			}
			if err := tx.Create(&records).Error; err != nil {
				return fmt.Errorf("insert stock holding history period %s: %w", period, err)
			}
		}
		return nil
	})
}
