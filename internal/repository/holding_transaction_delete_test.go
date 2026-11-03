package repository

import (
	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"testing"
	"time"
)

func checkHoldingTransactionDelete(t *testing.T, repo domain.UserFundHoldingRepository) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	records := []domain.UserFundHoldingTransaction{
		{ID: "early", UserID: "owner", HoldingID: "lot", FundID: "005827", CreatedAt: now, Metadata: map[string]string{"keep": "yes"}},
		{ID: "target", UserID: "owner", HoldingID: "lot", FundID: "005827", CreatedAt: now.Add(time.Second)},
		{ID: "same-clock", UserID: "owner", HoldingID: "lot", FundID: "005827", CreatedAt: now.Add(time.Second)},
		{ID: "future", UserID: "owner", HoldingID: "lot", FundID: "005827", CreatedAt: now.Add(2 * time.Second)},
		{ID: "other-lot", UserID: "owner", HoldingID: "other", FundID: "005827", CreatedAt: now},
		{ID: "other-user", UserID: "other", HoldingID: "lot", FundID: "005827", CreatedAt: now},
	}
	for i := range records {
		records[i].Type = domain.UserFundHoldingTransactionBuy
		if err := repo.SaveFundHoldingTransaction(t.Context(), &records[i]); err != nil {
			t.Fatal(err)
		}
	}
	if deleted, err := repo.DeleteFundHoldingTransaction(t.Context(), "other", "target"); err != nil || deleted {
		t.Fatalf("cross-owner deletion = %v, %v", deleted, err)
	}
	if deleted, err := repo.DeleteFundHoldingTransaction(t.Context(), "owner", "target"); err != nil || !deleted {
		t.Fatalf("delete = %v, %v", deleted, err)
	}
	if record, err := repo.GetFundHoldingTransaction(t.Context(), "owner", "target"); err != nil || record != nil {
		t.Fatalf("deleted detail = %v, %v", record, err)
	}
	for _, id := range []string{"early", "same-clock"} {
		record, err := repo.GetFundHoldingTransaction(t.Context(), "owner", id)
		if err != nil || record == nil || record.Metadata[domain.HoldingTransactionDeletedHistoryKey] != "true" {
			t.Fatalf("missing persistent guard on %s: %+v, %v", id, record, err)
		}
		if id == "early" && record.Metadata["keep"] != "yes" {
			t.Fatal("existing metadata lost")
		}
	}
	for _, id := range []string{"future", "other-lot", "other-user"} {
		user := "owner"
		if id == "other-user" {
			user = "other"
		}
		record, err := repo.GetFundHoldingTransaction(t.Context(), user, id)
		if err != nil || record == nil || record.Metadata[domain.HoldingTransactionDeletedHistoryKey] != "" {
			t.Fatalf("unrelated record changed: %+v, %v", record, err)
		}
	}
	if deleted, err := repo.DeleteFundHoldingTransaction(t.Context(), "owner", "target"); err != nil || deleted {
		t.Fatalf("repeat deletion = %v, %v", deleted, err)
	}
}

func TestMemoryHoldingTransactionDelete(t *testing.T) {
	checkHoldingTransactionDelete(t, NewMemoryUserRepository())
}

func TestPostgresHoldingTransactionDelete(t *testing.T) {
	db, cleanup := openPostgresUserRepoTestDB(t)
	defer cleanup()
	checkHoldingTransactionDelete(t, NewPostgresUserRepository(db))
	var count int64
	if err := db.Model(&database.UserFundHoldingTransaction{}).Where("id = ?", "target").Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("record not physically removed: count=%d err=%v", count, err)
	}
}
