package repository

import (
	"context"
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

func TestPostgresFundRepositorySearchFundsOnlyReturnsActiveCatalogFunds(t *testing.T) {
	db, cleanup := openPostgresUserRepoTestDB(t)
	defer cleanup()

	repo := NewPostgresFundRepository(db)
	ctx := context.Background()
	funds := []*domain.Fund{
		{ID: "000001", Name: "华夏成长混合", CatalogStatus: domain.FundCatalogStatusActive},
		{ID: "000002", Name: "华夏成长混合(后端)", CatalogStatus: domain.FundCatalogStatusUnavailable},
		{ID: "002755", Name: "华夏成长旧代码", CatalogStatus: domain.FundCatalogStatusCatalogMissing},
		{ID: "005827", Name: "易方达蓝筹精选混合", CatalogStatus: domain.FundCatalogStatusActive},
	}
	for _, fund := range funds {
		if err := repo.SaveFund(ctx, fund); err != nil {
			t.Fatalf("SaveFund(%s) error = %v", fund.ID, err)
		}
	}

	results, err := repo.SearchFunds(ctx, "华夏成长", 10)
	if err != nil {
		t.Fatalf("SearchFunds(华夏成长) error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1: %+v", len(results), results)
	}
	if results[0].ID != "000001" {
		t.Fatalf("results[0].ID = %s, want 000001", results[0].ID)
	}

	results, err = repo.SearchFunds(ctx, "蓝", 10)
	if err != nil {
		t.Fatalf("SearchFunds(蓝) error = %v", err)
	}
	if len(results) != 1 || results[0].ID != "005827" {
		t.Fatalf("SearchFunds(蓝) = %+v, want only 005827", results)
	}

	results, err = repo.SearchFunds(ctx, "000002", 10)
	if err != nil {
		t.Fatalf("SearchFunds(000002) error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("len(results) = %d, want 0 for unavailable catalog fund: %+v", len(results), results)
	}

	missingFund, err := repo.GetFundByID(ctx, "002755")
	if err != nil {
		t.Fatalf("GetFundByID(002755) error = %v", err)
	}
	if missingFund == nil {
		t.Fatal("GetFundByID(002755) = nil, want historical catalog_missing fund to remain readable by ID")
	}
}

func TestPostgresFundRepositorySaveFundPreservesExistingCatalogStatusWhenUnset(t *testing.T) {
	db, cleanup := openPostgresUserRepoTestDB(t)
	defer cleanup()

	repo := NewPostgresFundRepository(db)
	ctx := context.Background()
	if err := repo.SaveFund(ctx, &domain.Fund{
		ID:            "000002",
		Name:          "华夏成长混合(后端)",
		CatalogStatus: domain.FundCatalogStatusUnavailable,
	}); err != nil {
		t.Fatalf("SaveFund(initial) error = %v", err)
	}

	if err := repo.SaveFund(ctx, &domain.Fund{
		ID:   "000002",
		Name: "华夏成长混合(后端)",
		Type: "混合型",
	}); err != nil {
		t.Fatalf("SaveFund(update without catalog status) error = %v", err)
	}

	fund, err := repo.GetFundByID(ctx, "000002")
	if err != nil {
		t.Fatalf("GetFundByID() error = %v", err)
	}
	if fund == nil {
		t.Fatal("GetFundByID() = nil, want saved fund")
	}
	if fund.CatalogStatus != domain.FundCatalogStatusUnavailable {
		t.Fatalf("CatalogStatus = %s, want %s", fund.CatalogStatus, domain.FundCatalogStatusUnavailable)
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
