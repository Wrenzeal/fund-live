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

	quote, err := provider.parseQuote(fields)
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
