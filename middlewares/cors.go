package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CorsConfig holds CORS configuration settings.
type CorsConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

// CorsMiddleware creates a CORS middleware based on the provided configuration.
func CorsMiddleware(config *CorsConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if contains(config.AllowedOrigins, origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", commaSeparated(config.AllowedMethods))
		c.Header("Access-Control-Allow-Headers", commaSeparated(config.AllowedHeaders))
		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "deny")
		c.Header("X-XSS-Protection", "1; mode=block")

		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusOK)
			c.Abort()
			return
		}

		c.Next()
	}
}

func contains(arr []string, val string) bool {
	for _, item := range arr {
		if item == val {
			return true
		}
	}
	return false
}

func commaSeparated(arr []string) string {
	return strings.Join(arr, ", ")
}
