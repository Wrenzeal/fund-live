package middleware

import (
	"errors"
	"net/http"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	currentUserKey        = "current_user"
	currentSessionKey     = "current_session"
	currentQuoteSourceKey = "current_quote_source"
)

// RequireAuth validates the session cookie and injects the user into the request context.
func RequireAuth(authService domain.AuthenticationService, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if user, ok := CurrentUser(c); ok && user != nil {
			if session, ok := CurrentSession(c); ok && session != nil {
				c.Next()
				return
			}
		}

		sessionToken, err := c.Cookie(cookieName)
		if err != nil || sessionToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authentication required",
				},
			})
			return
		}

		authenticated, err := authService.AuthenticateSession(c.Request.Context(), sessionToken)
		if err != nil {
			statusCode := http.StatusUnauthorized
			message := "Authentication required"
			if errors.Is(err, service.ErrSessionExpired) {
				message = "Session expired"
			}

			c.AbortWithStatusJSON(statusCode, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": message,
				},
			})
			return
		}

		c.Set(currentUserKey, authenticated.User)
		c.Set(currentSessionKey, authenticated.Session)
		c.Next()
	}
}

// CurrentUser extracts the authenticated user from Gin context.
func CurrentUser(c *gin.Context) (*domain.User, bool) {
	value, ok := c.Get(currentUserKey)
	if !ok {
		return nil, false
	}

	user, ok := value.(*domain.User)
	return user, ok
}

// CurrentSession extracts the authenticated session from Gin context.
func CurrentSession(c *gin.Context) (*domain.UserSession, bool) {
	value, ok := c.Get(currentSessionKey)
	if !ok {
		return nil, false
	}

	session, ok := value.(*domain.UserSession)
	return session, ok
}

// CurrentQuoteSource extracts the resolved quote source from Gin context.
func CurrentQuoteSource(c *gin.Context) (domain.QuoteSource, bool) {
	value, ok := c.Get(currentQuoteSourceKey)
	if !ok {
		return domain.QuoteSourceSina, false
	}

	source, ok := value.(domain.QuoteSource)
	return source, ok
}
