package middleware

import (
	"strings"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/gin-gonic/gin"
)

// ResolveViewer optionally authenticates the current request and resolves the effective quote source.
// Invalid or missing session cookies do not abort the request; protected routes still rely on RequireAuth.
func ResolveViewer(authService domain.AuthenticationService, cookieName string, defaultQuoteSource domain.QuoteSource) gin.HandlerFunc {
	defaultQuoteSource = domain.ResolveQuoteSource(defaultQuoteSource, domain.QuoteSourceSina)

	return func(c *gin.Context) {
		source := defaultQuoteSource

		sessionToken, err := c.Cookie(cookieName)
		if err == nil && strings.TrimSpace(sessionToken) != "" {
			authenticated, authErr := authService.AuthenticateSession(c.Request.Context(), sessionToken)
			if authErr == nil && authenticated != nil && authenticated.User != nil {
				c.Set(currentUserKey, authenticated.User)
				c.Set(currentSessionKey, authenticated.Session)
				source = domain.ResolveQuoteSource(authenticated.User.PreferredQuoteSource, defaultQuoteSource)
			}
		}

		c.Set(currentQuoteSourceKey, source)
		c.Request = c.Request.WithContext(domain.WithQuoteSource(c.Request.Context(), source))
		c.Next()
	}
}
