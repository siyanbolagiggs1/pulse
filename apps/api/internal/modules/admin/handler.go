package admin

import (
	"errors"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
	"github.com/pulse/api/internal/utils"
)

// GET /api/admin/stats
func handleGetStats(c *gin.Context) {
	stats, err := getPlatformStats(c.Request.Context())
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch stats")
		return
	}
	utils.OK(c, http.StatusOK, "", stats)
}

// GET /api/admin/users
func handleListUsers(c *gin.Context) {
	var q UserListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Invalid query", err.Error())
		return
	}

	users, total, err := listUsers(c.Request.Context(), q)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}

	resp := make([]AdminUserResponse, 0, len(users))
	for i := range users {
		resp = append(resp, toAdminUserResponse(&users[i]))
	}

	utils.OKWithMeta(c, http.StatusOK, "", resp, ListMeta{
		Total: total,
		Page:  q.Page,
		Limit: q.Limit,
		Pages: int64(math.Ceil(float64(total) / float64(q.Limit))),
	})
}

// GET /api/admin/users/:id
func handleGetUser(c *gin.Context) {
	user, err := getUser(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			utils.Fail(c, http.StatusNotFound, "User not found")
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch user")
		return
	}
	utils.OK(c, http.StatusOK, "", toAdminUserResponse(user))
}

// POST /api/admin/users/:id/suspend
func handleSuspendUser(c *gin.Context) {
	var req SuspendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	adminID := middleware.GetUserID(c)
	if err := suspendUser(c.Request.Context(), adminID, c.Param("id"), req.Reason); err != nil {
		if errors.Is(err, ErrNotFound) {
			utils.Fail(c, http.StatusNotFound, "User not found")
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Failed to suspend user")
		return
	}

	utils.OK(c, http.StatusOK, "User suspended", nil)
}

// POST /api/admin/users/:id/unsuspend
func handleUnsuspendUser(c *gin.Context) {
	if err := unsuspendUser(c.Request.Context(), c.Param("id")); err != nil {
		if errors.Is(err, ErrNotFound) {
			utils.Fail(c, http.StatusNotFound, "User not found")
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Failed to unsuspend user")
		return
	}

	utils.OK(c, http.StatusOK, "User reinstated", nil)
}

// GET /api/admin/fraud-flags
func handleListFraudFlags(c *gin.Context) {
	var q FraudFlagQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Invalid query", err.Error())
		return
	}

	flags, total, err := listFraudFlags(c.Request.Context(), q)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch fraud flags")
		return
	}

	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}

	resp := make([]FraudFlagResponse, 0, len(flags))
	for i := range flags {
		resp = append(resp, toFraudFlagResponse(&flags[i]))
	}

	utils.OKWithMeta(c, http.StatusOK, "", resp, ListMeta{
		Total: total,
		Page:  q.Page,
		Limit: q.Limit,
		Pages: int64(math.Ceil(float64(total) / float64(q.Limit))),
	})
}

// POST /api/admin/fraud-flags/:id/resolve
func handleResolveFraudFlag(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	if err := resolveFraudFlag(c.Request.Context(), adminID, c.Param("id")); err != nil {
		if errors.Is(err, ErrNotFound) {
			utils.Fail(c, http.StatusNotFound, "Fraud flag not found or already resolved")
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Failed to resolve flag")
		return
	}

	utils.OK(c, http.StatusOK, "Fraud flag resolved", nil)
}

// GET /api/admin/withdrawals
func handleListWithdrawals(c *gin.Context) {
	var q WithdrawalQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Invalid query", err.Error())
		return
	}

	withdrawals, total, err := listWithdrawals(c.Request.Context(), q)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch withdrawals")
		return
	}

	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}

	resp := make([]WithdrawalAdminResponse, 0, len(withdrawals))
	for i := range withdrawals {
		resp = append(resp, toWithdrawalAdminResponse(&withdrawals[i]))
	}

	utils.OKWithMeta(c, http.StatusOK, "", resp, ListMeta{
		Total: total,
		Page:  q.Page,
		Limit: q.Limit,
		Pages: int64(math.Ceil(float64(total) / float64(q.Limit))),
	})
}

// POST /api/admin/withdrawals/:id/approve
func handleApproveWithdrawal(c *gin.Context) {
	w, err := approveWithdrawal(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			utils.Fail(c, http.StatusNotFound, "Withdrawal not found")
		case errors.Is(err, ErrNotReviewable):
			utils.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrNoConnectAccount):
			utils.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrStripeNotConfigured):
			utils.Fail(c, http.StatusServiceUnavailable, err.Error())
		default:
			utils.Fail(c, http.StatusInternalServerError, "Failed to approve withdrawal")
		}
		return
	}

	utils.OK(c, http.StatusOK, "Withdrawal approved and transfer initiated", toWithdrawalAdminResponse(w))
}

// POST /api/admin/withdrawals/:id/reject
func handleRejectWithdrawal(c *gin.Context) {
	var req RejectWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	w, err := rejectWithdrawal(c.Request.Context(), c.Param("id"), req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			utils.Fail(c, http.StatusNotFound, "Withdrawal not found")
		case errors.Is(err, ErrNotReviewable):
			utils.Fail(c, http.StatusBadRequest, err.Error())
		default:
			utils.Fail(c, http.StatusInternalServerError, "Failed to reject withdrawal")
		}
		return
	}

	utils.OK(c, http.StatusOK, "Withdrawal rejected and balance refunded", toWithdrawalAdminResponse(w))
}
