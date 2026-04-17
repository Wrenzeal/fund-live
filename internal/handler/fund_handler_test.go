package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/RomaticDOG/fund/internal/trading"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type stubTransientFundDataLoader struct {
	cachedCalls   int
	scheduleCalls int
	cacheHit      bool
	fund          *domain.Fund
	holdings      []domain.StockHolding
}

func (s *stubTransientFundDataLoader) PeekCachedFundData(fundID string) (*domain.Fund, []domain.StockHolding, bool) {
	s.cachedCalls++
	if !s.cacheHit {
		return nil, nil, false
	}
	return s.fund, s.holdings, true
}

func (s *stubTransientFundDataLoader) ScheduleEnsureFundData(fundID string) bool {
	s.scheduleCalls++
	return true
}

type stubHoldingsFallbackResolver struct {
	calls    int
	holdings []domain.StockHolding
	source   string
	err      error
}

func (s *stubHoldingsFallbackResolver) GetHoldingsWithFallback(ctx context.Context, fundID string, fundName string) ([]domain.StockHolding, string, error) {
	s.calls++
	return s.holdings, s.source, s.err
}

type fundResponseEnvelope struct {
	Success bool        `json:"success"`
	Data    domain.Fund `json:"data"`
	Meta    *APIMeta    `json:"meta,omitempty"`
}

type holdingsResponseEnvelope struct {
	Success bool `json:"success"`
	Data    struct {
		Fund     domain.Fund           `json:"fund"`
		Holdings []domain.StockHolding `json:"holdings"`
	} `json:"data"`
	Meta *APIMeta `json:"meta,omitempty"`
}

func TestGetFundHydratesMissingProfileFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	if err := fundRepo.SaveFund(context.Background(), &domain.Fund{
		ID:          "123456",
		Name:        "目录基金",
		Type:        "hybrid",
		Manager:     "",
		Company:     "",
		NetAssetVal: decimal.RequireFromString("1.0000"),
	}); err != nil {
		t.Fatalf("SaveFund() error = %v", err)
	}

	loader := &stubTransientFundDataLoader{
		cacheHit: true,
		fund: &domain.Fund{
			ID:          "123456",
			Name:        "目录基金",
			Type:        "hybrid",
			Manager:     "张三",
			Company:     "测试基金",
			NetAssetVal: decimal.RequireFromString("1.2345"),
		},
	}
	handler := &FundHandler{
		fundRepo:   fundRepo,
		dataLoader: loader,
	}

	router := gin.New()
	router.GET("/api/v1/fund/:id", handler.GetFund)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fund/123456", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response fundResponseEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false")
	}
	if response.Data.Manager != "张三" {
		t.Fatalf("manager = %q, want 张三", response.Data.Manager)
	}
	if response.Data.Company != "测试基金" {
		t.Fatalf("company = %q, want 测试基金", response.Data.Company)
	}
	if response.Meta == nil || response.Meta.CacheStatus != "warm_cache" {
		t.Fatalf("meta = %+v, want warm_cache", response.Meta)
	}
	if loader.cachedCalls != 1 {
		t.Fatalf("cached calls = %d, want 1", loader.cachedCalls)
	}
	if loader.scheduleCalls != 0 {
		t.Fatalf("schedule calls = %d, want 0", loader.scheduleCalls)
	}
}

func TestGetHoldingsHydratesMissingHoldings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	if err := fundRepo.SaveFund(context.Background(), &domain.Fund{
		ID:          "654321",
		Name:        "持仓缺失基金",
		Type:        "hybrid",
		Manager:     "",
		Company:     "",
		NetAssetVal: decimal.RequireFromString("1.0000"),
	}); err != nil {
		t.Fatalf("SaveFund() error = %v", err)
	}

	loader := &stubTransientFundDataLoader{
		cacheHit: true,
		fund: &domain.Fund{
			ID:          "654321",
			Name:        "持仓缺失基金",
			Type:        "hybrid",
			Manager:     "李四",
			Company:     "演示基金",
			NetAssetVal: decimal.RequireFromString("1.3456"),
		},
		holdings: []domain.StockHolding{
			{
				StockCode:    "600519",
				StockName:    "贵州茅台",
				Exchange:     domain.ExchangeSH,
				HoldingRatio: decimal.RequireFromString("9.90"),
			},
		},
	}
	handler := &FundHandler{
		fundRepo:   fundRepo,
		dataLoader: loader,
	}

	router := gin.New()
	router.GET("/api/v1/fund/:id/holdings", handler.GetHoldings)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fund/654321/holdings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response holdingsResponseEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false")
	}
	if response.Data.Fund.Manager != "李四" {
		t.Fatalf("manager = %q, want 李四", response.Data.Fund.Manager)
	}
	if len(response.Data.Holdings) != 1 {
		t.Fatalf("holdings len = %d, want 1", len(response.Data.Holdings))
	}
	if response.Data.Holdings[0].StockCode != "600519" {
		t.Fatalf("holding stock code = %q, want 600519", response.Data.Holdings[0].StockCode)
	}
	if response.Meta == nil || response.Meta.CacheStatus != "warm_cache" {
		t.Fatalf("meta = %+v, want warm_cache", response.Meta)
	}
	if loader.cachedCalls != 1 {
		t.Fatalf("cached calls = %d, want 1", loader.cachedCalls)
	}
	if loader.scheduleCalls != 0 {
		t.Fatalf("schedule calls = %d, want 0", loader.scheduleCalls)
	}
}

func TestGetHoldingsUsesResolverFallbackForFeederFund(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	if err := fundRepo.SaveFund(context.Background(), &domain.Fund{
		ID:          "023408",
		Name:        "示例ETF联接基金",
		Type:        "index",
		Manager:     "王五",
		Company:     "联接基金公司",
		NetAssetVal: decimal.RequireFromString("1.1111"),
	}); err != nil {
		t.Fatalf("SaveFund() error = %v", err)
	}

	resolver := &stubHoldingsFallbackResolver{
		holdings: []domain.StockHolding{
			{
				StockCode:    "510300",
				StockName:    "沪深300ETF",
				Exchange:     domain.ExchangeSH,
				HoldingRatio: decimal.RequireFromString("100"),
			},
		},
		source: "510300",
	}

	handler := &FundHandler{
		fundRepo:         fundRepo,
		dataLoader:       &stubTransientFundDataLoader{},
		holdingsResolver: resolver,
	}

	router := gin.New()
	router.GET("/api/v1/fund/:id/holdings", handler.GetHoldings)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fund/023408/holdings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response holdingsResponseEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false")
	}
	if len(response.Data.Holdings) != 1 {
		t.Fatalf("holdings len = %d, want 1", len(response.Data.Holdings))
	}
	if response.Data.Holdings[0].StockCode != "510300" {
		t.Fatalf("holding stock code = %q, want 510300", response.Data.Holdings[0].StockCode)
	}
	if response.Meta == nil || response.Meta.DataSource != "target_etf:510300" {
		t.Fatalf("meta = %+v, want target_etf:510300", response.Meta)
	}
	if resolver.calls != 1 {
		t.Fatalf("resolver calls = %d, want 1", resolver.calls)
	}
}

func TestGetFundSchedulesWarmupWhenCacheMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	if err := fundRepo.SaveFund(context.Background(), &domain.Fund{
		ID:          "888888",
		Name:        "待预热基金",
		Type:        "hybrid",
		Manager:     "",
		Company:     "",
		NetAssetVal: decimal.RequireFromString("1.0000"),
	}); err != nil {
		t.Fatalf("SaveFund() error = %v", err)
	}

	loader := &stubTransientFundDataLoader{}
	handler := &FundHandler{
		fundRepo:   fundRepo,
		dataLoader: loader,
	}

	router := gin.New()
	router.GET("/api/v1/fund/:id", handler.GetFund)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fund/888888", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response fundResponseEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Meta == nil || response.Meta.CacheStatus != "warming" {
		t.Fatalf("meta = %+v, want warming", response.Meta)
	}
	if loader.cachedCalls != 1 {
		t.Fatalf("cached calls = %d, want 1", loader.cachedCalls)
	}
	if loader.scheduleCalls != 1 {
		t.Fatalf("schedule calls = %d, want 1", loader.scheduleCalls)
	}
}

type stubValuationService struct {
	estimateErr error
}

func (s stubValuationService) CalculateEstimate(ctx context.Context, fundID string) (*domain.FundEstimate, error) {
	return nil, s.estimateErr
}

func (s stubValuationService) GetIntradayTimeSeries(ctx context.Context, fundID string) ([]domain.TimeSeriesPoint, error) {
	return nil, nil
}

func TestGetEstimateReturnsWarmupStatusForColdFunds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &FundHandler{
		valuationService: stubValuationService{estimateErr: service.ErrFundDataWarmupInProgress},
	}

	router := gin.New()
	router.GET("/api/v1/fund/:id/estimate", handler.GetEstimate)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fund/123456/estimate", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}

	var response struct {
		Success bool      `json:"success"`
		Error   *APIError `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error == nil || response.Error.Code != "FUND_DATA_WARMING" {
		t.Fatalf("error = %+v, want FUND_DATA_WARMING", response.Error)
	}
	if retryAfter := rec.Header().Get("Retry-After"); retryAfter != "5" {
		t.Fatalf("Retry-After = %q, want 5", retryAfter)
	}
}

func TestGetEstimateReturnsUnsupportedPricingModelForQDIIDetailsFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &FundHandler{
		valuationService: stubValuationService{estimateErr: errors.New("qdii details available without live estimate support")},
	}

	router := gin.New()
	router.GET("/api/v1/fund/:id/estimate", handler.GetEstimate)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fund/017437/estimate", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}

	var response struct {
		Success bool      `json:"success"`
		Error   *APIError `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error == nil || response.Error.Code != "UNSUPPORTED_PRICING_MODEL" {
		t.Fatalf("error = %+v, want UNSUPPORTED_PRICING_MODEL", response.Error)
	}
}

func TestResolveOfficialCloseInfoReturnsPendingAfterCloseBeforeSync(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	handler := &FundHandler{fundRepo: fundRepo}

	status := trading.GetMarketStatus(time.Date(2026, time.April, 8, 20, 30, 0, 0, trading.TradingLocation()))
	info := handler.resolveOfficialCloseInfo(context.Background(), "005827", status)

	if info == nil || info.DisplayStatus != OfficialCloseDisplayPending {
		t.Fatalf("info = %+v, want pending", info)
	}
	if info.Message == "" {
		t.Fatalf("pending info should include message")
	}
}

func TestResolveOfficialCloseInfoReturnsReadyBeforeNineWithLatestTradingDayHistory(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        "2026-04-08",
		NetAssetVal: decimal.RequireFromString("1.7877"),
		DailyReturn: decimal.RequireFromString("2.0027"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}

	handler := &FundHandler{fundRepo: fundRepo}
	status := trading.GetMarketStatus(time.Date(2026, time.April, 9, 8, 30, 0, 0, trading.TradingLocation()))
	info := handler.resolveOfficialCloseInfo(context.Background(), "005827", status)

	if info == nil || info.DisplayStatus != OfficialCloseDisplayReady {
		t.Fatalf("info = %+v, want ready", info)
	}
	if info.Date != "2026-04-08" || info.DailyReturn != "2.0027" {
		t.Fatalf("ready info = %+v", info)
	}
}

func TestResolveOfficialCloseInfoHidesAfterNineEvenIfHistoryExists(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        "2026-04-08",
		NetAssetVal: decimal.RequireFromString("1.7877"),
		DailyReturn: decimal.RequireFromString("2.0027"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}

	handler := &FundHandler{fundRepo: fundRepo}
	status := trading.GetMarketStatus(time.Date(2026, time.April, 9, 9, 5, 0, 0, trading.TradingLocation()))
	info := handler.resolveOfficialCloseInfo(context.Background(), "005827", status)

	if info == nil || info.DisplayStatus != OfficialCloseDisplayHidden {
		t.Fatalf("info = %+v, want hidden", info)
	}
}
