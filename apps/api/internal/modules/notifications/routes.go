package notifications

import (
	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	n := rg.Group("/notifications", middleware.RequireAuth())

	// Static segments must come before /:id
	n.POST("/read-all", handleMarkAllAsRead)

	n.GET("", handleListNotifications)
	n.POST("/:id/read", handleMarkAsRead)
}
