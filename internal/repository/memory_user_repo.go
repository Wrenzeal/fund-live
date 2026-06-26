package repository

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

// MemoryUserRepository stores user-related data in memory.
type MemoryUserRepository struct {
	mu               sync.RWMutex
	users            map[string]*domain.User
	usersByEmail     map[string]string
	usersByGoogleSub map[string]string
	sessionsByHash   map[string]*domain.UserSession
	favoriteFunds    map[string]map[string]domain.UserFavoriteFund
	watchlistGroups  map[string]map[string]domain.UserWatchlistGroup
	watchlistFunds   map[string]map[string][]domain.UserWatchlistFund
	fundHoldings     map[string]map[string]domain.UserFundHolding
	holdingTxs       map[string][]domain.UserFundHoldingTransaction
	holdingOverrides map[string]map[string][]domain.UserHoldingOverride
}

// NewMemoryUserRepository creates a new in-memory user repository.
func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{
		users:            make(map[string]*domain.User),
		usersByEmail:     make(map[string]string),
		usersByGoogleSub: make(map[string]string),
		sessionsByHash:   make(map[string]*domain.UserSession),
		favoriteFunds:    make(map[string]map[string]domain.UserFavoriteFund),
		watchlistGroups:  make(map[string]map[string]domain.UserWatchlistGroup),
		watchlistFunds:   make(map[string]map[string][]domain.UserWatchlistFund),
		fundHoldings:     make(map[string]map[string]domain.UserFundHolding),
		holdingTxs:       make(map[string][]domain.UserFundHoldingTransaction),
		holdingOverrides: make(map[string]map[string][]domain.UserHoldingOverride),
	}
}

func (r *MemoryUserRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[userID]
	if !ok {
		return nil, nil
	}
	copyUser := *user
	return &copyUser, nil
}

func (r *MemoryUserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userID, ok := r.usersByEmail[strings.ToLower(email)]
	if !ok {
		return nil, nil
	}
	user, ok := r.users[userID]
	if !ok {
		return nil, nil
	}
	copyUser := *user
	return &copyUser, nil
}

func (r *MemoryUserRepository) GetUserByGoogleSub(ctx context.Context, googleSub string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userID, ok := r.usersByGoogleSub[googleSub]
	if !ok {
		return nil, nil
	}
	user, ok := r.users[userID]
	if !ok {
		return nil, nil
	}
	copyUser := *user
	return &copyUser, nil
}

func (r *MemoryUserRepository) SaveUser(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	copyUser := *user
	copyUser.IsAdmin = user.IsAdmin
	copyUser.PreferredQuoteSource = domain.ResolveQuoteSource(user.PreferredQuoteSource, domain.QuoteSourceSina)
	r.users[user.ID] = &copyUser
	r.usersByEmail[strings.ToLower(user.Email)] = user.ID
	if user.GoogleSub != "" {
		r.usersByGoogleSub[user.GoogleSub] = user.ID
	}
	return nil
}

func (r *MemoryUserRepository) SaveSession(ctx context.Context, session *domain.UserSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	if session.LastSeenAt.IsZero() {
		session.LastSeenAt = now
	}

	copySession := *session
	r.sessionsByHash[session.TokenHash] = &copySession
	return nil
}

func (r *MemoryUserRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*domain.UserSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, ok := r.sessionsByHash[tokenHash]
	if !ok {
		return nil, nil
	}
	copySession := *session
	return &copySession, nil
}

func (r *MemoryUserRepository) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.sessionsByHash, tokenHash)
	return nil
}

func (r *MemoryUserRepository) DeleteSessionsByUserID(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for tokenHash, session := range r.sessionsByHash {
		if session.UserID == userID {
			delete(r.sessionsByHash, tokenHash)
		}
	}
	return nil
}

func (r *MemoryUserRepository) UpdateSessionLastSeen(ctx context.Context, sessionID string, seenAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for tokenHash, session := range r.sessionsByHash {
		if session.ID == sessionID {
			copySession := *session
			copySession.LastSeenAt = seenAt
			r.sessionsByHash[tokenHash] = &copySession
			return nil
		}
	}
	return nil
}

func (r *MemoryUserRepository) ListFavoriteFunds(ctx context.Context, userID string) ([]domain.UserFavoriteFund, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	favoritesByFund := r.favoriteFunds[userID]
	favorites := make([]domain.UserFavoriteFund, 0, len(favoritesByFund))
	for _, favorite := range favoritesByFund {
		favorites = append(favorites, favorite)
	}
	return favorites, nil
}

// ListDistinctFavoriteFundIDs returns all distinct legacy favorite fund IDs.
func (r *MemoryUserRepository) ListDistinctFavoriteFundIDs(ctx context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]struct{})
	for _, favoritesByFund := range r.favoriteFunds {
		for fundID := range favoritesByFund {
			fundID = strings.TrimSpace(fundID)
			if fundID == "" {
				continue
			}
			seen[fundID] = struct{}{}
		}
	}

	result := make([]string, 0, len(seen))
	for fundID := range seen {
		result = append(result, fundID)
	}
	sort.Strings(result)
	return result, nil
}

func (r *MemoryUserRepository) SaveFavoriteFund(ctx context.Context, favorite *domain.UserFavoriteFund) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if favorite.CreatedAt.IsZero() {
		favorite.CreatedAt = time.Now()
	}
	if _, ok := r.favoriteFunds[favorite.UserID]; !ok {
		r.favoriteFunds[favorite.UserID] = make(map[string]domain.UserFavoriteFund)
	}
	r.favoriteFunds[favorite.UserID][favorite.FundID] = *favorite
	return nil
}

func (r *MemoryUserRepository) DeleteFavoriteFund(ctx context.Context, userID, fundID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if favoritesByFund, ok := r.favoriteFunds[userID]; ok {
		delete(favoritesByFund, fundID)
	}
	return nil
}

func (r *MemoryUserRepository) ListHoldingOverrides(ctx context.Context, userID, fundID string) ([]domain.UserHoldingOverride, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	byFund := r.holdingOverrides[userID]
	if byFund == nil {
		return nil, nil
	}
	overrides := byFund[fundID]
	result := make([]domain.UserHoldingOverride, len(overrides))
	copy(result, overrides)
	return result, nil
}

func (r *MemoryUserRepository) ReplaceHoldingOverrides(ctx context.Context, userID, fundID string, overrides []domain.UserHoldingOverride) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.holdingOverrides[userID]; !ok {
		r.holdingOverrides[userID] = make(map[string][]domain.UserHoldingOverride)
	}

	result := make([]domain.UserHoldingOverride, len(overrides))
	copy(result, overrides)
	r.holdingOverrides[userID][fundID] = result
	return nil
}

func (r *MemoryUserRepository) ListWatchlistGroups(ctx context.Context, userID string) ([]domain.UserWatchlistGroup, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	groupMap := r.watchlistGroups[userID]
	result := make([]domain.UserWatchlistGroup, 0, len(groupMap))
	for _, group := range groupMap {
		result = append(result, group)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SortOrder != result[j].SortOrder {
			return result[i].SortOrder < result[j].SortOrder
		}
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (r *MemoryUserRepository) GetWatchlistGroupByID(ctx context.Context, userID, groupID string) (*domain.UserWatchlistGroup, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	groupMap := r.watchlistGroups[userID]
	if groupMap == nil {
		return nil, nil
	}

	group, ok := groupMap[groupID]
	if !ok {
		return nil, nil
	}

	copyGroup := group
	return &copyGroup, nil
}

func (r *MemoryUserRepository) SaveWatchlistGroup(ctx context.Context, group *domain.UserWatchlistGroup) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.watchlistGroups[group.UserID]; !ok {
		r.watchlistGroups[group.UserID] = make(map[string]domain.UserWatchlistGroup)
	}

	copyGroup := *group
	r.watchlistGroups[group.UserID][group.ID] = copyGroup
	return nil
}

func (r *MemoryUserRepository) DeleteWatchlistGroup(ctx context.Context, userID, groupID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if groupMap, ok := r.watchlistGroups[userID]; ok {
		delete(groupMap, groupID)
	}
	if fundMap, ok := r.watchlistFunds[userID]; ok {
		delete(fundMap, groupID)
	}
	return nil
}

func (r *MemoryUserRepository) ListWatchlistFunds(ctx context.Context, userID, groupID string) ([]domain.UserWatchlistFund, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	groupFunds := r.watchlistFunds[userID]
	if groupFunds == nil {
		return nil, nil
	}

	funds := groupFunds[groupID]
	result := make([]domain.UserWatchlistFund, len(funds))
	copy(result, funds)
	return result, nil
}

func (r *MemoryUserRepository) ListWatchlistFundsByGroupIDs(ctx context.Context, userID string, groupIDs []string) (map[string][]domain.UserWatchlistFund, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]domain.UserWatchlistFund, len(groupIDs))
	groupMap := r.watchlistFunds[userID]
	if groupMap == nil {
		return result, nil
	}

	for _, groupID := range groupIDs {
		funds := groupMap[groupID]
		copied := make([]domain.UserWatchlistFund, len(funds))
		copy(copied, funds)
		result[groupID] = copied
	}

	return result, nil
}

func (r *MemoryUserRepository) SaveWatchlistFund(ctx context.Context, fund *domain.UserWatchlistFund) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	userID, ok := r.findGroupOwnerLocked(fund.GroupID)
	if !ok {
		return nil
	}

	if _, ok := r.watchlistFunds[userID]; !ok {
		r.watchlistFunds[userID] = make(map[string][]domain.UserWatchlistFund)
	}

	current := r.watchlistFunds[userID][fund.GroupID]
	for _, existing := range current {
		if existing.FundID == fund.FundID {
			return nil
		}
	}

	r.watchlistFunds[userID][fund.GroupID] = append(current, *fund)
	return nil
}

func (r *MemoryUserRepository) DeleteWatchlistFund(ctx context.Context, userID, groupID, fundID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	groupFunds := r.watchlistFunds[userID]
	if groupFunds == nil {
		return nil
	}

	current := groupFunds[groupID]
	result := make([]domain.UserWatchlistFund, 0, len(current))
	for _, item := range current {
		if item.FundID != fundID {
			result = append(result, item)
		}
	}
	groupFunds[groupID] = result
	return nil
}

// ListDistinctWatchlistFundIDs returns all distinct funds tracked in grouped watchlists.
func (r *MemoryUserRepository) ListDistinctWatchlistFundIDs(ctx context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]struct{})
	for _, groupFunds := range r.watchlistFunds {
		for _, funds := range groupFunds {
			for _, fund := range funds {
				if strings.TrimSpace(fund.FundID) == "" {
					continue
				}
				seen[fund.FundID] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(seen))
	for fundID := range seen {
		result = append(result, fundID)
	}
	sort.Strings(result)
	return result, nil
}

func (r *MemoryUserRepository) ListFundHoldings(ctx context.Context, userID string) ([]domain.UserFundHolding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	holdingMap := r.fundHoldings[userID]
	result := make([]domain.UserFundHolding, 0, len(holdingMap))
	for _, holding := range holdingMap {
		result = append(result, holding)
	}
	return result, nil
}

func (r *MemoryUserRepository) GetFundHolding(ctx context.Context, userID, holdingID string) (*domain.UserFundHolding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if holdingMap, ok := r.fundHoldings[userID]; ok {
		if holding, exists := holdingMap[holdingID]; exists {
			copyHolding := holding
			return &copyHolding, nil
		}
	}
	return nil, nil
}

func (r *MemoryUserRepository) ListFundHoldingsMissingConfirmation(ctx context.Context) ([]domain.UserFundHolding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.UserFundHolding, 0)
	for _, holdingMap := range r.fundHoldings {
		for _, holding := range holdingMap {
			if holding.ManualConfirmation {
				continue
			}
			if holding.Shares.GreaterThan(decimal.Zero) && holding.ConfirmedNav.GreaterThan(decimal.Zero) && holding.ConfirmedNavDate != "" {
				continue
			}
			result = append(result, holding)
		}
	}
	return result, nil
}

func (r *MemoryUserRepository) ListDistinctFundIDs(ctx context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, holdingMap := range r.fundHoldings {
		for _, holding := range holdingMap {
			if _, ok := seen[holding.FundID]; ok {
				continue
			}
			seen[holding.FundID] = struct{}{}
			result = append(result, holding.FundID)
		}
	}
	return result, nil
}

func (r *MemoryUserRepository) SaveFundHolding(ctx context.Context, holding *domain.UserFundHolding) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.fundHoldings[holding.UserID]; !ok {
		r.fundHoldings[holding.UserID] = make(map[string]domain.UserFundHolding)
	}

	copyHolding := *holding
	r.fundHoldings[holding.UserID][holding.ID] = copyHolding
	return nil
}

func (r *MemoryUserRepository) DeleteFundHolding(ctx context.Context, userID, holdingID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if holdingMap, ok := r.fundHoldings[userID]; ok {
		delete(holdingMap, holdingID)
	}
	return nil
}

func (r *MemoryUserRepository) ListFundHoldingTransactions(ctx context.Context, userID string, limit int) ([]domain.UserFundHoldingTransaction, error) {
	return r.ListFundHoldingTransactionsFiltered(ctx, userID, domain.UserFundHoldingTransactionFilter{Limit: limit})
}

func (r *MemoryUserRepository) ListFundHoldingTransactionsFiltered(ctx context.Context, userID string, filter domain.UserFundHoldingTransactionFilter) ([]domain.UserFundHoldingTransaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	transactions := r.holdingTxs[userID]
	result := make([]domain.UserFundHoldingTransaction, 0, len(transactions))
	types := holdingTransactionTypeSet(filter.Types)
	for i := range transactions {
		tx := transactions[i]
		if filter.FundID != "" && tx.FundID != filter.FundID {
			continue
		}
		if len(types) > 0 {
			if _, ok := types[tx.Type]; !ok {
				continue
			}
		}
		if filter.Voided != nil && tx.Voided != *filter.Voided {
			continue
		}
		if filter.SourcePlatform != "" && tx.SourcePlatform != filter.SourcePlatform {
			continue
		}
		if filter.CreatedFrom != nil && tx.CreatedAt.Before(*filter.CreatedFrom) {
			continue
		}
		if filter.CreatedBefore != nil && !tx.CreatedAt.Before(*filter.CreatedBefore) {
			continue
		}
		if filter.Keyword != "" && !holdingTransactionMatchesKeyword(tx, filter.Keyword) {
			continue
		}
		result = append(result, copyFundHoldingTransaction(tx))
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	resolvedLimit := normalizeHoldingTransactionLimit(filter.Limit)
	if filter.Offset >= len(result) {
		return []domain.UserFundHoldingTransaction{}, nil
	}
	if filter.Offset > 0 {
		result = result[filter.Offset:]
	}
	if len(result) > resolvedLimit {
		result = result[:resolvedLimit]
	}
	return result, nil
}

func holdingTransactionMatchesKeyword(tx domain.UserFundHoldingTransaction, keyword string) bool {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return true
	}

	candidates := []string{
		tx.ID,
		tx.HoldingID,
		tx.FundID,
		string(tx.Type),
		tx.Note,
		tx.SourcePlatform,
		tx.SourceLabel,
		tx.ConfirmedNavDate,
		tx.TradeAt,
		tx.AsOfDate,
		tx.VoidReason,
	}
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(candidate), keyword) {
			return true
		}
	}
	for key, value := range tx.Metadata {
		if strings.Contains(strings.ToLower(key), keyword) || strings.Contains(strings.ToLower(value), keyword) {
			return true
		}
	}
	return false
}

func holdingTransactionTypeSet(types []domain.UserFundHoldingTransactionType) map[domain.UserFundHoldingTransactionType]struct{} {
	if len(types) == 0 {
		return nil
	}

	result := make(map[domain.UserFundHoldingTransactionType]struct{}, len(types))
	for _, txType := range types {
		if txType == "" {
			continue
		}
		result[txType] = struct{}{}
	}
	return result
}

func (r *MemoryUserRepository) GetFundHoldingTransaction(ctx context.Context, userID, transactionID string) (*domain.UserFundHoldingTransaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, tx := range r.holdingTxs[userID] {
		if tx.ID == transactionID {
			copyTx := copyFundHoldingTransaction(tx)
			return &copyTx, nil
		}
	}
	return nil, nil
}

func (r *MemoryUserRepository) SaveFundHoldingTransaction(ctx context.Context, tx *domain.UserFundHoldingTransaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tx.CreatedAt.IsZero() {
		tx.CreatedAt = time.Now()
	}
	copyTx := copyFundHoldingTransaction(*tx)
	r.holdingTxs[tx.UserID] = append(r.holdingTxs[tx.UserID], copyTx)
	return nil
}

func (r *MemoryUserRepository) VoidFundHoldingTransaction(ctx context.Context, userID, transactionID, reason string, voidedAt time.Time) (*domain.UserFundHoldingTransaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index, tx := range r.holdingTxs[userID] {
		if tx.ID != transactionID {
			continue
		}
		updated := copyFundHoldingTransaction(tx)
		updated.Voided = true
		updated.VoidedAt = &voidedAt
		updated.VoidReason = reason
		r.holdingTxs[userID][index] = updated
		copyTx := copyFundHoldingTransaction(updated)
		return &copyTx, nil
	}
	return nil, nil
}

func copyFundHoldingTransaction(tx domain.UserFundHoldingTransaction) domain.UserFundHoldingTransaction {
	copyTx := tx
	if tx.Metadata != nil {
		copyTx.Metadata = make(map[string]string, len(tx.Metadata))
		for key, value := range tx.Metadata {
			copyTx.Metadata[key] = value
		}
	}
	if tx.VoidedAt != nil {
		voidedAt := *tx.VoidedAt
		copyTx.VoidedAt = &voidedAt
	}
	return copyTx
}

func (r *MemoryUserRepository) findGroupOwnerLocked(groupID string) (string, bool) {
	for userID, groups := range r.watchlistGroups {
		if _, ok := groups[groupID]; ok {
			return userID, true
		}
	}
	return "", false
}
