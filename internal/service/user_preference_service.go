package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/trading"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

var (
	ErrFundNotFound           = errors.New("fund not found")
	ErrInvalidHoldingOverride = errors.New("invalid holding override")
	ErrWatchlistGroupNotFound = errors.New("watchlist group not found")
	ErrInvalidWatchlistGroup  = errors.New("invalid watchlist group")
	ErrInvalidHoldingAmount   = errors.New("invalid holding amount")
	ErrInvalidHoldingDate     = errors.New("invalid holding date")
	ErrInvalidHoldingTime     = errors.New("invalid holding time")
)

var holdingTradeLocation = trading.TradingLocation()

// UserPreferenceService handles user-owned watchlists and holding overrides.
type UserPreferenceService struct {
	fundRepo        domain.FundRepository
	favoriteRepo    domain.UserFavoriteRepository
	watchlistRepo   domain.UserWatchlistRepository
	fundHoldingRepo domain.UserFundHoldingRepository
	overrideRepo    domain.UserHoldingOverrideRepository
}

// NewUserPreferenceService creates a new UserPreferenceService.
func NewUserPreferenceService(
	fundRepo domain.FundRepository,
	favoriteRepo domain.UserFavoriteRepository,
	watchlistRepo domain.UserWatchlistRepository,
	fundHoldingRepo domain.UserFundHoldingRepository,
	overrideRepo domain.UserHoldingOverrideRepository,
) *UserPreferenceService {
	return &UserPreferenceService{
		fundRepo:        fundRepo,
		favoriteRepo:    favoriteRepo,
		watchlistRepo:   watchlistRepo,
		fundHoldingRepo: fundHoldingRepo,
		overrideRepo:    overrideRepo,
	}
}

// ListFavoriteFunds returns the authenticated user's favorite funds with fund metadata.
func (s *UserPreferenceService) ListFavoriteFunds(ctx context.Context, userID string) ([]domain.UserFavoriteFundDetail, error) {
	favorites, err := s.favoriteRepo.ListFavoriteFunds(ctx, userID)
	if err != nil {
		return nil, err
	}

	fundsByID, err := s.loadFundsByIDs(ctx, collectFavoriteFundIDs(favorites))
	if err != nil {
		return nil, err
	}

	result := make([]domain.UserFavoriteFundDetail, 0, len(favorites))
	for _, favorite := range favorites {
		result = append(result, domain.UserFavoriteFundDetail{
			FundID:    favorite.FundID,
			CreatedAt: favorite.CreatedAt,
			Fund:      fundsByID[favorite.FundID],
		})
	}

	return result, nil
}

// AddFavoriteFund adds a fund to the authenticated user's favorites.
func (s *UserPreferenceService) AddFavoriteFund(ctx context.Context, userID, fundID string) error {
	fundID = strings.TrimSpace(fundID)
	if fundID == "" {
		return ErrFundNotFound
	}

	fund, err := s.fundRepo.GetFundByID(ctx, fundID)
	if err != nil {
		return err
	}
	if fund == nil {
		return ErrFundNotFound
	}

	return s.favoriteRepo.SaveFavoriteFund(ctx, &domain.UserFavoriteFund{
		UserID:    userID,
		FundID:    fundID,
		CreatedAt: time.Now(),
	})
}

// RemoveFavoriteFund removes a fund from the authenticated user's favorites.
func (s *UserPreferenceService) RemoveFavoriteFund(ctx context.Context, userID, fundID string) error {
	return s.favoriteRepo.DeleteFavoriteFund(ctx, userID, strings.TrimSpace(fundID))
}

// ListWatchlistGroups returns grouped watchlists enriched with fund metadata.
func (s *UserPreferenceService) ListWatchlistGroups(ctx context.Context, userID string) ([]domain.UserWatchlistGroupDetail, error) {
	groups, err := s.watchlistRepo.ListWatchlistGroups(ctx, userID)
	if err != nil {
		return nil, err
	}

	groupIDs := collectWatchlistGroupIDs(groups)
	groupFundsByGroupID, err := s.watchlistRepo.ListWatchlistFundsByGroupIDs(ctx, userID, groupIDs)
	if err != nil {
		return nil, err
	}

	fundIDs := make([]string, 0)
	for _, groupFunds := range groupFundsByGroupID {
		for _, item := range groupFunds {
			fundIDs = append(fundIDs, item.FundID)
		}
	}

	fundsByID, err := s.loadFundsByIDs(ctx, fundIDs)
	if err != nil {
		return nil, err
	}

	result := make([]domain.UserWatchlistGroupDetail, 0, len(groups))
	for _, group := range groups {
		groupFunds := groupFundsByGroupID[group.ID]
		funds := make([]domain.UserWatchlistFundDetail, 0, len(groupFunds))
		for _, item := range groupFunds {
			funds = append(funds, domain.UserWatchlistFundDetail{
				FundID:    item.FundID,
				CreatedAt: item.CreatedAt,
				Fund:      fundsByID[item.FundID],
			})
		}

		result = append(result, domain.UserWatchlistGroupDetail{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Accent:      group.Accent,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
			Funds:       funds,
		})
	}

	return result, nil
}

// CreateWatchlistGroup creates a named watchlist bucket for the user.
func (s *UserPreferenceService) CreateWatchlistGroup(ctx context.Context, userID, name, description string) (*domain.UserWatchlistGroup, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" {
		return nil, ErrInvalidWatchlistGroup
	}

	groups, err := s.watchlistRepo.ListWatchlistGroups(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	group := &domain.UserWatchlistGroup{
		ID:          generateID("wlg"),
		UserID:      userID,
		Name:        name,
		Description: description,
		Accent:      pickWatchlistAccent(len(groups)),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.watchlistRepo.SaveWatchlistGroup(ctx, group); err != nil {
		return nil, err
	}
	return group, nil
}

// DeleteWatchlistGroup removes a watchlist group owned by the user.
func (s *UserPreferenceService) DeleteWatchlistGroup(ctx context.Context, userID, groupID string) error {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return ErrWatchlistGroupNotFound
	}

	group, err := s.watchlistRepo.GetWatchlistGroupByID(ctx, userID, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return ErrWatchlistGroupNotFound
	}

	return s.watchlistRepo.DeleteWatchlistGroup(ctx, userID, groupID)
}

// AddWatchlistFund adds a fund to a specific watchlist group.
func (s *UserPreferenceService) AddWatchlistFund(ctx context.Context, userID, groupID, fundID string) error {
	group, err := s.watchlistRepo.GetWatchlistGroupByID(ctx, userID, strings.TrimSpace(groupID))
	if err != nil {
		return err
	}
	if group == nil {
		return ErrWatchlistGroupNotFound
	}

	fundID = strings.TrimSpace(fundID)
	fund, err := s.fundRepo.GetFundByID(ctx, fundID)
	if err != nil {
		return err
	}
	if fund == nil {
		return ErrFundNotFound
	}

	return s.watchlistRepo.SaveWatchlistFund(ctx, &domain.UserWatchlistFund{
		GroupID:   group.ID,
		FundID:    fundID,
		CreatedAt: time.Now(),
	})
}

// RemoveWatchlistFund removes a fund from a specific watchlist group.
func (s *UserPreferenceService) RemoveWatchlistFund(ctx context.Context, userID, groupID, fundID string) error {
	group, err := s.watchlistRepo.GetWatchlistGroupByID(ctx, userID, strings.TrimSpace(groupID))
	if err != nil {
		return err
	}
	if group == nil {
		return ErrWatchlistGroupNotFound
	}

	return s.watchlistRepo.DeleteWatchlistFund(ctx, userID, group.ID, strings.TrimSpace(fundID))
}

// ListFundHoldings returns the user's stored fund position records enriched with fund metadata.
func (s *UserPreferenceService) ListFundHoldings(ctx context.Context, userID string) ([]domain.UserFundHoldingDetail, error) {
	holdings, err := s.fundHoldingRepo.ListFundHoldings(ctx, userID)
	if err != nil {
		return nil, err
	}

	expectedOfficialDate := expectedOfficialHistoryDate(time.Now())
	fundIDs := collectHoldingFundIDs(holdings)
	var (
		fundsByID   map[string]*domain.Fund
		historyByID map[string]*domain.FundHistory
	)

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		loadedFunds, loadErr := s.loadFundsByIDs(groupCtx, fundIDs)
		if loadErr != nil {
			return loadErr
		}
		fundsByID = loadedFunds
		return nil
	})
	group.Go(func() error {
		histories, loadErr := s.fundRepo.GetLatestFundHistoriesByFundIDs(groupCtx, fundIDs)
		if loadErr != nil {
			return loadErr
		}
		historyByID = histories
		return nil
	})
	if err := group.Wait(); err != nil {
		return nil, err
	}

	result := make([]domain.UserFundHoldingDetail, 0, len(holdings))
	for _, holding := range holdings {
		detail := domain.UserFundHoldingDetail{
			ID:        holding.ID,
			FundID:    holding.FundID,
			Amount:    holding.Amount,
			TradeAt:   holding.TradeAt,
			AsOfDate:  holding.AsOfDate,
			Note:      holding.Note,
			CreatedAt: holding.CreatedAt,
			UpdatedAt: holding.UpdatedAt,
			Fund:      fundsByID[holding.FundID],
		}

		history := historyByID[holding.FundID]
		if history != nil && history.Date == expectedOfficialDate {
			detail.ActualDate = history.Date
			detail.ActualNav = history.NetAssetVal.String()
			detail.ActualDailyReturn = history.DailyReturn.String()
		}

		result = append(result, detail)
	}
	return result, nil
}

// CreateFundHolding creates a user fund-level position record.
func (s *UserPreferenceService) CreateFundHolding(ctx context.Context, userID, fundID, amount, tradeAt, note string) (*domain.UserFundHoldingDetail, error) {
	fundID = strings.TrimSpace(fundID)
	if fundID == "" {
		return nil, ErrFundNotFound
	}

	fund, err := s.fundRepo.GetFundByID(ctx, fundID)
	if err != nil {
		return nil, err
	}
	if fund == nil {
		return nil, ErrFundNotFound
	}

	amountDecimal, err := decimal.NewFromString(strings.TrimSpace(amount))
	if err != nil || !amountDecimal.GreaterThan(decimal.Zero) {
		return nil, ErrInvalidHoldingAmount
	}

	tradeAtTime, err := parseHoldingTradeAt(tradeAt)
	if err != nil {
		return nil, err
	}
	pricingDate := resolveHoldingPricingDate(tradeAtTime)

	now := time.Now()
	holding := &domain.UserFundHolding{
		ID:        generateID("ufh"),
		UserID:    userID,
		FundID:    fundID,
		Amount:    amountDecimal,
		TradeAt:   tradeAtTime.In(holdingTradeLocation).Format(time.RFC3339),
		AsOfDate:  pricingDate.Format("2006-01-02"),
		Note:      strings.TrimSpace(note),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.fundHoldingRepo.SaveFundHolding(ctx, holding); err != nil {
		return nil, err
	}

	return &domain.UserFundHoldingDetail{
		ID:        holding.ID,
		FundID:    holding.FundID,
		Amount:    holding.Amount,
		TradeAt:   holding.TradeAt,
		AsOfDate:  holding.AsOfDate,
		Note:      holding.Note,
		CreatedAt: holding.CreatedAt,
		UpdatedAt: holding.UpdatedAt,
		Fund:      fund,
	}, nil
}

// DeleteFundHolding removes a user fund-level position record.
func (s *UserPreferenceService) DeleteFundHolding(ctx context.Context, userID, holdingID string) error {
	if strings.TrimSpace(holdingID) == "" {
		return ErrInvalidHoldingAmount
	}
	return s.fundHoldingRepo.DeleteFundHolding(ctx, userID, strings.TrimSpace(holdingID))
}

// GetHoldingOverrideSet returns all user-managed holdings for a specific fund.
func (s *UserPreferenceService) GetHoldingOverrideSet(ctx context.Context, userID, fundID string) (*domain.UserHoldingOverrideSet, error) {
	fund, err := s.fundRepo.GetFundByID(ctx, fundID)
	if err != nil {
		return nil, err
	}
	if fund == nil {
		return nil, ErrFundNotFound
	}

	overrides, err := s.overrideRepo.ListHoldingOverrides(ctx, userID, fundID)
	if err != nil {
		return nil, err
	}

	return &domain.UserHoldingOverrideSet{
		Fund:      fund,
		Overrides: overrides,
	}, nil
}

// ReplaceHoldingOverrides replaces all user-managed holdings for a specific fund.
func (s *UserPreferenceService) ReplaceHoldingOverrides(ctx context.Context, userID, fundID string, overrides []domain.UserHoldingOverride) error {
	fundID = strings.TrimSpace(fundID)
	fund, err := s.fundRepo.GetFundByID(ctx, fundID)
	if err != nil {
		return err
	}
	if fund == nil {
		return ErrFundNotFound
	}

	cleanedOverrides, err := sanitizeHoldingOverrides(userID, fundID, overrides)
	if err != nil {
		return err
	}

	return s.overrideRepo.ReplaceHoldingOverrides(ctx, userID, fundID, cleanedOverrides)
}

func sanitizeHoldingOverrides(userID, fundID string, overrides []domain.UserHoldingOverride) ([]domain.UserHoldingOverride, error) {
	if len(overrides) == 0 {
		return []domain.UserHoldingOverride{}, nil
	}

	totalRatio := decimal.Zero
	now := time.Now()
	result := make([]domain.UserHoldingOverride, 0, len(overrides))

	for _, override := range overrides {
		stockCode := strings.TrimSpace(override.StockCode)
		stockName := strings.TrimSpace(override.StockName)
		note := strings.TrimSpace(override.Note)
		exchange := override.Exchange
		ratio := override.HoldingRatio

		if stockCode == "" || stockName == "" {
			return nil, ErrInvalidHoldingOverride
		}
		if exchange != domain.ExchangeSH && exchange != domain.ExchangeSZ && exchange != domain.ExchangeBJ {
			return nil, ErrInvalidHoldingOverride
		}
		if !ratio.GreaterThan(decimal.Zero) || ratio.GreaterThan(decimal.NewFromInt(100)) {
			return nil, ErrInvalidHoldingOverride
		}

		totalRatio = totalRatio.Add(ratio)
		if totalRatio.GreaterThan(decimal.NewFromInt(100)) {
			return nil, ErrInvalidHoldingOverride
		}

		overrideID := strings.TrimSpace(override.ID)
		if overrideID == "" {
			overrideID = generateID("uho")
		}

		result = append(result, domain.UserHoldingOverride{
			ID:           overrideID,
			UserID:       userID,
			FundID:       fundID,
			StockCode:    stockCode,
			StockName:    stockName,
			Exchange:     exchange,
			HoldingRatio: ratio,
			Note:         note,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	return result, nil
}

func pickWatchlistAccent(index int) string {
	palette := []string{"cyan", "emerald", "amber", "fuchsia"}
	return palette[index%len(palette)]
}

func formatServiceError(action string, err error) error {
	return fmt.Errorf("%s: %w", action, err)
}

func (s *UserPreferenceService) loadFundsByIDs(ctx context.Context, fundIDs []string) (map[string]*domain.Fund, error) {
	fundsByID, err := s.fundRepo.GetFundsByIDs(ctx, fundIDs)
	if err != nil {
		return nil, err
	}
	if fundsByID == nil {
		return map[string]*domain.Fund{}, nil
	}
	return fundsByID, nil
}

func collectFavoriteFundIDs(favorites []domain.UserFavoriteFund) []string {
	fundIDs := make([]string, 0, len(favorites))
	for _, favorite := range favorites {
		fundIDs = append(fundIDs, favorite.FundID)
	}
	return uniqueFundIDs(fundIDs)
}

func collectWatchlistGroupIDs(groups []domain.UserWatchlistGroup) []string {
	groupIDs := make([]string, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
	}
	return groupIDs
}

func collectHoldingFundIDs(holdings []domain.UserFundHolding) []string {
	fundIDs := make([]string, 0, len(holdings))
	for _, holding := range holdings {
		fundIDs = append(fundIDs, holding.FundID)
	}
	return uniqueFundIDs(fundIDs)
}

func uniqueFundIDs(fundIDs []string) []string {
	seen := make(map[string]struct{}, len(fundIDs))
	result := make([]string, 0, len(fundIDs))
	for _, fundID := range fundIDs {
		fundID = strings.TrimSpace(fundID)
		if fundID == "" {
			continue
		}
		if _, ok := seen[fundID]; ok {
			continue
		}
		seen[fundID] = struct{}{}
		result = append(result, fundID)
	}
	return result
}

func expectedOfficialHistoryDate(now time.Time) string {
	return trading.GetLastTradingDay(now).Format("2006-01-02")
}

func parseHoldingTradeAt(raw string) (time.Time, error) {
	parsed, err := trading.ParseTradeAt(raw)
	switch {
	case errors.Is(err, trading.ErrInvalidTradeTime):
		return time.Time{}, ErrInvalidHoldingTime
	case errors.Is(err, trading.ErrInvalidTradeDate):
		return time.Time{}, ErrInvalidHoldingDate
	case err != nil:
		return time.Time{}, err
	default:
		return parsed.In(holdingTradeLocation), nil
	}
}

func resolveHoldingPricingDate(tradeAt time.Time) time.Time {
	resolution := trading.ResolvePricingDate(tradeAt)
	pricingDate, err := time.ParseInLocation("2006-01-02", resolution.PricingDate, holdingTradeLocation)
	if err != nil {
		return tradeAt.In(holdingTradeLocation)
	}
	return pricingDate
}
