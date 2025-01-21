package controllers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// rootHandler handles requests to the root path
func rootHandler(c *gin.Context) {
	// Set response status to 200 OK
	c.Status(http.StatusOK)

	// Write response body
	if _, err := c.Writer.Write([]byte("Welcome to the root route!")); err != nil {
		log.Fatalf("Error writing response: %v", err)
	}
}

// SetupRootRoute sets up routes for the application
func SetupRootRoute(router *gin.Engine) {
	// Define routes here
	router.GET("/", rootHandler)
}
