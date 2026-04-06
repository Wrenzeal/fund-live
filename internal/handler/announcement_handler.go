package handler

import (
	"errors"
	"net/http"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/middleware"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/gin-gonic/gin"
)

type AnnouncementHandler struct {
	announcementService domain.AnnouncementService
}

func NewAnnouncementHandler(announcementService domain.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{announcementService: announcementService}
}

type createAnnouncementRequest struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Content string `json:"content"`
}

func (h *AnnouncementHandler) List(c *gin.Context) {
	items, err := h.announcementService.ListAnnouncements(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   &APIError{Code: "ANNOUNCEMENTS_FAILED", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: items})
}

func (h *AnnouncementHandler) Get(c *gin.Context) {
	item, err := h.announcementService.GetAnnouncementByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		statusCode, apiErr := mapAnnouncementError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: item})
}

func (h *AnnouncementHandler) ListUnread(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	items, err := h.announcementService.ListUnreadAnnouncements(c.Request.Context(), user.ID, 5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   &APIError{Code: "ANNOUNCEMENTS_FAILED", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: items})
}

func (h *AnnouncementHandler) MarkRead(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	if err := h.announcementService.MarkAnnouncementRead(c.Request.Context(), user.ID, c.Param("id")); err != nil {
		statusCode, apiErr := mapAnnouncementError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: gin.H{
			"announcement_id": c.Param("id"),
			"read":            true,
		},
	})
}

func (h *AnnouncementHandler) Create(c *gin.Context) {
	var req createAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid announcement payload"},
		})
		return
	}

	item, err := h.announcementService.CreateAnnouncement(c.Request.Context(), domain.AnnouncementCreateInput{
		Title:   req.Title,
		Summary: req.Summary,
		Content: req.Content,
	})
	if err != nil {
		statusCode, apiErr := mapAnnouncementError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{Success: true, Data: item})
}

func (h *AnnouncementHandler) ImportChangelog(c *gin.Context) {
	count, err := h.announcementService.ImportAnnouncementsFromChangelog(c.Request.Context())
	if err != nil {
		statusCode, apiErr := mapAnnouncementError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: gin.H{
			"imported": count,
		},
	})
}

func mapAnnouncementError(err error) (int, *APIError) {
	switch {
	case errors.Is(err, service.ErrAnnouncementInvalidContent):
		return http.StatusBadRequest, &APIError{Code: "INVALID_ANNOUNCEMENT", Message: err.Error()}
	case errors.Is(err, service.ErrAnnouncementNotFound):
		return http.StatusNotFound, &APIError{Code: "ANNOUNCEMENT_NOT_FOUND", Message: err.Error()}
	default:
		return http.StatusInternalServerError, &APIError{Code: "ANNOUNCEMENT_FAILED", Message: err.Error()}
	}
}
