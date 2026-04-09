package service

import (
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

func buildEstimateSnapshotFromQuotes(
	fund *domain.Fund,
	holdings []domain.StockHolding,
	quotes map[string]domain.StockQuote,
	source domain.QuoteSource,
	calculatedAt time.Time,
) *domain.FundEstimate {
	estimate := &domain.FundEstimate{
		FundID:       fund.ID,
		FundName:     fund.Name,
		PrevNav:      fund.NetAssetVal,
		CalculatedAt: calculatedAt,
		DataSource:   string(source),
	}

	weightedSum := decimal.Zero
	totalHoldingRatio := decimal.Zero
	hundred := decimal.NewFromInt(100)

	for _, holding := range holdings {
		quote, exists := quotes[holding.StockCode]
		if !exists {
			continue
		}

		holdingRatioDecimal := holding.HoldingRatio.Div(hundred)
		stockChangeDecimal := quote.ChangePercent.Div(hundred)
		contribution := stockChangeDecimal.Mul(holdingRatioDecimal).Mul(hundred)

		weightedSum = weightedSum.Add(contribution)
		totalHoldingRatio = totalHoldingRatio.Add(holding.HoldingRatio)

		stockName := quote.StockName
		if stockName == "" {
			stockName = holding.StockName
		}

		estimate.HoldingDetails = append(estimate.HoldingDetails, domain.HoldingDetail{
			StockCode:    holding.StockCode,
			StockName:    stockName,
			HoldingRatio: holding.HoldingRatio,
			StockChange:  quote.ChangePercent,
			Contribution: contribution.Round(4),
			CurrentPrice: quote.CurrentPrice,
			PrevClose:    quote.PrevClose,
		})
	}

	estimate.TotalHoldRatio = totalHoldingRatio
	if !totalHoldingRatio.IsZero() {
		estimate.ChangePercent = weightedSum.Div(totalHoldingRatio).Mul(hundred).Round(4)
	}

	if !fund.NetAssetVal.IsZero() {
		changeFactor := decimal.NewFromInt(1).Add(estimate.ChangePercent.Div(hundred))
		estimate.EstimateNav = fund.NetAssetVal.Mul(changeFactor).Round(4)
		estimate.ChangeAmount = estimate.EstimateNav.Sub(fund.NetAssetVal).Round(4)
	}

	return estimate
}
