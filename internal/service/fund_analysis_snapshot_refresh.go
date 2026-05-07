package service

import (
	"context"
	"log"
	"time"
)

type analysisSnapshotCandidateProvider interface {
	ListRankingCandidateFundIDs(ctx context.Context, limit int) ([]string, error)
}

type FundAnalysisSnapshotRefreshService struct {
	candidateProvider analysisSnapshotCandidateProvider
	coordinator       *FundAnalysisCoordinator
	snapshotStore     *FundAnalysisSnapshotStore
	now               func() time.Time
	lastRunDate       string
}

func NewFundAnalysisSnapshotRefreshService(
	candidateProvider analysisSnapshotCandidateProvider,
	coordinator *FundAnalysisCoordinator,
	snapshotStore *FundAnalysisSnapshotStore,
) *FundAnalysisSnapshotRefreshService {
	return &FundAnalysisSnapshotRefreshService{
		candidateProvider: candidateProvider,
		coordinator:       coordinator,
		snapshotStore:     snapshotStore,
		now:               time.Now,
	}
}

func (s *FundAnalysisSnapshotRefreshService) Start(ctx context.Context) {
	if s == nil || s.candidateProvider == nil || s.coordinator == nil || s.snapshotStore == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for {
			s.maybeRun(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *FundAnalysisSnapshotRefreshService) maybeRun(ctx context.Context) {
	now := s.now()
	loc := time.FixedZone("CST", 8*3600)
	localNow := now.In(loc)
	if localNow.Hour() < 23 {
		return
	}
	runDate := localNow.Format("2006-01-02")
	if s.lastRunDate == runDate {
		return
	}

	fundIDs, err := s.candidateProvider.ListRankingCandidateFundIDs(ctx, 120)
	if err != nil {
		log.Printf("⚠️ analysis snapshot refresh list candidates failed: %v", err)
		return
	}
	successCount := 0
	for _, fundID := range fundIDs {
		_, analysis, buildErr := s.coordinator.BuildForFund(ctx, fundID, now)
		if buildErr != nil || analysis == nil {
			continue
		}
		if saveErr := s.snapshotStore.Save(ctx, fundID, analysis, now); saveErr != nil {
			log.Printf("⚠️ analysis snapshot refresh save %s failed: %v", fundID, saveErr)
			continue
		}
		successCount++
	}
	s.lastRunDate = runDate
	log.Printf("🧠 Fund analysis snapshot refresh completed | date=%s saved=%d", runDate, successCount)
}
