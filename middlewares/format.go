package middlewares

import (
	"log"

	"github.com/gin-gonic/gin"
)

// RespondJSON writes a JSON response to the client.
func RespondJSON(c *gin.Context, data interface{}, status int) {
	c.JSON(status, data)
}

// HttpError logs an error and writes an HTTP error response to the client.
func HttpError(c *gin.Context, message string, status int, err error) {
	log.Printf("HTTP %d - %s: %v", status, message, err)
	c.JSON(status, gin.H{"error": message})
}
