// Package handler contains HTTP handlers for the API.
package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/RomaticDOG/fund/internal/trading"
	"github.com/gin-gonic/gin"
)

type transientFundDataLoader interface {
	PeekCachedFundData(fundID string) (*domain.Fund, []domain.StockHolding, bool)
	ScheduleEnsureFundData(fundID string) bool
}

type holdingsFallbackResolver interface {
	GetHoldingsWithFallback(ctx context.Context, fundID string, fundName string) ([]domain.StockHolding, string, error)
}

// FundHandler handles fund-related HTTP requests.
type FundHandler struct {
	valuationService domain.ValuationService
	fundRepo         domain.FundRepository
	dataLoader       transientFundDataLoader
	holdingsResolver holdingsFallbackResolver
}

// NewFundHandler creates a new FundHandler instance.
func NewFundHandler(
	valuationService domain.ValuationService,
	fundRepo domain.FundRepository,
	holdingsResolver holdingsFallbackResolver,
) *FundHandler {
	return &FundHandler{
		valuationService: valuationService,
		fundRepo:         fundRepo,
		dataLoader:       service.NewFundDataLoader(fundRepo),
		holdingsResolver: holdingsResolver,
	}
}

// SetTransientFundDataLoader overrides the transient fund data loader used by read-only fund endpoints.
func (h *FundHandler) SetTransientFundDataLoader(loader *service.FundDataLoader) {
	if h != nil && loader != nil {
		h.dataLoader = loader
	}
}

// APIResponse represents a standard API response structure.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *APIMeta    `json:"meta,omitempty"`
}

// APIError represents an API error.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// APIMeta contains metadata about the response.
type APIMeta struct {
	DataSource  string `json:"data_source,omitempty"`
	CacheStatus string `json:"cache_status,omitempty"`
}

type OfficialCloseDisplayStatus string

const (
	OfficialCloseDisplayHidden  OfficialCloseDisplayStatus = "hidden"
	OfficialCloseDisplayPending OfficialCloseDisplayStatus = "pending"
	OfficialCloseDisplayReady   OfficialCloseDisplayStatus = "ready"
)

type OfficialCloseInfo struct {
	DisplayStatus OfficialCloseDisplayStatus `json:"display_status"`
	Date          string                     `json:"date,omitempty"`
	DailyReturn   string                     `json:"daily_return,omitempty"`
	NetAssetVal   string                     `json:"net_asset_val,omitempty"`
	Message       string                     `json:"message,omitempty"`
}

// Search handles fund search requests.
// GET /api/v1/fund/search?q=000001
func (h *FundHandler) Search(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_QUERY",
				Message: "Search query 'q' is required",
			},
		})
		return
	}

	limit := 20 // Default limit
	funds, err := h.fundRepo.SearchFunds(c.Request.Context(), query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "SEARCH_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    funds,
	})
}

// GetEstimate handles real-time fund valuation estimate requests.
// GET /api/v1/fund/:id/estimate
func (h *FundHandler) GetEstimate(c *gin.Context) {
	fundID := c.Param("id")
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

	estimate, err := h.valuationService.CalculateEstimate(c.Request.Context(), fundID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "ESTIMATE_FAILED"

		if errors.Is(err, service.ErrFundDataWarmupInProgress) {
			statusCode = http.StatusServiceUnavailable
			errorCode = "FUND_DATA_WARMING"
			c.Header("Retry-After", "5")
		} else if strings.Contains(err.Error(), "pricing profile not configured") || strings.Contains(err.Error(), "unsupported pricing method") {
			statusCode = http.StatusUnprocessableEntity
			errorCode = "UNSUPPORTED_PRICING_MODEL"
		} else if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "FUND_NOT_FOUND"
		}

		c.JSON(statusCode, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    errorCode,
				Message: err.Error(),
			},
		})
		return
	}

	officialClose := h.resolveOfficialCloseInfo(c.Request.Context(), fundID, trading.GetMarketStatus(time.Now()))

	type EstimateResponse struct {
		*domain.FundEstimate
		OfficialClose *OfficialCloseInfo `json:"official_close,omitempty"`
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: EstimateResponse{
			FundEstimate:  estimate,
			OfficialClose: officialClose,
		},
		Meta: &APIMeta{
			DataSource: estimate.DataSource,
		},
	})
}

// GetHoldings handles fund holdings requests.
// GET /api/v1/fund/:id/holdings
func (h *FundHandler) GetHoldings(c *gin.Context) {
	fundID := c.Param("id")
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

	// First check if fund exists
	fund, err := h.fundRepo.GetFundByID(c.Request.Context(), fundID)
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

	if fund == nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "FUND_NOT_FOUND",
				Message: "Fund not found: " + fundID,
			},
		})
		return
	}

	holdings, err := h.fundRepo.GetFundHoldings(c.Request.Context(), fundID)
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

	cacheStatus := ""
	if shouldHydrateFundHoldings(fund, holdings) {
		hydratedFund, hydratedHoldings, status := h.cachedFundDataOrScheduleWarmup(fundID)
		cacheStatus = status
		if hydratedFund != nil {
			fund = hydratedFund
		}
		if len(hydratedHoldings) > 0 {
			holdings = hydratedHoldings
		}
	}
	dataSource := ""

	if len(holdings) == 0 && h.holdingsResolver != nil {
		resolvedHoldings, holdingsSource, resolveErr := h.holdingsResolver.GetHoldingsWithFallback(c.Request.Context(), fundID, fund.Name)
		if resolveErr != nil {
			log.Printf("⚠️ Holdings resolver fallback failed for %s: %v", fundID, resolveErr)
		} else if len(resolvedHoldings) > 0 {
			holdings = resolvedHoldings
			if holdingsSource != "" && holdingsSource != fundID {
				dataSource = "target_etf:" + holdingsSource
			}
		}
	}

	type HoldingsResponse struct {
		Fund     *domain.Fund          `json:"fund"`
		Holdings []domain.StockHolding `json:"holdings"`
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: HoldingsResponse{
			Fund:     fund,
			Holdings: holdings,
		},
		Meta: buildResponseMeta(dataSource, cacheStatus),
	})
}

// GetTimeSeries handles intraday time series requests.
// GET /api/v1/fund/:id/timeseries
func (h *FundHandler) GetTimeSeries(c *gin.Context) {
	fundID := c.Param("id")
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

	// Get current market status
	now := time.Now()
	marketStatus := trading.GetMarketStatus(now)

	timeSeries, err := h.valuationService.GetIntradayTimeSeries(c.Request.Context(), fundID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "FETCH_FAILED"
		if errors.Is(err, service.ErrFundDataWarmupInProgress) {
			statusCode = http.StatusServiceUnavailable
			errorCode = "FUND_DATA_WARMING"
			c.Header("Retry-After", "5")
		}
		c.JSON(statusCode, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    errorCode,
				Message: err.Error(),
			},
		})
		return
	}

	// Determine if we're showing historical data (not from today)
	isHistorical := false
	dataDate := marketStatus.DisplayDate
	if len(timeSeries) > 0 {
		// Check if the first point's date differs from today
		firstPointDate := timeSeries[0].Timestamp.In(trading.TradingLocation()).Format("2006-01-02")
		todayDate := marketStatus.CurrentDate
		if firstPointDate != todayDate {
			isHistorical = true
			dataDate = firstPointDate
		}
	}

	// Enhanced response with market context
	type TimeSeriesResponse struct {
		Points         []domain.TimeSeriesPoint `json:"points"`
		DisplayDate    string                   `json:"display_date"`
		IsTrading      bool                     `json:"is_trading"`
		IsHistorical   bool                     `json:"is_historical"`
		Session        trading.SessionType      `json:"session"`
		LastTradingDay string                   `json:"last_trading_day"`
		OfficialClose  *OfficialCloseInfo       `json:"official_close,omitempty"`
	}

	officialClose := h.resolveOfficialCloseInfo(c.Request.Context(), fundID, marketStatus)

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: TimeSeriesResponse{
			Points:         timeSeries,
			DisplayDate:    dataDate,
			IsTrading:      marketStatus.IsTrading,
			IsHistorical:   isHistorical,
			Session:        marketStatus.Session,
			LastTradingDay: marketStatus.LastTradingDay,
			OfficialClose:  officialClose,
		},
	})
}

// GetMarketStatus returns the current A-Share market status.
// GET /api/v1/market/status
func (h *FundHandler) GetMarketStatus(c *gin.Context) {
	now := time.Now()
	status := trading.GetMarketStatus(now)

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    status,
	})
}

// GetPricingDatePreview resolves the confirmed NAV date for a proposed trade timestamp.
// GET /api/v1/market/pricing-date?trade_at=2026-03-31T14:59:00%2B08:00
func (h *FundHandler) GetPricingDatePreview(c *gin.Context) {
	rawTradeAt := strings.TrimSpace(c.Query("trade_at"))
	if rawTradeAt == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_TRADE_AT",
				Message: "Query parameter 'trade_at' is required",
			},
		})
		return
	}

	tradeAt, err := trading.ParseTradeAt(rawTradeAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_TRADE_AT",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    trading.ResolvePricingDate(tradeAt),
	})
}

// GetFund handles fund info requests.
// GET /api/v1/fund/:id
func (h *FundHandler) GetFund(c *gin.Context) {
	fundID := c.Param("id")
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

	fund, err := h.fundRepo.GetFundByID(c.Request.Context(), fundID)
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

	if fund == nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "FUND_NOT_FOUND",
				Message: "Fund not found: " + fundID,
			},
		})
		return
	}

	cacheStatus := ""
	fund, cacheStatus = h.hydrateFundProfile(fundID, fund)

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    fund,
		Meta:    buildResponseMeta("", cacheStatus),
	})
}

func (h *FundHandler) hydrateFundProfile(fundID string, fund *domain.Fund) (*domain.Fund, string) {
	if !shouldHydrateFundProfile(fund) {
		return fund, ""
	}

	hydratedFund, _, cacheStatus := h.cachedFundDataOrScheduleWarmup(fundID)
	if hydratedFund != nil {
		return hydratedFund, cacheStatus
	}
	return fund, cacheStatus
}

func (h *FundHandler) cachedFundDataOrScheduleWarmup(fundID string) (*domain.Fund, []domain.StockHolding, string) {
	if h == nil || h.dataLoader == nil {
		return nil, nil, ""
	}
	if fund, holdings, ok := h.dataLoader.PeekCachedFundData(fundID); ok {
		return fund, holdings, "warm_cache"
	}
	if h.dataLoader.ScheduleEnsureFundData(fundID) {
		return nil, nil, "warming"
	}
	return nil, nil, ""
}

func shouldHydrateFundProfile(fund *domain.Fund) bool {
	if fund == nil {
		return false
	}
	return fund.NetAssetVal.IsZero() || strings.TrimSpace(fund.Manager) == "" || strings.TrimSpace(fund.Company) == ""
}

func shouldHydrateFundHoldings(fund *domain.Fund, holdings []domain.StockHolding) bool {
	return shouldHydrateFundProfile(fund) || len(holdings) == 0
}

func buildResponseMeta(dataSource, cacheStatus string) *APIMeta {
	dataSource = strings.TrimSpace(dataSource)
	cacheStatus = strings.TrimSpace(cacheStatus)
	if dataSource == "" && cacheStatus == "" {
		return nil
	}
	return &APIMeta{
		DataSource:  dataSource,
		CacheStatus: cacheStatus,
	}
}

func (h *FundHandler) resolveOfficialCloseInfo(ctx context.Context, fundID string, marketStatus trading.MarketStatus) *OfficialCloseInfo {
	history, err := h.fundRepo.GetLatestFundHistory(ctx, fundID)
	if err != nil {
		log.Printf("⚠️ Official close info lookup failed for %s: %v", fundID, err)
		return &OfficialCloseInfo{DisplayStatus: OfficialCloseDisplayHidden}
	}

	switch marketStatus.Session {
	case trading.SessionAfterHours:
		if marketStatus.IsTradingDay {
			if history != nil && history.Date == marketStatus.CurrentDate {
				return &OfficialCloseInfo{
					DisplayStatus: OfficialCloseDisplayReady,
					Date:          history.Date,
					DailyReturn:   history.DailyReturn.String(),
					NetAssetVal:   history.NetAssetVal.String(),
				}
			}
			return &OfficialCloseInfo{
				DisplayStatus: OfficialCloseDisplayPending,
				Message:       "真实涨跌情况稍后更新",
			}
		}
	case trading.SessionPreMarket, trading.SessionWeekend, trading.SessionHoliday:
		if history != nil && history.Date == marketStatus.LastTradingDay {
			return &OfficialCloseInfo{
				DisplayStatus: OfficialCloseDisplayReady,
				Date:          history.Date,
				DailyReturn:   history.DailyReturn.String(),
				NetAssetVal:   history.NetAssetVal.String(),
			}
		}
	}

	return &OfficialCloseInfo{DisplayStatus: OfficialCloseDisplayHidden}
}
