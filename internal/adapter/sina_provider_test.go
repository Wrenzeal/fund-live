package adapter

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestParseQuoteFallsBackWhenCurrentPriceIsZero(t *testing.T) {
	provider := &SinaFinanceProvider{}
	fields := []string{
		"贵州茅台",
		"0.000",
		"1459.880",
		"0.000",
		"0.000",
		"0.000",
		"1457.760",
		"1457.760",
		"0",
		"0.000",
	}

	quote, err := provider.parseQuote("600519", fields)
	if err != nil {
		t.Fatalf("parseQuote() error = %v", err)
	}

	if !quote.CurrentPrice.Equal(decimal.RequireFromString("1457.760")) {
		t.Fatalf("current price = %s, want 1457.760", quote.CurrentPrice.String())
	}
	if quote.ChangePercent.Equal(decimal.NewFromInt(-100)) {
		t.Fatalf("change percent should not be -100 after fallback")
	}
	if quote.HighPrice.IsZero() || quote.LowPrice.IsZero() {
		t.Fatalf("high/low should be populated after fallback: high=%s low=%s", quote.HighPrice.String(), quote.LowPrice.String())
	}
}
