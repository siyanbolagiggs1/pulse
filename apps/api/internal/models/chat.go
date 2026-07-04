package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Conversation is a single DM thread between one business and one promoter.
// Fixed BusinessID/PromoterID fields (rather than a generic participant list)
// keep the opposite-role validation and the unique index simple, since
// same-role messaging is never allowed.
type Conversation struct {
	ID                 bson.ObjectID `bson:"_id,omitempty"      json:"id"`
	BusinessID         bson.ObjectID `bson:"businessId"         json:"businessId"`
	PromoterID         bson.ObjectID `bson:"promoterId"         json:"promoterId"`
	LastMessageAt      time.Time     `bson:"lastMessageAt"      json:"lastMessageAt"`
	LastMessagePreview string        `bson:"lastMessagePreview" json:"lastMessagePreview"`
	BusinessLastReadAt time.Time     `bson:"businessLastReadAt" json:"-"`
	PromoterLastReadAt time.Time     `bson:"promoterLastReadAt" json:"-"`
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
