package service

import (
	"context"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/shopspring/decimal"
)

func TestOfficialNAVSyncServiceNextRunAt(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo)

	beforeRun := time.Date(2026, 3, 31, 22, 15, 0, 0, officialNAVSyncLocation)
	next := service.nextRunAt(beforeRun)
	if next.Format(time.RFC3339) != "2026-03-31T23:00:00+08:00" {
		t.Fatalf("next run before cutoff = %s", next.Format(time.RFC3339))
	}

	afterRun := time.Date(2026, 3, 31, 23, 15, 0, 0, officialNAVSyncLocation)
	next = service.nextRunAt(afterRun)
	if next.Format(time.RFC3339) != "2026-04-01T23:00:00+08:00" {
		t.Fatalf("next run after cutoff = %s", next.Format(time.RFC3339))
	}
}

func TestOfficialNAVSyncServiceShouldSyncImmediatelyWhenCurrentTradingDayMissing(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo)

	if err := userRepo.SaveFundHolding(context.Background(), &domain.UserFundHolding{
		ID:       "ufh_1",
		UserID:   "user-1",
		FundID:   "005827",
		Amount:   decimal.RequireFromString("1000"),
		AsOfDate: "2026-03-31",
	}); err != nil {
		t.Fatalf("SaveFundHolding() error = %v", err)
	}

	service.now = func() time.Time {
		return time.Date(2026, 3, 31, 23, 15, 0, 0, officialNAVSyncLocation)
	}

	if !service.shouldSyncImmediately(context.Background()) {
		t.Fatalf("shouldSyncImmediately() = false, want true")
	}
}

func TestOfficialNAVSyncServiceSkipsImmediateSyncWhenAlreadyUpToDate(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo)

	if err := userRepo.SaveFundHolding(context.Background(), &domain.UserFundHolding{
		ID:       "ufh_1",
		UserID:   "user-1",
		FundID:   "005827",
		Amount:   decimal.RequireFromString("1000"),
		AsOfDate: "2026-03-31",
	}); err != nil {
		t.Fatalf("SaveFundHolding() error = %v", err)
	}
	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        "2026-03-31",
		NetAssetVal: decimal.RequireFromString("1.8000"),
		AccumVal:    decimal.RequireFromString("2.1000"),
		DailyReturn: decimal.RequireFromString("1.2345"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}

	service.now = func() time.Time {
		return time.Date(2026, 3, 31, 23, 15, 0, 0, officialNAVSyncLocation)
	}

	if service.shouldSyncImmediately(context.Background()) {
		t.Fatalf("shouldSyncImmediately() = true, want false")
	}
}

func TestOfficialNAVSyncServiceSkipsImmediateSyncOnWeekend(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo)

	service.now = func() time.Time {
		return time.Date(2026, 4, 4, 23, 15, 0, 0, officialNAVSyncLocation)
	}

	if service.shouldSyncImmediately(context.Background()) {
		t.Fatalf("shouldSyncImmediately() = true, want false on weekend")
	}
}

func TestOfficialNAVSyncServiceBackfillsHoldingConfirmationWhenHistoryExists(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo)

	if err := userRepo.SaveFundHolding(context.Background(), &domain.UserFundHolding{
		ID:       "ufh_1",
		UserID:   "user-1",
		FundID:   "005827",
		Amount:   decimal.RequireFromString("1000"),
		AsOfDate: "2026-03-31",
	}); err != nil {
		t.Fatalf("SaveFundHolding() error = %v", err)
	}
	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        "2026-03-31",
		NetAssetVal: decimal.RequireFromString("2.0000"),
		AccumVal:    decimal.RequireFromString("2.0000"),
		DailyReturn: decimal.RequireFromString("1.0000"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}

	backfilledCount, err := service.backfillHoldingConfirmations(context.Background())
	if err != nil {
		t.Fatalf("backfillHoldingConfirmations() error = %v", err)
	}
	if backfilledCount != 1 {
		t.Fatalf("backfilledCount = %d, want 1", backfilledCount)
	}

	holdings, err := userRepo.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings) != 1 {
		t.Fatalf("holdings len = %d, want 1", len(holdings))
	}
	if holdings[0].ConfirmedNavDate != "2026-03-31" {
		t.Fatalf("confirmed nav date = %s, want 2026-03-31", holdings[0].ConfirmedNavDate)
	}
	if holdings[0].ConfirmedNav.String() != "2" {
		t.Fatalf("confirmed nav = %s, want 2", holdings[0].ConfirmedNav.String())
	}
	if holdings[0].Shares.String() != "500" {
		t.Fatalf("shares = %s, want 500", holdings[0].Shares.String())
	}
}
