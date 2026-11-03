package repository

import (
	"context"
	"fmt"
	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"time"
)

func validateFundHistoryRange(fundID, start, end string, rows []domain.FundHistory) error {
	if fundID == "" || start > end || len(rows) == 0 {
		return fmt.Errorf("empty or invalid history range")
	}
	if _, err := time.Parse("2006-01-02", start); err != nil {
		return err
	}
	if _, err := time.Parse("2006-01-02", end); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		if _, err := time.Parse("2006-01-02", row.Date); err != nil {
			return err
		}
		if row.FundID != fundID || row.Date < start || row.Date > end || !row.NetAssetVal.GreaterThan(decimal.Zero) || seen[row.Date] {
			return fmt.Errorf("invalid official history record")
		}
		seen[row.Date] = true
	}
	return nil
}

func (r *MemoryFundRepository) ReplaceFundHistoryRange(ctx context.Context, fundID, start, end string, rows []domain.FundHistory) error {
	if err := validateFundHistoryRange(fundID, start, end, rows); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	retained := make([]domain.FundHistory, 0, len(r.history[fundID])+len(rows))
	for _, row := range r.history[fundID] {
		if row.Date < start || row.Date > end {
			retained = append(retained, row)
		}
	}
	r.history[fundID] = append(retained, rows...)
	return nil
}

func (r *PostgresFundRepository) ReplaceFundHistoryRange(ctx context.Context, fundID, start, end string, rows []domain.FundHistory) error {
	if err := validateFundHistoryRange(fundID, start, end, rows); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		if err := db.Where("fund_id = ? AND date >= ? AND date <= ?", fundID, start, end).Delete(&database.FundHistory{}).Error; err != nil {
			return err
		}
		repo := NewPostgresFundRepository(db)
		for index := range rows {
			if err := repo.SaveFundHistory(ctx, &rows[index]); err != nil {
				return err
			}
		}
		return nil
	})
}
