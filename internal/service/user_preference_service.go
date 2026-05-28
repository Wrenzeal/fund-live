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
	ErrFundNotFound                    = errors.New("fund not found")
	ErrInvalidHoldingOverride          = errors.New("invalid holding override")
	ErrWatchlistGroupNotFound          = errors.New("watchlist group not found")
	ErrInvalidWatchlistGroup           = errors.New("invalid watchlist group")
	ErrInvalidWatchlistOrder           = errors.New("invalid watchlist group order")
	ErrInvalidHoldingAmount            = errors.New("invalid holding amount")
	ErrInvalidHoldingDate              = errors.New("invalid holding date")
	ErrInvalidHoldingTime              = errors.New("invalid holding time")
	ErrFundHoldingNotFound             = errors.New("fund holding not found")
	ErrInvalidHoldingSell              = errors.New("invalid holding sell amount")
	ErrFundHoldingTransactionNotFound  = errors.New("fund holding transaction not found")
	ErrFundHoldingTransactionVoided    = errors.New("fund holding transaction already voided")
	ErrInvalidHoldingTransactionFilter = errors.New("invalid holding transaction filter")
	ErrInvalidHoldingSource            = errors.New("invalid holding source")
	ErrInvalidHoldingDividend          = errors.New("invalid holding dividend")
	ErrInvalidHoldingAdjustment        = errors.New("invalid holding adjustment")
	ErrUnsafeHoldingRollback           = errors.New("unsafe holding transaction rollback")
)

var holdingTradeLocation = trading.TradingLocation()

var holdingSourceLabels = map[string]string{
	"alipay":    "支付宝",
	"wechat":    "微信",
	"eastmoney": "天天基金",
	"bank":      "银行 App",
	"manual":    "手工迁移",
	"other":     "其他来源",
}

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
			SortOrder:   group.SortOrder,
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
	minSortOrder := 0
	if len(groups) > 0 {
		minSortOrder = groups[0].SortOrder - 1
	}
	group := &domain.UserWatchlistGroup{
		ID:          generateID("wlg"),
		UserID:      userID,
		Name:        name,
		Description: description,
		Accent:      pickWatchlistAccent(len(groups)),
		SortOrder:   minSortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.watchlistRepo.SaveWatchlistGroup(ctx, group); err != nil {
		return nil, err
	}
	return group, nil
}

// UpdateWatchlistGroup updates the name/description/accent of a watchlist group owned by the user.
func (s *UserPreferenceService) UpdateWatchlistGroup(ctx context.Context, userID, groupID, name, description, accent string) (*domain.UserWatchlistGroup, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, ErrWatchlistGroupNotFound
	}

	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" {
		return nil, ErrInvalidWatchlistGroup
	}
	normalizedAccent := normalizeWatchlistAccent(accent)
	if accent != "" && normalizedAccent == "" {
		return nil, ErrInvalidWatchlistGroup
	}

	group, err := s.watchlistRepo.GetWatchlistGroupByID(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrWatchlistGroupNotFound
	}

	group.Name = name
	group.Description = description
	if normalizedAccent != "" {
		group.Accent = normalizedAccent
	}
	group.UpdatedAt = time.Now()
	if err := s.watchlistRepo.SaveWatchlistGroup(ctx, group); err != nil {
		return nil, err
	}
	return group, nil
}

// ReorderWatchlistGroups persists a full ordered list of group ids for the user.
func (s *UserPreferenceService) ReorderWatchlistGroups(ctx context.Context, userID string, groupIDs []string) error {
	groups, err := s.watchlistRepo.ListWatchlistGroups(ctx, userID)
	if err != nil {
		return err
	}
	if len(groups) != len(groupIDs) {
		return ErrInvalidWatchlistOrder
	}

	groupByID := make(map[string]*domain.UserWatchlistGroup, len(groups))
	for i := range groups {
		groupByID[groups[i].ID] = &groups[i]
	}

	seen := make(map[string]struct{}, len(groupIDs))
	for _, id := range groupIDs {
		id = strings.TrimSpace(id)
		group := groupByID[id]
		if id == "" || group == nil {
			return ErrInvalidWatchlistOrder
		}
		if _, exists := seen[id]; exists {
			return ErrInvalidWatchlistOrder
		}
		seen[id] = struct{}{}
	}

	for index, id := range groupIDs {
		group := groupByID[strings.TrimSpace(id)]
		group.SortOrder = index
		group.UpdatedAt = time.Now()
		if err := s.watchlistRepo.SaveWatchlistGroup(ctx, group); err != nil {
			return err
		}
	}

	return nil
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

// ListFundHoldings returns the user's stored fund position records enriched with fund metadata and summary totals.
func (s *UserPreferenceService) ListFundHoldings(ctx context.Context, userID string) (*domain.UserFundHoldingList, error) {
	holdings, err := s.fundHoldingRepo.ListFundHoldings(ctx, userID)
	if err != nil {
		return nil, err
	}

	expectedOfficialDate := expectedOfficialHistoryDate(time.Now())
	fundIDs := collectHoldingFundIDs(holdings)
	confirmationKeys := collectHoldingHistoryLookupKeys(holdings)
	var (
		fundsByID             map[string]*domain.Fund
		historyByID           map[string]*domain.FundHistory
		confirmationHistoryBy map[domain.FundHistoryLookupKey]*domain.FundHistory
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
	group.Go(func() error {
		histories, loadErr := s.fundRepo.GetFundHistoriesByLookupKeys(groupCtx, confirmationKeys)
		if loadErr != nil {
			return loadErr
		}
		confirmationHistoryBy = histories
		return nil
	})
	if err := group.Wait(); err != nil {
		return nil, err
	}

	items := make([]domain.UserFundHoldingDetail, 0, len(holdings))
	aggregateAccumulators := make(map[string]*holdingAggregateAccumulator, len(holdings))
	aggregateOrder := make([]string, 0, len(holdings))
	totalCurrentMarketValue := decimal.Zero
	totalTodayProfit := decimal.Zero
	totalPreviousMarketValue := decimal.Zero
	readyPrincipal := decimal.Zero
	readyCount := 0

	for _, holding := range holdings {
		resolvedHolding := holding
		if needsHoldingConfirmationData(resolvedHolding) {
			if history := confirmationHistoryBy[holdingHistoryLookupKey(holding)]; history != nil {
				applyHoldingConfirmationData(&resolvedHolding, history)
			}
		}

		detail, metrics := buildUserFundHoldingDetail(
			resolvedHolding,
			fundsByID[holding.FundID],
			historyByID[holding.FundID],
			expectedOfficialDate,
		)
		if metrics != nil {
			readyCount++
			readyPrincipal = readyPrincipal.Add(resolvedHolding.Amount)
			totalCurrentMarketValue = totalCurrentMarketValue.Add(metrics.currentMarketValue)
			totalTodayProfit = totalTodayProfit.Add(metrics.todayProfit)
			totalPreviousMarketValue = totalPreviousMarketValue.Add(metrics.currentMarketValue.Sub(metrics.todayProfit))
		}
		accumulator, exists := aggregateAccumulators[holding.FundID]
		if !exists {
			accumulator = &holdingAggregateAccumulator{
				fundID: holding.FundID,
				fund:   fundsByID[holding.FundID],
			}
			aggregateAccumulators[holding.FundID] = accumulator
			aggregateOrder = append(aggregateOrder, holding.FundID)
		}
		accumulator.add(resolvedHolding, metrics)

		items = append(items, detail)
	}

	return &domain.UserFundHoldingList{
		Items:      items,
		Aggregates: buildUserFundHoldingAggregates(aggregateAccumulators, aggregateOrder),
		Summary: buildUserFundHoldingSummary(
			holdings,
			readyCount,
			readyPrincipal,
			totalCurrentMarketValue,
			totalTodayProfit,
			totalPreviousMarketValue,
		),
	}, nil
}

// CreateFundHolding creates a user fund-level position record.
func (s *UserPreferenceService) CreateFundHolding(ctx context.Context, userID, fundID, amount, tradeAt, note string) (*domain.UserFundHoldingDetail, error) {
	return s.createFundHolding(ctx, userID, domain.CreateFundHoldingInput{
		FundID:  fundID,
		Amount:  amount,
		TradeAt: tradeAt,
		Note:    note,
	})
}

// CreateFundHoldingsBatch safely imports multiple user fund holding rows.
func (s *UserPreferenceService) CreateFundHoldingsBatch(ctx context.Context, userID string, inputs []domain.CreateFundHoldingInput) (*domain.UserFundHoldingBatchCreateResult, error) {
	if len(inputs) == 0 || len(inputs) > 20 {
		return nil, ErrInvalidHoldingAmount
	}

	result := &domain.UserFundHoldingBatchCreateResult{
		Total:   len(inputs),
		Created: make([]domain.UserFundHoldingDetail, 0, len(inputs)),
		Failed:  make([]domain.UserFundHoldingBatchCreateFailure, 0),
	}
	for index, input := range inputs {
		holding, err := s.createFundHolding(ctx, userID, input)
		if err != nil {
			result.Failed = append(result.Failed, domain.UserFundHoldingBatchCreateFailure{
				Index:   index,
				FundID:  strings.TrimSpace(input.FundID),
				Code:    holdingImportErrorCode(err),
				Message: holdingImportErrorMessage(err),
			})
			continue
		}
		result.Created = append(result.Created, *holding)
	}
	result.CreatedCount = len(result.Created)
	result.FailedCount = len(result.Failed)
	return result, nil
}

func (s *UserPreferenceService) createFundHolding(ctx context.Context, userID string, input domain.CreateFundHoldingInput) (*domain.UserFundHoldingDetail, error) {
	fundID := strings.TrimSpace(input.FundID)
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

	amountDecimal, err := parsePositiveDecimal(input.Amount, ErrInvalidHoldingAmount)
	if err != nil {
		return nil, err
	}

	tradeAtTime, err := parseHoldingTradeAt(input.TradeAt)
	if err != nil {
		return nil, err
	}
	pricingDate := resolveHoldingPricingDate(tradeAtTime)
	sourcePlatform, sourceLabel, err := normalizeHoldingSource(input.SourcePlatform, input.SourceLabel)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	holding := &domain.UserFundHolding{
		ID:             generateID("ufh"),
		UserID:         userID,
		FundID:         fundID,
		Amount:         amountDecimal,
		TradeAt:        tradeAtTime.In(holdingTradeLocation).Format(time.RFC3339),
		AsOfDate:       pricingDate.Format("2006-01-02"),
		Note:           strings.TrimSpace(input.Note),
		SourcePlatform: sourcePlatform,
		SourceLabel:    sourceLabel,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if confirmationHistory, err := s.lookupFundHistory(ctx, fundID, holding.AsOfDate); err != nil {
		return nil, err
	} else if confirmationHistory != nil {
		applyHoldingConfirmationData(holding, confirmationHistory)
	}

	if err := s.fundHoldingRepo.SaveFundHolding(ctx, holding); err != nil {
		return nil, err
	}
	if err := s.recordFundHoldingTransaction(ctx, domain.UserFundHoldingTransactionBuy, holding, "记录买入/补仓", map[string]string{
		"source": "create_holding",
	}); err != nil {
		return nil, err
	}

	return buildStoredUserFundHoldingDetail(holding, fund), nil
}

func holdingImportErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrFundNotFound):
		return "FUND_NOT_FOUND"
	case errors.Is(err, ErrInvalidHoldingDate):
		return "INVALID_DATE"
	case errors.Is(err, ErrInvalidHoldingTime):
		return "INVALID_TIME"
	case errors.Is(err, ErrInvalidHoldingSource):
		return "INVALID_SOURCE"
	case errors.Is(err, ErrInvalidHoldingAmount):
		return "INVALID_AMOUNT"
	default:
		return "CREATE_FAILED"
	}
}

func holdingImportErrorMessage(err error) string {
	switch holdingImportErrorCode(err) {
	case "FUND_NOT_FOUND":
		return "基金不存在或基金代码不正确"
	case "INVALID_DATE":
		return "交易日期不是有效交易日"
	case "INVALID_TIME":
		return "交易时间格式不正确"
	case "INVALID_SOURCE":
		return "平台来源不支持"
	case "INVALID_AMOUNT":
		return "金额需要大于 0"
	default:
		return "导入失败，请逐笔核对"
	}
}

// UpdateFundHolding corrects user-editable values for an existing fund-level position record.
func (s *UserPreferenceService) UpdateFundHolding(ctx context.Context, userID, holdingID string, input domain.UpdateFundHoldingInput) (*domain.UserFundHoldingDetail, error) {
	holdingID = strings.TrimSpace(holdingID)
	if holdingID == "" {
		return nil, ErrFundHoldingNotFound
	}

	holding, err := s.fundHoldingRepo.GetFundHolding(ctx, userID, holdingID)
	if err != nil {
		return nil, err
	}
	if holding == nil {
		return nil, ErrFundHoldingNotFound
	}

	fund, err := s.fundRepo.GetFundByID(ctx, holding.FundID)
	if err != nil {
		return nil, err
	}
	if fund == nil {
		return nil, ErrFundNotFound
	}

	amountDecimal, err := parsePositiveDecimal(input.Amount, ErrInvalidHoldingAmount)
	if err != nil {
		return nil, err
	}

	updated := *holding
	updated.Amount = amountDecimal
	updated.Note = strings.TrimSpace(input.Note)
	sourcePlatform, sourceLabel, err := normalizeHoldingSource(input.SourcePlatform, input.SourceLabel)
	if err != nil {
		return nil, err
	}
	updated.SourcePlatform = sourcePlatform
	updated.SourceLabel = sourceLabel

	if strings.TrimSpace(input.TradeAt) != "" {
		tradeAtTime, err := parseHoldingTradeAt(input.TradeAt)
		if err != nil {
			return nil, err
		}
		updated.TradeAt = tradeAtTime.In(holdingTradeLocation).Format(time.RFC3339)
		updated.AsOfDate = resolveHoldingPricingDate(tradeAtTime).Format("2006-01-02")
	}

	manualShares, sharesProvided, err := parseOptionalPositiveDecimal(input.Shares, ErrInvalidHoldingAmount)
	if err != nil {
		return nil, err
	}
	manualNav, navProvided, err := parseOptionalPositiveDecimal(input.ConfirmedNav, ErrInvalidHoldingAmount)
	if err != nil {
		return nil, err
	}
	confirmedNavDate := strings.TrimSpace(input.ConfirmedNavDate)
	if confirmedNavDate != "" {
		if _, err := time.ParseInLocation("2006-01-02", confirmedNavDate, holdingTradeLocation); err != nil {
			return nil, ErrInvalidHoldingDate
		}
	}

	hasManualConfirmation := sharesProvided || navProvided || confirmedNavDate != ""
	if hasManualConfirmation {
		if !sharesProvided || !navProvided || confirmedNavDate == "" {
			return nil, ErrInvalidHoldingAmount
		}
		updated.Shares = manualShares
		updated.ConfirmedNav = manualNav
		updated.ConfirmedNavDate = confirmedNavDate
		updated.AsOfDate = confirmedNavDate
		updated.ManualConfirmation = true
	} else {
		updated.Shares = decimal.Zero
		updated.ConfirmedNav = decimal.Zero
		updated.ConfirmedNavDate = ""
		updated.ManualConfirmation = false
		if confirmationHistory, err := s.lookupFundHistory(ctx, updated.FundID, updated.AsOfDate); err != nil {
			return nil, err
		} else if confirmationHistory != nil {
			applyHoldingConfirmationData(&updated, confirmationHistory)
		}
	}

	updated.UpdatedAt = time.Now()
	if updated.CreatedAt.IsZero() {
		updated.CreatedAt = updated.UpdatedAt
	}

	if err := s.fundHoldingRepo.SaveFundHolding(ctx, &updated); err != nil {
		return nil, err
	}
	if err := s.recordFundHoldingTransaction(ctx, domain.UserFundHoldingTransactionCorrection, &updated, "校正持仓数据", buildHoldingCorrectionMetadata(holding, &updated)); err != nil {
		return nil, err
	}

	return buildStoredUserFundHoldingDetail(&updated, fund), nil
}

func parsePositiveDecimal(raw string, invalidErr error) (decimal.Decimal, error) {
	value, err := decimal.NewFromString(strings.ReplaceAll(strings.TrimSpace(raw), ",", ""))
	if err != nil || !value.GreaterThan(decimal.Zero) {
		return decimal.Zero, invalidErr
	}
	return value, nil
}

func parseOptionalPositiveDecimal(raw string, invalidErr error) (decimal.Decimal, bool, error) {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
	if raw == "" {
		return decimal.Zero, false, nil
	}
	value, err := decimal.NewFromString(raw)
	if err != nil || !value.GreaterThan(decimal.Zero) {
		return decimal.Zero, false, invalidErr
	}
	return value, true, nil
}

func parseOptionalNonZeroDecimal(raw string, invalidErr error) (decimal.Decimal, bool, error) {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
	if raw == "" {
		return decimal.Zero, false, nil
	}
	value, err := decimal.NewFromString(raw)
	if err != nil || value.Equal(decimal.Zero) {
		return decimal.Zero, false, invalidErr
	}
	return value, true, nil
}

// SellFundHolding records a redemption/decrease and updates the current holding snapshot.
func (s *UserPreferenceService) SellFundHolding(ctx context.Context, userID, holdingID string, input domain.SellFundHoldingInput) (*domain.UserFundHoldingDetail, error) {
	holdingID = strings.TrimSpace(holdingID)
	if holdingID == "" {
		return nil, ErrFundHoldingNotFound
	}

	holding, err := s.fundHoldingRepo.GetFundHolding(ctx, userID, holdingID)
	if err != nil {
		return nil, err
	}
	if holding == nil {
		return nil, ErrFundHoldingNotFound
	}

	fund, err := s.fundRepo.GetFundByID(ctx, holding.FundID)
	if err != nil {
		return nil, err
	}
	if fund == nil {
		return nil, ErrFundNotFound
	}

	sellAmount, amountProvided, err := parseOptionalPositiveDecimal(input.Amount, ErrInvalidHoldingSell)
	if err != nil {
		return nil, err
	}
	sellShares, sharesProvided, err := parseOptionalPositiveDecimal(input.Shares, ErrInvalidHoldingSell)
	if err != nil {
		return nil, err
	}
	if input.SellAll {
		sellAmount = holding.Amount
		amountProvided = true
		if holding.Shares.GreaterThan(decimal.Zero) {
			sellShares = holding.Shares
			sharesProvided = true
		}
	}
	if !amountProvided && !sharesProvided {
		return nil, ErrInvalidHoldingSell
	}
	if amountProvided && sellAmount.GreaterThan(holding.Amount) {
		return nil, ErrInvalidHoldingSell
	}
	if sharesProvided {
		if !holding.Shares.GreaterThan(decimal.Zero) || sellShares.GreaterThan(holding.Shares) {
			return nil, ErrInvalidHoldingSell
		}
	}

	tradeAt := strings.TrimSpace(input.TradeAt)
	tradeAtTime := time.Now().In(holdingTradeLocation)
	if tradeAt != "" {
		tradeAtTime, err = parseHoldingTradeAt(tradeAt)
		if err != nil {
			return nil, err
		}
	}

	remaining := *holding
	remaining.Note = strings.TrimSpace(input.Note)
	remaining.UpdatedAt = time.Now()
	if remaining.CreatedAt.IsZero() {
		remaining.CreatedAt = remaining.UpdatedAt
	}

	if amountProvided {
		remaining.Amount = holding.Amount.Sub(sellAmount)
	} else if sharesProvided && holding.Shares.GreaterThan(decimal.Zero) {
		ratio := sellShares.DivRound(holding.Shares, 8)
		sellAmount = holding.Amount.Mul(ratio).Round(2)
		remaining.Amount = holding.Amount.Sub(sellAmount)
	}

	if sharesProvided {
		remaining.Shares = holding.Shares.Sub(sellShares)
	} else if amountProvided && holding.Amount.GreaterThan(decimal.Zero) && holding.Shares.GreaterThan(decimal.Zero) {
		ratio := sellAmount.DivRound(holding.Amount, 8)
		sellShares = holding.Shares.Mul(ratio).Round(6)
		remaining.Shares = holding.Shares.Sub(sellShares)
	}

	sellSnapshot := *holding
	sellSnapshot.Amount = sellAmount
	sellSnapshot.Shares = sellShares
	sellSnapshot.TradeAt = tradeAtTime.In(holdingTradeLocation).Format(time.RFC3339)
	sellSnapshot.AsOfDate = resolveHoldingPricingDate(tradeAtTime).Format("2006-01-02")
	sellSnapshot.Note = strings.TrimSpace(input.Note)

	if input.SellAll {
		remaining.Amount = decimal.Zero
		remaining.Shares = decimal.Zero
		metadata := buildHoldingSellMetadata(holding, &remaining)
		metadata["sell_all"] = "true"
		if err := s.recordFundHoldingTransaction(ctx, domain.UserFundHoldingTransactionSell, &sellSnapshot, "记录卖出/清仓", metadata); err != nil {
			return nil, err
		}
		if err := s.fundHoldingRepo.DeleteFundHolding(ctx, userID, holdingID); err != nil {
			return nil, err
		}
		return nil, nil
	}

	if remaining.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidHoldingSell
	}
	if holding.Shares.GreaterThan(decimal.Zero) && remaining.Shares.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidHoldingSell
	}
	if remaining.Shares.LessThan(decimal.Zero) {
		return nil, ErrInvalidHoldingSell
	}

	if err := s.fundHoldingRepo.SaveFundHolding(ctx, &remaining); err != nil {
		return nil, err
	}

	if err := s.recordFundHoldingTransaction(ctx, domain.UserFundHoldingTransactionSell, &sellSnapshot, "记录卖出/减仓", buildHoldingSellMetadata(holding, &remaining)); err != nil {
		return nil, err
	}

	return buildStoredUserFundHoldingDetail(&remaining, fund), nil
}

// RecordFundHoldingDividend records a cash dividend or dividend reinvestment event.
func (s *UserPreferenceService) RecordFundHoldingDividend(ctx context.Context, userID, holdingID string, input domain.DividendFundHoldingInput) (*domain.UserFundHoldingDetail, error) {
	holdingID = strings.TrimSpace(holdingID)
	if holdingID == "" {
		return nil, ErrFundHoldingNotFound
	}

	holding, err := s.fundHoldingRepo.GetFundHolding(ctx, userID, holdingID)
	if err != nil {
		return nil, err
	}
	if holding == nil {
		return nil, ErrFundHoldingNotFound
	}

	fund, err := s.fundRepo.GetFundByID(ctx, holding.FundID)
	if err != nil {
		return nil, err
	}
	if fund == nil {
		return nil, ErrFundNotFound
	}

	dividendAmount, err := parsePositiveDecimal(input.Amount, ErrInvalidHoldingDividend)
	if err != nil {
		return nil, err
	}
	dividendShares, sharesProvided, err := parseOptionalPositiveDecimal(input.Shares, ErrInvalidHoldingDividend)
	if err != nil {
		return nil, err
	}
	if input.Reinvest && !sharesProvided {
		return nil, ErrInvalidHoldingDividend
	}

	tradeAtTime, err := parseOptionalHoldingTradeAt(input.TradeAt)
	if err != nil {
		return nil, err
	}
	asOfDate := resolveHoldingPricingDate(tradeAtTime).Format("2006-01-02")
	sourcePlatform, sourceLabel, err := normalizeHoldingSource(input.SourcePlatform, input.SourceLabel)
	if err != nil {
		return nil, err
	}

	dividendSnapshot := *holding
	dividendSnapshot.Amount = dividendAmount
	dividendSnapshot.Shares = dividendShares
	dividendSnapshot.TradeAt = tradeAtTime.In(holdingTradeLocation).Format(time.RFC3339)
	dividendSnapshot.AsOfDate = asOfDate
	dividendSnapshot.Note = strings.TrimSpace(input.Note)
	dividendSnapshot.SourcePlatform = sourcePlatform
	dividendSnapshot.SourceLabel = sourceLabel
	if input.Reinvest && dividendShares.GreaterThan(decimal.Zero) {
		dividendSnapshot.ConfirmedNav = dividendAmount.DivRound(dividendShares, 6)
		dividendSnapshot.ConfirmedNavDate = asOfDate
	}

	if !input.Reinvest {
		if err := s.recordFundHoldingTransaction(ctx, domain.UserFundHoldingTransactionDividend, &dividendSnapshot, "记录现金分红", buildHoldingDividendMetadata(holding, holding, input.Reinvest)); err != nil {
			return nil, err
		}
		return buildStoredUserFundHoldingDetail(holding, fund), nil
	}

	updated := *holding
	updated.Shares = holding.Shares.Add(dividendShares)
	updated.Note = strings.TrimSpace(input.Note)
	updated.TradeAt = tradeAtTime.In(holdingTradeLocation).Format(time.RFC3339)
	updated.AsOfDate = asOfDate
	updated.ManualConfirmation = true
	updated.SourcePlatform = sourcePlatform
	updated.SourceLabel = sourceLabel
	if dividendSnapshot.ConfirmedNav.GreaterThan(decimal.Zero) {
		updated.ConfirmedNav = dividendSnapshot.ConfirmedNav
		updated.ConfirmedNavDate = dividendSnapshot.ConfirmedNavDate
	}
	updated.UpdatedAt = time.Now()
	if updated.CreatedAt.IsZero() {
		updated.CreatedAt = updated.UpdatedAt
	}

	if err := s.fundHoldingRepo.SaveFundHolding(ctx, &updated); err != nil {
		return nil, err
	}
	if err := s.recordFundHoldingTransaction(ctx, domain.UserFundHoldingTransactionDividend, &dividendSnapshot, "记录红利再投", buildHoldingDividendMetadata(holding, &updated, input.Reinvest)); err != nil {
		return nil, err
	}

	return buildStoredUserFundHoldingDetail(&updated, fund), nil
}

// AdjustFundHoldingShares records a non-trade share adjustment and updates the current snapshot.
func (s *UserPreferenceService) AdjustFundHoldingShares(ctx context.Context, userID, holdingID string, input domain.AdjustFundHoldingSharesInput) (*domain.UserFundHoldingDetail, error) {
	holdingID = strings.TrimSpace(holdingID)
	if holdingID == "" {
		return nil, ErrFundHoldingNotFound
	}

	holding, err := s.fundHoldingRepo.GetFundHolding(ctx, userID, holdingID)
	if err != nil {
		return nil, err
	}
	if holding == nil {
		return nil, ErrFundHoldingNotFound
	}

	fund, err := s.fundRepo.GetFundByID(ctx, holding.FundID)
	if err != nil {
		return nil, err
	}
	if fund == nil {
		return nil, ErrFundNotFound
	}

	targetShares, targetProvided, err := parseOptionalPositiveDecimal(input.TargetShares, ErrInvalidHoldingAdjustment)
	if err != nil {
		return nil, err
	}
	sharesDelta, deltaProvided, err := parseOptionalNonZeroDecimal(input.SharesDelta, ErrInvalidHoldingAdjustment)
	if err != nil {
		return nil, err
	}
	if targetProvided == deltaProvided {
		return nil, ErrInvalidHoldingAdjustment
	}
	if deltaProvided {
		targetShares = holding.Shares.Add(sharesDelta)
		if !targetShares.GreaterThan(decimal.Zero) {
			return nil, ErrInvalidHoldingAdjustment
		}
	} else {
		sharesDelta = targetShares.Sub(holding.Shares)
		if sharesDelta.Equal(decimal.Zero) {
			return nil, ErrInvalidHoldingAdjustment
		}
	}

	confirmedNav, navProvided, err := parseOptionalPositiveDecimal(input.ConfirmedNav, ErrInvalidHoldingAdjustment)
	if err != nil {
		return nil, err
	}
	confirmedNavDate := strings.TrimSpace(input.ConfirmedNavDate)
	if navProvided != (confirmedNavDate != "") {
		return nil, ErrInvalidHoldingAdjustment
	}
	if confirmedNavDate != "" {
		if _, err := time.ParseInLocation("2006-01-02", confirmedNavDate, holdingTradeLocation); err != nil {
			return nil, ErrInvalidHoldingDate
		}
	}

	tradeAtTime, err := parseOptionalHoldingTradeAt(input.TradeAt)
	if err != nil {
		return nil, err
	}
	sourcePlatform, sourceLabel, err := normalizeHoldingSource(input.SourcePlatform, input.SourceLabel)
	if err != nil {
		return nil, err
	}

	updated := *holding
	updated.Shares = targetShares
	updated.Note = strings.TrimSpace(input.Note)
	updated.TradeAt = tradeAtTime.In(holdingTradeLocation).Format(time.RFC3339)
	updated.AsOfDate = resolveHoldingPricingDate(tradeAtTime).Format("2006-01-02")
	updated.ManualConfirmation = true
	updated.SourcePlatform = sourcePlatform
	updated.SourceLabel = sourceLabel
	if navProvided {
		updated.ConfirmedNav = confirmedNav
		updated.ConfirmedNavDate = confirmedNavDate
		updated.AsOfDate = confirmedNavDate
	}
	updated.UpdatedAt = time.Now()
	if updated.CreatedAt.IsZero() {
		updated.CreatedAt = updated.UpdatedAt
	}

	adjustmentSnapshot := updated
	if err := s.fundHoldingRepo.SaveFundHolding(ctx, &updated); err != nil {
		return nil, err
	}
	if err := s.recordFundHoldingTransaction(ctx, domain.UserFundHoldingTransactionAdjustment, &adjustmentSnapshot, "调整持仓份额", buildHoldingAdjustmentMetadata(holding, &updated, sharesDelta)); err != nil {
		return nil, err
	}

	return buildStoredUserFundHoldingDetail(&updated, fund), nil
}

func buildStoredUserFundHoldingDetail(holding *domain.UserFundHolding, fund *domain.Fund) *domain.UserFundHoldingDetail {
	if holding == nil {
		return nil
	}
	return &domain.UserFundHoldingDetail{
		ID:                 holding.ID,
		FundID:             holding.FundID,
		Amount:             holding.Amount,
		Shares:             decimalStringOrEmpty(holding.Shares),
		ConfirmedNav:       decimalStringOrEmpty(holding.ConfirmedNav),
		ConfirmedNavDate:   holding.ConfirmedNavDate,
		ManualConfirmation: holding.ManualConfirmation,
		TradeAt:            holding.TradeAt,
		AsOfDate:           holding.AsOfDate,
		Note:               holding.Note,
		SourcePlatform:     holding.SourcePlatform,
		SourceLabel:        holding.SourceLabel,
		CreatedAt:          holding.CreatedAt,
		UpdatedAt:          holding.UpdatedAt,
		Fund:               fund,
	}
}

// ListFundHoldingTransactions returns recent holding activity enriched with fund metadata.
func (s *UserPreferenceService) ListFundHoldingTransactions(ctx context.Context, userID string, limit int) ([]domain.UserFundHoldingTransaction, error) {
	return s.ListFundHoldingTransactionsFiltered(ctx, userID, domain.UserFundHoldingTransactionFilter{Limit: limit})
}

// ListFundHoldingTransactionsFiltered returns recent holding activity matching the requested filters.
func (s *UserPreferenceService) ListFundHoldingTransactionsFiltered(ctx context.Context, userID string, filter domain.UserFundHoldingTransactionFilter) ([]domain.UserFundHoldingTransaction, error) {
	filter, err := sanitizeHoldingTransactionFilter(filter)
	if err != nil {
		return nil, err
	}

	transactions, err := s.fundHoldingRepo.ListFundHoldingTransactionsFiltered(ctx, userID, filter)
	if err != nil {
		return nil, err
	}

	fundsByID, err := s.loadFundsByIDs(ctx, collectHoldingTransactionFundIDs(transactions))
	if err != nil {
		return nil, err
	}

	result := make([]domain.UserFundHoldingTransaction, 0, len(transactions))
	for _, transaction := range transactions {
		transaction.Fund = fundsByID[transaction.FundID]
		result = append(result, transaction)
	}
	return result, nil
}

// GetFundHoldingTransactionDetail returns one historical activity with related context.
func (s *UserPreferenceService) GetFundHoldingTransactionDetail(ctx context.Context, userID, transactionID string) (*domain.UserFundHoldingTransactionDetail, error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return nil, ErrFundHoldingTransactionNotFound
	}

	preview, err := s.PreviewFundHoldingTransactionRollback(ctx, userID, transactionID)
	if err != nil {
		return nil, err
	}
	if preview == nil {
		return nil, ErrFundHoldingTransactionNotFound
	}

	related, err := s.ListFundHoldingTransactionsFiltered(ctx, userID, domain.UserFundHoldingTransactionFilter{
		FundID: preview.Transaction.FundID,
		Limit:  maxHoldingTransactionContextLimit,
	})
	if err != nil {
		return nil, err
	}

	relatedTransactions := make([]domain.UserFundHoldingTransaction, 0, 8)
	subsequentTransactions := make([]domain.UserFundHoldingTransaction, 0, 6)
	for _, candidate := range related {
		if candidate.ID == preview.Transaction.ID {
			continue
		}
		sameHolding := strings.TrimSpace(preview.Transaction.HoldingID) != "" && candidate.HoldingID == preview.Transaction.HoldingID
		if sameHolding || len(relatedTransactions) < 8 {
			relatedTransactions = append(relatedTransactions, candidate)
		}
		if candidate.CreatedAt.After(preview.Transaction.CreatedAt) && len(subsequentTransactions) < 6 {
			subsequentTransactions = append(subsequentTransactions, candidate)
		}
		if len(relatedTransactions) >= 8 && len(subsequentTransactions) >= 6 {
			break
		}
	}
	if len(relatedTransactions) > 8 {
		relatedTransactions = relatedTransactions[:8]
	}

	return &domain.UserFundHoldingTransactionDetail{
		Transaction:            preview.Transaction,
		CurrentHolding:         preview.CurrentHolding,
		RollbackPreview:        preview,
		RelatedTransactions:    relatedTransactions,
		SubsequentTransactions: subsequentTransactions,
		ImpactChain:            buildHoldingTransactionImpactChain(preview.Transaction, preview.CurrentHolding, subsequentTransactions),
	}, nil
}

const maxHoldingTransactionContextLimit = 50

func buildHoldingTransactionImpactChain(
	transaction domain.UserFundHoldingTransaction,
	currentHolding *domain.UserFundHoldingDetail,
	subsequentTransactions []domain.UserFundHoldingTransaction,
) []string {
	chain := []string{
		fmt.Sprintf("%s 记录为“%s”，金额 %s 元。", formatHoldingTransactionDate(transaction.CreatedAt), holdingTransactionTypeLabel(transaction.Type), transaction.Amount.StringFixedBank(2)),
	}
	if transaction.Voided {
		chain = append(chain, "这条流水已作废：历史痕迹保留，但不建议继续作为当前资产判断依据。")
	}
	if len(subsequentTransactions) > 0 {
		chain = append(chain, fmt.Sprintf("这条流水之后还有 %d 条同基金流水，回看时需要一起核对后续买卖、分红或校正。", len(subsequentTransactions)))
	}
	if currentHolding != nil {
		chain = append(chain, fmt.Sprintf("当前快照仍存在：本金 %s 元，确认份额 %s，确认净值日 %s。", currentHolding.Amount.StringFixedBank(2), emptyAsDash(currentHolding.Shares), emptyAsDash(currentHolding.ConfirmedNavDate)))
	} else {
		chain = append(chain, "当前快照已不存在，可能已经清仓或删除；恢复时建议按外部平台重新记录或校正。")
	}
	chain = append(chain, "当前详情页只做追溯与人工对账提示，不会自动改写持仓快照。")
	return chain
}

func holdingTransactionTypeLabel(txType domain.UserFundHoldingTransactionType) string {
	switch txType {
	case domain.UserFundHoldingTransactionBuy:
		return "买入/补仓"
	case domain.UserFundHoldingTransactionCorrection:
		return "校正"
	case domain.UserFundHoldingTransactionDelete:
		return "删除"
	case domain.UserFundHoldingTransactionSell:
		return "卖出/清仓"
	case domain.UserFundHoldingTransactionDividend:
		return "分红"
	case domain.UserFundHoldingTransactionAdjustment:
		return "份额调整"
	default:
		return string(txType)
	}
}

func formatHoldingTransactionDate(value time.Time) string {
	if value.IsZero() {
		return "未知时间"
	}
	return value.In(holdingTradeLocation).Format("2006-01-02 15:04")
}

func emptyAsDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "--"
	}
	return value
}

func sanitizeHoldingTransactionFilter(filter domain.UserFundHoldingTransactionFilter) (domain.UserFundHoldingTransactionFilter, error) {
	filter.FundID = strings.TrimSpace(filter.FundID)
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	if filter.Offset < 0 {
		return domain.UserFundHoldingTransactionFilter{}, ErrInvalidHoldingTransactionFilter
	}
	if len([]rune(filter.Keyword)) > 80 {
		return domain.UserFundHoldingTransactionFilter{}, ErrInvalidHoldingTransactionFilter
	}
	if filter.CreatedFrom != nil && filter.CreatedBefore != nil && filter.CreatedFrom.After(*filter.CreatedBefore) {
		return domain.UserFundHoldingTransactionFilter{}, ErrInvalidHoldingTransactionFilter
	}
	filter.SourcePlatform = strings.TrimSpace(filter.SourcePlatform)
	if filter.SourcePlatform == "all" {
		filter.SourcePlatform = ""
	}
	if filter.SourcePlatform != "" {
		platform, _, err := normalizeHoldingSource(filter.SourcePlatform, "")
		if err != nil {
			return domain.UserFundHoldingTransactionFilter{}, ErrInvalidHoldingTransactionFilter
		}
		filter.SourcePlatform = platform
	}

	if len(filter.Types) == 0 {
		return filter, nil
	}

	seen := make(map[domain.UserFundHoldingTransactionType]struct{}, len(filter.Types))
	types := make([]domain.UserFundHoldingTransactionType, 0, len(filter.Types))
	for _, rawType := range filter.Types {
		txType := domain.UserFundHoldingTransactionType(strings.TrimSpace(string(rawType)))
		if txType == "" {
			continue
		}
		if !isValidHoldingTransactionType(txType) {
			return domain.UserFundHoldingTransactionFilter{}, ErrInvalidHoldingTransactionFilter
		}
		if _, exists := seen[txType]; exists {
			continue
		}
		seen[txType] = struct{}{}
		types = append(types, txType)
	}
	filter.Types = types
	return filter, nil
}

func isValidHoldingTransactionType(txType domain.UserFundHoldingTransactionType) bool {
	switch txType {
	case domain.UserFundHoldingTransactionBuy,
		domain.UserFundHoldingTransactionCorrection,
		domain.UserFundHoldingTransactionDelete,
		domain.UserFundHoldingTransactionSell,
		domain.UserFundHoldingTransactionDividend,
		domain.UserFundHoldingTransactionAdjustment:
		return true
	default:
		return false
	}
}

func normalizeHoldingSource(platform, label string) (string, string, error) {
	platform = normalizeHoldingSourcePlatform(platform)
	label = strings.TrimSpace(label)
	if platform == "" {
		if label != "" {
			return "", "", ErrInvalidHoldingSource
		}
		return "", "", nil
	}

	defaultLabel, ok := holdingSourceLabels[platform]
	if !ok {
		return "", "", ErrInvalidHoldingSource
	}
	if label == "" {
		label = defaultLabel
	}
	if len([]rune(label)) > 64 {
		return "", "", ErrInvalidHoldingSource
	}
	return platform, label, nil
}

func normalizeHoldingSourcePlatform(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "", "all":
		return ""
	case "alipay", "ali", "支付宝":
		return "alipay"
	case "wechat", "weixin", "微信":
		return "wechat"
	case "eastmoney", "tiantian", "天天基金":
		return "eastmoney"
	case "bank", "银行", "银行app", "银行 app":
		return "bank"
	case "manual", "migration", "手工", "手工迁移":
		return "manual"
	case "other", "其他":
		return "other"
	default:
		return strings.ToLower(strings.TrimSpace(platform))
	}
}

// VoidFundHoldingTransaction marks a historical holding activity as ignored without changing current holdings.
func (s *UserPreferenceService) VoidFundHoldingTransaction(ctx context.Context, userID, transactionID, reason string) (*domain.UserFundHoldingTransaction, error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return nil, ErrFundHoldingTransactionNotFound
	}

	transaction, err := s.fundHoldingRepo.GetFundHoldingTransaction(ctx, userID, transactionID)
	if err != nil {
		return nil, err
	}
	if transaction == nil {
		return nil, ErrFundHoldingTransactionNotFound
	}
	if transaction.Voided {
		return nil, ErrFundHoldingTransactionVoided
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "用户标记该流水无效"
	}
	voidedAt := time.Now()
	voided, err := s.fundHoldingRepo.VoidFundHoldingTransaction(ctx, userID, transactionID, reason, voidedAt)
	if err != nil {
		return nil, err
	}
	if voided == nil {
		return nil, ErrFundHoldingTransactionNotFound
	}

	fundsByID, err := s.loadFundsByIDs(ctx, []string{voided.FundID})
	if err != nil {
		return nil, err
	}
	voided.Fund = fundsByID[voided.FundID]
	return voided, nil
}

// PreviewFundHoldingTransactionRollback returns a read-only impact preview for a historical activity.
func (s *UserPreferenceService) PreviewFundHoldingTransactionRollback(ctx context.Context, userID, transactionID string) (*domain.UserFundHoldingTransactionRollbackPreview, error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return nil, ErrFundHoldingTransactionNotFound
	}

	transaction, err := s.fundHoldingRepo.GetFundHoldingTransaction(ctx, userID, transactionID)
	if err != nil {
		return nil, err
	}
	if transaction == nil {
		return nil, ErrFundHoldingTransactionNotFound
	}

	fundsByID, err := s.loadFundsByIDs(ctx, []string{transaction.FundID})
	if err != nil {
		return nil, err
	}
	transaction.Fund = fundsByID[transaction.FundID]

	var currentHolding *domain.UserFundHoldingDetail
	var currentSnapshot *domain.UserFundHolding
	if strings.TrimSpace(transaction.HoldingID) != "" {
		holding, err := s.fundHoldingRepo.GetFundHolding(ctx, userID, transaction.HoldingID)
		if err != nil {
			return nil, err
		}
		if holding != nil {
			currentSnapshot = holding
			currentHolding = buildStoredUserFundHoldingDetail(holding, fundsByID[holding.FundID])
		}
	}

	preview := buildHoldingTransactionRollbackPreview(*transaction, currentSnapshot, currentHolding)
	hasSubsequent, err := s.hasSubsequentActiveHoldingTransaction(ctx, userID, *transaction)
	if err != nil {
		return nil, err
	}
	preview.CanApplyAutomatically = canApplyHoldingRollbackAutomatically(*transaction, currentSnapshot, hasSubsequent)
	if preview.CanApplyAutomatically {
		preview.SuggestedAction = "确认这条流水录错后，可直接应用自动冲正：系统会先作废原流水，再按预览安全更新当前持仓快照，并追加一条冲正痕迹。"
		preview.Warnings = append(preview.Warnings, "自动冲正只在没有后续有效流水、且当前快照可安全计算时开放。")
	} else if hasSubsequent {
		preview.Warnings = append(preview.Warnings, "这条流水之后还有有效流水，自动冲正可能影响后续链路，因此当前只提供人工校正建议。")
	}
	return &preview, nil
}

// ApplyFundHoldingTransactionRollback applies a user-confirmed rollback for low-risk activity types.
func (s *UserPreferenceService) ApplyFundHoldingTransactionRollback(ctx context.Context, userID, transactionID, reason string) (*domain.UserFundHoldingTransactionRollbackApplyResult, error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return nil, ErrFundHoldingTransactionNotFound
	}

	transaction, err := s.fundHoldingRepo.GetFundHoldingTransaction(ctx, userID, transactionID)
	if err != nil {
		return nil, err
	}
	if transaction == nil {
		return nil, ErrFundHoldingTransactionNotFound
	}
	if transaction.Voided {
		return nil, ErrFundHoldingTransactionVoided
	}

	fundsByID, err := s.loadFundsByIDs(ctx, []string{transaction.FundID})
	if err != nil {
		return nil, err
	}
	transaction.Fund = fundsByID[transaction.FundID]

	var currentSnapshot *domain.UserFundHolding
	if strings.TrimSpace(transaction.HoldingID) != "" {
		currentSnapshot, err = s.fundHoldingRepo.GetFundHolding(ctx, userID, transaction.HoldingID)
		if err != nil {
			return nil, err
		}
	}

	currentHolding := buildStoredUserFundHoldingDetail(currentSnapshot, fundsByID[transaction.FundID])
	preview := buildHoldingTransactionRollbackPreview(*transaction, currentSnapshot, currentHolding)
	hasSubsequent, err := s.hasSubsequentActiveHoldingTransaction(ctx, userID, *transaction)
	if err != nil {
		return nil, err
	}
	preview.CanApplyAutomatically = canApplyHoldingRollbackAutomatically(*transaction, currentSnapshot, hasSubsequent)
	if !preview.CanApplyAutomatically {
		return nil, ErrUnsafeHoldingRollback
	}

	updatedHolding, holdingRemoved, holdingRestored, err := s.applyHoldingRollbackSnapshot(ctx, *transaction, currentSnapshot)
	if err != nil {
		return nil, err
	}

	if reason = strings.TrimSpace(reason); reason == "" {
		reason = "用户确认自动冲正"
	}
	voided, err := s.fundHoldingRepo.VoidFundHoldingTransaction(ctx, userID, transactionID, "自动冲正："+reason, time.Now())
	if err != nil {
		return nil, err
	}
	if voided == nil {
		return nil, ErrFundHoldingTransactionNotFound
	}
	voided.Fund = fundsByID[voided.FundID]

	var detail *domain.UserFundHoldingDetail
	if updatedHolding != nil {
		detail = buildStoredUserFundHoldingDetail(updatedHolding, fundsByID[updatedHolding.FundID])
	}
	preview.Transaction = *voided
	preview.CurrentHolding = detail
	preview.State = "applied"

	return &domain.UserFundHoldingTransactionRollbackApplyResult{
		Transaction:     *voided,
		CurrentHolding:  detail,
		Preview:         preview,
		Applied:         true,
		HoldingRemoved:  holdingRemoved,
		HoldingRestored: holdingRestored,
		Message:         "自动冲正已完成：原流水已作废，当前持仓快照已按安全规则更新。",
	}, nil
}

func (s *UserPreferenceService) hasSubsequentActiveHoldingTransaction(ctx context.Context, userID string, transaction domain.UserFundHoldingTransaction) (bool, error) {
	related, err := s.fundHoldingRepo.ListFundHoldingTransactionsFiltered(ctx, userID, domain.UserFundHoldingTransactionFilter{
		FundID: transaction.FundID,
		Limit:  maxHoldingTransactionContextLimit,
	})
	if err != nil {
		return false, err
	}
	for _, candidate := range related {
		if candidate.ID == transaction.ID || candidate.Voided || !candidate.CreatedAt.After(transaction.CreatedAt) {
			continue
		}
		if transaction.HoldingID != "" && candidate.HoldingID == transaction.HoldingID {
			return true, nil
		}
		if transaction.HoldingID == "" && candidate.FundID == transaction.FundID {
			return true, nil
		}
	}
	return false, nil
}

func canApplyHoldingRollbackAutomatically(transaction domain.UserFundHoldingTransaction, currentSnapshot *domain.UserFundHolding, hasSubsequent bool) bool {
	if transaction.Voided || hasSubsequent {
		return false
	}

	switch transaction.Type {
	case domain.UserFundHoldingTransactionBuy:
		return currentSnapshot != nil && currentSnapshot.Amount.GreaterThanOrEqual(transaction.Amount)
	case domain.UserFundHoldingTransactionSell:
		if transaction.Metadata["sell_all"] == "true" {
			return currentSnapshot == nil && strings.TrimSpace(transaction.Metadata["previous_amount"]) != ""
		}
		return currentSnapshot != nil && transaction.Amount.GreaterThan(decimal.Zero)
	case domain.UserFundHoldingTransactionCorrection:
		return currentSnapshot != nil && strings.TrimSpace(transaction.Metadata["previous_amount"]) != ""
	case domain.UserFundHoldingTransactionDividend:
		if transaction.Metadata["reinvest"] == "true" {
			return currentSnapshot != nil && currentSnapshot.Shares.GreaterThanOrEqual(transaction.Shares)
		}
		return true
	case domain.UserFundHoldingTransactionAdjustment:
		return currentSnapshot != nil && strings.TrimSpace(transaction.Metadata["previous_shares"]) != ""
	case domain.UserFundHoldingTransactionDelete:
		return currentSnapshot == nil && transaction.Amount.GreaterThan(decimal.Zero)
	default:
		return false
	}
}

func (s *UserPreferenceService) applyHoldingRollbackSnapshot(ctx context.Context, transaction domain.UserFundHoldingTransaction, currentSnapshot *domain.UserFundHolding) (*domain.UserFundHolding, bool, bool, error) {
	switch transaction.Type {
	case domain.UserFundHoldingTransactionBuy:
		return s.rollbackBuyTransaction(ctx, transaction, currentSnapshot)
	case domain.UserFundHoldingTransactionSell:
		return s.rollbackSellTransaction(ctx, transaction, currentSnapshot)
	case domain.UserFundHoldingTransactionCorrection:
		return s.rollbackCorrectionTransaction(ctx, transaction, currentSnapshot)
	case domain.UserFundHoldingTransactionDividend:
		return s.rollbackDividendTransaction(ctx, transaction, currentSnapshot)
	case domain.UserFundHoldingTransactionAdjustment:
		return s.rollbackAdjustmentTransaction(ctx, transaction, currentSnapshot)
	case domain.UserFundHoldingTransactionDelete:
		return s.restoreHoldingFromTransaction(ctx, transaction, "自动冲正：恢复误删持仓")
	default:
		return nil, false, false, ErrUnsafeHoldingRollback
	}
}

func (s *UserPreferenceService) rollbackBuyTransaction(ctx context.Context, transaction domain.UserFundHoldingTransaction, current *domain.UserFundHolding) (*domain.UserFundHolding, bool, bool, error) {
	if current == nil || current.Amount.LessThan(transaction.Amount) {
		return nil, false, false, ErrUnsafeHoldingRollback
	}
	updated := *current
	updated.Amount = updated.Amount.Sub(transaction.Amount)
	if transaction.Shares.GreaterThan(decimal.Zero) && updated.Shares.GreaterThanOrEqual(transaction.Shares) {
		updated.Shares = updated.Shares.Sub(transaction.Shares)
	}
	updated.Note = appendRollbackNote(updated.Note, "自动冲正买入流水")
	updated.UpdatedAt = time.Now()

	if !updated.Amount.GreaterThan(decimal.Zero) {
		if err := s.recordFundHoldingTransaction(ctx, domain.UserFundHoldingTransactionDelete, current, "自动冲正：移除误录买入", map[string]string{
			"source":                  "rollback_apply",
			"rolled_back_transaction": transaction.ID,
		}); err != nil {
			return nil, false, false, err
		}
		if err := s.fundHoldingRepo.DeleteFundHolding(ctx, transaction.UserID, current.ID); err != nil {
			return nil, false, false, err
		}
		return nil, true, false, nil
	}
	if err := s.fundHoldingRepo.SaveFundHolding(ctx, &updated); err != nil {
		return nil, false, false, err
	}
	if err := s.recordFundHoldingTransaction(ctx, domain.UserFundHoldingTransactionCorrection, &updated, "自动冲正：扣回误录买入", rollbackMetadata(transaction, current, &updated)); err != nil {
		return nil, false, false, err
	}
	return &updated, false, false, nil
}

func (s *UserPreferenceService) rollbackSellTransaction(ctx context.Context, transaction domain.UserFundHoldingTransaction, current *domain.UserFundHolding) (*domain.UserFundHolding, bool, bool, error) {
	if transaction.Metadata["sell_all"] == "true" {
		return s.restoreHoldingFromTransaction(ctx, transaction, "自动冲正：恢复误清仓持仓")
	}
	if current == nil {
		return nil, false, false, ErrUnsafeHoldingRollback
	}
	updated := *current
	updated.Amount = updated.Amount.Add(transaction.Amount)
	if transaction.Shares.GreaterThan(decimal.Zero) {
		updated.Shares = updated.Shares.Add(transaction.Shares)
	}
	updated.Note = appendRollbackNote(updated.Note, "自动冲正卖出流水")
	updated.UpdatedAt = time.Now()
	if err := s.fundHoldingRepo.SaveFundHolding(ctx, &updated); err != nil {
		return nil, false, false, err
	}
	if err := s.recordFundHoldingTransaction(ctx, domain.UserFundHoldingTransactionCorrection, &updated, "自动冲正：加回误录卖出", rollbackMetadata(transaction, current, &updated)); err != nil {
		return nil, false, false, err
	}
	return &updated, false, false, nil
}

func (s *UserPreferenceService) rollbackCorrectionTransaction(ctx context.Context, transaction domain.UserFundHoldingTransaction, current *domain.UserFundHolding) (*domain.UserFundHolding, bool, bool, error) {
	if current == nil {
		return nil, false, false, ErrUnsafeHoldingRollback
	}
	updated, err := restoreHoldingPreviousSnapshot(transaction, current)
	if err != nil {
		return nil, false, false, err
	}
	updated.Note = appendRollbackNote(updated.Note, "自动冲正校正流水")
	if err := s.fundHoldingRepo.SaveFundHolding(ctx, updated); err != nil {
		return nil, false, false, err
	}
	if err := s.recordFundHoldingTransaction(ctx, domain.UserFundHoldingTransactionCorrection, updated, "自动冲正：恢复校正前口径", rollbackMetadata(transaction, current, updated)); err != nil {
		return nil, false, false, err
	}
	return updated, false, false, nil
}

func (s *UserPreferenceService) rollbackDividendTransaction(ctx context.Context, transaction domain.UserFundHoldingTransaction, current *domain.UserFundHolding) (*domain.UserFundHolding, bool, bool, error) {
	if transaction.Metadata["reinvest"] != "true" {
		return current, false, false, nil
	}
	if current == nil || current.Shares.LessThan(transaction.Shares) {
		return nil, false, false, ErrUnsafeHoldingRollback
	}
	updated := *current
	updated.Shares = updated.Shares.Sub(transaction.Shares)
	restoreOptionalPreviousConfirmation(transaction, &updated)
	updated.Note = appendRollbackNote(updated.Note, "自动冲正红利再投")
	updated.UpdatedAt = time.Now()
	if err := s.fundHoldingRepo.SaveFundHolding(ctx, &updated); err != nil {
		return nil, false, false, err
	}
	if err := s.recordFundHoldingTransaction(ctx, domain.UserFundHoldingTransactionCorrection, &updated, "自动冲正：扣回红利再投份额", rollbackMetadata(transaction, current, &updated)); err != nil {
		return nil, false, false, err
	}
	return &updated, false, false, nil
}

func (s *UserPreferenceService) rollbackAdjustmentTransaction(ctx context.Context, transaction domain.UserFundHoldingTransaction, current *domain.UserFundHolding) (*domain.UserFundHolding, bool, bool, error) {
	return s.rollbackCorrectionTransaction(ctx, transaction, current)
}

func (s *UserPreferenceService) restoreHoldingFromTransaction(ctx context.Context, transaction domain.UserFundHoldingTransaction, note string) (*domain.UserFundHolding, bool, bool, error) {
	amount := transaction.Amount
	if previousAmount, ok, err := decimalFromMetadata(transaction.Metadata, "previous_amount"); err != nil {
		return nil, false, false, err
	} else if ok {
		amount = previousAmount
	}
	if !amount.GreaterThan(decimal.Zero) {
		return nil, false, false, ErrUnsafeHoldingRollback
	}
	now := time.Now()
	restored := &domain.UserFundHolding{
		ID:                 transaction.HoldingID,
		UserID:             transaction.UserID,
		FundID:             transaction.FundID,
		Amount:             amount,
		Shares:             transaction.Shares,
		ConfirmedNav:       transaction.ConfirmedNav,
		ConfirmedNavDate:   transaction.ConfirmedNavDate,
		ManualConfirmation: transaction.ManualConfirmation,
		TradeAt:            transaction.TradeAt,
		AsOfDate:           nonEmpty(transaction.AsOfDate, transaction.ConfirmedNavDate),
		Note:               appendRollbackNote(transaction.Note, note),
		SourcePlatform:     transaction.SourcePlatform,
		SourceLabel:        transaction.SourceLabel,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if restored.ID == "" {
		restored.ID = generateID("ufh")
	}
	if restored.AsOfDate == "" {
		restored.AsOfDate = time.Now().In(holdingTradeLocation).Format("2006-01-02")
	}
	restoreOptionalPreviousConfirmation(transaction, restored)
	if err := s.fundHoldingRepo.SaveFundHolding(ctx, restored); err != nil {
		return nil, false, false, err
	}
	if err := s.recordFundHoldingTransaction(ctx, domain.UserFundHoldingTransactionCorrection, restored, note, map[string]string{
		"source":                  "rollback_apply",
		"rolled_back_transaction": transaction.ID,
		"restored_holding":        "true",
	}); err != nil {
		return nil, false, false, err
	}
	return restored, false, true, nil
}

func restoreHoldingPreviousSnapshot(transaction domain.UserFundHoldingTransaction, current *domain.UserFundHolding) (*domain.UserFundHolding, error) {
	updated := *current
	amount, ok, err := decimalFromMetadata(transaction.Metadata, "previous_amount")
	if err != nil || !ok || !amount.GreaterThan(decimal.Zero) {
		return nil, ErrUnsafeHoldingRollback
	}
	updated.Amount = amount
	if shares, ok, err := decimalFromMetadata(transaction.Metadata, "previous_shares"); err != nil {
		return nil, err
	} else if ok {
		updated.Shares = shares
	}
	restoreOptionalPreviousConfirmation(transaction, &updated)
	if manual, ok := boolFromMetadata(transaction.Metadata, "previous_manual_confirmation"); ok {
		updated.ManualConfirmation = manual
	}
	if tradeAt := strings.TrimSpace(transaction.Metadata["previous_trade_at"]); tradeAt != "" {
		updated.TradeAt = tradeAt
	}
	if asOfDate := strings.TrimSpace(transaction.Metadata["previous_as_of_date"]); asOfDate != "" {
		updated.AsOfDate = asOfDate
	}
	updated.UpdatedAt = time.Now()
	return &updated, nil
}

func restoreOptionalPreviousConfirmation(transaction domain.UserFundHoldingTransaction, holding *domain.UserFundHolding) {
	if metadataHasKey(transaction.Metadata, "previous_confirmed_nav") {
		if nav, ok, err := decimalFromMetadata(transaction.Metadata, "previous_confirmed_nav"); err == nil && ok {
			holding.ConfirmedNav = nav
		} else {
			holding.ConfirmedNav = decimal.Zero
		}
	}
	if metadataHasKey(transaction.Metadata, "previous_confirmed_nav_date") {
		holding.ConfirmedNavDate = strings.TrimSpace(transaction.Metadata["previous_confirmed_nav_date"])
	}
	if metadataHasKey(transaction.Metadata, "previous_shares") {
		if shares, ok, err := decimalFromMetadata(transaction.Metadata, "previous_shares"); err == nil && ok {
			holding.Shares = shares
		} else {
			holding.Shares = decimal.Zero
		}
	}
	if manual, ok := boolFromMetadata(transaction.Metadata, "previous_manual_confirmation"); ok {
		holding.ManualConfirmation = manual
	}
}

func metadataHasKey(metadata map[string]string, key string) bool {
	if metadata == nil {
		return false
	}
	_, ok := metadata[key]
	return ok
}

func decimalFromMetadata(metadata map[string]string, key string) (decimal.Decimal, bool, error) {
	raw := strings.TrimSpace(metadata[key])
	if raw == "" {
		return decimal.Zero, false, nil
	}
	value, err := decimal.NewFromString(raw)
	if err != nil || value.LessThan(decimal.Zero) {
		return decimal.Zero, false, ErrUnsafeHoldingRollback
	}
	return value, true, nil
}

func boolFromMetadata(metadata map[string]string, key string) (bool, bool) {
	raw := strings.TrimSpace(metadata[key])
	if raw == "" {
		return false, false
	}
	return raw == "true", true
}

func appendRollbackNote(note, suffix string) string {
	note = strings.TrimSpace(note)
	if note == "" {
		return suffix
	}
	return note + "；" + suffix
}

func rollbackMetadata(transaction domain.UserFundHoldingTransaction, before, after *domain.UserFundHolding) map[string]string {
	metadata := map[string]string{
		"source":                  "rollback_apply",
		"rolled_back_transaction": transaction.ID,
		"rolled_back_type":        string(transaction.Type),
	}
	if before != nil {
		metadata["previous_amount"] = before.Amount.String()
		metadata["previous_shares"] = decimalStringOrEmpty(before.Shares)
		metadata["previous_confirmed_nav"] = decimalStringOrEmpty(before.ConfirmedNav)
		metadata["previous_confirmed_nav_date"] = before.ConfirmedNavDate
	}
	if after != nil {
		metadata["remaining_amount"] = after.Amount.String()
		metadata["remaining_shares"] = decimalStringOrEmpty(after.Shares)
	}
	return metadata
}

func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func buildHoldingTransactionRollbackPreview(
	transaction domain.UserFundHoldingTransaction,
	currentSnapshot *domain.UserFundHolding,
	currentHolding *domain.UserFundHoldingDetail,
) domain.UserFundHoldingTransactionRollbackPreview {
	preview := domain.UserFundHoldingTransactionRollbackPreview{
		Transaction:           transaction,
		CurrentHolding:        currentHolding,
		PreviewOnly:           true,
		CanApplyAutomatically: false,
		State:                 "preview",
		Title:                 "回滚影响预览",
		Summary:               "这是只读预览，不会自动修改当前持仓快照。",
		SuggestedAction:       "如确认历史流水录错，请先作废流水；如果当前金额或份额也需要变化，再使用持仓校正手动调整。",
		Warnings: []string{
			"当前版本不会自动回滚资产，避免误操作影响真实持仓。",
			"若这条流水之后还有卖出、分红、校正或份额调整，预览只作为人工对账参考。",
		},
	}

	if transaction.Voided {
		preview.State = "voided"
		preview.Warnings = append(preview.Warnings, "这条流水已作废；重复按预览调整前，请先核对当前平台实际持仓。")
	}
	if currentSnapshot == nil {
		preview.State = "no_current_holding"
		preview.Warnings = append(preview.Warnings, "当前持仓快照已不存在，可能已真实清仓或删除；本预览无法自动计算最终资产。")
	}

	switch transaction.Type {
	case domain.UserFundHoldingTransactionBuy:
		preview.Title = "买入/补仓作废影响"
		preview.Summary = "如果这笔买入是误录，理论上需要从当前本金与份额中扣除该笔记录。"
		preview.AffectedFields = append(preview.AffectedFields,
			rollbackAmountField("amount", "本金", currentDecimalString(currentSnapshot, "amount"), "", transaction.Amount.Neg(), "减少"),
			rollbackAmountField("shares", "份额", currentDecimalString(currentSnapshot, "shares"), "", transaction.Shares.Neg(), "减少"),
		)
	case domain.UserFundHoldingTransactionCorrection:
		preview.Title = "校正流水回滚影响"
		preview.Summary = "如果这次校正录错，建议把当前持仓手动校正回流水 metadata 中记录的校正前口径。"
		preview.AffectedFields = append(preview.AffectedFields,
			rollbackStringField("amount", "本金", currentDecimalString(currentSnapshot, "amount"), transaction.Metadata["previous_amount"]),
			rollbackStringField("shares", "份额", currentDecimalString(currentSnapshot, "shares"), transaction.Metadata["previous_shares"]),
			rollbackStringField("confirmed_nav", "确认净值", currentDecimalString(currentSnapshot, "confirmed_nav"), transaction.Metadata["previous_confirmed_nav"]),
			rollbackStringField("confirmed_nav_date", "确认净值日", currentStringValue(currentSnapshot, "confirmed_nav_date"), transaction.Metadata["previous_confirmed_nav_date"]),
		)
	case domain.UserFundHoldingTransactionSell:
		preview.Title = "卖出/清仓作废影响"
		preview.Summary = "如果这笔卖出录错，理论上需要把卖出的本金与份额加回当前持仓。"
		if transaction.Metadata["sell_all"] == "true" {
			preview.Summary = "这是一笔全部清仓流水；如果作废后仍需要恢复持仓，建议按外部平台数据重新记录或手动校正。"
			preview.Warnings = append(preview.Warnings, "清仓流水通常已删除当前快照，无法安全自动恢复。")
		}
		preview.AffectedFields = append(preview.AffectedFields,
			rollbackAmountField("amount", "本金", currentDecimalString(currentSnapshot, "amount"), transaction.Metadata["previous_amount"], transaction.Amount, "增加"),
			rollbackAmountField("shares", "份额", currentDecimalString(currentSnapshot, "shares"), transaction.Metadata["previous_shares"], transaction.Shares, "增加"),
		)
	case domain.UserFundHoldingTransactionDividend:
		preview.Title = "分红流水作废影响"
		if transaction.Metadata["reinvest"] == "true" {
			preview.Summary = "红利再投会增加份额；如果这笔分红录错，理论上需要扣回再投份额。"
			preview.AffectedFields = append(preview.AffectedFields,
				rollbackAmountField("shares", "份额", currentDecimalString(currentSnapshot, "shares"), transaction.Metadata["previous_shares"], transaction.Shares.Neg(), "减少"),
				rollbackStringField("confirmed_nav", "确认净值", currentDecimalString(currentSnapshot, "confirmed_nav"), transaction.Metadata["previous_confirmed_nav"]),
			)
		} else {
			preview.Summary = "现金分红只记录账本流水，不改变当前持仓快照；作废后通常不需要校正本金或份额。"
			preview.AffectedFields = append(preview.AffectedFields,
				rollbackAmountField("dividend_amount", "现金分红", transaction.Amount.StringFixedBank(2), "", transaction.Amount.Neg(), "移除流水"),
			)
		}
	case domain.UserFundHoldingTransactionAdjustment:
		preview.Title = "份额调整回滚影响"
		preview.Summary = "如果这次份额调整录错，建议把份额和确认净值手动校正回调整前口径。"
		preview.AffectedFields = append(preview.AffectedFields,
			rollbackStringField("shares", "份额", currentDecimalString(currentSnapshot, "shares"), transaction.Metadata["previous_shares"]),
			rollbackStringField("confirmed_nav", "确认净值", currentDecimalString(currentSnapshot, "confirmed_nav"), transaction.Metadata["previous_confirmed_nav"]),
			rollbackStringField("confirmed_nav_date", "确认净值日", currentStringValue(currentSnapshot, "confirmed_nav_date"), transaction.Metadata["previous_confirmed_nav_date"]),
		)
	case domain.UserFundHoldingTransactionDelete:
		preview.Title = "删除流水恢复影响"
		preview.Summary = "删除记录已移除当前快照；如果这是误删，建议按外部平台数据重新记录一笔持仓。"
		preview.AffectedFields = append(preview.AffectedFields,
			rollbackStringField("amount", "本金", currentDecimalString(currentSnapshot, "amount"), transaction.Amount.StringFixedBank(2)),
			rollbackStringField("shares", "份额", currentDecimalString(currentSnapshot, "shares"), decimalStringOrEmpty(transaction.Shares)),
			rollbackStringField("confirmed_nav", "确认净值", currentDecimalString(currentSnapshot, "confirmed_nav"), decimalStringOrEmpty(transaction.ConfirmedNav)),
			rollbackStringField("confirmed_nav_date", "确认净值日", currentStringValue(currentSnapshot, "confirmed_nav_date"), transaction.ConfirmedNavDate),
		)
	default:
		preview.Warnings = append(preview.Warnings, "暂未识别的流水类型，只展示基础作废提示。")
	}

	preview.AffectedFields = compactRollbackFields(preview.AffectedFields)
	return preview
}

func rollbackStringField(field, label, currentValue, rollbackValue string) domain.UserFundHoldingTransactionRollbackField {
	return domain.UserFundHoldingTransactionRollbackField{
		Field:         field,
		Label:         label,
		CurrentValue:  strings.TrimSpace(currentValue),
		RollbackValue: strings.TrimSpace(rollbackValue),
	}
}

func rollbackAmountField(field, label, currentValue, rollbackValue string, delta decimal.Decimal, direction string) domain.UserFundHoldingTransactionRollbackField {
	result := rollbackStringField(field, label, currentValue, rollbackValue)
	if !delta.Equal(decimal.Zero) {
		result.Delta = delta.String()
		result.Direction = direction
	}
	return result
}

func compactRollbackFields(fields []domain.UserFundHoldingTransactionRollbackField) []domain.UserFundHoldingTransactionRollbackField {
	result := make([]domain.UserFundHoldingTransactionRollbackField, 0, len(fields))
	for _, field := range fields {
		if field.CurrentValue == "" && field.RollbackValue == "" && field.Delta == "" {
			continue
		}
		result = append(result, field)
	}
	return result
}

func currentDecimalString(holding *domain.UserFundHolding, field string) string {
	if holding == nil {
		return ""
	}
	switch field {
	case "amount":
		return holding.Amount.StringFixedBank(2)
	case "shares":
		return decimalStringOrEmpty(holding.Shares)
	case "confirmed_nav":
		return decimalStringOrEmpty(holding.ConfirmedNav)
	default:
		return ""
	}
}

func currentStringValue(holding *domain.UserFundHolding, field string) string {
	if holding == nil {
		return ""
	}
	switch field {
	case "confirmed_nav_date":
		return holding.ConfirmedNavDate
	default:
		return ""
	}
}

// DeleteFundHolding removes a user fund-level position record.
func (s *UserPreferenceService) DeleteFundHolding(ctx context.Context, userID, holdingID string) error {
	holdingID = strings.TrimSpace(holdingID)
	if holdingID == "" {
		return ErrFundHoldingNotFound
	}
	holding, err := s.fundHoldingRepo.GetFundHolding(ctx, userID, holdingID)
	if err != nil {
		return err
	}
	if holding == nil {
		return ErrFundHoldingNotFound
	}
	if err := s.recordFundHoldingTransaction(ctx, domain.UserFundHoldingTransactionDelete, holding, "删除持仓记录", map[string]string{
		"source": "delete_holding",
	}); err != nil {
		return err
	}
	return s.fundHoldingRepo.DeleteFundHolding(ctx, userID, holdingID)
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

func normalizeWatchlistAccent(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "cyan":
		return "cyan"
	case "emerald":
		return "emerald"
	case "amber":
		return "amber"
	case "fuchsia":
		return "fuchsia"
	default:
		return ""
	}
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

func collectHoldingTransactionFundIDs(transactions []domain.UserFundHoldingTransaction) []string {
	fundIDs := make([]string, 0, len(transactions))
	for _, transaction := range transactions {
		fundIDs = append(fundIDs, transaction.FundID)
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

func parseOptionalHoldingTradeAt(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Now().In(holdingTradeLocation), nil
	}
	return parseHoldingTradeAt(raw)
}

func resolveHoldingPricingDate(tradeAt time.Time) time.Time {
	resolution := trading.ResolvePricingDate(tradeAt)
	pricingDate, err := time.ParseInLocation("2006-01-02", resolution.PricingDate, holdingTradeLocation)
	if err != nil {
		return tradeAt.In(holdingTradeLocation)
	}
	return pricingDate
}

type holdingRealMetrics struct {
	currentMarketValue decimal.Decimal
	todayProfit        decimal.Decimal
	todayChangePercent decimal.Decimal
}

type holdingAggregateAccumulator struct {
	fundID                string
	fund                  *domain.Fund
	holdingCount          int
	confirmedHoldingCount int
	realMetricsReadyCount int
	totalPrincipal        decimal.Decimal
	confirmedPrincipal    decimal.Decimal
	readyPrincipal        decimal.Decimal
	confirmedShares       decimal.Decimal
	officialCurrentValue  decimal.Decimal
	officialTodayProfit   decimal.Decimal
	officialPreviousValue decimal.Decimal
}

func (a *holdingAggregateAccumulator) add(holding domain.UserFundHolding, metrics *holdingRealMetrics) {
	if a == nil {
		return
	}

	a.holdingCount++
	a.totalPrincipal = a.totalPrincipal.Add(holding.Amount)

	if holding.Shares.GreaterThan(decimal.Zero) {
		a.confirmedHoldingCount++
		a.confirmedPrincipal = a.confirmedPrincipal.Add(holding.Amount)
		a.confirmedShares = a.confirmedShares.Add(holding.Shares)
	}

	if metrics == nil {
		return
	}

	a.realMetricsReadyCount++
	a.readyPrincipal = a.readyPrincipal.Add(holding.Amount)
	a.officialCurrentValue = a.officialCurrentValue.Add(metrics.currentMarketValue)
	a.officialTodayProfit = a.officialTodayProfit.Add(metrics.todayProfit)
	a.officialPreviousValue = a.officialPreviousValue.Add(metrics.currentMarketValue.Sub(metrics.todayProfit))
}

func buildUserFundHoldingDetail(
	holding domain.UserFundHolding,
	fund *domain.Fund,
	latestHistory *domain.FundHistory,
	expectedOfficialDate string,
) (domain.UserFundHoldingDetail, *holdingRealMetrics) {
	detail := domain.UserFundHoldingDetail{
		ID:                 holding.ID,
		FundID:             holding.FundID,
		Amount:             holding.Amount,
		Shares:             decimalStringOrEmpty(holding.Shares),
		ConfirmedNav:       decimalStringOrEmpty(holding.ConfirmedNav),
		ConfirmedNavDate:   holding.ConfirmedNavDate,
		ManualConfirmation: holding.ManualConfirmation,
		TradeAt:            holding.TradeAt,
		AsOfDate:           holding.AsOfDate,
		Note:               holding.Note,
		CreatedAt:          holding.CreatedAt,
		UpdatedAt:          holding.UpdatedAt,
		Fund:               fund,
	}

	if latestHistory != nil && latestHistory.Date == expectedOfficialDate {
		detail.ActualDate = latestHistory.Date
		detail.ActualNav = latestHistory.NetAssetVal.String()
		detail.ActualDailyReturn = latestHistory.DailyReturn.String()
	}

	metrics, ready, message := calculateHoldingRealMetrics(holding, latestHistory, expectedOfficialDate)
	if !ready {
		detail.RealMetricsMessage = message
		return detail, nil
	}

	detail.CurrentMarketValue = metrics.currentMarketValue.StringFixedBank(2)
	detail.TodayProfit = metrics.todayProfit.StringFixedBank(2)
	detail.TodayChangePercent = metrics.todayChangePercent.String()
	detail.RealMetricsReady = true
	return detail, metrics
}

func buildUserFundHoldingSummary(
	holdings []domain.UserFundHolding,
	readyCount int,
	readyPrincipal decimal.Decimal,
	totalCurrentMarketValue decimal.Decimal,
	totalTodayProfit decimal.Decimal,
	totalPreviousMarketValue decimal.Decimal,
) domain.UserFundHoldingSummary {
	summary := domain.UserFundHoldingSummary{
		TotalPrincipal:          decimal.Zero,
		RealMetricsReady:        false,
		RealMetricsReadyCount:   readyCount,
		TotalHoldings:           len(holdings),
		IncompleteHoldingsCount: max(len(holdings)-readyCount, 0),
	}

	for _, holding := range holdings {
		summary.TotalPrincipal = summary.TotalPrincipal.Add(holding.Amount)
	}

	switch {
	case len(holdings) == 0:
		summary.MetricsScope = "none"
		return summary
	case readyCount == 0:
		summary.MetricsScope = "none"
		summary.Message = "待官方净值与确认份额齐备后展示真实市值与盈亏。"
		return summary
	}

	summary.ReadyPrincipal = readyPrincipal.StringFixedBank(2)
	summary.TotalCurrentMarketValue = totalCurrentMarketValue.StringFixedBank(2)
	summary.TotalTodayProfit = totalTodayProfit.StringFixedBank(2)
	if totalPreviousMarketValue.GreaterThan(decimal.Zero) {
		summary.TotalTodayChangePercent = totalTodayProfit.
			DivRound(totalPreviousMarketValue, 8).
			Mul(decimal.NewFromInt(100)).
			String()
	}

	if readyCount < len(holdings) {
		summary.MetricsScope = "partial"
		summary.Message = fmt.Sprintf(
			"已按最新官方净值汇总 %d/%d 条持仓，剩余持仓待确认净值或今日官方净值同步。",
			readyCount,
			len(holdings),
		)
		return summary
	}

	summary.MetricsScope = "full"
	summary.RealMetricsReady = true
	return summary
}

func buildUserFundHoldingAggregates(
	accumulators map[string]*holdingAggregateAccumulator,
	order []string,
) []domain.UserFundHoldingAggregate {
	if len(order) == 0 {
		return []domain.UserFundHoldingAggregate{}
	}

	aggregates := make([]domain.UserFundHoldingAggregate, 0, len(order))
	for _, fundID := range order {
		accumulator := accumulators[fundID]
		if accumulator == nil {
			continue
		}

		aggregate := domain.UserFundHoldingAggregate{
			FundID:                  accumulator.fundID,
			HoldingCount:            accumulator.holdingCount,
			ConfirmedHoldingCount:   accumulator.confirmedHoldingCount,
			RealMetricsReadyCount:   accumulator.realMetricsReadyCount,
			IncompleteHoldingsCount: max(accumulator.holdingCount-accumulator.realMetricsReadyCount, 0),
			TotalPrincipal:          accumulator.totalPrincipal,
			MetricsScope:            "none",
			RealMetricsReady:        false,
			Fund:                    accumulator.fund,
		}

		if accumulator.confirmedPrincipal.GreaterThan(decimal.Zero) {
			aggregate.ConfirmedPrincipal = accumulator.confirmedPrincipal.StringFixedBank(2)
		}
		if accumulator.readyPrincipal.GreaterThan(decimal.Zero) {
			aggregate.ReadyPrincipal = accumulator.readyPrincipal.StringFixedBank(2)
		}
		if accumulator.confirmedShares.GreaterThan(decimal.Zero) {
			aggregate.ConfirmedShares = accumulator.confirmedShares.StringFixedBank(6)
		}

		switch {
		case accumulator.holdingCount == 0:
			aggregate.MetricsScope = "none"
		case accumulator.realMetricsReadyCount == 0:
			aggregate.MetricsScope = "none"
			if accumulator.confirmedHoldingCount > 0 {
				aggregate.Message = "已确认部分份额，待最近官方净值同步后展示官方汇总。"
			} else {
				aggregate.Message = "待确认净值补齐后展示官方汇总。"
			}
		default:
			aggregate.OfficialCurrentMarketValue = accumulator.officialCurrentValue.StringFixedBank(2)
			aggregate.OfficialTodayProfit = accumulator.officialTodayProfit.StringFixedBank(2)
			if accumulator.officialPreviousValue.GreaterThan(decimal.Zero) {
				aggregate.OfficialTodayChangePercent = accumulator.officialTodayProfit.
					DivRound(accumulator.officialPreviousValue, 8).
					Mul(decimal.NewFromInt(100)).
					String()
			}

			if accumulator.realMetricsReadyCount < accumulator.holdingCount {
				aggregate.MetricsScope = "partial"
				aggregate.Message = fmt.Sprintf(
					"已按最新官方净值汇总 %d/%d 笔，剩余分笔待补齐。",
					accumulator.realMetricsReadyCount,
					accumulator.holdingCount,
				)
			} else {
				aggregate.MetricsScope = "full"
				aggregate.RealMetricsReady = true
			}
		}

		aggregates = append(aggregates, aggregate)
	}

	return aggregates
}

func calculateHoldingRealMetrics(
	holding domain.UserFundHolding,
	latestHistory *domain.FundHistory,
	expectedOfficialDate string,
) (*holdingRealMetrics, bool, string) {
	if latestHistory == nil || latestHistory.Date != expectedOfficialDate {
		return nil, false, "待今日官方净值同步完成后展示真实市值与盈亏。"
	}
	if !holding.Shares.GreaterThan(decimal.Zero) {
		return nil, false, "待确认净值补齐后展示真实市值与盈亏。"
	}
	if !latestHistory.NetAssetVal.GreaterThan(decimal.Zero) {
		return nil, false, "官方净值异常，暂无法计算真实市值。"
	}

	changeRatio := latestHistory.DailyReturn.DivRound(decimal.NewFromInt(100), 8)
	divisor := decimal.NewFromInt(1).Add(changeRatio)
	if !divisor.GreaterThan(decimal.Zero) {
		return nil, false, "真实涨跌幅异常，暂无法计算今日盈亏。"
	}

	previousNav := latestHistory.NetAssetVal.DivRound(divisor, 8)
	currentMarketValue := holding.Shares.Mul(latestHistory.NetAssetVal)
	todayProfit := holding.Shares.Mul(latestHistory.NetAssetVal.Sub(previousNav))

	return &holdingRealMetrics{
		currentMarketValue: currentMarketValue,
		todayProfit:        todayProfit,
		todayChangePercent: latestHistory.DailyReturn,
	}, true, ""
}

func (s *UserPreferenceService) lookupFundHistory(ctx context.Context, fundID, date string) (*domain.FundHistory, error) {
	histories, err := s.fundRepo.GetFundHistoriesByLookupKeys(ctx, []domain.FundHistoryLookupKey{{
		FundID: strings.TrimSpace(fundID),
		Date:   strings.TrimSpace(date),
	}})
	if err != nil {
		return nil, err
	}
	return histories[domain.FundHistoryLookupKey{
		FundID: strings.TrimSpace(fundID),
		Date:   strings.TrimSpace(date),
	}], nil
}

func needsHoldingConfirmationData(holding domain.UserFundHolding) bool {
	return !holding.Shares.GreaterThan(decimal.Zero) || !holding.ConfirmedNav.GreaterThan(decimal.Zero) || strings.TrimSpace(holding.ConfirmedNavDate) == ""
}

func collectHoldingHistoryLookupKeys(holdings []domain.UserFundHolding) []domain.FundHistoryLookupKey {
	keys := make([]domain.FundHistoryLookupKey, 0, len(holdings))
	seen := make(map[domain.FundHistoryLookupKey]struct{}, len(holdings))
	for _, holding := range holdings {
		if !needsHoldingConfirmationData(holding) {
			continue
		}
		key := holdingHistoryLookupKey(holding)
		if key.FundID == "" || key.Date == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}

func holdingHistoryLookupKey(holding domain.UserFundHolding) domain.FundHistoryLookupKey {
	return domain.FundHistoryLookupKey{
		FundID: strings.TrimSpace(holding.FundID),
		Date:   strings.TrimSpace(holding.AsOfDate),
	}
}

func applyHoldingConfirmationData(holding *domain.UserFundHolding, history *domain.FundHistory) bool {
	if holding == nil || history == nil {
		return false
	}
	if !holding.Amount.GreaterThan(decimal.Zero) || !history.NetAssetVal.GreaterThan(decimal.Zero) {
		return false
	}

	holding.ConfirmedNav = history.NetAssetVal
	holding.ConfirmedNavDate = history.Date
	holding.Shares = holding.Amount.DivRound(history.NetAssetVal, 6)
	holding.ManualConfirmation = false
	return holding.Shares.GreaterThan(decimal.Zero)
}

func (s *UserPreferenceService) recordFundHoldingTransaction(
	ctx context.Context,
	txType domain.UserFundHoldingTransactionType,
	holding *domain.UserFundHolding,
	note string,
	metadata map[string]string,
) error {
	if holding == nil {
		return nil
	}

	now := time.Now()
	transaction := &domain.UserFundHoldingTransaction{
		ID:                 generateID("uht"),
		UserID:             holding.UserID,
		HoldingID:          holding.ID,
		FundID:             holding.FundID,
		Type:               txType,
		Amount:             holding.Amount,
		Shares:             holding.Shares,
		ConfirmedNav:       holding.ConfirmedNav,
		ConfirmedNavDate:   holding.ConfirmedNavDate,
		ManualConfirmation: holding.ManualConfirmation,
		TradeAt:            holding.TradeAt,
		AsOfDate:           holding.AsOfDate,
		Note:               strings.TrimSpace(note),
		SourcePlatform:     holding.SourcePlatform,
		SourceLabel:        holding.SourceLabel,
		Metadata:           metadata,
		CreatedAt:          now,
	}

	return s.fundHoldingRepo.SaveFundHoldingTransaction(ctx, transaction)
}

func buildHoldingCorrectionMetadata(before, after *domain.UserFundHolding) map[string]string {
	if before == nil || after == nil {
		return map[string]string{"source": "update_holding"}
	}

	metadata := map[string]string{
		"source":                       "update_holding",
		"previous_amount":              before.Amount.String(),
		"previous_shares":              decimalStringOrEmpty(before.Shares),
		"previous_confirmed_nav":       decimalStringOrEmpty(before.ConfirmedNav),
		"previous_confirmed_nav_date":  before.ConfirmedNavDate,
		"previous_manual_confirmation": fmt.Sprintf("%t", before.ManualConfirmation),
	}
	if before.TradeAt != after.TradeAt {
		metadata["previous_trade_at"] = before.TradeAt
	}
	if before.AsOfDate != after.AsOfDate {
		metadata["previous_as_of_date"] = before.AsOfDate
	}
	return metadata
}

func buildHoldingSellMetadata(before, after *domain.UserFundHolding) map[string]string {
	if before == nil || after == nil {
		return map[string]string{"source": "sell_holding"}
	}

	return map[string]string{
		"source":                   "sell_holding",
		"previous_amount":          before.Amount.String(),
		"remaining_amount":         after.Amount.String(),
		"previous_shares":          decimalStringOrEmpty(before.Shares),
		"remaining_shares":         decimalStringOrEmpty(after.Shares),
		"confirmed_nav":            decimalStringOrEmpty(before.ConfirmedNav),
		"confirmed_nav_date":       before.ConfirmedNavDate,
		"manual_confirmation":      fmt.Sprintf("%t", before.ManualConfirmation),
		"remaining_manual_confirm": fmt.Sprintf("%t", after.ManualConfirmation),
	}
}

func buildHoldingDividendMetadata(before, after *domain.UserFundHolding, reinvest bool) map[string]string {
	if before == nil || after == nil {
		return map[string]string{"source": "dividend_holding"}
	}

	return map[string]string{
		"source":                       "dividend_holding",
		"reinvest":                     fmt.Sprintf("%t", reinvest),
		"previous_amount":              before.Amount.String(),
		"remaining_amount":             after.Amount.String(),
		"previous_shares":              decimalStringOrEmpty(before.Shares),
		"remaining_shares":             decimalStringOrEmpty(after.Shares),
		"previous_confirmed_nav":       decimalStringOrEmpty(before.ConfirmedNav),
		"remaining_confirmed_nav":      decimalStringOrEmpty(after.ConfirmedNav),
		"previous_confirmed_nav_date":  before.ConfirmedNavDate,
		"remaining_confirmed_nav_date": after.ConfirmedNavDate,
		"previous_manual_confirmation": fmt.Sprintf("%t", before.ManualConfirmation),
		"remaining_manual_confirm":     fmt.Sprintf("%t", after.ManualConfirmation),
	}
}

func buildHoldingAdjustmentMetadata(before, after *domain.UserFundHolding, sharesDelta decimal.Decimal) map[string]string {
	if before == nil || after == nil {
		return map[string]string{"source": "adjust_holding"}
	}

	return map[string]string{
		"source":                       "adjust_holding",
		"shares_delta":                 sharesDelta.String(),
		"previous_amount":              before.Amount.String(),
		"remaining_amount":             after.Amount.String(),
		"previous_shares":              decimalStringOrEmpty(before.Shares),
		"remaining_shares":             decimalStringOrEmpty(after.Shares),
		"previous_confirmed_nav":       decimalStringOrEmpty(before.ConfirmedNav),
		"remaining_confirmed_nav":      decimalStringOrEmpty(after.ConfirmedNav),
		"previous_confirmed_nav_date":  before.ConfirmedNavDate,
		"remaining_confirmed_nav_date": after.ConfirmedNavDate,
		"previous_manual_confirmation": fmt.Sprintf("%t", before.ManualConfirmation),
		"remaining_manual_confirm":     fmt.Sprintf("%t", after.ManualConfirmation),
	}
}

func decimalStringOrEmpty(value decimal.Decimal) string {
	if !value.GreaterThan(decimal.Zero) {
		return ""
	}
	return value.String()
}
