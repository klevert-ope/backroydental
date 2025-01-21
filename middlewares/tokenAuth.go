package middlewares

import (
	"RoyDental/utils"
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ContextKey defines a custom context key type to store user details in the context.
type contextKey string

const (
	// Define the keys used to store userID and userRole in the context
	userIDKey   contextKey = "userID"
	userRoleKey contextKey = "userRole"
)

// TokenAuthMiddleware validates the token and adds user details to the request context.
func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the accessToken from the URL query parameter.
		token := c.DefaultQuery("accessToken", "")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing access token"})
			c.Abort()
			return
		}

		// Validate the token and extract claims.
		claims, err := utils.ValidateToken(token, "Admin", "Doctor", "Receptionist", "Patient")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Add user details (UserID and Role) to the context for later use in handlers.
		ctx := context.WithValue(c.Request.Context(), userIDKey, claims.UserID)
		ctx = context.WithValue(ctx, userRoleKey, claims.Role)
		c.Request = c.Request.WithContext(ctx)

		// Continue to the next middleware/handler.
		c.Next()
	}
}

// RoleAuthMiddleware restricts access to users with the specified role.
func RoleAuthMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user role from context.
		role, err := ExtractUserRoleFromContext(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found in context"})
			c.Abort()
			return
		}

		// Check if the user's role matches the required role.
		if role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient privileges"})
			c.Abort()
			return
		}

		// Role matches, so continue processing the request.
		c.Next()
	}
}

// ExtractUserIDFromContext retrieves the userID from the context.
func ExtractUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok {
		return "", errors.New("user ID not found in context")
	}
	return userID, nil
}

// ExtractUserRoleFromContext retrieves the user role from the context.
func ExtractUserRoleFromContext(ctx context.Context) (string, error) {
	userRole, ok := ctx.Value(userRoleKey).(string)
	if !ok {
		return "", errors.New("user role not found in context")
	}
	return userRole, nil
}
