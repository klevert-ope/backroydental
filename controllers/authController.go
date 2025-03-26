package controllers

import (
	"RoyDental/handlers"
	"RoyDental/middlewares"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	Handler *handlers.AuthHandler
}

// NewAuthController creates a new AuthController with the given AuthHandler
func NewAuthController(authHandler *handlers.AuthHandler) *AuthController {
	return &AuthController{
		Handler: authHandler,
	}
}

// RegisterRoutes initializes all authentication routes directly on the router
func (ac *AuthController) RegisterRoutes(router *gin.Engine) {
	// Public routes: No authentication required
	router.POST("/auth/register", ac.Handler.Register)
	router.POST("/auth/login", ac.Handler.Login)
	router.DELETE("auth/delete-account/:id", ac.Handler.DeleteAccount)
	router.POST("auth/decrypt", ac.Handler.DecryptHandler)
	router.POST("/send-reset-code", ac.Handler.SendResetCode)
	router.POST("/change-password", ac.Handler.ChangePassword)

	// Protected routes: Requires a valid token
	authGroup := router.Group("/auth").Use(middlewares.TokenAuthMiddleware())
	{
		authGroup.POST("/change-email", ac.Handler.ChangeEmail)
		authGroup.POST("/logoff", ac.Handler.Logoff)
		authGroup.GET("/user/profile", ac.Handler.GetUserProfile)
		authGroup.PUT("/user/update-profile", ac.Handler.UpdateUserProfile)
		authGroup.POST("/refresh-token", ac.Handler.RefreshToken)
	}

	// Admin routes: Requires a valid token and "Admin" role
	adminGroup := router.Group("/auth/admin").Use(
		middlewares.TokenAuthMiddleware(),
		middlewares.RoleAuthMiddleware("Admin"),
	)
	{
		adminGroup.GET("/manage-users", ac.Handler.AdminManageUsers)
	}
}
