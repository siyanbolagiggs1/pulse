package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/config"
	"github.com/pulse/api/internal/middleware"
	"github.com/pulse/api/internal/utils"
)

const refreshTokenCookie = "refresh_token"

// POST /api/auth/register
func handleRegister(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	user, accessToken, err := register(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			utils.Fail(c, http.StatusConflict, err.Error())
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Registration failed")
		return
	}

	utils.OK(c, http.StatusCreated, "Account created. Check your email to verify.", AuthResponse{
		User:        toUserResponse(user),
		AccessToken: accessToken,
	})
}

// POST /api/auth/login
func handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	user, accessToken, refreshToken, err := login(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			utils.Fail(c, http.StatusUnauthorized, err.Error())
		case errors.Is(err, ErrEmailNotVerified):
			utils.Fail(c, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrAccountSuspended):
			utils.Fail(c, http.StatusForbidden, err.Error())
		default:
			utils.Fail(c, http.StatusInternalServerError, "Login failed")
		}
		return
	}

	setRefreshCookie(c, refreshToken)

	utils.OK(c, http.StatusOK, "Login successful", AuthResponse{
		User:        toUserResponse(user),
		AccessToken: accessToken,
	})
}

// POST /api/auth/logout
func handleLogout(c *gin.Context) {
	token, err := c.Cookie(refreshTokenCookie)
	if err == nil && token != "" {
		_ = logout(c.Request.Context(), token)
	}
	clearRefreshCookie(c)
	utils.OK(c, http.StatusOK, "Logged out", nil)
}

// POST /api/auth/refresh
func handleRefresh(c *gin.Context) {
	token, err := c.Cookie(refreshTokenCookie)
	if err != nil || token == "" {
		utils.Fail(c, http.StatusUnauthorized, "No refresh token")
		return
	}

	accessToken, err := refresh(c.Request.Context(), token)
	if err != nil {
		clearRefreshCookie(c)
		utils.Fail(c, http.StatusUnauthorized, "Session expired — please log in again")
		return
	}

	utils.OK(c, http.StatusOK, "Token refreshed", gin.H{"accessToken": accessToken})
}

// GET /api/auth/verify-email/:token
func handleVerifyEmail(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.Fail(c, http.StatusBadRequest, "Token required")
		return
	}

	if err := verifyEmail(c.Request.Context(), token); err != nil {
		if errors.Is(err, ErrInvalidToken) {
			utils.Fail(c, http.StatusBadRequest, "Invalid or expired verification link")
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Verification failed")
		return
	}

	utils.OK(c, http.StatusOK, "Email verified. You can now log in.", nil)
}

// POST /api/auth/forgot-password
func handleForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Always respond with 200 — don't reveal if email exists
	_ = forgotPassword(c.Request.Context(), req.Email)
	utils.OK(c, http.StatusOK, "If that email is registered, a reset link has been sent.", nil)
}

// POST /api/auth/reset-password/:token
func handleResetPassword(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.Fail(c, http.StatusBadRequest, "Token required")
		return
	}

	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	if err := resetPassword(c.Request.Context(), token, req.Password); err != nil {
		if errors.Is(err, ErrInvalidToken) {
			utils.Fail(c, http.StatusBadRequest, "Invalid or expired reset link")
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Password reset failed")
		return
	}

	clearRefreshCookie(c)
	utils.OK(c, http.StatusOK, "Password updated. Please log in.", nil)
}

// GET /api/auth/me  (requires auth)
func handleMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := me(c.Request.Context(), userID)
	if err != nil {
		utils.Fail(c, http.StatusNotFound, "User not found")
		return
	}
	utils.OK(c, http.StatusOK, "", toUserResponse(user))
}

// POST /api/auth/google
func handleGoogleSignIn(c *gin.Context) {
	var req GoogleSignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	user, accessToken, err := googleSignIn(c.Request.Context(), req.Credential, req.Role)
	if err != nil {
		if errors.Is(err, ErrAccountSuspended) {
			utils.Fail(c, http.StatusForbidden, err.Error())
			return
		}
		utils.Fail(c, http.StatusUnauthorized, err.Error())
		return
	}

	utils.OK(c, http.StatusOK, "Signed in with Google", AuthResponse{
		User:        toUserResponse(user),
		AccessToken: accessToken,
	})
}

// ── Cookie helpers ───────────────────────────────────────────

func setRefreshCookie(c *gin.Context, token string) {
	ttl := refreshTokenTTL(token)
	secure := config.App.Env == "production"
	c.SetCookie(
		refreshTokenCookie,
		token,
		int(ttl.Seconds()),
		"/",
		"",
		secure,
		true, // httpOnly
	)
}

func clearRefreshCookie(c *gin.Context) {
	secure := config.App.Env == "production"
	c.SetCookie(refreshTokenCookie, "", -1, "/", "", secure, true)
}

// suppress unused import if time is only used transitively
var _ = time.Now
