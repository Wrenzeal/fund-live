package service

import (
	"context"
	"testing"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

type fakeHoldingsYearFetcher struct {
	calls  int
	result map[string][]domain.StockHolding
	err    error
}

func (f *fakeHoldingsYearFetcher) FetchHoldingsByYear(ctx context.Context, fundCode string, year int) (map[string][]domain.StockHolding, error) {
	f.calls++
	return f.result, f.err
}

type fakeHistoricalHoldingsStore struct {
	loaded map[string][]domain.StockHolding
	saved  map[string][]domain.StockHolding
}

func (f *fakeHistoricalHoldingsStore) LoadByPeriod(ctx context.Context, fundCode string, reportingPeriod string) ([]domain.StockHolding, error) {
	if f.loaded == nil {
		return nil, nil
	}
	return cloneAnalysisHoldings(f.loaded[fundCode+"|"+reportingPeriod]), nil
}

func (f *fakeHistoricalHoldingsStore) SaveHistoryPeriods(ctx context.Context, fundCode string, holdingsByPeriod map[string][]domain.StockHolding, source string) error {
	if f.saved == nil {
		f.saved = make(map[string][]domain.StockHolding)
	}
	for period, holdings := range holdingsByPeriod {
		f.saved[fundCode+"|"+period] = cloneAnalysisHoldings(holdings)
	}
	return nil
}

func TestHistoricalHoldingsProviderUsesPersistedSnapshotFirst(t *testing.T) {
	fetcher := &fakeHoldingsYearFetcher{}
	store := &fakeHistoricalHoldingsStore{
		loaded: map[string][]domain.StockHolding{
			"159813|2025-12-31": {
				{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("9.9900"), ReportingPeriod: "2025-12-31"},
			},
		},
	}
	provider := &HistoricalHoldingsProvider{
		fetcher: fetcher,
		store:   store,
		cache:   make(map[string]historicalHoldingsCacheEntry),
	}

	holdings, period, err := provider.LoadPreviousQuarterHoldings(context.Background(), "159813", []domain.StockHolding{
		{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("10.0000"), ReportingPeriod: "2026-03-31"},
	})
	if err != nil {
		t.Fatalf("LoadPreviousQuarterHoldings() error = %v", err)
	}
	if period != "2025-12-31" {
		t.Fatalf("period = %s, want 2025-12-31", period)
	}
	if len(holdings) != 1 || holdings[0].StockCode != "688041" {
		t.Fatalf("holdings = %+v, want persisted holdings", holdings)
	}
	if fetcher.calls != 0 {
		t.Fatalf("fetcher calls = %d, want 0", fetcher.calls)
	}
}

func TestHistoricalHoldingsProviderPersistsFetchedHistory(t *testing.T) {
	fetcher := &fakeHoldingsYearFetcher{
		result: map[string][]domain.StockHolding{
			"2025-12-31": {
				{StockCode: "688981", StockName: "中芯国际", HoldingRatio: decimal.RequireFromString("8.1200"), ReportingPeriod: "2025-12-31"},
			},
		},
	}
	store := &fakeHistoricalHoldingsStore{}
	provider := &HistoricalHoldingsProvider{
		fetcher: fetcher,
		store:   store,
		cache:   make(map[string]historicalHoldingsCacheEntry),
	}

	holdings, period, err := provider.LoadPreviousQuarterHoldings(context.Background(), "159813", []domain.StockHolding{
		{StockCode: "688041", StockName: "海光信息", HoldingRatio: decimal.RequireFromString("10.0000"), ReportingPeriod: "2026-03-31"},
	})
	if err != nil {
		t.Fatalf("LoadPreviousQuarterHoldings() error = %v", err)
	}
	if period != "2025-12-31" {
		t.Fatalf("period = %s, want 2025-12-31", period)
	}
	if len(holdings) != 1 || holdings[0].StockCode != "688981" {
		t.Fatalf("holdings = %+v, want fetched holdings", holdings)
	}
	if fetcher.calls != 1 {
		t.Fatalf("fetcher calls = %d, want 1", fetcher.calls)
	}
	if len(store.saved["159813|2025-12-31"]) != 1 {
		t.Fatalf("saved snapshot = %+v, want persisted period", store.saved)
	}
}
