package domain

import "context"

// UserPreferenceService defines user-owned favorite fund and holding override use cases.
type UserPreferenceService interface {
	ListFavoriteFunds(ctx context.Context, userID string) ([]UserFavoriteFundDetail, error)
	AddFavoriteFund(ctx context.Context, userID, fundID string) error
	RemoveFavoriteFund(ctx context.Context, userID, fundID string) error
	ListWatchlistGroups(ctx context.Context, userID string) ([]UserWatchlistGroupDetail, error)
	CreateWatchlistGroup(ctx context.Context, userID, name, description string) (*UserWatchlistGroup, error)
	UpdateWatchlistGroup(ctx context.Context, userID, groupID, name, description, accent string) (*UserWatchlistGroup, error)
	ReorderWatchlistGroups(ctx context.Context, userID string, groupIDs []string) error
	DeleteWatchlistGroup(ctx context.Context, userID, groupID string) error
	AddWatchlistFund(ctx context.Context, userID, groupID, fundID string) error
	RemoveWatchlistFund(ctx context.Context, userID, groupID, fundID string) error
	ListFundHoldings(ctx context.Context, userID string) (*UserFundHoldingList, error)
	CreateFundHolding(ctx context.Context, userID, fundID, amount, tradeAt, note string) (*UserFundHoldingDetail, error)
	CreateFundHoldingsBatch(ctx context.Context, userID string, inputs []CreateFundHoldingInput) (*UserFundHoldingBatchCreateResult, error)
	UpdateFundHolding(ctx context.Context, userID, holdingID string, input UpdateFundHoldingInput) (*UserFundHoldingDetail, error)
	SellFundHolding(ctx context.Context, userID, holdingID string, input SellFundHoldingInput) (*UserFundHoldingDetail, error)
	RecordFundHoldingDividend(ctx context.Context, userID, holdingID string, input DividendFundHoldingInput) (*UserFundHoldingDetail, error)
	AdjustFundHoldingShares(ctx context.Context, userID, holdingID string, input AdjustFundHoldingSharesInput) (*UserFundHoldingDetail, error)
	DeleteFundHolding(ctx context.Context, userID, holdingID string) error
	ListFundHoldingTransactions(ctx context.Context, userID string, limit int) ([]UserFundHoldingTransaction, error)
	ListFundHoldingTransactionsFiltered(ctx context.Context, userID string, filter UserFundHoldingTransactionFilter) ([]UserFundHoldingTransaction, error)
	GetFundHoldingTransactionDetail(ctx context.Context, userID, transactionID string) (*UserFundHoldingTransactionDetail, error)
	VoidFundHoldingTransaction(ctx context.Context, userID, transactionID, reason string) (*UserFundHoldingTransaction, error)
	PreviewFundHoldingTransactionRollback(ctx context.Context, userID, transactionID string) (*UserFundHoldingTransactionRollbackPreview, error)
	ApplyFundHoldingTransactionRollback(ctx context.Context, userID, transactionID, reason string) (*UserFundHoldingTransactionRollbackApplyResult, error)
	GetHoldingOverrideSet(ctx context.Context, userID, fundID string) (*UserHoldingOverrideSet, error)
	ReplaceHoldingOverrides(ctx context.Context, userID, fundID string, overrides []UserHoldingOverride) error
}
