package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PostgresUserRepository stores user-related data in PostgreSQL.
type PostgresUserRepository struct {
	db *gorm.DB
}

// NewPostgresUserRepository creates a PostgreSQL-backed user repository.
func NewPostgresUserRepository(db *gorm.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	var dbUser database.User
	result := r.db.WithContext(ctx).First(&dbUser, "id = ?", userID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by id: %w", result.Error)
	}
	return r.toDomainUser(&dbUser), nil
}

func (r *PostgresUserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var dbUser database.User
	result := r.db.WithContext(ctx).First(&dbUser, "email = ?", email)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", result.Error)
	}
	return r.toDomainUser(&dbUser), nil
}

func (r *PostgresUserRepository) GetUserByGoogleSub(ctx context.Context, googleSub string) (*domain.User, error) {
	if googleSub == "" {
		return nil, nil
	}

	var dbUser database.User
	result := r.db.WithContext(ctx).First(&dbUser, "google_sub = ?", googleSub)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by google sub: %w", result.Error)
	}
	return r.toDomainUser(&dbUser), nil
}

func (r *PostgresUserRepository) SaveUser(ctx context.Context, user *domain.User) error {
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	dbUser := r.toDBUser(user)
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"email",
			"display_name",
			"avatar_url",
			"preferred_quote_source",
			"password_hash",
			"google_sub",
			"provider",
			"email_verified",
			"last_login_at",
			"updated_at",
		}),
	}).Create(dbUser)
	if result.Error != nil {
		return fmt.Errorf("failed to save user: %w", result.Error)
	}
	return nil
}

func (r *PostgresUserRepository) SaveSession(ctx context.Context, session *domain.UserSession) error {
	now := time.Now()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	if session.LastSeenAt.IsZero() {
		session.LastSeenAt = now
	}

	dbSession := &database.UserSession{
		ID:         session.ID,
		UserID:     session.UserID,
		TokenHash:  session.TokenHash,
		UserAgent:  session.UserAgent,
		IPAddress:  session.IPAddress,
		ExpiresAt:  session.ExpiresAt,
		CreatedAt:  session.CreatedAt,
		LastSeenAt: session.LastSeenAt,
	}
	if err := r.db.WithContext(ctx).Create(dbSession).Error; err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*domain.UserSession, error) {
	var dbSession database.UserSession
	result := r.db.WithContext(ctx).First(&dbSession, "token_hash = ?", tokenHash)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", result.Error)
	}
	return &domain.UserSession{
		ID:         dbSession.ID,
		UserID:     dbSession.UserID,
		TokenHash:  dbSession.TokenHash,
		UserAgent:  dbSession.UserAgent,
		IPAddress:  dbSession.IPAddress,
		ExpiresAt:  dbSession.ExpiresAt,
		CreatedAt:  dbSession.CreatedAt,
		LastSeenAt: dbSession.LastSeenAt,
	}, nil
}

func (r *PostgresUserRepository) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).Delete(&database.UserSession{}).Error; err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) DeleteSessionsByUserID(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&database.UserSession{}).Error; err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) UpdateSessionLastSeen(ctx context.Context, sessionID string, seenAt time.Time) error {
	if err := r.db.WithContext(ctx).Model(&database.UserSession{}).Where("id = ?", sessionID).Update("last_seen_at", seenAt).Error; err != nil {
		return fmt.Errorf("failed to update session timestamp: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) ListFavoriteFunds(ctx context.Context, userID string) ([]domain.UserFavoriteFund, error) {
	var dbFavorites []database.UserFavoriteFund
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&dbFavorites).Error; err != nil {
		return nil, fmt.Errorf("failed to list favorite funds: %w", err)
	}

	favorites := make([]domain.UserFavoriteFund, 0, len(dbFavorites))
	for _, favorite := range dbFavorites {
		favorites = append(favorites, domain.UserFavoriteFund{
			UserID:    favorite.UserID,
			FundID:    favorite.FundID,
			CreatedAt: favorite.CreatedAt,
		})
	}
	return favorites, nil
}

func (r *PostgresUserRepository) SaveFavoriteFund(ctx context.Context, favorite *domain.UserFavoriteFund) error {
	if favorite.CreatedAt.IsZero() {
		favorite.CreatedAt = time.Now()
	}

	dbFavorite := &database.UserFavoriteFund{
		UserID:    favorite.UserID,
		FundID:    favorite.FundID,
		CreatedAt: favorite.CreatedAt,
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(dbFavorite).Error; err != nil {
		return fmt.Errorf("failed to save favorite fund: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) DeleteFavoriteFund(ctx context.Context, userID, fundID string) error {
	if err := r.db.WithContext(ctx).Where("user_id = ? AND fund_id = ?", userID, fundID).Delete(&database.UserFavoriteFund{}).Error; err != nil {
		return fmt.Errorf("failed to delete favorite fund: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) ListHoldingOverrides(ctx context.Context, userID, fundID string) ([]domain.UserHoldingOverride, error) {
	var dbOverrides []database.UserHoldingOverride
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND fund_id = ?", userID, fundID).
		Order("holding_ratio DESC").
		Find(&dbOverrides).Error; err != nil {
		return nil, fmt.Errorf("failed to list holding overrides: %w", err)
	}

	overrides := make([]domain.UserHoldingOverride, 0, len(dbOverrides))
	for _, override := range dbOverrides {
		overrides = append(overrides, domain.UserHoldingOverride{
			ID:           override.ID,
			UserID:       override.UserID,
			FundID:       override.FundID,
			StockCode:    override.StockCode,
			StockName:    override.StockName,
			Exchange:     domain.Exchange(override.Exchange),
			HoldingRatio: override.HoldingRatio,
			Note:         override.Note,
			CreatedAt:    override.CreatedAt,
			UpdatedAt:    override.UpdatedAt,
		})
	}
	return overrides, nil
}

func (r *PostgresUserRepository) ReplaceHoldingOverrides(ctx context.Context, userID, fundID string, overrides []domain.UserHoldingOverride) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND fund_id = ?", userID, fundID).Delete(&database.UserHoldingOverride{}).Error; err != nil {
			return fmt.Errorf("failed to delete old holding overrides: %w", err)
		}

		if len(overrides) == 0 {
			return nil
		}

		now := time.Now()
		dbOverrides := make([]database.UserHoldingOverride, 0, len(overrides))
		for _, override := range overrides {
			createdAt := override.CreatedAt
			if createdAt.IsZero() {
				createdAt = now
			}
			updatedAt := override.UpdatedAt
			if updatedAt.IsZero() {
				updatedAt = now
			}

			dbOverrides = append(dbOverrides, database.UserHoldingOverride{
				ID:           override.ID,
				UserID:       userID,
				FundID:       fundID,
				StockCode:    override.StockCode,
				StockName:    override.StockName,
				Exchange:     string(override.Exchange),
				HoldingRatio: override.HoldingRatio,
				Note:         override.Note,
				CreatedAt:    createdAt,
				UpdatedAt:    updatedAt,
			})
		}

		if err := tx.Create(&dbOverrides).Error; err != nil {
			return fmt.Errorf("failed to insert holding overrides: %w", err)
		}
		return nil
	})
}

func (r *PostgresUserRepository) ListWatchlistGroups(ctx context.Context, userID string) ([]domain.UserWatchlistGroup, error) {
	var dbGroups []database.UserWatchlistGroup
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&dbGroups).Error; err != nil {
		return nil, fmt.Errorf("failed to list watchlist groups: %w", err)
	}

	result := make([]domain.UserWatchlistGroup, 0, len(dbGroups))
	for _, group := range dbGroups {
		result = append(result, domain.UserWatchlistGroup{
			ID:          group.ID,
			UserID:      group.UserID,
			Name:        group.Name,
			Description: group.Description,
			Accent:      group.Accent,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		})
	}
	return result, nil
}

func (r *PostgresUserRepository) GetWatchlistGroupByID(ctx context.Context, userID, groupID string) (*domain.UserWatchlistGroup, error) {
	var dbGroup database.UserWatchlistGroup
	result := r.db.WithContext(ctx).First(&dbGroup, "id = ? AND user_id = ?", groupID, userID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get watchlist group: %w", result.Error)
	}

	return &domain.UserWatchlistGroup{
		ID:          dbGroup.ID,
		UserID:      dbGroup.UserID,
		Name:        dbGroup.Name,
		Description: dbGroup.Description,
		Accent:      dbGroup.Accent,
		CreatedAt:   dbGroup.CreatedAt,
		UpdatedAt:   dbGroup.UpdatedAt,
	}, nil
}

func (r *PostgresUserRepository) SaveWatchlistGroup(ctx context.Context, group *domain.UserWatchlistGroup) error {
	dbGroup := &database.UserWatchlistGroup{
		ID:          group.ID,
		UserID:      group.UserID,
		Name:        group.Name,
		Description: group.Description,
		Accent:      group.Accent,
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
	}

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name",
			"description",
			"accent",
			"updated_at",
		}),
	}).Create(dbGroup)
	if result.Error != nil {
		return fmt.Errorf("failed to save watchlist group: %w", result.Error)
	}
	return nil
}

func (r *PostgresUserRepository) DeleteWatchlistGroup(ctx context.Context, userID, groupID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Joins("JOIN tb_user_watchlist_group ON tb_user_watchlist_group.id = tb_user_watchlist_fund.group_id").
			Where("tb_user_watchlist_group.user_id = ? AND tb_user_watchlist_fund.group_id = ?", userID, groupID).
			Delete(&database.UserWatchlistFund{}).Error; err != nil {
			return fmt.Errorf("failed to delete watchlist group funds: %w", err)
		}

		if err := tx.Where("user_id = ? AND id = ?", userID, groupID).Delete(&database.UserWatchlistGroup{}).Error; err != nil {
			return fmt.Errorf("failed to delete watchlist group: %w", err)
		}
		return nil
	})
}

func (r *PostgresUserRepository) ListWatchlistFunds(ctx context.Context, userID, groupID string) ([]domain.UserWatchlistFund, error) {
	var dbFunds []database.UserWatchlistFund
	if err := r.db.WithContext(ctx).
		Table("tb_user_watchlist_fund").
		Select("tb_user_watchlist_fund.*").
		Joins("JOIN tb_user_watchlist_group ON tb_user_watchlist_group.id = tb_user_watchlist_fund.group_id").
		Where("tb_user_watchlist_group.user_id = ? AND tb_user_watchlist_fund.group_id = ?", userID, groupID).
		Order("tb_user_watchlist_fund.created_at ASC").
		Scan(&dbFunds).Error; err != nil {
		return nil, fmt.Errorf("failed to list watchlist funds: %w", err)
	}

	result := make([]domain.UserWatchlistFund, 0, len(dbFunds))
	for _, fund := range dbFunds {
		result = append(result, domain.UserWatchlistFund{
			GroupID:   fund.GroupID,
			FundID:    fund.FundID,
			CreatedAt: fund.CreatedAt,
		})
	}
	return result, nil
}

func (r *PostgresUserRepository) ListWatchlistFundsByGroupIDs(ctx context.Context, userID string, groupIDs []string) (map[string][]domain.UserWatchlistFund, error) {
	resultMap := make(map[string][]domain.UserWatchlistFund)
	if len(groupIDs) == 0 {
		return resultMap, nil
	}

	var dbFunds []database.UserWatchlistFund
	if err := r.db.WithContext(ctx).
		Table("tb_user_watchlist_fund").
		Select("tb_user_watchlist_fund.*").
		Joins("JOIN tb_user_watchlist_group ON tb_user_watchlist_group.id = tb_user_watchlist_fund.group_id").
		Where("tb_user_watchlist_group.user_id = ? AND tb_user_watchlist_fund.group_id IN ?", userID, groupIDs).
		Order("tb_user_watchlist_fund.group_id ASC, tb_user_watchlist_fund.created_at ASC").
		Scan(&dbFunds).Error; err != nil {
		return nil, fmt.Errorf("failed to list watchlist funds by group ids: %w", err)
	}

	for _, fund := range dbFunds {
		resultMap[fund.GroupID] = append(resultMap[fund.GroupID], domain.UserWatchlistFund{
			GroupID:   fund.GroupID,
			FundID:    fund.FundID,
			CreatedAt: fund.CreatedAt,
		})
	}

	return resultMap, nil
}

func (r *PostgresUserRepository) SaveWatchlistFund(ctx context.Context, fund *domain.UserWatchlistFund) error {
	dbFund := &database.UserWatchlistFund{
		GroupID:   fund.GroupID,
		FundID:    fund.FundID,
		CreatedAt: fund.CreatedAt,
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(dbFund).Error; err != nil {
		return fmt.Errorf("failed to save watchlist fund: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) DeleteWatchlistFund(ctx context.Context, userID, groupID, fundID string) error {
	if err := r.db.WithContext(ctx).
		Table("tb_user_watchlist_fund").
		Joins("JOIN tb_user_watchlist_group ON tb_user_watchlist_group.id = tb_user_watchlist_fund.group_id").
		Where("tb_user_watchlist_group.user_id = ? AND tb_user_watchlist_fund.group_id = ? AND tb_user_watchlist_fund.fund_id = ?", userID, groupID, fundID).
		Delete(&database.UserWatchlistFund{}).Error; err != nil {
		return fmt.Errorf("failed to delete watchlist fund: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) ListFundHoldings(ctx context.Context, userID string) ([]domain.UserFundHolding, error) {
	var dbHoldings []database.UserFundHolding
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("trade_at DESC NULLS LAST, created_at DESC").
		Find(&dbHoldings).Error; err != nil {
		return nil, fmt.Errorf("failed to list fund holdings: %w", err)
	}

	result := make([]domain.UserFundHolding, 0, len(dbHoldings))
	for _, holding := range dbHoldings {
		tradeAt := ""
		if holding.TradeAt != nil {
			tradeAt = holding.TradeAt.Format(time.RFC3339)
		}

		result = append(result, domain.UserFundHolding{
			ID:        holding.ID,
			UserID:    holding.UserID,
			FundID:    holding.FundID,
			Amount:    holding.Amount,
			TradeAt:   tradeAt,
			AsOfDate:  holding.AsOfDate.Format("2006-01-02"),
			Note:      holding.Note,
			CreatedAt: holding.CreatedAt,
			UpdatedAt: holding.UpdatedAt,
		})
	}
	return result, nil
}

func (r *PostgresUserRepository) ListDistinctFundIDs(ctx context.Context) ([]string, error) {
	var fundIDs []string
	if err := r.db.WithContext(ctx).
		Model(&database.UserFundHolding{}).
		Distinct("fund_id").
		Pluck("fund_id", &fundIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to list distinct user holding fund ids: %w", err)
	}
	return fundIDs, nil
}

func (r *PostgresUserRepository) SaveFundHolding(ctx context.Context, holding *domain.UserFundHolding) error {
	asOfDate, err := time.Parse("2006-01-02", holding.AsOfDate)
	if err != nil {
		return fmt.Errorf("failed to parse fund holding date: %w", err)
	}

	var tradeAt *time.Time
	if holding.TradeAt != "" {
		parsedTradeAt, err := time.Parse(time.RFC3339, holding.TradeAt)
		if err != nil {
			return fmt.Errorf("failed to parse fund holding trade time: %w", err)
		}
		tradeAt = &parsedTradeAt
	}

	dbHolding := &database.UserFundHolding{
		ID:        holding.ID,
		UserID:    holding.UserID,
		FundID:    holding.FundID,
		Amount:    holding.Amount,
		TradeAt:   tradeAt,
		AsOfDate:  asOfDate,
		Note:      holding.Note,
		CreatedAt: holding.CreatedAt,
		UpdatedAt: holding.UpdatedAt,
	}
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"fund_id",
			"amount",
			"trade_at",
			"as_of_date",
			"note",
			"updated_at",
		}),
	}).Create(dbHolding)
	if result.Error != nil {
		return fmt.Errorf("failed to save fund holding: %w", result.Error)
	}
	return nil
}

func (r *PostgresUserRepository) DeleteFundHolding(ctx context.Context, userID, holdingID string) error {
	if err := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, holdingID).Delete(&database.UserFundHolding{}).Error; err != nil {
		return fmt.Errorf("failed to delete fund holding: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) toDomainUser(dbUser *database.User) *domain.User {
	googleSub := ""
	if dbUser.GoogleSub != nil {
		googleSub = *dbUser.GoogleSub
	}

	return &domain.User{
		ID:                   dbUser.ID,
		Email:                dbUser.Email,
		DisplayName:          dbUser.DisplayName,
		AvatarURL:            dbUser.AvatarURL,
		PreferredQuoteSource: domain.NormalizeQuoteSource(dbUser.PreferredQuoteSource),
		PasswordHash:         dbUser.PasswordHash,
		GoogleSub:            googleSub,
		Provider:             domain.AuthProvider(dbUser.Provider),
		EmailVerified:        dbUser.EmailVerified,
		LastLoginAt:          dbUser.LastLoginAt,
		CreatedAt:            dbUser.CreatedAt,
		UpdatedAt:            dbUser.UpdatedAt,
	}
}

func (r *PostgresUserRepository) toDBUser(user *domain.User) *database.User {
	var googleSub *string
	if user.GoogleSub != "" {
		googleSub = &user.GoogleSub
	}

	return &database.User{
		ID:                   user.ID,
		Email:                user.Email,
		DisplayName:          user.DisplayName,
		AvatarURL:            user.AvatarURL,
		PreferredQuoteSource: string(domain.ResolveQuoteSource(user.PreferredQuoteSource, domain.QuoteSourceSina)),
		PasswordHash:         user.PasswordHash,
		GoogleSub:            googleSub,
		Provider:             string(user.Provider),
		EmailVerified:        user.EmailVerified,
		LastLoginAt:          user.LastLoginAt,
		CreatedAt:            user.CreatedAt,
		UpdatedAt:            user.UpdatedAt,
	}
}
