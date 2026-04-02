package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type stubTransientFundDataLoader struct {
	calls    int
	fund     *domain.Fund
	holdings []domain.StockHolding
	err      error
}

func (s *stubTransientFundDataLoader) FetchTransientFundData(ctx context.Context, fundID string) (*domain.Fund, []domain.StockHolding, error) {
	s.calls++
	return s.fund, s.holdings, s.err
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
	if loader.calls != 1 {
		t.Fatalf("loader calls = %d, want 1", loader.calls)
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
	if loader.calls != 1 {
		t.Fatalf("loader calls = %d, want 1", loader.calls)
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
