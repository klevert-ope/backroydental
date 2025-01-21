package middlewares

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiterConfig holds the configuration for the rate limiter
type RateLimiterConfig struct {
	RequestsPerSecond float64
	Burst             int
}

// rateLimiterData holds the rate limiter instance and a mutex for thread-safe operations
type rateLimiterData struct {
	limiter *rate.Limiter
	mu      sync.Mutex
}

// NewRateLimiterMiddleware creates a new rate limiter middleware
func NewRateLimiterMiddleware(config RateLimiterConfig) gin.HandlerFunc {
	// Initialize a global rate limiter
	data := &rateLimiterData{
		limiter: rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.Burst),
	}

	return func(c *gin.Context) {
		data.mu.Lock()
		defer data.mu.Unlock()

		// Check if the request can proceed
		if !data.limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		// Proceed to the next middleware/handler
		c.Next()
	}
}
