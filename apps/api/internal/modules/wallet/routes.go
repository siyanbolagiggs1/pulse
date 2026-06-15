package wallet

import (
	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	auth := middleware.RequireAuth()

	// Paystack webhook — no auth, Paystack verifies via HMAC signature.
	rg.POST("/wallet/topup/webhook", handleTopupWebhook)

	w := rg.Group("/wallet", auth)
	{
		w.GET("", handleGetWallet)
		w.GET("/transactions", handleGetTransactions)
		w.POST("/topup", handleCreateTopup)
		w.GET("/topup/verify", handleVerifyTopup)
		w.POST("/withdraw", handleWithdraw)
		w.GET("/withdrawals", handleGetWithdrawals)
	}
}
