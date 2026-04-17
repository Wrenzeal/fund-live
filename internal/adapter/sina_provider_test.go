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

func TestBuildSinaSymbolSupportsUSCodes(t *testing.T) {
	if got := buildSinaSymbol("NVDA"); got != "gb_nvda" {
		t.Fatalf("buildSinaSymbol() = %q, want gb_nvda", got)
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

func TestParseUSQuote(t *testing.T) {
	provider := &SinaFinanceProvider{}
	fields := []string{
		"英伟达", "198.3500", "-0.26", "2026-04-17 20:12:36", "-0.5200",
		"197.4300", "199.8500", "195.8100", "212.1700", "95.0000",
		"134012859", "142216902", "4819905000000", "4.93", "40.230000",
		"0.00", "0.00", "0.01", "0.00", "24300000000", "69", "198.9400",
		"0.30", "0.59", "Apr 17 08:12AM EDT", "Apr 16 04:00PM EDT", "198.8700",
	}

	quote, err := provider.parseUSQuote("NVDA", fields)
	if err != nil {
		t.Fatalf("parseUSQuote() error = %v", err)
	}
	if quote.StockCode != "NVDA" {
		t.Fatalf("stock code = %q, want NVDA", quote.StockCode)
	}
	if !quote.CurrentPrice.Equal(decimal.RequireFromString("198.3500")) {
		t.Fatalf("current price = %s, want 198.3500", quote.CurrentPrice.String())
	}
	if !quote.PrevClose.Equal(decimal.RequireFromString("198.8700")) {
		t.Fatalf("prev close = %s, want 198.8700", quote.PrevClose.String())
	}
	if !quote.ChangePercent.Equal(decimal.RequireFromString("-0.26")) {
		t.Fatalf("change percent = %s, want -0.26", quote.ChangePercent.String())
	}
}
