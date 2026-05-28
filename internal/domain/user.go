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
	SortOrder   int       `json:"sort_order"`
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
	SortOrder   int                       `json:"sort_order"`
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
	ID                 string          `json:"id"`
	UserID             string          `json:"user_id"`
	FundID             string          `json:"fund_id"`
	Amount             decimal.Decimal `json:"amount"`
	Shares             decimal.Decimal `json:"shares"`
	ConfirmedNav       decimal.Decimal `json:"confirmed_nav"`
	ConfirmedNavDate   string          `json:"confirmed_nav_date,omitempty"`
	ManualConfirmation bool            `json:"manual_confirmation,omitempty"`
	TradeAt            string          `json:"trade_at,omitempty"`
	AsOfDate           string          `json:"as_of_date"`
	Note               string          `json:"note"`
	SourcePlatform     string          `json:"source_platform,omitempty"`
	SourceLabel        string          `json:"source_label,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

// UserFundHoldingDetail enriches a fund holding with current fund profile data.
type UserFundHoldingDetail struct {
	ID                 string          `json:"id"`
	FundID             string          `json:"fund_id"`
	Amount             decimal.Decimal `json:"amount"`
	Shares             string          `json:"shares,omitempty"`
	ConfirmedNav       string          `json:"confirmed_nav,omitempty"`
	ConfirmedNavDate   string          `json:"confirmed_nav_date,omitempty"`
	ManualConfirmation bool            `json:"manual_confirmation,omitempty"`
	TradeAt            string          `json:"trade_at,omitempty"`
	AsOfDate           string          `json:"as_of_date"`
	ActualDate         string          `json:"actual_date,omitempty"`
	ActualNav          string          `json:"actual_nav,omitempty"`
	ActualDailyReturn  string          `json:"actual_daily_return,omitempty"`
	CurrentMarketValue string          `json:"current_market_value,omitempty"`
	TodayProfit        string          `json:"today_profit,omitempty"`
	TodayChangePercent string          `json:"today_change_percent,omitempty"`
	RealMetricsReady   bool            `json:"real_metrics_ready"`
	RealMetricsMessage string          `json:"real_metrics_message,omitempty"`
	Note               string          `json:"note"`
	SourcePlatform     string          `json:"source_platform,omitempty"`
	SourceLabel        string          `json:"source_label,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	Fund               *Fund           `json:"fund,omitempty"`
}

// UpdateFundHoldingInput describes user-editable correction fields for a fund holding record.
type UpdateFundHoldingInput struct {
	Amount           string
	Shares           string
	ConfirmedNav     string
	ConfirmedNavDate string
	TradeAt          string
	Note             string
	SourcePlatform   string
	SourceLabel      string
}

// CreateFundHoldingInput describes one user fund-level position record to create.
type CreateFundHoldingInput struct {
	FundID         string `json:"fund_id"`
	Amount         string `json:"amount"`
	TradeAt        string `json:"trade_at"`
	Note           string `json:"note"`
	SourcePlatform string `json:"source_platform,omitempty"`
	SourceLabel    string `json:"source_label,omitempty"`
}

// UserFundHoldingBatchCreateFailure records one rejected row during safe batch import.
type UserFundHoldingBatchCreateFailure struct {
	Index   int    `json:"index"`
	FundID  string `json:"fund_id,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// UserFundHoldingBatchCreateResult summarizes a safe batch import attempt.
type UserFundHoldingBatchCreateResult struct {
	Total        int                                 `json:"total"`
	CreatedCount int                                 `json:"created_count"`
	FailedCount  int                                 `json:"failed_count"`
	Created      []UserFundHoldingDetail             `json:"created"`
	Failed       []UserFundHoldingBatchCreateFailure `json:"failed,omitempty"`
}

// SellFundHoldingInput describes a user redemption/decrease operation.
type SellFundHoldingInput struct {
	Amount  string
	Shares  string
	TradeAt string
	Note    string
	SellAll bool
}

// DividendFundHoldingInput describes a cash dividend or dividend reinvestment record.
type DividendFundHoldingInput struct {
	Amount         string
	Shares         string
	TradeAt        string
	Note           string
	Reinvest       bool
	SourcePlatform string
	SourceLabel    string
}

// AdjustFundHoldingSharesInput describes a non-trade share adjustment.
type AdjustFundHoldingSharesInput struct {
	SharesDelta      string
	TargetShares     string
	ConfirmedNav     string
	ConfirmedNavDate string
	TradeAt          string
	Note             string
	SourcePlatform   string
	SourceLabel      string
}

// UserFundHoldingTransactionType identifies the purpose of a user fund holding activity.
type UserFundHoldingTransactionType string

const (
	UserFundHoldingTransactionBuy        UserFundHoldingTransactionType = "buy"
	UserFundHoldingTransactionCorrection UserFundHoldingTransactionType = "correction"
	UserFundHoldingTransactionDelete     UserFundHoldingTransactionType = "delete"
	UserFundHoldingTransactionSell       UserFundHoldingTransactionType = "sell"
	UserFundHoldingTransactionDividend   UserFundHoldingTransactionType = "dividend"
	UserFundHoldingTransactionAdjustment UserFundHoldingTransactionType = "adjustment"
)

// UserFundHoldingTransaction records user-visible activity around a fund holding.
type UserFundHoldingTransaction struct {
	ID                 string                         `json:"id"`
	UserID             string                         `json:"user_id"`
	HoldingID          string                         `json:"holding_id,omitempty"`
	FundID             string                         `json:"fund_id"`
	Type               UserFundHoldingTransactionType `json:"type"`
	Amount             decimal.Decimal                `json:"amount"`
	Shares             decimal.Decimal                `json:"shares"`
	ConfirmedNav       decimal.Decimal                `json:"confirmed_nav"`
	ConfirmedNavDate   string                         `json:"confirmed_nav_date,omitempty"`
	ManualConfirmation bool                           `json:"manual_confirmation,omitempty"`
	TradeAt            string                         `json:"trade_at,omitempty"`
	AsOfDate           string                         `json:"as_of_date,omitempty"`
	Note               string                         `json:"note"`
	SourcePlatform     string                         `json:"source_platform,omitempty"`
	SourceLabel        string                         `json:"source_label,omitempty"`
	Metadata           map[string]string              `json:"metadata,omitempty"`
	Voided             bool                           `json:"voided"`
	VoidedAt           *time.Time                     `json:"voided_at,omitempty"`
	VoidReason         string                         `json:"void_reason,omitempty"`
	CreatedAt          time.Time                      `json:"created_at"`
	Fund               *Fund                          `json:"fund,omitempty"`
}

// UserFundHoldingTransactionFilter narrows recent holding activity lookups.
type UserFundHoldingTransactionFilter struct {
	Limit          int
	Offset         int
	FundID         string
	Types          []UserFundHoldingTransactionType
	Voided         *bool
	SourcePlatform string
	Keyword        string
	CreatedFrom    *time.Time
	CreatedBefore  *time.Time
}

// UserFundHoldingTransactionRollbackField describes one field that a manual rollback/correction would touch.
type UserFundHoldingTransactionRollbackField struct {
	Field         string `json:"field"`
	Label         string `json:"label"`
	CurrentValue  string `json:"current_value,omitempty"`
	RollbackValue string `json:"rollback_value,omitempty"`
	Delta         string `json:"delta,omitempty"`
	Direction     string `json:"direction,omitempty"`
}

// UserFundHoldingTransactionRollbackPreview is a read-only impact summary for voiding/reversing one transaction.
type UserFundHoldingTransactionRollbackPreview struct {
	Transaction           UserFundHoldingTransaction                `json:"transaction"`
	CurrentHolding        *UserFundHoldingDetail                    `json:"current_holding,omitempty"`
	PreviewOnly           bool                                      `json:"preview_only"`
	CanApplyAutomatically bool                                      `json:"can_apply_automatically"`
	State                 string                                    `json:"state"`
	Title                 string                                    `json:"title"`
	Summary               string                                    `json:"summary"`
	SuggestedAction       string                                    `json:"suggested_action"`
	AffectedFields        []UserFundHoldingTransactionRollbackField `json:"affected_fields"`
	Warnings              []string                                  `json:"warnings,omitempty"`
}

// UserFundHoldingTransactionRollbackApplyResult reports a user-confirmed automatic rollback.
type UserFundHoldingTransactionRollbackApplyResult struct {
	Transaction     UserFundHoldingTransaction                `json:"transaction"`
	CurrentHolding  *UserFundHoldingDetail                    `json:"current_holding,omitempty"`
	Preview         UserFundHoldingTransactionRollbackPreview `json:"preview"`
	Applied         bool                                      `json:"applied"`
	HoldingRemoved  bool                                      `json:"holding_removed,omitempty"`
	HoldingRestored bool                                      `json:"holding_restored,omitempty"`
	Message         string                                    `json:"message"`
}

// UserFundHoldingTransactionDetail is the drill-down view for one historical holding activity.
type UserFundHoldingTransactionDetail struct {
	Transaction            UserFundHoldingTransaction                 `json:"transaction"`
	CurrentHolding         *UserFundHoldingDetail                     `json:"current_holding,omitempty"`
	RollbackPreview        *UserFundHoldingTransactionRollbackPreview `json:"rollback_preview,omitempty"`
	RelatedTransactions    []UserFundHoldingTransaction               `json:"related_transactions,omitempty"`
	SubsequentTransactions []UserFundHoldingTransaction               `json:"subsequent_transactions,omitempty"`
	ImpactChain            []string                                   `json:"impact_chain,omitempty"`
}

// UserFundHoldingSummary aggregates the user's holdings page totals.
type UserFundHoldingSummary struct {
	TotalPrincipal          decimal.Decimal `json:"total_principal"`
	ReadyPrincipal          string          `json:"ready_principal,omitempty"`
	TotalCurrentMarketValue string          `json:"total_current_market_value,omitempty"`
	TotalTodayProfit        string          `json:"total_today_profit,omitempty"`
	TotalTodayChangePercent string          `json:"total_today_change_percent,omitempty"`
	MetricsScope            string          `json:"metrics_scope,omitempty"`
	RealMetricsReady        bool            `json:"real_metrics_ready"`
	RealMetricsReadyCount   int             `json:"real_metrics_ready_count"`
	TotalHoldings           int             `json:"total_holdings"`
	IncompleteHoldingsCount int             `json:"incomplete_holdings_count"`
	Message                 string          `json:"message,omitempty"`
}

// UserFundHoldingAggregate groups multiple holding records for the same fund.
type UserFundHoldingAggregate struct {
	FundID                     string          `json:"fund_id"`
	HoldingCount               int             `json:"holding_count"`
	ConfirmedHoldingCount      int             `json:"confirmed_holding_count"`
	RealMetricsReadyCount      int             `json:"real_metrics_ready_count"`
	IncompleteHoldingsCount    int             `json:"incomplete_holdings_count"`
	TotalPrincipal             decimal.Decimal `json:"total_principal"`
	ConfirmedPrincipal         string          `json:"confirmed_principal,omitempty"`
	ReadyPrincipal             string          `json:"ready_principal,omitempty"`
	ConfirmedShares            string          `json:"confirmed_shares,omitempty"`
	OfficialCurrentMarketValue string          `json:"official_current_market_value,omitempty"`
	OfficialTodayProfit        string          `json:"official_today_profit,omitempty"`
	OfficialTodayChangePercent string          `json:"official_today_change_percent,omitempty"`
	MetricsScope               string          `json:"metrics_scope,omitempty"`
	RealMetricsReady           bool            `json:"real_metrics_ready"`
	Message                    string          `json:"message,omitempty"`
	Fund                       *Fund           `json:"fund,omitempty"`
}

// UserFundHoldingList groups holding items and summary totals.
type UserFundHoldingList struct {
	Items      []UserFundHoldingDetail    `json:"items"`
	Aggregates []UserFundHoldingAggregate `json:"aggregates,omitempty"`
	Summary    UserFundHoldingSummary     `json:"summary"`
}
