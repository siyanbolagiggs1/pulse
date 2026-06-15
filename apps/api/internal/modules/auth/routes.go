package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
)

// RegisterRoutes mounts all auth routes onto the given router group.
func RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register",         handleRegister)
		auth.POST("/login",            handleLogin)
		auth.POST("/logout",           handleLogout)
		auth.POST("/refresh",          handleRefresh)
		auth.POST("/google",             handleGoogleSignIn)
		auth.GET("/verify-email/:token", handleVerifyEmail)
		auth.POST("/forgot-password",  handleForgotPassword)
		auth.POST("/reset-password/:token", handleResetPassword)

		// Protected — requires valid access token
		auth.GET("/me", middleware.RequireAuth(), handleMe)
	}
}
