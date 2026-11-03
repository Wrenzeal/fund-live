package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/shopspring/decimal"
)

func TestDeleteHoldingTransactionRemovesHistoryWithoutChangingPosition(t *testing.T) {
	for _, voided := range []bool{false, true} {
		t.Run(map[bool]string{false: "active", true: "voided"}[voided], func(t *testing.T) {
			repo := repository.NewMemoryUserRepository()
			svc := NewUserPreferenceService(repository.NewMemoryFundRepository(), repo, repo, repo, repo)
			position := &domain.UserFundHolding{ID: "holding", UserID: "owner", FundID: "005827", Amount: decimal.NewFromInt(1000), Shares: decimal.NewFromInt(500)}
			if err := repo.SaveFundHolding(t.Context(), position); err != nil {
				t.Fatal(err)
			}
			record := &domain.UserFundHoldingTransaction{ID: "tx", UserID: "owner", HoldingID: "holding", FundID: "005827", Type: domain.UserFundHoldingTransactionBuy, Amount: decimal.NewFromInt(1000), Voided: voided}
			if err := repo.SaveFundHoldingTransaction(t.Context(), record); err != nil {
				t.Fatal(err)
			}
			if err := svc.DeleteFundHoldingTransaction(t.Context(), "other", "tx"); !errors.Is(err, ErrFundHoldingTransactionNotFound) {
				t.Fatalf("cross-owner error = %v", err)
			}
			if err := svc.DeleteFundHoldingTransaction(t.Context(), "owner", "tx"); err != nil {
				t.Fatal(err)
			}
			if tx, _ := repo.GetFundHoldingTransaction(t.Context(), "owner", "tx"); tx != nil {
				t.Fatal("deleted transaction still readable")
			}
			records, err := svc.ListFundHoldingTransactions(t.Context(), "owner", 50)
			if err != nil || len(records) != 0 {
				t.Fatalf("history = %v, %v", records, err)
			}
			if _, err := svc.GetFundHoldingTransactionDetail(t.Context(), "owner", "tx"); !errors.Is(err, ErrFundHoldingTransactionNotFound) {
				t.Fatalf("detail error = %v", err)
			}
			after, _ := repo.GetFundHolding(t.Context(), "owner", "holding")
			if after == nil || !after.Amount.Equal(position.Amount) || !after.Shares.Equal(position.Shares) {
				t.Fatalf("position changed: %+v", after)
			}
			if err := svc.DeleteFundHoldingTransaction(t.Context(), "owner", "tx"); !errors.Is(err, ErrFundHoldingTransactionNotFound) {
				t.Fatalf("repeat error = %v", err)
			}
		})
	}
}

func TestConcurrentDeleteAndRollbackCannotFailAfterMutatingHoldings(t *testing.T) {
	for i := 0; i < 20; i++ {
		repo := repository.NewMemoryUserRepository()
		svc := NewUserPreferenceService(repository.NewMemoryFundRepository(), repo, repo, repo, repo)
		holding := &domain.UserFundHolding{ID: "holding", UserID: "owner", FundID: "005827", Amount: decimal.NewFromInt(1000)}
		if err := repo.SaveFundHolding(t.Context(), holding); err != nil {
			t.Fatal(err)
		}
		transaction := &domain.UserFundHoldingTransaction{ID: "buy", UserID: "owner", HoldingID: "holding", FundID: "005827", Type: domain.UserFundHoldingTransactionBuy, Amount: decimal.NewFromInt(1000)}
		if err := repo.SaveFundHoldingTransaction(t.Context(), transaction); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		var rollbackErr error
		start := make(chan struct{})
		wg.Add(2)
		go func() { defer wg.Done(); <-start; _ = svc.DeleteFundHoldingTransaction(t.Context(), "owner", "buy") }()
		go func() {
			defer wg.Done()
			<-start
			_, rollbackErr = svc.ApplyFundHoldingTransactionRollback(t.Context(), "owner", "buy", "concurrent test")
		}()
		close(start)
		wg.Wait()
		if rollbackErr != nil {
			after, _ := repo.GetFundHolding(t.Context(), "owner", "holding")
			if after == nil || !after.Amount.Equal(holding.Amount) {
				t.Fatal("failed concurrent rollback mutated the current position")
			}
		}
	}
}

func TestDeletedLaterHistoryCannotUnlockEarlierRollback(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	svc := NewUserPreferenceService(repository.NewMemoryFundRepository(), repo, repo, repo, repo)
	position := &domain.UserFundHolding{ID: "holding", UserID: "owner", FundID: "005827", Amount: decimal.NewFromInt(1200), Shares: decimal.NewFromInt(600)}
	if err := repo.SaveFundHolding(t.Context(), position); err != nil {
		t.Fatal(err)
	}
	start := time.Now().Add(-time.Hour)
	transactions := []domain.UserFundHoldingTransaction{
		{ID: "buy", UserID: "owner", HoldingID: "holding", FundID: "005827", Type: domain.UserFundHoldingTransactionBuy, Amount: decimal.NewFromInt(1000), CreatedAt: start},
		{ID: "later", UserID: "owner", HoldingID: "holding", FundID: "005827", Type: domain.UserFundHoldingTransactionBuy, Amount: decimal.NewFromInt(200), CreatedAt: start.Add(time.Minute)},
		{ID: "other-lot", UserID: "owner", HoldingID: "unrelated", FundID: "005827", Type: domain.UserFundHoldingTransactionBuy, Amount: decimal.NewFromInt(50), CreatedAt: start},
	}
	for i := range transactions {
		if err := repo.SaveFundHoldingTransaction(t.Context(), &transactions[i]); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.DeleteFundHoldingTransaction(t.Context(), "owner", "later"); err != nil {
		t.Fatal(err)
	}
	preview, err := svc.PreviewFundHoldingTransactionRollback(t.Context(), "owner", "buy")
	if err != nil {
		t.Fatal(err)
	}
	if preview.CanApplyAutomatically {
		t.Fatal("deleting later history unlocked unsafe rollback")
	}
	if _, err := svc.ApplyFundHoldingTransactionRollback(t.Context(), "owner", "buy", "test"); !errors.Is(err, ErrUnsafeHoldingRollback) {
		t.Fatalf("rollback error = %v", err)
	}
	after, _ := repo.GetFundHolding(t.Context(), "owner", "holding")
	if !after.Amount.Equal(position.Amount) {
		t.Fatal("denied rollback changed holding")
	}
	unrelated, _ := repo.GetFundHoldingTransaction(t.Context(), "owner", "other-lot")
	if unrelated.Metadata[domain.HoldingTransactionDeletedHistoryKey] != "" {
		t.Fatal("unrelated lot was blocked")
	}
}
