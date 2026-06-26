package crawler

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestParsePingzhongJSBuildsHistoryFromNetWorthAndAccumulatedTrend(t *testing.T) {
	parser := NewEastmoneyAPIParser()
	content := `
var fS_name = "测试基金";
var fS_companySname = "测试公司";
var Data_netWorthTrend = [{"x":1774540800000,"y":1.0000,"equityReturn":0},{"x":1774627200000,"y":1.1000,"equityReturn":1.23},{"x":1774886400000,"y":1.0500,"equityReturn":-0.56}];
var Data_ACWorthTrend = [[1774540800000,1.0000],[1774627200000,1.1200],[1774886400000,1.0800]];
`

	detail, err := parser.ParsePingzhongJS(content, "000001")
	if err != nil {
		t.Fatalf("ParsePingzhongJS() error = %v", err)
	}
	if len(detail.History) != 3 {
		t.Fatalf("history len = %d, want 3", len(detail.History))
	}
	last := detail.History[2]
	if last.FundID != "000001" || last.NetAssetVal.String() != "1.05" {
		t.Fatalf("last history = %+v", last)
	}
	if !last.DailyReturn.Equal(decimal.RequireFromString("-0.56")) {
		t.Fatalf("daily return = %s, want -0.56", last.DailyReturn.String())
	}
	if !last.AccumVal.Equal(decimal.RequireFromString("1.08")) {
		t.Fatalf("accum val = %s, want 1.08", last.AccumVal.String())
	}
}

func TestParseNavTrendFallsBackToComputedDailyReturnWhenEquityReturnMissing(t *testing.T) {
	parser := NewEastmoneyAPIParser()
	_, _, _, histories := parser.parseNavTrend(`[{"x":1774540800000,"y":1.0000},{"x":1774627200000,"y":1.1000}]`, "000001")

	if len(histories) != 2 {
		t.Fatalf("history len = %d, want 2", len(histories))
	}
	if !histories[1].DailyReturn.Equal(decimal.RequireFromString("10")) {
		t.Fatalf("daily return = %s, want 10", histories[1].DailyReturn.String())
	}
}
