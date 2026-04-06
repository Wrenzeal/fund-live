package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/gin-gonic/gin"
)

type vipReportEnvelope struct {
	Success bool             `json:"success"`
	Data    domain.VIPReport `json:"data"`
}

func TestGetVIPReportPublicSample(t *testing.T) {
	gin.SetMode(gin.TestMode)

	vipHandler := NewVIPHandler(service.NewVIPService(repository.NewMemoryVIPRepository()))
	router := gin.New()
	router.GET("/api/v1/vip/reports/:id", vipHandler.GetReport)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vip/reports/portfolio-core-balance", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response vipReportEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("response success = false")
	}
	if response.Data.ID != "portfolio-core-balance" {
		t.Fatalf("report id = %q, want portfolio-core-balance", response.Data.ID)
	}
}
