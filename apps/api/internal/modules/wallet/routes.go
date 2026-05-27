package wallet

import (
	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	auth := middleware.RequireAuth()
	bizOnly := middleware.RequireRole("business")
	promoterOnly := middleware.RequireRole("promoter")

	// Stripe webhook has no auth — Stripe verifies via signature instead.
	rg.POST("/wallet/topup/webhook", handleTopupWebhook)

	w := rg.Group("/wallet", auth)
	{
		w.GET("", handleGetWallet)
		w.GET("/transactions", handleGetTransactions)

		// Business: top-up via Stripe Payment Intent.
		w.POST("/topup", bizOnly, handleCreateTopup)

		// Promoter: Stripe Connect onboarding + withdrawals.
		w.POST("/connect", promoterOnly, handleCreateConnect)
		w.GET("/connect/status", promoterOnly, handleGetConnectStatus)
		w.POST("/withdraw", promoterOnly, handleWithdraw)
		w.GET("/withdrawals", promoterOnly, handleGetWithdrawals)
	}
}
