package service

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RomaticDOG/fund/internal/crawler"
	"github.com/RomaticDOG/fund/internal/domain"
	"golang.org/x/sync/errgroup"
)

// FundHoldingsRefreshService refreshes persisted holdings for funds that already have holdings records.
// It is intended for low-frequency maintenance refreshes (for example, monthly).
type FundHoldingsRefreshService struct {
	fundRepo        domain.FundRepository
	crawler         *crawler.CrawlService
	location        *time.Location
	now             func() time.Time
	maxConcurrency  int
	refreshDay      int
	refreshHour     int
	lastRunMu       sync.Mutex
	lastRunPeriodID string
}

func NewFundHoldingsRefreshService(fundRepo domain.FundRepository) *FundHoldingsRefreshService {
	return &FundHoldingsRefreshService{
		fundRepo:       fundRepo,
		crawler:        crawler.NewCrawlService(3),
		location:       tradingLocation(),
		now:            time.Now,
		maxConcurrency: 3,
		refreshDay:     1,
		refreshHour:    1,
	}
}

func (s *FundHoldingsRefreshService) Start(ctx context.Context) {
	if s == nil || s.fundRepo == nil {
		return
	}

	go s.run(ctx)
}

func (s *FundHoldingsRefreshService) run(ctx context.Context) {
	if s.shouldRunImmediately() {
		if err := s.RefreshExistingFunds(ctx); err != nil {
			log.Printf("⚠️ Monthly holdings refresh on startup failed: %v", err)
		}
	}

	for {
		nextRun := s.nextRunAt(s.now())
		timer := time.NewTimer(time.Until(nextRun))

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if err := s.RefreshExistingFunds(ctx); err != nil {
				log.Printf("⚠️ Monthly holdings refresh failed: %v", err)
			}
		}
	}
}

func (s *FundHoldingsRefreshService) shouldRunImmediately() bool {
	now := s.now().In(s.location)
	if now.Day() != s.refreshDay || now.Hour() < s.refreshHour {
		return false
	}

	periodID := now.Format("2006-01")
	s.lastRunMu.Lock()
	defer s.lastRunMu.Unlock()
	if s.lastRunPeriodID == periodID {
		return false
	}
	return true
}

func (s *FundHoldingsRefreshService) nextRunAt(now time.Time) time.Time {
	localNow := now.In(s.location)
	year := localNow.Year()
	month := localNow.Month()

	next := time.Date(year, month, s.refreshDay, s.refreshHour, 0, 0, 0, s.location)
	if !localNow.Before(next) {
		next = next.AddDate(0, 1, 0)
		next = time.Date(next.Year(), next.Month(), s.refreshDay, s.refreshHour, 0, 0, 0, s.location)
	}

	return next
}

func (s *FundHoldingsRefreshService) markRun(periodID string) {
	s.lastRunMu.Lock()
	s.lastRunPeriodID = periodID
	s.lastRunMu.Unlock()
}

func (s *FundHoldingsRefreshService) RefreshExistingFunds(ctx context.Context) error {
	if s == nil || s.fundRepo == nil {
		return nil
	}

	fundIDs, err := s.fundRepo.ListFundIDsWithHoldings(ctx)
	if err != nil {
		return err
	}
	if len(fundIDs) == 0 {
		log.Printf("ℹ️ Monthly holdings refresh skipped: no funds with existing holdings")
		return nil
	}

	now := s.now().In(s.location)
	periodID := now.Format("2006-01")
	log.Printf("🗓️ Monthly holdings refresh started for %d funds with persisted holdings", len(fundIDs))

	var refreshedFundCount atomic.Int64
	var refreshedHoldingCount atomic.Int64
	var skippedHoldingCount atomic.Int64
	var failureCount atomic.Int64

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(s.maxConcurrency)

	for _, fundID := range fundIDs {
		fundID := fundID
		group.Go(func() error {
			fund, holdings, fetchErr := s.crawler.FetchFundData(groupCtx, fundID)
			if fetchErr != nil {
				failureCount.Add(1)
				log.Printf("⚠️ Monthly holdings refresh: fetch %s failed: %v", fundID, fetchErr)
				return nil
			}

			if fund != nil {
				if saveErr := s.fundRepo.SaveFund(groupCtx, fund); saveErr != nil {
					failureCount.Add(1)
					log.Printf("⚠️ Monthly holdings refresh: save fund %s failed: %v", fundID, saveErr)
					return nil
				}
				refreshedFundCount.Add(1)
			}

			if holdings == nil {
				skippedHoldingCount.Add(1)
				log.Printf("⚠️ Monthly holdings refresh: skip replacing holdings for %s because upstream holdings fetch returned nil", fundID)
				return nil
			}

			if saveErr := s.fundRepo.SaveHoldings(groupCtx, fundID, holdings); saveErr != nil {
				failureCount.Add(1)
				log.Printf("⚠️ Monthly holdings refresh: save holdings %s failed: %v", fundID, saveErr)
				return nil
			}
			refreshedHoldingCount.Add(1)
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return err
	}

	s.markRun(periodID)
	log.Printf(
		"✅ Monthly holdings refresh completed: funds=%d holdings=%d skipped_holdings=%d failed=%d",
		refreshedFundCount.Load(),
		refreshedHoldingCount.Load(),
		skippedHoldingCount.Load(),
		failureCount.Load(),
	)

	return nil
}
