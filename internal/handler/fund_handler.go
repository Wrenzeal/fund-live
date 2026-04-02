// Package handler contains HTTP handlers for the API.
package handler

import (
	"context"
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
	FetchTransientFundData(ctx context.Context, fundID string) (*domain.Fund, []domain.StockHolding, error)
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

		if strings.Contains(err.Error(), "pricing profile not configured") || strings.Contains(err.Error(), "unsupported pricing method") {
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

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    estimate,
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

	if shouldHydrateFundHoldings(fund, holdings) {
		hydratedFund, hydratedHoldings, hydrateErr := h.fetchTransientFundData(c.Request.Context(), fundID)
		if hydrateErr != nil {
			log.Printf("⚠️ Transient fund holdings hydration failed for %s: %v", fundID, hydrateErr)
		} else {
			if hydratedFund != nil {
				fund = hydratedFund
			}
			if len(hydratedHoldings) > 0 {
				holdings = hydratedHoldings
			}
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
		Meta: buildDataSourceMeta(dataSource),
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
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "FETCH_FAILED",
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
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: TimeSeriesResponse{
			Points:         timeSeries,
			DisplayDate:    dataDate,
			IsTrading:      marketStatus.IsTrading,
			IsHistorical:   isHistorical,
			Session:        marketStatus.Session,
			LastTradingDay: marketStatus.LastTradingDay,
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

	fund = h.hydrateFundProfile(c.Request.Context(), fundID, fund)

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    fund,
	})
}

func (h *FundHandler) hydrateFundProfile(ctx context.Context, fundID string, fund *domain.Fund) *domain.Fund {
	if !shouldHydrateFundProfile(fund) {
		return fund
	}

	hydratedFund, _, err := h.fetchTransientFundData(ctx, fundID)
	if err != nil {
		log.Printf("⚠️ Transient fund profile hydration failed for %s: %v", fundID, err)
		return fund
	}
	if hydratedFund != nil {
		return hydratedFund
	}
	return fund
}

func (h *FundHandler) fetchTransientFundData(ctx context.Context, fundID string) (*domain.Fund, []domain.StockHolding, error) {
	if h == nil || h.dataLoader == nil {
		return nil, nil, nil
	}
	return h.dataLoader.FetchTransientFundData(ctx, fundID)
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

func buildDataSourceMeta(dataSource string) *APIMeta {
	dataSource = strings.TrimSpace(dataSource)
	if dataSource == "" {
		return nil
	}
	return &APIMeta{DataSource: dataSource}
}
