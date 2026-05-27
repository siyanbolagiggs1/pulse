package notifications

import (
	"time"

	"github.com/pulse/api/internal/models"
)

type NotificationResponse struct {
	ID        string                 `json:"id"`
	Type      models.NotificationType `json:"type"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	IsRead    bool                   `json:"isRead"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"createdAt"`
}

type NotifListMeta struct {
	Total       int64 `json:"total"`
	Page        int   `json:"page"`
	Limit       int   `json:"limit"`
	Pages       int64 `json:"pages"`
	UnreadCount int64 `json:"unreadCount"`
}

func toNotificationResponse(n *models.Notification) NotificationResponse {
	return NotificationResponse{
		ID:        n.ID.Hex(),
		Type:      n.Type,
		Title:     n.Title,
		Message:   n.Message,
		IsRead:    n.IsRead,
		Metadata:  n.Metadata,
		CreatedAt: n.CreatedAt,
	}
}
