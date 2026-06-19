package router

import (
	"net/http"
	"strings"

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

	clientURL := strings.TrimRight(config.App.ClientURL, "/")
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}
	if clientURL == "" || clientURL == "*" ||
		(!strings.HasPrefix(clientURL, "http://") && !strings.HasPrefix(clientURL, "https://")) {
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowCredentials = false
	} else {
		// Allow both the configured origin and any Vercel preview URLs for the same project
		corsConfig.AllowOriginFunc = func(origin string) bool {
			if origin == clientURL {
				return true
			}
			// Allow Vercel preview deployments (*.vercel.app) if clientURL is on vercel.app
			if strings.HasSuffix(clientURL, ".vercel.app") && strings.HasSuffix(origin, ".vercel.app") {
				return true
			}
			return false
		}
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
