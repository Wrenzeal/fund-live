package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

func TestNormalizeFundTimeSeriesPointsDeduplicatesUsingLastValue(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	timestamp := time.Date(2026, 7, 24, 9, 30, 0, 1234, loc)
	points := normalizeFundTimeSeriesPoints("005827", []domain.TimeSeriesPoint{
		{Timestamp: timestamp, ChangePercent: decimal.NewFromFloat(1.1), EstimateNav: decimal.NewFromFloat(2.1)},
		{Timestamp: timestamp.Add(500 * time.Nanosecond), ChangePercent: decimal.NewFromFloat(1.2), EstimateNav: decimal.NewFromFloat(2.2)},
		{Timestamp: timestamp.Add(5 * time.Minute), ChangePercent: decimal.NewFromFloat(1.3), EstimateNav: decimal.NewFromFloat(2.3)},
	})
	if len(points) != 2 {
		t.Fatalf("len(points) = %d, want 2", len(points))
	}
	if points[0].ChangePercent.String() != "1.2" || points[0].EstimateNav.String() != "2.2" {
		t.Fatalf("duplicate timestamp did not keep last value: %#v", points[0])
	}
}

func TestPostgresFundRepositoryReplaceTimeSeriesByDateSerializesConcurrentWriters(t *testing.T) {
	db, cleanup := openPostgresUserRepoTestDB(t)
	defer cleanup()

	repo := NewPostgresFundRepository(db)
	ctx := context.Background()
	loc := time.FixedZone("CST", 8*60*60)
	date := time.Date(2026, 7, 24, 0, 0, 0, 0, loc)
	makePoints := func(offset float64) []domain.TimeSeriesPoint {
		return []domain.TimeSeriesPoint{
			{Timestamp: time.Date(2026, 7, 24, 9, 30, 0, 0, loc), ChangePercent: decimal.NewFromFloat(offset), EstimateNav: decimal.NewFromFloat(2 + offset)},
			{Timestamp: time.Date(2026, 7, 24, 9, 35, 0, 0, loc), ChangePercent: decimal.NewFromFloat(offset + 0.1), EstimateNav: decimal.NewFromFloat(2.1 + offset)},
		}
	}

	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for _, offset := range []float64{0.1, 0.2} {
		wg.Add(1)
		go func(value float64) {
			defer wg.Done()
			errors <- repo.ReplaceTimeSeriesByDate(ctx, "005827", date, makePoints(value))
		}(offset)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("ReplaceTimeSeriesByDate() error = %v", err)
		}
	}

	stored, err := repo.GetTimeSeriesByDate(ctx, "005827", date)
	if err != nil {
		t.Fatalf("GetTimeSeriesByDate() error = %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("len(stored) = %d, want 2", len(stored))
	}
}
