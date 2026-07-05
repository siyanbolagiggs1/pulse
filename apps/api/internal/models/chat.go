package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Conversation is a two-party DM thread. UserAID/UserBID are stored in a
// canonical order (see chat.canonicalOrder) so the same pair of users always
// maps to exactly one document regardless of who initiated it — this keeps
// the unique (userAId, userBId) index meaningful. Pair validity (business<->
// promoter, admin<->business, admin<->promoter) is enforced at the service
// layer via isValidPair, not by the model.
type Conversation struct {
	ID                 bson.ObjectID `bson:"_id,omitempty"      json:"id"`
	UserAID            bson.ObjectID `bson:"userAId"            json:"userAId"`
	UserBID            bson.ObjectID `bson:"userBId"            json:"userBId"`
	LastMessageAt      time.Time     `bson:"lastMessageAt"      json:"lastMessageAt"`
	LastMessagePreview string        `bson:"lastMessagePreview" json:"lastMessagePreview"`
	UserALastReadAt    time.Time     `bson:"userALastReadAt"    json:"-"`
	UserBLastReadAt    time.Time     `bson:"userBLastReadAt"    json:"-"`
	// NeedsAdminReview is set when the AI support assistant couldn't answer a
	// message and escalated it, and cleared as soon as a real admin replies.
	NeedsAdminReview bool      `bson:"needsAdminReview"   json:"needsAdminReview"`
	CreatedAt        time.Time `bson:"createdAt"          json:"createdAt"`
}

const ConversationsCollection = "conversations"

type Message struct {
	ID             bson.ObjectID `bson:"_id,omitempty"  json:"id"`
	ConversationID bson.ObjectID `bson:"conversationId" json:"conversationId"`
	SenderID       bson.ObjectID `bson:"senderId"       json:"senderId"`
	Body           string        `bson:"body"           json:"body"`
	// IsBot marks messages sent automatically (welcome message, AI support
	// replies) rather than typed by the human behind SenderID.
	IsBot     bool      `bson:"isBot"          json:"isBot"`
	CreatedAt time.Time `bson:"createdAt"      json:"createdAt"`
}

const MessagesCollection = "messages"

// KnowledgeEntry is a (question -> answer) pair learned from a real admin
// reply in a support conversation, used to let the AI support assistant
// answer similar future questions automatically. Embedding is populated via
// the configured embeddings provider (Gemini) and never serialized to any
// API response — it's only used for in-process cosine-similarity search.
type KnowledgeEntry struct {
	ID                   bson.ObjectID `bson:"_id,omitempty"          json:"id"`
	Question             string        `bson:"question"               json:"question"`
	Answer               string        `bson:"answer"                 json:"answer"`
	Embedding            []float32     `bson:"embedding"               json:"-"`
	SourceConversationID bson.ObjectID `bson:"sourceConversationId"    json:"sourceConversationId"`
	CreatedAt            time.Time     `bson:"createdAt"               json:"createdAt"`
}

const KnowledgeCollection = "support_knowledge"
