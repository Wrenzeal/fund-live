package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"golang.org/x/sync/singleflight"
)

type holdingNAVCache struct {
	rows    []domain.FundHistory
	expires time.Time
}

// HoldingNAVRefresher shares one verified source fetch across requests and keeps persistence bounded.
type HoldingNAVRefresher struct {
	repo     domain.FundRepository
	crawler  officialNAVHistoryCrawler
	mu       sync.Mutex
	cache    map[string]holdingNAVCache
	group    singleflight.Group
	requests chan struct{}
}

func NewHoldingNAVRefresher(repo domain.FundRepository, source officialNAVHistoryCrawler) *HoldingNAVRefresher {
	return &HoldingNAVRefresher{repo: repo, crawler: source, cache: make(map[string]holdingNAVCache), requests: make(chan struct{}, 4)}
}

func (s *HoldingNAVRefresher) fetch(ctx context.Context, fundID string) ([]domain.FundHistory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	cached := s.cache[fundID]
	s.mu.Unlock()
	if time.Now().Before(cached.expires) {
		return cached.rows, nil
	}
	result := s.group.DoChan(fundID, func() (interface{}, error) {
		fetchCtx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		select {
		case s.requests <- struct{}{}:
			defer func() { <-s.requests }()
		case <-fetchCtx.Done():
			return nil, fetchCtx.Err()
		}
		// The source JS already contains its full history. Retain it only in a bounded short-lived cache.
		rows, err := s.crawler.FetchRecentFundHistory(fetchCtx, fundID, 100000)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return nil, fmt.Errorf("official history unavailable")
		}
		rows = append([]domain.FundHistory(nil), rows...)
		sort.Slice(rows, func(i, j int) bool { return rows[i].Date < rows[j].Date })
		if err := validateOfficialHistory(fundID, rows); err != nil {
			return nil, err
		}
		s.mu.Lock()
		if len(s.cache) >= 16 {
			var oldest string
			var expiry time.Time
			for id, item := range s.cache {
				if oldest == "" || item.expires.Before(expiry) {
					oldest = id
					expiry = item.expires
				}
			}
			delete(s.cache, oldest)
		}
		s.cache[fundID] = holdingNAVCache{rows: rows, expires: time.Now().Add(2 * time.Minute)}
		s.mu.Unlock()
		return rows, nil
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case fetched := <-result:
		if fetched.Err != nil {
			return nil, fetched.Err
		}
		return fetched.Val.([]domain.FundHistory), nil
	}
}

func validateOfficialHistory(fundID string, rows []domain.FundHistory) error {
	seen := make(map[string]bool)
	for _, row := range rows {
		if _, err := time.Parse("2006-01-02", row.Date); err != nil {
			return err
		}
		if row.FundID != fundID || row.NetAssetVal.Sign() <= 0 || seen[row.Date] {
			return fmt.Errorf("invalid official NAV response")
		}
		seen[row.Date] = true
	}
	return nil
}

func (s *HoldingNAVRefresher) Ensure(ctx context.Context, fundID string, dates []string) error {
	rows, err := s.fetch(ctx, fundID)
	if err != nil {
		return err
	}
	start := len(rows) - 30
	if start < 0 {
		start = 0
	}
	recent := rows[start:]
	latestDate, _ := time.Parse("2006-01-02", recent[len(recent)-1].Date)
	boundedStart := latestDate.AddDate(0, 0, -90).Format("2006-01-02")
	for len(recent) > 1 && recent[0].Date < boundedStart {
		recent = recent[1:]
	}
	if err := s.repo.ReplaceFundHistoryRange(ctx, fundID, recent[0].Date, recent[len(recent)-1].Date, recent); err != nil {
		return err
	}
	byDate := make(map[string]domain.FundHistory, len(rows))
	for _, row := range rows {
		byDate[row.Date] = row
	}
	for _, date := range dates {
		row, found := byDate[date]
		if !found {
			return fmt.Errorf("official NAV unavailable for %s", date)
		}
		if date < recent[0].Date {
			if err := s.repo.SaveFundHistory(ctx, &row); err != nil {
				return err
			}
		}
	}
	return nil
}
