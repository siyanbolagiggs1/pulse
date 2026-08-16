package chat

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	auth := middleware.RequireAuth()
	// 20 messages/hour per user. Sending a message can trigger an LLM call
	// (MaybeRespondAsSupportAI) plus an embeddings call for retrieval —
	// both cost money and hit external providers, so this route needs its
	// own tighter guardrail rather than relying on general goodwill.
	messageLimit := middleware.RateLimit(20, time.Hour, "chat-messages", middleware.PerUser)

	convs := rg.Group("/conversations", auth)
	{
		convs.POST("", handleStartConversation)
		convs.POST("/support", handleStartSupportConversation)
		convs.GET("", handleListConversations)
		convs.GET("/:id", handleGetConversation)
		convs.GET("/:id/messages", handleGetMessages)
		convs.POST("/:id/messages", messageLimit, handleSendMessage)
		convs.POST("/:id/resume-ai", handleResumeAI)
		convs.POST("/:id/read", handleMarkRead)
		convs.POST("/:id/typing", handleTyping)
	}

	adminConvs := rg.Group("/admin/conversations", auth, middleware.RequireRole("admin"))
	{
		adminConvs.GET("", handleAdminListConversations)
		adminConvs.GET("/:id/messages", handleAdminGetMessages)
		adminConvs.POST("/broadcast-welcome", handleBroadcastWelcome)
	}
}
