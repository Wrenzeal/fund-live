package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/RomaticDOG/fund/internal/crawler"
	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
)

type historicalHoldingsCacheEntry struct {
	expiresAt time.Time
	period    string
	holdings  []domain.StockHolding
}

type HistoricalHoldingsProvider struct {
	fetcher holdingsYearFetcher
	store   historicalHoldingsStore
	ttl     time.Duration

	mu    sync.Mutex
	cache map[string]historicalHoldingsCacheEntry
}

type holdingsYearFetcher interface {
	FetchHoldingsByYear(ctx context.Context, fundCode string, year int) (map[string][]domain.StockHolding, error)
}

func NewHistoricalHoldingsProvider() *HistoricalHoldingsProvider {
	return &HistoricalHoldingsProvider{
		fetcher: crawler.NewCrawlService(1),
		ttl:     6 * time.Hour,
		cache:   make(map[string]historicalHoldingsCacheEntry),
	}
}

var defaultHistoricalHoldingsProvider = NewHistoricalHoldingsProvider()

func LoadPreviousQuarterHoldings(ctx context.Context, fundCode string, currentHoldings []domain.StockHolding) ([]domain.StockHolding, string, error) {
	return defaultHistoricalHoldingsProvider.LoadPreviousQuarterHoldings(ctx, fundCode, currentHoldings)
}

func (p *HistoricalHoldingsProvider) LoadPreviousQuarterHoldings(ctx context.Context, fundCode string, currentHoldings []domain.StockHolding) ([]domain.StockHolding, string, error) {
	latestPeriod, _ := latestHoldingPeriod(currentHoldings)
	if latestPeriod == "" {
		return nil, "", nil
	}

	previousPeriod, ok := previousReportingPeriod(latestPeriod)
	if !ok {
		return nil, "", nil
	}

	cacheKey := fundCode + "|" + previousPeriod
	now := time.Now()

	p.mu.Lock()
	if entry, ok := p.cache[cacheKey]; ok && entry.expiresAt.After(now) {
		holdings := cloneAnalysisHoldings(entry.holdings)
		p.mu.Unlock()
		return holdings, entry.period, nil
	}
	p.mu.Unlock()

	store := p.ensureStore()
	if store != nil {
		if persisted, err := store.LoadByPeriod(ctx, fundCode, previousPeriod); err == nil && len(persisted) > 0 {
			p.mu.Lock()
			p.cache[cacheKey] = historicalHoldingsCacheEntry{
				expiresAt: now.Add(p.ttl),
				period:    previousPeriod,
				holdings:  cloneAnalysisHoldings(persisted),
			}
			p.mu.Unlock()
			return persisted, previousPeriod, nil
		}
	}

	previousDate, ok := parseReportingPeriod(previousPeriod)
	if !ok {
		return nil, "", fmt.Errorf("parse previous reporting period %q failed", previousPeriod)
	}

	holdingsByPeriod, err := p.fetcher.FetchHoldingsByYear(ctx, fundCode, previousDate.Year())
	if err != nil {
		return nil, "", err
	}
	if store != nil {
		if err := store.SaveHistoryPeriods(ctx, fundCode, holdingsByPeriod, "eastmoney_year_page"); err != nil {
			return nil, "", err
		}
	}

	holdings := cloneAnalysisHoldings(holdingsByPeriod[previousPeriod])
	if len(holdings) == 0 {
		return nil, previousPeriod, nil
	}

	p.mu.Lock()
	p.cache[cacheKey] = historicalHoldingsCacheEntry{
		expiresAt: now.Add(p.ttl),
		period:    previousPeriod,
		holdings:  cloneAnalysisHoldings(holdings),
	}
	p.mu.Unlock()

	return holdings, previousPeriod, nil
}

func (p *HistoricalHoldingsProvider) ensureStore() historicalHoldingsStore {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.store != nil {
		return p.store
	}
	if db := database.GetDB(); db != nil {
		p.store = NewHistoricalHoldingsSnapshotStore(db)
	}
	return p.store
}

func previousReportingPeriod(currentPeriod string) (string, bool) {
	currentDate, ok := parseReportingPeriod(currentPeriod)
	if !ok {
		return "", false
	}

	switch currentDate.Month() {
	case time.March:
		return time.Date(currentDate.Year()-1, time.December, 31, 0, 0, 0, 0, currentDate.Location()).Format("2006-01-02"), true
	case time.June:
		return time.Date(currentDate.Year(), time.March, 31, 0, 0, 0, 0, currentDate.Location()).Format("2006-01-02"), true
	case time.September:
		return time.Date(currentDate.Year(), time.June, 30, 0, 0, 0, 0, currentDate.Location()).Format("2006-01-02"), true
	case time.December:
		return time.Date(currentDate.Year(), time.September, 30, 0, 0, 0, 0, currentDate.Location()).Format("2006-01-02"), true
	default:
		return "", false
	}
}

func cloneAnalysisHoldings(holdings []domain.StockHolding) []domain.StockHolding {
	if len(holdings) == 0 {
		return nil
	}
	cloned := make([]domain.StockHolding, len(holdings))
	copy(cloned, holdings)
	return cloned
}
