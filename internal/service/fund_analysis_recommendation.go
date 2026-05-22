package service

import (
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

const (
	FundAnalysisIncreaseThreshold = 55.0
	FundAnalysisDecreaseThreshold = 60.0
)

func DominantRecommendationFromPercents(increasePercent, decreasePercent float64) string {
	if increasePercent >= FundAnalysisIncreaseThreshold {
		return "increase"
	}
	if decreasePercent >= FundAnalysisDecreaseThreshold {
		return "decrease"
	}
	return "hold"
}

func DominantRecommendationFromAnalysis(analysis *domain.FundAnalysis) string {
	if analysis == nil {
		return "hold"
	}
	return DominantRecommendationFromDecimals(analysis.IncreasePercent, analysis.DecreasePercent)
}

func DominantRecommendationFromDecimals(increasePercent, decreasePercent decimal.Decimal) string {
	if increasePercent.GreaterThanOrEqual(decimal.NewFromFloat(FundAnalysisIncreaseThreshold)) {
		return "increase"
	}
	if decreasePercent.GreaterThanOrEqual(decimal.NewFromFloat(FundAnalysisDecreaseThreshold)) {
		return "decrease"
	}
	return "hold"
}
