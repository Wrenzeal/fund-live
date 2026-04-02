// Package middleware contains HTTP middleware functions.
package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const defaultAllowHeaders = "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With"

// CORS returns a middleware that handles Cross-Origin Resource Sharing.
// Allowed origins must be explicitly configured when credentialed browser access is needed.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		normalized := normalizeOrigin(origin)
		if normalized == "" {
			continue
		}
		allowedSet[normalized] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := normalizeOrigin(c.GetHeader("Origin"))
		if origin == "" {
			c.Next()
			return
		}

		if _, ok := allowedSet[origin]; !ok {
			if isPreflightRequest(c) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}

		headers := c.Writer.Header()
		headers.Add("Vary", "Origin")
		headers.Add("Vary", "Access-Control-Request-Method")
		headers.Add("Vary", "Access-Control-Request-Headers")
		headers.Set("Access-Control-Allow-Origin", origin)
		headers.Set("Access-Control-Allow-Credentials", "true")
		headers.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		requestHeaders := strings.TrimSpace(c.GetHeader("Access-Control-Request-Headers"))
		if requestHeaders == "" {
			requestHeaders = defaultAllowHeaders
		}
		headers.Set("Access-Control-Allow-Headers", requestHeaders)

		if isPreflightRequest(c) {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Logger returns a middleware that logs request details.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Printf("[GIN] %v | %3d | %13v | %15s | %-7s %s\n",
			time.Now().Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)
	}
}

// Recovery returns a middleware that recovers from panics.
func Recovery() gin.HandlerFunc {
	return gin.Recovery()
}

func normalizeOrigin(origin string) string {
	origin = strings.TrimSpace(strings.ToLower(origin))
	return strings.TrimRight(origin, "/")
}

func isPreflightRequest(c *gin.Context) bool {
	return c.Request.Method == http.MethodOptions && strings.TrimSpace(c.GetHeader("Access-Control-Request-Method")) != ""
}
