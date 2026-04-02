package domain

import "context"

// UserPreferenceService defines user-owned favorite fund and holding override use cases.
type UserPreferenceService interface {
	ListFavoriteFunds(ctx context.Context, userID string) ([]UserFavoriteFundDetail, error)
	AddFavoriteFund(ctx context.Context, userID, fundID string) error
	RemoveFavoriteFund(ctx context.Context, userID, fundID string) error
	ListWatchlistGroups(ctx context.Context, userID string) ([]UserWatchlistGroupDetail, error)
	CreateWatchlistGroup(ctx context.Context, userID, name, description string) (*UserWatchlistGroup, error)
	DeleteWatchlistGroup(ctx context.Context, userID, groupID string) error
	AddWatchlistFund(ctx context.Context, userID, groupID, fundID string) error
	RemoveWatchlistFund(ctx context.Context, userID, groupID, fundID string) error
	ListFundHoldings(ctx context.Context, userID string) ([]UserFundHoldingDetail, error)
	CreateFundHolding(ctx context.Context, userID, fundID, amount, tradeAt, note string) (*UserFundHoldingDetail, error)
	DeleteFundHolding(ctx context.Context, userID, holdingID string) error
	GetHoldingOverrideSet(ctx context.Context, userID, fundID string) (*UserHoldingOverrideSet, error)
	ReplaceHoldingOverrides(ctx context.Context, userID, fundID string, overrides []UserHoldingOverride) error
}
