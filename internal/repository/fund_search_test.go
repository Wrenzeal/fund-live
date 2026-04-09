package repository

import (
	"testing"

	"github.com/RomaticDOG/fund/internal/domain"
)

func TestRankAndLimitFundsPrioritizesExactAndPrefixMatches(t *testing.T) {
	funds := []*domain.Fund{
		{ID: "320007", Name: "诺安成长混合", Manager: "蔡嵩松"},
		{ID: "005827", Name: "易方达蓝筹精选混合", Manager: "张坤"},
		{ID: "005820", Name: "测试成长先锋", Manager: "王强"},
		{ID: "1005827", Name: "数字成长增强", Manager: "李四"},
	}

	results := rankAndLimitFunds(funds, "005827", 10)
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	if results[0].ID != "005827" {
		t.Fatalf("results[0].ID = %s, want 005827", results[0].ID)
	}
	if results[1].ID != "1005827" {
		t.Fatalf("results[1].ID = %s, want 1005827", results[1].ID)
	}
}

func TestRankAndLimitFundsPrioritizesPrefixNameBeforeContainsName(t *testing.T) {
	funds := []*domain.Fund{
		{ID: "320007", Name: "诺安成长混合", Manager: "蔡嵩松"},
		{ID: "000001", Name: "成长先锋混合", Manager: "张三"},
		{ID: "000002", Name: "稳健价值混合", Manager: "成长研究员"},
	}

	results := rankAndLimitFunds(funds, "成长", 10)
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}

	if results[0].ID != "000001" {
		t.Fatalf("results[0].ID = %s, want 000001", results[0].ID)
	}
	if results[1].ID != "320007" {
		t.Fatalf("results[1].ID = %s, want 320007", results[1].ID)
	}
	if results[2].ID != "000002" {
		t.Fatalf("results[2].ID = %s, want 000002", results[2].ID)
	}
}

func TestMemoryFundRepositorySearchFundsUsesStableRanking(t *testing.T) {
	repo := &MemoryFundRepository{
		funds: map[string]*domain.Fund{
			"320007": {ID: "320007", Name: "诺安成长混合", Manager: "蔡嵩松"},
			"000001": {ID: "000001", Name: "成长先锋混合", Manager: "张三"},
			"000002": {ID: "000002", Name: "稳健价值混合", Manager: "成长研究员"},
		},
		holdings: make(map[string][]domain.StockHolding),
		history:  make(map[string][]domain.FundHistory),
	}

	results, err := repo.SearchFunds(t.Context(), "成长", 2)
	if err != nil {
		t.Fatalf("SearchFunds() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	if results[0].ID != "000001" {
		t.Fatalf("results[0].ID = %s, want 000001", results[0].ID)
	}
	if results[1].ID != "320007" {
		t.Fatalf("results[1].ID = %s, want 320007", results[1].ID)
	}
}

func TestMemoryFundRepositorySaveHoldingsNilKeepsExistingHoldings(t *testing.T) {
	repo := NewMemoryFundRepository()

	before, err := repo.GetFundHoldings(t.Context(), "005827")
	if err != nil {
		t.Fatalf("GetFundHoldings() error = %v", err)
	}

	if err := repo.SaveHoldings(t.Context(), "005827", nil); err != nil {
		t.Fatalf("SaveHoldings(nil) error = %v", err)
	}

	after, err := repo.GetFundHoldings(t.Context(), "005827")
	if err != nil {
		t.Fatalf("GetFundHoldings() after error = %v", err)
	}

	if len(after) != len(before) {
		t.Fatalf("len(after) = %d, want %d", len(after), len(before))
	}
}

func TestMemoryFundRepositoryListFundIDsWithHoldings(t *testing.T) {
	repo := NewMemoryFundRepository()

	fundIDs, err := repo.ListFundIDsWithHoldings(t.Context())
	if err != nil {
		t.Fatalf("ListFundIDsWithHoldings() error = %v", err)
	}

	if len(fundIDs) == 0 {
		t.Fatal("ListFundIDsWithHoldings() returned no fund IDs")
	}
}
