package notifications

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
	"github.com/pulse/api/internal/services/sse"
	"github.com/pulse/api/internal/utils"
)

// GET /api/notifications
func handleListNotifications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	notifs, total, unread, err := getNotifications(c.Request.Context(), middleware.GetUserID(c), page, limit)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch notifications")
		return
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	resp := make([]NotificationResponse, 0, len(notifs))
	for i := range notifs {
		resp = append(resp, toNotificationResponse(&notifs[i]))
	}

	utils.OKWithMeta(c, http.StatusOK, "", resp, NotifListMeta{
		Total:       total,
		Page:        page,
		Limit:       limit,
		Pages:       pages(total, limit),
		UnreadCount: unread,
	})
}

// POST /api/notifications/:id/read
func handleMarkAsRead(c *gin.Context) {
	if err := markAsRead(c.Request.Context(), middleware.GetUserID(c), c.Param("id")); err != nil {
		if err == ErrNotFound {
			utils.Fail(c, http.StatusNotFound, "Notification not found")
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Failed to update notification")
		return
	}
	utils.OK(c, http.StatusOK, "Marked as read", nil)
}

// POST /api/notifications/read-all
func handleMarkAllAsRead(c *gin.Context) {
	if err := markAllAsRead(c.Request.Context(), middleware.GetUserID(c)); err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to update notifications")
		return
	}
	utils.OK(c, http.StatusOK, "All notifications marked as read", nil)
}

// GET /api/notifications/stream
func handleStream(c *gin.Context) {
	userID := middleware.GetUserID(c)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch := sse.Global.Register(userID)
	defer sse.Global.Unregister(userID)

	// Initial ping so the client knows the stream is open.
	fmt.Fprintf(c.Writer, "event: connected\ndata: {}\n\n")
	c.Writer.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(c.Writer, "event: notification\ndata: %s\n\n", data)
			c.Writer.Flush()
		case <-ticker.C:
			fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}
