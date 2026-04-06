package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/middleware"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/gin-gonic/gin"
)

type VIPHandler struct {
	vipService domain.VIPService
}

func NewVIPHandler(vipService domain.VIPService) *VIPHandler {
	return &VIPHandler{vipService: vipService}
}

type activatePreviewMembershipRequest struct {
	BillingCycle domain.VIPBillingCycle `json:"billing_cycle"`
}

type createVIPTaskRequest struct {
	Type       domain.VIPTaskType   `json:"type"`
	TargetType domain.VIPTargetType `json:"target_type"`
	TargetID   string               `json:"target_id"`
	TargetName string               `json:"target_name"`
}

type createVIPOrderRequest struct {
	BillingCycle domain.VIPBillingCycle `json:"billing_cycle"`
}

func (h *VIPHandler) GetMembership(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	membership, err := h.vipService.GetMembership(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   &APIError{Code: "VIP_MEMBERSHIP_FAILED", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: membership})
}

func (h *VIPHandler) PreviewActivateMembership(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	var req activatePreviewMembershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid preview activate payload"},
		})
		return
	}

	membership, err := h.vipService.ActivatePreviewMembership(c.Request.Context(), user.ID, req.BillingCycle)
	if err != nil {
		statusCode, apiErr := mapVIPError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: membership})
}

func (h *VIPHandler) PreviewReset(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	if err := h.vipService.ResetPreview(c.Request.Context(), user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   &APIError{Code: "VIP_RESET_FAILED", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: gin.H{
			"reset": true,
		},
	})
}

func (h *VIPHandler) GetQuota(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	quota, err := h.vipService.GetQuota(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   &APIError{Code: "VIP_QUOTA_FAILED", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: quota})
}

func (h *VIPHandler) CreateTask(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	var req createVIPTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid vip task payload"},
		})
		return
	}

	result, err := h.vipService.CreateTask(c.Request.Context(), user.ID, domain.VIPTaskCreateInput{
		Type:       req.Type,
		TargetType: req.TargetType,
		TargetID:   strings.TrimSpace(req.TargetID),
		TargetName: strings.TrimSpace(req.TargetName),
	})
	if err != nil {
		statusCode, apiErr := mapVIPError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{Success: true, Data: result})
}

func (h *VIPHandler) ListTasks(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	tasks, err := h.vipService.ListTasks(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   &APIError{Code: "VIP_TASKS_FAILED", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: tasks})
}

func (h *VIPHandler) GetReport(c *gin.Context) {
	userID := ""
	if user, ok := middleware.CurrentUser(c); ok && user != nil {
		userID = user.ID
	}

	report, err := h.vipService.GetReportByID(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		statusCode, apiErr := mapVIPError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: report})
}

func (h *VIPHandler) CreateOrder(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	var req createVIPOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid vip order payload"},
		})
		return
	}

	order, err := h.vipService.CreateOrder(c.Request.Context(), user.ID, domain.VIPOrderCreateInput{
		BillingCycle: req.BillingCycle,
	})
	if err != nil {
		statusCode, apiErr := mapVIPError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{Success: true, Data: order})
}

func (h *VIPHandler) GetOrder(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	order, err := h.vipService.GetOrder(c.Request.Context(), user.ID, c.Param("orderId"))
	if err != nil {
		statusCode, apiErr := mapVIPError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: order})
}

func (h *VIPHandler) HandleWeChatPayNotify(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "FAIL",
			"message": "invalid notify payload",
		})
		return
	}

	headers := map[string]string{
		"Wechatpay-Timestamp": c.GetHeader("Wechatpay-Timestamp"),
		"Wechatpay-Nonce":     c.GetHeader("Wechatpay-Nonce"),
		"Wechatpay-Signature": c.GetHeader("Wechatpay-Signature"),
		"Wechatpay-Serial":    c.GetHeader("Wechatpay-Serial"),
	}

	if err := h.vipService.HandleWeChatPayNotify(c.Request.Context(), headers, body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "FAIL",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "成功",
	})
}

func mapVIPError(err error) (int, *APIError) {
	switch {
	case errors.Is(err, service.ErrVIPInvalidBillingCycle):
		return http.StatusBadRequest, &APIError{Code: "INVALID_BILLING_CYCLE", Message: err.Error()}
	case errors.Is(err, service.ErrVIPInvalidTaskInput):
		return http.StatusBadRequest, &APIError{Code: "INVALID_VIP_TASK", Message: err.Error()}
	case errors.Is(err, service.ErrVIPMembershipRequired):
		return http.StatusForbidden, &APIError{Code: "VIP_REQUIRED", Message: err.Error()}
	case errors.Is(err, service.ErrVIPQuotaExceeded):
		return http.StatusConflict, &APIError{Code: "VIP_QUOTA_EXHAUSTED", Message: err.Error()}
	case errors.Is(err, service.ErrVIPReportNotFound):
		return http.StatusNotFound, &APIError{Code: "VIP_REPORT_NOT_FOUND", Message: err.Error()}
	case errors.Is(err, service.ErrVIPOrderNotFound):
		return http.StatusNotFound, &APIError{Code: "VIP_ORDER_NOT_FOUND", Message: err.Error()}
	case errors.Is(err, service.ErrVIPPaymentNotConfigured), errors.Is(err, service.ErrWeChatPayNotConfigured):
		return http.StatusServiceUnavailable, &APIError{Code: "PAYMENT_NOT_CONFIGURED", Message: err.Error()}
	default:
		return http.StatusInternalServerError, &APIError{Code: "VIP_FAILED", Message: err.Error()}
	}
}
