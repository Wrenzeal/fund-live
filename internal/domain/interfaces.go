// Package domain contains the core business interfaces.
package domain

import (
	"context"
	"time"
)

// QuoteProvider defines the interface for fetching real-time stock quotes.
// This follows the Adapter Pattern to allow multiple data source implementations.
type QuoteProvider interface {
	// GetRealTimeQuotes fetches detailed quote information for the given stock codes.
	// Returns a map of stockCode -> StockQuote.
	GetRealTimeQuotes(ctx context.Context, stockCodes []string) (map[string]StockQuote, error)

	// GetName returns the name of the quote provider.
	GetName() string
}

// FundRepository defines the interface for fund data persistence.
type FundRepository interface {
	// GetFundByID retrieves a fund by its ID.
	GetFundByID(ctx context.Context, fundID string) (*Fund, error)

	// GetFundsByIDs retrieves multiple funds keyed by fund ID.
	GetFundsByIDs(ctx context.Context, fundIDs []string) (map[string]*Fund, error)

	// SearchFunds searches for funds by name or code.
	SearchFunds(ctx context.Context, query string, limit int) ([]*Fund, error)

	// GetFundHoldings retrieves the top holdings for a fund.
	GetFundHoldings(ctx context.Context, fundID string) ([]StockHolding, error)

	// ListFundIDsWithHoldings returns fund IDs that currently have persisted holdings.
	ListFundIDsWithHoldings(ctx context.Context) ([]string, error)

	// SaveFund saves or updates a fund.
	SaveFund(ctx context.Context, fund *Fund) error

	// SaveHoldings saves the holdings for a fund.
	SaveHoldings(ctx context.Context, fundID string, holdings []StockHolding) error

	// SaveTimeSeriesPoint saves a time series data point.
	SaveTimeSeriesPoint(ctx context.Context, point *TimeSeriesPoint, fundID string) error

	// ReplaceTimeSeriesByDate replaces all time series data for a fund on a specific date.
	ReplaceTimeSeriesByDate(ctx context.Context, fundID string, date time.Time, points []TimeSeriesPoint) error

	// GetTimeSeriesByDate retrieves time series data for a fund on a specific date.
	GetTimeSeriesByDate(ctx context.Context, fundID string, date time.Time) ([]TimeSeriesPoint, error)

	// SaveFundHistory saves or updates an official daily NAV snapshot.
	SaveFundHistory(ctx context.Context, history *FundHistory) error

	// GetLatestFundHistory retrieves the latest official NAV snapshot for a fund.
	GetLatestFundHistory(ctx context.Context, fundID string) (*FundHistory, error)

	// GetLatestFundHistoriesByFundIDs retrieves the latest official NAV snapshots for multiple funds.
	GetLatestFundHistoriesByFundIDs(ctx context.Context, fundIDs []string) (map[string]*FundHistory, error)

	// GetFundHistoriesByLookupKeys retrieves specific official NAV snapshots keyed by fund/date pairs.
	GetFundHistoriesByLookupKeys(ctx context.Context, keys []FundHistoryLookupKey) (map[FundHistoryLookupKey]*FundHistory, error)
}

// CacheRepository defines the interface for caching.
type CacheRepository interface {
	// Get retrieves a value from cache.
	Get(ctx context.Context, key string) (interface{}, bool)

	// Set stores a value in cache with TTL in seconds.
	Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error
}

// ValuationService defines the core business logic interface.
type ValuationService interface {
	// CalculateEstimate computes the real-time fund valuation estimate.
	CalculateEstimate(ctx context.Context, fundID string) (*FundEstimate, error)

	// GetIntradayTimeSeries returns the intraday time series for a fund.
	GetIntradayTimeSeries(ctx context.Context, fundID string) ([]TimeSeriesPoint, error)
}
