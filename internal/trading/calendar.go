// Package trading provides A-Share trading calendar utilities.
package trading

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	tradingTimeZone     = "Asia/Shanghai"
	marketOpenHour      = 9
	marketOpenMinute    = 30
	lunchBreakHour      = 11
	lunchBreakMinute    = 30
	afternoonOpenHour   = 13
	afternoonOpenMinute = 0
	marketCloseHour     = 15
	marketCloseMinute   = 0
	calendarSource      = "SSE holiday notices"
)

var (
	ErrInvalidTradeTime = errors.New("invalid trade time")
	ErrInvalidTradeDate = errors.New("invalid trade date")
)

// Beijing timezone.
var beijingLoc = loadTradingLocation()

var defaultCalendar = newCalendar(beijingLoc, holidayDatesByYear)

// SessionType indicates the current market session.
type SessionType string

const (
	SessionPreMarket  SessionType = "pre_market"
	SessionMorning    SessionType = "morning"
	SessionLunchBreak SessionType = "lunch_break"
	SessionAfternoon  SessionType = "afternoon"
	SessionAfterHours SessionType = "after_hours"
	SessionWeekend    SessionType = "weekend"
	SessionHoliday    SessionType = "holiday"
)

// PricingDateRule describes how a pricing date was resolved.
type PricingDateRule string

const (
	PricingDateSameDayClose   PricingDateRule = "same_day_close"
	PricingDateNextTradingDay PricingDateRule = "next_trading_day"
)

// MarketStatus represents the current market status.
type MarketStatus struct {
	IsTrading                      bool        `json:"is_trading"`
	IsTradingDay                   bool        `json:"is_trading_day"`
	Session                        SessionType `json:"session"`
	CurrentTime                    time.Time   `json:"current_time"`
	CurrentDate                    string      `json:"current_date"`
	DisplayDate                    string      `json:"display_date"`
	PreviousTradingDay             string      `json:"previous_trading_day"`
	LastTradingDay                 string      `json:"last_trading_day"`
	NextTradingDay                 string      `json:"next_trading_day"`
	NextSessionStart               *time.Time  `json:"next_session_start,omitempty"`
	NextTransitionAt               *time.Time  `json:"next_transition_at,omitempty"`
	TimeUntilNextSessionSeconds    int64       `json:"time_until_next_session_seconds"`
	TimeUntilNextTransitionSeconds int64       `json:"time_until_next_transition_seconds"`
	Timezone                       string      `json:"timezone"`
	CalendarSource                 string      `json:"calendar_source"`
	CoveredYears                   []int       `json:"covered_years"`
}

// PricingDateResolution describes how a holding pricing date is derived.
type PricingDateResolution struct {
	TradeAt        time.Time       `json:"trade_at"`
	TradeDate      string          `json:"trade_date"`
	PricingDate    string          `json:"pricing_date"`
	Rule           PricingDateRule `json:"rule"`
	IsTradingDay   bool            `json:"is_trading_day"`
	AfterCutoff    bool            `json:"after_cutoff"`
	CutoffTime     time.Time       `json:"cutoff_time"`
	NextTradingDay string          `json:"next_trading_day,omitempty"`
	Message        string          `json:"message"`
	Timezone       string          `json:"timezone"`
}

type Calendar struct {
	location     *time.Location
	holidaySet   map[string]struct{}
	coveredYears []int
}

func loadTradingLocation() *time.Location {
	loc, err := time.LoadLocation(tradingTimeZone)
	if err == nil {
		return loc
	}
	return time.FixedZone("CST", 8*60*60)
}

func newCalendar(location *time.Location, holidayData map[int][]string) *Calendar {
	holidaySet := make(map[string]struct{})
	coveredYears := make([]int, 0, len(holidayData))

	for year, dates := range holidayData {
		coveredYears = append(coveredYears, year)
		for _, date := range dates {
			holidaySet[date] = struct{}{}
		}
	}

	sort.Ints(coveredYears)

	return &Calendar{
		location:     location,
		holidaySet:   holidaySet,
		coveredYears: coveredYears,
	}
}

func normalizeToTradingLocation(t time.Time) time.Time {
	return t.In(defaultCalendar.location)
}

func startOfTradingDay(t time.Time) time.Time {
	local := normalizeToTradingLocation(t)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, defaultCalendar.location)
}

func formatTradingDate(t time.Time) string {
	return normalizeToTradingLocation(t).Format("2006-01-02")
}

func isTradeMinutesInRange(totalMinutes, startHour, startMinute, endHour, endMinute int) bool {
	start := startHour*60 + startMinute
	end := endHour*60 + endMinute
	return totalMinutes >= start && totalMinutes < end
}

func durationSecondsFrom(now time.Time, future *time.Time) int64 {
	if future == nil {
		return 0
	}
	duration := future.Sub(now)
	if duration <= 0 {
		return 0
	}
	return int64(duration / time.Second)
}

func (c *Calendar) isHoliday(t time.Time) bool {
	_, ok := c.holidaySet[formatTradingDate(t)]
	return ok
}

func (c *Calendar) isWeekend(t time.Time) bool {
	day := normalizeToTradingLocation(t).Weekday()
	return day == time.Saturday || day == time.Sunday
}

func (c *Calendar) isCoveredYear(t time.Time) bool {
	year := normalizeToTradingLocation(t).Year()
	for _, covered := range c.coveredYears {
		if covered == year {
			return true
		}
	}
	return false
}

// IsWeekend checks if the given date is a weekend.
func IsWeekend(t time.Time) bool {
	return defaultCalendar.isWeekend(t)
}

// IsTradingDay checks if the given date is an A-share trading day.
// Years outside the embedded holiday dataset gracefully fall back to a weekend-only rule.
func IsTradingDay(t time.Time) bool {
	if defaultCalendar.isWeekend(t) {
		return false
	}
	if !defaultCalendar.isCoveredYear(t) {
		return true
	}
	return !defaultCalendar.isHoliday(t)
}

// GetCurrentSession returns the current trading session type.
func GetCurrentSession(t time.Time) SessionType {
	local := normalizeToTradingLocation(t)
	if defaultCalendar.isWeekend(local) {
		return SessionWeekend
	}
	if defaultCalendar.isCoveredYear(local) && defaultCalendar.isHoliday(local) {
		return SessionHoliday
	}

	totalMinutes := local.Hour()*60 + local.Minute()
	switch {
	case totalMinutes < marketOpenHour*60+marketOpenMinute:
		return SessionPreMarket
	case isTradeMinutesInRange(totalMinutes, marketOpenHour, marketOpenMinute, lunchBreakHour, lunchBreakMinute):
		return SessionMorning
	case totalMinutes < afternoonOpenHour*60+afternoonOpenMinute:
		return SessionLunchBreak
	case isTradeMinutesInRange(totalMinutes, afternoonOpenHour, afternoonOpenMinute, marketCloseHour, marketCloseMinute):
		return SessionAfternoon
	default:
		return SessionAfterHours
	}
}

// IsTradingHours checks if the current time is within A-share trading hours.
func IsTradingHours(t time.Time) bool {
	session := GetCurrentSession(t)
	return session == SessionMorning || session == SessionAfternoon
}

// GetPreviousTradingDay returns the closest trading day strictly before the given date.
func GetPreviousTradingDay(t time.Time) time.Time {
	candidate := startOfTradingDay(t).AddDate(0, 0, -1)
	for i := 0; i < 370; i++ {
		if IsTradingDay(candidate) {
			return candidate
		}
		candidate = candidate.AddDate(0, 0, -1)
	}
	return startOfTradingDay(t).AddDate(0, 0, -1)
}

// GetLastTradingDay returns the most recent trading day with data available.
// If today is a trading day and the market has opened, it returns today.
func GetLastTradingDay(t time.Time) time.Time {
	local := normalizeToTradingLocation(t)
	if IsTradingDay(local) {
		if local.Hour() > marketOpenHour || (local.Hour() == marketOpenHour && local.Minute() >= marketOpenMinute) {
			return startOfTradingDay(local)
		}
	}
	return GetPreviousTradingDay(local)
}

// GetNextTradingDay returns the closest trading day strictly after the given date.
func GetNextTradingDay(t time.Time) time.Time {
	candidate := startOfTradingDay(t).AddDate(0, 0, 1)
	for i := 0; i < 370; i++ {
		if IsTradingDay(candidate) {
			return candidate
		}
		candidate = candidate.AddDate(0, 0, 1)
	}
	return startOfTradingDay(t).AddDate(0, 0, 1)
}

func nextSessionStartAt(t time.Time) *time.Time {
	local := normalizeToTradingLocation(t)
	dayStart := startOfTradingDay(local)

	switch GetCurrentSession(local) {
	case SessionPreMarket:
		next := dayStart.Add(marketOpenHour*time.Hour + marketOpenMinute*time.Minute)
		return &next
	case SessionLunchBreak:
		next := dayStart.Add(afternoonOpenHour*time.Hour + afternoonOpenMinute*time.Minute)
		return &next
	default:
		nextTradingDay := GetNextTradingDay(local)
		next := time.Date(
			nextTradingDay.Year(),
			nextTradingDay.Month(),
			nextTradingDay.Day(),
			marketOpenHour,
			marketOpenMinute,
			0,
			0,
			defaultCalendar.location,
		)
		return &next
	}
}

func nextTransitionAt(t time.Time) *time.Time {
	local := normalizeToTradingLocation(t)
	dayStart := startOfTradingDay(local)

	switch GetCurrentSession(local) {
	case SessionPreMarket:
		next := dayStart.Add(marketOpenHour*time.Hour + marketOpenMinute*time.Minute)
		return &next
	case SessionMorning:
		next := dayStart.Add(lunchBreakHour*time.Hour + lunchBreakMinute*time.Minute)
		return &next
	case SessionLunchBreak:
		next := dayStart.Add(afternoonOpenHour*time.Hour + afternoonOpenMinute*time.Minute)
		return &next
	case SessionAfternoon:
		next := dayStart.Add(marketCloseHour*time.Hour + marketCloseMinute*time.Minute)
		return &next
	default:
		return nextSessionStartAt(local)
	}
}

// GetMarketStatus returns comprehensive market status information.
func GetMarketStatus(t time.Time) MarketStatus {
	local := normalizeToTradingLocation(t)
	session := GetCurrentSession(local)
	isTrading := session == SessionMorning || session == SessionAfternoon
	nextSessionStart := nextSessionStartAt(local)
	nextTransition := nextTransitionAt(local)

	status := MarketStatus{
		IsTrading:                      isTrading,
		IsTradingDay:                   IsTradingDay(local),
		Session:                        session,
		CurrentTime:                    local,
		CurrentDate:                    formatTradingDate(local),
		DisplayDate:                    GetLastTradingDay(local).Format("2006-01-02"),
		PreviousTradingDay:             GetPreviousTradingDay(local).Format("2006-01-02"),
		LastTradingDay:                 GetLastTradingDay(local).Format("2006-01-02"),
		NextTradingDay:                 GetNextTradingDay(local).Format("2006-01-02"),
		NextSessionStart:               nextSessionStart,
		NextTransitionAt:               nextTransition,
		TimeUntilNextSessionSeconds:    durationSecondsFrom(local, nextSessionStart),
		TimeUntilNextTransitionSeconds: durationSecondsFrom(local, nextTransition),
		Timezone:                       tradingTimeZone,
		CalendarSource:                 calendarSource,
		CoveredYears:                   append([]int(nil), defaultCalendar.coveredYears...),
	}

	if session == SessionMorning || session == SessionLunchBreak || session == SessionAfternoon {
		status.DisplayDate = formatTradingDate(local)
	}

	return status
}

// ParseTradeAt parses trade timestamps in the formats accepted by the holdings API.
func ParseTradeAt(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, ErrInvalidTradeTime
	}

	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.In(defaultCalendar.location), nil
	}

	layouts := []string{
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02",
	}

	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, raw, defaultCalendar.location)
		if err != nil {
			continue
		}
		return parsed, nil
	}

	if _, err := time.Parse("2006-01-02", raw); err != nil {
		return time.Time{}, ErrInvalidTradeTime
	}

	return time.Time{}, ErrInvalidTradeDate
}

// ResolvePricingDate calculates the confirmed NAV date for a holding trade timestamp.
func ResolvePricingDate(tradeAt time.Time) PricingDateResolution {
	local := normalizeToTradingLocation(tradeAt)
	tradeDate := startOfTradingDay(local)
	cutoff := time.Date(local.Year(), local.Month(), local.Day(), marketCloseHour, marketCloseMinute, 0, 0, defaultCalendar.location)

	result := PricingDateResolution{
		TradeAt:      local,
		TradeDate:    tradeDate.Format("2006-01-02"),
		PricingDate:  tradeDate.Format("2006-01-02"),
		Rule:         PricingDateSameDayClose,
		IsTradingDay: IsTradingDay(local),
		AfterCutoff:  !local.Before(cutoff),
		CutoffTime:   cutoff,
		Timezone:     tradingTimeZone,
		Message:      "15:00 前提交，按当日收盘净值确认",
	}

	if !result.IsTradingDay || result.AfterCutoff {
		nextTradingDay := GetNextTradingDay(local)
		result.PricingDate = nextTradingDay.Format("2006-01-02")
		result.Rule = PricingDateNextTradingDay
		result.NextTradingDay = result.PricingDate

		switch {
		case !result.IsTradingDay:
			result.Message = "非交易日提交，顺延至下个交易日确认"
		default:
			result.Message = "15:00 起提交，顺延至下个交易日确认"
		}
	}

	return result
}

// CalendarCoverage returns the holiday dataset coverage years.
func CalendarCoverage() []int {
	return append([]int(nil), defaultCalendar.coveredYears...)
}

// TradingLocation returns the shared market timezone.
func TradingLocation() *time.Location {
	return defaultCalendar.location
}

// TradingTimeZone returns the market timezone name used in API responses.
func TradingTimeZone() string {
	return tradingTimeZone
}

// FormatCoverage returns a readable coverage string for diagnostics.
func FormatCoverage() string {
	years := CalendarCoverage()
	if len(years) == 0 {
		return "no holiday coverage"
	}
	return fmt.Sprintf("%d-%d", years[0], years[len(years)-1])
}
