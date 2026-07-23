package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type quantBacktestEnqueuer interface {
	EnqueueBacktest(ctx context.Context, jobID string) error
}

type QuantHandler struct {
	eventStore    *service.QuantEventStore
	researchStore *service.QuantResearchStore
	queue         quantBacktestEnqueuer
}

func NewQuantHandler(eventStore *service.QuantEventStore, researchStore *service.QuantResearchStore, queue quantBacktestEnqueuer) *QuantHandler {
	return &QuantHandler{eventStore: eventStore, researchStore: researchStore, queue: queue}
}

// ListFundEvents returns the latest event version that was known at the requested time.
// GET /api/v1/fund/:id/events?as_of=...&status=...&limit=50
func (h *QuantHandler) ListFundEvents(c *gin.Context) {
	if h == nil || h.eventStore == nil {
		c.JSON(http.StatusServiceUnavailable, APIResponse{Success: false, Error: &APIError{Code: "QUANT_EVENTS_UNAVAILABLE", Message: "事件历史仅在 PostgreSQL 模式可用"}})
		return
	}
	fundID := strings.TrimSpace(c.Param("id"))
	if fundID == "" {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: &APIError{Code: "INVALID_FUND_ID", Message: "Fund ID is required"}})
		return
	}
	asOf, err := parseQuantAsOf(c.Query("as_of"))
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: &APIError{Code: "INVALID_AS_OF", Message: err.Error()}})
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	if status != "" && !validQuantEventStatus(status) {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: &APIError{Code: "INVALID_EVENT_STATUS", Message: "status must be expected, disclosed, active, expired or cancelled"}})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	events, err := h.eventStore.ListAsOf(c.Request.Context(), fundID, asOf, status, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: &APIError{Code: "EVENT_QUERY_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: gin.H{"fund_id": fundID, "as_of": asOf, "events": events, "count": len(events)}})
}

func parseQuantAsOf(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now(), nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if parsed, err := time.ParseInLocation(layout, value, time.FixedZone("CST", 8*60*60)); err == nil {
			if layout == "2006-01-02" {
				parsed = parsed.Add(24*time.Hour - time.Nanosecond)
			}
			return parsed, nil
		}
	}
	return time.Time{}, &quantInputError{message: "as_of must be RFC3339 or YYYY-MM-DD"}
}

type quantInputError struct{ message string }

func (e *quantInputError) Error() string { return e.message }

func validQuantEventStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "expected", "disclosed", "active", "expired", "cancelled":
		return true
	default:
		return false
	}
}

func (h *QuantHandler) ListPilotUniverse(c *gin.Context) {
	items := service.PilotV1Instruments()
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: gin.H{
		"version": service.QuantUniversePilotV1,
		"count":   len(items),
		"items":   items,
	}})
}

func (h *QuantHandler) GetValidationSummary(c *gin.Context) {
	if h == nil || h.researchStore == nil {
		c.JSON(http.StatusServiceUnavailable, APIResponse{Success: false, Error: &APIError{Code: "QUANT_RESEARCH_UNAVAILABLE", Message: "量化验证仅在 PostgreSQL 模式可用"}})
		return
	}
	summary, err := h.researchStore.ValidationSummary(c.Request.Context(), strings.TrimSpace(c.Query("mode")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: &APIError{Code: "QUANT_VALIDATION_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: summary})
}

// CreateBacktest creates one idempotent Lean job. This endpoint is admin-only.
func (h *QuantHandler) CreateBacktest(c *gin.Context) {
	if h == nil || h.researchStore == nil {
		c.JSON(http.StatusServiceUnavailable, APIResponse{Success: false, Error: &APIError{Code: "QUANT_RESEARCH_UNAVAILABLE", Message: "回测服务仅在 PostgreSQL 模式可用"}})
		return
	}
	var request service.QuantBacktestRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: &APIError{Code: "INVALID_BACKTEST_REQUEST", Message: err.Error()}})
		return
	}
	job, created, err := h.researchStore.CreateBacktestJob(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: &APIError{Code: "INVALID_BACKTEST_REQUEST", Message: err.Error()}})
		return
	}
	if !created {
		if job.Status != "failed" && job.Status != "queue_failed" {
			c.JSON(http.StatusOK, APIResponse{Success: true, Data: job, Meta: &APIMeta{CacheStatus: "idempotent_hit"}})
			return
		}
		retried, retryErr := h.researchStore.RetryBacktestJob(c.Request.Context(), job.ID)
		if retryErr != nil || !retried {
			c.JSON(http.StatusConflict, APIResponse{Success: false, Data: job, Error: &APIError{Code: "BACKTEST_RETRY_FAILED", Message: "回测任务暂时无法重试"}})
			return
		}
		job.Status = "queued"
		job.ErrorMessage = ""
	}
	if h.queue == nil {
		_ = h.researchStore.MarkBacktestQueueFailed(c.Request.Context(), job.ID, nil)
		job.Status = "queue_failed"
		job.ErrorMessage = "Dragonfly quant queue is unavailable"
		c.JSON(http.StatusServiceUnavailable, APIResponse{Success: false, Data: job, Error: &APIError{Code: "QUANT_QUEUE_UNAVAILABLE", Message: "Dragonfly 回测队列不可用"}})
		return
	}
	if err := h.queue.EnqueueBacktest(c.Request.Context(), job.ID); err != nil {
		_ = h.researchStore.MarkBacktestQueueFailed(c.Request.Context(), job.ID, err)
		job.Status = "queue_failed"
		job.ErrorMessage = err.Error()
		c.JSON(http.StatusServiceUnavailable, APIResponse{Success: false, Data: job, Error: &APIError{Code: "QUANT_QUEUE_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusAccepted, APIResponse{Success: true, Data: job})
}

func (h *QuantHandler) GetBacktest(c *gin.Context) {
	if h == nil || h.researchStore == nil {
		c.JSON(http.StatusServiceUnavailable, APIResponse{Success: false, Error: &APIError{Code: "QUANT_RESEARCH_UNAVAILABLE", Message: "回测服务仅在 PostgreSQL 模式可用"}})
		return
	}
	job, err := h.researchStore.GetBacktestJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: &APIError{Code: "BACKTEST_NOT_FOUND", Message: "Backtest job not found"}})
			return
		}
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: &APIError{Code: "BACKTEST_QUERY_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: job})
}

func (h *QuantHandler) ListBacktests(c *gin.Context) {
	if h == nil || h.researchStore == nil {
		c.JSON(http.StatusServiceUnavailable, APIResponse{Success: false, Error: &APIError{Code: "QUANT_RESEARCH_UNAVAILABLE", Message: "回测服务仅在 PostgreSQL 模式可用"}})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	jobs, err := h.researchStore.ListBacktestJobs(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: &APIError{Code: "BACKTEST_QUERY_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: gin.H{"items": jobs, "count": len(jobs)}})
}
