package submissions

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	auth := middleware.RequireAuth()
	adminOnly := middleware.RequireRole("admin")
	// 3 submissions/hour per promoter — same limit as before, now a
	// reusable sliding-window middleware instead of an inline Redis check.
	submitLimit := middleware.RateLimit(3, time.Hour, "submissions", middleware.PerUser)

	s := rg.Group("/submissions", auth)
	{
		// Static segments before /:id to avoid Gin route conflicts.
		s.POST("/upload", handleUploadScreenshot)
		s.POST("", submitLimit, handleCreateSubmission)
		s.GET("", handleGetSubmissions)
		s.GET("/:id", handleGetSubmission)
		s.POST("/:id/approve", adminOnly, handleApproveSubmission)
		s.POST("/:id/reject", adminOnly, handleRejectSubmission)
	}
}
