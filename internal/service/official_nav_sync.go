package service

import (
	"context"
	"log"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/RomaticDOG/fund/internal/crawler"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/trading"
	"golang.org/x/sync/errgroup"
)

var officialNAVSyncLocation = trading.TradingLocation()

// OfficialNAVSyncService fetches official end-of-day NAV data and reconciles held funds.
type OfficialNAVSyncService struct {
	fundRepo        domain.FundRepository
	fundHoldingRepo domain.UserFundHoldingRepository
	favoriteRepo    domain.UserFavoriteRepository
	watchlistRepo   domain.UserWatchlistRepository
	crawler         *crawler.CrawlService
	location        *time.Location
	now             func() time.Time
	maxConcurrency  int
	syncHour        int
}

// NewOfficialNAVSyncService creates a new nightly official NAV sync service.
func NewOfficialNAVSyncService(
	fundRepo domain.FundRepository,
	fundHoldingRepo domain.UserFundHoldingRepository,
	favoriteRepo domain.UserFavoriteRepository,
	watchlistRepo domain.UserWatchlistRepository,
) *OfficialNAVSyncService {
	return &OfficialNAVSyncService{
		fundRepo:        fundRepo,
		fundHoldingRepo: fundHoldingRepo,
		favoriteRepo:    favoriteRepo,
		watchlistRepo:   watchlistRepo,
		crawler:         crawler.NewCrawlService(4),
		location:        officialNAVSyncLocation,
		now:             time.Now,
		maxConcurrency:  6,
		syncHour:        23,
	}
}

// Start launches the background loop.
func (s *OfficialNAVSyncService) Start(ctx context.Context) {
	if s == nil {
		return
	}

	go s.run(ctx)
}

func (s *OfficialNAVSyncService) run(ctx context.Context) {
	if s.shouldSyncImmediately(ctx) {
		if err := s.SyncOnce(ctx); err != nil {
			log.Printf("⚠️ Official NAV sync on startup failed: %v", err)
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
			if err := s.SyncOnce(ctx); err != nil {
				log.Printf("⚠️ Official NAV sync failed: %v", err)
			}
		}
	}
}

func (s *OfficialNAVSyncService) shouldSyncImmediately(ctx context.Context) bool {
	now := s.now().In(s.location)
	if now.Hour() < s.syncHour || !trading.IsTradingDay(now) {
		return false
	}

	fundIDs, err := s.listSyncFundIDs(ctx)
	if err != nil {
		log.Printf("⚠️ Official NAV sync startup check failed to load tracked funds: %v", err)
		return true
	}
	if len(fundIDs) == 0 {
		return false
	}

	expectedDate := trading.GetLastTradingDay(now).Format("2006-01-02")
	histories, err := s.fundRepo.GetLatestFundHistoriesByFundIDs(ctx, fundIDs)
	if err != nil {
		log.Printf("⚠️ Official NAV sync startup check failed to load histories: %v", err)
		return true
	}

	for _, fundID := range fundIDs {
		history := histories[fundID]
		if history == nil || history.Date != expectedDate {
			return true
		}
	}

	log.Printf("ℹ️ Official NAV sync already up to date for %s, skipping startup catch-up", expectedDate)
	return false
}

func (s *OfficialNAVSyncService) nextRunAt(now time.Time) time.Time {
	localNow := now.In(s.location)
	next := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), s.syncHour, 0, 0, 0, s.location)
	if !localNow.Before(next) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// SyncOnce fetches recent official NAV data for all currently tracked funds.
func (s *OfficialNAVSyncService) SyncOnce(ctx context.Context) error {
	fundIDs, err := s.listSyncFundIDs(ctx)
	if err != nil {
		return err
	}
	if len(fundIDs) == 0 {
		log.Printf("ℹ️ Official NAV sync skipped: no user holdings, favorites, or watchlist funds found")
		return nil
	}

	log.Printf("🕚 Official NAV sync started for %d tracked funds", len(fundIDs))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(s.maxConcurrency)

	var successCount atomic.Int64
	var failureCount atomic.Int64

	for _, fundID := range fundIDs {
		fundID := fundID
		g.Go(func() error {
			histories, err := s.crawler.FetchRecentFundHistory(ctx, fundID, 30)
			if err != nil {
				failureCount.Add(1)
				log.Printf("⚠️ Official NAV sync: fetch %s failed: %v", fundID, err)
				return nil
			}
			if len(histories) == 0 {
				failureCount.Add(1)
				log.Printf("⚠️ Official NAV sync: no history returned for %s", fundID)
				return nil
			}

			for index := range histories {
				history := histories[index]
				if err := s.fundRepo.SaveFundHistory(ctx, &history); err != nil {
					failureCount.Add(1)
					log.Printf("⚠️ Official NAV sync: save history %s/%s failed: %v", fundID, history.Date, err)
					return nil
				}
			}

			latestHistory := histories[len(histories)-1]
			fund, err := s.fundRepo.GetFundByID(ctx, fundID)
			if err == nil && fund != nil {
				fund.NetAssetVal = latestHistory.NetAssetVal
				fund.UpdatedAt = s.now()
				if saveErr := s.fundRepo.SaveFund(ctx, fund); saveErr != nil {
					log.Printf("⚠️ Official NAV sync: update fund nav %s failed: %v", fundID, saveErr)
				}
			}

			successCount.Add(1)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	log.Printf(
		"✅ Official NAV sync completed: %d success, %d failed",
		successCount.Load(),
		failureCount.Load(),
	)

	backfilledCount, backfillErr := s.backfillHoldingConfirmations(ctx)
	if backfillErr != nil {
		return backfillErr
	}
	if backfilledCount > 0 {
		log.Printf("🧮 Official NAV sync backfilled %d holding confirmation records", backfilledCount)
	}

	return nil
}

func (s *OfficialNAVSyncService) listSyncFundIDs(ctx context.Context) ([]string, error) {
	seen := make(map[string]struct{})

	addIDs := func(fundIDs []string) {
		for _, fundID := range fundIDs {
			fundID = strings.TrimSpace(fundID)
			if fundID == "" {
				continue
			}
			seen[fundID] = struct{}{}
		}
	}

	holdingIDs, err := s.fundHoldingRepo.ListDistinctFundIDs(ctx)
	if err != nil {
		return nil, err
	}
	addIDs(holdingIDs)

	if s.favoriteRepo != nil {
		favoriteIDs, err := s.favoriteRepo.ListDistinctFavoriteFundIDs(ctx)
		if err != nil {
			return nil, err
		}
		addIDs(favoriteIDs)
	}

	if s.watchlistRepo != nil {
		watchlistIDs, err := s.watchlistRepo.ListDistinctWatchlistFundIDs(ctx)
		if err != nil {
			return nil, err
		}
		addIDs(watchlistIDs)
	}

	fundIDs := make([]string, 0, len(seen))
	for fundID := range seen {
		fundIDs = append(fundIDs, fundID)
	}
	sort.Strings(fundIDs)
	return fundIDs, nil
}

func (s *OfficialNAVSyncService) backfillHoldingConfirmations(ctx context.Context) (int, error) {
	holdings, err := s.fundHoldingRepo.ListFundHoldingsMissingConfirmation(ctx)
	if err != nil {
		return 0, err
	}
	if len(holdings) == 0 {
		return 0, nil
	}

	historiesByKey, err := s.fundRepo.GetFundHistoriesByLookupKeys(ctx, collectHoldingHistoryLookupKeys(holdings))
	if err != nil {
		return 0, err
	}

	backfilledCount := 0
	for _, holding := range holdings {
		if !needsHoldingConfirmationData(holding) {
			continue
		}

		history := historiesByKey[holdingHistoryLookupKey(holding)]
		if history == nil {
			continue
		}

		updatedHolding := holding
		if !applyHoldingConfirmationData(&updatedHolding, history) {
			continue
		}
		updatedHolding.UpdatedAt = time.Now()

		if err := s.fundHoldingRepo.SaveFundHolding(ctx, &updatedHolding); err != nil {
			return backfilledCount, err
		}
		backfilledCount++
	}

	return backfilledCount, nil
}
