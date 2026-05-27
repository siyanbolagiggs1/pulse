package submissions

import (
	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	auth := middleware.RequireAuth()
	promoterOnly := middleware.RequireRole("promoter")
	adminOnly := middleware.RequireRole("admin")

	s := rg.Group("/submissions", auth)
	{
		// Static segments before /:id to avoid Gin route conflicts.
		s.POST("/upload", promoterOnly, handleUploadScreenshot)
		s.POST("", promoterOnly, handleCreateSubmission)
		s.GET("", handleGetSubmissions)
		s.GET("/:id", handleGetSubmission)
		s.POST("/:id/approve", adminOnly, handleApproveSubmission)
		s.POST("/:id/reject", adminOnly, handleRejectSubmission)
	}
}
