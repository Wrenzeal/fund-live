package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// AuthProvider identifies the primary authentication source for a user.
type AuthProvider string

const (
	AuthProviderPassword AuthProvider = "password"
	AuthProviderGoogle   AuthProvider = "google"
	AuthProviderHybrid   AuthProvider = "hybrid"
)

// User represents a system user in the domain layer.
type User struct {
	ID                   string       `json:"id"`
	Email                string       `json:"email"`
	DisplayName          string       `json:"display_name"`
	AvatarURL            string       `json:"avatar_url"`
	IsAdmin              bool         `json:"is_admin"`
	PreferredQuoteSource QuoteSource  `json:"preferred_quote_source"`
	PasswordHash         string       `json:"-"`
	GoogleSub            string       `json:"-"`
	Provider             AuthProvider `json:"provider"`
	EmailVerified        bool         `json:"email_verified"`
	LastLoginAt          *time.Time   `json:"last_login_at,omitempty"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

// UserSession represents a server-side authenticated session.
type UserSession struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	TokenHash  string    `json:"-"`
	UserAgent  string    `json:"user_agent"`
	IPAddress  string    `json:"ip_address"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

// UserFavoriteFund stores a fund selected by a user.
type UserFavoriteFund struct {
	UserID    string    `json:"user_id"`
	FundID    string    `json:"fund_id"`
	CreatedAt time.Time `json:"created_at"`
}

// UserFavoriteFundDetail enriches a favorite fund with the current fund profile.
type UserFavoriteFundDetail struct {
	FundID    string    `json:"fund_id"`
	CreatedAt time.Time `json:"created_at"`
	Fund      *Fund     `json:"fund,omitempty"`
}

// UserWatchlistGroup stores a user's named watchlist bucket.
type UserWatchlistGroup struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Accent      string    `json:"accent"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserWatchlistFund stores a fund assigned to a watchlist group.
type UserWatchlistFund struct {
	GroupID   string    `json:"group_id"`
	FundID    string    `json:"fund_id"`
	CreatedAt time.Time `json:"created_at"`
}

// UserWatchlistFundDetail enriches a watchlist fund with fund profile data.
type UserWatchlistFundDetail struct {
	FundID    string    `json:"fund_id"`
	CreatedAt time.Time `json:"created_at"`
	Fund      *Fund     `json:"fund,omitempty"`
}

// UserWatchlistGroupDetail returns a watchlist group with its funds.
type UserWatchlistGroupDetail struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Accent      string                    `json:"accent"`
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
	Funds       []UserWatchlistFundDetail `json:"funds"`
}

// UserHoldingOverride stores user-managed holdings for a specific fund.
type UserHoldingOverride struct {
	ID           string          `json:"id"`
	UserID       string          `json:"user_id"`
	FundID       string          `json:"fund_id"`
	StockCode    string          `json:"stock_code"`
	StockName    string          `json:"stock_name"`
	Exchange     Exchange        `json:"exchange"`
	HoldingRatio decimal.Decimal `json:"holding_ratio"`
	Note         string          `json:"note"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// UserHoldingOverrideSet groups holding overrides with their parent fund.
type UserHoldingOverrideSet struct {
	Fund      *Fund                 `json:"fund,omitempty"`
	Overrides []UserHoldingOverride `json:"overrides"`
}

// UserFundHolding stores a user's fund-level position record.
type UserFundHolding struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	FundID    string          `json:"fund_id"`
	Amount    decimal.Decimal `json:"amount"`
	TradeAt   string          `json:"trade_at,omitempty"`
	AsOfDate  string          `json:"as_of_date"`
	Note      string          `json:"note"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// UserFundHoldingDetail enriches a fund holding with current fund profile data.
type UserFundHoldingDetail struct {
	ID                string          `json:"id"`
	FundID            string          `json:"fund_id"`
	Amount            decimal.Decimal `json:"amount"`
	TradeAt           string          `json:"trade_at,omitempty"`
	AsOfDate          string          `json:"as_of_date"`
	ActualDate        string          `json:"actual_date,omitempty"`
	ActualNav         string          `json:"actual_nav,omitempty"`
	ActualDailyReturn string          `json:"actual_daily_return,omitempty"`
	Note              string          `json:"note"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	Fund              *Fund           `json:"fund,omitempty"`
}
