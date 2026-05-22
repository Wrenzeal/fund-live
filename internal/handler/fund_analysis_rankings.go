package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

const (
	analysisRankingLimit                  = 12
	analysisRankingSnapshotCandidateLimit = analysisRankingLimit * 4
)

type analysisRankingBucket string

const (
	analysisRankingBucketIncrease analysisRankingBucket = "increase"
	analysisRankingBucketWatch    analysisRankingBucket = "watch"
	analysisRankingBucketRisk     analysisRankingBucket = "risk"
)

type analysisRankingCandidate struct {
	fund     *domain.Fund
	analysis *domain.FundAnalysis
}

// GetAnalysisRankings returns a lightweight leaderboard built from the same standalone analysis surface.
// GET /api/v1/analysis/rankings
func (h *FundHandler) GetAnalysisRankings(c *gin.Context) {
	now := time.Now()
	if h.snapshotStore != nil {
		increaseRows, watchRows, riskRows, err := h.snapshotStore.ListRankings(c.Request.Context(), analysisRankingSnapshotCandidateLimit, now)
		if err == nil && len(increaseRows)+len(watchRows)+len(riskRows) > 0 {
			fundIDs := make([]string, 0, len(increaseRows)+len(watchRows)+len(riskRows))
			for _, row := range increaseRows {
				fundIDs = append(fundIDs, row.FundID)
			}
			for _, row := range watchRows {
				fundIDs = append(fundIDs, row.FundID)
			}
			for _, row := range riskRows {
				fundIDs = append(fundIDs, row.FundID)
			}
			fundMap, _ := h.fundRepo.GetFundsByIDs(c.Request.Context(), uniqueStrings(fundIDs))
			response := AnalysisRankingsResponse{
				GeneratedAt:   now,
				IncreaseIdeas: buildRankingItemsFromSnapshots(increaseRows, fundMap, analysisRankingBucketIncrease, analysisRankingLimit),
				WatchIdeas:    buildRankingItemsFromSnapshots(watchRows, fundMap, analysisRankingBucketWatch, analysisRankingLimit),
				RiskAlerts:    buildRankingItemsFromSnapshots(riskRows, fundMap, analysisRankingBucketRisk, analysisRankingLimit),
			}
			if len(response.IncreaseIdeas)+len(response.WatchIdeas)+len(response.RiskAlerts) > 0 {
				c.JSON(http.StatusOK, APIResponse{Success: true, Data: response})
				return
			}
		}
	}

	candidateIDs, err := h.listAnalysisRankingCandidateFundIDs(c.Request.Context(), 36)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "FETCH_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	candidates := make([]analysisRankingCandidate, len(candidateIDs))
	group, ctx := errgroup.WithContext(c.Request.Context())
	group.SetLimit(6)
	for index, fundID := range candidateIDs {
		index := index
		fundID := fundID
		group.Go(func() error {
			fund, analysis, buildErr := h.buildSingleFundAnalysis(ctx, fundID, now)
			if buildErr != nil {
				log.Printf("⚠️ Failed to build ranking analysis for %s: %v", fundID, buildErr)
				return nil
			}
			if fund == nil || analysis == nil {
				return nil
			}
			candidates[index] = analysisRankingCandidate{fund: fund, analysis: analysis}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		log.Printf("⚠️ ranking build group wait failed: %v", err)
	}
	filteredCandidates := make([]analysisRankingCandidate, 0, len(candidates))
	for _, item := range candidates {
		if item.fund == nil || item.analysis == nil {
			continue
		}
		filteredCandidates = append(filteredCandidates, item)
		if h.snapshotStore != nil && item.analysis != nil {
			_ = h.snapshotStore.Save(c.Request.Context(), item.fund.ID, item.analysis, now)
		}
	}

	increaseItems, watchItems, riskItems := bucketAnalysisRankingCandidates(filteredCandidates)

	sort.SliceStable(increaseItems, func(i, j int) bool {
		return compareIncreaseRanking(increaseItems[i].analysis, increaseItems[j].analysis)
	})
	sort.SliceStable(watchItems, func(i, j int) bool {
		return compareWatchRanking(watchItems[i].analysis, watchItems[j].analysis)
	})
	sort.SliceStable(riskItems, func(i, j int) bool {
		return compareRiskRanking(riskItems[i].analysis, riskItems[j].analysis)
	})

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: AnalysisRankingsResponse{
			GeneratedAt:   now,
			IncreaseIdeas: buildRankingItems(increaseItems, analysisRankingLimit),
			WatchIdeas:    buildRankingItems(watchItems, analysisRankingLimit),
			RiskAlerts:    buildUniqueRiskItems(riskItems, analysisRankingLimit),
		},
	})
}

func (h *FundHandler) listAnalysisRankingCandidateFundIDs(ctx context.Context, limit int) ([]string, error) {
	if h != nil && h.rankingCandidates != nil {
		fundIDs, err := h.rankingCandidates.ListRankingCandidateFundIDs(ctx, limit)
		if err != nil {
			return nil, err
		}
		if len(fundIDs) > 0 {
			return fundIDs, nil
		}
	}
	return h.fundRepo.ListFundIDsWithHoldings(ctx)
}

func bucketAnalysisRankingCandidates(items []analysisRankingCandidate) (increaseItems, watchItems, riskItems []analysisRankingCandidate) {
	increaseItems = make([]analysisRankingCandidate, 0, len(items))
	watchItems = make([]analysisRankingCandidate, 0, len(items))
	riskItems = make([]analysisRankingCandidate, 0, len(items))
	for _, item := range items {
		if item.analysis == nil {
			continue
		}
		if analysisBelongsToRankingBucket(item.analysis, analysisRankingBucketIncrease) {
			increaseItems = append(increaseItems, item)
		}
		if analysisBelongsToRankingBucket(item.analysis, analysisRankingBucketWatch) {
			watchItems = append(watchItems, item)
		}
		if analysisBelongsToRankingBucket(item.analysis, analysisRankingBucketRisk) {
			riskItems = append(riskItems, item)
		}
	}
	return increaseItems, watchItems, riskItems
}

func analysisBelongsToRankingBucket(analysis *domain.FundAnalysis, bucket analysisRankingBucket) bool {
	if analysis == nil {
		return false
	}
	dominant := service.DominantRecommendationFromAnalysis(analysis)
	switch bucket {
	case analysisRankingBucketIncrease:
		return dominant == "increase"
	case analysisRankingBucketWatch:
		return dominant != "increase" && dominant != "decrease"
	case analysisRankingBucketRisk:
		return dominant == "decrease" || analysis.RiskLevel == "high"
	default:
		return true
	}
}

func compareIncreaseRanking(left *domain.FundAnalysis, right *domain.FundAnalysis) bool {
	if left == nil || right == nil {
		return right != nil
	}
	if !left.IncreasePercent.Equal(right.IncreasePercent) {
		return left.IncreasePercent.GreaterThan(right.IncreasePercent)
	}
	if !left.TotalScore.Equal(right.TotalScore) {
		return left.TotalScore.GreaterThan(right.TotalScore)
	}
	return left.Confidence > right.Confidence
}

func compareWatchRanking(left *domain.FundAnalysis, right *domain.FundAnalysis) bool {
	if left == nil || right == nil {
		return right != nil
	}
	if !left.HoldPercent.Equal(right.HoldPercent) {
		return left.HoldPercent.GreaterThan(right.HoldPercent)
	}
	if !left.TotalScore.Equal(right.TotalScore) {
		return left.TotalScore.GreaterThan(right.TotalScore)
	}
	return left.Confidence > right.Confidence
}

func compareRiskRanking(left *domain.FundAnalysis, right *domain.FundAnalysis) bool {
	if left == nil || right == nil {
		return right != nil
	}
	leftHigh := left.RiskLevel == "high"
	rightHigh := right.RiskLevel == "high"
	if leftHigh != rightHigh {
		return leftHigh
	}
	if !left.DecreasePercent.Equal(right.DecreasePercent) {
		return left.DecreasePercent.GreaterThan(right.DecreasePercent)
	}
	return left.TotalScore.LessThan(right.TotalScore)
}

func buildRankingItems(items []analysisRankingCandidate, limit int) []AnalysisRankingItem {
	if limit <= 0 || len(items) == 0 {
		return []AnalysisRankingItem{}
	}
	if len(items) > limit {
		items = items[:limit]
	}
	result := make([]AnalysisRankingItem, 0, len(items))
	for _, item := range items {
		result = append(result, AnalysisRankingItem{
			Fund:     item.fund,
			Analysis: item.analysis,
		})
	}
	return result
}

func buildUniqueRiskItems(items []analysisRankingCandidate, limit int) []AnalysisRankingItem {
	if limit <= 0 || len(items) == 0 {
		return []AnalysisRankingItem{}
	}
	result := make([]AnalysisRankingItem, 0, min(limit, len(items)))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.fund == nil {
			continue
		}
		if _, ok := seen[item.fund.ID]; ok {
			continue
		}
		seen[item.fund.ID] = struct{}{}
		result = append(result, AnalysisRankingItem{Fund: item.fund, Analysis: item.analysis})
		if len(result) >= limit {
			break
		}
	}
	return result
}

func buildRankingItemsFromSnapshots(records []database.FundAnalysisSnapshot, fundMap map[string]*domain.Fund, bucket analysisRankingBucket, limit int) []AnalysisRankingItem {
	if limit <= 0 || len(records) == 0 {
		return []AnalysisRankingItem{}
	}
	result := make([]AnalysisRankingItem, 0, min(limit, len(records)))
	for _, record := range records {
		var analysis domain.FundAnalysis
		if err := json.Unmarshal(record.AnalysisJSON, &analysis); err != nil {
			continue
		}
		if !analysisBelongsToRankingBucket(&analysis, bucket) {
			continue
		}
		analysis.AIExplanation = service.MarkAIExplanationSnapshotHit(analysis.AIExplanation)
		fund := &domain.Fund{ID: record.FundID}
		if existing := fundMap[record.FundID]; existing != nil {
			fund = existing
		}
		result = append(result, AnalysisRankingItem{
			Fund:     fund,
			Analysis: &analysis,
		})
		if len(result) >= limit {
			break
		}
	}
	return result
}
