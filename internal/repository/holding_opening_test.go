package repository

import (
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
	"strings"
	"testing"
	"time"
)

func TestMemoryConfirmationDoesNotOverwriteConcurrentManualCorrection(t *testing.T) {
	repo := NewMemoryUserRepository()
	original := &domain.UserFundHolding{ID: "lot", UserID: "owner", FundID: "014662", EntryType: "buy", AsOfDate: "2025-03-03", Amount: decimal.NewFromInt(1000), UpdatedAt: time.Now()}
	_ = repo.SaveFundHolding(t.Context(), original)
	manual := *original
	manual.Shares = decimal.NewFromInt(777)
	manual.ManualConfirmation = true
	manual.UpdatedAt = time.Now().Add(time.Second)
	_ = repo.SaveFundHolding(t.Context(), &manual)
	confirmed := *original
	confirmed.Shares = decimal.NewFromInt(500)
	confirmed.ConfirmedNav = decimal.NewFromInt(2)
	confirmed.ConfirmedNavDate = "2025-03-03"
	if changed, err := repo.ConfirmFundHoldingIfUnchanged(t.Context(), original, &confirmed); err != nil || changed {
		t.Fatalf("overwrote manual correction: %v %v", changed, err)
	}
	stored, _ := repo.GetFundHolding(t.Context(), "owner", "lot")
	if !stored.Shares.Equal(manual.Shares) {
		t.Fatal("manual shares changed")
	}
}

func TestPostgresOpeningCreationRollsBackIfActivityInsertFails(t *testing.T) {
	db, cleanup := openPostgresUserRepoTestDB(t)
	defer cleanup()
	repo := NewPostgresUserRepository(db)
	holding := &domain.UserFundHolding{ID: "opening-atomic", UserID: "owner", FundID: "014662", EntryType: "opening", Amount: decimal.NewFromInt(900), Shares: decimal.NewFromInt(500), AsOfDate: "2026-11-02", ManualConfirmation: true}
	activity := &domain.UserFundHoldingTransaction{ID: "opening-activity", UserID: "owner", HoldingID: holding.ID, FundID: strings.Repeat("1", 11), Type: domain.UserFundHoldingTransactionOpening}
	if err := repo.CreateFundHoldingWithTransaction(t.Context(), holding, activity); err == nil {
		t.Fatal("invalid activity insert should fail")
	}
	stored, err := repo.GetFundHolding(t.Context(), "owner", holding.ID)
	if err != nil || stored != nil {
		t.Fatalf("partial holding escaped transaction: %+v %v", stored, err)
	}
}
