package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/middleware"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// UserHandler handles authenticated user preference requests.
type UserHandler struct {
	userPreferenceService domain.UserPreferenceService
}

// NewUserHandler creates a new UserHandler instance.
func NewUserHandler(userPreferenceService domain.UserPreferenceService) *UserHandler {
	return &UserHandler{
		userPreferenceService: userPreferenceService,
	}
}

type addFavoriteFundRequest struct {
	FundID string `json:"fund_id"`
}

type createWatchlistGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type watchlistFundRequest struct {
	FundID string `json:"fund_id"`
}

type createFundHoldingRequest struct {
	FundID   string `json:"fund_id"`
	Amount   string `json:"amount"`
	AsOfDate string `json:"as_of_date"`
	TradeAt  string `json:"trade_at"`
	Note     string `json:"note"`
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
	case errors.Is(err, service.ErrInvalidWatchlistGroup):
		return http.StatusBadRequest, &APIError{Code: "INVALID_WATCHLIST_GROUP", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidHoldingAmount):
		return http.StatusBadRequest, &APIError{Code: "INVALID_HOLDING_AMOUNT", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidHoldingDate):
		return http.StatusBadRequest, &APIError{Code: "INVALID_HOLDING_DATE", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidHoldingTime):
		return http.StatusBadRequest, &APIError{Code: "INVALID_HOLDING_TIME", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidHoldingOverride):
		return http.StatusBadRequest, &APIError{Code: "INVALID_OVERRIDE", Message: err.Error()}
	default:
		return http.StatusInternalServerError, &APIError{Code: "USER_PREFERENCE_FAILED", Message: err.Error()}
	}
}
