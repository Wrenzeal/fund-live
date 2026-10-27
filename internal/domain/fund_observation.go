package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// FundObservation reports disclosed holdings and recent notices without scores.
type FundObservation struct {
	CheckedAt              time.Time               `json:"checked_at"`
	WindowDays             int                     `json:"window_days"`
	LatestHoldingPeriod    string                  `json:"latest_holding_period"`
	HoldingAgeDays         *int                    `json:"holding_age_days"`
	DisclosedWeightPercent decimal.Decimal         `json:"disclosed_weight_percent"`
	HoldingsCount          int                     `json:"holdings_count"`
	HoldingsSource         string                  `json:"holdings_source"`
	HoldingsStatus         string                  `json:"holdings_status"`
	Notices                []FundObservationNotice `json:"notices"`
	Sources                []FundObservationSource `json:"sources"`
	Notes                  []string                `json:"notes"`
}

type FundObservationNotice struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Scope         string `json:"scope"`
	PublishedAt   string `json:"published_at"`
	SourceName    string `json:"source_name"`
	SourceURL     string `json:"source_url"`
	RelatedSymbol string `json:"related_symbol,omitempty"`
	RelatedName   string `json:"related_name,omitempty"`
}

type FundObservationSource struct {
	Code      string    `json:"code"`
	Status    string    `json:"status"`
	CheckedAt time.Time `json:"checked_at"`
	Message   string    `json:"message,omitempty"`
}
