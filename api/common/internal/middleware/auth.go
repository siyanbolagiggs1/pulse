package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/utils"
)

const (
	ContextUserID = "userID"
	ContextRole   = "userRole"
)

// RequireAuth validates the Bearer access token in the Authorization header.
// Sets userID and userRole in the Gin context for downstream handlers.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			utils.Fail(c, http.StatusUnauthorized, "Authorization header required")
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")
		claims, err := utils.ValidateAccessToken(tokenString)
		if err != nil {
			utils.Fail(c, http.StatusUnauthorized, "Invalid or expired token")
			c.Abort()
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextRole, claims.Role)
		c.Next()
	}
}

// GetUserID extracts the authenticated user's ID from the Gin context.
// Must be called after RequireAuth middleware.
func GetUserID(c *gin.Context) string {
	id, _ := c.Get(ContextUserID)
	return id.(string)
}

// GetUserRole extracts the authenticated user's role from the Gin context.
func GetUserRole(c *gin.Context) string {
	role, _ := c.Get(ContextRole)
	return role.(string)
}
