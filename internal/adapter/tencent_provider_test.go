package adapter

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestTencentParseQuoteUsesCurrentPrice(t *testing.T) {
	provider := &TencentQuoteProvider{}
	fields := []string{
		"1", "贵州茅台", "600519", "1463.03", "1459.88", "1459.54",
		"3296", "1997", "1292",
	}

	quote, err := provider.parseQuote("sh600519", fields)
	if err != nil {
		t.Fatalf("parseQuote() error = %v", err)
	}

	if !quote.CurrentPrice.Equal(decimal.RequireFromString("1463.03")) {
		t.Fatalf("current price = %s, want 1463.03", quote.CurrentPrice.String())
	}
}

func TestBuildTencentSymbolSupportsHongKongCodes(t *testing.T) {
	if got := buildTencentSymbol("00700"); got != "hk00700" {
		t.Fatalf("buildTencentSymbol() = %q, want hk00700", got)
	}
}

func TestBuildTencentSymbolSupportsUSCodes(t *testing.T) {
	if got := buildTencentSymbol("NVDA"); got != "usNVDA" {
		t.Fatalf("buildTencentSymbol() = %q, want usNVDA", got)
	}
}

func TestTencentParseUSQuote(t *testing.T) {
	provider := &TencentQuoteProvider{}
	fields := []string{
		"200", "英伟达", "NVDA.OQ", "198.35", "198.87", "197.43",
		"134012859", "0", "0", "197.89", "100", "0", "0", "0", "0", "0", "0", "0", "0",
		"197.90", "200", "0", "0", "0", "0", "0", "0", "0", "0", "",
		"2026-04-16 16:00:01", "-0.52", "-0.26", "199.85", "195.81", "USD", "134012859", "26548786422",
	}

	quote, err := provider.parseQuote("usNVDA", fields)
	if err != nil {
		t.Fatalf("parseQuote() error = %v", err)
	}
	if quote.StockCode != "NVDA" {
		t.Fatalf("stock code = %q, want NVDA", quote.StockCode)
	}
	if !quote.CurrentPrice.Equal(decimal.RequireFromString("198.35")) {
		t.Fatalf("current price = %s, want 198.35", quote.CurrentPrice.String())
	}
	if !quote.PrevClose.Equal(decimal.RequireFromString("198.87")) {
		t.Fatalf("prev close = %s, want 198.87", quote.PrevClose.String())
	}
	if !quote.ChangePercent.Equal(decimal.RequireFromString("-0.26")) {
		t.Fatalf("change percent = %s, want -0.26", quote.ChangePercent.String())
	}
}
