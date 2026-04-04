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
