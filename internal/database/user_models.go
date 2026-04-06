package database

import (
	"time"

	"github.com/shopspring/decimal"
)

// User is the GORM model for application users.
type User struct {
	ID                   string     `gorm:"primaryKey;type:varchar(40)" json:"id"`
	Email                string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	DisplayName          string     `gorm:"type:varchar(100);not null" json:"display_name"`
	AvatarURL            string     `gorm:"type:text" json:"avatar_url"`
	IsAdmin              bool       `gorm:"not null;default:false;index" json:"is_admin"`
	PreferredQuoteSource string     `gorm:"type:varchar(20);default:'sina'" json:"preferred_quote_source"`
	PasswordHash         string     `gorm:"type:varchar(255)" json:"-"`
	GoogleSub            *string    `gorm:"type:varchar(255);uniqueIndex" json:"-"`
	Provider             string     `gorm:"type:varchar(20);index;not null" json:"provider"`
	EmailVerified        bool       `gorm:"default:false" json:"email_verified"`
	LastLoginAt          *time.Time `json:"last_login_at,omitempty"`
	CreatedAt            time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string {
	return "tb_user"
}

// UserSession is the GORM model for server-side sessions.
type UserSession struct {
	ID         string    `gorm:"primaryKey;type:varchar(40)" json:"id"`
	UserID     string    `gorm:"type:varchar(40);index;not null" json:"user_id"`
	TokenHash  string    `gorm:"type:char(64);uniqueIndex;not null" json:"-"`
	UserAgent  string    `gorm:"type:text" json:"user_agent"`
	IPAddress  string    `gorm:"type:varchar(64)" json:"ip_address"`
	ExpiresAt  time.Time `gorm:"index;not null" json:"expires_at"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	LastSeenAt time.Time `gorm:"autoUpdateTime" json:"last_seen_at"`
}

func (UserSession) TableName() string {
	return "tb_user_session"
}

// UserFavoriteFund is the GORM model for user favorite funds.
type UserFavoriteFund struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    string    `gorm:"type:varchar(40);uniqueIndex:idx_user_favorite_fund,priority:1;not null" json:"user_id"`
	FundID    string    `gorm:"type:varchar(10);uniqueIndex:idx_user_favorite_fund,priority:2;index;not null" json:"fund_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (UserFavoriteFund) TableName() string {
	return "tb_user_favorite_fund"
}

// UserWatchlistGroup is the GORM model for grouped watchlists.
type UserWatchlistGroup struct {
	ID          string    `gorm:"primaryKey;type:varchar(40)" json:"id"`
	UserID      string    `gorm:"type:varchar(40);index;not null" json:"user_id"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Accent      string    `gorm:"type:varchar(32);not null" json:"accent"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserWatchlistGroup) TableName() string {
	return "tb_user_watchlist_group"
}

// UserWatchlistFund is the GORM model for watchlist group membership.
type UserWatchlistFund struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID   string    `gorm:"type:varchar(40);uniqueIndex:idx_user_watchlist_group_fund,priority:1;index;not null" json:"group_id"`
	FundID    string    `gorm:"type:varchar(10);uniqueIndex:idx_user_watchlist_group_fund,priority:2;index;not null" json:"fund_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (UserWatchlistFund) TableName() string {
	return "tb_user_watchlist_fund"
}

// UserHoldingOverride is the GORM model for user-defined holdings.
type UserHoldingOverride struct {
	ID           string          `gorm:"primaryKey;type:varchar(40)" json:"id"`
	UserID       string          `gorm:"type:varchar(40);index:idx_user_holding_fund,priority:1;uniqueIndex:idx_user_holding_stock,priority:1;not null" json:"user_id"`
	FundID       string          `gorm:"type:varchar(10);index:idx_user_holding_fund,priority:2;uniqueIndex:idx_user_holding_stock,priority:2;not null" json:"fund_id"`
	StockCode    string          `gorm:"type:varchar(10);uniqueIndex:idx_user_holding_stock,priority:3;not null" json:"stock_code"`
	StockName    string          `gorm:"type:varchar(100);not null" json:"stock_name"`
	Exchange     string          `gorm:"type:varchar(8);not null" json:"exchange"`
	HoldingRatio decimal.Decimal `gorm:"type:decimal(8,4);not null" json:"holding_ratio"`
	Note         string          `gorm:"type:text" json:"note"`
	CreatedAt    time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserHoldingOverride) TableName() string {
	return "tb_user_holding_override"
}

// UserFundHolding is the GORM model for user fund-level positions.
type UserFundHolding struct {
	ID        string          `gorm:"primaryKey;type:varchar(40)" json:"id"`
	UserID    string          `gorm:"type:varchar(40);index:idx_user_fund_holding_user_created,priority:1;not null" json:"user_id"`
	FundID    string          `gorm:"type:varchar(10);index;not null" json:"fund_id"`
	Amount    decimal.Decimal `gorm:"type:decimal(18,2);not null" json:"amount"`
	TradeAt   *time.Time      `gorm:"type:timestamptz;index" json:"trade_at,omitempty"`
	AsOfDate  time.Time       `gorm:"type:date;not null" json:"as_of_date"`
	Note      string          `gorm:"type:text" json:"note"`
	CreatedAt time.Time       `gorm:"autoCreateTime;index:idx_user_fund_holding_user_created,priority:2" json:"created_at"`
	UpdatedAt time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserFundHolding) TableName() string {
	return "tb_user_fund_holding"
}

// UserModels returns all user-related models.
func UserModels() []interface{} {
	return []interface{}{
		&User{},
		&UserSession{},
		&UserFavoriteFund{},
		&UserWatchlistGroup{},
		&UserWatchlistFund{},
		&UserHoldingOverride{},
		&UserFundHolding{},
		&UserMembership{},
		&VIPUsageDaily{},
		&AnalysisTask{},
		&AnalysisReport{},
		&AnalysisReportSource{},
		&VIPOrder{},
		&Issue{},
		&Announcement{},
		&AnnouncementRead{},
	}
}
