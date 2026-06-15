package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	a := rg.Group("/admin",
		middleware.RequireAuth(),
		middleware.RequireRole("admin"),
	)

	a.GET("/stats", handleGetStats)

	a.GET("/users", handleListUsers)
	a.GET("/users/:id", handleGetUser)
	a.POST("/users/:id/suspend", handleSuspendUser)
	a.POST("/users/:id/unsuspend", handleUnsuspendUser)

	a.GET("/fraud-flags", handleListFraudFlags)
	a.POST("/fraud-flags/:id/resolve", handleResolveFraudFlag)

	a.GET("/withdrawals", handleListWithdrawals)
	a.POST("/withdrawals/:id/approve", handleApproveWithdrawal)
	a.POST("/withdrawals/:id/reject", handleRejectWithdrawal)

	a.GET("/social-accounts", handleListPendingSocialAccounts)
	a.POST("/social-accounts/:id/approve", handleApproveSocialAccount)
	a.POST("/social-accounts/:id/reject", handleRejectSocialAccount)
}
