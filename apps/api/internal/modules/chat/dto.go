package chat

import (
	"time"

	"github.com/pulse/api/internal/models"
)

// ── Requests ─────────────────────────────────────────────────

type StartConversationRequest struct {
	RecipientID string `json:"recipientId" binding:"required"`
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required,max=4000"`
}

// ── Responses ────────────────────────────────────────────────

type UserSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
	Role   string `json:"role"`
}

// ConversationResponse is the caller-scoped view: "otherParty" is whichever
// participant isn't the requesting user, and unreadCount is relative to them.
type ConversationResponse struct {
	ID                 string      `json:"id"`
	OtherParty         UserSummary `json:"otherParty"`
	LastMessageAt      time.Time   `json:"lastMessageAt"`
	LastMessagePreview string      `json:"lastMessagePreview"`
	UnreadCount        int64       `json:"unreadCount"`
	CreatedAt          time.Time   `json:"createdAt"`
}

// AdminConversationResponse names both participants explicitly since admin
// oversight has no single "caller" to be relative to.
type AdminConversationResponse struct {
	ID                 string      `json:"id"`
	Business           UserSummary `json:"business"`
	Promoter           UserSummary `json:"promoter"`
	LastMessageAt      time.Time   `json:"lastMessageAt"`
	LastMessagePreview string      `json:"lastMessagePreview"`
	CreatedAt          time.Time   `json:"createdAt"`
}

type MessageResponse struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type ListMeta struct {
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Pages int64 `json:"pages"`
}

// ── Mappers ──────────────────────────────────────────────────

func toUserSummary(u *models.User) UserSummary {
	return UserSummary{ID: u.ID.Hex(), Name: u.Name, Avatar: u.Avatar, Role: string(u.Role)}
}

func toMessageResponse(m *models.Message) MessageResponse {
	return MessageResponse{
		ID:             m.ID.Hex(),
		ConversationID: m.ConversationID.Hex(),
		SenderID:       m.SenderID.Hex(),
		Body:           m.Body,
		CreatedAt:      m.CreatedAt,
	}
}
