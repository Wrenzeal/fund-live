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

func TestFundSectorStoreBuildThemeSnapshotAggregatesCPOTopic(t *testing.T) {
	store := NewFundSectorStore(nil)

	snapshot, err := store.BuildThemeSnapshot(context.Background(), "theme-cpo", []domain.StockHolding{
		{StockCode: "300308", StockName: "中际旭创", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.RequireFromString("9.80"), ReportingPeriod: "2025-12-31"},
		{StockCode: "300502", StockName: "新易盛", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.RequireFromString("8.20"), ReportingPeriod: "2025-12-31"},
		{StockCode: "300394", StockName: "天孚通信", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.RequireFromString("7.10"), ReportingPeriod: "2025-12-31"},
	}, ThemeSourceDirectHoldings)
	if err != nil {
		t.Fatalf("BuildThemeSnapshot() error = %v", err)
	}
	if snapshot == nil {
		t.Fatal("snapshot = nil")
	}
	if snapshot.PrimaryThemeCode != "cpo_optical_module" {
		t.Fatalf("primary theme = %s, want cpo_optical_module", snapshot.PrimaryThemeCode)
	}
}

func TestFundSectorStoreBuildThemeSnapshotAggregatesCommercialAerospaceTopic(t *testing.T) {
	store := NewFundSectorStore(nil)

	snapshot, err := store.BuildThemeSnapshot(context.Background(), "theme-space", []domain.StockHolding{
		{StockCode: "601698", StockName: "中国卫通", Exchange: domain.ExchangeSH, HoldingRatio: decimal.RequireFromString("9.50"), ReportingPeriod: "2025-12-31"},
		{StockCode: "300762", StockName: "上海瀚讯", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.RequireFromString("7.60"), ReportingPeriod: "2025-12-31"},
		{StockCode: "600118", StockName: "中国卫星", Exchange: domain.ExchangeSH, HoldingRatio: decimal.RequireFromString("6.20"), ReportingPeriod: "2025-12-31"},
	}, ThemeSourceDirectHoldings)
	if err != nil {
		t.Fatalf("BuildThemeSnapshot() error = %v", err)
	}
	if snapshot == nil {
		t.Fatal("snapshot = nil")
	}
	if snapshot.PrimaryThemeCode != "commercial_aerospace" {
		t.Fatalf("primary theme = %s, want commercial_aerospace", snapshot.PrimaryThemeCode)
	}
}

func TestFundSectorStoreBuildSnapshotPrefersRecognizedSectorOverOther(t *testing.T) {
	store := NewFundSectorStore(nil)

	snapshot, err := store.BuildSnapshot(context.Background(), "sector-pref", []domain.StockHolding{
		{StockCode: "999999", StockName: "示例未映射股票", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.RequireFromString("42.25"), ReportingPeriod: "2026-03-31"},
		{StockCode: "600519", StockName: "贵州茅台", Exchange: domain.ExchangeSH, HoldingRatio: decimal.RequireFromString("13.29"), ReportingPeriod: "2026-03-31"},
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
	if snapshot.Confidence != SectorConfidenceLow {
		t.Fatalf("confidence = %s, want low", snapshot.Confidence)
	}
}

func TestFundSectorStoreBuildThemeSnapshotFallsBackFromSector(t *testing.T) {
	store := NewFundSectorStore(nil)

	snapshot, err := store.BuildThemeSnapshot(context.Background(), "theme-sector-fallback", []domain.StockHolding{
		{StockCode: "688981", StockName: "中芯国际", Exchange: domain.ExchangeSH, HoldingRatio: decimal.RequireFromString("10.28"), ReportingPeriod: "2025-12-31"},
		{StockCode: "688041", StockName: "海光信息", Exchange: domain.ExchangeSH, HoldingRatio: decimal.RequireFromString("9.97"), ReportingPeriod: "2025-12-31"},
	}, ThemeSourceTargetETFFallback)
	if err != nil {
		t.Fatalf("BuildThemeSnapshot() error = %v", err)
	}
	if snapshot == nil {
		t.Fatal("snapshot = nil")
	}
	if snapshot.PrimaryThemeCode != "semiconductor_chip" {
		t.Fatalf("primary theme = %s, want semiconductor_chip", snapshot.PrimaryThemeCode)
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
