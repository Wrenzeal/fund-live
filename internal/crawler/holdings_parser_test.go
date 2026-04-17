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

func TestEastmoneyHoldingsParserSupportsOverseasTickerCodes(t *testing.T) {
	parser := NewEastmoneyHoldingsParser()

	if !parser.isOverseasTicker("NVDA") {
		t.Fatal("expected NVDA to be treated as a valid overseas ticker")
	}
	if exchange := parser.inferExchange("NVDA"); exchange != domain.ExchangeUS {
		t.Fatalf("inferExchange() = %s, want %s", exchange, domain.ExchangeUS)
	}
}

func TestEastmoneyHoldingsParserDropsZeroRatioHoldings(t *testing.T) {
	parser := NewEastmoneyHoldingsParser()

	holdings := parser.ToStockHoldings([]HoldingRawData{
		{
			StockCode:    "688012",
			StockName:    "中微公司",
			HoldingRatio: "0.00%",
			ReportPeriod: "2025-12-31",
		},
		{
			StockCode:    "688256",
			StockName:    "寒武纪",
			HoldingRatio: "9.27%",
			ReportPeriod: "2025-12-31",
		},
	})

	if len(holdings) != 1 {
		t.Fatalf("len(holdings) = %d, want 1", len(holdings))
	}
	if holdings[0].StockCode != "688256" {
		t.Fatalf("holding stock code = %s, want 688256", holdings[0].StockCode)
	}
}
