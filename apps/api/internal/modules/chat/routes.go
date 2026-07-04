package chat

import (
	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	auth := middleware.RequireAuth()

	convs := rg.Group("/conversations", auth)
	{
		convs.POST("", handleStartConversation)
		convs.GET("", handleListConversations)
		convs.GET("/:id", handleGetConversation)
		convs.GET("/:id/messages", handleGetMessages)
		convs.POST("/:id/messages", handleSendMessage)
		convs.POST("/:id/read", handleMarkRead)
		convs.POST("/:id/typing", handleTyping)
	}

	adminConvs := rg.Group("/admin/conversations", auth, middleware.RequireRole("admin"))
	{
		adminConvs.GET("", handleAdminListConversations)
		adminConvs.GET("/:id/messages", handleAdminGetMessages)
	}
}
