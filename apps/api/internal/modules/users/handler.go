package users

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
	"github.com/pulse/api/internal/utils"
)

// GET /api/users/me
func handleGetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, accounts, err := getMe(c.Request.Context(), userID)
	if err != nil {
		utils.Fail(c, http.StatusNotFound, "User not found")
		return
	}
	utils.OK(c, http.StatusOK, "", toUserResponse(user, accounts))
}

// PATCH /api/users/me
func handleUpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	user, err := updateProfile(c.Request.Context(), userID, req)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	utils.OK(c, http.StatusOK, "Profile updated", toUserResponse(user, nil))
}

// POST /api/users/social-accounts
func handleConnectSocialAccount(c *gin.Context) {
	var req ConnectSocialAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	acc, err := connectSocialAccount(c.Request.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrDuplicatePlatform), errors.Is(err, ErrDuplicateSocialAccount), errors.Is(err, ErrCooldownActive):
			utils.Fail(c, http.StatusConflict, err.Error())
		default:
			utils.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	utils.OK(c, http.StatusCreated, "Social account connected", toSocialAccountResponse(acc))
}

// GET /api/users/search
func handleSearchUsers(c *gin.Context) {
	users, err := searchUsers(c.Request.Context(), middleware.GetUserRole(c), c.Query("q"), 20)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Search failed")
		return
	}

	resp := make([]SearchUserResponse, 0, len(users))
	for i := range users {
		resp = append(resp, toSearchUserResponse(&users[i]))
	}
	utils.OK(c, http.StatusOK, "", resp)
}

// GET /api/users/banks
func handleListBanks(c *gin.Context) {
	banks, err := listBanks(c.Request.Context())
	if err != nil {
		if errors.Is(err, ErrPaystackNotConfigured) {
			utils.Fail(c, http.StatusServiceUnavailable, err.Error())
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch banks")
		return
	}

	resp := make([]BankResponse, 0, len(banks))
	for _, b := range banks {
		resp = append(resp, BankResponse{Code: b.Code, Name: b.Name})
	}
	utils.OK(c, http.StatusOK, "", resp)
}

// POST /api/users/bank-account
func handleSetBankAccount(c *gin.Context) {
	var req SetBankAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	bankAccount, err := setBankAccount(c.Request.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrPaystackNotConfigured):
			utils.Fail(c, http.StatusServiceUnavailable, err.Error())
		case errors.Is(err, ErrInvalidBankAccount):
			utils.Fail(c, http.StatusBadRequest, err.Error())
		default:
			utils.Fail(c, http.StatusInternalServerError, "Failed to save bank account")
		}
		return
	}

	utils.OK(c, http.StatusOK, "Bank account verified and saved", toBankAccountResponse(bankAccount))
}

// DELETE /api/users/me
func handleDeleteAccount(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if err := DeleteAccount(c.Request.Context(), userID); err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			utils.Fail(c, http.StatusNotFound, "User not found")
		case errors.Is(err, ErrNonZeroBalance):
			utils.Fail(c, http.StatusConflict, err.Error())
		default:
			utils.Fail(c, http.StatusInternalServerError, "Failed to delete account")
		}
		return
	}

	utils.OK(c, http.StatusOK, "Account deleted", nil)
}

// DELETE /api/users/social-accounts/:id
func handleDeleteSocialAccount(c *gin.Context) {
	id := c.Param("id")
	userID := middleware.GetUserID(c)

	if err := deleteSocialAccount(c.Request.Context(), userID, id); err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			utils.Fail(c, http.StatusNotFound, "Social account not found")
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Failed to remove social account")
		return
	}

	utils.OK(c, http.StatusOK, "Social account removed", nil)
}
