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

type stubFundSectorStore struct {
	snapshot        *domain.FundSectorSnapshot
	snapshotsByFund map[string]*domain.FundSectorSnapshot
}

func (s *stubFundSectorStore) UpsertFromHoldings(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundSectorSnapshot, error) {
	if s.snapshotsByFund != nil {
		if snapshot, ok := s.snapshotsByFund[fundID]; ok && snapshot != nil {
			copySnapshot := *snapshot
			return &copySnapshot, nil
		}
		return nil, nil
	}
	if s.snapshot == nil {
		return nil, nil
	}
	copySnapshot := *s.snapshot
	return &copySnapshot, nil
}

func (s *stubFundSectorStore) GetLatestSnapshot(ctx context.Context, fundID string) (*domain.FundSectorSnapshot, error) {
	if s.snapshotsByFund != nil {
		if snapshot, ok := s.snapshotsByFund[fundID]; ok && snapshot != nil {
			copySnapshot := *snapshot
			return &copySnapshot, nil
		}
		return nil, nil
	}
	if s.snapshot == nil {
		return nil, nil
	}
	copySnapshot := *s.snapshot
	return &copySnapshot, nil
}

func (s *stubFundSectorStore) ResolveFundCategory(ctx context.Context, fund *domain.Fund, snapshot *domain.FundSectorSnapshot) (*domain.FundCategory, error) {
	if fund == nil {
		return nil, nil
	}
	if fund.CategoryCode == "" {
		return nil, nil
	}
	return &domain.FundCategory{
		Code: fund.CategoryCode,
		Name: fund.CategoryName,
	}, nil
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

type dashboardResponseEnvelope struct {
	Success bool `json:"success"`
	Data    struct {
		Estimate *struct {
			ChangePercent string `json:"change_percent"`
		} `json:"estimate"`
		SectorSnapshot *domain.FundSectorSnapshot `json:"sector_snapshot"`
		TimeSeries     []struct {
			Timestamp     string `json:"timestamp"`
			ChangePercent string `json:"change_percent"`
		} `json:"time_series"`
	} `json:"data"`
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

func TestSearchFiltersByCategoryAndSector(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	if err := fundRepo.SaveFund(context.Background(), &domain.Fund{
		ID:           "005827",
		Name:         "易方达蓝筹精选混合",
		Type:         "hybrid",
		CategoryCode: "hybrid",
		CategoryName: "混合型",
	}); err != nil {
		t.Fatalf("SaveFund() error = %v", err)
	}
	if err := fundRepo.SaveFund(context.Background(), &domain.Fund{
		ID:           "320007",
		Name:         "诺安成长混合",
		Type:         "hybrid",
		CategoryCode: "hybrid",
		CategoryName: "混合型",
	}); err != nil {
		t.Fatalf("SaveFund() error = %v", err)
	}

	handler := &FundHandler{
		fundRepo: fundRepo,
		sectorStore: &stubFundSectorStore{
			snapshotsByFund: map[string]*domain.FundSectorSnapshot{
				"005827": {
					FundID:            "005827",
					AsOfDate:          "2025-12-31",
					PrimarySectorCode: "liquor",
					PrimarySectorName: "白酒",
				},
				"320007": {
					FundID:            "320007",
					AsOfDate:          "2025-12-31",
					PrimarySectorCode: "semiconductor",
					PrimarySectorName: "半导体",
				},
			},
		},
	}

	router := gin.New()
	router.GET("/api/v1/fund/search", handler.Search)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fund/search?q=混合&category=hybrid&sector=liquor", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response struct {
		Success bool          `json:"success"`
		Data    []domain.Fund `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data) != 1 {
		t.Fatalf("len(data) = %d, want 1", len(response.Data))
	}
	if response.Data[0].ID != "005827" {
		t.Fatalf("fund id = %s, want 005827", response.Data[0].ID)
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
	estimateErr   error
	estimate      *domain.FundEstimate
	timeSeries    []domain.TimeSeriesPoint
	timeSeriesErr error
}

func (s stubValuationService) CalculateEstimate(ctx context.Context, fundID string) (*domain.FundEstimate, error) {
	if s.estimate != nil || s.estimateErr != nil {
		return s.estimate, s.estimateErr
	}
	return nil, s.estimateErr
}

func (s stubValuationService) GetIntradayTimeSeries(ctx context.Context, fundID string) ([]domain.TimeSeriesPoint, error) {
	return s.timeSeries, s.timeSeriesErr
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

func TestGetDashboardAlignsTimeSeriesLastPointWithEstimateSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	if err := fundRepo.SaveFund(context.Background(), &domain.Fund{
		ID:          "005827",
		Name:        "易方达蓝筹精选混合",
		Type:        "hybrid",
		NetAssetVal: decimal.RequireFromString("1.7643"),
		UpdatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFund() error = %v", err)
	}

	estimate := &domain.FundEstimate{
		FundID:        "005827",
		FundName:      "易方达蓝筹精选混合",
		EstimateNav:   decimal.RequireFromString("1.7346"),
		PrevNav:       decimal.RequireFromString("1.7643"),
		ChangePercent: decimal.RequireFromString("-1.6819"),
		CalculatedAt:  time.Date(2026, time.April, 18, 9, 17, 51, 0, trading.TradingLocation()),
		DataSource:    "sina",
	}
	points := []domain.TimeSeriesPoint{
		{
			Timestamp:     time.Date(2026, time.April, 17, 14, 55, 0, 0, trading.TradingLocation()),
			ChangePercent: decimal.RequireFromString("-1.8508"),
			EstimateNav:   decimal.RequireFromString("1.7573"),
		},
		{
			Timestamp:     time.Date(2026, time.April, 17, 15, 0, 0, 0, trading.TradingLocation()),
			ChangePercent: decimal.RequireFromString("-1.8119"),
			EstimateNav:   decimal.RequireFromString("1.7580"),
		},
	}

	handler := &FundHandler{
		valuationService: stubValuationService{
			estimate:   estimate,
			timeSeries: points,
		},
		fundRepo:   fundRepo,
		dataLoader: &stubTransientFundDataLoader{},
		sectorStore: &stubFundSectorStore{
			snapshot: &domain.FundSectorSnapshot{
				FundID:            "005827",
				AsOfDate:          "2025-12-31",
				PrimarySectorCode: "liquor",
				PrimarySectorName: "白酒",
			},
		},
	}

	router := gin.New()
	router.GET("/api/v1/fund/:id/dashboard", handler.GetDashboard)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fund/005827/dashboard", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response dashboardResponseEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false")
	}
	if response.Data.Estimate == nil {
		t.Fatalf("estimate should not be nil")
	}
	if got := response.Data.Estimate.ChangePercent; got != "-1.6819" {
		t.Fatalf("estimate change percent = %s, want -1.6819", got)
	}
	if len(response.Data.TimeSeries) == 0 {
		t.Fatalf("time series should not be empty")
	}
	if response.Data.SectorSnapshot == nil || response.Data.SectorSnapshot.PrimarySectorCode != "liquor" {
		t.Fatalf("sector snapshot = %+v, want liquor", response.Data.SectorSnapshot)
	}
	lastPoint := response.Data.TimeSeries[len(response.Data.TimeSeries)-1]
	if got := lastPoint.ChangePercent; got != "-1.6819" {
		t.Fatalf("last point change percent = %s, want -1.6819", got)
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
