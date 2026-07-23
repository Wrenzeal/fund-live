package service

import "testing"

func TestParseEastmoneyQuantKLine(t *testing.T) {
	bar, err := parseEastmoneyQuantKLine("2026-07-22,4.123,4.200,4.260,4.100,1200000,50123456.78,3.8,1.2,0.05,2.1")
	if err != nil {
		t.Fatalf("parseEastmoneyQuantKLine() error = %v", err)
	}
	if bar.Open.String() != "4.123" || bar.Close.String() != "4.2" || bar.Amount.String() != "50123456.78" {
		t.Fatalf("unexpected bar: %#v", bar)
	}
}

func TestEastmoneySecurityID(t *testing.T) {
	tests := map[string]string{"510300": "1.510300", "159915": "0.159915", "000300": "1.000300"}
	for symbol, expected := range tests {
		if actual := eastmoneySecurityID(symbol); actual != expected {
			t.Fatalf("eastmoneySecurityID(%s) = %s, want %s", symbol, actual, expected)
		}
	}
}
