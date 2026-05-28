package service

import (
	"context"
	"errors"
	"strings"
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
	getHistoryLookupCalls       int
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

func (r *countingFundRepository) GetFundHistoriesByLookupKeys(ctx context.Context, keys []domain.FundHistoryLookupKey) (map[domain.FundHistoryLookupKey]*domain.FundHistory, error) {
	r.getHistoryLookupCalls++
	return r.MemoryFundRepository.GetFundHistoriesByLookupKeys(ctx, keys)
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

func TestUserPreferenceServiceUpdatesWatchlistGroup(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	group, err := service.CreateWatchlistGroup(context.Background(), "user-1", "核心观察", "长期重点跟踪")
	if err != nil {
		t.Fatalf("CreateWatchlistGroup() error = %v", err)
	}

	updated, err := service.UpdateWatchlistGroup(context.Background(), "user-1", group.ID, "核心跟踪", "聚焦长期配置与核心风格切换", "amber")
	if err != nil {
		t.Fatalf("UpdateWatchlistGroup() error = %v", err)
	}
	if updated.Name != "核心跟踪" {
		t.Fatalf("updated name = %q", updated.Name)
	}
	if updated.Description != "聚焦长期配置与核心风格切换" {
		t.Fatalf("updated description = %q", updated.Description)
	}
	if updated.Accent != "amber" {
		t.Fatalf("updated accent = %q", updated.Accent)
	}
	if !updated.UpdatedAt.After(group.UpdatedAt) && !updated.UpdatedAt.Equal(group.UpdatedAt) {
		t.Fatalf("updated timestamp = %v, created timestamp = %v", updated.UpdatedAt, group.UpdatedAt)
	}

	groups, err := service.ListWatchlistGroups(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListWatchlistGroups() error = %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("groups len = %d, want 1", len(groups))
	}
	if groups[0].Name != "核心跟踪" {
		t.Fatalf("persisted name = %q", groups[0].Name)
	}
	if groups[0].Description != "聚焦长期配置与核心风格切换" {
		t.Fatalf("persisted description = %q", groups[0].Description)
	}
	if groups[0].Accent != "amber" {
		t.Fatalf("persisted accent = %q", groups[0].Accent)
	}
}

func TestUserPreferenceServiceRejectsEmptyWatchlistGroupNameUpdate(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	group, err := service.CreateWatchlistGroup(context.Background(), "user-1", "核心观察", "长期重点跟踪")
	if err != nil {
		t.Fatalf("CreateWatchlistGroup() error = %v", err)
	}

	_, err = service.UpdateWatchlistGroup(context.Background(), "user-1", group.ID, "   ", "新的说明", "cyan")
	if !errors.Is(err, ErrInvalidWatchlistGroup) {
		t.Fatalf("UpdateWatchlistGroup() error = %v, want %v", err, ErrInvalidWatchlistGroup)
	}
}

func TestUserPreferenceServiceReordersWatchlistGroups(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	first, err := service.CreateWatchlistGroup(context.Background(), "user-1", "第一组", "说明一")
	if err != nil {
		t.Fatalf("CreateWatchlistGroup(first) error = %v", err)
	}
	second, err := service.CreateWatchlistGroup(context.Background(), "user-1", "第二组", "说明二")
	if err != nil {
		t.Fatalf("CreateWatchlistGroup(second) error = %v", err)
	}
	third, err := service.CreateWatchlistGroup(context.Background(), "user-1", "第三组", "说明三")
	if err != nil {
		t.Fatalf("CreateWatchlistGroup(third) error = %v", err)
	}

	if err := service.ReorderWatchlistGroups(context.Background(), "user-1", []string{first.ID, third.ID, second.ID}); err != nil {
		t.Fatalf("ReorderWatchlistGroups() error = %v", err)
	}

	groups, err := service.ListWatchlistGroups(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListWatchlistGroups() error = %v", err)
	}
	if len(groups) != 3 {
		t.Fatalf("groups len = %d, want 3", len(groups))
	}
	if groups[0].ID != first.ID || groups[1].ID != third.ID || groups[2].ID != second.ID {
		t.Fatalf("group order = [%s %s %s]", groups[0].ID, groups[1].ID, groups[2].ID)
	}
	if groups[0].SortOrder != 0 || groups[1].SortOrder != 1 || groups[2].SortOrder != 2 {
		t.Fatalf("sort orders = [%d %d %d]", groups[0].SortOrder, groups[1].SortOrder, groups[2].SortOrder)
	}
}

func TestUserPreferenceServiceRejectsInvalidWatchlistGroupReorder(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	first, err := service.CreateWatchlistGroup(context.Background(), "user-1", "第一组", "说明一")
	if err != nil {
		t.Fatalf("CreateWatchlistGroup(first) error = %v", err)
	}
	_, err = service.CreateWatchlistGroup(context.Background(), "user-1", "第二组", "说明二")
	if err != nil {
		t.Fatalf("CreateWatchlistGroup(second) error = %v", err)
	}

	err = service.ReorderWatchlistGroups(context.Background(), "user-1", []string{first.ID})
	if !errors.Is(err, ErrInvalidWatchlistOrder) {
		t.Fatalf("ReorderWatchlistGroups() error = %v, want %v", err, ErrInvalidWatchlistOrder)
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
	if len(holdings.Items) != 1 || holdings.Items[0].Fund == nil || holdings.Items[0].Fund.ID != "005827" {
		t.Fatalf("holdings = %+v", holdings)
	}
}

func TestUserPreferenceServiceSellsFundHoldingAndRecordsTransaction(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	updated, err := service.UpdateFundHolding(context.Background(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "50000",
		Shares:           "40000",
		ConfirmedNav:     "1.25",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:30:00+08:00",
		Note:             "确认份额",
	})
	if err != nil {
		t.Fatalf("UpdateFundHolding() error = %v", err)
	}

	remaining, err := service.SellFundHolding(context.Background(), "user-1", updated.ID, domain.SellFundHoldingInput{
		Amount:  "12500",
		TradeAt: "2026-04-01T14:50:00+08:00",
		Note:    "减仓四分之一",
	})
	if err != nil {
		t.Fatalf("SellFundHolding() error = %v", err)
	}
	if remaining.Amount.String() != "37500" {
		t.Fatalf("remaining amount = %s, want 37500", remaining.Amount.String())
	}
	if remaining.Shares != "30000" {
		t.Fatalf("remaining shares = %q, want 30000", remaining.Shares)
	}
	if remaining.Note != "减仓四分之一" {
		t.Fatalf("remaining note = %q", remaining.Note)
	}

	transactions, err := service.ListFundHoldingTransactions(context.Background(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListFundHoldingTransactions() error = %v", err)
	}
	var sellTx *domain.UserFundHoldingTransaction
	for i := range transactions {
		if transactions[i].Type == domain.UserFundHoldingTransactionSell {
			sellTx = &transactions[i]
			break
		}
	}
	if sellTx == nil {
		t.Fatalf("sell transaction not found: %+v", transactions)
	}
	if !sellTx.Amount.Equal(decimal.RequireFromString("12500")) {
		t.Fatalf("sell amount = %s, want 12500", sellTx.Amount.String())
	}
	if !sellTx.Shares.Equal(decimal.RequireFromString("10000")) {
		t.Fatalf("sell shares = %s, want 10000", sellTx.Shares.String())
	}
	if sellTx.Metadata["remaining_amount"] != "37500" {
		t.Fatalf("remaining_amount metadata = %q", sellTx.Metadata["remaining_amount"])
	}
}

func TestUserPreferenceServiceSellsAllFundHoldingAndRemovesSnapshot(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	updated, err := service.UpdateFundHolding(context.Background(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "50000",
		Shares:           "40000",
		ConfirmedNav:     "1.25",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:30:00+08:00",
		Note:             "确认份额",
	})
	if err != nil {
		t.Fatalf("UpdateFundHolding() error = %v", err)
	}

	remaining, err := service.SellFundHolding(context.Background(), "user-1", updated.ID, domain.SellFundHoldingInput{
		TradeAt: "2026-04-01T14:50:00+08:00",
		Note:    "全部赎回",
		SellAll: true,
	})
	if err != nil {
		t.Fatalf("SellFundHolding(sell all) error = %v", err)
	}
	if remaining != nil {
		t.Fatalf("remaining = %+v, want nil for closed snapshot", remaining)
	}

	holdings, err := service.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings.Items) != 0 {
		t.Fatalf("holdings len = %d, want 0 after sell all", len(holdings.Items))
	}

	transactions, err := service.ListFundHoldingTransactions(context.Background(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListFundHoldingTransactions() error = %v", err)
	}
	var sellTx *domain.UserFundHoldingTransaction
	for i := range transactions {
		if transactions[i].Type == domain.UserFundHoldingTransactionSell && transactions[i].Metadata["sell_all"] == "true" {
			sellTx = &transactions[i]
			break
		}
	}
	if sellTx == nil {
		t.Fatalf("sell-all transaction not found: %+v", transactions)
	}
	if !sellTx.Amount.Equal(decimal.RequireFromString("50000")) {
		t.Fatalf("sell-all amount = %s, want 50000", sellTx.Amount.String())
	}
	if !sellTx.Shares.Equal(decimal.RequireFromString("40000")) {
		t.Fatalf("sell-all shares = %s, want 40000", sellTx.Shares.String())
	}
}

func TestUserPreferenceServiceRecordsFundHoldingDividend(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	updated, err := service.UpdateFundHolding(context.Background(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "50000",
		Shares:           "40000",
		ConfirmedNav:     "1.25",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:30:00+08:00",
		Note:             "确认份额",
	})
	if err != nil {
		t.Fatalf("UpdateFundHolding() error = %v", err)
	}

	cash, err := service.RecordFundHoldingDividend(context.Background(), "user-1", updated.ID, domain.DividendFundHoldingInput{
		Amount:         "320.50",
		TradeAt:        "2026-04-02T14:30:00+08:00",
		Note:           "现金分红",
		SourcePlatform: "alipay",
	})
	if err != nil {
		t.Fatalf("RecordFundHoldingDividend(cash) error = %v", err)
	}
	if cash.Amount.String() != "50000" || cash.Shares != "40000" {
		t.Fatalf("cash dividend changed snapshot = amount %s shares %s, want unchanged", cash.Amount.String(), cash.Shares)
	}

	reinvested, err := service.RecordFundHoldingDividend(context.Background(), "user-1", updated.ID, domain.DividendFundHoldingInput{
		Amount:         "125",
		Shares:         "100",
		TradeAt:        "2026-04-03T14:30:00+08:00",
		Note:           "红利再投",
		Reinvest:       true,
		SourcePlatform: "wechat",
	})
	if err != nil {
		t.Fatalf("RecordFundHoldingDividend(reinvest) error = %v", err)
	}
	if reinvested.Amount.String() != "50000" {
		t.Fatalf("reinvest amount = %s, want 50000", reinvested.Amount.String())
	}
	if reinvested.Shares != "40100" {
		t.Fatalf("reinvested shares = %s, want 40100", reinvested.Shares)
	}
	if reinvested.ConfirmedNav != "1.25" || reinvested.ConfirmedNavDate != "2026-04-03" {
		t.Fatalf("reinvested nav/date = %s/%s, want 1.25/2026-04-03", reinvested.ConfirmedNav, reinvested.ConfirmedNavDate)
	}
	if reinvested.SourcePlatform != "wechat" || reinvested.SourceLabel != "微信" {
		t.Fatalf("reinvested source = %s/%s, want wechat/微信", reinvested.SourcePlatform, reinvested.SourceLabel)
	}

	transactions, err := service.ListFundHoldingTransactions(context.Background(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListFundHoldingTransactions() error = %v", err)
	}
	var cashTx, reinvestTx *domain.UserFundHoldingTransaction
	for i := range transactions {
		if transactions[i].Type != domain.UserFundHoldingTransactionDividend {
			continue
		}
		if transactions[i].Metadata["reinvest"] == "true" {
			reinvestTx = &transactions[i]
		} else {
			cashTx = &transactions[i]
		}
	}
	if cashTx == nil || reinvestTx == nil {
		t.Fatalf("dividend transactions not found: %+v", transactions)
	}
	if !cashTx.Amount.Equal(decimal.RequireFromString("320.50")) || cashTx.SourcePlatform != "alipay" {
		t.Fatalf("cash dividend tx = %+v, want amount 320.50 source alipay", cashTx)
	}
	if !reinvestTx.Amount.Equal(decimal.RequireFromString("125")) || !reinvestTx.Shares.Equal(decimal.RequireFromString("100")) {
		t.Fatalf("reinvest tx amount/shares = %s/%s, want 125/100", reinvestTx.Amount.String(), reinvestTx.Shares.String())
	}
}

func TestUserPreferenceServiceAdjustsFundHoldingShares(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	updated, err := service.UpdateFundHolding(context.Background(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "50000",
		Shares:           "40000",
		ConfirmedNav:     "1.25",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:30:00+08:00",
		Note:             "确认份额",
	})
	if err != nil {
		t.Fatalf("UpdateFundHolding() error = %v", err)
	}

	adjusted, err := service.AdjustFundHoldingShares(context.Background(), "user-1", updated.ID, domain.AdjustFundHoldingSharesInput{
		TargetShares:     "41000",
		ConfirmedNav:     "1.22",
		ConfirmedNavDate: "2026-04-03",
		TradeAt:          "2026-04-03T14:30:00+08:00",
		Note:             "平台迁移份额调整",
		SourcePlatform:   "eastmoney",
	})
	if err != nil {
		t.Fatalf("AdjustFundHoldingShares(target) error = %v", err)
	}
	if adjusted.Shares != "41000" {
		t.Fatalf("adjusted shares = %s, want 41000", adjusted.Shares)
	}
	if adjusted.ConfirmedNav != "1.22" || adjusted.ConfirmedNavDate != "2026-04-03" {
		t.Fatalf("adjusted nav/date = %s/%s, want 1.22/2026-04-03", adjusted.ConfirmedNav, adjusted.ConfirmedNavDate)
	}
	if !adjusted.ManualConfirmation {
		t.Fatalf("adjusted manual_confirmation = false, want true")
	}
	if adjusted.SourcePlatform != "eastmoney" || adjusted.SourceLabel != "天天基金" {
		t.Fatalf("adjusted source = %s/%s, want eastmoney/天天基金", adjusted.SourcePlatform, adjusted.SourceLabel)
	}

	adjusted, err = service.AdjustFundHoldingShares(context.Background(), "user-1", updated.ID, domain.AdjustFundHoldingSharesInput{
		SharesDelta:    "-500",
		TradeAt:        "2026-04-04T14:30:00+08:00",
		Note:           "手续费/份额修正",
		SourcePlatform: "bank",
	})
	if err != nil {
		t.Fatalf("AdjustFundHoldingShares(delta) error = %v", err)
	}
	if adjusted.Shares != "40500" {
		t.Fatalf("delta adjusted shares = %s, want 40500", adjusted.Shares)
	}

	transactions, err := service.ListFundHoldingTransactionsFiltered(context.Background(), "user-1", domain.UserFundHoldingTransactionFilter{
		Types: []domain.UserFundHoldingTransactionType{domain.UserFundHoldingTransactionAdjustment},
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered(adjustment) error = %v", err)
	}
	if len(transactions) != 2 {
		t.Fatalf("adjustment transactions len = %d, want 2; all=%+v", len(transactions), transactions)
	}
	var targetTx *domain.UserFundHoldingTransaction
	for i := range transactions {
		if transactions[i].Metadata["shares_delta"] == "1000" {
			targetTx = &transactions[i]
			break
		}
	}
	if targetTx == nil || targetTx.SourcePlatform != "eastmoney" {
		t.Fatalf("target adjustment tx = %+v, want source eastmoney and shares_delta 1000", targetTx)
	}
}

func TestUserPreferenceServiceRejectsInvalidDividendAndAdjustment(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	if _, err := service.RecordFundHoldingDividend(context.Background(), "user-1", holding.ID, domain.DividendFundHoldingInput{
		Amount:   "100",
		Reinvest: true,
	}); !errors.Is(err, ErrInvalidHoldingDividend) {
		t.Fatalf("reinvest without shares error = %v, want %v", err, ErrInvalidHoldingDividend)
	}

	if _, err := service.RecordFundHoldingDividend(context.Background(), "user-1", holding.ID, domain.DividendFundHoldingInput{
		Amount: "-1",
	}); !errors.Is(err, ErrInvalidHoldingDividend) {
		t.Fatalf("negative dividend error = %v, want %v", err, ErrInvalidHoldingDividend)
	}

	if _, err := service.AdjustFundHoldingShares(context.Background(), "user-1", holding.ID, domain.AdjustFundHoldingSharesInput{
		TargetShares: "41000",
		SharesDelta:  "100",
	}); !errors.Is(err, ErrInvalidHoldingAdjustment) {
		t.Fatalf("target and delta adjustment error = %v, want %v", err, ErrInvalidHoldingAdjustment)
	}

	if _, err := service.AdjustFundHoldingShares(context.Background(), "user-1", holding.ID, domain.AdjustFundHoldingSharesInput{
		SharesDelta: "-999999",
	}); !errors.Is(err, ErrInvalidHoldingAdjustment) {
		t.Fatalf("negative resulting shares error = %v, want %v", err, ErrInvalidHoldingAdjustment)
	}
}

func TestUserPreferenceServiceRejectsOversellFundHolding(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	_, err = service.SellFundHolding(context.Background(), "user-1", holding.ID, domain.SellFundHoldingInput{
		Amount:  "50000",
		TradeAt: "2026-04-01T14:50:00+08:00",
	})
	if !errors.Is(err, ErrInvalidHoldingSell) {
		t.Fatalf("SellFundHolding() error = %v, want %v", err, ErrInvalidHoldingSell)
	}
}

func TestUserPreferenceServiceRecordsFundHoldingTransactions(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	transactions, err := service.ListFundHoldingTransactions(context.Background(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListFundHoldingTransactions() after create error = %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("transactions len after create = %d, want 1", len(transactions))
	}
	if transactions[0].Type != domain.UserFundHoldingTransactionBuy {
		t.Fatalf("transaction type after create = %s, want buy", transactions[0].Type)
	}
	if transactions[0].Fund == nil || transactions[0].Fund.ID != "005827" {
		t.Fatalf("transaction fund not enriched: %+v", transactions[0].Fund)
	}
	if !transactions[0].Amount.Equal(decimal.NewFromInt(50000)) {
		t.Fatalf("transaction amount = %s, want 50000", transactions[0].Amount.String())
	}

	if _, err := service.UpdateFundHolding(context.Background(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "52000",
		Shares:           "41000.123456",
		ConfirmedNav:     "1.2683",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:59:00+08:00",
		Note:             "按支付宝校正",
	}); err != nil {
		t.Fatalf("UpdateFundHolding() error = %v", err)
	}

	if err := service.DeleteFundHolding(context.Background(), "user-1", holding.ID); err != nil {
		t.Fatalf("DeleteFundHolding() error = %v", err)
	}

	transactions, err = service.ListFundHoldingTransactions(context.Background(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListFundHoldingTransactions() after update/delete error = %v", err)
	}
	if len(transactions) != 3 {
		t.Fatalf("transactions len after update/delete = %d, want 3", len(transactions))
	}

	counts := map[domain.UserFundHoldingTransactionType]int{}
	for _, transaction := range transactions {
		counts[transaction.Type]++
		if transaction.HoldingID != holding.ID {
			t.Fatalf("transaction holding id = %s, want %s", transaction.HoldingID, holding.ID)
		}
	}
	for _, txType := range []domain.UserFundHoldingTransactionType{
		domain.UserFundHoldingTransactionBuy,
		domain.UserFundHoldingTransactionCorrection,
		domain.UserFundHoldingTransactionDelete,
	} {
		if counts[txType] != 1 {
			t.Fatalf("transaction count for %s = %d, want 1; all=%+v", txType, counts[txType], transactions)
		}
	}
}

func TestUserPreferenceServiceVoidsFundHoldingTransactionWithoutChangingSnapshot(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	transactions, err := service.ListFundHoldingTransactions(context.Background(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListFundHoldingTransactions() error = %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("transactions len = %d, want 1", len(transactions))
	}

	voided, err := service.VoidFundHoldingTransaction(context.Background(), "user-1", transactions[0].ID, "录入重复，保留痕迹")
	if err != nil {
		t.Fatalf("VoidFundHoldingTransaction() error = %v", err)
	}
	if !voided.Voided {
		t.Fatalf("voided transaction Voided = false, want true")
	}
	if voided.VoidedAt == nil {
		t.Fatalf("voided_at is nil")
	}
	if voided.VoidReason != "录入重复，保留痕迹" {
		t.Fatalf("void reason = %q", voided.VoidReason)
	}
	if voided.Fund == nil || voided.Fund.ID != "005827" {
		t.Fatalf("voided transaction fund not enriched: %+v", voided.Fund)
	}

	holdings, err := service.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings.Items) != 1 || holdings.Items[0].ID != holding.ID || holdings.Items[0].Amount.String() != "50000" {
		t.Fatalf("holding snapshot changed after void: %+v", holdings.Items)
	}

	transactions, err = service.ListFundHoldingTransactions(context.Background(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListFundHoldingTransactions() after void error = %v", err)
	}
	if len(transactions) != 1 || !transactions[0].Voided || transactions[0].VoidReason != "录入重复，保留痕迹" {
		t.Fatalf("listed transactions after void = %+v", transactions)
	}

	if _, err := service.VoidFundHoldingTransaction(context.Background(), "user-1", transactions[0].ID, "再次作废"); !errors.Is(err, ErrFundHoldingTransactionVoided) {
		t.Fatalf("repeat VoidFundHoldingTransaction() error = %v, want %v", err, ErrFundHoldingTransactionVoided)
	}
	if _, err := service.VoidFundHoldingTransaction(context.Background(), "user-2", transactions[0].ID, "越权"); !errors.Is(err, ErrFundHoldingTransactionNotFound) {
		t.Fatalf("other user VoidFundHoldingTransaction() error = %v, want %v", err, ErrFundHoldingTransactionNotFound)
	}
}

func TestUserPreferenceServicePreviewFundHoldingTransactionRollback(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	if _, err := service.UpdateFundHolding(context.Background(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "52000",
		Shares:           "41000.123456",
		ConfirmedNav:     "1.2683",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:59:00+08:00",
		Note:             "按平台校正",
		SourcePlatform:   "alipay",
	}); err != nil {
		t.Fatalf("UpdateFundHolding() error = %v", err)
	}

	transactions, err := service.ListFundHoldingTransactionsFiltered(context.Background(), "user-1", domain.UserFundHoldingTransactionFilter{
		Types: []domain.UserFundHoldingTransactionType{domain.UserFundHoldingTransactionCorrection},
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered() error = %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("correction transactions len = %d, want 1; all=%+v", len(transactions), transactions)
	}

	preview, err := service.PreviewFundHoldingTransactionRollback(context.Background(), "user-1", transactions[0].ID)
	if err != nil {
		t.Fatalf("PreviewFundHoldingTransactionRollback() error = %v", err)
	}
	if preview == nil {
		t.Fatalf("preview is nil")
	}
	if !preview.PreviewOnly || !preview.CanApplyAutomatically || preview.State != "preview" {
		t.Fatalf("preview flags = %+v, want preview with safe automatic apply enabled", preview)
	}
	if preview.Transaction.ID != transactions[0].ID || preview.Transaction.Fund == nil || preview.Transaction.Fund.ID != "005827" {
		t.Fatalf("preview transaction not enriched: %+v", preview.Transaction)
	}
	if preview.CurrentHolding == nil || preview.CurrentHolding.Amount.String() != "52000" {
		t.Fatalf("current holding = %+v, want unchanged 52000 snapshot", preview.CurrentHolding)
	}

	fields := make(map[string]domain.UserFundHoldingTransactionRollbackField, len(preview.AffectedFields))
	for _, field := range preview.AffectedFields {
		fields[field.Field] = field
	}
	amountField, ok := fields["amount"]
	if !ok {
		t.Fatalf("amount field missing: %+v", preview.AffectedFields)
	}
	if amountField.CurrentValue != "52000.00" || amountField.RollbackValue != "50000" {
		t.Fatalf("amount field = %+v, want current 52000.00 rollback 50000", amountField)
	}
	if fields["shares"].RollbackValue != "" {
		t.Fatalf("shares rollback = %+v, want empty previous shares for first correction", fields["shares"])
	}

	holdings, err := service.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings.Items) != 1 || holdings.Items[0].Amount.String() != "52000" {
		t.Fatalf("holding snapshot changed after preview: %+v", holdings.Items)
	}

	if _, err := service.PreviewFundHoldingTransactionRollback(context.Background(), "user-2", transactions[0].ID); !errors.Is(err, ErrFundHoldingTransactionNotFound) {
		t.Fatalf("other user preview error = %v, want %v", err, ErrFundHoldingTransactionNotFound)
	}

	applied, err := service.ApplyFundHoldingTransactionRollback(context.Background(), "user-1", transactions[0].ID, "校正录错，自动冲正")
	if err != nil {
		t.Fatalf("ApplyFundHoldingTransactionRollback() error = %v", err)
	}
	if !applied.Applied || applied.CurrentHolding == nil {
		t.Fatalf("applied result = %+v, want applied with current holding", applied)
	}
	if !applied.Transaction.Voided || !strings.Contains(applied.Transaction.VoidReason, "自动冲正") {
		t.Fatalf("applied transaction = %+v, want voided rollback transaction", applied.Transaction)
	}

	holdings, err = service.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() after apply error = %v", err)
	}
	if len(holdings.Items) != 1 || holdings.Items[0].Amount.String() != "50000" {
		t.Fatalf("holding snapshot after apply = %+v, want amount 50000", holdings.Items)
	}
	if holdings.Items[0].ManualConfirmation {
		t.Fatalf("holding manual confirmation after apply = true, want false")
	}

	if _, err := service.ApplyFundHoldingTransactionRollback(context.Background(), "user-1", transactions[0].ID, "重复冲正"); !errors.Is(err, ErrFundHoldingTransactionVoided) {
		t.Fatalf("repeat apply error = %v, want %v", err, ErrFundHoldingTransactionVoided)
	}
}

func TestUserPreferenceServiceCreateFundHoldingsBatch(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	result, err := service.CreateFundHoldingsBatch(context.Background(), "user-1", []domain.CreateFundHoldingInput{
		{
			FundID:         "005827",
			Amount:         "50000",
			TradeAt:        "2026-03-30T14:30:00+08:00",
			Note:           "支付宝迁移",
			SourcePlatform: "alipay",
		},
		{
			FundID:  "bad",
			Amount:  "100",
			TradeAt: "2026-03-30T14:30:00+08:00",
		},
	})
	if err != nil {
		t.Fatalf("CreateFundHoldingsBatch() error = %v", err)
	}
	if result.Total != 2 || result.CreatedCount != 1 || result.FailedCount != 1 {
		t.Fatalf("batch result = %+v, want 1 created and 1 failed", result)
	}
	if len(result.Created) != 1 || result.Created[0].FundID != "005827" || result.Created[0].SourcePlatform != "alipay" {
		t.Fatalf("created rows = %+v, want 005827 with alipay source", result.Created)
	}
	if len(result.Failed) != 1 || result.Failed[0].Code != "FUND_NOT_FOUND" {
		t.Fatalf("failed rows = %+v, want fund not found", result.Failed)
	}

	holdings, err := service.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings.Items) != 1 {
		t.Fatalf("holdings len = %d, want 1", len(holdings.Items))
	}
}

func TestUserPreferenceServiceGetFundHoldingTransactionDetail(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	if _, err := service.SellFundHolding(context.Background(), "user-1", holding.ID, domain.SellFundHoldingInput{
		Amount:  "10000",
		TradeAt: "2026-04-01T14:30:00+08:00",
		Note:    "阶段减仓",
	}); err != nil {
		t.Fatalf("SellFundHolding() error = %v", err)
	}

	transactions, err := service.ListFundHoldingTransactionsFiltered(context.Background(), "user-1", domain.UserFundHoldingTransactionFilter{
		Types: []domain.UserFundHoldingTransactionType{domain.UserFundHoldingTransactionSell},
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered() error = %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("sell transactions len = %d, want 1; all=%+v", len(transactions), transactions)
	}

	detail, err := service.GetFundHoldingTransactionDetail(context.Background(), "user-1", transactions[0].ID)
	if err != nil {
		t.Fatalf("GetFundHoldingTransactionDetail() error = %v", err)
	}
	if detail.Transaction.ID != transactions[0].ID {
		t.Fatalf("detail transaction id = %s, want %s", detail.Transaction.ID, transactions[0].ID)
	}
	if detail.RollbackPreview == nil || len(detail.RollbackPreview.AffectedFields) == 0 {
		t.Fatalf("rollback preview missing in detail: %+v", detail)
	}
	if detail.CurrentHolding == nil || detail.CurrentHolding.Amount.String() != "40000" {
		t.Fatalf("current holding = %+v, want remaining 40000", detail.CurrentHolding)
	}
	if len(detail.RelatedTransactions) == 0 {
		t.Fatalf("related transactions empty: %+v", detail)
	}
	if len(detail.ImpactChain) == 0 {
		t.Fatalf("impact chain empty: %+v", detail)
	}
	if _, err := service.GetFundHoldingTransactionDetail(context.Background(), "user-2", transactions[0].ID); !errors.Is(err, ErrFundHoldingTransactionNotFound) {
		t.Fatalf("other user detail error = %v, want %v", err, ErrFundHoldingTransactionNotFound)
	}
}

func TestUserPreferenceServiceFiltersFundHoldingTransactions(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding(005827) error = %v", err)
	}
	if _, err := service.CreateFundHolding(context.Background(), "user-1", "003095", "12000", "2026-03-30T14:30:00+08:00", "医疗仓"); err != nil {
		t.Fatalf("CreateFundHolding(003095) error = %v", err)
	}
	if _, err := service.UpdateFundHolding(context.Background(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "52000",
		Shares:           "41000.123456",
		ConfirmedNav:     "1.2683",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:59:00+08:00",
		Note:             "按平台校正",
		SourcePlatform:   "支付宝",
	}); err != nil {
		t.Fatalf("UpdateFundHolding() error = %v", err)
	}

	transactions, err := service.ListFundHoldingTransactions(context.Background(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListFundHoldingTransactions() error = %v", err)
	}
	var medicalBuyID string
	for _, transaction := range transactions {
		if transaction.FundID == "003095" && transaction.Type == domain.UserFundHoldingTransactionBuy {
			medicalBuyID = transaction.ID
			break
		}
	}
	if medicalBuyID == "" {
		t.Fatalf("003095 buy transaction not found: %+v", transactions)
	}
	if _, err := service.VoidFundHoldingTransaction(context.Background(), "user-1", medicalBuyID, "测试作废"); err != nil {
		t.Fatalf("VoidFundHoldingTransaction() error = %v", err)
	}

	byFund, err := service.ListFundHoldingTransactionsFiltered(context.Background(), "user-1", domain.UserFundHoldingTransactionFilter{
		FundID: "005827",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered(by fund) error = %v", err)
	}
	if len(byFund) != 2 {
		t.Fatalf("by fund len = %d, want 2; all=%+v", len(byFund), byFund)
	}
	for _, transaction := range byFund {
		if transaction.FundID != "005827" {
			t.Fatalf("filtered fund id = %s, want 005827", transaction.FundID)
		}
		if transaction.Fund == nil || transaction.Fund.ID != "005827" {
			t.Fatalf("filtered transaction fund not enriched: %+v", transaction.Fund)
		}
	}

	byType, err := service.ListFundHoldingTransactionsFiltered(context.Background(), "user-1", domain.UserFundHoldingTransactionFilter{
		Types: []domain.UserFundHoldingTransactionType{domain.UserFundHoldingTransactionCorrection},
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered(by type) error = %v", err)
	}
	if len(byType) != 1 || byType[0].Type != domain.UserFundHoldingTransactionCorrection {
		t.Fatalf("by type = %+v, want one correction", byType)
	}
	if byType[0].SourcePlatform != "alipay" || byType[0].SourceLabel != "支付宝" {
		t.Fatalf("correction source = %s/%s, want alipay/支付宝", byType[0].SourcePlatform, byType[0].SourceLabel)
	}

	bySource, err := service.ListFundHoldingTransactionsFiltered(context.Background(), "user-1", domain.UserFundHoldingTransactionFilter{
		SourcePlatform: "alipay",
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered(by source) error = %v", err)
	}
	if len(bySource) != 1 || bySource[0].SourcePlatform != "alipay" || bySource[0].Type != domain.UserFundHoldingTransactionCorrection {
		t.Fatalf("by source = %+v, want one alipay correction", bySource)
	}

	voidedOnly := true
	byVoided, err := service.ListFundHoldingTransactionsFiltered(context.Background(), "user-1", domain.UserFundHoldingTransactionFilter{
		Voided: &voidedOnly,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered(voided) error = %v", err)
	}
	if len(byVoided) != 1 || !byVoided[0].Voided || byVoided[0].FundID != "003095" {
		t.Fatalf("voided transactions = %+v, want one voided 003095 transaction", byVoided)
	}

	activeOnly := false
	byActive, err := service.ListFundHoldingTransactionsFiltered(context.Background(), "user-1", domain.UserFundHoldingTransactionFilter{
		Voided: &activeOnly,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered(active) error = %v", err)
	}
	if len(byActive) != 2 {
		t.Fatalf("active transactions len = %d, want 2; all=%+v", len(byActive), byActive)
	}
	for _, transaction := range byActive {
		if transaction.Voided {
			t.Fatalf("active filter returned voided transaction: %+v", transaction)
		}
	}

	byKeyword, err := service.ListFundHoldingTransactionsFiltered(context.Background(), "user-1", domain.UserFundHoldingTransactionFilter{
		Keyword: "005827",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered(keyword) error = %v", err)
	}
	if len(byKeyword) != 2 {
		t.Fatalf("keyword transactions len = %d, want 2; all=%+v", len(byKeyword), byKeyword)
	}

	createdFrom := time.Now().Add(-time.Hour)
	createdBefore := time.Now().Add(time.Hour)
	byDate, err := service.ListFundHoldingTransactionsFiltered(context.Background(), "user-1", domain.UserFundHoldingTransactionFilter{
		CreatedFrom:   &createdFrom,
		CreatedBefore: &createdBefore,
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered(date range) error = %v", err)
	}
	if len(byDate) != 3 {
		t.Fatalf("date range transactions len = %d, want 3; all=%+v", len(byDate), byDate)
	}

	byOffset, err := service.ListFundHoldingTransactionsFiltered(context.Background(), "user-1", domain.UserFundHoldingTransactionFilter{
		Voided: &activeOnly,
		Offset: 1,
		Limit:  1,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered(offset) error = %v", err)
	}
	if len(byOffset) != 1 || byOffset[0].ID == byActive[0].ID {
		t.Fatalf("offset transactions = %+v, want second active transaction", byOffset)
	}

	if _, err := service.ListFundHoldingTransactionsFiltered(context.Background(), "user-1", domain.UserFundHoldingTransactionFilter{
		Types: []domain.UserFundHoldingTransactionType{"unknown"},
	}); !errors.Is(err, ErrInvalidHoldingTransactionFilter) {
		t.Fatalf("invalid type filter error = %v, want %v", err, ErrInvalidHoldingTransactionFilter)
	}
	if _, err := service.ListFundHoldingTransactionsFiltered(context.Background(), "user-1", domain.UserFundHoldingTransactionFilter{
		SourcePlatform: "unsupported-platform",
	}); !errors.Is(err, ErrInvalidHoldingTransactionFilter) {
		t.Fatalf("invalid source filter error = %v, want %v", err, ErrInvalidHoldingTransactionFilter)
	}
}

func TestUserPreferenceServiceUpdateFundHoldingRecordsSource(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	updated, err := service.UpdateFundHolding(context.Background(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "52000",
		Shares:           "41000.123456",
		ConfirmedNav:     "1.2683",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:59:00+08:00",
		Note:             "按微信校正",
		SourcePlatform:   "wechat",
	})
	if err != nil {
		t.Fatalf("UpdateFundHolding() error = %v", err)
	}
	if updated.SourcePlatform != "wechat" || updated.SourceLabel != "微信" {
		t.Fatalf("updated source = %s/%s, want wechat/微信", updated.SourcePlatform, updated.SourceLabel)
	}

	transactions, err := service.ListFundHoldingTransactionsFiltered(context.Background(), "user-1", domain.UserFundHoldingTransactionFilter{
		SourcePlatform: "微信",
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered(source) error = %v", err)
	}
	if len(transactions) != 1 || transactions[0].Type != domain.UserFundHoldingTransactionCorrection {
		t.Fatalf("source transactions = %+v, want one correction", transactions)
	}
	if transactions[0].SourcePlatform != "wechat" || transactions[0].SourceLabel != "微信" {
		t.Fatalf("transaction source = %s/%s, want wechat/微信", transactions[0].SourcePlatform, transactions[0].SourceLabel)
	}

	if _, err := service.UpdateFundHolding(context.Background(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "52000",
		Shares:           "41000.123456",
		ConfirmedNav:     "1.2683",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:59:00+08:00",
		Note:             "非法来源",
		SourcePlatform:   "unsupported-platform",
	}); !errors.Is(err, ErrInvalidHoldingSource) {
		t.Fatalf("invalid source error = %v, want %v", err, ErrInvalidHoldingSource)
	}

	if _, err := service.UpdateFundHolding(context.Background(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "52000",
		Shares:           "41000.123456",
		ConfirmedNav:     "1.2683",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:59:00+08:00",
		Note:             "只有来源标签",
		SourceLabel:      "支付宝截图",
	}); !errors.Is(err, ErrInvalidHoldingSource) {
		t.Fatalf("source label without platform error = %v, want %v", err, ErrInvalidHoldingSource)
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

func TestUserPreferenceServiceUpdatesFundHoldingWithManualConfirmation(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	updated, err := service.UpdateFundHolding(context.Background(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "52000",
		Shares:           "41000.123456",
		ConfirmedNav:     "1.2683",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:59:00+08:00",
		Note:             "按支付宝校正",
	})
	if err != nil {
		t.Fatalf("UpdateFundHolding() error = %v", err)
	}

	if updated.Amount.String() != "52000" {
		t.Fatalf("updated amount = %s, want 52000", updated.Amount.String())
	}
	if updated.Shares != "41000.123456" {
		t.Fatalf("updated shares = %q, want 41000.123456", updated.Shares)
	}
	if updated.ConfirmedNav != "1.2683" {
		t.Fatalf("updated nav = %q, want 1.2683", updated.ConfirmedNav)
	}
	if updated.ConfirmedNavDate != "2026-03-30" {
		t.Fatalf("updated nav date = %q, want 2026-03-30", updated.ConfirmedNavDate)
	}
	if !updated.ManualConfirmation {
		t.Fatalf("updated manual_confirmation = false, want true")
	}
	if updated.Note != "按支付宝校正" {
		t.Fatalf("updated note = %q", updated.Note)
	}

	missing, err := userRepo.ListFundHoldingsMissingConfirmation(context.Background())
	if err != nil {
		t.Fatalf("ListFundHoldingsMissingConfirmation() error = %v", err)
	}
	for _, item := range missing {
		if item.ID == holding.ID {
			t.Fatalf("manual correction should not be treated as missing confirmation: %+v", item)
		}
	}
}

func TestUserPreferenceServiceUpdateFundHoldingClearsManualConfirmation(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        "2026-03-30",
		NetAssetVal: decimal.RequireFromString("1.2500"),
		DailyReturn: decimal.RequireFromString("0.50"),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	if _, err := service.UpdateFundHolding(context.Background(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "52000",
		Shares:           "41000.123456",
		ConfirmedNav:     "1.2683",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:59:00+08:00",
	}); err != nil {
		t.Fatalf("manual UpdateFundHolding() error = %v", err)
	}

	updated, err := service.UpdateFundHolding(context.Background(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:  "52000",
		TradeAt: "2026-03-30T14:59:00+08:00",
		Note:    "恢复自动确认",
	})
	if err != nil {
		t.Fatalf("clear UpdateFundHolding() error = %v", err)
	}

	if updated.ManualConfirmation {
		t.Fatalf("updated manual_confirmation = true, want false")
	}
	if updated.Shares != "41600" {
		t.Fatalf("updated shares = %q, want 41600", updated.Shares)
	}
	if updated.ConfirmedNav != "1.25" {
		t.Fatalf("updated nav = %q, want 1.25", updated.ConfirmedNav)
	}
	if updated.ConfirmedNavDate != "2026-03-30" {
		t.Fatalf("updated nav date = %q, want 2026-03-30", updated.ConfirmedNavDate)
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
	if len(holdings.Items) != 1 {
		t.Fatalf("holdings len = %d, want 1", len(holdings.Items))
	}
	if holdings.Items[0].ID != holding.ID {
		t.Fatalf("holding id = %s, want %s", holdings.Items[0].ID, holding.ID)
	}
	if holdings.Items[0].ActualDate != expectedDate {
		t.Fatalf("actual date = %s, want %s", holdings.Items[0].ActualDate, expectedDate)
	}
	if holdings.Items[0].ActualDailyReturn != "1.2345" {
		t.Fatalf("actual daily return = %s, want 1.2345", holdings.Items[0].ActualDailyReturn)
	}
}

func TestUserPreferenceServiceCreateFundHoldingStoresConfirmedNavAndSharesWhenHistoryExists(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)

	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        "2026-03-30",
		NetAssetVal: decimal.RequireFromString("1.2500"),
		AccumVal:    decimal.RequireFromString("1.2500"),
		DailyReturn: decimal.RequireFromString("0.1000"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}

	holding, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	if holding.ConfirmedNav != "1.25" {
		t.Fatalf("confirmed nav = %s, want 1.25", holding.ConfirmedNav)
	}
	if holding.ConfirmedNavDate != "2026-03-30" {
		t.Fatalf("confirmed nav date = %s, want 2026-03-30", holding.ConfirmedNavDate)
	}
	if holding.Shares != "40000" {
		t.Fatalf("shares = %s, want 40000", holding.Shares)
	}
}

func TestUserPreferenceServiceListFundHoldingsComputesRealMetricsAndSummary(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	expectedDate := expectedOfficialHistoryDate(time.Now())

	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        "2026-03-30",
		NetAssetVal: decimal.RequireFromString("1.2500"),
		AccumVal:    decimal.RequireFromString("1.2500"),
		DailyReturn: decimal.RequireFromString("0.1000"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}
	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        expectedDate,
		NetAssetVal: decimal.RequireFromString("1.5000"),
		AccumVal:    decimal.RequireFromString("1.7000"),
		DailyReturn: decimal.RequireFromString("2.0000"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}

	if _, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓"); err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	holdings, err := service.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings.Items) != 1 {
		t.Fatalf("holdings len = %d, want 1", len(holdings.Items))
	}
	if len(holdings.Aggregates) != 1 {
		t.Fatalf("aggregates len = %d, want 1", len(holdings.Aggregates))
	}

	item := holdings.Items[0]
	if !item.RealMetricsReady {
		t.Fatalf("expected real metrics ready, got %+v", item)
	}
	if item.CurrentMarketValue != "60000.00" {
		t.Fatalf("current market value = %s, want 60000.00", item.CurrentMarketValue)
	}
	if item.TodayProfit != "1176.47" {
		t.Fatalf("today profit = %s, want 1176.47", item.TodayProfit)
	}
	if item.TodayChangePercent != "2" {
		t.Fatalf("today change percent = %s, want 2", item.TodayChangePercent)
	}

	if !holdings.Summary.RealMetricsReady {
		t.Fatalf("expected summary real metrics ready, got %+v", holdings.Summary)
	}
	if holdings.Summary.TotalPrincipal.String() != "50000" {
		t.Fatalf("total principal = %s, want 50000", holdings.Summary.TotalPrincipal.String())
	}
	if holdings.Summary.TotalCurrentMarketValue != "60000.00" {
		t.Fatalf("total current market value = %s, want 60000.00", holdings.Summary.TotalCurrentMarketValue)
	}
	if holdings.Summary.TotalTodayProfit != "1176.47" {
		t.Fatalf("total today profit = %s, want 1176.47", holdings.Summary.TotalTodayProfit)
	}
	if holdings.Summary.TotalTodayChangePercent != "2" {
		t.Fatalf("total today change percent = %s, want 2", holdings.Summary.TotalTodayChangePercent)
	}
	if holdings.Summary.MetricsScope != "full" {
		t.Fatalf("metrics scope = %s, want full", holdings.Summary.MetricsScope)
	}
	if holdings.Summary.IncompleteHoldingsCount != 0 {
		t.Fatalf("incomplete holdings count = %d, want 0", holdings.Summary.IncompleteHoldingsCount)
	}
	if holdings.Summary.ReadyPrincipal != "50000.00" {
		t.Fatalf("ready principal = %s, want 50000.00", holdings.Summary.ReadyPrincipal)
	}

	aggregatesByFundID := make(map[string]domain.UserFundHoldingAggregate, len(holdings.Aggregates))
	for _, aggregate := range holdings.Aggregates {
		aggregatesByFundID[aggregate.FundID] = aggregate
	}

	aggregate, ok := aggregatesByFundID["005827"]
	if !ok {
		t.Fatalf("missing aggregate for 005827: %+v", holdings.Aggregates)
	}
	if aggregate.FundID != "005827" {
		t.Fatalf("aggregate fund id = %s, want 005827", aggregate.FundID)
	}
	if aggregate.HoldingCount != 1 {
		t.Fatalf("aggregate holding count = %d, want 1", aggregate.HoldingCount)
	}
	if aggregate.ConfirmedHoldingCount != 1 {
		t.Fatalf("aggregate confirmed holding count = %d, want 1", aggregate.ConfirmedHoldingCount)
	}
	if aggregate.MetricsScope != "full" {
		t.Fatalf("aggregate metrics scope = %s, want full", aggregate.MetricsScope)
	}
	if !aggregate.RealMetricsReady {
		t.Fatalf("expected aggregate real metrics ready, got %+v", aggregate)
	}
	if aggregate.OfficialCurrentMarketValue != "60000.00" {
		t.Fatalf("aggregate current market value = %s, want 60000.00", aggregate.OfficialCurrentMarketValue)
	}
	if aggregate.OfficialTodayProfit != "1176.47" {
		t.Fatalf("aggregate today profit = %s, want 1176.47", aggregate.OfficialTodayProfit)
	}
	if aggregate.OfficialTodayChangePercent != "2" {
		t.Fatalf("aggregate today change percent = %s, want 2", aggregate.OfficialTodayChangePercent)
	}
	if aggregate.ConfirmedShares != "40000.000000" {
		t.Fatalf("aggregate confirmed shares = %s, want 40000.000000", aggregate.ConfirmedShares)
	}
}

func TestUserPreferenceServiceListFundHoldingsComputesPartialOfficialSummary(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	expectedDate := expectedOfficialHistoryDate(time.Now())

	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        "2026-03-30",
		NetAssetVal: decimal.RequireFromString("1.2500"),
		AccumVal:    decimal.RequireFromString("1.2500"),
		DailyReturn: decimal.RequireFromString("0.1000"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}
	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        expectedDate,
		NetAssetVal: decimal.RequireFromString("1.5000"),
		AccumVal:    decimal.RequireFromString("1.7000"),
		DailyReturn: decimal.RequireFromString("2.0000"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}

	if _, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓"); err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	if _, err := service.CreateFundHolding(context.Background(), "user-1", "003095", "28000", "2026-03-30T14:30:00+08:00", "主题仓位"); err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	holdings, err := service.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings.Items) != 2 {
		t.Fatalf("holdings len = %d, want 2", len(holdings.Items))
	}
	if len(holdings.Aggregates) != 2 {
		t.Fatalf("aggregates len = %d, want 2", len(holdings.Aggregates))
	}

	if holdings.Summary.RealMetricsReady {
		t.Fatalf("expected summary to remain partial, got %+v", holdings.Summary)
	}
	if holdings.Summary.MetricsScope != "partial" {
		t.Fatalf("metrics scope = %s, want partial", holdings.Summary.MetricsScope)
	}
	if holdings.Summary.RealMetricsReadyCount != 1 {
		t.Fatalf("ready count = %d, want 1", holdings.Summary.RealMetricsReadyCount)
	}
	if holdings.Summary.IncompleteHoldingsCount != 1 {
		t.Fatalf("incomplete holdings count = %d, want 1", holdings.Summary.IncompleteHoldingsCount)
	}
	if holdings.Summary.TotalCurrentMarketValue != "60000.00" {
		t.Fatalf("total current market value = %s, want 60000.00", holdings.Summary.TotalCurrentMarketValue)
	}
	if holdings.Summary.TotalTodayProfit != "1176.47" {
		t.Fatalf("total today profit = %s, want 1176.47", holdings.Summary.TotalTodayProfit)
	}
	if holdings.Summary.TotalTodayChangePercent != "2" {
		t.Fatalf("total today change percent = %s, want 2", holdings.Summary.TotalTodayChangePercent)
	}
	if holdings.Summary.ReadyPrincipal != "50000.00" {
		t.Fatalf("ready principal = %s, want 50000.00", holdings.Summary.ReadyPrincipal)
	}
	if !strings.Contains(holdings.Summary.Message, "1/2") {
		t.Fatalf("message = %q, want partial coverage hint", holdings.Summary.Message)
	}

	aggregatesByFundID := make(map[string]domain.UserFundHoldingAggregate, len(holdings.Aggregates))
	for _, item := range holdings.Aggregates {
		aggregatesByFundID[item.FundID] = item
	}

	aggregate, ok := aggregatesByFundID["005827"]
	if !ok {
		t.Fatalf("missing aggregate for 005827: %+v", holdings.Aggregates)
	}
	if aggregate.FundID != "005827" {
		t.Fatalf("aggregate fund id = %s, want 005827", aggregate.FundID)
	}
	if aggregate.MetricsScope != "full" {
		t.Fatalf("aggregate metrics scope = %s, want full", aggregate.MetricsScope)
	}
	if !aggregate.RealMetricsReady {
		t.Fatalf("expected aggregate 005827 full ready, got %+v", aggregate)
	}

	partialAggregate, ok := aggregatesByFundID["003095"]
	if !ok {
		t.Fatalf("missing aggregate for 003095: %+v", holdings.Aggregates)
	}
	if partialAggregate.FundID != "003095" {
		t.Fatalf("aggregate fund id = %s, want 003095", partialAggregate.FundID)
	}
	if partialAggregate.MetricsScope != "none" {
		t.Fatalf("aggregate metrics scope = %s, want none", partialAggregate.MetricsScope)
	}
	if partialAggregate.RealMetricsReady {
		t.Fatalf("expected aggregate 003095 not ready, got %+v", partialAggregate)
	}
	if partialAggregate.Message == "" {
		t.Fatalf("expected aggregate message for 003095")
	}
}

func TestUserPreferenceServiceListFundHoldingsAggregatesMultipleLotsOfSameFund(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	expectedDate := expectedOfficialHistoryDate(time.Now())

	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        "2026-03-30",
		NetAssetVal: decimal.RequireFromString("1.2500"),
		AccumVal:    decimal.RequireFromString("1.2500"),
		DailyReturn: decimal.RequireFromString("0.1000"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}
	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        expectedDate,
		NetAssetVal: decimal.RequireFromString("1.5000"),
		AccumVal:    decimal.RequireFromString("1.7000"),
		DailyReturn: decimal.RequireFromString("2.0000"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}

	if _, err := service.CreateFundHolding(context.Background(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "第一笔"); err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	if err := userRepo.SaveFundHolding(context.Background(), &domain.UserFundHolding{
		ID:        "ufh_manual",
		UserID:    "user-1",
		FundID:    "005827",
		Amount:    decimal.RequireFromString("20000"),
		TradeAt:   "2026-04-02T15:01:00+08:00",
		AsOfDate:  "2026-04-03",
		Note:      "第二笔",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHolding() error = %v", err)
	}

	holdings, err := service.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings.Aggregates) != 1 {
		t.Fatalf("aggregates len = %d, want 1", len(holdings.Aggregates))
	}

	aggregate := holdings.Aggregates[0]
	if aggregate.HoldingCount != 2 {
		t.Fatalf("aggregate holding count = %d, want 2", aggregate.HoldingCount)
	}
	if aggregate.ConfirmedHoldingCount != 1 {
		t.Fatalf("aggregate confirmed holding count = %d, want 1", aggregate.ConfirmedHoldingCount)
	}
	if aggregate.RealMetricsReadyCount != 1 {
		t.Fatalf("aggregate real metrics ready count = %d, want 1", aggregate.RealMetricsReadyCount)
	}
	if aggregate.IncompleteHoldingsCount != 1 {
		t.Fatalf("aggregate incomplete holdings count = %d, want 1", aggregate.IncompleteHoldingsCount)
	}
	if aggregate.TotalPrincipal.String() != "70000" {
		t.Fatalf("aggregate total principal = %s, want 70000", aggregate.TotalPrincipal.String())
	}
	if aggregate.ConfirmedPrincipal != "50000.00" {
		t.Fatalf("aggregate confirmed principal = %s, want 50000.00", aggregate.ConfirmedPrincipal)
	}
	if aggregate.ReadyPrincipal != "50000.00" {
		t.Fatalf("aggregate ready principal = %s, want 50000.00", aggregate.ReadyPrincipal)
	}
	if aggregate.MetricsScope != "partial" {
		t.Fatalf("aggregate metrics scope = %s, want partial", aggregate.MetricsScope)
	}
	if aggregate.RealMetricsReady {
		t.Fatalf("expected aggregate partial not full ready, got %+v", aggregate)
	}
	if !strings.Contains(aggregate.Message, "1/2") {
		t.Fatalf("aggregate message = %q, want partial coverage hint", aggregate.Message)
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
	fundRepo.getHistoryLookupCalls = 0

	holdings, err := service.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings.Items) != 2 {
		t.Fatalf("holdings len = %d, want 2", len(holdings.Items))
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
	if fundRepo.getHistoryLookupCalls != 1 {
		t.Fatalf("GetFundHistoriesByLookupKeys() calls = %d, want 1", fundRepo.getHistoryLookupCalls)
	}
}
