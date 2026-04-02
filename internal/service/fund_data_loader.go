package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/RomaticDOG/fund/internal/crawler"
	"github.com/RomaticDOG/fund/internal/domain"
)

type fundDataFetcher interface {
	FetchFundData(ctx context.Context, fundID string) (*domain.Fund, []domain.StockHolding, error)
}

type cachedFundData struct {
	fund      *domain.Fund
	holdings  []domain.StockHolding
	expiresAt time.Time
}

// FundDataLoader fetches missing fund data on demand.
// Read paths use the transient cache only; explicit hydration persists via FundRepository.
type FundDataLoader struct {
	fundRepo domain.FundRepository
	fetcher  fundDataFetcher
	cacheTTL time.Duration

	cacheMu sync.RWMutex
	cache   map[string]cachedFundData
}

// NewFundDataLoader creates a loader for on-demand fund hydration.
func NewFundDataLoader(fundRepo domain.FundRepository) *FundDataLoader {
	return NewFundDataLoaderWithFetcher(fundRepo, crawler.NewCrawlService(1), 10*time.Minute)
}

// NewFundDataLoaderWithFetcher creates a loader with an injected fetcher and cache TTL.
func NewFundDataLoaderWithFetcher(fundRepo domain.FundRepository, fetcher fundDataFetcher, cacheTTL time.Duration) *FundDataLoader {
	if fetcher == nil {
		fetcher = crawler.NewCrawlService(1)
	}
	if cacheTTL <= 0 {
		cacheTTL = 10 * time.Minute
	}
	return &FundDataLoader{
		fundRepo: fundRepo,
		fetcher:  fetcher,
		cacheTTL: cacheTTL,
		cache:    make(map[string]cachedFundData),
	}
}

// FetchTransientFundData fetches the latest fund profile and holdings without writing them to the repository.
// Results are cached in-process so repeated read requests do not repeatedly hit upstream providers.
func (l *FundDataLoader) FetchTransientFundData(ctx context.Context, fundID string) (*domain.Fund, []domain.StockHolding, error) {
	if l == nil {
		return nil, nil, fmt.Errorf("fund data loader is nil")
	}

	fundID = strings.TrimSpace(fundID)
	if fundID == "" {
		return nil, nil, fmt.Errorf("fund id is required")
	}

	now := time.Now()
	if fund, holdings, ok := l.loadCached(fundID, now); ok {
		return fund, holdings, nil
	}

	fund, holdings, err := l.fetcher.FetchFundData(ctx, fundID)
	if err != nil {
		return nil, nil, err
	}
	if fund == nil {
		return nil, nil, fmt.Errorf("crawler returned nil fund for %s", fundID)
	}

	l.storeCache(fundID, fund, holdings, now)
	return cloneFund(fund), cloneHoldings(holdings), nil
}

// EnsureFundData fetches the latest fund profile and holdings, then stores them in the repository.
func (l *FundDataLoader) EnsureFundData(ctx context.Context, fundID string) (*domain.Fund, []domain.StockHolding, error) {
	fund, holdings, err := l.FetchTransientFundData(ctx, fundID)
	if err != nil {
		return nil, nil, err
	}

	if err := l.fundRepo.SaveFund(ctx, fund); err != nil {
		return nil, nil, fmt.Errorf("failed to save hydrated fund %s: %w", fundID, err)
	}

	if len(holdings) > 0 {
		if err := l.fundRepo.SaveHoldings(ctx, fundID, holdings); err != nil {
			return nil, nil, fmt.Errorf("failed to save hydrated holdings for %s: %w", fundID, err)
		}
	}

	freshFund, err := l.fundRepo.GetFundByID(ctx, fundID)
	if err != nil {
		log.Printf("⚠️ Failed to reload hydrated fund %s: %v", fundID, err)
		freshFund = fund
	}

	freshHoldings, err := l.fundRepo.GetFundHoldings(ctx, fundID)
	if err != nil {
		log.Printf("⚠️ Failed to reload hydrated holdings %s: %v", fundID, err)
		freshHoldings = holdings
	}

	return freshFund, freshHoldings, nil
}

func (l *FundDataLoader) loadCached(fundID string, now time.Time) (*domain.Fund, []domain.StockHolding, bool) {
	l.cacheMu.RLock()
	cached, ok := l.cache[fundID]
	l.cacheMu.RUnlock()
	if !ok {
		return nil, nil, false
	}
	if now.After(cached.expiresAt) {
		l.cacheMu.Lock()
		delete(l.cache, fundID)
		l.cacheMu.Unlock()
		return nil, nil, false
	}
	return cloneFund(cached.fund), cloneHoldings(cached.holdings), true
}

func (l *FundDataLoader) storeCache(fundID string, fund *domain.Fund, holdings []domain.StockHolding, now time.Time) {
	if l.cacheTTL <= 0 {
		return
	}

	l.cacheMu.Lock()
	l.cache[fundID] = cachedFundData{
		fund:      cloneFund(fund),
		holdings:  cloneHoldings(holdings),
		expiresAt: now.Add(l.cacheTTL),
	}
	l.cacheMu.Unlock()
}

func cloneFund(fund *domain.Fund) *domain.Fund {
	if fund == nil {
		return nil
	}
	copyFund := *fund
	return &copyFund
}

func cloneHoldings(holdings []domain.StockHolding) []domain.StockHolding {
	if len(holdings) == 0 {
		return nil
	}

	copied := make([]domain.StockHolding, len(holdings))
	copy(copied, holdings)
	return copied
}

func needsRuntimeFundData(fund *domain.Fund, holdings []domain.StockHolding) bool {
	if fund == nil {
		return true
	}
	return fund.NetAssetVal.IsZero() || len(holdings) == 0
}
