package crawler

import (
	"testing"

	"github.com/RomaticDOG/fund/internal/domain"
)

func TestEastmoneyHoldingsParserSupportsHongKongStockCodes(t *testing.T) {
	parser := NewEastmoneyHoldingsParser()

	if !parser.isStockCode("00700") {
		t.Fatal("expected 00700 to be treated as a valid stock code")
	}
	if exchange := parser.inferExchange("00700"); exchange != domain.ExchangeHK {
		t.Fatalf("inferExchange() = %s, want %s", exchange, domain.ExchangeHK)
	}
}
