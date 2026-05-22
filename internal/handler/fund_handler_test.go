package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
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
	calls           int
	displayCalls    int
	holdings        []domain.StockHolding
	source          string
	err             error
	displayHoldings *domain.FundHoldingsDisplay
	displayErr      error
}

func (s *stubHoldingsFallbackResolver) GetHoldingsWithFallback(ctx context.Context, fundID string, fundName string) ([]domain.StockHolding, string, error) {
	s.calls++
	return s.holdings, s.source, s.err
}

func (s *stubHoldingsFallbackResolver) ResolveDisplayHoldings(ctx context.Context, fundID string, fundName string) (*domain.FundHoldingsDisplay, error) {
	s.displayCalls++
	return s.displayHoldings, s.displayErr
}

type stubFundSectorStore struct {
	snapshot        *domain.FundSectorSnapshot
	snapshotsByFund map[string]*domain.FundSectorSnapshot
	themeSnapshot   *domain.FundThemeSnapshot
	themesByFund    map[string]*domain.FundThemeSnapshot
}

type stubAnalysisRankingCandidateProvider struct {
	fundIDs []string
	err     error
}

func (s *stubAnalysisRankingCandidateProvider) ListRankingCandidateFundIDs(ctx context.Context, limit int) ([]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	if limit > 0 && len(s.fundIDs) > limit {
		return append([]string(nil), s.fundIDs[:limit]...), nil
	}
	return append([]string(nil), s.fundIDs...), nil
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

func (s *stubFundSectorStore) UpsertThemeFromHoldings(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundThemeSnapshot, error) {
	if s.themesByFund != nil {
		if snapshot, ok := s.themesByFund[fundID]; ok && snapshot != nil {
			copySnapshot := *snapshot
			return &copySnapshot, nil
		}
		return nil, nil
	}
	if s.themeSnapshot == nil {
		return nil, nil
	}
	copySnapshot := *s.themeSnapshot
	return &copySnapshot, nil
}

func (s *stubFundSectorStore) BuildSnapshot(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundSectorSnapshot, error) {
	return s.UpsertFromHoldings(ctx, fundID, holdings, source)
}

func (s *stubFundSectorStore) BuildThemeSnapshot(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundThemeSnapshot, error) {
	return s.UpsertThemeFromHoldings(ctx, fundID, holdings, source)
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

func (s *stubFundSectorStore) GetLatestThemeSnapshot(ctx context.Context, fundID string) (*domain.FundThemeSnapshot, error) {
	if s.themesByFund != nil {
		if snapshot, ok := s.themesByFund[fundID]; ok && snapshot != nil {
			copySnapshot := *snapshot
			return &copySnapshot, nil
		}
		return nil, nil
	}
	if s.themeSnapshot == nil {
		return nil, nil
	}
	copySnapshot := *s.themeSnapshot
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
		Fund                 domain.Fund                      `json:"fund"`
		Holdings             []domain.StockHolding            `json:"holdings"`
		DisplayLevel         string                           `json:"display_level"`
		DisplayItems         []domain.FundHoldingsDisplayItem `json:"display_items"`
		LookthroughAvailable bool                             `json:"lookthrough_available"`
	} `json:"data"`
	Meta *APIMeta `json:"meta,omitempty"`
}

type dashboardResponseEnvelope struct {
	Success bool `json:"success"`
	Data    struct {
		Estimate *struct {
			ChangePercent string `json:"change_percent"`
		} `json:"estimate"`
		Analysis *struct {
			AnalysisType    string `json:"analysis_type"`
			AnalysisBasis   string `json:"analysis_basis"`
			TotalScore      string `json:"total_score"`
			IncreasePercent string `json:"increase_percent"`
			HoldPercent     string `json:"hold_percent"`
			DecreasePercent string `json:"decrease_percent"`
			RiskLevel       string `json:"risk_level"`
			Confidence      string `json:"confidence"`
			EventImpacts    []struct {
				Code   string `json:"code"`
				Impact string `json:"impact"`
			} `json:"event_impacts"`
			AIExplanation *struct {
				Status         string `json:"status"`
				Provider       string `json:"provider"`
				BoundaryNotice string `json:"boundary_notice"`
			} `json:"ai_explanation"`
		} `json:"analysis"`
		SectorSnapshot *domain.FundSectorSnapshot `json:"sector_snapshot"`
		ThemeSnapshot  *domain.FundThemeSnapshot  `json:"theme_snapshot"`
		TimeSeries     []struct {
			Timestamp     string `json:"timestamp"`
			ChangePercent string `json:"change_percent"`
		} `json:"time_series"`
	} `json:"data"`
}

type analysisResponseEnvelope struct {
	Success bool `json:"success"`
	Data    struct {
		Fund     *domain.Fund `json:"fund"`
		Analysis *struct {
			AnalysisType  string `json:"analysis_type"`
			AnalysisBasis string `json:"analysis_basis"`
			TotalScore    string `json:"total_score"`
			EventImpacts  []struct {
				Code string `json:"code"`
			} `json:"event_impacts"`
			AIExplanation *struct {
				Status         string `json:"status"`
				Provider       string `json:"provider"`
				BoundaryNotice string `json:"boundary_notice"`
			} `json:"ai_explanation"`
		} `json:"analysis"`
	} `json:"data"`
	Meta *APIMeta `json:"meta,omitempty"`
}

type analysisRankingsResponseEnvelope struct {
	Success bool `json:"success"`
	Data    struct {
		GeneratedAt   string `json:"generated_at"`
		IncreaseIdeas []struct {
			Fund struct {
				ID string `json:"id"`
			} `json:"fund"`
		} `json:"increase_ideas"`
		WatchIdeas []struct {
			Fund struct {
				ID string `json:"id"`
			} `json:"fund"`
		} `json:"watch_ideas"`
		RiskAlerts []struct {
			Fund struct {
				ID string `json:"id"`
			} `json:"fund"`
		} `json:"risk_alerts"`
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
	if response.Data.DisplayLevel != domain.FundHoldingsDisplayLevelStock {
		t.Fatalf("display level = %q, want %q", response.Data.DisplayLevel, domain.FundHoldingsDisplayLevelStock)
	}
	if len(response.Data.DisplayItems) != 1 || response.Data.DisplayItems[0].Code != "600519" {
		t.Fatalf("display items = %+v, want stock layer for 600519", response.Data.DisplayItems)
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
	if response.Data.DisplayLevel != domain.FundHoldingsDisplayLevelStock {
		t.Fatalf("display level = %q, want %q", response.Data.DisplayLevel, domain.FundHoldingsDisplayLevelStock)
	}
	if len(response.Data.DisplayItems) != 1 || response.Data.DisplayItems[0].Code != "510300" {
		t.Fatalf("display items = %+v, want stock layer for 510300", response.Data.DisplayItems)
	}
	if response.Meta == nil || response.Meta.DataSource != "target_etf:510300" {
		t.Fatalf("meta = %+v, want target_etf:510300", response.Meta)
	}
	if resolver.calls != 1 {
		t.Fatalf("resolver calls = %d, want 1", resolver.calls)
	}
}

func TestGetHoldingsUsesTargetLayerDisplayWhenResolverProvidesDisplayHoldings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	if err := fundRepo.SaveFund(context.Background(), &domain.Fund{
		ID:          "006480",
		Name:        "广发纳斯达克100ETF联接美元(QDII)C",
		Type:        "qdii",
		Manager:     "刘杰",
		Company:     "广发基金",
		NetAssetVal: decimal.RequireFromString("1.1111"),
	}); err != nil {
		t.Fatalf("SaveFund() error = %v", err)
	}
	if err := fundRepo.SaveHoldings(context.Background(), "006480", []domain.StockHolding{
		{
			StockCode:    "AAPL",
			StockName:    "苹果",
			Exchange:     domain.ExchangeUS,
			HoldingRatio: decimal.RequireFromString("1.46"),
		},
	}); err != nil {
		t.Fatalf("SaveHoldings() error = %v", err)
	}

	resolver := &stubHoldingsFallbackResolver{
		displayHoldings: &domain.FundHoldingsDisplay{
			DisplayLevel: domain.FundHoldingsDisplayLevelTarget,
			DisplayItems: []domain.FundHoldingsDisplayItem{
				{
					ItemType:   domain.FundHoldingsDisplayItemTypeTargetFund,
					TargetType: domain.FundHoldingsDisplayTargetTypeETFFund,
					Code:       "159941",
					Name:       "纳指ETF广发",
					IsPrimary:  true,
					Source:     "mapping",
				},
			},
			LookthroughAvailable: true,
		},
	}

	handler := &FundHandler{
		fundRepo:         fundRepo,
		dataLoader:       &stubTransientFundDataLoader{},
		holdingsResolver: resolver,
	}

	router := gin.New()
	router.GET("/api/v1/fund/:id/holdings", handler.GetHoldings)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fund/006480/holdings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response holdingsResponseEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data.DisplayLevel != domain.FundHoldingsDisplayLevelTarget {
		t.Fatalf("display level = %q, want %q", response.Data.DisplayLevel, domain.FundHoldingsDisplayLevelTarget)
	}
	if !response.Data.LookthroughAvailable {
		t.Fatalf("expected lookthrough available")
	}
	if len(response.Data.DisplayItems) != 1 {
		t.Fatalf("display items len = %d, want 1", len(response.Data.DisplayItems))
	}
	if response.Data.DisplayItems[0].Code != "159941" {
		t.Fatalf("display target code = %q, want 159941", response.Data.DisplayItems[0].Code)
	}
	if response.Data.DisplayItems[0].Name != "纳指ETF广发" {
		t.Fatalf("display target name = %q, want 纳指ETF广发", response.Data.DisplayItems[0].Name)
	}
	if len(response.Data.Holdings) != 1 || response.Data.Holdings[0].StockCode != "AAPL" {
		t.Fatalf("raw holdings = %+v, want preserved lookthrough stock", response.Data.Holdings)
	}
	if resolver.displayCalls != 1 {
		t.Fatalf("display resolver calls = %d, want 1", resolver.displayCalls)
	}
}

func TestGetHoldingsUsesTargetETFHoldingsWhenTargetLayerHasLookthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	if err := fundRepo.SaveFund(context.Background(), &domain.Fund{
		ID:          "012970",
		Name:        "鹏华国证半导体芯片ETF联接C",
		Type:        "index",
		Manager:     "罗英宇",
		NetAssetVal: decimal.RequireFromString("1.1498"),
	}); err != nil {
		t.Fatalf("SaveFund(feeder) error = %v", err)
	}
	if err := fundRepo.SaveFund(context.Background(), &domain.Fund{
		ID:          "159813",
		Name:        "半导体ETF鹏华",
		Type:        "index",
		NetAssetVal: decimal.RequireFromString("1.1766"),
	}); err != nil {
		t.Fatalf("SaveFund(target) error = %v", err)
	}
	if err := fundRepo.SaveHoldings(context.Background(), "012970", []domain.StockHolding{
		{
			StockCode:    "002049",
			StockName:    "紫光国微",
			Exchange:     domain.ExchangeSZ,
			HoldingRatio: decimal.RequireFromString("0.04"),
		},
	}); err != nil {
		t.Fatalf("SaveHoldings(feeder residual) error = %v", err)
	}
	if err := fundRepo.SaveHoldings(context.Background(), "159813", []domain.StockHolding{
		{
			StockCode:    "688981",
			StockName:    "中芯国际",
			Exchange:     domain.ExchangeSH,
			HoldingRatio: decimal.RequireFromString("10.28"),
		},
		{
			StockCode:    "688041",
			StockName:    "海光信息",
			Exchange:     domain.ExchangeSH,
			HoldingRatio: decimal.RequireFromString("9.97"),
		},
	}); err != nil {
		t.Fatalf("SaveHoldings(target) error = %v", err)
	}

	resolver := &stubHoldingsFallbackResolver{
		displayHoldings: &domain.FundHoldingsDisplay{
			DisplayLevel: domain.FundHoldingsDisplayLevelTarget,
			DisplayItems: []domain.FundHoldingsDisplayItem{
				{
					ItemType:      domain.FundHoldingsDisplayItemTypeTargetFund,
					TargetType:    domain.FundHoldingsDisplayTargetTypeETFFund,
					Code:          "159813",
					Name:          "半导体ETF鹏华",
					WeightPercent: decimal.NewFromInt(100),
					IsPrimary:     true,
					Source:        "mapping",
				},
			},
			LookthroughAvailable: true,
		},
	}

	handler := &FundHandler{
		fundRepo:         fundRepo,
		dataLoader:       &stubTransientFundDataLoader{},
		holdingsResolver: resolver,
	}

	router := gin.New()
	router.GET("/api/v1/fund/:id/holdings", handler.GetHoldings)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fund/012970/holdings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response holdingsResponseEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data.DisplayLevel != domain.FundHoldingsDisplayLevelTarget {
		t.Fatalf("display level = %q, want %q", response.Data.DisplayLevel, domain.FundHoldingsDisplayLevelTarget)
	}
	if len(response.Data.DisplayItems) != 1 || response.Data.DisplayItems[0].WeightPercent.String() != "100" {
		t.Fatalf("display items = %+v, want target weight 100", response.Data.DisplayItems)
	}
	if len(response.Data.Holdings) != 2 || response.Data.Holdings[0].StockCode != "688981" {
		t.Fatalf("raw holdings = %+v, want target ETF holdings", response.Data.Holdings)
	}
	if response.Meta == nil || response.Meta.DataSource != "target_etf:159813" {
		t.Fatalf("meta = %+v, want target_etf:159813", response.Meta)
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

type stubMultiFundValuationService struct {
	estimates   map[string]*domain.FundEstimate
	timeSeries  map[string][]domain.TimeSeriesPoint
	estimateErr error
	seriesErr   error
}

func (s stubMultiFundValuationService) CalculateEstimate(ctx context.Context, fundID string) (*domain.FundEstimate, error) {
	if s.estimateErr != nil {
		return nil, s.estimateErr
	}
	return s.estimates[fundID], nil
}

func (s stubMultiFundValuationService) GetIntradayTimeSeries(ctx context.Context, fundID string) ([]domain.TimeSeriesPoint, error) {
	if s.seriesErr != nil {
		return nil, s.seriesErr
	}
	return s.timeSeries[fundID], nil
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
			themeSnapshot: &domain.FundThemeSnapshot{
				FundID:           "005827",
				AsOfDate:         "2025-12-31",
				PrimaryThemeCode: "ai_application",
				PrimaryThemeName: "AI应用",
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
	if response.Data.ThemeSnapshot == nil || response.Data.ThemeSnapshot.PrimaryThemeCode != "ai_application" {
		t.Fatalf("theme snapshot = %+v, want ai_application", response.Data.ThemeSnapshot)
	}
	if response.Data.Analysis == nil {
		t.Fatalf("analysis should not be nil")
	}
	if response.Data.Analysis.AnalysisType != service.FundAnalysisTypeDirectHoldings {
		t.Fatalf("analysis type = %s, want %s", response.Data.Analysis.AnalysisType, service.FundAnalysisTypeDirectHoldings)
	}
	if response.Data.Analysis.TotalScore == "" {
		t.Fatalf("analysis total score should not be empty")
	}
	if len(response.Data.Analysis.EventImpacts) == 0 {
		t.Fatalf("analysis event impacts should not be empty")
	}
	if response.Data.Analysis.AIExplanation == nil {
		t.Fatalf("analysis AI explanation should expose disabled/fallback boundary")
	}
	if response.Data.Analysis.AIExplanation.Status != service.AIExplanationStatusDisabled && response.Data.Analysis.AIExplanation.Status != service.AIExplanationStatusRejected {
		t.Fatalf("AI explanation status = %s, want disabled or rejected", response.Data.Analysis.AIExplanation.Status)
	}
	lastPoint := response.Data.TimeSeries[len(response.Data.TimeSeries)-1]
	if got := lastPoint.ChangePercent; got != "-1.6819" {
		t.Fatalf("last point change percent = %s, want -1.6819", got)
	}
}

func TestGetAnalysisReturnsStandaloneAnalysisPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	if err := fundRepo.SaveFund(context.Background(), &domain.Fund{
		ID:           "005827",
		Name:         "易方达蓝筹精选混合",
		Type:         "hybrid",
		CategoryCode: "equity",
		CategoryName: "权益基金",
		NetAssetVal:  decimal.RequireFromString("1.7643"),
		UpdatedAt:    time.Now(),
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
			themeSnapshot: &domain.FundThemeSnapshot{
				FundID:           "005827",
				AsOfDate:         "2025-12-31",
				PrimaryThemeCode: "ai_application",
				PrimaryThemeName: "AI应用",
			},
		},
	}

	router := gin.New()
	router.GET("/api/v1/fund/:id/analysis", handler.GetAnalysis)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fund/005827/analysis", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response analysisResponseEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false")
	}
	if response.Data.Fund == nil || response.Data.Fund.ID != "005827" {
		t.Fatalf("fund = %+v, want 005827", response.Data.Fund)
	}
	if response.Data.Analysis == nil {
		t.Fatal("analysis should not be nil")
	}
	if response.Data.Analysis.AnalysisType != service.FundAnalysisTypeDirectHoldings {
		t.Fatalf("analysis type = %s, want %s", response.Data.Analysis.AnalysisType, service.FundAnalysisTypeDirectHoldings)
	}
	if response.Data.Analysis.AnalysisBasis == "" || response.Data.Analysis.TotalScore == "" {
		t.Fatalf("analysis payload incomplete = %+v", response.Data.Analysis)
	}
	if len(response.Data.Analysis.EventImpacts) == 0 {
		t.Fatal("analysis event impacts should not be empty")
	}
	if response.Data.Analysis.AIExplanation == nil {
		t.Fatal("analysis AI explanation should not be nil")
	}
	if response.Data.Analysis.AIExplanation.Status != service.AIExplanationStatusDisabled && response.Data.Analysis.AIExplanation.Status != service.AIExplanationStatusRejected {
		t.Fatalf("AI explanation status = %s, want disabled or rejected", response.Data.Analysis.AIExplanation.Status)
	}
	if !strings.Contains(response.Data.Analysis.AIExplanation.BoundaryNotice, "不得改写") {
		t.Fatalf("AI explanation boundary = %q, want scoring boundary", response.Data.Analysis.AIExplanation.BoundaryNotice)
	}
	if response.Meta == nil || response.Meta.DataSource != "sina" {
		t.Fatalf("meta = %+v, want data source sina", response.Meta)
	}
}

func TestGetAnalysisRankingsReturnsBuckets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fundRepo := repository.NewMemoryFundRepository()
	valuation := stubMultiFundValuationService{
		estimates: map[string]*domain.FundEstimate{
			"005827": {
				FundID:        "005827",
				FundName:      "易方达蓝筹精选混合",
				EstimateNav:   decimal.RequireFromString("1.7346"),
				PrevNav:       decimal.RequireFromString("1.7643"),
				ChangePercent: decimal.RequireFromString("-1.6819"),
				CalculatedAt:  time.Date(2026, time.April, 18, 9, 17, 51, 0, trading.TradingLocation()),
				DataSource:    "sina",
			},
			"003095": {
				FundID:        "003095",
				FundName:      "中欧医疗健康混合A",
				EstimateNav:   decimal.RequireFromString("1.6120"),
				PrevNav:       decimal.RequireFromString("1.5432"),
				ChangePercent: decimal.RequireFromString("4.4570"),
				CalculatedAt:  time.Date(2026, time.April, 18, 9, 18, 10, 0, trading.TradingLocation()),
				DataSource:    "sina",
			},
			"320007": {
				FundID:        "320007",
				FundName:      "诺安成长混合",
				EstimateNav:   decimal.RequireFromString("1.1820"),
				PrevNav:       decimal.RequireFromString("1.2345"),
				ChangePercent: decimal.RequireFromString("-4.2537"),
				CalculatedAt:  time.Date(2026, time.April, 18, 9, 18, 30, 0, trading.TradingLocation()),
				DataSource:    "sina",
			},
		},
		timeSeries: map[string][]domain.TimeSeriesPoint{
			"005827": {{
				Timestamp:     time.Date(2026, time.April, 18, 9, 15, 0, 0, trading.TradingLocation()),
				ChangePercent: decimal.RequireFromString("-1.7000"),
				EstimateNav:   decimal.RequireFromString("1.7340"),
			}},
			"003095": {{
				Timestamp:     time.Date(2026, time.April, 18, 9, 15, 0, 0, trading.TradingLocation()),
				ChangePercent: decimal.RequireFromString("4.4000"),
				EstimateNav:   decimal.RequireFromString("1.6110"),
			}},
			"320007": {{
				Timestamp:     time.Date(2026, time.April, 18, 9, 15, 0, 0, trading.TradingLocation()),
				ChangePercent: decimal.RequireFromString("-4.2000"),
				EstimateNav:   decimal.RequireFromString("1.1830"),
			}},
		},
	}

	handler := &FundHandler{
		valuationService:  valuation,
		fundRepo:          fundRepo,
		dataLoader:        &stubTransientFundDataLoader{},
		rankingCandidates: &stubAnalysisRankingCandidateProvider{fundIDs: []string{"005827", "003095", "320007"}},
	}

	router := gin.New()
	router.GET("/api/v1/analysis/rankings", handler.GetAnalysisRankings)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analysis/rankings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response analysisRankingsResponseEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false")
	}
	totalItems := len(response.Data.IncreaseIdeas) + len(response.Data.WatchIdeas) + len(response.Data.RiskAlerts)
	if totalItems == 0 {
		t.Fatalf("rankings should not be empty: %+v", response.Data)
	}
	if response.Data.GeneratedAt == "" {
		t.Fatalf("generated_at should not be empty")
	}
}

func TestAnalysisRankingsUseThresholdedDominantRecommendation(t *testing.T) {
	items := []analysisRankingCandidate{
		{
			fund: &domain.Fund{ID: "slight-positive", Name: "略偏积极样本"},
			analysis: &domain.FundAnalysis{
				IncreasePercent: decimal.RequireFromString("46.5"),
				HoldPercent:     decimal.RequireFromString("42.6"),
				DecreasePercent: decimal.RequireFromString("10.9"),
				TotalScore:      decimal.RequireFromString("58.0"),
				Confidence:      "medium",
			},
		},
	}

	increaseItems, watchItems, _ := bucketAnalysisRankingCandidates(items)

	if len(increaseItems) != 0 {
		t.Fatalf("slight positive sample should not enter increase bucket before threshold: %+v", increaseItems)
	}
	if len(watchItems) != 1 || watchItems[0].fund.ID != "slight-positive" {
		t.Fatalf("watch items = %+v, want slight-positive", watchItems)
	}
}

func TestBuildRankingItemsFromSnapshotsUsesThresholdedBuckets(t *testing.T) {
	records := []database.FundAnalysisSnapshot{
		analysisSnapshotRecord(t, "slight-positive", &domain.FundAnalysis{
			IncreasePercent: decimal.RequireFromString("46.5"),
			HoldPercent:     decimal.RequireFromString("42.6"),
			DecreasePercent: decimal.RequireFromString("10.9"),
			TotalScore:      decimal.RequireFromString("58.0"),
			Confidence:      "medium",
		}),
		analysisSnapshotRecord(t, "strong-positive", &domain.FundAnalysis{
			IncreasePercent: decimal.RequireFromString("56.0"),
			HoldPercent:     decimal.RequireFromString("35.0"),
			DecreasePercent: decimal.RequireFromString("9.0"),
			TotalScore:      decimal.RequireFromString("76.0"),
			Confidence:      "high",
		}),
		analysisSnapshotRecord(t, "high-risk-watch", &domain.FundAnalysis{
			IncreasePercent: decimal.RequireFromString("28.0"),
			HoldPercent:     decimal.RequireFromString("52.0"),
			DecreasePercent: decimal.RequireFromString("20.0"),
			TotalScore:      decimal.RequireFromString("42.0"),
			Confidence:      "low",
			RiskLevel:       "high",
		}),
	}

	fundMap := map[string]*domain.Fund{
		"slight-positive": &domain.Fund{ID: "slight-positive", Name: "略偏积极样本"},
		"strong-positive": &domain.Fund{ID: "strong-positive", Name: "强积极样本"},
		"high-risk-watch": &domain.Fund{ID: "high-risk-watch", Name: "高风险观察样本"},
	}

	increaseItems := buildRankingItemsFromSnapshots(records, fundMap, analysisRankingBucketIncrease, 12)
	watchItems := buildRankingItemsFromSnapshots(records, fundMap, analysisRankingBucketWatch, 12)
	riskItems := buildRankingItemsFromSnapshots(records, fundMap, analysisRankingBucketRisk, 12)

	if len(increaseItems) != 1 || increaseItems[0].Fund.ID != "strong-positive" {
		t.Fatalf("increase snapshot items = %+v, want only strong-positive", increaseItems)
	}
	if len(watchItems) != 2 || watchItems[0].Fund.ID != "slight-positive" || watchItems[1].Fund.ID != "high-risk-watch" {
		t.Fatalf("watch snapshot items = %+v, want slight-positive and high-risk-watch", watchItems)
	}
	if len(riskItems) != 1 || riskItems[0].Fund.ID != "high-risk-watch" {
		t.Fatalf("risk snapshot items = %+v, want high-risk-watch", riskItems)
	}
}

func analysisSnapshotRecord(t *testing.T, fundID string, analysis *domain.FundAnalysis) database.FundAnalysisSnapshot {
	t.Helper()
	payload, err := json.Marshal(analysis)
	if err != nil {
		t.Fatalf("marshal analysis snapshot: %v", err)
	}
	return database.FundAnalysisSnapshot{FundID: fundID, AnalysisJSON: payload}
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
