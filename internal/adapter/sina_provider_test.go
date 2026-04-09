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

func TestBuildSinaSymbolSupportsHongKongCodes(t *testing.T) {
	if got := buildSinaSymbol("00700"); got != "hk00700" {
		t.Fatalf("buildSinaSymbol() = %q, want hk00700", got)
	}
}

func TestParseHKQuote(t *testing.T) {
	provider := &SinaFinanceProvider{}
	fields := []string{
		"TENCENT",
		"腾讯控股",
		"504.500",
		"489.200",
		"507.000",
		"501.000",
		"504.500",
		"15.300",
		"3.128",
		"504.50000",
		"505.00000",
		"8854877273",
		"17564238",
	}

	quote, err := provider.parseHKQuote("00700", fields)
	if err != nil {
		t.Fatalf("parseHKQuote() error = %v", err)
	}

	if quote.StockName != "腾讯控股" {
		t.Fatalf("stock name = %q, want 腾讯控股", quote.StockName)
	}
	if !quote.CurrentPrice.Equal(decimal.RequireFromString("504.500")) {
		t.Fatalf("current price = %s, want 504.500", quote.CurrentPrice.String())
	}
	if quote.ChangePercent.IsZero() {
		t.Fatalf("change percent should be populated for HK quote")
	}
}
