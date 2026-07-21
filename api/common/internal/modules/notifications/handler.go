package notifications

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
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
