package trading

import (
	"errors"
	"testing"
	"time"
)

func TestIsTradingDayUsesEmbeddedHolidayCalendar(t *testing.T) {
	holiday := time.Date(2025, time.April, 4, 12, 0, 0, 0, TradingLocation())
	if IsTradingDay(holiday) {
		t.Fatalf("expected %s to be a holiday", holiday.Format("2006-01-02"))
	}

	tradingDay := time.Date(2025, time.April, 7, 12, 0, 0, 0, TradingLocation())
	if !IsTradingDay(tradingDay) {
		t.Fatalf("expected %s to be a trading day", tradingDay.Format("2006-01-02"))
	}
}

func TestGetMarketStatusPreMarketUsesUnifiedTradingCalendar(t *testing.T) {
	now := time.Date(2025, time.April, 7, 8, 59, 0, 0, TradingLocation())
	status := GetMarketStatus(now)

	if status.Session != SessionPreMarket {
		t.Fatalf("expected pre-market session, got %s", status.Session)
	}
	if status.DisplayDate != "2025-04-03" {
		t.Fatalf("expected display date 2025-04-03, got %s", status.DisplayDate)
	}
	if status.PreviousTradingDay != "2025-04-03" {
		t.Fatalf("expected previous trading day 2025-04-03, got %s", status.PreviousTradingDay)
	}
	if status.NextTradingDay != "2025-04-08" {
		t.Fatalf("expected next trading day 2025-04-08, got %s", status.NextTradingDay)
	}
	if status.NextSessionStart == nil || status.NextSessionStart.Format(time.RFC3339) != "2025-04-07T09:30:00+08:00" {
		t.Fatalf("unexpected next session start: %#v", status.NextSessionStart)
	}
}

func TestGetMarketStatusHolidayUsesUnifiedTradingCalendar(t *testing.T) {
	now := time.Date(2025, time.October, 2, 10, 0, 0, 0, TradingLocation())
	status := GetMarketStatus(now)

	if status.Session != SessionHoliday {
		t.Fatalf("expected holiday session, got %s", status.Session)
	}
	if status.LastTradingDay != "2025-09-30" {
		t.Fatalf("expected last trading day 2025-09-30, got %s", status.LastTradingDay)
	}
	if status.NextTradingDay != "2025-10-09" {
		t.Fatalf("expected next trading day 2025-10-09, got %s", status.NextTradingDay)
	}
	if status.DisplayDate != "2025-09-30" {
		t.Fatalf("expected display date 2025-09-30, got %s", status.DisplayDate)
	}
}

func TestResolvePricingDate(t *testing.T) {
	sameDay := ResolvePricingDate(time.Date(2026, time.March, 31, 14, 59, 0, 0, TradingLocation()))
	if sameDay.PricingDate != "2026-03-31" || sameDay.Rule != PricingDateSameDayClose {
		t.Fatalf("expected same-day pricing, got %+v", sameDay)
	}

	afterCutoff := ResolvePricingDate(time.Date(2026, time.March, 31, 15, 0, 0, 0, TradingLocation()))
	if afterCutoff.PricingDate != "2026-04-01" || afterCutoff.Rule != PricingDateNextTradingDay {
		t.Fatalf("expected next trading day after cutoff, got %+v", afterCutoff)
	}

	holiday := ResolvePricingDate(time.Date(2025, time.April, 4, 14, 59, 0, 0, TradingLocation()))
	if holiday.PricingDate != "2025-04-07" || holiday.Rule != PricingDateNextTradingDay {
		t.Fatalf("expected next trading day for holiday trade, got %+v", holiday)
	}
}

func TestParseTradeAt(t *testing.T) {
	parsed, err := ParseTradeAt("2026-03-31T14:59:00+08:00")
	if err != nil {
		t.Fatalf("expected RFC3339 timestamp to parse, got %v", err)
	}
	if parsed.Format(time.RFC3339) != "2026-03-31T14:59:00+08:00" {
		t.Fatalf("unexpected parsed timestamp: %s", parsed.Format(time.RFC3339))
	}

	if _, err := ParseTradeAt(""); !errors.Is(err, ErrInvalidTradeTime) {
		t.Fatalf("expected invalid trade time for empty input, got %v", err)
	}
}
