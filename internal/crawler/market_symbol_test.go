package crawler

import "testing"

func TestBuildSinaKLineSymbolSupportsHongKongCodes(t *testing.T) {
	if got := buildSinaKLineSymbol("00700"); got != "hk00700" {
		t.Fatalf("buildSinaKLineSymbol() = %q, want hk00700", got)
	}
}

func TestBuildTencentMinuteSymbolSupportsHongKongCodes(t *testing.T) {
	if got := buildTencentMinuteSymbol("00700"); got != "hk00700" {
		t.Fatalf("buildTencentMinuteSymbol() = %q, want hk00700", got)
	}
}
