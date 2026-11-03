package handler

import (
	"encoding/json"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpeningPositionsBothBasesCreateConfirmedSnapshotNotBuy(t *testing.T) {
	for _, input := range []string{
		`{"fund_id":"014662","entry_type":"opening","amount":"900","shares":"500","snapshot_date":"2026-11-02"}`,
		`{"fund_id":"014662","entry_type":"opening","market_value":"1000","holding_profit":"100","snapshot_date":"2026-11-02"}`,
	} {
		t.Run(input, func(t *testing.T) {
			fundRepo := repository.NewMemoryFundRepository()
			repo := repository.NewMemoryUserRepository()
			_ = fundRepo.SaveFund(t.Context(), &domain.Fund{ID: "014662", Name: "测试黄金联接"})
			_ = fundRepo.SaveFundHistory(t.Context(), &domain.FundHistory{FundID: "014662", Date: "2026-11-02", NetAssetVal: decimal.NewFromInt(2), DailyReturn: decimal.NewFromInt(1)})
			svc := service.NewUserPreferenceService(fundRepo, repo, repo, repo, repo)
			handler := NewUserHandler(svc, repo, domain.QuoteSourceSina)
			router := gin.New()
			router.Use(func(c *gin.Context) { c.Set("current_user", &domain.User{ID: "owner"}) })
			router.POST("/holdings", handler.CreateFundHolding)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/holdings", strings.NewReader(input)))
			if response.Code != 201 {
				t.Fatalf("status=%d %s", response.Code, response.Body.String())
			}
			var payload struct {
				Data domain.UserFundHoldingDetail `json:"data"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Data.EntryType != "opening" || payload.Data.Shares != "500" || !payload.Data.Amount.Equal(decimal.NewFromInt(900)) || !payload.Data.RealMetricsReady || payload.Data.CurrentMarketValue != "1000.00" {
				t.Fatalf("snapshot=%+v", payload.Data)
			}
			holdings, _ := repo.ListFundHoldings(t.Context(), "owner")
			activities, _ := repo.ListFundHoldingTransactions(t.Context(), "owner", 20)
			if len(holdings) != 1 || holdings[0].TradeAt != "" || len(activities) != 1 || activities[0].Type != domain.UserFundHoldingTransactionOpening {
				t.Fatalf("holdings=%+v activities=%+v", holdings, activities)
			}
			if holdings[0].ConfirmedNav.GreaterThan(decimal.Zero) {
				t.Fatal("cost per share was misrepresented as official buy NAV")
			}
		})
	}
}

func TestOpeningMissingNAVAndInvalidCostDoNotCreatePartialHolding(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	repo := repository.NewMemoryUserRepository()
	_ = fundRepo.SaveFund(t.Context(), &domain.Fund{ID: "014662"})
	svc := service.NewUserPreferenceService(fundRepo, repo, repo, repo, repo)
	for _, input := range []domain.CreateFundHoldingInput{
		{FundID: "014662", EntryType: "opening", MarketValue: "1000", HoldingProfit: "100", SnapshotDate: "2026-11-02"},
		{FundID: "014662", EntryType: "opening", Amount: "-100", Shares: "500", SnapshotDate: "2026-11-02"},
		{FundID: "014662", EntryType: "opening", MarketValue: "1000", HoldingProfit: "1100", SnapshotDate: "2026-11-02"},
	} {
		if _, err := svc.CreateFundHoldingWithInput(t.Context(), "owner", input); err == nil {
			t.Fatal("invalid opening accepted")
		}
	}
	holdings, _ := repo.ListFundHoldings(t.Context(), "owner")
	activities, _ := repo.ListFundHoldingTransactions(t.Context(), "owner", 50)
	if len(holdings) != 0 || len(activities) != 0 {
		t.Fatal("failed opening left partial state")
	}
}

func TestHistoricalBuyUsesAvailableOlderOfficialNAV(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	repo := repository.NewMemoryUserRepository()
	_ = fundRepo.SaveFund(t.Context(), &domain.Fund{ID: "014662"})
	_ = fundRepo.SaveFundHistory(t.Context(), &domain.FundHistory{FundID: "014662", Date: "2025-03-03", NetAssetVal: decimal.NewFromInt(2)})
	_ = fundRepo.SaveFundHistory(t.Context(), &domain.FundHistory{FundID: "014662", Date: "2026-11-02", NetAssetVal: decimal.RequireFromString("2.2"), DailyReturn: decimal.NewFromInt(1)})
	svc := service.NewUserPreferenceService(fundRepo, repo, repo, repo, repo)
	created, err := svc.CreateFundHolding(t.Context(), "owner", "014662", "1000", "2025-03-03T14:00:00+08:00", "旧交易")
	if err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListFundHoldings(t.Context(), "owner")
	if err != nil {
		t.Fatal(err)
	}
	if created.Shares != "500" || len(list.Items) != 1 || !list.Items[0].RealMetricsReady || list.Items[0].CurrentMarketValue != "1100.00" || list.Items[0].ActualDate != "2026-11-02" {
		t.Fatalf("history remains unpriced: %+v", list.Items)
	}
}

func TestOpeningIdentitySurvivesHistoryDeletionAndSaleRollback(t *testing.T) {
	funds := repository.NewMemoryFundRepository()
	repo := repository.NewMemoryUserRepository()
	_ = funds.SaveFund(t.Context(), &domain.Fund{ID: "014662"})
	_ = funds.SaveFundHistory(t.Context(), &domain.FundHistory{FundID: "014662", Date: "2026-11-02", NetAssetVal: decimal.NewFromInt(2)})
	svc := service.NewUserPreferenceService(funds, repo, repo, repo, repo)
	opening, err := svc.CreateFundHoldingWithInput(t.Context(), "owner", domain.CreateFundHoldingInput{FundID: "014662", EntryType: "opening", Amount: "900", Shares: "500", SnapshotDate: "2026-11-02"})
	if err != nil {
		t.Fatal(err)
	}
	events, _ := repo.ListFundHoldingTransactions(t.Context(), "owner", 20)
	if err := svc.DeleteFundHoldingTransaction(t.Context(), "owner", events[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SellFundHolding(t.Context(), "owner", opening.ID, domain.SellFundHoldingInput{SellAll: true}); err != nil {
		t.Fatal(err)
	}
	events, _ = repo.ListFundHoldingTransactions(t.Context(), "owner", 20)
	if _, err := svc.ApplyFundHoldingTransactionRollback(t.Context(), "owner", events[0].ID, "撤回误录清仓"); err != nil {
		t.Fatal(err)
	}
	restored, _ := repo.GetFundHolding(t.Context(), "owner", opening.ID)
	if restored == nil || restored.EntryType != "opening" || restored.TradeAt != "" || !restored.Shares.Equal(decimal.NewFromInt(500)) {
		t.Fatalf("opening origin lost: %+v", restored)
	}
}

func TestOpeningLossDerivesCostWithoutTreatingMarketValueAsPrincipal(t *testing.T) {
	funds := repository.NewMemoryFundRepository()
	repo := repository.NewMemoryUserRepository()
	_ = funds.SaveFund(t.Context(), &domain.Fund{ID: "014662"})
	_ = funds.SaveFundHistory(t.Context(), &domain.FundHistory{FundID: "014662", Date: "2026-11-02", NetAssetVal: decimal.NewFromInt(2)})
	svc := service.NewUserPreferenceService(funds, repo, repo, repo, repo)
	position, err := svc.CreateFundHoldingWithInput(t.Context(), "owner", domain.CreateFundHoldingInput{FundID: "014662", EntryType: "opening", MarketValue: "900", HoldingProfit: "-100", SnapshotDate: "2026-11-02"})
	if err != nil {
		t.Fatal(err)
	}
	if !position.Amount.Equal(decimal.NewFromInt(1000)) || position.Shares != "450" || position.CurrentMarketValue != "900.00" {
		t.Fatalf("incorrect loss basis: %+v", position)
	}
}
