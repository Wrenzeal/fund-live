package service

import (
	"context"
	"testing"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

func TestFundSectorStoreBuildSnapshotAggregatesDirectHoldings(t *testing.T) {
	store := NewFundSectorStore(nil)

	snapshot, err := store.BuildSnapshot(context.Background(), "005827", []domain.StockHolding{
		{StockCode: "600519", StockName: "贵州茅台", Exchange: domain.ExchangeSH, HoldingRatio: decimal.RequireFromString("9.90"), ReportingPeriod: "2025-12-31"},
		{StockCode: "000858", StockName: "五粮液", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.RequireFromString("9.63"), ReportingPeriod: "2025-12-31"},
		{StockCode: "00700", StockName: "腾讯控股", Exchange: domain.ExchangeHK, HoldingRatio: decimal.RequireFromString("9.98"), ReportingPeriod: "2025-12-31"},
	}, SectorSourceDirectHoldings)
	if err != nil {
		t.Fatalf("BuildSnapshot() error = %v", err)
	}
	if snapshot == nil {
		t.Fatal("snapshot = nil")
	}
	if snapshot.PrimarySectorCode != "liquor" {
		t.Fatalf("primary sector = %s, want liquor", snapshot.PrimarySectorCode)
	}
	if len(snapshot.Breakdown) < 2 {
		t.Fatalf("breakdown len = %d, want >= 2", len(snapshot.Breakdown))
	}
	if snapshot.Breakdown[0].SectorCode != "liquor" {
		t.Fatalf("breakdown[0] = %s, want liquor", snapshot.Breakdown[0].SectorCode)
	}
}

func TestFundSectorStoreBuildSnapshotAggregatesQDIIHoldings(t *testing.T) {
	store := NewFundSectorStore(nil)

	snapshot, err := store.BuildSnapshot(context.Background(), "017437", []domain.StockHolding{
		{StockCode: "NVDA", StockName: "英伟达", Exchange: domain.ExchangeUS, HoldingRatio: decimal.RequireFromString("9.83"), ReportingPeriod: "2025-12-31"},
		{StockCode: "AVGO", StockName: "博通", Exchange: domain.ExchangeUS, HoldingRatio: decimal.RequireFromString("7.54"), ReportingPeriod: "2025-12-31"},
		{StockCode: "MRVL", StockName: "迈威尔科技", Exchange: domain.ExchangeUS, HoldingRatio: decimal.RequireFromString("4.79"), ReportingPeriod: "2025-12-31"},
		{StockCode: "AAPL", StockName: "苹果", Exchange: domain.ExchangeUS, HoldingRatio: decimal.RequireFromString("9.14"), ReportingPeriod: "2025-12-31"},
	}, SectorSourceQDIIHoldings)
	if err != nil {
		t.Fatalf("BuildSnapshot() error = %v", err)
	}
	if snapshot == nil {
		t.Fatal("snapshot = nil")
	}
	if snapshot.PrimarySectorCode != "semiconductor" {
		t.Fatalf("primary sector = %s, want semiconductor", snapshot.PrimarySectorCode)
	}
}

func TestFundSectorStoreBuildSnapshotAggregatesFallbackHoldings(t *testing.T) {
	store := NewFundSectorStore(nil)

	snapshot, err := store.BuildSnapshot(context.Background(), "020465", []domain.StockHolding{
		{StockCode: "688012", StockName: "中微公司", Exchange: domain.ExchangeSH, HoldingRatio: decimal.RequireFromString("14.51"), ReportingPeriod: "2025-12-31"},
		{StockCode: "002371", StockName: "北方华创", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.RequireFromString("13.98"), ReportingPeriod: "2025-12-31"},
	}, SectorSourceTargetETFFallback)
	if err != nil {
		t.Fatalf("BuildSnapshot() error = %v", err)
	}
	if snapshot == nil {
		t.Fatal("snapshot = nil")
	}
	if snapshot.Source != SectorSourceTargetETFFallback {
		t.Fatalf("source = %s, want %s", snapshot.Source, SectorSourceTargetETFFallback)
	}
	if snapshot.PrimarySectorCode != "semiconductor" {
		t.Fatalf("primary sector = %s, want semiconductor", snapshot.PrimarySectorCode)
	}
}

func TestDeriveFundCategoryCode(t *testing.T) {
	tests := []struct {
		name string
		fund *domain.Fund
		want string
	}{
		{"feeder", &domain.Fund{Name: "招商中证半导体产业ETF发起式联接C", Type: "index"}, "feeder"},
		{"qdii", &domain.Fund{Name: "华宝纳斯达克精选股票发起式(QDII)C", Type: "qdii"}, "qdii"},
		{"bond", &domain.Fund{Name: "中长期纯债", Type: "债券型-长债"}, "bond"},
		{"money", &domain.Fund{Name: "货币增强", Type: "货币型"}, "money"},
		{"index", &domain.Fund{Name: "招商中证半导体产业ETF", Type: "指数型-股票"}, "index"},
		{"stock", &domain.Fund{Name: "股票精选", Type: "股票型"}, "stock"},
		{"hybrid", &domain.Fund{Name: "蓝筹混合", Type: "混合型-偏股"}, "hybrid"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := deriveFundCategoryCode(tc.fund); got != tc.want {
				t.Fatalf("deriveFundCategoryCode() = %s, want %s", got, tc.want)
			}
		})
	}
}
