package repository

import (
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
)

func TestMemoryUserRepositoryTransactionsEqualCreatedAtUseLatestInsertion(t *testing.T) {
	repo := NewMemoryUserRepository()
	createdAt := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	transactions := []domain.UserFundHoldingTransaction{
		{ID: "z-buy", UserID: "user-1", FundID: "005827", Type: domain.UserFundHoldingTransactionBuy, CreatedAt: createdAt},
		{ID: "a-sell", UserID: "user-1", FundID: "005827", Type: domain.UserFundHoldingTransactionSell, CreatedAt: createdAt},
		{ID: "m-correction", UserID: "user-1", FundID: "000001", Type: domain.UserFundHoldingTransactionCorrection, CreatedAt: createdAt},
		{ID: "newer", UserID: "user-1", FundID: "005827", Type: domain.UserFundHoldingTransactionBuy, CreatedAt: createdAt.Add(time.Second)},
		{ID: "older-late-insertion", UserID: "user-1", FundID: "005827", Type: domain.UserFundHoldingTransactionBuy, CreatedAt: createdAt.Add(-time.Second)},
	}
	for i := range transactions {
		if err := repo.SaveFundHoldingTransaction(t.Context(), &transactions[i]); err != nil {
			t.Fatalf("SaveFundHoldingTransaction() error = %v", err)
		}
	}

	tests := []struct {
		name   string
		filter domain.UserFundHoldingTransactionFilter
		want   []string
	}{
		{
			name:   "creation time then latest insertion",
			filter: domain.UserFundHoldingTransactionFilter{Limit: 10},
			want:   []string{"newer", "m-correction", "a-sell", "z-buy", "older-late-insertion"},
		},
		{
			name:   "filtered latest transaction",
			filter: domain.UserFundHoldingTransactionFilter{FundID: "005827", Types: []domain.UserFundHoldingTransactionType{domain.UserFundHoldingTransactionBuy, domain.UserFundHoldingTransactionSell}, Limit: 10},
			want:   []string{"newer", "a-sell", "z-buy", "older-late-insertion"},
		},
		{
			name:   "stable page within equal timestamps",
			filter: domain.UserFundHoldingTransactionFilter{Offset: 1, Limit: 2},
			want:   []string{"m-correction", "a-sell"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := repo.ListFundHoldingTransactionsFiltered(t.Context(), "user-1", test.filter)
			if err != nil {
				t.Fatalf("ListFundHoldingTransactionsFiltered() error = %v", err)
			}
			if len(got) != len(test.want) {
				t.Fatalf("transactions count = %d, want %d", len(got), len(test.want))
			}
			for i, want := range test.want {
				if got[i].ID != want {
					t.Fatalf("transaction[%d].ID = %q, want %q", i, got[i].ID, want)
				}
			}
		})
	}

	for _, transaction := range transactions[:3] {
		stored, err := repo.GetFundHoldingTransaction(t.Context(), "user-1", transaction.ID)
		if err != nil || stored == nil || !stored.CreatedAt.Equal(createdAt) {
			t.Fatalf("original timestamp changed for %q: %+v, error = %v", transaction.ID, stored, err)
		}
	}
}
