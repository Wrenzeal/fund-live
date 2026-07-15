package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/middleware"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/gin-gonic/gin"
)

const maxAuthRequestBodyBytes int64 = 16 * 1024

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authService       domain.AuthenticationService
	cookieName        string
	cookieSecure      bool
	googleWebClientID string
}

// NewAuthHandler creates a new AuthHandler instance.
func NewAuthHandler(authService domain.AuthenticationService, cookieName string, cookieSecure bool) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		cookieName:   cookieName,
		cookieSecure: cookieSecure,
	}
}

// SetGoogleWebClientID configures the public Google Web Client ID exposed to the frontend.
// The value is not a secret; it lets browser builds render Google Identity Services even
// when NEXT_PUBLIC_GOOGLE_CLIENT_ID was not injected at build time.
func (h *AuthHandler) SetGoogleWebClientID(clientID string) {
	h.googleWebClientID = clientID
}

type registerRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type googleLoginRequest struct {
	IDToken string `json:"id_token"`
}

type emailCodeStartRequest struct {
	Email string `json:"email"`
}

type emailCodeVerifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type authConfigResponse struct {
	GoogleClientID        string `json:"google_client_id"`
	GoogleLoginEnabled    bool   `json:"google_login_enabled"`
	EmailCodeLoginEnabled bool   `json:"email_code_login_enabled"`
}

type authSuccessResponse struct {
	User      *domain.User `json:"user"`
	ExpiresAt time.Time    `json:"expires_at"`
}

// Config returns non-sensitive authentication settings required by the browser.
func (h *AuthHandler) Config(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: authConfigResponse{
			GoogleClientID:        h.googleWebClientID,
			GoogleLoginEnabled:    h.googleWebClientID != "",
			EmailCodeLoginEnabled: h.authService.EmailCodeLoginAvailable(c.Request.Context()),
		},
	})
}

// Register creates a new password-based user account and starts a session.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if !bindAuthJSON(c, &req, "Invalid register payload") {
		return
	}

	result, err := h.authService.RegisterWithPassword(c.Request.Context(), domain.PasswordRegistrationInput{
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Password:    req.Password,
	}, requestSessionMetadata(c))
	if err != nil {
		statusCode, apiErr := mapAuthError(err)
		c.JSON(statusCode, APIResponse{
			Success: false,
			Error:   apiErr,
		})
		return
	}

	h.setSessionCookie(c, result.SessionToken, result.ExpiresAt)
	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data: authSuccessResponse{
			User:      result.User,
			ExpiresAt: result.ExpiresAt,
		},
	})
}

// Login validates email/password and starts a session.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if !bindAuthJSON(c, &req, "Invalid login payload") {
		return
	}

	result, err := h.authService.LoginWithPassword(c.Request.Context(), domain.PasswordLoginInput{
		Email:    req.Email,
		Password: req.Password,
	}, requestSessionMetadata(c))
	if err != nil {
		statusCode, apiErr := mapAuthError(err)
		c.JSON(statusCode, APIResponse{
			Success: false,
			Error:   apiErr,
		})
		return
	}

	h.setSessionCookie(c, result.SessionToken, result.ExpiresAt)
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: authSuccessResponse{
			User:      result.User,
			ExpiresAt: result.ExpiresAt,
		},
	})
}

// GoogleLogin verifies a Google ID token and creates or resumes a local account.
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	var req googleLoginRequest
	if !bindAuthJSON(c, &req, "Invalid Google login payload") {
		return
	}

	result, err := h.authService.LoginWithGoogle(c.Request.Context(), domain.GoogleLoginInput{
		IDToken: req.IDToken,
	}, requestSessionMetadata(c))
	if err != nil {
		statusCode, apiErr := mapAuthError(err)
		c.JSON(statusCode, APIResponse{
			Success: false,
			Error:   apiErr,
		})
		return
	}

	h.setSessionCookie(c, result.SessionToken, result.ExpiresAt)
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: authSuccessResponse{
			User:      result.User,
			ExpiresAt: result.ExpiresAt,
		},
	})
}

// StartEmailCode creates and sends a one-time email login challenge.
func (h *AuthHandler) StartEmailCode(c *gin.Context) {
	var req emailCodeStartRequest
	if !bindAuthJSON(c, &req, "Invalid email code request") {
		return
	}

	result, err := h.authService.StartEmailCodeLogin(c.Request.Context(), domain.EmailCodeStartInput{
		Email: req.Email,
	}, requestSessionMetadata(c))
	if err != nil {
		statusCode, apiErr := mapAuthError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: result})
}

// VerifyEmailCode consumes a one-time code and starts a normal FundLive session.
func (h *AuthHandler) VerifyEmailCode(c *gin.Context) {
	var req emailCodeVerifyRequest
	if !bindAuthJSON(c, &req, "Invalid email verification payload") {
		return
	}

	result, err := h.authService.LoginWithEmailCode(c.Request.Context(), domain.EmailCodeVerifyInput{
		Email: req.Email,
		Code:  req.Code,
	}, requestSessionMetadata(c))
	if err != nil {
		statusCode, apiErr := mapAuthError(err)
		c.JSON(statusCode, APIResponse{Success: false, Error: apiErr})
		return
	}

	h.setSessionCookie(c, result.SessionToken, result.ExpiresAt)
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: authSuccessResponse{
			User:      result.User,
			ExpiresAt: result.ExpiresAt,
		},
	})
}

// Me returns the currently authenticated user.
func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "UNAUTHORIZED",
				Message: "Authentication required",
			},
		})
		return
	}

	session, _ := middleware.CurrentSession(c)
	expiresAt := time.Time{}
	if session != nil {
		expiresAt = session.ExpiresAt
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: authSuccessResponse{
			User:      user,
			ExpiresAt: expiresAt,
		},
	})
}

// Logout revokes the current session cookie.
func (h *AuthHandler) Logout(c *gin.Context) {
	if sessionToken, err := c.Cookie(h.cookieName); err == nil {
		_ = h.authService.LogoutByToken(c.Request.Context(), sessionToken)
	}
	h.clearSessionCookie(c)

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: gin.H{
			"logged_out": true,
		},
	})
}

func (h *AuthHandler) setSessionCookie(c *gin.Context, token string, expiresAt time.Time) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		h.cookieName,
		token,
		maxAgeSeconds(expiresAt),
		"/",
		"",
		h.cookieSecure,
		true,
	)
}

func (h *AuthHandler) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.cookieName, "", -1, "/", "", h.cookieSecure, true)
}

func requestSessionMetadata(c *gin.Context) domain.SessionMetadata {
	return domain.SessionMetadata{
		UserAgent: c.Request.UserAgent(),
		IPAddress: c.ClientIP(),
	}
}

func bindAuthJSON(c *gin.Context, target interface{}, invalidMessage string) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAuthRequestBodyBytes)
	if err := c.ShouldBindJSON(target); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, APIResponse{
				Success: false,
				Error: &APIError{
					Code:    "REQUEST_TOO_LARGE",
					Message: "Authentication payload is too large",
				},
			})
			return false
		}

		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_REQUEST",
				Message: invalidMessage,
			},
		})
		return false
	}
	return true
}

func maxAgeSeconds(expiresAt time.Time) int {
	seconds := int(time.Until(expiresAt).Seconds())
	if seconds < 0 {
		return 0
	}
	return seconds
}

func mapAuthError(err error) (int, *APIError) {
	switch {
	case errors.Is(err, service.ErrInvalidEmail):
		return http.StatusBadRequest, &APIError{Code: "INVALID_EMAIL", Message: err.Error()}
	case errors.Is(err, service.ErrWeakPassword):
		return http.StatusBadRequest, &APIError{Code: "WEAK_PASSWORD", Message: err.Error()}
	case errors.Is(err, service.ErrVerificationCodeCooldown):
		var cooldownErr *service.VerificationCooldownError
		errors.As(err, &cooldownErr)
		return http.StatusTooManyRequests, &APIError{
			Code:              "VERIFICATION_CODE_COOLDOWN",
			Message:           "验证码已发送，请稍后再试。",
			RetryAfterSeconds: retryAfterSeconds(cooldownErr),
		}
	case errors.Is(err, service.ErrAuthRateLimited):
		var rateErr *service.AuthRateLimitError
		errors.As(err, &rateErr)
		return http.StatusTooManyRequests, &APIError{
			Code:              "AUTH_RATE_LIMITED",
			Message:           "请求过于频繁，请稍后再试。",
			RetryAfterSeconds: retryAfterSeconds(rateErr),
		}
	case errors.Is(err, service.ErrEmailAlreadyRegistered):
		return http.StatusConflict, &APIError{Code: "EMAIL_ALREADY_REGISTERED", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidCredentials):
		return http.StatusUnauthorized, &APIError{Code: "INVALID_CREDENTIALS", Message: err.Error()}
	case errors.Is(err, service.ErrGoogleLoginDisabled):
		return http.StatusServiceUnavailable, &APIError{Code: "GOOGLE_LOGIN_DISABLED", Message: err.Error()}
	case errors.Is(err, service.ErrGoogleEmailNotVerified):
		return http.StatusUnauthorized, &APIError{Code: "GOOGLE_EMAIL_NOT_VERIFIED", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidGoogleToken):
		return http.StatusUnauthorized, &APIError{Code: "INVALID_GOOGLE_TOKEN", Message: err.Error()}
	case errors.Is(err, service.ErrInvalidVerificationCode):
		return http.StatusBadRequest, &APIError{Code: "INVALID_VERIFICATION_CODE", Message: "验证码错误或已失效，请检查后重试。"}
	case errors.Is(err, service.ErrEmailDeliveryFailed):
		return http.StatusBadGateway, &APIError{Code: "EMAIL_DELIVERY_FAILED", Message: "验证码邮件发送失败，请稍后重试。"}
	case errors.Is(err, service.ErrEmailCodeLoginUnavailable):
		return http.StatusServiceUnavailable, &APIError{Code: "EMAIL_CODE_LOGIN_UNAVAILABLE", Message: "验证码登录暂不可用，请使用密码或 Google 登录。"}
	case errors.Is(err, service.ErrInvalidSession), errors.Is(err, service.ErrSessionExpired):
		return http.StatusUnauthorized, &APIError{Code: "UNAUTHORIZED", Message: err.Error()}
	default:
		return http.StatusInternalServerError, &APIError{Code: "AUTH_FAILED", Message: "Authentication failed"}
	}
}

func retryAfterSeconds(value interface{}) int64 {
	var retryAfter time.Duration
	switch typed := value.(type) {
	case *service.VerificationCooldownError:
		if typed != nil {
			retryAfter = typed.RetryAfter
		}
	case *service.AuthRateLimitError:
		if typed != nil {
			retryAfter = typed.RetryAfter
		}
	}
	if retryAfter <= 0 {
		return 0
	}
	return int64((retryAfter + time.Second - 1) / time.Second)
}
