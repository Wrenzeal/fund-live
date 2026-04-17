package service

import (
	"context"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/shopspring/decimal"
)

type stubFundMappingStore struct {
	mapping *database.FundMapping
	saved   *database.FundMapping
}

func (s *stubFundMappingStore) GetByFeederCode(ctx context.Context, feederCode string) (*database.FundMapping, error) {
	if s.mapping == nil {
		return nil, nil
	}
	copyMapping := *s.mapping
	return &copyMapping, nil
}

func (s *stubFundMappingStore) Save(ctx context.Context, mapping *database.FundMapping) error {
	if mapping == nil {
		s.saved = nil
		return nil
	}
	copyMapping := *mapping
	s.saved = &copyMapping
	s.mapping = &copyMapping
	return nil
}

func TestFundResolverSkipsSearchWhenRecentFailureIsCached(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	store := &stubFundMappingStore{
		mapping: &database.FundMapping{
			FeederCode:   "023408",
			FeederName:   "示例联接基金",
			IsResolved:   false,
			ResolveError: "search fallback could not find target ETF",
			UpdatedAt:    now.Add(-1 * time.Hour),
		},
	}
	repo := repository.NewMemoryFundRepository()
	searchCalls := 0
	resolver := &FundResolver{
		mappingStore: store,
		fundRepo:     repo,
		searchByQuery: func(ctx context.Context, query string) ([]eastmoneySearchResult, error) {
			searchCalls++
			return []eastmoneySearchResult{{Code: "510300", Name: "沪深300ETF"}}, nil
		},
		now:           func() time.Time { return now },
		retryCooldown: 12 * time.Hour,
	}

	holdings, source, err := resolver.GetHoldingsWithFallback(context.Background(), "023408", "示例联接基金")
	if err == nil {
		t.Fatalf("expected cooldown error, got nil")
	}
	if searchCalls != 0 {
		t.Fatalf("search calls = %d, want 0", searchCalls)
	}
	if source != "023408" {
		t.Fatalf("source = %s, want 023408", source)
	}
	if len(holdings) != 0 {
		t.Fatalf("holdings len = %d, want 0", len(holdings))
	}
	if store.saved != nil {
		t.Fatalf("unexpected mapping save during cooldown: %+v", store.saved)
	}
}

func TestFundResolverUsesRelatedETFLinkBeforeSearchAndBypassesCooldown(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	store := &stubFundMappingStore{
		mapping: &database.FundMapping{
			FeederCode:   "010524",
			FeederName:   "银华中证5G通信主题ETF联接C",
			IsResolved:   false,
			ResolveError: "search fallback could not find target ETF",
			UpdatedAt:    now.Add(-10 * time.Minute),
		},
	}
	repo := repository.NewMemoryFundRepository()
	if err := repo.SaveHoldings(context.Background(), "159994", []domain.StockHolding{
		{
			StockCode:    "300308",
			StockName:    "中际旭创",
			Exchange:     domain.ExchangeSZ,
			HoldingRatio: decimal.RequireFromString("8.80"),
		},
	}); err != nil {
		t.Fatalf("SaveHoldings() error = %v", err)
	}

	searchCalls := 0
	resolver := &FundResolver{
		mappingStore: store,
		fundRepo:     repo,
		searchByQuery: func(ctx context.Context, query string) ([]eastmoneySearchResult, error) {
			searchCalls++
			return []eastmoneySearchResult{{Code: "515050", Name: "华夏中证5G通信主题ETF"}}, nil
		},
		loadDetailHints: func(ctx context.Context, fundCode string) (*fundDetailResolutionHints, error) {
			return &fundDetailResolutionHints{
				RelatedETFCode: "159994",
				TrackingTarget: "中证5G通信主题指数",
			}, nil
		},
		now:           func() time.Time { return now },
		retryCooldown: 30 * time.Minute,
	}

	holdings, source, err := resolver.GetHoldingsWithFallback(context.Background(), "010524", "银华中证5G通信主题ETF联接C")
	if err != nil {
		t.Fatalf("GetHoldingsWithFallback() error = %v", err)
	}
	if searchCalls != 0 {
		t.Fatalf("search calls = %d, want 0", searchCalls)
	}
	if source != "159994" {
		t.Fatalf("source = %s, want 159994", source)
	}
	if len(holdings) != 1 {
		t.Fatalf("holdings len = %d, want 1", len(holdings))
	}
	if store.saved == nil || !store.saved.IsResolved || store.saved.TargetCode != "159994" {
		t.Fatalf("saved mapping = %+v", store.saved)
	}
}

func TestFundResolverRetriesAfterFailureCooldownExpires(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	store := &stubFundMappingStore{
		mapping: &database.FundMapping{
			FeederCode:   "023408",
			FeederName:   "示例联接基金",
			IsResolved:   false,
			ResolveError: "search fallback could not find target ETF",
			UpdatedAt:    now.Add(-13 * time.Hour),
		},
	}
	repo := repository.NewMemoryFundRepository()
	if err := repo.SaveHoldings(context.Background(), "510300", []domain.StockHolding{
		{
			StockCode:    "600519",
			StockName:    "贵州茅台",
			Exchange:     domain.ExchangeSH,
			HoldingRatio: decimal.RequireFromString("9.90"),
		},
	}); err != nil {
		t.Fatalf("SaveHoldings() error = %v", err)
	}

	searchCalls := 0
	resolver := &FundResolver{
		mappingStore: store,
		fundRepo:     repo,
		searchByQuery: func(ctx context.Context, query string) ([]eastmoneySearchResult, error) {
			searchCalls++
			return []eastmoneySearchResult{{Code: "510300", Name: "沪深300ETF"}}, nil
		},
		now:           func() time.Time { return now },
		retryCooldown: 12 * time.Hour,
	}

	holdings, source, err := resolver.GetHoldingsWithFallback(context.Background(), "023408", "示例联接基金")
	if err != nil {
		t.Fatalf("GetHoldingsWithFallback() error = %v", err)
	}
	if searchCalls != 1 {
		t.Fatalf("search calls = %d, want 1", searchCalls)
	}
	if source != "510300" {
		t.Fatalf("source = %s, want 510300", source)
	}
	if len(holdings) != 1 {
		t.Fatalf("holdings len = %d, want 1", len(holdings))
	}
	if store.saved == nil {
		t.Fatalf("expected resolved mapping to be saved")
	}
	if !store.saved.IsResolved || store.saved.TargetCode != "510300" {
		t.Fatalf("saved mapping = %+v", store.saved)
	}
}

func TestFundResolverReturnsTargetCodeWhenCachedResolvedTargetHasNoHoldings(t *testing.T) {
	store := &stubFundMappingStore{
		mapping: &database.FundMapping{
			FeederCode: "010524",
			FeederName: "银华中证5G通信主题ETF联接C",
			TargetCode: "159994",
			IsResolved: true,
		},
	}
	repo := repository.NewMemoryFundRepository()
	resolver := &FundResolver{
		mappingStore: store,
		fundRepo:     repo,
		loadDetailHints: func(ctx context.Context, fundCode string) (*fundDetailResolutionHints, error) {
			t.Fatalf("detail hints should not be loaded when resolved cache exists")
			return nil, nil
		},
	}

	holdings, source, err := resolver.GetHoldingsWithFallback(context.Background(), "010524", "银华中证5G通信主题ETF联接C")
	if err != nil {
		t.Fatalf("GetHoldingsWithFallback() error = %v", err)
	}
	if source != "159994" {
		t.Fatalf("source = %s, want 159994", source)
	}
	if len(holdings) != 0 {
		t.Fatalf("holdings len = %d, want 0 for direct quote fallback", len(holdings))
	}
}

func TestFundResolverTreatsZeroRatioHoldingsAsNoEffectiveHoldings(t *testing.T) {
	store := &stubFundMappingStore{
		mapping: &database.FundMapping{
			FeederCode: "020465",
			FeederName: "招商中证半导体产业ETF发起式联接C",
			TargetCode: "561980",
			IsResolved: true,
		},
	}
	repo := repository.NewMemoryFundRepository()
	if err := repo.SaveHoldings(context.Background(), "020465", []domain.StockHolding{
		{
			StockCode:    "688012",
			StockName:    "中微公司",
			Exchange:     domain.ExchangeSH,
			HoldingRatio: decimal.Zero,
		},
	}); err != nil {
		t.Fatalf("SaveHoldings() source error = %v", err)
	}
	if err := repo.SaveHoldings(context.Background(), "561980", []domain.StockHolding{
		{
			StockCode:    "688256",
			StockName:    "寒武纪",
			Exchange:     domain.ExchangeSH,
			HoldingRatio: decimal.RequireFromString("9.27"),
		},
	}); err != nil {
		t.Fatalf("SaveHoldings() target error = %v", err)
	}

	resolver := &FundResolver{
		mappingStore: store,
		fundRepo:     repo,
	}

	holdings, source, err := resolver.GetHoldingsWithFallback(context.Background(), "020465", "招商中证半导体产业ETF发起式联接C")
	if err != nil {
		t.Fatalf("GetHoldingsWithFallback() error = %v", err)
	}
	if source != "561980" {
		t.Fatalf("source = %s, want 561980", source)
	}
	if len(holdings) != 1 {
		t.Fatalf("holdings len = %d, want 1", len(holdings))
	}
	if holdings[0].StockCode != "688256" {
		t.Fatalf("holding stock code = %s, want 688256", holdings[0].StockCode)
	}
}

func TestParseFundDetailResolutionHintsHTML(t *testing.T) {
	html := `银华中证5G通信主题ETF联接C<a style='float: right;' href="http://fund.eastmoney.com/159994.html">查看相关ETF></a><tr><td class='specialData'><a href="http://fundf10.eastmoney.com/tsdata_010524.html">跟踪标的：</a>中证5G通信主题指数 | <a href="http://fundf10.eastmoney.com/tsdata_010524.html">年化跟踪误差：</a>2.81%</td></tr>`

	hints := parseFundDetailResolutionHintsHTML(html)
	if hints == nil {
		t.Fatalf("expected hints, got nil")
	}
	if hints.RelatedETFCode != "159994" {
		t.Fatalf("related ETF code = %s, want 159994", hints.RelatedETFCode)
	}
	if hints.TrackingTarget != "中证5G通信主题指数" {
		t.Fatalf("tracking target = %s, want 中证5G通信主题指数", hints.TrackingTarget)
	}
}
