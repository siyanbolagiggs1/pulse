package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/utils"
)

// RequireRole returns a middleware that allows only the specified roles.
// Must be used after RequireAuth.
//
// Usage:
//
//	router.GET("/admin/users", middleware.RequireAuth(), middleware.RequireRole("admin"), handler)
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(c *gin.Context) {
		role := GetUserRole(c)
		if !allowed[role] {
			utils.Fail(c, http.StatusForbidden, "Insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}
