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
