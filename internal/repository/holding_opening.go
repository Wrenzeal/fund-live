package repository

import (
	"context"
	"fmt"
	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"gorm.io/gorm"
	"time"
)

func (r *MemoryUserRepository) CreateFundHoldingWithTransaction(ctx context.Context, holding *domain.UserFundHolding, transaction *domain.UserFundHoldingTransaction) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if holding == nil || transaction == nil || holding.UserID != transaction.UserID || holding.ID != transaction.HoldingID {
		return fmt.Errorf("position and activity must belong together")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fundHoldings[holding.UserID] == nil {
		r.fundHoldings[holding.UserID] = make(map[string]domain.UserFundHolding)
	}
	if transaction.CreatedAt.IsZero() {
		transaction.CreatedAt = time.Now()
	}
	r.fundHoldings[holding.UserID][holding.ID] = *holding
	r.holdingTxs[transaction.UserID] = append(r.holdingTxs[transaction.UserID], copyFundHoldingTransaction(*transaction))
	return nil
}

func (r *PostgresUserRepository) CreateFundHoldingWithTransaction(ctx context.Context, holding *domain.UserFundHolding, transaction *domain.UserFundHoldingTransaction) error {
	if holding == nil || transaction == nil || holding.UserID != transaction.UserID || holding.ID != transaction.HoldingID {
		return fmt.Errorf("position and activity must belong together")
	}
	return r.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		repo := NewPostgresUserRepository(db)
		if err := repo.SaveFundHolding(ctx, holding); err != nil {
			return err
		}
		return repo.SaveFundHoldingTransaction(ctx, transaction)
	})
}

func (r *MemoryUserRepository) ConfirmFundHoldingIfUnchanged(ctx context.Context, original, confirmed *domain.UserFundHolding) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, found := r.fundHoldings[original.UserID][original.ID]
	if !found || current.EntryType == "opening" || current.ManualConfirmation || !current.Amount.Equal(original.Amount) || !current.UpdatedAt.Equal(original.UpdatedAt) {
		return false, nil
	}
	current.Shares = confirmed.Shares
	current.ConfirmedNav = confirmed.ConfirmedNav
	current.ConfirmedNavDate = confirmed.ConfirmedNavDate
	current.UpdatedAt = confirmed.UpdatedAt
	r.fundHoldings[original.UserID][original.ID] = current
	return true, nil
}

func (r *PostgresUserRepository) ConfirmFundHoldingIfUnchanged(ctx context.Context, original, confirmed *domain.UserFundHolding) (bool, error) {
	date, err := time.Parse("2006-01-02", confirmed.ConfirmedNavDate)
	if err != nil {
		return false, err
	}
	result := r.db.WithContext(ctx).Model(&database.UserFundHolding{}).Where("id = ? AND user_id = ? AND entry_type <> 'opening' AND manual_confirmation = false AND amount = ? AND updated_at = ?", original.ID, original.UserID, original.Amount, original.UpdatedAt).Updates(map[string]interface{}{"shares": confirmed.Shares, "confirmed_nav": confirmed.ConfirmedNav, "confirmed_nav_date": date, "updated_at": confirmed.UpdatedAt})
	return result.RowsAffected > 0, result.Error
}
