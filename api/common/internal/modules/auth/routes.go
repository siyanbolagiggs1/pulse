package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
)

// RegisterRoutes mounts all auth routes onto the given router group.
func RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		// Per-IP, not per-user — there's no authenticated identity yet on
		// any of these routes, and per-IP is exactly what stops credential
		// stuffing / brute-force login and registration spam.
		loginLimit := middleware.RateLimit(10, time.Hour, "auth-login", middleware.PerIP)
		registerLimit := middleware.RateLimit(5, time.Hour, "auth-register", middleware.PerIP)
		forgotPwLimit := middleware.RateLimit(3, time.Hour, "auth-forgot-password", middleware.PerIP)

		auth.POST("/register",         registerLimit, handleRegister)
		auth.POST("/login",            loginLimit, handleLogin)
		auth.POST("/logout",           handleLogout)
		auth.POST("/refresh",          handleRefresh)
		auth.POST("/google",             handleGoogleSignIn)
		auth.GET("/verify-email/:token", handleVerifyEmail)
		auth.POST("/resend-verification", handleResendVerification)
		auth.POST("/forgot-password",  forgotPwLimit, handleForgotPassword)
		auth.POST("/reset-password/:token", handleResetPassword)

		// Protected — requires valid access token
		auth.GET("/me", middleware.RequireAuth(), handleMe)
		auth.POST("/accept-terms", middleware.RequireAuth(), handleAcceptTerms)
	}
}
