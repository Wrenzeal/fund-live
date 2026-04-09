package service

import (
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/repository"
)

func TestFundHoldingsRefreshServiceNextRunAt(t *testing.T) {
	svc := NewFundHoldingsRefreshService(repository.NewMemoryFundRepository())
	loc := tradingLocation()

	beforeRun := time.Date(2026, time.April, 1, 0, 30, 0, 0, loc)
	next := svc.nextRunAt(beforeRun)
	want := time.Date(2026, time.April, 1, 1, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("nextRunAt(before) = %s, want %s", next, want)
	}

	afterRun := time.Date(2026, time.April, 1, 1, 30, 0, 0, loc)
	next = svc.nextRunAt(afterRun)
	want = time.Date(2026, time.May, 1, 1, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("nextRunAt(after) = %s, want %s", next, want)
	}
}
