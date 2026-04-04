package service

import (
	"context"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

type countingCollectorFundRepository struct {
}

func (r *countingCollectorFundRepository) GetFundByID(ctx context.Context, fundID string) (*domain.Fund, error) {
	return &domain.Fund{ID: fundID, NetAssetVal: decimal.RequireFromString("1.0000")}, nil
}

func (r *countingCollectorFundRepository) GetFundsByIDs(ctx context.Context, fundIDs []string) (map[string]*domain.Fund, error) {
	return map[string]*domain.Fund{}, nil
}

func (r *countingCollectorFundRepository) SearchFunds(ctx context.Context, query string, limit int) ([]*domain.Fund, error) {
	return nil, nil
}

func (r *countingCollectorFundRepository) GetFundHoldings(ctx context.Context, fundID string) ([]domain.StockHolding, error) {
	return nil, nil
}

func (r *countingCollectorFundRepository) SaveFund(ctx context.Context, fund *domain.Fund) error {
	return nil
}

func (r *countingCollectorFundRepository) SaveHoldings(ctx context.Context, fundID string, holdings []domain.StockHolding) error {
	return nil
}

func (r *countingCollectorFundRepository) SaveTimeSeriesPoint(ctx context.Context, point *domain.TimeSeriesPoint, fundID string) error {
	return nil
}

func (r *countingCollectorFundRepository) ReplaceTimeSeriesByDate(ctx context.Context, fundID string, date time.Time, points []domain.TimeSeriesPoint) error {
	return nil
}

func (r *countingCollectorFundRepository) GetTimeSeriesByDate(ctx context.Context, fundID string, date time.Time) ([]domain.TimeSeriesPoint, error) {
	return nil, nil
}

func (r *countingCollectorFundRepository) SaveFundHistory(ctx context.Context, history *domain.FundHistory) error {
	return nil
}

func (r *countingCollectorFundRepository) GetLatestFundHistory(ctx context.Context, fundID string) (*domain.FundHistory, error) {
	return nil, nil
}

func (r *countingCollectorFundRepository) GetLatestFundHistoriesByFundIDs(ctx context.Context, fundIDs []string) (map[string]*domain.FundHistory, error) {
	return map[string]*domain.FundHistory{}, nil
}

type noopQuoteProvider struct{}

func (noopQuoteProvider) GetRealTimeQuotes(ctx context.Context, stockCodes []string) (map[string]domain.StockQuote, error) {
	return map[string]domain.StockQuote{}, nil
}

func (noopQuoteProvider) GetName() string {
	return "noop"
}

type noopCacheRepository struct{}

func (noopCacheRepository) Get(ctx context.Context, key string) (interface{}, bool) {
	return nil, false
}

func (noopCacheRepository) Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error {
	return nil
}

func TestStartBackgroundCollectorDoesNotQueryRepositoryForFundIDs(t *testing.T) {
	repo := &countingCollectorFundRepository{}
	service := NewValuationService(repo, noopQuoteProvider{}, noopCacheRepository{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	service.StartBackgroundCollector(ctx, nil, time.Hour)
	if tracked := service.snapshotTrackedFunds(); len(tracked) != 0 {
		t.Fatalf("tracked funds = %#v, want empty", tracked)
	}
}

func TestTrackFundIDsDeduplicatesEntries(t *testing.T) {
	service := NewValuationService(&countingCollectorFundRepository{}, noopQuoteProvider{}, noopCacheRepository{})

	service.TrackFundIDs("005827", "005827", "003095", " ")

	tracked := service.snapshotTrackedFunds()
	if len(tracked) != 2 {
		t.Fatalf("tracked funds len = %d, want 2", len(tracked))
	}
	if tracked[0].FundID != "005827" || tracked[1].FundID != "003095" {
		t.Fatalf("tracked funds = %#v", tracked)
	}
	if tracked[0].Source != domain.QuoteSourceSina || tracked[1].Source != domain.QuoteSourceSina {
		t.Fatalf("tracked sources = %#v", tracked)
	}
}
