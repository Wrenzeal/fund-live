// Package handler contains HTTP handlers for the API.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/RomaticDOG/fund/internal/trading"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

type transientFundDataLoader interface {
	PeekCachedFundData(fundID string) (*domain.Fund, []domain.StockHolding, bool)
	ScheduleEnsureFundData(fundID string) bool
}

type holdingsFallbackResolver interface {
	GetHoldingsWithFallback(ctx context.Context, fundID string, fundName string) ([]domain.StockHolding, string, error)
}

type holdingsDisplayResolver interface {
	ResolveDisplayHoldings(ctx context.Context, fundID string, fundName string) (*domain.FundHoldingsDisplay, error)
}

type analysisRankingCandidateProvider interface {
	ListRankingCandidateFundIDs(ctx context.Context, limit int) ([]string, error)
}

type analysisSnapshotReader interface {
	Get(ctx context.Context, fundID string) (*domain.FundAnalysis, time.Time, error)
	GetByFundIDs(ctx context.Context, fundIDs []string) (map[string]*domain.FundAnalysis, error)
	ListRankings(ctx context.Context, limit int) (increase, watch, risk []database.FundAnalysisSnapshot, err error)
	Save(ctx context.Context, fundID string, analysis *domain.FundAnalysis, generatedAt time.Time) error
}

type fundSectorSnapshotStore interface {
	UpsertFromHoldings(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundSectorSnapshot, error)
	UpsertThemeFromHoldings(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundThemeSnapshot, error)
	BuildSnapshot(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundSectorSnapshot, error)
	BuildThemeSnapshot(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundThemeSnapshot, error)
	GetLatestSnapshot(ctx context.Context, fundID string) (*domain.FundSectorSnapshot, error)
	GetLatestThemeSnapshot(ctx context.Context, fundID string) (*domain.FundThemeSnapshot, error)
	ResolveFundCategory(ctx context.Context, fund *domain.Fund, snapshot *domain.FundSectorSnapshot) (*domain.FundCategory, error)
}

// FundHandler handles fund-related HTTP requests.
type FundHandler struct {
	valuationService domain.ValuationService
	fundRepo         domain.FundRepository
	dataLoader       transientFundDataLoader
	holdingsResolver holdingsFallbackResolver
	sectorStore      fundSectorSnapshotStore
	rankingCandidates analysisRankingCandidateProvider
	snapshotStore    analysisSnapshotReader
	coordinator      *service.FundAnalysisCoordinator
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

func (h *FundHandler) SetFundSectorStore(store *service.FundSectorStore) {
	if h != nil && store != nil {
		h.sectorStore = store
	}
}

func (h *FundHandler) SetAnalysisRankingCandidateProvider(provider analysisRankingCandidateProvider) {
	if h != nil && provider != nil {
		h.rankingCandidates = provider
	}
}

func (h *FundHandler) SetAnalysisSnapshotStore(store analysisSnapshotReader) {
	if h != nil && store != nil {
		h.snapshotStore = store
	}
}

func (h *FundHandler) SetAnalysisCoordinator(coordinator *service.FundAnalysisCoordinator) {
	if h != nil && coordinator != nil {
		h.coordinator = coordinator
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

type EstimateResponse struct {
	*domain.FundEstimate
	OfficialClose *OfficialCloseInfo `json:"official_close,omitempty"`
}

type FundDashboardResponse struct {
	Fund           *domain.Fund               `json:"fund,omitempty"`
	Estimate       *EstimateResponse          `json:"estimate,omitempty"`
	Analysis       *domain.FundAnalysis       `json:"analysis,omitempty"`
	SectorSnapshot *domain.FundSectorSnapshot `json:"sector_snapshot,omitempty"`
	ThemeSnapshot  *domain.FundThemeSnapshot  `json:"theme_snapshot,omitempty"`
	TimeSeries     []domain.TimeSeriesPoint   `json:"time_series"`
	DisplayDate    string                     `json:"display_date"`
	IsTrading      bool                       `json:"is_trading"`
	IsHistorical   bool                       `json:"is_historical"`
	Session        trading.SessionType        `json:"session"`
	LastTradingDay string                     `json:"last_trading_day"`
}

type FundAnalysisResponse struct {
	Fund     *domain.Fund         `json:"fund,omitempty"`
	Analysis *domain.FundAnalysis `json:"analysis,omitempty"`
}

type AnalysisRankingItem struct {
	Fund     *domain.Fund         `json:"fund,omitempty"`
	Analysis *domain.FundAnalysis `json:"analysis,omitempty"`
}

type AnalysisRankingsResponse struct {
	GeneratedAt   time.Time             `json:"generated_at"`
	IncreaseIdeas []AnalysisRankingItem `json:"increase_ideas"`
	WatchIdeas    []AnalysisRankingItem `json:"watch_ideas"`
	RiskAlerts    []AnalysisRankingItem `json:"risk_alerts"`
}

type AnalysisBatchResponse struct {
	Analyses map[string]*domain.FundAnalysis `json:"analyses"`
}

type analysisRankingCandidate struct {
	fund     *domain.Fund
	analysis *domain.FundAnalysis
}

// Search handles fund search requests.
// GET /api/v1/fund/search?q=000001
func (h *FundHandler) Search(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	categoryFilter := strings.TrimSpace(c.Query("category"))
	sectorFilter := strings.TrimSpace(c.Query("sector"))
	if query == "" {
		if categoryFilter != "" || sectorFilter != "" {
			c.JSON(http.StatusBadRequest, APIResponse{
				Success: false,
				Error: &APIError{
					Code:    "INVALID_QUERY",
					Message: "Search query 'q' is required when using filters",
				},
			})
			return
		}
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

	filteredFunds := make([]*domain.Fund, 0, len(funds))
	for _, fund := range funds {
		if fund == nil {
			continue
		}
		snapshot, sectorErr := h.buildFundSectorSnapshot(c.Request.Context(), fund.ID, fund)
		if sectorErr != nil {
			log.Printf("⚠️ Search sector snapshot failed for %s: %v", fund.ID, sectorErr)
		}
		if h.sectorStore != nil {
			category, categoryErr := h.resolveFundCategory(c.Request.Context(), fund, snapshot)
			if categoryErr != nil {
				log.Printf("⚠️ Search fund category resolve failed for %s: %v", fund.ID, categoryErr)
			} else if category != nil {
				fund.CategoryCode = category.Code
				fund.CategoryName = category.Name
			}
		}

		if categoryFilter != "" && !strings.EqualFold(strings.TrimSpace(fund.CategoryCode), categoryFilter) {
			continue
		}
		if sectorFilter != "" {
			if snapshot == nil {
				continue
			}
			matchedSector := strings.EqualFold(strings.TrimSpace(snapshot.PrimarySectorCode), sectorFilter)
			if !matchedSector {
				for _, item := range snapshot.Breakdown {
					if strings.EqualFold(strings.TrimSpace(item.SectorCode), sectorFilter) {
						matchedSector = true
						break
					}
				}
			}
			if !matchedSector {
				continue
			}
		}

		filteredFunds = append(filteredFunds, fund)
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    filteredFunds,
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
		} else if strings.Contains(err.Error(), "qdii details available without live estimate support") {
			statusCode = http.StatusUnprocessableEntity
			errorCode = "UNSUPPORTED_PRICING_MODEL"
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

// GetDashboard returns a unified homepage snapshot so the estimate card and intraday chart share one snapshot source.
// GET /api/v1/fund/:id/dashboard
func (h *FundHandler) GetDashboard(c *gin.Context) {
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

	now := time.Now()
	marketStatus := trading.GetMarketStatus(now)

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
	officialClose := h.resolveOfficialCloseInfo(c.Request.Context(), fundID, marketStatus)

	var estimateResponse *EstimateResponse
	estimate, estimateErr := h.valuationService.CalculateEstimate(c.Request.Context(), fundID)
	if estimateErr != nil && !errors.Is(estimateErr, service.ErrFundDataWarmupInProgress) {
		statusCode, apiErr := mapEstimateErrorToResponse(estimateErr)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}
	if estimateErr == nil && estimate != nil {
		estimateResponse = &EstimateResponse{
			FundEstimate:  estimate,
			OfficialClose: officialClose,
		}
	} else if errors.Is(estimateErr, service.ErrFundDataWarmupInProgress) {
		cacheStatus = "warming"
	}

	timeSeries, timeSeriesErr := h.valuationService.GetIntradayTimeSeries(c.Request.Context(), fundID)
	if timeSeriesErr != nil && !errors.Is(timeSeriesErr, service.ErrFundDataWarmupInProgress) {
		statusCode, apiErr := mapEstimateErrorToResponse(timeSeriesErr)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}
	if errors.Is(timeSeriesErr, service.ErrFundDataWarmupInProgress) {
		cacheStatus = "warming"
		timeSeries = []domain.TimeSeriesPoint{}
	}

	displayDate, isHistorical := deriveTimeSeriesDisplayContext(timeSeries, marketStatus)
	if estimate != nil {
		timeSeries = alignTimeSeriesWithEstimateSnapshot(timeSeries, estimate, displayDate, marketStatus)
		if alignedDate := displayDateFromTimeSeries(timeSeries); !alignedDate.IsZero() {
			displayDate = alignedDate.In(trading.TradingLocation()).Format("2006-01-02")
			isHistorical = displayDate != marketStatus.CurrentDate
		}
	}

	sectorSnapshot, sectorErr := h.buildFundSectorSnapshot(c.Request.Context(), fundID, fund)
	if sectorErr != nil {
		log.Printf("⚠️ Failed to build fund sector snapshot for %s: %v", fundID, sectorErr)
	}
	themeSnapshot, themeErr := h.buildFundThemeSnapshot(c.Request.Context(), fundID, fund)
	if themeErr != nil {
		log.Printf("⚠️ Failed to load fund theme snapshot for %s: %v", fundID, themeErr)
	}
	category, categoryErr := h.resolveFundCategory(c.Request.Context(), fund, sectorSnapshot)
	if categoryErr != nil {
		log.Printf("⚠️ Failed to resolve fund category for %s: %v", fundID, categoryErr)
	} else if category != nil {
		fund.CategoryCode = category.Code
		fund.CategoryName = category.Name
	}
	var analysis *domain.FundAnalysis
	if estimate != nil {
		analysis, err = h.buildFundAnalysis(c.Request.Context(), fundID, fund, estimate, timeSeries, sectorSnapshot, themeSnapshot, now)
		if err != nil {
			log.Printf("⚠️ Failed to build fund analysis for %s: %v", fundID, err)
		}
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: FundDashboardResponse{
			Fund:           fund,
			Estimate:       estimateResponse,
			Analysis:       analysis,
			SectorSnapshot: sectorSnapshot,
			ThemeSnapshot:  themeSnapshot,
			TimeSeries:     timeSeries,
			DisplayDate:    displayDate,
			IsTrading:      marketStatus.IsTrading,
			IsHistorical:   isHistorical,
			Session:        marketStatus.Session,
			LastTradingDay: marketStatus.LastTradingDay,
		},
		Meta: buildResponseMeta("", cacheStatus),
	})
}

// GetAnalysis returns standalone quant analysis so rankings/holdings/watchlist can reuse one analysis surface.
// GET /api/v1/fund/:id/analysis
func (h *FundHandler) GetAnalysis(c *gin.Context) {
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

	now := time.Now()
	fund, analysis, err := h.buildSingleFundAnalysis(c.Request.Context(), fundID, now)
	if err != nil {
		statusCode, apiErr := mapEstimateErrorToResponse(err)
		if apiErr.Code == "FETCH_FAILED" {
			apiErr.Code = "ANALYSIS_FAILED"
		}
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
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
	if h.snapshotStore != nil && analysis != nil {
		_ = h.snapshotStore.Save(c.Request.Context(), fundID, analysis, now)
	}
	dataSource := ""
	if h.valuationService != nil {
		if estimate, estimateErr := h.valuationService.CalculateEstimate(c.Request.Context(), fundID); estimateErr == nil && estimate != nil {
			dataSource = estimate.DataSource
		}
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: FundAnalysisResponse{
			Fund:     fund,
			Analysis: analysis,
		},
		Meta: buildResponseMeta(dataSource, ""),
	})
}

// GetAnalysisBatch returns snapshot-backed analyses for a list of funds, computing missing ones on demand.
// GET /api/v1/analysis/batch?fund_ids=000001,000002
func (h *FundHandler) GetAnalysisBatch(c *gin.Context) {
	rawIDs := strings.Split(strings.TrimSpace(c.Query("fund_ids")), ",")
	fundIDs := uniqueStrings(rawIDs)
	if len(fundIDs) == 0 {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: &APIError{Code: "INVALID_FUND_IDS", Message: "fund_ids is required"}})
		return
	}

	result := make(map[string]*domain.FundAnalysis, len(fundIDs))
	missing := append([]string(nil), fundIDs...)
	if h.snapshotStore != nil {
		snapshots, err := h.snapshotStore.GetByFundIDs(c.Request.Context(), fundIDs)
		if err == nil {
			missing = missing[:0]
			for _, fundID := range fundIDs {
				if analysis := snapshots[fundID]; analysis != nil {
					result[fundID] = analysis
					continue
				}
				missing = append(missing, fundID)
			}
		}
	}

	now := time.Now()
	for _, fundID := range missing {
		_, analysis, err := h.buildSingleFundAnalysis(c.Request.Context(), fundID, now)
		if err != nil || analysis == nil {
			result[fundID] = nil
			continue
		}
		result[fundID] = analysis
		if h.snapshotStore != nil {
			_ = h.snapshotStore.Save(c.Request.Context(), fundID, analysis, now)
		}
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: AnalysisBatchResponse{Analyses: result}})
}

// GetAnalysisRankings returns a lightweight leaderboard built from the same standalone analysis surface.
// GET /api/v1/analysis/rankings
func (h *FundHandler) GetAnalysisRankings(c *gin.Context) {
	now := time.Now()
	if h.snapshotStore != nil {
		increaseRows, watchRows, riskRows, err := h.snapshotStore.ListRankings(c.Request.Context(), 12)
		if err == nil && len(increaseRows) > 0 {
			fundIDs := make([]string, 0, len(increaseRows)+len(watchRows)+len(riskRows))
			for _, row := range increaseRows {
				fundIDs = append(fundIDs, row.FundID)
			}
			for _, row := range watchRows {
				fundIDs = append(fundIDs, row.FundID)
			}
			for _, row := range riskRows {
				fundIDs = append(fundIDs, row.FundID)
			}
			fundMap, _ := h.fundRepo.GetFundsByIDs(c.Request.Context(), uniqueStrings(fundIDs))
			c.JSON(http.StatusOK, APIResponse{
				Success: true,
				Data: AnalysisRankingsResponse{
					GeneratedAt:   now,
					IncreaseIdeas: buildRankingItemsFromSnapshots(increaseRows, fundMap),
					WatchIdeas:    buildRankingItemsFromSnapshots(watchRows, fundMap),
					RiskAlerts:    buildRankingItemsFromSnapshots(riskRows, fundMap),
				},
			})
			return
		}
	}

	candidateIDs, err := h.listAnalysisRankingCandidateFundIDs(c.Request.Context(), 36)
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

	candidates := make([]analysisRankingCandidate, len(candidateIDs))
	group, ctx := errgroup.WithContext(c.Request.Context())
	group.SetLimit(6)
	for index, fundID := range candidateIDs {
		index := index
		fundID := fundID
		group.Go(func() error {
			fund, analysis, buildErr := h.buildSingleFundAnalysis(ctx, fundID, now)
			if buildErr != nil {
				log.Printf("⚠️ Failed to build ranking analysis for %s: %v", fundID, buildErr)
				return nil
			}
			if fund == nil || analysis == nil {
				return nil
			}
			candidates[index] = analysisRankingCandidate{fund: fund, analysis: analysis}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		log.Printf("⚠️ ranking build group wait failed: %v", err)
	}
	filteredCandidates := make([]analysisRankingCandidate, 0, len(candidates))
	for _, item := range candidates {
		if item.fund == nil || item.analysis == nil {
			continue
		}
		filteredCandidates = append(filteredCandidates, item)
		if h.snapshotStore != nil && item.analysis != nil {
			_ = h.snapshotStore.Save(c.Request.Context(), item.fund.ID, item.analysis, now)
		}
	}

	increaseItems := make([]analysisRankingCandidate, 0, len(filteredCandidates))
	watchItems := make([]analysisRankingCandidate, 0, len(filteredCandidates))
	riskItems := make([]analysisRankingCandidate, 0, len(filteredCandidates))
	for _, item := range filteredCandidates {
		if item.analysis == nil {
			continue
		}
		dominant := dominantRecommendation(item.analysis)
		switch dominant {
		case "increase":
			increaseItems = append(increaseItems, item)
		case "decrease":
			riskItems = append(riskItems, item)
		default:
			watchItems = append(watchItems, item)
		}
		if item.analysis.RiskLevel == "high" && dominant != "decrease" {
			riskItems = append(riskItems, item)
		}
	}

	sort.SliceStable(increaseItems, func(i, j int) bool {
		return compareIncreaseRanking(increaseItems[i].analysis, increaseItems[j].analysis)
	})
	sort.SliceStable(watchItems, func(i, j int) bool {
		return compareWatchRanking(watchItems[i].analysis, watchItems[j].analysis)
	})
	sort.SliceStable(riskItems, func(i, j int) bool {
		return compareRiskRanking(riskItems[i].analysis, riskItems[j].analysis)
	})

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: AnalysisRankingsResponse{
			GeneratedAt:   now,
			IncreaseIdeas: buildRankingItems(increaseItems, 12),
			WatchIdeas:    buildRankingItems(watchItems, 12),
			RiskAlerts:    buildUniqueRiskItems(riskItems, 12),
		},
	})
}

func (h *FundHandler) buildFundThemeSnapshot(ctx context.Context, fundID string, fund *domain.Fund) (*domain.FundThemeSnapshot, error) {
	if h == nil || h.sectorStore == nil || fund == nil {
		return nil, nil
	}

	holdings, source, err := h.resolveClassificationHoldings(ctx, fundID, fund)
	if err != nil {
		return nil, err
	}
	if !hasEffectiveFundHoldings(holdings) {
		return nil, nil
	}
	return h.sectorStore.UpsertThemeFromHoldings(ctx, fundID, holdings, source)
}

func (h *FundHandler) buildFundAnalysis(
	ctx context.Context,
	fundID string,
	fund *domain.Fund,
	estimate *domain.FundEstimate,
	timeSeries []domain.TimeSeriesPoint,
	sectorSnapshot *domain.FundSectorSnapshot,
	themeSnapshot *domain.FundThemeSnapshot,
	now time.Time,
) (*domain.FundAnalysis, error) {
	if h == nil || fund == nil || estimate == nil {
		return nil, nil
	}

	holdings, source, err := h.resolveClassificationHoldings(ctx, fundID, fund)
	if err != nil {
		return nil, err
	}
	historySourceCode := fundID
	targetCode := ""
	if source == service.SectorSourceTargetETFFallback {
		targetCode = h.resolveTrackedETFCode(ctx, fundID, fund)
		if targetCode != "" {
			historySourceCode = targetCode
		}
	}

	currentHoldingEvents := service.LoadCurrentHoldingNewsEvents(ctx, holdings, now)
	currentFundEvents := service.LoadCurrentFundNoticeEvents(ctx, fundID, now)
	currentMacroEvents := service.LoadMacroPolicyEvents(now, sectorSnapshot, themeSnapshot)
	currentIndexEvents := service.LoadIndexLayerEvents(now, fund, source, sectorSnapshot, themeSnapshot)
	currentTargetEvents := make([]domain.FundAnalysisEventImpact, 0)
	if source == service.SectorSourceTargetETFFallback {
		if targetCode != "" {
			currentTargetEvents = service.LoadCurrentFundNoticeEvents(ctx, targetCode, now)
		}
	}
	previousHoldings, previousHoldingPeriod, previousHoldingsErr := service.LoadPreviousQuarterHoldings(ctx, historySourceCode, holdings)
	if previousHoldingsErr != nil {
		log.Printf("⚠️ Failed to load previous quarter holdings for %s: %v", fundID, previousHoldingsErr)
	}

	var previousSectorSnapshot *domain.FundSectorSnapshot
	var previousThemeSnapshot *domain.FundThemeSnapshot
	if h.sectorStore != nil && hasEffectiveFundHoldings(previousHoldings) {
		previousSectorSnapshot, err = h.sectorStore.BuildSnapshot(ctx, fundID, previousHoldings, source)
		if err != nil {
			log.Printf("⚠️ Failed to build previous sector snapshot for %s: %v", fundID, err)
		}
		previousThemeSnapshot, err = h.sectorStore.BuildThemeSnapshot(ctx, fundID, previousHoldings, source)
		if err != nil {
			log.Printf("⚠️ Failed to build previous theme snapshot for %s: %v", fundID, err)
		}
	}

	return service.NewFundAnalysisService().Build(service.FundAnalysisInput{
		Fund:                   fund,
		Estimate:               estimate,
		TimeSeries:             timeSeries,
		SectorSnapshot:         sectorSnapshot,
		ThemeSnapshot:          themeSnapshot,
		PreviousSectorSnapshot: previousSectorSnapshot,
		PreviousThemeSnapshot:  previousThemeSnapshot,
		CurrentMacroEvents:     currentMacroEvents,
		CurrentIndexEvents:     currentIndexEvents,
		CurrentHoldingEvents:   currentHoldingEvents,
		CurrentFundEvents:      currentFundEvents,
		CurrentTargetEvents:    currentTargetEvents,
		Holdings:               holdings,
		PreviousHoldings:       previousHoldings,
		PreviousHoldingPeriod:  previousHoldingPeriod,
		HoldingsSource:         source,
		Now:                    now,
	}), nil
}

func (h *FundHandler) buildSingleFundAnalysis(
	ctx context.Context,
	fundID string,
	now time.Time,
) (*domain.Fund, *domain.FundAnalysis, error) {
	if h != nil && h.coordinator != nil {
		return h.coordinator.BuildForFund(ctx, fundID, now)
	}
	fund, err := h.fundRepo.GetFundByID(ctx, fundID)
	if err != nil {
		return nil, nil, err
	}
	if fund == nil {
		return nil, nil, nil
	}

	cacheStatus := ""
	fund, cacheStatus = h.hydrateFundProfile(fundID, fund)
	_ = cacheStatus

	estimate, err := h.valuationService.CalculateEstimate(ctx, fundID)
	if err != nil {
		return nil, nil, err
	}

	timeSeries, err := h.valuationService.GetIntradayTimeSeries(ctx, fundID)
	if err != nil {
		return nil, nil, err
	}

	marketStatus := trading.GetMarketStatus(now)
	displayDate, _ := deriveTimeSeriesDisplayContext(timeSeries, marketStatus)
	timeSeries = alignTimeSeriesWithEstimateSnapshot(timeSeries, estimate, displayDate, marketStatus)

	sectorSnapshot, sectorErr := h.buildFundSectorSnapshot(ctx, fundID, fund)
	if sectorErr != nil {
		log.Printf("⚠️ Failed to build fund sector snapshot for %s: %v", fundID, sectorErr)
	}
	themeSnapshot, themeErr := h.buildFundThemeSnapshot(ctx, fundID, fund)
	if themeErr != nil {
		log.Printf("⚠️ Failed to build fund theme snapshot for %s: %v", fundID, themeErr)
	}
	category, categoryErr := h.resolveFundCategory(ctx, fund, sectorSnapshot)
	if categoryErr != nil {
		log.Printf("⚠️ Failed to resolve fund category for %s: %v", fundID, categoryErr)
	} else if category != nil {
		fund.CategoryCode = category.Code
		fund.CategoryName = category.Name
	}

	analysis, err := h.buildFundAnalysis(ctx, fundID, fund, estimate, timeSeries, sectorSnapshot, themeSnapshot, now)
	if err != nil {
		return nil, nil, err
	}
	return fund, analysis, nil
}

func (h *FundHandler) listAnalysisRankingCandidateFundIDs(ctx context.Context, limit int) ([]string, error) {
	if h != nil && h.rankingCandidates != nil {
		fundIDs, err := h.rankingCandidates.ListRankingCandidateFundIDs(ctx, limit)
		if err != nil {
			return nil, err
		}
		if len(fundIDs) > 0 {
			return fundIDs, nil
		}
	}
	return h.fundRepo.ListFundIDsWithHoldings(ctx)
}

func dominantRecommendation(analysis *domain.FundAnalysis) string {
	if analysis == nil {
		return "hold"
	}
	if analysis.IncreasePercent.GreaterThanOrEqual(analysis.HoldPercent) && analysis.IncreasePercent.GreaterThanOrEqual(analysis.DecreasePercent) {
		return "increase"
	}
	if analysis.DecreasePercent.GreaterThanOrEqual(analysis.IncreasePercent) && analysis.DecreasePercent.GreaterThanOrEqual(analysis.HoldPercent) {
		return "decrease"
	}
	return "hold"
}

func compareIncreaseRanking(left *domain.FundAnalysis, right *domain.FundAnalysis) bool {
	if left == nil || right == nil {
		return right != nil
	}
	if !left.IncreasePercent.Equal(right.IncreasePercent) {
		return left.IncreasePercent.GreaterThan(right.IncreasePercent)
	}
	if !left.TotalScore.Equal(right.TotalScore) {
		return left.TotalScore.GreaterThan(right.TotalScore)
	}
	return left.Confidence > right.Confidence
}

func compareWatchRanking(left *domain.FundAnalysis, right *domain.FundAnalysis) bool {
	if left == nil || right == nil {
		return right != nil
	}
	if !left.HoldPercent.Equal(right.HoldPercent) {
		return left.HoldPercent.GreaterThan(right.HoldPercent)
	}
	if !left.TotalScore.Equal(right.TotalScore) {
		return left.TotalScore.GreaterThan(right.TotalScore)
	}
	return left.Confidence > right.Confidence
}

func compareRiskRanking(left *domain.FundAnalysis, right *domain.FundAnalysis) bool {
	if left == nil || right == nil {
		return right != nil
	}
	leftHigh := left.RiskLevel == "high"
	rightHigh := right.RiskLevel == "high"
	if leftHigh != rightHigh {
		return leftHigh
	}
	if !left.DecreasePercent.Equal(right.DecreasePercent) {
		return left.DecreasePercent.GreaterThan(right.DecreasePercent)
	}
	return left.TotalScore.LessThan(right.TotalScore)
}

func buildRankingItems(items []analysisRankingCandidate, limit int) []AnalysisRankingItem {
	if limit <= 0 || len(items) == 0 {
		return []AnalysisRankingItem{}
	}
	if len(items) > limit {
		items = items[:limit]
	}
	result := make([]AnalysisRankingItem, 0, len(items))
	for _, item := range items {
		result = append(result, AnalysisRankingItem{
			Fund:     item.fund,
			Analysis: item.analysis,
		})
	}
	return result
}

func buildUniqueRiskItems(items []analysisRankingCandidate, limit int) []AnalysisRankingItem {
	if limit <= 0 || len(items) == 0 {
		return []AnalysisRankingItem{}
	}
	result := make([]AnalysisRankingItem, 0, min(limit, len(items)))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.fund == nil {
			continue
		}
		if _, ok := seen[item.fund.ID]; ok {
			continue
		}
		seen[item.fund.ID] = struct{}{}
		result = append(result, AnalysisRankingItem{Fund: item.fund, Analysis: item.analysis})
		if len(result) >= limit {
			break
		}
	}
	return result
}

func buildRankingItemsFromSnapshots(records []database.FundAnalysisSnapshot, fundMap map[string]*domain.Fund) []AnalysisRankingItem {
	result := make([]AnalysisRankingItem, 0, len(records))
	for _, record := range records {
		var analysis domain.FundAnalysis
		if err := json.Unmarshal(record.AnalysisJSON, &analysis); err != nil {
			continue
		}
		fund := &domain.Fund{ID: record.FundID}
		if existing := fundMap[record.FundID]; existing != nil {
			fund = existing
		}
		result = append(result, AnalysisRankingItem{
			Fund: fund,
			Analysis: &analysis,
		})
	}
	return result
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (h *FundHandler) resolveTrackedETFCode(ctx context.Context, fundID string, fund *domain.Fund) string {
	if h == nil || fund == nil {
		return ""
	}
	resolver, ok := h.holdingsResolver.(holdingsDisplayResolver)
	if !ok {
		return ""
	}
	displayHoldings, err := resolver.ResolveDisplayHoldings(ctx, fundID, fund.Name)
	if err != nil {
		return ""
	}
	if targetItem, ok := domain.PrimaryTrackedETF(displayHoldings); ok {
		return strings.TrimSpace(targetItem.Code)
	}
	return ""
}

func (h *FundHandler) buildFundSectorSnapshot(ctx context.Context, fundID string, fund *domain.Fund) (*domain.FundSectorSnapshot, error) {
	if h == nil || h.sectorStore == nil || fund == nil {
		return nil, nil
	}

	holdings, source, err := h.resolveClassificationHoldings(ctx, fundID, fund)
	if err != nil {
		return nil, err
	}
	if !hasEffectiveFundHoldings(holdings) {
		return nil, nil
	}
	return h.sectorStore.UpsertFromHoldings(ctx, fundID, holdings, source)
}

func (h *FundHandler) resolveClassificationHoldings(ctx context.Context, fundID string, fund *domain.Fund) ([]domain.StockHolding, string, error) {
	if h == nil || fund == nil {
		return nil, "", nil
	}

	holdings, err := h.fundRepo.GetFundHoldings(ctx, fundID)
	if err != nil {
		return nil, "", err
	}

	source := service.SectorSourceDirectHoldings
	if shouldHydrateFundHoldings(fund, holdings) {
		hydratedFund, hydratedHoldings, _ := h.cachedFundDataOrScheduleWarmup(fundID)
		if hydratedFund != nil {
			fund = hydratedFund
		}
		if len(hydratedHoldings) > 0 {
			holdings = hydratedHoldings
		}
	}

	if resolver, ok := h.holdingsResolver.(holdingsDisplayResolver); ok {
		displayHoldings, displayErr := resolver.ResolveDisplayHoldings(ctx, fundID, fund.Name)
		if displayErr != nil {
			return nil, "", displayErr
		}
		if targetItem, ok := domain.PrimaryTrackedETF(displayHoldings); ok {
			targetCode := strings.TrimSpace(targetItem.Code)
			if targetCode != "" {
				targetHoldings, _ := h.resolveTrackedETFHoldings(ctx, targetCode)
				if hasEffectiveFundHoldings(targetHoldings) {
					holdings = targetHoldings
					source = service.SectorSourceTargetETFFallback
				}
			}
		}
	}

	if !hasEffectiveFundHoldings(holdings) && h.holdingsResolver != nil {
		resolvedHoldings, holdingsSource, resolveErr := h.holdingsResolver.GetHoldingsWithFallback(ctx, fundID, fund.Name)
		if resolveErr != nil {
			return nil, "", resolveErr
		}
		if len(resolvedHoldings) > 0 {
			holdings = resolvedHoldings
			if holdingsSource != "" && holdingsSource != fundID {
				source = service.SectorSourceTargetETFFallback
			}
		}
	}

	if strings.Contains(strings.ToLower(fund.Type), "qdii") || strings.Contains(strings.ToLower(fund.Name), "qdii") {
		source = service.SectorSourceQDIIHoldings
	}

	return holdings, source, nil
}

func (h *FundHandler) resolveFundCategory(ctx context.Context, fund *domain.Fund, snapshot *domain.FundSectorSnapshot) (*domain.FundCategory, error) {
	if h == nil || h.sectorStore == nil || fund == nil {
		return nil, nil
	}
	return h.sectorStore.ResolveFundCategory(ctx, fund, snapshot)
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

	if !hasEffectiveFundHoldings(holdings) && h.holdingsResolver != nil {
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
		Fund                 *domain.Fund                     `json:"fund"`
		Holdings             []domain.StockHolding            `json:"holdings"`
		DisplayLevel         string                           `json:"display_level"`
		DisplayItems         []domain.FundHoldingsDisplayItem `json:"display_items"`
		LookthroughAvailable bool                             `json:"lookthrough_available"`
	}

	displayLevel := domain.FundHoldingsDisplayLevelStock
	displayItems := buildStockDisplayItems(holdings)
	lookthroughAvailable := false
	trackedETFCode := ""
	if resolver, ok := h.holdingsResolver.(holdingsDisplayResolver); ok {
		displayHoldings, displayErr := resolver.ResolveDisplayHoldings(c.Request.Context(), fundID, fund.Name)
		if displayErr != nil {
			log.Printf("⚠️ Holdings display resolver failed for %s: %v", fundID, displayErr)
		} else if displayHoldings != nil && len(displayHoldings.DisplayItems) > 0 {
			displayLevel = strings.TrimSpace(displayHoldings.DisplayLevel)
			if displayLevel == "" {
				displayLevel = domain.FundHoldingsDisplayLevelStock
			}
			displayItems = displayHoldings.DisplayItems
			lookthroughAvailable = displayHoldings.LookthroughAvailable
			if targetItem, ok := domain.PrimaryTrackedETF(displayHoldings); ok {
				trackedETFCode = strings.TrimSpace(targetItem.Code)
			}
		}
	}

	if trackedETFCode != "" {
		targetHoldings, targetCacheStatus := h.resolveTrackedETFHoldings(c.Request.Context(), trackedETFCode)
		if hasEffectiveFundHoldings(targetHoldings) {
			holdings = targetHoldings
			dataSource = "target_etf:" + trackedETFCode
			if targetCacheStatus != "" {
				cacheStatus = targetCacheStatus
			}
		}
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: HoldingsResponse{
			Fund:                 fund,
			Holdings:             holdings,
			DisplayLevel:         displayLevel,
			DisplayItems:         displayItems,
			LookthroughAvailable: lookthroughAvailable,
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
	dataDate, isHistorical := deriveTimeSeriesDisplayContext(timeSeries, marketStatus)

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

func mapEstimateErrorToResponse(err error) (int, *APIError) {
	statusCode := http.StatusInternalServerError
	errorCode := "ESTIMATE_FAILED"

	switch {
	case errors.Is(err, service.ErrFundDataWarmupInProgress):
		statusCode = http.StatusServiceUnavailable
		errorCode = "FUND_DATA_WARMING"
	case strings.Contains(err.Error(), "qdii details available without live estimate support"):
		statusCode = http.StatusUnprocessableEntity
		errorCode = "UNSUPPORTED_PRICING_MODEL"
	case strings.Contains(err.Error(), "pricing profile not configured") || strings.Contains(err.Error(), "unsupported pricing method"):
		statusCode = http.StatusUnprocessableEntity
		errorCode = "UNSUPPORTED_PRICING_MODEL"
	case strings.Contains(err.Error(), "not found"):
		statusCode = http.StatusNotFound
		errorCode = "FUND_NOT_FOUND"
	}

	return statusCode, &APIError{
		Code:    errorCode,
		Message: err.Error(),
	}
}

func deriveTimeSeriesDisplayContext(points []domain.TimeSeriesPoint, marketStatus trading.MarketStatus) (string, bool) {
	if len(points) == 0 {
		return marketStatus.DisplayDate, false
	}

	firstPointDate := points[0].Timestamp.In(trading.TradingLocation()).Format("2006-01-02")
	isHistorical := firstPointDate != marketStatus.CurrentDate
	if isHistorical {
		return firstPointDate, true
	}
	return marketStatus.DisplayDate, false
}

func alignTimeSeriesWithEstimateSnapshot(points []domain.TimeSeriesPoint, estimate *domain.FundEstimate, displayDate string, marketStatus trading.MarketStatus) []domain.TimeSeriesPoint {
	if estimate == nil || displayDate == "" {
		return points
	}

	loc := trading.TradingLocation()
	calculatedAt := estimate.CalculatedAt.In(loc)
	displayDay, err := time.ParseInLocation("2006-01-02", displayDate, loc)
	if err != nil {
		return points
	}

	targetTimestamp := time.Date(displayDay.Year(), displayDay.Month(), displayDay.Day(), 15, 0, 0, 0, loc)
	if displayDate == marketStatus.CurrentDate && marketStatus.IsTradingDay {
		targetTimestamp = time.Date(calculatedAt.Year(), calculatedAt.Month(), calculatedAt.Day(), calculatedAt.Hour(), (calculatedAt.Minute()/5)*5, 0, 0, loc)
	}

	alignedPoint := domain.TimeSeriesPoint{
		Timestamp:     targetTimestamp,
		ChangePercent: estimate.ChangePercent,
		EstimateNav:   estimate.EstimateNav,
	}

	if len(points) == 0 {
		return []domain.TimeSeriesPoint{alignedPoint}
	}

	updated := false
	aligned := make([]domain.TimeSeriesPoint, len(points))
	copy(aligned, points)
	for i := range aligned {
		localTs := aligned[i].Timestamp.In(loc)
		if localTs.Equal(targetTimestamp) {
			aligned[i].ChangePercent = alignedPoint.ChangePercent
			aligned[i].EstimateNav = alignedPoint.EstimateNav
			updated = true
			break
		}
	}

	if !updated {
		aligned = append(aligned, alignedPoint)
		sort.Slice(aligned, func(i, j int) bool {
			return aligned[i].Timestamp.Before(aligned[j].Timestamp)
		})
	}
	return aligned
}

func displayDateFromTimeSeries(points []domain.TimeSeriesPoint) time.Time {
	if len(points) == 0 {
		return time.Time{}
	}
	return points[0].Timestamp
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
	return shouldHydrateFundProfile(fund) || !hasEffectiveFundHoldings(holdings)
}

func (h *FundHandler) resolveTrackedETFHoldings(ctx context.Context, targetCode string) ([]domain.StockHolding, string) {
	if strings.TrimSpace(targetCode) == "" {
		return nil, ""
	}

	targetHoldings, err := h.fundRepo.GetFundHoldings(ctx, targetCode)
	if err == nil && hasEffectiveFundHoldings(targetHoldings) {
		return targetHoldings, ""
	}

	targetFund, cachedHoldings, cacheStatus := h.cachedFundDataOrScheduleWarmup(targetCode)
	_ = targetFund
	if hasEffectiveFundHoldings(cachedHoldings) {
		return cachedHoldings, cacheStatus
	}

	return targetHoldings, cacheStatus
}

func hasEffectiveFundHoldings(holdings []domain.StockHolding) bool {
	if len(holdings) == 0 {
		return false
	}

	totalRatio := decimal.Zero
	for _, holding := range holdings {
		if holding.HoldingRatio.GreaterThan(decimal.Zero) {
			totalRatio = totalRatio.Add(holding.HoldingRatio)
		}
	}
	return totalRatio.GreaterThan(decimal.Zero)
}

func buildStockDisplayItems(holdings []domain.StockHolding) []domain.FundHoldingsDisplayItem {
	if len(holdings) == 0 {
		return []domain.FundHoldingsDisplayItem{}
	}

	items := make([]domain.FundHoldingsDisplayItem, 0, len(holdings))
	for _, holding := range holdings {
		items = append(items, domain.FundHoldingsDisplayItem{
			ItemType:        domain.FundHoldingsDisplayItemTypeStock,
			Code:            holding.StockCode,
			Name:            holding.StockName,
			Exchange:        holding.Exchange,
			HoldingRatio:    holding.HoldingRatio,
			WeightPercent:   holding.HoldingRatio,
			ReportingPeriod: holding.ReportingPeriod,
		})
	}
	return items
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
