package utils

import (
	"time"

	"github.com/gin-gonic/gin"
)

func SetAuthCookies(c *gin.Context, accessToken, refreshToken string) {
	setCookie(c, "accessToken", accessToken, AccessTokenExpiry)
	setCookie(c, "refreshToken", refreshToken, RefreshTokenExpiry)
}

func setCookie(c *gin.Context, name, value string, expiry time.Duration) {
	secure := true
	if gin.Mode() == gin.DebugMode { // Toggle for local dev
		secure = false
	}
	c.SetCookie(name, value, int(expiry.Seconds()), "/", "", secure, true)
}

func ClearAuthCookies(c *gin.Context) {
	clearCookie(c, "accessToken")
	clearCookie(c, "refreshToken")
}

func clearCookie(c *gin.Context, name string) {
	secure := true
	if gin.Mode() == gin.DebugMode { // Toggle for local dev
		secure = false
	}
	c.SetCookie(name, "", -1, "/", "", secure, true)
}
