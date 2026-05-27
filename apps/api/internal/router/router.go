package router

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/config"
	"github.com/pulse/api/internal/modules/admin"
	"github.com/pulse/api/internal/modules/auth"
	"github.com/pulse/api/internal/modules/campaigns"
	"github.com/pulse/api/internal/modules/notifications"
	"github.com/pulse/api/internal/modules/submissions"
	"github.com/pulse/api/internal/modules/users"
	"github.com/pulse/api/internal/modules/wallet"
	"github.com/pulse/api/internal/utils"
)

func Setup() *gin.Engine {
	if config.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}
	if config.App.ClientURL == "*" {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowOrigins = []string{config.App.ClientURL}
	}
	r.Use(cors.New(corsConfig))

	r.Static("/uploads", config.App.UploadDir)

	r.GET("/health", func(c *gin.Context) {
		utils.OK(c, http.StatusOK, "Pulse API is running", gin.H{
			"status": "ok",
			"env":    config.App.Env,
		})
	})

	api := r.Group("/api")

	// ── Mounted modules ──────────────────────────────────────
	auth.RegisterRoutes(api)
	users.RegisterRoutes(api)
	campaigns.RegisterRoutes(api)
	submissions.RegisterRoutes(api)
	wallet.RegisterRoutes(api)
	admin.RegisterRoutes(api)
	notifications.RegisterRoutes(api)

	return r
}
