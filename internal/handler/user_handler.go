package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/middleware"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// UserHandler handles authenticated user preference requests.
type UserHandler struct {
	userPreferenceService domain.UserPreferenceService
	userRepo              domain.UserRepository
	defaultQuoteSource    domain.QuoteSource
}

// NewUserHandler creates a new UserHandler instance.
func NewUserHandler(
	userPreferenceService domain.UserPreferenceService,
	userRepo domain.UserRepository,
	defaultQuoteSource domain.QuoteSource,
) *UserHandler {
	return &UserHandler{
		userPreferenceService: userPreferenceService,
		userRepo:              userRepo,
		defaultQuoteSource:    domain.ResolveQuoteSource(defaultQuoteSource, domain.QuoteSourceSina),
	}
}

type addFavoriteFundRequest struct {
	FundID string `json:"fund_id"`
}

type createWatchlistGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateWatchlistGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Accent      string `json:"accent"`
}

type reorderWatchlistGroupsRequest struct {
	GroupIDs []string `json:"group_ids"`
}

type watchlistFundRequest struct {
	FundID string `json:"fund_id"`
}

type createFundHoldingRequest struct {
	FundID         string `json:"fund_id"`
	Amount         string `json:"amount"`
	AsOfDate       string `json:"as_of_date"`
	TradeAt        string `json:"trade_at"`
	Note           string `json:"note"`
	SourcePlatform string `json:"source_platform"`
	SourceLabel    string `json:"source_label"`
}

type createFundHoldingsBatchRequest struct {
	Items []createFundHoldingRequest `json:"items"`
}

type updateFundHoldingRequest struct {
	Amount           string `json:"amount"`
	Shares           string `json:"shares"`
	ConfirmedNav     string `json:"confirmed_nav"`
	ConfirmedNavDate string `json:"confirmed_nav_date"`
	TradeAt          string `json:"trade_at"`
	Note             string `json:"note"`
	SourcePlatform   string `json:"source_platform"`
	SourceLabel      string `json:"source_label"`
}

type sellFundHoldingRequest struct {
	Amount  string `json:"amount"`
	Shares  string `json:"shares"`
	TradeAt string `json:"trade_at"`
	Note    string `json:"note"`
	SellAll bool   `json:"sell_all"`
}

type dividendFundHoldingRequest struct {
	Amount         string `json:"amount"`
	Shares         string `json:"shares"`
	TradeAt        string `json:"trade_at"`
	Note           string `json:"note"`
	Reinvest       bool   `json:"reinvest"`
	SourcePlatform string `json:"source_platform"`
	SourceLabel    string `json:"source_label"`
}

type adjustFundHoldingSharesRequest struct {
	SharesDelta      string `json:"shares_delta"`
	TargetShares     string `json:"target_shares"`
	ConfirmedNav     string `json:"confirmed_nav"`
	ConfirmedNavDate string `json:"confirmed_nav_date"`
	TradeAt          string `json:"trade_at"`
	Note             string `json:"note"`
	SourcePlatform   string `json:"source_platform"`
	SourceLabel      string `json:"source_label"`
}

type voidFundHoldingTransactionRequest struct {
	Reason string `json:"reason"`
}

type applyFundHoldingTransactionRollbackRequest struct {
	Reason string `json:"reason"`
}

type holdingOverrideRequest struct {
	ID           string `json:"id"`
	StockCode    string `json:"stock_code"`
	StockName    string `json:"stock_name"`
	Exchange     string `json:"exchange"`
	HoldingRatio string `json:"holding_ratio"`
	Note         string `json:"note"`
}

type replaceHoldingOverridesRequest struct {
	Overrides []holdingOverrideRequest `json:"overrides"`
}

type updateQuoteSourceRequest struct {
	QuoteSource string `json:"quote_source"`
}

type quoteSourcePreferenceResponse struct {
	PreferredQuoteSource domain.QuoteSource `json:"preferred_quote_source"`
	EffectiveQuoteSource domain.QuoteSource `json:"effective_quote_source"`
}

// ListWatchlistGroups returns the authenticated user's grouped watchlists.
func (h *UserHandler) ListWatchlistGroups(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	groups, err := h.userPreferenceService.ListWatchlistGroups(c.Request.Context(), user.ID)
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: groups})
}

// GetQuoteSource returns the authenticated user's quote source preference.
func (h *UserHandler) GetQuoteSource(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: quoteSourcePreferenceResponse{
			PreferredQuoteSource: domain.NormalizeQuoteSource(string(user.PreferredQuoteSource)),
			EffectiveQuoteSource: domain.ResolveQuoteSource(user.PreferredQuoteSource, h.defaultQuoteSource),
		},
	})
}

// UpdateQuoteSource updates the authenticated user's preferred quote source.
func (h *UserHandler) UpdateQuoteSource(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	var req updateQuoteSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid quote source payload"},
		})
		return
	}

	quoteSource := domain.NormalizeQuoteSource(req.QuoteSource)
	if quoteSource == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_QUOTE_SOURCE", Message: "Unsupported quote source"},
		})
		return
	}
	if h.userRepo == nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   &APIError{Code: "SETTINGS_UNAVAILABLE", Message: "User settings storage is unavailable"},
		})
		return
	}

	currentUser, err := h.userRepo.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UPDATE_FAILED", Message: err.Error()},
		})
		return
	}
	if currentUser == nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   &APIError{Code: "USER_NOT_FOUND", Message: "User not found"},
		})
		return
	}

	currentUser.PreferredQuoteSource = quoteSource
	if err := h.userRepo.SaveUser(c.Request.Context(), currentUser); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UPDATE_FAILED", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: quoteSourcePreferenceResponse{
			PreferredQuoteSource: quoteSource,
			EffectiveQuoteSource: domain.ResolveQuoteSource(quoteSource, h.defaultQuoteSource),
		},
	})
}

// CreateWatchlistGroup creates a named watchlist bucket for the authenticated user.
func (h *UserHandler) CreateWatchlistGroup(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	var req createWatchlistGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid watchlist group payload"},
		})
		return
	}

	group, err := h.userPreferenceService.CreateWatchlistGroup(c.Request.Context(), user.ID, req.Name, req.Description)
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{Success: true, Data: group})
}

// UpdateWatchlistGroup updates a watchlist group owned by the authenticated user.
func (h *UserHandler) UpdateWatchlistGroup(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	groupID := c.Param("groupId")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_GROUP_ID", Message: "Group ID is required"},
		})
		return
	}

	var req updateWatchlistGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid watchlist group payload"},
		})
		return
	}

	group, err := h.userPreferenceService.UpdateWatchlistGroup(c.Request.Context(), user.ID, groupID, req.Name, req.Description, req.Accent)
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: group})
}

// ReorderWatchlistGroups persists a full watchlist group order for the authenticated user.
func (h *UserHandler) ReorderWatchlistGroups(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	var req reorderWatchlistGroupsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid watchlist reorder payload"},
		})
		return
	}

	if err := h.userPreferenceService.ReorderWatchlistGroups(c.Request.Context(), user.ID, req.GroupIDs); err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    gin.H{"group_ids": req.GroupIDs},
	})
}

// DeleteWatchlistGroup removes a watchlist group owned by the authenticated user.
func (h *UserHandler) DeleteWatchlistGroup(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	groupID := c.Param("groupId")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_GROUP_ID", Message: "Group ID is required"},
		})
		return
	}

	if err := h.userPreferenceService.DeleteWatchlistGroup(c.Request.Context(), user.ID, groupID); err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    gin.H{"group_id": groupID, "removed": true},
	})
}

// AddWatchlistFund adds a fund to a specific watchlist group.
func (h *UserHandler) AddWatchlistFund(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	groupID := c.Param("groupId")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_GROUP_ID", Message: "Group ID is required"},
		})
		return
	}

	var req watchlistFundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid watchlist fund payload"},
		})
		return
	}

	if err := h.userPreferenceService.AddWatchlistFund(c.Request.Context(), user.ID, groupID, req.FundID); err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    gin.H{"group_id": groupID, "fund_id": strings.TrimSpace(req.FundID)},
	})
}

// RemoveWatchlistFund removes a fund from a watchlist group.
func (h *UserHandler) RemoveWatchlistFund(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	groupID := c.Param("groupId")
	fundID := c.Param("fundId")
	if groupID == "" || fundID == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Group ID and Fund ID are required"},
		})
		return
	}

	if err := h.userPreferenceService.RemoveWatchlistFund(c.Request.Context(), user.ID, groupID, fundID); err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    gin.H{"group_id": groupID, "fund_id": fundID, "removed": true},
	})
}

// ListFundHoldings returns the authenticated user's fund-level position records.
func (h *UserHandler) ListFundHoldings(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	holdings, err := h.userPreferenceService.ListFundHoldings(c.Request.Context(), user.ID)
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: holdings})
}

// ListFundHoldingTransactions returns the authenticated user's recent holding activity.
func (h *UserHandler) ListFundHoldingTransactions(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	limit := 0
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit < 0 {
			c.JSON(http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   &APIError{Code: "INVALID_LIMIT", Message: "Invalid transaction limit"},
			})
			return
		}
		limit = parsedLimit
	}

	offset := 0
	if rawOffset := strings.TrimSpace(c.Query("offset")); rawOffset != "" {
		parsedOffset, err := strconv.Atoi(rawOffset)
		if err != nil || parsedOffset < 0 {
			c.JSON(http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   &APIError{Code: "INVALID_OFFSET", Message: "Invalid transaction offset"},
			})
			return
		}
		offset = parsedOffset
	}

	filter := domain.UserFundHoldingTransactionFilter{
		Limit:          limit,
		Offset:         offset,
		FundID:         strings.TrimSpace(c.Query("fund_id")),
		SourcePlatform: strings.TrimSpace(c.Query("source_platform")),
		Keyword:        strings.TrimSpace(c.Query("keyword")),
	}
	if rawTypes := strings.TrimSpace(c.Query("type")); rawTypes != "" && rawTypes != "all" {
		for _, item := range strings.Split(rawTypes, ",") {
			txType := strings.TrimSpace(item)
			if txType == "" {
				continue
			}
			filter.Types = append(filter.Types, domain.UserFundHoldingTransactionType(txType))
		}
	}
	if rawVoided := strings.TrimSpace(c.Query("voided")); rawVoided != "" && rawVoided != "all" {
		parsedVoided, err := strconv.ParseBool(rawVoided)
		if err != nil {
			c.JSON(http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   &APIError{Code: "INVALID_VOIDED_FILTER", Message: "Invalid transaction voided filter"},
			})
			return
		}
		filter.Voided = &parsedVoided
	}
	if rawStart := strings.TrimSpace(c.Query("start_date")); rawStart != "" {
		createdFrom, err := parseHoldingTransactionBoundary(rawStart, false)
		if err != nil {
			c.JSON(http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   &APIError{Code: "INVALID_DATE_RANGE", Message: "Invalid transaction start date"},
			})
			return
		}
		filter.CreatedFrom = &createdFrom
	}
	if rawEnd := strings.TrimSpace(c.Query("end_date")); rawEnd != "" {
		createdBefore, err := parseHoldingTransactionBoundary(rawEnd, true)
		if err != nil {
			c.JSON(http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   &APIError{Code: "INVALID_DATE_RANGE", Message: "Invalid transaction end date"},
			})
			return
		}
		filter.CreatedBefore = &createdBefore
	}

	transactions, err := h.userPreferenceService.ListFundHoldingTransactionsFiltered(c.Request.Context(), user.ID, filter)
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: transactions})
}

func parseHoldingTransactionBoundary(raw string, endExclusive bool) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed, nil
	}
	parsedDate, err := time.ParseInLocation("2006-01-02", raw, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	if endExclusive {
		return parsedDate.AddDate(0, 0, 1), nil
	}
	return parsedDate, nil
}

// GetFundHoldingTransactionDetail returns a single holding activity with context for drill-down review.
func (h *UserHandler) GetFundHoldingTransactionDetail(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	transactionID := strings.TrimSpace(c.Param("transactionId"))
	if transactionID == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_TRANSACTION_ID", Message: "Transaction ID is required"},
		})
		return
	}

	detail, err := h.userPreferenceService.GetFundHoldingTransactionDetail(c.Request.Context(), user.ID, transactionID)
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: detail})
}

// VoidFundHoldingTransaction marks a historical holding activity as ignored without changing current holdings.
func (h *UserHandler) VoidFundHoldingTransaction(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	transactionID := strings.TrimSpace(c.Param("transactionId"))
	if transactionID == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_TRANSACTION_ID", Message: "Transaction ID is required"},
		})
		return
	}

	var req voidFundHoldingTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid fund holding transaction payload"},
		})
		return
	}

	transaction, err := h.userPreferenceService.VoidFundHoldingTransaction(c.Request.Context(), user.ID, transactionID, req.Reason)
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: transaction})
}

// PreviewFundHoldingTransactionRollback returns a read-only manual rollback impact preview.
func (h *UserHandler) PreviewFundHoldingTransactionRollback(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	transactionID := strings.TrimSpace(c.Param("transactionId"))
	if transactionID == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_TRANSACTION_ID", Message: "Transaction ID is required"},
		})
		return
	}

	preview, err := h.userPreferenceService.PreviewFundHoldingTransactionRollback(c.Request.Context(), user.ID, transactionID)
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: preview})
}

// ApplyFundHoldingTransactionRollback applies a user-confirmed safe rollback.
func (h *UserHandler) ApplyFundHoldingTransactionRollback(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	transactionID := strings.TrimSpace(c.Param("transactionId"))
	if transactionID == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_TRANSACTION_ID", Message: "Transaction ID is required"},
		})
		return
	}

	var req applyFundHoldingTransactionRollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid fund holding rollback payload"},
		})
		return
	}

	result, err := h.userPreferenceService.ApplyFundHoldingTransactionRollback(c.Request.Context(), user.ID, transactionID, req.Reason)
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: result})
}

// CreateFundHolding creates a user fund-level position record.
func (h *UserHandler) CreateFundHolding(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	var req createFundHoldingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid fund holding payload"},
		})
		return
	}

	tradeAt := strings.TrimSpace(req.TradeAt)
	if tradeAt == "" {
		tradeAt = strings.TrimSpace(req.AsOfDate)
	}

	holding, err := h.userPreferenceService.CreateFundHolding(c.Request.Context(), user.ID, req.FundID, req.Amount, tradeAt, req.Note)
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{Success: true, Data: holding})
}

// CreateFundHoldingsBatch safely creates multiple fund-level holding records.
func (h *UserHandler) CreateFundHoldingsBatch(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	var req createFundHoldingsBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid fund holding batch payload"},
		})
		return
	}

	inputs := make([]domain.CreateFundHoldingInput, 0, len(req.Items))
	for _, item := range req.Items {
		tradeAt := strings.TrimSpace(item.TradeAt)
		if tradeAt == "" {
			tradeAt = strings.TrimSpace(item.AsOfDate)
		}
		inputs = append(inputs, domain.CreateFundHoldingInput{
			FundID:         item.FundID,
			Amount:         item.Amount,
			TradeAt:        tradeAt,
			Note:           item.Note,
			SourcePlatform: item.SourcePlatform,
			SourceLabel:    item.SourceLabel,
		})
	}

	result, err := h.userPreferenceService.CreateFundHoldingsBatch(c.Request.Context(), user.ID, inputs)
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{Success: true, Data: result})
}

// UpdateFundHolding corrects a fund-level position record without recording a buy/sell transaction.
func (h *UserHandler) UpdateFundHolding(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	holdingID := c.Param("holdingId")
	if strings.TrimSpace(holdingID) == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_HOLDING_ID", Message: "Holding ID is required"},
		})
		return
	}

	var req updateFundHoldingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid fund holding payload"},
		})
		return
	}

	holding, err := h.userPreferenceService.UpdateFundHolding(c.Request.Context(), user.ID, holdingID, domain.UpdateFundHoldingInput{
		Amount:           req.Amount,
		Shares:           req.Shares,
		ConfirmedNav:     req.ConfirmedNav,
		ConfirmedNavDate: req.ConfirmedNavDate,
		TradeAt:          req.TradeAt,
		Note:             req.Note,
		SourcePlatform:   req.SourcePlatform,
		SourceLabel:      req.SourceLabel,
	})
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: holding})
}

// SellFundHolding records a redemption/decrease for a fund-level position.
func (h *UserHandler) SellFundHolding(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	holdingID := c.Param("holdingId")
	if strings.TrimSpace(holdingID) == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_HOLDING_ID", Message: "Holding ID is required"},
		})
		return
	}

	var req sellFundHoldingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid fund holding sell payload"},
		})
		return
	}

	holding, err := h.userPreferenceService.SellFundHolding(c.Request.Context(), user.ID, holdingID, domain.SellFundHoldingInput{
		Amount:  req.Amount,
		Shares:  req.Shares,
		TradeAt: req.TradeAt,
		Note:    req.Note,
		SellAll: req.SellAll,
	})
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: holding})
}

// RecordFundHoldingDividend records a cash dividend or dividend reinvestment for a fund-level position.
func (h *UserHandler) RecordFundHoldingDividend(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	holdingID := c.Param("holdingId")
	if strings.TrimSpace(holdingID) == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_HOLDING_ID", Message: "Holding ID is required"},
		})
		return
	}

	var req dividendFundHoldingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid fund holding dividend payload"},
		})
		return
	}

	holding, err := h.userPreferenceService.RecordFundHoldingDividend(c.Request.Context(), user.ID, holdingID, domain.DividendFundHoldingInput{
		Amount:         req.Amount,
		Shares:         req.Shares,
		TradeAt:        req.TradeAt,
		Note:           req.Note,
		Reinvest:       req.Reinvest,
		SourcePlatform: req.SourcePlatform,
		SourceLabel:    req.SourceLabel,
	})
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: holding})
}

// AdjustFundHoldingShares records a non-trade share adjustment for a fund-level position.
func (h *UserHandler) AdjustFundHoldingShares(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	holdingID := c.Param("holdingId")
	if strings.TrimSpace(holdingID) == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_HOLDING_ID", Message: "Holding ID is required"},
		})
		return
	}

	var req adjustFundHoldingSharesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid fund holding adjustment payload"},
		})
		return
	}

	holding, err := h.userPreferenceService.AdjustFundHoldingShares(c.Request.Context(), user.ID, holdingID, domain.AdjustFundHoldingSharesInput{
		SharesDelta:      req.SharesDelta,
		TargetShares:     req.TargetShares,
		ConfirmedNav:     req.ConfirmedNav,
		ConfirmedNavDate: req.ConfirmedNavDate,
		TradeAt:          req.TradeAt,
		Note:             req.Note,
		SourcePlatform:   req.SourcePlatform,
		SourceLabel:      req.SourceLabel,
	})
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: holding})
}

// DeleteFundHolding removes a fund-level position record.
func (h *UserHandler) DeleteFundHolding(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	holdingID := c.Param("holdingId")
	if holdingID == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_HOLDING_ID", Message: "Holding ID is required"},
		})
		return
	}

	if err := h.userPreferenceService.DeleteFundHolding(c.Request.Context(), user.ID, holdingID); err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    gin.H{"holding_id": holdingID, "removed": true},
	})
}

// ListFavoriteFunds returns the authenticated user's favorite funds.
func (h *UserHandler) ListFavoriteFunds(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "UNAUTHORIZED",
				Message: "Authentication required",
			},
		})
		return
	}

	favorites, err := h.userPreferenceService.ListFavoriteFunds(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "FETCH_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    favorites,
	})
}

// AddFavoriteFund adds a fund to the authenticated user's watchlist.
func (h *UserHandler) AddFavoriteFund(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "UNAUTHORIZED",
				Message: "Authentication required",
			},
		})
		return
	}

	var req addFavoriteFundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_REQUEST",
				Message: "Invalid favorite fund payload",
			},
		})
		return
	}

	if err := h.userPreferenceService.AddFavoriteFund(c.Request.Context(), user.ID, req.FundID); err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{
			Success: false,
			Error:   apiErr,
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data: gin.H{
			"fund_id": strings.TrimSpace(req.FundID),
		},
	})
}

// RemoveFavoriteFund removes a fund from the authenticated user's watchlist.
func (h *UserHandler) RemoveFavoriteFund(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "UNAUTHORIZED",
				Message: "Authentication required",
			},
		})
		return
	}

	fundID := c.Param("fundId")
	if fundID == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_FUND_ID",
				Message: "Fund ID is required",
			},
		})
		return
	}

	if err := h.userPreferenceService.RemoveFavoriteFund(c.Request.Context(), user.ID, fundID); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "DELETE_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: gin.H{
			"fund_id": fundID,
			"removed": true,
		},
	})
}

// GetHoldingOverrides returns the authenticated user's holding overrides for a fund.
func (h *UserHandler) GetHoldingOverrides(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "UNAUTHORIZED",
				Message: "Authentication required",
			},
		})
		return
	}

	fundID := c.Param("fundId")
	if fundID == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_FUND_ID",
				Message: "Fund ID is required",
			},
		})
		return
	}

	overrideSet, err := h.userPreferenceService.GetHoldingOverrideSet(c.Request.Context(), user.ID, fundID)
	if err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{
			Success: false,
			Error:   apiErr,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    overrideSet,
	})
}

// ReplaceHoldingOverrides replaces the authenticated user's holding overrides for a fund.
func (h *UserHandler) ReplaceHoldingOverrides(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "UNAUTHORIZED",
				Message: "Authentication required",
			},
		})
		return
	}

	fundID := c.Param("fundId")
	if fundID == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_FUND_ID",
				Message: "Fund ID is required",
			},
		})
		return
	}

	var req replaceHoldingOverridesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_REQUEST",
				Message: "Invalid holding override payload",
			},
		})
		return
	}

	overrides, err := parseHoldingOverrides(req.Overrides)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_OVERRIDE",
				Message: err.Error(),
			},
		})
		return
	}

	if err := h.userPreferenceService.ReplaceHoldingOverrides(c.Request.Context(), user.ID, fundID, overrides); err != nil {
		statusCode, apiErr := mapUserPreferenceError(err)
		c.JSON(statusCode, APIResponse{
			Success: false,
			Error:   apiErr,
		})
		return
	}

	overrideSet, err := h.userPreferenceService.GetHoldingOverrideSet(c.Request.Context(), user.ID, fundID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "FETCH_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    overrideSet,
	})
}

func parseHoldingOverrides(raw []holdingOverrideRequest) ([]domain.UserHoldingOverride, error) {
	result := make([]domain.UserHoldingOverride, 0, len(raw))
	for _, item := range raw {
		ratio, err := decimal.NewFromString(strings.TrimSpace(item.HoldingRatio))
		if err != nil {
			return nil, service.ErrInvalidHoldingOverride
		}

		result = append(result, domain.UserHoldingOverride{
			ID:           strings.TrimSpace(item.ID),
			StockCode:    strings.TrimSpace(item.StockCode),
			StockName:    strings.TrimSpace(item.StockName),
			Exchange:     domain.Exchange(strings.ToUpper(strings.TrimSpace(item.Exchange))),
			HoldingRatio: ratio,
			Note:         strings.TrimSpace(item.Note),
		})
	}
	return result, nil
}

func mapUserPreferenceError(err error) (int, *APIError) {
	switch {
	case errors.Is(err, service.ErrFundNotFound):
		return http.StatusNotFound, &APIError{Code: "FUND_NOT_FOUND", Message: err.Error()}
	case errors.Is(err, service.ErrWatchlistGroupNotFound):
		return http.StatusNotFound, &APIError{Code: "WATCHLIST_GROUP_NOT_FOUND", Message: err.Error()}
	case errors.Is(err, service.ErrFundHoldingNotFound):
		return http.StatusNotFound, &APIError{Code: "FUND_HOLDING_NOT_FOUND", Message: err.Error()}
	case errors.Is(err, service.ErrFundHoldingTransactionNotFound):
		return http.StatusNotFound, &APIError{Code: "FUND_HOLDING_TRANSACTION_NOT_FOUND", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidWatchlistGroup):
		return http.StatusBadRequest, &APIError{Code: "INVALID_WATCHLIST_GROUP", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidWatchlistOrder):
		return http.StatusBadRequest, &APIError{Code: "INVALID_WATCHLIST_ORDER", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidHoldingAmount):
		return http.StatusBadRequest, &APIError{Code: "INVALID_HOLDING_AMOUNT", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidHoldingDate):
		return http.StatusBadRequest, &APIError{Code: "INVALID_HOLDING_DATE", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidHoldingTime):
		return http.StatusBadRequest, &APIError{Code: "INVALID_HOLDING_TIME", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidHoldingSell):
		return http.StatusBadRequest, &APIError{Code: "INVALID_HOLDING_SELL", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidHoldingDividend):
		return http.StatusBadRequest, &APIError{Code: "INVALID_HOLDING_DIVIDEND", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidHoldingAdjustment):
		return http.StatusBadRequest, &APIError{Code: "INVALID_HOLDING_ADJUSTMENT", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidHoldingTransactionFilter):
		return http.StatusBadRequest, &APIError{Code: "INVALID_HOLDING_TRANSACTION_FILTER", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidHoldingSource):
		return http.StatusBadRequest, &APIError{Code: "INVALID_HOLDING_SOURCE", Message: err.Error()}
	case errors.Is(err, service.ErrFundHoldingTransactionVoided):
		return http.StatusConflict, &APIError{Code: "FUND_HOLDING_TRANSACTION_VOIDED", Message: err.Error()}
	case errors.Is(err, service.ErrUnsafeHoldingRollback):
		return http.StatusConflict, &APIError{Code: "UNSAFE_HOLDING_ROLLBACK", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidHoldingOverride):
		return http.StatusBadRequest, &APIError{Code: "INVALID_OVERRIDE", Message: err.Error()}
	default:
		return http.StatusInternalServerError, &APIError{Code: "USER_PREFERENCE_FAILED", Message: err.Error()}
	}
}
