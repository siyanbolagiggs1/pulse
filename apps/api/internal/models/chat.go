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
	CreatedAt          time.Time     `bson:"createdAt"          json:"createdAt"`
}

const ConversationsCollection = "conversations"

type Message struct {
	ID             bson.ObjectID `bson:"_id,omitempty"  json:"id"`
	ConversationID bson.ObjectID `bson:"conversationId" json:"conversationId"`
	SenderID       bson.ObjectID `bson:"senderId"       json:"senderId"`
	Body           string        `bson:"body"           json:"body"`
	CreatedAt      time.Time     `bson:"createdAt"      json:"createdAt"`
}

const MessagesCollection = "messages"
