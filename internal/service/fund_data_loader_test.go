package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/shopspring/decimal"
)

type stubFundDataFetcher struct {
	calls    int
	fund     *domain.Fund
	holdings []domain.StockHolding
	err      error
}

func (s *stubFundDataFetcher) FetchFundData(ctx context.Context, fundID string) (*domain.Fund, []domain.StockHolding, error) {
	s.calls++
	if s.err != nil {
		return nil, nil, s.err
	}
	return cloneFund(s.fund), cloneHoldings(s.holdings), nil
}

type countingPersistFundRepository struct {
	*repository.MemoryFundRepository

	saveFundCalls     int
	saveHoldingsCalls int
}

func newCountingPersistFundRepository() *countingPersistFundRepository {
	return &countingPersistFundRepository{
		MemoryFundRepository: repository.NewMemoryFundRepository(),
	}
}

func (r *countingPersistFundRepository) SaveFund(ctx context.Context, fund *domain.Fund) error {
	r.saveFundCalls++
	return r.MemoryFundRepository.SaveFund(ctx, fund)
}

func (r *countingPersistFundRepository) SaveHoldings(ctx context.Context, fundID string, holdings []domain.StockHolding) error {
	r.saveHoldingsCalls++
	return r.MemoryFundRepository.SaveHoldings(ctx, fundID, holdings)
}

func TestFundDataLoaderFetchTransientFundDataDoesNotPersist(t *testing.T) {
	repo := newCountingPersistFundRepository()
	fetcher := &stubFundDataFetcher{
		fund: &domain.Fund{
			ID:          "123456",
			Name:        "测试基金",
			Type:        "hybrid",
			NetAssetVal: decimal.RequireFromString("1.2345"),
			UpdatedAt:   time.Now(),
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
	loader := &FundDataLoader{
		fundRepo: repo,
		fetcher:  fetcher,
		cacheTTL: time.Minute,
		fetchTTL: time.Second,
		cache:    make(map[string]cachedFundData),
	}

	firstFund, firstHoldings, err := loader.FetchTransientFundData(context.Background(), "123456")
	if err != nil {
		t.Fatalf("FetchTransientFundData() error = %v", err)
	}
	secondFund, secondHoldings, err := loader.FetchTransientFundData(context.Background(), "123456")
	if err != nil {
		t.Fatalf("FetchTransientFundData() second call error = %v", err)
	}

	if fetcher.calls != 1 {
		t.Fatalf("fetcher calls = %d, want 1", fetcher.calls)
	}
	if repo.saveFundCalls != 0 {
		t.Fatalf("SaveFund() calls = %d, want 0", repo.saveFundCalls)
	}
	if repo.saveHoldingsCalls != 0 {
		t.Fatalf("SaveHoldings() calls = %d, want 0", repo.saveHoldingsCalls)
	}
	if firstFund == nil || secondFund == nil || firstFund.ID != "123456" || secondFund.ID != "123456" {
		t.Fatalf("unexpected fund results: first=%+v second=%+v", firstFund, secondFund)
	}
	if len(firstHoldings) != 1 || len(secondHoldings) != 1 {
		t.Fatalf("unexpected holdings lengths: first=%d second=%d", len(firstHoldings), len(secondHoldings))
	}
}

func TestFundDataLoaderEnsureFundDataPersistsFetchedResult(t *testing.T) {
	repo := newCountingPersistFundRepository()
	fetcher := &stubFundDataFetcher{
		fund: &domain.Fund{
			ID:          "654321",
			Name:        "落库基金",
			Type:        "hybrid",
			NetAssetVal: decimal.RequireFromString("2.3456"),
			UpdatedAt:   time.Now(),
		},
		holdings: []domain.StockHolding{
			{
				StockCode:    "000858",
				StockName:    "五粮液",
				Exchange:     domain.ExchangeSZ,
				HoldingRatio: decimal.RequireFromString("8.80"),
			},
		},
	}
	loader := &FundDataLoader{
		fundRepo: repo,
		fetcher:  fetcher,
		cacheTTL: time.Minute,
		fetchTTL: time.Second,
		cache:    make(map[string]cachedFundData),
	}

	fund, holdings, err := loader.EnsureFundData(context.Background(), "654321")
	if err != nil {
		t.Fatalf("EnsureFundData() error = %v", err)
	}

	if fetcher.calls != 1 {
		t.Fatalf("fetcher calls = %d, want 1", fetcher.calls)
	}
	if repo.saveFundCalls != 1 {
		t.Fatalf("SaveFund() calls = %d, want 1", repo.saveFundCalls)
	}
	if repo.saveHoldingsCalls != 1 {
		t.Fatalf("SaveHoldings() calls = %d, want 1", repo.saveHoldingsCalls)
	}
	if fund == nil || fund.ID != "654321" {
		t.Fatalf("fund = %+v", fund)
	}
	if len(holdings) != 1 || holdings[0].StockCode != "000858" {
		t.Fatalf("holdings = %+v", holdings)
	}
}

func TestNeedsRuntimeFundDataIgnoresMissingDisplayFields(t *testing.T) {
	fund := &domain.Fund{
		ID:          "005827",
		Name:        "易方达蓝筹精选混合",
		Type:        "hybrid",
		Manager:     "",
		Company:     "",
		NetAssetVal: decimal.RequireFromString("1.8888"),
	}
	holdings := []domain.StockHolding{
		{
			StockCode:    "600519",
			StockName:    "贵州茅台",
			Exchange:     domain.ExchangeSH,
			HoldingRatio: decimal.RequireFromString("9.90"),
		},
	}

	if needsRuntimeFundData(fund, holdings) {
		t.Fatalf("needsRuntimeFundData() = true, want false when NAV and holdings already exist")
	}
}

func TestFundDataLoaderDeduplicatesConcurrentTransientFetches(t *testing.T) {
	repo := newCountingPersistFundRepository()
	fetcher := &stubFundDataFetcher{
		fund: &domain.Fund{
			ID:          "777777",
			Name:        "并发基金",
			Type:        "hybrid",
			NetAssetVal: decimal.RequireFromString("1.1111"),
			UpdatedAt:   time.Now(),
		},
	}
	loader := &FundDataLoader{
		fundRepo: repo,
		fetcher:  fetcher,
		cacheTTL: time.Minute,
		fetchTTL: time.Second,
		cache:    make(map[string]cachedFundData),
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := loader.FetchTransientFundData(context.Background(), "777777")
			errs <- err
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("FetchTransientFundData() error = %v", err)
		}
	}
	if fetcher.calls != 1 {
		t.Fatalf("fetcher calls = %d, want 1", fetcher.calls)
	}
}
