package service

import (
	"testing"
	"time"
)

func mustShanghaiTime(t *testing.T, value string) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, loc)
	if err != nil {
		t.Fatalf("parse time: %v", err)
	}
	return parsed
}

func TestPreferredTimeSeriesDateLunchBreakUsesToday(t *testing.T) {
	svc := &ValuationServiceImpl{}
	now := mustShanghaiTime(t, "2026-03-25 12:02:00")

	got := svc.preferredTimeSeriesDate(now)
	if got.Format("2006-01-02") != "2026-03-25" {
		t.Fatalf("expected today during lunch break, got %s", got.Format("2006-01-02"))
	}
}

func TestPreferredTimeSeriesDateAfterHoursUsesPreviousTradingDay(t *testing.T) {
	svc := &ValuationServiceImpl{}
	now := mustShanghaiTime(t, "2026-03-25 15:30:00")

	got := svc.preferredTimeSeriesDate(now)
	if got.Format("2006-01-02") != "2026-03-24" {
		t.Fatalf("expected previous trading day after close, got %s", got.Format("2006-01-02"))
	}
}
