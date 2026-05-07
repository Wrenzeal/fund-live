// Package domain contains the core business entities and interfaces.
package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Fund represents a mutual fund entity.
type Fund struct {
	ID           string          `json:"id"`   // Fund code, e.g., "000001"
	Name         string          `json:"name"` // Fund name
	Type         string          `json:"type"` // Fund type: "stock", "bond", "hybrid", etc.
	CategoryCode string          `json:"category_code,omitempty"`
	CategoryName string          `json:"category_name,omitempty"`
	Manager      string          `json:"manager"`    // Fund manager name
	Company      string          `json:"company"`    // Fund company
	NetAssetVal  decimal.Decimal `json:"nav"`        // Latest net asset value (NAV)
	TotalScale   decimal.Decimal `json:"scale"`      // Total fund scale (亿元)
	UpdatedAt    time.Time       `json:"updated_at"` // Last NAV update time
}

// StockHolding represents a stock holding within a fund's portfolio.
type StockHolding struct {
	StockCode       string          `json:"stock_code"`       // Stock code, e.g., "600519" (SH), "000858" (SZ)
	StockName       string          `json:"stock_name"`       // Stock name
	Exchange        Exchange        `json:"exchange"`         // Exchange: SH or SZ
	HoldingRatio    decimal.Decimal `json:"holding_ratio"`    // Holding ratio as percentage, e.g., 8.56 means 8.56%
	HoldingShares   decimal.Decimal `json:"holding_shares"`   // Number of shares held
	MarketValue     decimal.Decimal `json:"market_value"`     // Market value in CNY
	ReportingPeriod string          `json:"reporting_period"` // e.g., "2024Q4"
}

// Exchange represents stock exchange.
type Exchange string

const (
	ExchangeSH Exchange = "SH" // Shanghai Stock Exchange
	ExchangeSZ Exchange = "SZ" // Shenzhen Stock Exchange
	ExchangeBJ Exchange = "BJ" // Beijing Stock Exchange
	ExchangeHK Exchange = "HK" // Hong Kong Stock Exchange
	ExchangeUS Exchange = "US" // United States exchanges / overseas tickers
)

// StockQuote represents real-time stock quote data.
type StockQuote struct {
	StockCode     string          `json:"stock_code"`
	StockName     string          `json:"stock_name"`
	CurrentPrice  decimal.Decimal `json:"current_price"`  // 现价
	PrevClose     decimal.Decimal `json:"prev_close"`     // 昨收
	OpenPrice     decimal.Decimal `json:"open_price"`     // 今开
	HighPrice     decimal.Decimal `json:"high_price"`     // 最高
	LowPrice      decimal.Decimal `json:"low_price"`      // 最低
	ChangePercent decimal.Decimal `json:"change_percent"` // 涨跌幅 (%)
	ChangeAmount  decimal.Decimal `json:"change_amount"`  // 涨跌额
	Volume        decimal.Decimal `json:"volume"`         // 成交量
	Turnover      decimal.Decimal `json:"turnover"`       // 成交额
	UpdatedAt     time.Time       `json:"updated_at"`
}

// FundEstimate represents the real-time fund valuation estimate.
type FundEstimate struct {
	FundID         string          `json:"fund_id"`
	FundName       string          `json:"fund_name"`
	EstimateNav    decimal.Decimal `json:"estimate_nav"`     // Estimated NAV
	PrevNav        decimal.Decimal `json:"prev_nav"`         // Previous NAV (昨日净值)
	ChangePercent  decimal.Decimal `json:"change_percent"`   // Estimated change percent
	ChangeAmount   decimal.Decimal `json:"change_amount"`    // Estimated change amount
	TotalHoldRatio decimal.Decimal `json:"total_hold_ratio"` // Sum of top holdings ratio
	HoldingDetails []HoldingDetail `json:"holding_details"`  // Individual stock contributions
	CalculatedAt   time.Time       `json:"calculated_at"`
	DataSource     string          `json:"data_source"`
}

// HoldingDetail represents the contribution of a single stock to the fund estimate.
type HoldingDetail struct {
	StockCode    string          `json:"stock_code"`
	StockName    string          `json:"stock_name"`
	HoldingRatio decimal.Decimal `json:"holding_ratio"` // Holding ratio (%)
	StockChange  decimal.Decimal `json:"stock_change"`  // Individual stock change (%)
	Contribution decimal.Decimal `json:"contribution"`  // Contribution to fund change
	CurrentPrice decimal.Decimal `json:"current_price"`
	PrevClose    decimal.Decimal `json:"prev_close"`
}

const (
	FundHoldingsDisplayLevelStock  = "stock_layer"
	FundHoldingsDisplayLevelTarget = "target_layer"

	FundHoldingsDisplayItemTypeStock      = "stock"
	FundHoldingsDisplayItemTypeTargetFund = "target_fund"

	FundHoldingsDisplayTargetTypeETFFund = "etf_fund"
	FundHoldingsDisplayTargetTypeFund    = "fund"
	FundHoldingsDisplayTargetTypeIndex   = "index"
)

// FundHoldingsDisplayItem represents a single user-facing holding display item.
// It may describe either a stock-layer holding or a next-layer target (ETF / fund / index).
type FundHoldingsDisplayItem struct {
	ItemType        string          `json:"item_type"`
	TargetType      string          `json:"target_type,omitempty"`
	Code            string          `json:"code"`
	Name            string          `json:"name"`
	Exchange        Exchange        `json:"exchange,omitempty"`
	HoldingRatio    decimal.Decimal `json:"holding_ratio,omitempty"`
	WeightPercent   decimal.Decimal `json:"weight_percent,omitempty"`
	ReportingPeriod string          `json:"reporting_period,omitempty"`
	IsPrimary       bool            `json:"is_primary,omitempty"`
	Source          string          `json:"source,omitempty"`
}

// FundHoldingsDisplay describes which layer the frontend should display by default.
type FundHoldingsDisplay struct {
	DisplayLevel         string                    `json:"display_level"`
	DisplayItems         []FundHoldingsDisplayItem `json:"display_items"`
	LookthroughAvailable bool                      `json:"lookthrough_available"`
}

func PrimaryTrackedETF(display *FundHoldingsDisplay) (FundHoldingsDisplayItem, bool) {
	if display == nil || display.DisplayLevel != FundHoldingsDisplayLevelTarget || len(display.DisplayItems) == 0 {
		return FundHoldingsDisplayItem{}, false
	}

	for _, item := range display.DisplayItems {
		if item.ItemType == FundHoldingsDisplayItemTypeTargetFund &&
			item.TargetType == FundHoldingsDisplayTargetTypeETFFund &&
			item.Code != "" {
			return item, true
		}
	}

	return FundHoldingsDisplayItem{}, false
}

// TimeSeriesPoint represents a single point in the intraday time series.
type TimeSeriesPoint struct {
	Timestamp     time.Time       `json:"timestamp"`
	ChangePercent decimal.Decimal `json:"change_percent"`
	EstimateNav   decimal.Decimal `json:"estimate_nav"`
}

// FundHistory stores official daily NAV snapshots and returns.
type FundHistory struct {
	FundID      string          `json:"fund_id"`
	Date        string          `json:"date"`
	NetAssetVal decimal.Decimal `json:"net_asset_val"`
	AccumVal    decimal.Decimal `json:"accum_val"`
	DailyReturn decimal.Decimal `json:"daily_return"`
	CreatedAt   time.Time       `json:"created_at"`
}

// FundHistoryLookupKey identifies a single official NAV snapshot.
type FundHistoryLookupKey struct {
	FundID string
	Date   string
}

// FundSector describes a stable sector dictionary entry.
type FundSector struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	ParentCode string `json:"parent_code,omitempty"`
	Level      int    `json:"level"`
	SortOrder  int    `json:"sort_order"`
}

// FundSectorBreakdown represents an aggregated sector weight for a fund snapshot.
type FundSectorBreakdown struct {
	SectorCode    string          `json:"sector_code"`
	SectorName    string          `json:"sector_name"`
	WeightPercent decimal.Decimal `json:"weight_percent"`
	Rank          int             `json:"rank"`
}

// FundSectorSnapshot stores the current sector classification snapshot for a fund.
type FundSectorSnapshot struct {
	FundID            string                `json:"fund_id"`
	AsOfDate          string                `json:"as_of_date"`
	PrimarySectorCode string                `json:"primary_sector_code"`
	PrimarySectorName string                `json:"primary_sector_name"`
	Source            string                `json:"source"`
	Confidence        string                `json:"confidence"`
	Breakdown         []FundSectorBreakdown `json:"breakdown"`
}

// FundTheme describes a stable theme dictionary entry.
type FundTheme struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	ParentCode string `json:"parent_code,omitempty"`
	Level      int    `json:"level"`
	SortOrder  int    `json:"sort_order"`
}

// FundThemeBreakdown represents an aggregated theme weight for a fund snapshot.
type FundThemeBreakdown struct {
	ThemeCode     string          `json:"theme_code"`
	ThemeName     string          `json:"theme_name"`
	WeightPercent decimal.Decimal `json:"weight_percent"`
	Rank          int             `json:"rank"`
}

// FundThemeSnapshot stores the current theme classification snapshot for a fund.
type FundThemeSnapshot struct {
	FundID           string               `json:"fund_id"`
	AsOfDate         string               `json:"as_of_date"`
	PrimaryThemeCode string               `json:"primary_theme_code"`
	PrimaryThemeName string               `json:"primary_theme_name"`
	Source           string               `json:"source"`
	Confidence       string               `json:"confidence"`
	Breakdown        []FundThemeBreakdown `json:"breakdown"`
}

// FundCategory describes the stable main classification for a fund.
type FundCategory struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	SortOrder   int    `json:"sort_order"`
}

// FundAnalysisModuleScore represents a single quant-analysis module score.
type FundAnalysisModuleScore struct {
	Code    string          `json:"code"`
	Name    string          `json:"name"`
	Score   decimal.Decimal `json:"score"`
	Summary string          `json:"summary,omitempty"`
}

// FundAnalysisEventImpact represents a structured disclosure / event signal.
type FundAnalysisEventImpact struct {
	Code           string           `json:"code"`
	Title          string           `json:"title"`
	Impact         string           `json:"impact"`
	Summary        string           `json:"summary"`
	TargetScope    string           `json:"target_scope,omitempty"`
	Strength       string           `json:"strength,omitempty"`
	Horizon        string           `json:"horizon,omitempty"`
	RelatedSymbols []string         `json:"related_symbols,omitempty"`
	WeightHint     *decimal.Decimal `json:"weight_hint,omitempty"`
}

// FundAnalysis summarizes the current rule-based quant analysis result for a fund.
type FundAnalysis struct {
	AnalysisVersion     string                    `json:"analysis_version"`
	AnalysisType        string                    `json:"analysis_type"`
	AnalysisBasis       string                    `json:"analysis_basis"`
	AsOfTime            time.Time                 `json:"as_of_time"`
	TotalScore          decimal.Decimal           `json:"total_score"`
	Confidence          string                    `json:"confidence"`
	RiskLevel           string                    `json:"risk_level"`
	IncreasePercent     decimal.Decimal           `json:"increase_percent"`
	HoldPercent         decimal.Decimal           `json:"hold_percent"`
	DecreasePercent     decimal.Decimal           `json:"decrease_percent"`
	LatestHoldingPeriod string                    `json:"latest_holding_period,omitempty"`
	Summary             string                    `json:"summary"`
	Reasons             []string                  `json:"reasons"`
	Warnings            []string                  `json:"warnings"`
	EventImpacts        []FundAnalysisEventImpact `json:"event_impacts"`
	ModuleScores        []FundAnalysisModuleScore `json:"module_scores"`
}
