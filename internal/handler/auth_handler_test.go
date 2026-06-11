package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/gin-gonic/gin"
)

func TestAuthHandlerConfigExposesGoogleClientID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := NewAuthHandler(stubAuthService{}, "fundlive_session", false)
	handler.SetGoogleWebClientID("web-client.apps.googleusercontent.com")
	router.GET("/api/v1/auth/config", handler.Config)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/config", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			GoogleClientID     string `json:"google_client_id"`
			GoogleLoginEnabled bool   `json:"google_login_enabled"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("success = false, want true")
	}
	if response.Data.GoogleClientID != "web-client.apps.googleusercontent.com" {
		t.Fatalf("google client id = %q", response.Data.GoogleClientID)
	}
	if !response.Data.GoogleLoginEnabled {
		t.Fatalf("google login enabled = false, want true")
	}
}

func TestMapAuthErrorRateLimit(t *testing.T) {
	status, apiErr := mapAuthError(service.ErrAuthRateLimited)

	if status != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", status, http.StatusTooManyRequests)
	}
	if apiErr == nil || apiErr.Code != "AUTH_RATE_LIMITED" {
		t.Fatalf("api error = %#v, want AUTH_RATE_LIMITED", apiErr)
	}
}

func TestMapAuthErrorHidesUnexpectedDetails(t *testing.T) {
	status, apiErr := mapAuthError(errors.New("database password leaked in stack"))

	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", status, http.StatusInternalServerError)
	}
	if apiErr == nil || apiErr.Code != "AUTH_FAILED" {
		t.Fatalf("api error = %#v, want AUTH_FAILED", apiErr)
	}
	if apiErr.Message != "Authentication failed" {
		t.Fatalf("message = %q, want generic authentication failure", apiErr.Message)
	}
}

func TestAuthHandlerRejectsOversizedLoginPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := NewAuthHandler(stubAuthService{}, "fundlive_session", false)
	router.POST("/api/v1/auth/login", handler.Login)

	body := `{"email":"oversized@example.com","password":"` + strings.Repeat("x", int(maxAuthRequestBodyBytes)) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}

	var response APIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error == nil || response.Error.Code != "REQUEST_TOO_LARGE" {
		t.Fatalf("error = %#v, want REQUEST_TOO_LARGE", response.Error)
	}
}

type stubAuthService struct{}

func (stubAuthService) RegisterWithPassword(ctx context.Context, input domain.PasswordRegistrationInput, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	return nil, service.ErrInvalidCredentials
}

func (stubAuthService) LoginWithPassword(ctx context.Context, input domain.PasswordLoginInput, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	return nil, service.ErrInvalidCredentials
}

func (stubAuthService) LoginWithGoogle(ctx context.Context, input domain.GoogleLoginInput, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	return nil, service.ErrInvalidGoogleToken
}

func (stubAuthService) AuthenticateSession(ctx context.Context, sessionToken string) (*domain.AuthenticatedSession, error) {
	return nil, service.ErrInvalidSession
}

func (stubAuthService) LogoutByToken(ctx context.Context, sessionToken string) error {
	return nil
}
