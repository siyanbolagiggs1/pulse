package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/utils"
)

// RequireAuthWS validates the access token for a WebSocket upgrade request.
// A browser's native WebSocket constructor can't set custom headers (the same
// constraint that forced the old SSE hook to use fetch() instead of
// EventSource), so this checks a ?token= query param first and falls back to
// the Authorization header (useful for non-browser clients/tools like wscat).
// Kept separate from RequireAuth() so the header-only REST middleware is
// never touched.
func RequireAuthWS() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			header := c.GetHeader("Authorization")
			if strings.HasPrefix(header, "Bearer ") {
				token = strings.TrimPrefix(header, "Bearer ")
			}
		}
		if token == "" {
			utils.Fail(c, http.StatusUnauthorized, "token required")
			c.Abort()
			return
		}

		claims, err := utils.ValidateAccessToken(token)
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
