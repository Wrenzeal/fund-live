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

func TestMapAuthErrorVerificationCooldownIncludesRetry(t *testing.T) {
	status, apiErr := mapAuthError(&service.VerificationCooldownError{RetryAfter: 42 * time.Second})

	if status != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", status, http.StatusTooManyRequests)
	}
	if apiErr == nil || apiErr.Code != "VERIFICATION_CODE_COOLDOWN" || apiErr.RetryAfterSeconds != 42 {
		t.Fatalf("api error = %#v", apiErr)
	}
}

func TestAuthHandlerEmailCodeFlowSetsSessionCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	serviceStub := emailHandlerAuthService{stubAuthService: stubAuthService{}}
	handler := NewAuthHandler(serviceStub, "fundlive_session", true)
	router := gin.New()
	router.GET("/api/v1/auth/config", handler.Config)
	router.POST("/api/v1/auth/email/start", handler.StartEmailCode)
	router.POST("/api/v1/auth/email/verify", handler.VerifyEmailCode)

	configReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/config", nil)
	configRec := httptest.NewRecorder()
	router.ServeHTTP(configRec, configReq)
	if !strings.Contains(configRec.Body.String(), `"email_code_login_enabled":true`) {
		t.Fatalf("config response = %s", configRec.Body.String())
	}

	startReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/start", strings.NewReader(`{"email":"user@example.com"}`))
	startReq.Header.Set("Content-Type", "application/json")
	startRec := httptest.NewRecorder()
	router.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK || !strings.Contains(startRec.Body.String(), `"dev_code":"123456"`) {
		t.Fatalf("start status=%d body=%s", startRec.Code, startRec.Body.String())
	}

	verifyReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/verify", strings.NewReader(`{"email":"user@example.com","code":"123456"}`))
	verifyReq.Header.Set("Content-Type", "application/json")
	verifyRec := httptest.NewRecorder()
	router.ServeHTTP(verifyRec, verifyReq)
	if verifyRec.Code != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", verifyRec.Code, verifyRec.Body.String())
	}
	cookies := verifyRec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "fundlive_session" || cookies[0].Value != "session-token" || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatalf("cookies = %#v", cookies)
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

type emailHandlerAuthService struct {
	stubAuthService
}

func (emailHandlerAuthService) StartEmailCodeLogin(ctx context.Context, input domain.EmailCodeStartInput, meta domain.SessionMetadata) (*domain.EmailCodeStartResult, error) {
	return &domain.EmailCodeStartResult{
		Email:              input.Email,
		DevCode:            "123456",
		ExpiresInSeconds:   600,
		ResendAfterSeconds: 60,
	}, nil
}

func (emailHandlerAuthService) LoginWithEmailCode(ctx context.Context, input domain.EmailCodeVerifyInput, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	return &domain.AuthSessionResult{
		User: &domain.User{
			ID:            "usr_email",
			Email:         input.Email,
			DisplayName:   "user",
			Provider:      domain.AuthProviderEmailCode,
			EmailVerified: true,
		},
		SessionToken: "session-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}, nil
}

func (emailHandlerAuthService) EmailCodeLoginAvailable(ctx context.Context) bool {
	return true
}

func (stubAuthService) RegisterWithPassword(ctx context.Context, input domain.PasswordRegistrationInput, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	return nil, service.ErrInvalidCredentials
}

func (stubAuthService) LoginWithPassword(ctx context.Context, input domain.PasswordLoginInput, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	return nil, service.ErrInvalidCredentials
}

func (stubAuthService) LoginWithGoogle(ctx context.Context, input domain.GoogleLoginInput, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	return nil, service.ErrInvalidGoogleToken
}

func (stubAuthService) StartEmailCodeLogin(ctx context.Context, input domain.EmailCodeStartInput, meta domain.SessionMetadata) (*domain.EmailCodeStartResult, error) {
	return nil, service.ErrEmailCodeLoginUnavailable
}

func (stubAuthService) LoginWithEmailCode(ctx context.Context, input domain.EmailCodeVerifyInput, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	return nil, service.ErrInvalidVerificationCode
}

func (stubAuthService) EmailCodeLoginAvailable(ctx context.Context) bool {
	return false
}

func (stubAuthService) AuthenticateSession(ctx context.Context, sessionToken string) (*domain.AuthenticatedSession, error) {
	return nil, service.ErrInvalidSession
}

func (stubAuthService) LogoutByToken(ctx context.Context, sessionToken string) error {
	return nil
}
