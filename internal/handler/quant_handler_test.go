package handler

import (
	"testing"
	"time"
)

func TestParseQuantAsOfDateUsesShanghaiEndOfDay(t *testing.T) {
	parsed, err := parseQuantAsOf("2026-07-23")
	if err != nil {
		t.Fatalf("parseQuantAsOf() error = %v", err)
	}
	local := parsed.In(time.FixedZone("CST", 8*60*60))
	if local.Hour() != 23 || local.Minute() != 59 || local.Day() != 23 {
		t.Fatalf("parsed time = %s", local)
	}
}

func TestValidQuantEventStatus(t *testing.T) {
	for _, status := range []string{"expected", "disclosed", "active", "expired", "cancelled"} {
		if !validQuantEventStatus(status) {
			t.Fatalf("expected status %s to be valid", status)
		}
	}
	if validQuantEventStatus("latest") {
		t.Fatal("latest is a view, not an event status")
	}
}
