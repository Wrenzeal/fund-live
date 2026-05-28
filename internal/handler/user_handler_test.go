package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/middleware"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/gin-gonic/gin"
)

type holdingTransactionsEnvelope struct {
	Success bool                                `json:"success"`
	Data    []domain.UserFundHoldingTransaction `json:"data"`
	Error   *APIError                           `json:"error"`
}

type holdingTransactionEnvelope struct {
	Success bool                              `json:"success"`
	Data    domain.UserFundHoldingTransaction `json:"data"`
	Error   *APIError                         `json:"error"`
}

type holdingTransactionRollbackPreviewEnvelope struct {
	Success bool                                             `json:"success"`
	Data    domain.UserFundHoldingTransactionRollbackPreview `json:"data"`
	Error   *APIError                                        `json:"error"`
}

type holdingTransactionRollbackApplyEnvelope struct {
	Success bool                                                 `json:"success"`
	Data    domain.UserFundHoldingTransactionRollbackApplyResult `json:"data"`
	Error   *APIError                                            `json:"error"`
}

type holdingTransactionDetailEnvelope struct {
	Success bool                                    `json:"success"`
	Data    domain.UserFundHoldingTransactionDetail `json:"data"`
	Error   *APIError                               `json:"error"`
}

type fundHoldingsBatchEnvelope struct {
	Success bool                                    `json:"success"`
	Data    domain.UserFundHoldingBatchCreateResult `json:"data"`
	Error   *APIError                               `json:"error"`
}

type fundHoldingEnvelope struct {
	Success bool                         `json:"success"`
	Data    domain.UserFundHoldingDetail `json:"data"`
	Error   *APIError                    `json:"error"`
}

func TestUserHandlerSellFundHolding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	userPreferenceService := service.NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	holding, err := userPreferenceService.CreateFundHolding(t.Context(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	userHandler := NewUserHandler(userPreferenceService, userRepo, domain.QuoteSourceSina)
	router := gin.New()
	router.POST("/api/v1/user/holdings/:holdingId/sell", func(c *gin.Context) {
		c.Set("current_user", &domain.User{ID: "user-1"})
		c.Set("current_session", &domain.UserSession{ID: "session-1", UserID: "user-1"})
	}, middleware.RequireAuth(nil, "session"), userHandler.SellFundHolding)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/holdings/"+holding.ID+"/sell", strings.NewReader(`{
		"amount":"10000",
		"trade_at":"2026-04-01T14:50:00+08:00",
		"note":"减仓"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var response struct {
		Success bool                         `json:"success"`
		Data    domain.UserFundHoldingDetail `json:"data"`
		Error   *APIError                    `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false: %+v", response.Error)
	}
	if response.Data.Amount.String() != "40000" {
		t.Fatalf("remaining amount = %s, want 40000", response.Data.Amount.String())
	}

	transactions, err := userPreferenceService.ListFundHoldingTransactions(t.Context(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListFundHoldingTransactions() error = %v", err)
	}
	if transactions[0].Type != domain.UserFundHoldingTransactionSell {
		t.Fatalf("latest transaction = %s, want sell", transactions[0].Type)
	}
}

func TestUserHandlerSellAllFundHolding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	userPreferenceService := service.NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	holding, err := userPreferenceService.CreateFundHolding(t.Context(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	userHandler := NewUserHandler(userPreferenceService, userRepo, domain.QuoteSourceSina)
	router := gin.New()
	router.POST("/api/v1/user/holdings/:holdingId/sell", func(c *gin.Context) {
		c.Set("current_user", &domain.User{ID: "user-1"})
		c.Set("current_session", &domain.UserSession{ID: "session-1", UserID: "user-1"})
	}, middleware.RequireAuth(nil, "session"), userHandler.SellFundHolding)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/holdings/"+holding.ID+"/sell", strings.NewReader(`{
		"sell_all":true,
		"trade_at":"2026-04-01T14:50:00+08:00",
		"note":"全部赎回"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var response struct {
		Success bool                          `json:"success"`
		Data    *domain.UserFundHoldingDetail `json:"data"`
		Error   *APIError                     `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false: %+v", response.Error)
	}
	if response.Data != nil {
		t.Fatalf("response data = %+v, want nil after sell all", response.Data)
	}

	holdings, err := userPreferenceService.ListFundHoldings(t.Context(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings.Items) != 0 {
		t.Fatalf("holdings len = %d, want 0", len(holdings.Items))
	}

	transactions, err := userPreferenceService.ListFundHoldingTransactions(t.Context(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListFundHoldingTransactions() error = %v", err)
	}
	if transactions[0].Type != domain.UserFundHoldingTransactionSell || transactions[0].Metadata["sell_all"] != "true" {
		t.Fatalf("latest transaction = %+v, want sell_all sell", transactions[0])
	}
}

func TestUserHandlerRecordFundHoldingDividend(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	userPreferenceService := service.NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	holding, err := userPreferenceService.CreateFundHolding(t.Context(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	if _, err := userPreferenceService.UpdateFundHolding(t.Context(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "50000",
		Shares:           "40000",
		ConfirmedNav:     "1.25",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:30:00+08:00",
	}); err != nil {
		t.Fatalf("UpdateFundHolding() error = %v", err)
	}

	userHandler := NewUserHandler(userPreferenceService, userRepo, domain.QuoteSourceSina)
	router := gin.New()
	router.POST("/api/v1/user/holdings/:holdingId/dividend", func(c *gin.Context) {
		c.Set("current_user", &domain.User{ID: "user-1"})
		c.Set("current_session", &domain.UserSession{ID: "session-1", UserID: "user-1"})
	}, middleware.RequireAuth(nil, "session"), userHandler.RecordFundHoldingDividend)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/holdings/"+holding.ID+"/dividend", strings.NewReader(`{
		"amount":"125",
		"shares":"100",
		"trade_at":"2026-04-03T14:30:00+08:00",
		"note":"红利再投",
		"reinvest":true,
		"source_platform":"wechat"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var response fundHoldingEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false: %+v", response.Error)
	}
	if response.Data.Shares != "40100" {
		t.Fatalf("response shares = %s, want 40100", response.Data.Shares)
	}
	if response.Data.SourcePlatform != "wechat" || response.Data.SourceLabel != "微信" {
		t.Fatalf("response source = %s/%s, want wechat/微信", response.Data.SourcePlatform, response.Data.SourceLabel)
	}

	transactions, err := userPreferenceService.ListFundHoldingTransactionsFiltered(t.Context(), "user-1", domain.UserFundHoldingTransactionFilter{
		Types: []domain.UserFundHoldingTransactionType{domain.UserFundHoldingTransactionDividend},
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered() error = %v", err)
	}
	if len(transactions) != 1 || transactions[0].Metadata["reinvest"] != "true" || transactions[0].SourcePlatform != "wechat" {
		t.Fatalf("dividend transactions = %+v, want one reinvest wechat tx", transactions)
	}
}

func TestUserHandlerAdjustFundHoldingShares(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	userPreferenceService := service.NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	holding, err := userPreferenceService.CreateFundHolding(t.Context(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	if _, err := userPreferenceService.UpdateFundHolding(t.Context(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "50000",
		Shares:           "40000",
		ConfirmedNav:     "1.25",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:30:00+08:00",
	}); err != nil {
		t.Fatalf("UpdateFundHolding() error = %v", err)
	}

	userHandler := NewUserHandler(userPreferenceService, userRepo, domain.QuoteSourceSina)
	router := gin.New()
	router.POST("/api/v1/user/holdings/:holdingId/adjustment", func(c *gin.Context) {
		c.Set("current_user", &domain.User{ID: "user-1"})
		c.Set("current_session", &domain.UserSession{ID: "session-1", UserID: "user-1"})
	}, middleware.RequireAuth(nil, "session"), userHandler.AdjustFundHoldingShares)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/holdings/"+holding.ID+"/adjustment", strings.NewReader(`{
		"target_shares":"41000",
		"confirmed_nav":"1.22",
		"confirmed_nav_date":"2026-04-03",
		"trade_at":"2026-04-03T14:30:00+08:00",
		"note":"平台迁移份额调整",
		"source_platform":"eastmoney"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var response fundHoldingEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false: %+v", response.Error)
	}
	if response.Data.Shares != "41000" || response.Data.ConfirmedNav != "1.22" || response.Data.ConfirmedNavDate != "2026-04-03" {
		t.Fatalf("response data = %+v, want adjusted shares/nav/date", response.Data)
	}
	if response.Data.SourcePlatform != "eastmoney" || response.Data.SourceLabel != "天天基金" {
		t.Fatalf("response source = %s/%s, want eastmoney/天天基金", response.Data.SourcePlatform, response.Data.SourceLabel)
	}

	invalidReq := httptest.NewRequest(http.MethodPost, "/api/v1/user/holdings/"+holding.ID+"/adjustment", strings.NewReader(`{
		"target_shares":"41000",
		"shares_delta":"100"
	}`))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidRec := httptest.NewRecorder()
	router.ServeHTTP(invalidRec, invalidReq)
	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, want 400; body=%s", invalidRec.Code, invalidRec.Body.String())
	}

	var invalidResponse fundHoldingEnvelope
	if err := json.Unmarshal(invalidRec.Body.Bytes(), &invalidResponse); err != nil {
		t.Fatalf("decode invalid response: %v", err)
	}
	if invalidResponse.Error == nil || invalidResponse.Error.Code != "INVALID_HOLDING_ADJUSTMENT" {
		t.Fatalf("invalid error = %+v, want INVALID_HOLDING_ADJUSTMENT", invalidResponse.Error)
	}
}

func TestUserHandlerListFundHoldingTransactions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	userPreferenceService := service.NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	holding, err := userPreferenceService.CreateFundHolding(t.Context(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	if _, err := userPreferenceService.UpdateFundHolding(t.Context(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "52000",
		Shares:           "41000.123456",
		ConfirmedNav:     "1.2683",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:59:00+08:00",
		SourcePlatform:   "alipay",
	}); err != nil {
		t.Fatalf("UpdateFundHolding() error = %v", err)
	}

	userHandler := NewUserHandler(userPreferenceService, userRepo, domain.QuoteSourceSina)
	router := gin.New()
	router.GET("/api/v1/user/holdings/transactions", func(c *gin.Context) {
		c.Set("current_user", &domain.User{ID: "user-1"})
		c.Set("current_session", &domain.UserSession{ID: "session-1", UserID: "user-1"})
	}, middleware.RequireAuth(nil, "session"), userHandler.ListFundHoldingTransactions)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/holdings/transactions?limit=1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var response holdingTransactionsEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false: %+v", response.Error)
	}
	if len(response.Data) != 1 {
		t.Fatalf("transactions len = %d, want 1", len(response.Data))
	}
	if response.Data[0].Type != domain.UserFundHoldingTransactionCorrection {
		t.Fatalf("transaction type = %s, want correction", response.Data[0].Type)
	}
	if response.Data[0].Fund == nil || response.Data[0].Fund.ID != "005827" {
		t.Fatalf("transaction fund not enriched: %+v", response.Data[0].Fund)
	}
}

func TestUserHandlerListFundHoldingTransactionsAppliesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	userPreferenceService := service.NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	holding, err := userPreferenceService.CreateFundHolding(t.Context(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding(005827) error = %v", err)
	}
	if _, err := userPreferenceService.CreateFundHolding(t.Context(), "user-1", "003095", "12000", "2026-03-30T14:30:00+08:00", "医疗仓"); err != nil {
		t.Fatalf("CreateFundHolding(003095) error = %v", err)
	}
	if _, err := userPreferenceService.UpdateFundHolding(t.Context(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:           "52000",
		Shares:           "41000.123456",
		ConfirmedNav:     "1.2683",
		ConfirmedNavDate: "2026-03-30",
		TradeAt:          "2026-03-30T14:59:00+08:00",
		SourcePlatform:   "alipay",
	}); err != nil {
		t.Fatalf("UpdateFundHolding() error = %v", err)
	}
	transactions, err := userPreferenceService.ListFundHoldingTransactions(t.Context(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListFundHoldingTransactions() error = %v", err)
	}
	for _, transaction := range transactions {
		if transaction.FundID == "003095" {
			if _, err := userPreferenceService.VoidFundHoldingTransaction(t.Context(), "user-1", transaction.ID, "筛选测试"); err != nil {
				t.Fatalf("VoidFundHoldingTransaction() error = %v", err)
			}
			break
		}
	}

	userHandler := NewUserHandler(userPreferenceService, userRepo, domain.QuoteSourceSina)
	router := gin.New()
	router.GET("/api/v1/user/holdings/transactions", func(c *gin.Context) {
		c.Set("current_user", &domain.User{ID: "user-1"})
		c.Set("current_session", &domain.UserSession{ID: "session-1", UserID: "user-1"})
	}, middleware.RequireAuth(nil, "session"), userHandler.ListFundHoldingTransactions)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/holdings/transactions?fund_id=005827&type=correction&voided=false&source_platform=alipay&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var response holdingTransactionsEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false: %+v", response.Error)
	}
	if len(response.Data) != 1 {
		t.Fatalf("transactions len = %d, want 1; data=%+v", len(response.Data), response.Data)
	}
	if response.Data[0].FundID != "005827" || response.Data[0].Type != domain.UserFundHoldingTransactionCorrection || response.Data[0].Voided {
		t.Fatalf("filtered transaction = %+v, want active 005827 correction", response.Data[0])
	}
	if response.Data[0].SourcePlatform != "alipay" || response.Data[0].SourceLabel != "支付宝" {
		t.Fatalf("filtered source = %s/%s, want alipay/支付宝", response.Data[0].SourcePlatform, response.Data[0].SourceLabel)
	}

	voidedReq := httptest.NewRequest(http.MethodGet, "/api/v1/user/holdings/transactions?voided=true", nil)
	voidedRec := httptest.NewRecorder()
	router.ServeHTTP(voidedRec, voidedReq)
	if voidedRec.Code != http.StatusOK {
		t.Fatalf("voided status = %d, want 200; body=%s", voidedRec.Code, voidedRec.Body.String())
	}
	var voidedResponse holdingTransactionsEnvelope
	if err := json.Unmarshal(voidedRec.Body.Bytes(), &voidedResponse); err != nil {
		t.Fatalf("decode voided response: %v", err)
	}
	if len(voidedResponse.Data) != 1 || !voidedResponse.Data[0].Voided || voidedResponse.Data[0].FundID != "003095" {
		t.Fatalf("voided response = %+v, want one voided 003095 transaction", voidedResponse.Data)
	}

	keywordReq := httptest.NewRequest(http.MethodGet, "/api/v1/user/holdings/transactions?keyword=005827&start_date=2000-01-01&end_date=2999-12-31&offset=1&limit=1", nil)
	keywordRec := httptest.NewRecorder()
	router.ServeHTTP(keywordRec, keywordReq)
	if keywordRec.Code != http.StatusOK {
		t.Fatalf("keyword status = %d, want 200; body=%s", keywordRec.Code, keywordRec.Body.String())
	}
	var keywordResponse holdingTransactionsEnvelope
	if err := json.Unmarshal(keywordRec.Body.Bytes(), &keywordResponse); err != nil {
		t.Fatalf("decode keyword response: %v", err)
	}
	if len(keywordResponse.Data) != 1 || keywordResponse.Data[0].FundID != "005827" {
		t.Fatalf("keyword response = %+v, want second 005827 transaction", keywordResponse.Data)
	}
}

func TestUserHandlerUpdateFundHoldingRecordsSource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	userPreferenceService := service.NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	holding, err := userPreferenceService.CreateFundHolding(t.Context(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	userHandler := NewUserHandler(userPreferenceService, userRepo, domain.QuoteSourceSina)
	router := gin.New()
	router.PUT("/api/v1/user/holdings/:holdingId", func(c *gin.Context) {
		c.Set("current_user", &domain.User{ID: "user-1"})
		c.Set("current_session", &domain.UserSession{ID: "session-1", UserID: "user-1"})
	}, middleware.RequireAuth(nil, "session"), userHandler.UpdateFundHolding)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/user/holdings/"+holding.ID, strings.NewReader(`{
		"amount":"52000",
		"shares":"41000.123456",
		"confirmed_nav":"1.2683",
		"confirmed_nav_date":"2026-03-30",
		"trade_at":"2026-03-30T14:59:00+08:00",
		"note":"按微信校正",
		"source_platform":"wechat"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var response fundHoldingEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false: %+v", response.Error)
	}
	if response.Data.SourcePlatform != "wechat" || response.Data.SourceLabel != "微信" {
		t.Fatalf("response source = %s/%s, want wechat/微信", response.Data.SourcePlatform, response.Data.SourceLabel)
	}

	transactions, err := userPreferenceService.ListFundHoldingTransactionsFiltered(t.Context(), "user-1", domain.UserFundHoldingTransactionFilter{
		SourcePlatform: "wechat",
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered() error = %v", err)
	}
	if len(transactions) != 1 || transactions[0].SourcePlatform != "wechat" {
		t.Fatalf("transactions = %+v, want one wechat correction", transactions)
	}
}

func TestUserHandlerUpdateFundHoldingRejectsInvalidSource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	userPreferenceService := service.NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	holding, err := userPreferenceService.CreateFundHolding(t.Context(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}

	userHandler := NewUserHandler(userPreferenceService, userRepo, domain.QuoteSourceSina)
	router := gin.New()
	router.PUT("/api/v1/user/holdings/:holdingId", func(c *gin.Context) {
		c.Set("current_user", &domain.User{ID: "user-1"})
		c.Set("current_session", &domain.UserSession{ID: "session-1", UserID: "user-1"})
	}, middleware.RequireAuth(nil, "session"), userHandler.UpdateFundHolding)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/user/holdings/"+holding.ID, strings.NewReader(`{
		"amount":"52000",
		"shares":"41000.123456",
		"confirmed_nav":"1.2683",
		"confirmed_nav_date":"2026-03-30",
		"trade_at":"2026-03-30T14:59:00+08:00",
		"source_platform":"unsupported-platform"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}

	var response fundHoldingEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error == nil || response.Error.Code != "INVALID_HOLDING_SOURCE" {
		t.Fatalf("error = %+v, want INVALID_HOLDING_SOURCE", response.Error)
	}
}

func TestUserHandlerListFundHoldingTransactionsRejectsInvalidLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	userPreferenceService := service.NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	userHandler := NewUserHandler(userPreferenceService, userRepo, domain.QuoteSourceSina)

	router := gin.New()
	router.GET("/api/v1/user/holdings/transactions", func(c *gin.Context) {
		c.Set("current_user", &domain.User{ID: "user-1"})
		c.Set("current_session", &domain.UserSession{ID: "session-1", UserID: "user-1"})
	}, middleware.RequireAuth(nil, "session"), userHandler.ListFundHoldingTransactions)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/holdings/transactions?limit=bad", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}

	var response holdingTransactionsEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error == nil || response.Error.Code != "INVALID_LIMIT" {
		t.Fatalf("error = %+v, want INVALID_LIMIT", response.Error)
	}
}

func TestUserHandlerListFundHoldingTransactionsRejectsInvalidVoidedFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	userPreferenceService := service.NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	userHandler := NewUserHandler(userPreferenceService, userRepo, domain.QuoteSourceSina)

	router := gin.New()
	router.GET("/api/v1/user/holdings/transactions", func(c *gin.Context) {
		c.Set("current_user", &domain.User{ID: "user-1"})
		c.Set("current_session", &domain.UserSession{ID: "session-1", UserID: "user-1"})
	}, middleware.RequireAuth(nil, "session"), userHandler.ListFundHoldingTransactions)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/holdings/transactions?voided=maybe", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}

	var response holdingTransactionsEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error == nil || response.Error.Code != "INVALID_VOIDED_FILTER" {
		t.Fatalf("error = %+v, want INVALID_VOIDED_FILTER", response.Error)
	}
}

func TestUserHandlerVoidFundHoldingTransaction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	userPreferenceService := service.NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	_, err := userPreferenceService.CreateFundHolding(t.Context(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	transactions, err := userPreferenceService.ListFundHoldingTransactions(t.Context(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListFundHoldingTransactions() error = %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("transactions len = %d, want 1", len(transactions))
	}

	userHandler := NewUserHandler(userPreferenceService, userRepo, domain.QuoteSourceSina)
	router := gin.New()
	router.POST("/api/v1/user/holdings/transactions/:transactionId/void", func(c *gin.Context) {
		c.Set("current_user", &domain.User{ID: "user-1"})
		c.Set("current_session", &domain.UserSession{ID: "session-1", UserID: "user-1"})
	}, middleware.RequireAuth(nil, "session"), userHandler.VoidFundHoldingTransaction)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/holdings/transactions/"+transactions[0].ID+"/void", strings.NewReader(`{
		"reason":"重复录入"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var response holdingTransactionEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false: %+v", response.Error)
	}
	if !response.Data.Voided || response.Data.VoidReason != "重复录入" || response.Data.VoidedAt == nil {
		t.Fatalf("voided transaction = %+v", response.Data)
	}
	if response.Data.Fund == nil || response.Data.Fund.ID != "005827" {
		t.Fatalf("voided transaction fund not enriched: %+v", response.Data.Fund)
	}

	repeatReq := httptest.NewRequest(http.MethodPost, "/api/v1/user/holdings/transactions/"+transactions[0].ID+"/void", strings.NewReader(`{
		"reason":"再次作废"
	}`))
	repeatReq.Header.Set("Content-Type", "application/json")
	repeatRec := httptest.NewRecorder()
	router.ServeHTTP(repeatRec, repeatReq)

	if repeatRec.Code != http.StatusConflict {
		t.Fatalf("repeat status = %d, want 409; body=%s", repeatRec.Code, repeatRec.Body.String())
	}
}

func TestUserHandlerPreviewFundHoldingTransactionRollback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	userPreferenceService := service.NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	holding, err := userPreferenceService.CreateFundHolding(t.Context(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	if _, err := userPreferenceService.SellFundHolding(t.Context(), "user-1", holding.ID, domain.SellFundHoldingInput{
		Amount:  "10000",
		TradeAt: "2026-04-01T14:50:00+08:00",
		Note:    "减仓",
	}); err != nil {
		t.Fatalf("SellFundHolding() error = %v", err)
	}
	transactions, err := userPreferenceService.ListFundHoldingTransactionsFiltered(t.Context(), "user-1", domain.UserFundHoldingTransactionFilter{
		Types: []domain.UserFundHoldingTransactionType{domain.UserFundHoldingTransactionSell},
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered() error = %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("sell transactions len = %d, want 1; all=%+v", len(transactions), transactions)
	}

	userHandler := NewUserHandler(userPreferenceService, userRepo, domain.QuoteSourceSina)
	router := gin.New()
	router.GET("/api/v1/user/holdings/transactions/:transactionId/rollback-preview", func(c *gin.Context) {
		c.Set("current_user", &domain.User{ID: "user-1"})
		c.Set("current_session", &domain.UserSession{ID: "session-1", UserID: "user-1"})
	}, middleware.RequireAuth(nil, "session"), userHandler.PreviewFundHoldingTransactionRollback)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/holdings/transactions/"+transactions[0].ID+"/rollback-preview", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var response holdingTransactionRollbackPreviewEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false: %+v", response.Error)
	}
	if !response.Data.PreviewOnly || !response.Data.CanApplyAutomatically {
		t.Fatalf("preview flags = %+v, want safe automatic apply enabled", response.Data)
	}
	if response.Data.Transaction.ID != transactions[0].ID || response.Data.Transaction.Fund == nil || response.Data.Transaction.Fund.ID != "005827" {
		t.Fatalf("preview transaction not enriched: %+v", response.Data.Transaction)
	}
	if len(response.Data.AffectedFields) == 0 {
		t.Fatalf("affected fields empty: %+v", response.Data)
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/api/v1/user/holdings/transactions/missing/rollback-preview", nil)
	missingRec := httptest.NewRecorder()
	router.ServeHTTP(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want 404; body=%s", missingRec.Code, missingRec.Body.String())
	}
}

func TestUserHandlerApplyFundHoldingTransactionRollback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	userPreferenceService := service.NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	holding, err := userPreferenceService.CreateFundHolding(t.Context(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	if _, err := userPreferenceService.UpdateFundHolding(t.Context(), "user-1", holding.ID, domain.UpdateFundHoldingInput{
		Amount:         "52000",
		TradeAt:        "2026-03-30T14:30:00+08:00",
		Note:           "平台校正",
		SourcePlatform: "alipay",
	}); err != nil {
		t.Fatalf("UpdateFundHolding() error = %v", err)
	}
	transactions, err := userPreferenceService.ListFundHoldingTransactionsFiltered(t.Context(), "user-1", domain.UserFundHoldingTransactionFilter{
		Types: []domain.UserFundHoldingTransactionType{domain.UserFundHoldingTransactionCorrection},
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered() error = %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("correction transactions len = %d, want 1; all=%+v", len(transactions), transactions)
	}

	userHandler := NewUserHandler(userPreferenceService, userRepo, domain.QuoteSourceSina)
	router := gin.New()
	router.POST("/api/v1/user/holdings/transactions/:transactionId/rollback-apply", func(c *gin.Context) {
		c.Set("current_user", &domain.User{ID: "user-1"})
		c.Set("current_session", &domain.UserSession{ID: "session-1", UserID: "user-1"})
	}, middleware.RequireAuth(nil, "session"), userHandler.ApplyFundHoldingTransactionRollback)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/holdings/transactions/"+transactions[0].ID+"/rollback-apply", strings.NewReader(`{
		"reason":"校正录错，自动冲正"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var response holdingTransactionRollbackApplyEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false: %+v", response.Error)
	}
	if !response.Data.Applied || !response.Data.Transaction.Voided {
		t.Fatalf("apply result = %+v, want applied voided transaction", response.Data)
	}
	if response.Data.CurrentHolding == nil || response.Data.CurrentHolding.Amount.String() != "50000" {
		t.Fatalf("current holding = %+v, want amount 50000", response.Data.CurrentHolding)
	}

	repeatReq := httptest.NewRequest(http.MethodPost, "/api/v1/user/holdings/transactions/"+transactions[0].ID+"/rollback-apply", strings.NewReader(`{
		"reason":"重复冲正"
	}`))
	repeatReq.Header.Set("Content-Type", "application/json")
	repeatRec := httptest.NewRecorder()
	router.ServeHTTP(repeatRec, repeatReq)
	if repeatRec.Code != http.StatusConflict {
		t.Fatalf("repeat status = %d, want 409; body=%s", repeatRec.Code, repeatRec.Body.String())
	}
}

func TestUserHandlerCreateFundHoldingsBatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	userPreferenceService := service.NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	userHandler := NewUserHandler(userPreferenceService, userRepo, domain.QuoteSourceSina)

	router := gin.New()
	router.POST("/api/v1/user/holdings/batch", func(c *gin.Context) {
		c.Set("current_user", &domain.User{ID: "user-1"})
		c.Set("current_session", &domain.UserSession{ID: "session-1", UserID: "user-1"})
	}, middleware.RequireAuth(nil, "session"), userHandler.CreateFundHoldingsBatch)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/holdings/batch", strings.NewReader(`{
		"items":[
			{"fund_id":"005827","amount":"50000","trade_at":"2026-03-30","note":"支付宝迁移","source_platform":"alipay"},
			{"fund_id":"bad","amount":"100","trade_at":"2026-03-30","note":"错误行"}
		]
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}

	var response fundHoldingsBatchEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false: %+v", response.Error)
	}
	if response.Data.Total != 2 || response.Data.CreatedCount != 1 || response.Data.FailedCount != 1 {
		t.Fatalf("batch result = %+v, want 1 created and 1 failed", response.Data)
	}
	if len(response.Data.Created) != 1 || response.Data.Created[0].SourcePlatform != "alipay" {
		t.Fatalf("created rows = %+v, want alipay source", response.Data.Created)
	}
	if len(response.Data.Failed) != 1 || response.Data.Failed[0].Code != "FUND_NOT_FOUND" {
		t.Fatalf("failed rows = %+v, want FUND_NOT_FOUND", response.Data.Failed)
	}
}

func TestUserHandlerGetFundHoldingTransactionDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	userPreferenceService := service.NewUserPreferenceService(fundRepo, userRepo, userRepo, userRepo, userRepo)
	holding, err := userPreferenceService.CreateFundHolding(t.Context(), "user-1", "005827", "50000", "2026-03-30T14:30:00+08:00", "长期底仓")
	if err != nil {
		t.Fatalf("CreateFundHolding() error = %v", err)
	}
	if _, err := userPreferenceService.SellFundHolding(t.Context(), "user-1", holding.ID, domain.SellFundHoldingInput{
		Amount:  "10000",
		TradeAt: "2026-04-01T14:50:00+08:00",
		Note:    "减仓",
	}); err != nil {
		t.Fatalf("SellFundHolding() error = %v", err)
	}
	transactions, err := userPreferenceService.ListFundHoldingTransactionsFiltered(t.Context(), "user-1", domain.UserFundHoldingTransactionFilter{
		Types: []domain.UserFundHoldingTransactionType{domain.UserFundHoldingTransactionSell},
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("ListFundHoldingTransactionsFiltered() error = %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("sell transactions len = %d, want 1; all=%+v", len(transactions), transactions)
	}

	userHandler := NewUserHandler(userPreferenceService, userRepo, domain.QuoteSourceSina)
	router := gin.New()
	router.GET("/api/v1/user/holdings/transactions/:transactionId", func(c *gin.Context) {
		c.Set("current_user", &domain.User{ID: "user-1"})
		c.Set("current_session", &domain.UserSession{ID: "session-1", UserID: "user-1"})
	}, middleware.RequireAuth(nil, "session"), userHandler.GetFundHoldingTransactionDetail)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/holdings/transactions/"+transactions[0].ID, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var response holdingTransactionDetailEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false: %+v", response.Error)
	}
	if response.Data.Transaction.ID != transactions[0].ID {
		t.Fatalf("detail transaction id = %s, want %s", response.Data.Transaction.ID, transactions[0].ID)
	}
	if response.Data.RollbackPreview == nil || len(response.Data.RollbackPreview.AffectedFields) == 0 {
		t.Fatalf("rollback preview missing from detail: %+v", response.Data)
	}
	if response.Data.CurrentHolding == nil || response.Data.CurrentHolding.Amount.String() != "40000" {
		t.Fatalf("current holding = %+v, want 40000", response.Data.CurrentHolding)
	}
	if len(response.Data.ImpactChain) == 0 {
		t.Fatalf("impact chain empty: %+v", response.Data)
	}
}
