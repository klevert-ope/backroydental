package utils

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/o1egl/paseto"
)

const (
	// Set expiration times for access and refresh tokens.
	AccessTokenExpiry  = 24 * time.Hour
	RefreshTokenExpiry = 7 * 24 * time.Hour
)

// TokenClaims struct represents the data in the token (UserID, Role, Expiry).
type TokenClaims struct {
	UserID string    `json:"userId"`
	Role   string    `json:"role"`
	Expiry time.Time `json:"expiry"`
}

// GetSymmetricKey retrieves the symmetric key from the environment variable.
// Ensures it has the correct length (32 bytes).
func GetSymmetricKey() []byte {
	key := os.Getenv("SYMMETRIC_KEY")
	if len(key) != 32 {
		log.Fatalf("SYMMETRIC_KEY must be 32 bytes long. Current length: %d", len(key))
	}
	return []byte(key)
}

// GenerateTokens generates both the access token and refresh token for the given user ID and role.
func GenerateTokens(userID, role string) (accessToken, refreshToken string, err error) {
	// Generate the access token
	accessToken, err = generatePASEToken(userID, role, AccessTokenExpiry)
	if err != nil {
		log.Printf("Error generating access token: %v", err)
		return "", "", err
	}

	// Generate the refresh token
	refreshToken, err = generatePASEToken(userID, role, RefreshTokenExpiry)
	if err != nil {
		log.Printf("Error generating refresh token: %v", err)
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// GenerateAccessToken generates only the access token for a user.
func GenerateAccessToken(userID, role string) (string, error) {
	token, err := generatePASEToken(userID, role, AccessTokenExpiry)
	if err != nil {
		log.Printf("Error generating access token: %v", err)
		return "", err
	}
	return token, nil
}

// generatePASEToken generates a PASETO token for the given user ID, role, and expiry duration.
func generatePASEToken(userID, role string, expiry time.Duration) (string, error) {
	// Create token claims
	claims := TokenClaims{
		UserID: userID,
		Role:   role,
		Expiry: time.Now().Add(expiry),
	}

	// Encrypt the token using the symmetric key
	symmetricKey := GetSymmetricKey()
	token, err := paseto.NewV2().Encrypt(symmetricKey, claims, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return token, nil
}

// ValidateToken validates the given token string and checks for expiry and required roles.
func ValidateToken(tokenString string, requiredRoles ...string) (*TokenClaims, error) {
	claims, err := parseToken(tokenString)
	if err != nil {
		log.Printf("Token parsing failed: %v", err)
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Check if the token has expired
	if time.Now().After(claims.Expiry) {
		log.Printf("Token expired: %v", claims)
		return nil, errors.New("token expired")
	}

	// If no roles are required, any valid token is acceptable
	if len(requiredRoles) == 0 {
		return claims, nil
	}

	// Check if the user role matches any of the required roles
	for _, role := range requiredRoles {
		if claims.Role == role {
			return claims, nil
		}
	}

	// Log role mismatch for debugging purposes
	log.Printf("Insufficient permissions. Required roles: %v, found role: %v", requiredRoles, claims.Role)
	return nil, errors.New("insufficient permissions")
}

// parseToken decrypts the token and extracts claims from it.
func parseToken(tokenString string) (*TokenClaims, error) {
	var claims TokenClaims
	symmetricKey := GetSymmetricKey()

	// Decrypt the token
	err := paseto.NewV2().Decrypt(tokenString, symmetricKey, &claims, nil)
	if err != nil {
		log.Printf("Token decryption failed: %v", err)
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	return &claims, nil
}
