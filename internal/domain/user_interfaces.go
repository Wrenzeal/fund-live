package domain

import (
	"context"
	"time"
)

// UserRepository defines persistence operations for users.
type UserRepository interface {
	GetUserByID(ctx context.Context, userID string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByGoogleSub(ctx context.Context, googleSub string) (*User, error)
	SaveUser(ctx context.Context, user *User) error
}

// UserSessionRepository defines persistence operations for authenticated sessions.
type UserSessionRepository interface {
	SaveSession(ctx context.Context, session *UserSession) error
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (*UserSession, error)
	DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error
	DeleteSessionsByUserID(ctx context.Context, userID string) error
	UpdateSessionLastSeen(ctx context.Context, sessionID string, seenAt time.Time) error
}

// UserFavoriteRepository defines persistence operations for user favorite funds.
type UserFavoriteRepository interface {
	ListFavoriteFunds(ctx context.Context, userID string) ([]UserFavoriteFund, error)
	SaveFavoriteFund(ctx context.Context, favorite *UserFavoriteFund) error
	DeleteFavoriteFund(ctx context.Context, userID, fundID string) error
}

// UserHoldingOverrideRepository defines persistence operations for user-defined holdings.
type UserHoldingOverrideRepository interface {
	ListHoldingOverrides(ctx context.Context, userID, fundID string) ([]UserHoldingOverride, error)
	ReplaceHoldingOverrides(ctx context.Context, userID, fundID string, overrides []UserHoldingOverride) error
}

// UserWatchlistRepository defines persistence operations for grouped user watchlists.
type UserWatchlistRepository interface {
	ListWatchlistGroups(ctx context.Context, userID string) ([]UserWatchlistGroup, error)
	GetWatchlistGroupByID(ctx context.Context, userID, groupID string) (*UserWatchlistGroup, error)
	SaveWatchlistGroup(ctx context.Context, group *UserWatchlistGroup) error
	DeleteWatchlistGroup(ctx context.Context, userID, groupID string) error
	ListWatchlistFunds(ctx context.Context, userID, groupID string) ([]UserWatchlistFund, error)
	ListWatchlistFundsByGroupIDs(ctx context.Context, userID string, groupIDs []string) (map[string][]UserWatchlistFund, error)
	SaveWatchlistFund(ctx context.Context, fund *UserWatchlistFund) error
	DeleteWatchlistFund(ctx context.Context, userID, groupID, fundID string) error
}

// UserFundHoldingRepository defines persistence operations for user fund-level positions.
type UserFundHoldingRepository interface {
	ListFundHoldings(ctx context.Context, userID string) ([]UserFundHolding, error)
	ListDistinctFundIDs(ctx context.Context) ([]string, error)
	SaveFundHolding(ctx context.Context, holding *UserFundHolding) error
	DeleteFundHolding(ctx context.Context, userID, holdingID string) error
}
