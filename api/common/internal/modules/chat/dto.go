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
	NeedsAdminReview   bool        `json:"needsAdminReview"`
	CreatedAt          time.Time   `json:"createdAt"`
}

// AdminConversationResponse names both participants generically since admin
// oversight has no single "caller" to be relative to, and either participant
// may now be an admin, business, or promoter. Each UserSummary still carries
// its own Role so the oversight UI can label them.
type AdminConversationResponse struct {
	ID                 string      `json:"id"`
	ParticipantA       UserSummary `json:"participantA"`
	ParticipantB       UserSummary `json:"participantB"`
	LastMessageAt      time.Time   `json:"lastMessageAt"`
	LastMessagePreview string      `json:"lastMessagePreview"`
	NeedsAdminReview   bool        `json:"needsAdminReview"`
	CreatedAt          time.Time   `json:"createdAt"`
}

type BroadcastWelcomeResponse struct {
	Sent    int `json:"sent"`
	Skipped int `json:"skipped"`
}

type MessageResponse struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	IsBot          bool      `json:"isBot"`
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
		IsBot:          m.IsBot,
		CreatedAt:      m.CreatedAt,
	}
}
