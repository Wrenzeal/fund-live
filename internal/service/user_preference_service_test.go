package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/shopspring/decimal"
)

type countingFundRepository struct {
	*repository.MemoryFundRepository

	getFundByIDCalls            int
	getFundsByIDsCalls          int
	getLatestFundHistoryCalls   int
	getLatestFundHistoriesCalls int
}

func newCountingFundRepository() *countingFundRepository {
	return &countingFundRepository{
		MemoryFundRepository: repository.NewMemoryFundRepository(),
	}
}

func (r *countingFundRepository) GetFundByID(ctx context.Context, fundID string) (*domain.Fund, error) {
	r.getFundByIDCalls++
	return r.MemoryFundRepository.GetFundByID(ctx, fundID)
}

func (r *countingFundRepository) GetFundsByIDs(ctx context.Context, fundIDs []string) (map[string]*domain.Fund, error) {
	r.getFundsByIDsCalls++
	return r.MemoryFundRepository.GetFundsByIDs(ctx, fundIDs)
}

func (r *countingFundRepository) GetLatestFundHistory(ctx context.Context, fundID string) (*domain.FundHistory, error) {
	r.getLatestFundHistoryCalls++
	return r.MemoryFundRepository.GetLatestFundHistory(ctx, fundID)
}

func (r *countingFundRepository) GetLatestFundHistoriesByFundIDs(ctx context.Context, fundIDs []string) (map[string]*domain.FundHistory, error) {
	r.getLatestFundHistoriesCalls++
	return r.MemoryFundRepository.GetLatestFundHistoriesByFundIDs(ctx, fundIDs)
}

func TestUserPreferenceServiceAddFavoriteFund(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	if err := service.AddFavoriteFund(context.Background(), "user-1", "005827"); err != nil {
		t.Fatalf("AddFavoriteFund() error = %v", err)
	}

	favorites, err := service.ListFavoriteFunds(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFavoriteFunds() error = %v", err)
	}
	if len(favorites) != 1 {
		t.Fatalf("favorites len = %d, want 1", len(favorites))
	}
	if favorites[0].Fund == nil || favorites[0].Fund.ID != "005827" {
		t.Fatalf("favorite fund = %+v", favorites[0].Fund)
	}
}

func TestUserPreferenceServiceRejectsInvalidHoldingOverrides(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	err := service.ReplaceHoldingOverrides(context.Background(), "user-1", "005827", []domain.UserHoldingOverride{
		{
			StockCode:    "600519",
			StockName:    "贵州茅台",
			Exchange:     domain.ExchangeSH,
			HoldingRatio: decimal.NewFromFloat(70),
		},
		{
			StockCode:    "000858",
			StockName:    "五粮液",
			Exchange:     domain.ExchangeSZ,
			HoldingRatio: decimal.NewFromFloat(40),
		},
	})
	if !errors.Is(err, ErrInvalidHoldingOverride) {
		t.Fatalf("ReplaceHoldingOverrides() error = %v, want %v", err, ErrInvalidHoldingOverride)
	}
}

func TestUserPreferenceServiceCreatesWatchlistGroupAndFund(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	group, err := service.CreateWatchlistGroup(context.Background(), "user-1", "核心观察", "长期重点跟踪")
	if err != nil {
		t.Fatalf("CreateWatchlistGroup() error = %v", err)
	}

	if err := service.AddWatchlistFund(context.Background(), "user-1", group.ID, "005827"); err != nil {
		t.Fatalf("AddWatchlistFund() error = %v", err)
	}

	groups, err := service.ListWatchlistGroups(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListWatchlistGroups() error = %v", err)
	}
	if len(groups) != 1 || len(groups[0].Funds) != 1 {
		t.Fatalf("groups = %+v", groups)
	}
	if groups[0].Funds[0].Fund == nil || groups[0].Funds[0].Fund.ID != "005827" {
		t.Fatalf("watchlist fund = %+v", groups[0].Funds[0].Fund)
	}
}

func TestUserPreferenceServiceCreatesFundHolding(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	if !holding.Amount.Equal(decimal.NewFromInt(50000)) {
		t.Fatalf("holding amount = %s", holding.Amount.String())
	}
	if holding.AsOfDate != "2026-03-30" {
		t.Fatalf("holding as_of_date = %s, want 2026-03-30", holding.AsOfDate)
	}
	if holding.TradeAt == "" {
		t.Fatalf("holding trade_at should be populated")
	}

	holdings, err := service.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings) != 1 || holdings[0].Fund == nil || holdings[0].Fund.ID != "005827" {
		t.Fatalf("holdings = %+v", holdings)
	}
}

func TestUserPreferenceServiceDelaysHoldingPricingDateAfterCutoff(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T15:00:00+08:00", "收盘后申购")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	if holding.AsOfDate != "2026-03-31" {
		t.Fatalf("holding as_of_date = %s, want 2026-03-31", holding.AsOfDate)
	}
}

func TestUserPreferenceServiceMovesWeekendHoldingToNextTradingDay(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-29T10:00:00+08:00", "周末申购")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	if holding.AsOfDate != "2026-03-30" {
		t.Fatalf("holding as_of_date = %s, want 2026-03-30", holding.AsOfDate)
	}
}

func TestUserPreferenceServiceListFundHoldingsUsesLatestOfficialHistory(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	expectedDate := expectedOfficialHistoryDate(time.Now())

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        expectedDate,
		NetAssetVal: decimal.RequireFromString("1.8000"),
		AccumVal:    decimal.RequireFromString("2.1000"),
		DailyReturn: decimal.RequireFromString("1.2345"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}

	holdings, err := service.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings) != 1 {
		t.Fatalf("holdings len = %d, want 1", len(holdings))
	}
	if holdings[0].ID != holding.ID {
		t.Fatalf("holding id = %s, want %s", holdings[0].ID, holding.ID)
	}
	if holdings[0].ActualDate != expectedDate {
		t.Fatalf("actual date = %s, want %s", holdings[0].ActualDate, expectedDate)
	}
	if holdings[0].ActualDailyReturn != "1.2345" {
		t.Fatalf("actual daily return = %s, want 1.2345", holdings[0].ActualDailyReturn)
	}
}

func TestUserPreferenceServiceListWatchlistGroupsUsesBatchFundLookup(t *testing.T) {
	fundRepo := newCountingFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	group, err := service.CreateWatchlistGroup(context.Background(), "user-1", "核心观察", "长期重点跟踪")
	if err != nil {
		t.Fatalf("CreateWatchlistGroup() error = %v", err)
	}
	if err := service.AddWatchlistFund(context.Background(), "user-1", group.ID, "005827"); err != nil {
		t.Fatalf("AddWatchlistFund() error = %v", err)
	}
	if err := service.AddWatchlistFund(context.Background(), "user-1", group.ID, "003095"); err != nil {
		t.Fatalf("AddWatchlistFund() error = %v", err)
	}
	fundRepo.getFundByIDCalls = 0
	fundRepo.getFundsByIDsCalls = 0

	groups, err := service.ListWatchlistGroups(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListWatchlistGroups() error = %v", err)
	}
	if len(groups) != 1 || len(groups[0].Funds) != 2 {
		t.Fatalf("groups = %+v", groups)
	}
	if fundRepo.getFundByIDCalls != 0 {
		t.Fatalf("GetFundByID() calls = %d, want 0", fundRepo.getFundByIDCalls)
	}
	if fundRepo.getFundsByIDsCalls != 1 {
		t.Fatalf("GetFundsByIDs() calls = %d, want 1", fundRepo.getFundsByIDsCalls)
	}
}

func TestUserPreferenceServiceListFundHoldingsUsesBatchHistoryLookup(t *testing.T) {
	fundRepo := newCountingFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	expectedDate := expectedOfficialHistoryDate(time.Now())

	if _, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓"); err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	if _, err := service.CreateFundHolding(context.Background(), "user-1", "003095", "28000", "2026-03-30T14:30:00+08:00", "主题仓位"); err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        expectedDate,
		NetAssetVal: decimal.RequireFromString("1.8000"),
		AccumVal:    decimal.RequireFromString("2.1000"),
		DailyReturn: decimal.RequireFromString("1.2345"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}
	fundRepo.getFundByIDCalls = 0
	fundRepo.getFundsByIDsCalls = 0
	fundRepo.getLatestFundHistoryCalls = 0
	fundRepo.getLatestFundHistoriesCalls = 0

	holdings, err := service.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings) != 2 {
		t.Fatalf("holdings len = %d, want 2", len(holdings))
	}
	if fundRepo.getFundByIDCalls != 0 {
		t.Fatalf("GetFundByID() calls = %d, want 0", fundRepo.getFundByIDCalls)
	}
	if fundRepo.getLatestFundHistoryCalls != 0 {
		t.Fatalf("GetLatestFundHistory() calls = %d, want 0", fundRepo.getLatestFundHistoryCalls)
	}
	if fundRepo.getFundsByIDsCalls != 1 {
		t.Fatalf("GetFundsByIDs() calls = %d, want 1", fundRepo.getFundsByIDsCalls)
	}
	if fundRepo.getLatestFundHistoriesCalls != 1 {
		t.Fatalf("GetLatestFundHistoriesByFundIDs() calls = %d, want 1", fundRepo.getLatestFundHistoriesCalls)
	}
}
