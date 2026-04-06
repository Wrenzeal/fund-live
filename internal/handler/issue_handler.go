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

type IssueHandler struct {
	issueService domain.IssueService
}

func NewIssueHandler(issueService domain.IssueService) *IssueHandler {
	return &IssueHandler{issueService: issueService}
}

type createIssueRequest struct {
	Title string           `json:"title"`
	Body  string           `json:"body"`
	Type  domain.IssueType `json:"type"`
}

type updateIssueStatusRequest struct {
	Status domain.IssueStatus `json:"status"`
}

func (h *IssueHandler) List(c *gin.Context) {
	issues, err := h.issueService.ListPublicIssues(c.Request.Context(), domain.IssueSearchParams{
		Query:  c.Query("q"),
		Type:   domain.IssueType(strings.TrimSpace(c.Query("type"))),
		Status: domain.IssueStatus(strings.TrimSpace(c.Query("status"))),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   &APIError{Code: "ISSUES_FAILED", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: issues})
}

func (h *IssueHandler) Get(c *gin.Context) {
	issue, err := h.issueService.GetIssueByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		statusCode, apiErr := mapIssueError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: issue})
}

func (h *IssueHandler) Create(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error:   &APIError{Code: "UNAUTHORIZED", Message: "Authentication required"},
		})
		return
	}

	var req createIssueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid issue payload"},
		})
		return
	}

	issue, err := h.issueService.CreateIssue(c.Request.Context(), user, domain.IssueCreateInput{
		Title: req.Title,
		Body:  req.Body,
		Type:  req.Type,
	})
	if err != nil {
		statusCode, apiErr := mapIssueError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{Success: true, Data: issue})
}

func (h *IssueHandler) UpdateStatus(c *gin.Context) {
	var req updateIssueStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: "Invalid issue status payload"},
		})
		return
	}

	issue, err := h.issueService.UpdateIssueStatus(c.Request.Context(), c.Param("id"), req.Status)
	if err != nil {
		statusCode, apiErr := mapIssueError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	c.JSON(http.StatusOK, APIResponse{Success: true, Data: issue})
}

func mapIssueError(err error) (int, *APIError) {
	switch {
	case errors.Is(err, service.ErrIssueInvalidType):
		return http.StatusBadRequest, &APIError{Code: "INVALID_ISSUE_TYPE", Message: err.Error()}
	case errors.Is(err, service.ErrIssueInvalidStatus):
		return http.StatusBadRequest, &APIError{Code: "INVALID_ISSUE_STATUS", Message: err.Error()}
	case errors.Is(err, service.ErrIssueInvalidContent):
		return http.StatusBadRequest, &APIError{Code: "INVALID_ISSUE_CONTENT", Message: err.Error()}
	case errors.Is(err, service.ErrIssueNotFound):
		return http.StatusNotFound, &APIError{Code: "ISSUE_NOT_FOUND", Message: err.Error()}
	default:
		return http.StatusInternalServerError, &APIError{Code: "ISSUE_FAILED", Message: err.Error()}
	}
}
