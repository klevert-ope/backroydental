package handlers

import (
	"RoyDental/models"
	"RoyDental/services"
	"RoyDental/utils"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	UserService services.UserService
}

func NewAuthHandler(userService services.UserService) *AuthHandler {
	return &AuthHandler{
		UserService: userService,
	}
}

// Helper function to extract token from URL query parameters
func extractAccessToken(c *gin.Context) (string, error) {
	token := c.DefaultQuery("accessToken", "")
	if token == "" {
		return "", fmt.Errorf("access token is required")
	}
	return token, nil
}

// Register handles new user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	ctx := c.Request.Context()
	if err := h.UserService.ValidateAndCreateUser(ctx, &user); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Validation failed: %v", err)})
		return
	}

	createdUser, err := h.UserService.GetUserByUsername(ctx, user.Username)
	if err != nil || createdUser == nil {
		c.JSON(500, gin.H{"error": "Failed to retrieve user after creation"})
		return
	}

	c.Status(201)
}

// Login authenticates the user and returns tokens along with user info
func (h *AuthHandler) Login(c *gin.Context) {
	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	ctx := c.Request.Context()
	user, err := h.UserService.AuthenticateUser(ctx, credentials.Email, credentials.Password)
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid username or password"})
		return
	}

	accessToken, refreshToken, err := utils.GenerateTokens(strconv.FormatInt(user.ID, 10), user.Role.Name)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to generate tokens: %v", err)})
		return
	}

	c.JSON(200, gin.H{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	})
}

// RefreshToken refreshes the user's access token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// Extract token from URL query parameters
	token, err := extractAccessToken(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	claims, err := utils.ValidateToken(token, "Admin", "Doctor", "Receptionist", "Patient")
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid access token"})
		return
	}

	accessToken, err := utils.GenerateAccessToken(claims.UserID, claims.Role)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to generate access token: %v", err)})
		return
	}

	c.JSON(200, gin.H{
		"accessToken": accessToken,
	})
}

// Logoff logs the user out by clearing cookies
func (h *AuthHandler) Logoff(c *gin.Context) {
	utils.ClearAuthCookies(c)
	c.Status(200)
}

// DeleteAccount removes the user's account
func (h *AuthHandler) DeleteAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64) // Parse as int64
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid appointment ID"})
		return
	}

	ctx := c.Request.Context()
	if err := h.UserService.DeleteUser(ctx, id); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to delete user account: %v", err)})
		return
	}

	c.Status(200)
}

// SendResetCode sends a password reset code to the user's email
func (h *AuthHandler) SendResetCode(c *gin.Context) {
	var data struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	ctx := c.Request.Context()
	user, err := h.UserService.GetUserByEmail(ctx, data.Email)
	if err != nil || user == nil {
		c.JSON(404, gin.H{"error": "User not found"})
		return
	}

	code := utils.GenerateResetCode()
	if err := utils.SetResetCode(ctx, user.Email, code); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to set reset code: %v", err)})
		return
	}

	if err := utils.SendResetCodeEmail(user.Email, code); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to send reset code email: %v", err)})
		return
	}

	c.Status(200)
}

// ChangeEmail updates the user's email
func (h *AuthHandler) ChangeEmail(c *gin.Context) {
	// Extract token from URL query parameters
	token, err := extractAccessToken(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	claims, err := utils.ValidateToken(token, "Admin", "Doctor", "Receptionist", "Patient")
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid access token"})
		return
	}

	var data struct {
		CurrentPassword string `json:"current_password"`
		NewEmail        string `json:"new_email"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	ctx := c.Request.Context()
	userID, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		c.JSON(500, gin.H{"error": "Invalid user ID"})
		return
	}

	// Corrected call to UpdateUserEmail
	if err := h.UserService.UpdateUserEmail(ctx, userID, data.NewEmail); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to change email: %v", err)})
		return
	}

	c.Status(200)
}

// GetUserProfile retrieves the current user's profile
func (h *AuthHandler) GetUserProfile(c *gin.Context) {
	// Extract token from URL query parameters
	token, err := extractAccessToken(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	claims, err := utils.ValidateToken(token, "Admin", "Doctor", "Receptionist", "Patient")
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid access token"})
		return
	}

	ctx := c.Request.Context()
	userID, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		c.JSON(500, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.UserService.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		c.JSON(404, gin.H{"error": "User not found"})
		return
	}

	c.JSON(200, gin.H{"user": user})
}

// UpdateUserProfile updates the user's profile information
func (h *AuthHandler) UpdateUserProfile(c *gin.Context) {
	// Extract token from URL query parameters
	token, err := extractAccessToken(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	claims, err := utils.ValidateToken(token, "Admin", "Doctor", "Receptionist", "Patient")
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid access token"})
		return
	}

	ctx := c.Request.Context()
	userID, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		c.JSON(500, gin.H{"error": "Invalid user ID"})
		return
	}

	var updateData struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	// Update the user profile
	if err := h.UserService.UpdateUserProfile(ctx, userID, updateData.Username, updateData.Email); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to update profile: %v", err)})
		return
	}

	c.Status(200)
}

// ChangePassword updates the user's password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var data struct {
		Email       string `json:"email"`
		Code        string `json:"code"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	ctx := c.Request.Context()
	storedCode, err := utils.GetResetCode(ctx, data.Email)
	if err != nil || storedCode == nil || *storedCode != data.Code {
		c.JSON(401, gin.H{"error": "Invalid reset code"})
		return
	}

	user, err := h.UserService.GetUserByEmail(ctx, data.Email)
	if err != nil || user == nil {
		c.JSON(404, gin.H{"error": "User not found"})
		return
	}

	hashedPassword, err := utils.HashPassword(data.NewPassword)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to hash password: %v", err)})
		return
	}

	if err := h.UserService.UpdateUserPassword(ctx, user.ID, hashedPassword); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to update password: %v", err)})
		return
	}

	utils.DeleteResetCode(ctx, data.Email)
	c.Status(200)
}

// AdminManageUsers allows an admin to manage users
func (h *AuthHandler) AdminManageUsers(c *gin.Context) {
	// Extract token from URL query parameters
	token, err := extractAccessToken(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Validate the token
	claims, err := utils.ValidateToken(token, "Admin")
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid access token"})
		return
	}

	ctx := c.Request.Context()

	// Log the claims for auditing purposes (optional)
	log.Printf("Admin claims: %+v", claims)

	users, err := h.UserService.GetAllUsers(ctx)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to retrieve users: %v", err)})
		return
	}

	c.JSON(200, users)
}

// DecryptRequest represents the expected JSON request body
type DecryptRequest struct {
	Token string `json:"token" binding:"required"`
}

// DecryptHandler decrypts a PASETO token and returns the extracted claims
func (h *AuthHandler) DecryptHandler(c *gin.Context) {
	var req DecryptRequest

	// Bind JSON request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Validate and decrypt the token
	claims, err := utils.ValidateToken(req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		return
	}

	// Return the decoded claims
	c.JSON(200, gin.H{
		"userId": claims.UserID,
		"role":   claims.Role,
		"expiry": claims.Expiry,
	})
}