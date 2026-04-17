package service

import (
	"context"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/shopspring/decimal"
)

type qdiiQuoteProvider struct {
	quotes map[string]domain.StockQuote
}

func (p qdiiQuoteProvider) GetRealTimeQuotes(ctx context.Context, stockCodes []string) (map[string]domain.StockQuote, error) {
	result := make(map[string]domain.StockQuote, len(stockCodes))
	for _, code := range stockCodes {
		if quote, ok := p.quotes[code]; ok {
			result[code] = quote
		}
	}
	return result, nil
}

func (p qdiiQuoteProvider) GetName() string {
	return "qdii-test"
}

func TestCalculateEstimateUsesLiveUSQuotesForQDIIFund(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	if err := fundRepo.SaveFund(context.Background(), &domain.Fund{
		ID:          "017437",
		Name:        "华宝纳斯达克精选股票发起式(QDII)C",
		Type:        "qdii",
		NetAssetVal: decimal.RequireFromString("2.2451"),
		UpdatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFund() error = %v", err)
	}
	if err := fundRepo.SaveHoldings(context.Background(), "017437", []domain.StockHolding{
		{
			StockCode:    "NVDA",
			StockName:    "英伟达",
			Exchange:     domain.ExchangeUS,
			HoldingRatio: decimal.RequireFromString("9.83"),
		},
		{
			StockCode:    "AAPL",
			StockName:    "苹果",
			Exchange:     domain.ExchangeUS,
			HoldingRatio: decimal.RequireFromString("9.14"),
		},
	}); err != nil {
		t.Fatalf("SaveHoldings() error = %v", err)
	}

	quoteProvider := qdiiQuoteProvider{
		quotes: map[string]domain.StockQuote{
			"NVDA": {
				StockCode:     "NVDA",
				StockName:     "英伟达",
				CurrentPrice:  decimal.RequireFromString("198.35"),
				PrevClose:     decimal.RequireFromString("198.87"),
				ChangeAmount:  decimal.RequireFromString("-0.52"),
				ChangePercent: decimal.RequireFromString("-0.26"),
			},
			"AAPL": {
				StockCode:     "AAPL",
				StockName:     "苹果",
				CurrentPrice:  decimal.RequireFromString("263.40"),
				PrevClose:     decimal.RequireFromString("266.43"),
				ChangeAmount:  decimal.RequireFromString("-3.03"),
				ChangePercent: decimal.RequireFromString("-1.14"),
			},
		},
	}

	service := NewValuationService(fundRepo, quoteProvider, noopCacheRepository{})
	estimate, err := service.CalculateEstimate(context.Background(), "017437")
	if err != nil {
		t.Fatalf("CalculateEstimate() error = %v", err)
	}

	if estimate.DataSource != "sina" {
		t.Fatalf("data source = %q, want sina", estimate.DataSource)
	}
	if estimate.TotalHoldRatio.String() != "18.97" {
		t.Fatalf("total hold ratio = %s, want 18.97", estimate.TotalHoldRatio.String())
	}
	if len(estimate.HoldingDetails) != 2 {
		t.Fatalf("holding details len = %d, want 2", len(estimate.HoldingDetails))
	}
	if estimate.HoldingDetails[0].StockCode != "NVDA" {
		t.Fatalf("first holding stock code = %s, want NVDA", estimate.HoldingDetails[0].StockCode)
	}
	if estimate.ChangePercent.IsZero() {
		t.Fatalf("change percent should not be zero when live US quotes are available")
	}
}
