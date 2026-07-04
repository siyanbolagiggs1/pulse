package users

import (
	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	u := rg.Group("/users", middleware.RequireAuth())
	{
		u.GET("/me",                        handleGetMe)
		u.PATCH("/me",                      handleUpdateProfile)
		u.GET("/search",                    handleSearchUsers)
		u.POST("/social-accounts",          handleConnectSocialAccount)
		u.DELETE("/social-accounts/:id",    handleDeleteSocialAccount)
	}
}
