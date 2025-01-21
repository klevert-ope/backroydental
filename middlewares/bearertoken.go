package middlewares

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ValidateBearerToken validates the Bearer token in the Authorization header.
func ValidateBearerToken(expectedBearerToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the Bearer token from the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
			c.Abort()
			return
		}

		// Check if the Authorization header has the Bearer scheme
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			c.Abort()
			return
		}

		// Extract the token from the Authorization header
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Constant-time comparison to mitigate timing attacks
		if !secureCompare(token, expectedBearerToken) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Bearer Token"})
			c.Abort()
			return
		}

		// Proceed to the next middleware or handler if token is valid
		c.Next()
	}
}

// secureCompare performs a constant-time comparison of two strings to mitigate timing attacks.
func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	result := byte(0)
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

// LoggingMiddleware logs information about incoming requests.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process the request
		c.Next()

		// Log method, path, and the duration taken
		log.Printf("Request: %s %s | Duration: %v", c.Request.Method, c.Request.URL.Path, time.Since(start))
	}
}
