package service

import (
	"context"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/shopspring/decimal"
)

func TestHoldingNAVRefreshRepairsShiftedDatesAndBackfillsOldDate(t *testing.T) {
	repo := repository.NewMemoryFundRepository()
	for _, date := range []string{"2026-11-01", "2025-10-10"} {
		_ = repo.SaveFundHistory(t.Context(), &domain.FundHistory{FundID: "014662", Date: date, NetAssetVal: decimal.NewFromInt(9)})
	}
	source := stubOfficialNAVHistoryCrawler{histories: map[string][]domain.FundHistory{"014662": {
		{FundID: "014662", Date: "2025-03-03", NetAssetVal: decimal.NewFromInt(1)},
		{FundID: "014662", Date: "2026-10-30", NetAssetVal: decimal.NewFromInt(2)},
		{FundID: "014662", Date: "2026-11-02", NetAssetVal: decimal.RequireFromString("2.2")},
	}}}
	loader := NewHoldingNAVRefresher(repo, source)
	if err := loader.Ensure(t.Context(), "014662", []string{"2025-03-03"}); err != nil {
		t.Fatal(err)
	}
	rows, _ := repo.GetFundHistoriesByLookupKeys(t.Context(), []domain.FundHistoryLookupKey{{FundID: "014662", Date: "2026-11-01"}, {FundID: "014662", Date: "2025-03-03"}, {FundID: "014662", Date: "2025-10-10"}})
	if rows[domain.FundHistoryLookupKey{FundID: "014662", Date: "2026-11-01"}] != nil {
		t.Fatal("UTC-shifted recent row remains")
	}
	if rows[domain.FundHistoryLookupKey{FundID: "014662", Date: "2025-03-03"}] == nil {
		t.Fatal("old confirmation NAV not fetched")
	}
	if rows[domain.FundHistoryLookupKey{FundID: "014662", Date: "2025-10-10"}] == nil {
		t.Fatal("unrelated old history removed outside bounded refresh")
	}
}

func TestPendingHistoricalConfirmationPersistsWithoutOverwritingOpeningShares(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	repo := repository.NewMemoryUserRepository()
	_ = fundRepo.SaveFund(t.Context(), &domain.Fund{ID: "014662"})
	for _, date := range []string{"2025-03-03", "2026-11-02"} {
		_ = fundRepo.SaveFundHistory(t.Context(), &domain.FundHistory{FundID: "014662", Date: date, NetAssetVal: decimal.NewFromInt(2)})
	}
	now := time.Now()
	pending := &domain.UserFundHolding{ID: "pending", UserID: "owner", FundID: "014662", Amount: decimal.NewFromInt(1000), AsOfDate: "2025-03-03", UpdatedAt: now, EntryType: "buy"}
	opening := &domain.UserFundHolding{ID: "opening", UserID: "owner", FundID: "014662", Amount: decimal.NewFromInt(900), Shares: decimal.NewFromInt(777), AsOfDate: "2026-11-02", EntryType: "opening", ManualConfirmation: true, UpdatedAt: now}
	_ = repo.SaveFundHolding(t.Context(), pending)
	_ = repo.SaveFundHolding(t.Context(), opening)
	svc := NewUserPreferenceService(fundRepo, repo, repo, repo, repo)
	if _, err := svc.ListFundHoldings(context.Background(), "owner"); err != nil {
		t.Fatal(err)
	}
	stored, _ := repo.GetFundHolding(t.Context(), "owner", "pending")
	kept, _ := repo.GetFundHolding(t.Context(), "owner", "opening")
	if !stored.Shares.Equal(decimal.NewFromInt(500)) {
		t.Fatalf("pending shares not persisted: %+v", stored)
	}
	if !kept.Shares.Equal(decimal.NewFromInt(777)) {
		t.Fatal("opening shares were rederived from principal")
	}
}

func TestCorrectedNAVRefreshesOnlySafeAutomaticShares(t *testing.T) {
	funds := repository.NewMemoryFundRepository()
	repo := repository.NewMemoryUserRepository()
	_ = funds.SaveFund(t.Context(), &domain.Fund{ID: "014662"})
	_ = funds.SaveFundHistory(t.Context(), &domain.FundHistory{FundID: "014662", Date: "2025-03-03", NetAssetVal: decimal.NewFromInt(2)})
	_ = funds.SaveFundHistory(t.Context(), &domain.FundHistory{FundID: "014662", Date: "2026-11-02", NetAssetVal: decimal.RequireFromString("2.2")})
	oldNAV := decimal.RequireFromString("1.5")
	oldShares := decimal.NewFromInt(1000).DivRound(oldNAV, 6)
	for _, id := range []string{"automatic", "reinvested"} {
		shares := oldShares
		if id == "reinvested" {
			shares = shares.Add(decimal.NewFromInt(100))
		}
		_ = repo.SaveFundHolding(t.Context(), &domain.UserFundHolding{ID: id, UserID: "owner", FundID: "014662", EntryType: "buy", Amount: decimal.NewFromInt(1000), Shares: shares, ConfirmedNav: oldNAV, ConfirmedNavDate: "2025-03-03", AsOfDate: "2025-03-03", UpdatedAt: time.Now()})
	}
	svc := NewUserPreferenceService(funds, repo, repo, repo, repo)
	if _, err := svc.ListFundHoldings(t.Context(), "owner"); err != nil {
		t.Fatal(err)
	}
	auto, _ := repo.GetFundHolding(t.Context(), "owner", "automatic")
	reinvested, _ := repo.GetFundHolding(t.Context(), "owner", "reinvested")
	if !auto.Shares.Equal(decimal.NewFromInt(500)) {
		t.Fatal("automatic share confirmation was not repaired")
	}
	if !reinvested.Shares.Equal(oldShares.Add(decimal.NewFromInt(100))) {
		t.Fatal("reinvested shares were overwritten")
	}
}
