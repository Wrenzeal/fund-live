package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/gin-gonic/gin"
)

type observationServiceStub struct {
	fund        *domain.Fund
	observation *domain.FundObservation
	err         error
}

func (s observationServiceStub) Observe(context.Context, string, time.Time) (*domain.Fund, *domain.FundObservation, error) {
	return s.fund, s.observation, s.err
}

func TestFundObservationHandlerContractAndErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name string
		stub observationServiceStub
		want int
	}{
		{"success", observationServiceStub{fund: &domain.Fund{ID: "000001"}, observation: &domain.FundObservation{WindowDays: 45, Notices: []domain.FundObservationNotice{}, Sources: []domain.FundObservationSource{}, Notes: []string{}}}, http.StatusOK},
		{"missing", observationServiceStub{}, http.StatusNotFound},
		{"internal error", observationServiceStub{err: errors.New("password=private")}, http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &FundObservationHandler{service: tc.stub}
			router := gin.New()
			router.GET("/fund/:id/observation", h.Get)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/fund/000001/observation", nil))
			if response.Code != tc.want {
				t.Fatalf("status=%d body=%s", response.Code, response.Body)
			}
			var payload struct {
				Success bool `json:"success"`
				Data    struct {
					Fund        domain.Fund            `json:"fund"`
					Observation domain.FundObservation `json:"observation"`
					Analysis    json.RawMessage        `json:"analysis"`
				} `json:"data"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if tc.want == http.StatusOK && (!payload.Success || payload.Data.Fund.ID != "000001" || payload.Data.Observation.WindowDays != 45 || payload.Data.Analysis != nil) {
				t.Fatalf("invalid observation contract: %s", response.Body)
			}
			if tc.want != http.StatusOK && (payload.Success || strings.Contains(response.Body.String(), "password")) {
				t.Fatalf("unexpected error response: %s", response.Body)
			}
		})
	}
}

func TestFundObservationHandlerRejectsInvalidIDWithoutReadingSources(t *testing.T) {
	h := &FundObservationHandler{}
	router := gin.New()
	router.GET("/fund/:id/observation", h.Get)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/fund/not-a-fund/observation", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}
}
