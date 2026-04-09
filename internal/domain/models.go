// Package domain contains the core business entities and interfaces.
package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Fund represents a mutual fund entity.
type Fund struct {
	ID          string          `json:"id"`         // Fund code, e.g., "000001"
	Name        string          `json:"name"`       // Fund name
	Type        string          `json:"type"`       // Fund type: "stock", "bond", "hybrid", etc.
	Manager     string          `json:"manager"`    // Fund manager name
	Company     string          `json:"company"`    // Fund company
	NetAssetVal decimal.Decimal `json:"nav"`        // Latest net asset value (NAV)
	TotalScale  decimal.Decimal `json:"scale"`      // Total fund scale (亿元)
	UpdatedAt   time.Time       `json:"updated_at"` // Last NAV update time
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
