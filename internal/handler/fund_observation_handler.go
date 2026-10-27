package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/gin-gonic/gin"
)

type fundObservationReader interface {
	Observe(context.Context, string, time.Time) (*domain.Fund, *domain.FundObservation, error)
}

type FundObservationHandler struct {
	service fundObservationReader
}

func NewFundObservationHandler(fundRepo domain.FundRepository, resolver service.ObservationHoldingsResolver) *FundObservationHandler {
	return &FundObservationHandler{service: service.NewFundObservationService(fundRepo, resolver)}
}

func (h *FundObservationHandler) Get(c *gin.Context) {
	fundID := c.Param("id")
	if !validObservationFundID(fundID) {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: &APIError{Code: "INVALID_FUND_ID", Message: "请输入六位基金代码。"}})
		return
	}
	fund, observation, err := h.service.Observe(c.Request.Context(), fundID, time.Now())
	if err != nil {
		log.Printf("Fund observation unavailable for %s: %v", fundID, err)
		c.JSON(http.StatusServiceUnavailable, APIResponse{Success: false, Error: &APIError{Code: "OBSERVATION_UNAVAILABLE", Message: "基金资料暂不可用，请稍后重试。"}})
		return
	}
	if fund == nil {
		c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: &APIError{Code: "FUND_NOT_FOUND", Message: "未找到该基金。"}})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: gin.H{"fund": fund, "observation": observation}})
}

func validObservationFundID(value string) bool {
	if len(value) != 6 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
