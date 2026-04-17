package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/RomaticDOG/fund/internal/crawler"
	"github.com/RomaticDOG/fund/internal/domain"
	"golang.org/x/sync/singleflight"
)

var ErrFundDataWarmupInProgress = errors.New("fund data warmup in progress")

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
	fundRepo  domain.FundRepository
	fetcher   fundDataFetcher
	cacheTTL  time.Duration
	fetchTTL  time.Duration
	ensureTTL time.Duration

	cacheMu sync.RWMutex
	cache   map[string]cachedFundData
	group   singleflight.Group

	warmMu  sync.Mutex
	warming map[string]struct{}
}

// NewFundDataLoader creates a loader for on-demand fund hydration.
func NewFundDataLoader(fundRepo domain.FundRepository) *FundDataLoader {
	return NewFundDataLoaderWithFetcher(fundRepo, crawler.NewCrawlService(1), 10*time.Minute, 20*time.Second)
}

// NewFundDataLoaderWithFetcher creates a loader with an injected fetcher, cache TTL and fetch timeout.
func NewFundDataLoaderWithFetcher(fundRepo domain.FundRepository, fetcher fundDataFetcher, cacheTTL, fetchTTL time.Duration) *FundDataLoader {
	if fetcher == nil {
		fetcher = crawler.NewCrawlService(1)
	}
	if cacheTTL <= 0 {
		cacheTTL = 10 * time.Minute
	}
	if fetchTTL <= 0 {
		fetchTTL = 20 * time.Second
	}
	ensureTTL := fetchTTL + 10*time.Second
	if ensureTTL < 30*time.Second {
		ensureTTL = 30 * time.Second
	}
	return &FundDataLoader{
		fundRepo:  fundRepo,
		fetcher:   fetcher,
		cacheTTL:  cacheTTL,
		fetchTTL:  fetchTTL,
		ensureTTL: ensureTTL,
		cache:     make(map[string]cachedFundData),
		warming:   make(map[string]struct{}),
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

	type result struct {
		fund     *domain.Fund
		holdings []domain.StockHolding
	}
	loaded, err, _ := l.group.Do(fundID, func() (interface{}, error) {
		if cachedFund, cachedHoldings, ok := l.loadCached(fundID, time.Now()); ok {
			return result{fund: cachedFund, holdings: cachedHoldings}, nil
		}

		fetchCtx := ctx
		cancel := func() {}
		if l.fetchTTL > 0 {
			fetchCtx, cancel = context.WithTimeout(ctx, l.fetchTTL)
		}
		defer cancel()

		fund, holdings, err := l.fetcher.FetchFundData(fetchCtx, fundID)
		if err != nil {
			return nil, err
		}
		if fund == nil {
			return nil, fmt.Errorf("crawler returned nil fund for %s", fundID)
		}

		l.storeCache(fundID, fund, holdings, time.Now())
		return result{fund: cloneFund(fund), holdings: cloneHoldings(holdings)}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	typed := loaded.(result)
	return cloneFund(typed.fund), cloneHoldings(typed.holdings), nil
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

// PeekCachedFundData returns transient cached data without triggering upstream fetches.
func (l *FundDataLoader) PeekCachedFundData(fundID string) (*domain.Fund, []domain.StockHolding, bool) {
	if l == nil {
		return nil, nil, false
	}

	fundID = strings.TrimSpace(fundID)
	if fundID == "" {
		return nil, nil, false
	}

	return l.loadCached(fundID, time.Now())
}

// ScheduleEnsureFundData enqueues a background hydration job and deduplicates inflight requests.
// It returns true when a warmup job is active or has just been scheduled.
func (l *FundDataLoader) ScheduleEnsureFundData(fundID string) bool {
	if l == nil {
		return false
	}

	fundID = strings.TrimSpace(fundID)
	if fundID == "" {
		return false
	}

	l.warmMu.Lock()
	if l.warming == nil {
		l.warming = make(map[string]struct{})
	}
	if _, exists := l.warming[fundID]; exists {
		l.warmMu.Unlock()
		return true
	}
	l.warming[fundID] = struct{}{}
	l.warmMu.Unlock()

	go func() {
		defer l.finishWarmup(fundID)

		ctx := context.Background()
		cancel := func() {}
		if l.ensureTTL > 0 {
			ctx, cancel = context.WithTimeout(context.Background(), l.ensureTTL)
		}
		defer cancel()

		if _, _, err := l.EnsureFundData(ctx, fundID); err != nil {
			log.Printf("⚠️ Background fund warmup failed for %s: %v", fundID, err)
			return
		}
		log.Printf("✅ Background fund warmup completed for %s", fundID)
	}()

	return true
}

func (l *FundDataLoader) finishWarmup(fundID string) {
	l.warmMu.Lock()
	delete(l.warming, fundID)
	l.warmMu.Unlock()
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
	return fund.NetAssetVal.IsZero() || !hasEffectiveHoldings(holdings)
}

func useCachedFundDataOrScheduleWarmup(loader *FundDataLoader, fundID string, fund *domain.Fund, holdings []domain.StockHolding) (*domain.Fund, []domain.StockHolding, bool) {
	if loader == nil || !needsRuntimeFundData(fund, holdings) {
		return fund, holdings, false
	}

	if cachedFund, cachedHoldings, ok := loader.PeekCachedFundData(fundID); ok {
		if cachedFund != nil {
			fund = cachedFund
		}
		if len(cachedHoldings) > 0 {
			holdings = cachedHoldings
		}
		return fund, holdings, false
	}

	return fund, holdings, loader.ScheduleEnsureFundData(fundID)
}
