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

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authService  domain.AuthenticationService
	cookieName   string
	cookieSecure bool
}

// NewAuthHandler creates a new AuthHandler instance.
func NewAuthHandler(authService domain.AuthenticationService, cookieName string, cookieSecure bool) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		cookieName:   cookieName,
		cookieSecure: cookieSecure,
	}
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

type authSuccessResponse struct {
	User      *domain.User `json:"user"`
	ExpiresAt time.Time    `json:"expires_at"`
}

// Register creates a new password-based user account and starts a session.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_REQUEST",
				Message: "Invalid register payload",
			},
		})
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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_REQUEST",
				Message: "Invalid login payload",
			},
		})
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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_REQUEST",
				Message: "Invalid Google login payload",
			},
		})
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
	case errors.Is(err, service.ErrInvalidSession), errors.Is(err, service.ErrSessionExpired):
		return http.StatusUnauthorized, &APIError{Code: "UNAUTHORIZED", Message: err.Error()}
	default:
		return http.StatusInternalServerError, &APIError{Code: "AUTH_FAILED", Message: err.Error()}
	}
}
