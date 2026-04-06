package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireAdmin ensures the current authenticated user has administrator access.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authentication required",
				},
			})
			return
		}

		if !user.IsAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ADMIN_REQUIRED",
					"message": "Administrator access required",
				},
			})
			return
		}

		c.Next()
	}
}
