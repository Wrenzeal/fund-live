package service

import (
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
)

const fundAnalysisSnapshotMaxAge = 26 * time.Hour

// IsFundAnalysisSnapshotFresh gates snapshot reuse for analysis, rankings and batch reads.
//
// The policy intentionally combines a coarse trading-day boundary with explicit
// version and explanation-cache metadata. That keeps old JSON snapshots from
// silently bypassing new scoring/explanation rules while still allowing nightly
// PostgreSQL snapshots to serve read-heavy pages during the current day.
func IsFundAnalysisSnapshotFresh(analysis *domain.FundAnalysis, generatedAt time.Time, now time.Time) bool {
	if analysis == nil || generatedAt.IsZero() {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}
	if generatedAt.After(now.Add(5 * time.Minute)) {
		return false
	}
	if now.Sub(generatedAt) > fundAnalysisSnapshotMaxAge {
		return false
	}
	if !sameAnalysisCacheDay(generatedAt, now) {
		return false
	}
	if analysis.AnalysisVersion != CurrentFundAnalysisVersion {
		return false
	}
	if analysis.AIExplanation == nil {
		return false
	}
	return CanReuseAIExplanation(analysis.AIExplanation, analysis.AIExplanation.CacheKey, now)
}

func sameAnalysisCacheDay(left time.Time, right time.Time) bool {
	loc := analysisCacheLocation()
	left = left.In(loc)
	right = right.In(loc)
	return left.Year() == right.Year() && left.YearDay() == right.YearDay()
}
