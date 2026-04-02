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
