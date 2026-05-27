package campaigns

import (
	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	auth := middleware.RequireAuth()
	bizOnly := middleware.RequireRole("business")

	c := rg.Group("/campaigns")
	{
		// Static segment /my must be registered before the wildcard /:id.
		c.GET("", auth, handleGetCampaigns)
		c.POST("", auth, bizOnly, handleCreateCampaign)
		c.GET("/my", auth, bizOnly, handleGetMyCampaigns)
		c.GET("/:id", auth, handleGetCampaign)
		c.PATCH("/:id", auth, bizOnly, handleUpdateCampaign)
		c.DELETE("/:id", auth, bizOnly, handleDeleteCampaign)
	}
}
