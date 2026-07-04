package chat

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidRecipient     = errors.New("you can only message users of the opposite role (business <-> promoter)")
	ErrUserSuspended        = errors.New("cannot start a conversation with a suspended user")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrNotParticipant       = errors.New("you are not a participant in this conversation")
)

// ── Start / get conversation ──────────────────────────────────

func startOrGetConversation(ctx context.Context, callerID, callerRole, recipientID string) (*ConversationResponse, error) {
	callerObjID, err := bson.ObjectIDFromHex(callerID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	recipientObjID, err := bson.ObjectIDFromHex(recipientID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	var recipient models.User
	if err := database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"_id": recipientObjID}).Decode(&recipient); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if !isValidPair(callerRole, string(recipient.Role)) {
		return nil, ErrInvalidRecipient
	}
	if recipient.IsSuspended {
		return nil, ErrUserSuspended
	}

	var businessID, promoterID bson.ObjectID
	if callerRole == string(models.RoleBusiness) {
		businessID, promoterID = callerObjID, recipientObjID
	} else {
		businessID, promoterID = recipientObjID, callerObjID
	}

	col := database.GetCollection(models.ConversationsCollection)
	now := time.Now().UTC()

	var conv models.Conversation
	err = col.FindOneAndUpdate(ctx,
		bson.M{"businessId": businessID, "promoterId": promoterID},
		bson.M{"$setOnInsert": bson.M{
			"businessId":         businessID,
			"promoterId":         promoterID,
			"lastMessageAt":      time.Time{},
			"lastMessagePreview": "",
			"businessLastReadAt": time.Time{},
			"promoterLastReadAt": time.Time{},
			"createdAt":          now,
		}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&conv)
	if err != nil {
		return nil, err
	}

	resp := ConversationResponse{
		ID:                 conv.ID.Hex(),
		OtherParty:         toUserSummary(&recipient),
		LastMessageAt:      conv.LastMessageAt,
		LastMessagePreview: conv.LastMessagePreview,
		UnreadCount:        0,
		CreatedAt:          conv.CreatedAt,
	}
	return &resp, nil
}

func isValidPair(callerRole, recipientRole string) bool {
	return (callerRole == string(models.RoleBusiness) && recipientRole == string(models.RolePromoter)) ||
		(callerRole == string(models.RolePromoter) && recipientRole == string(models.RoleBusiness))
}

// ── List conversations ────────────────────────────────────────

func listConversations(ctx context.Context, userID, role string, page, limit int) ([]ConversationResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, ErrUserNotFound
	}

	isBusiness := role == string(models.RoleBusiness)
	filter := bson.M{"promoterId": objID}
	if isBusiness {
		filter = bson.M{"businessId": objID}
	}

	col := database.GetCollection(models.ConversationsCollection)
	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * limit)
	cursor, err := col.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "lastMessageAt", Value: -1}}).
			SetSkip(skip).
			SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var convs []models.Conversation
	if err := cursor.All(ctx, &convs); err != nil {
		return nil, 0, err
	}

	otherIDs := make([]bson.ObjectID, 0, len(convs))
	for _, c := range convs {
		if isBusiness {
			otherIDs = append(otherIDs, c.PromoterID)
		} else {
			otherIDs = append(otherIDs, c.BusinessID)
		}
	}
	usersByID := fetchUsersByID(ctx, otherIDs)

	msgCol := database.GetCollection(models.MessagesCollection)
	resp := make([]ConversationResponse, 0, len(convs))
	for _, c := range convs {
		otherID := c.BusinessID
		lastRead := c.PromoterLastReadAt
		if isBusiness {
			otherID = c.PromoterID
			lastRead = c.BusinessLastReadAt
		}

		unread, _ := msgCol.CountDocuments(ctx, bson.M{
			"conversationId": c.ID,
			"senderId":       bson.M{"$ne": objID},
			"createdAt":      bson.M{"$gt": lastRead},
		})

		other := usersByID[otherID]
		resp = append(resp, ConversationResponse{
			ID:                 c.ID.Hex(),
			OtherParty:         toUserSummary(&other),
			LastMessageAt:      c.LastMessageAt,
			LastMessagePreview: c.LastMessagePreview,
			UnreadCount:        unread,
			CreatedAt:          c.CreatedAt,
		})
	}

	return resp, total, nil
}

// getConversation returns a single conversation from the caller's point of
// view — used by the thread page to resolve the other party's name/avatar
// without fetching the whole list.
func getConversation(ctx context.Context, conversationID, userID, role string) (*ConversationResponse, error) {
	conv, convObjID, err := loadConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	userObjID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	otherID, err := otherParticipant(conv, userObjID)
	if err != nil {
		return nil, err
	}

	usersByID := fetchUsersByID(ctx, []bson.ObjectID{otherID})
	other := usersByID[otherID]

	lastRead := conv.PromoterLastReadAt
	if role == string(models.RoleBusiness) {
		lastRead = conv.BusinessLastReadAt
	}
	unread, _ := database.GetCollection(models.MessagesCollection).CountDocuments(ctx, bson.M{
		"conversationId": convObjID,
		"senderId":       bson.M{"$ne": userObjID},
		"createdAt":      bson.M{"$gt": lastRead},
	})

	return &ConversationResponse{
		ID:                 conv.ID.Hex(),
		OtherParty:         toUserSummary(&other),
		LastMessageAt:      conv.LastMessageAt,
		LastMessagePreview: conv.LastMessagePreview,
		UnreadCount:        unread,
		CreatedAt:          conv.CreatedAt,
	}, nil
}

// listAllConversations is the admin, read-only view across every conversation.
func listAllConversations(ctx context.Context, page, limit int) ([]AdminConversationResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	col := database.GetCollection(models.ConversationsCollection)
	total, err := col.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * limit)
	cursor, err := col.Find(ctx, bson.M{},
		options.Find().
			SetSort(bson.D{{Key: "lastMessageAt", Value: -1}}).
			SetSkip(skip).
			SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var convs []models.Conversation
	if err := cursor.All(ctx, &convs); err != nil {
		return nil, 0, err
	}

	ids := make([]bson.ObjectID, 0, len(convs)*2)
	for _, c := range convs {
		ids = append(ids, c.BusinessID, c.PromoterID)
	}
	usersByID := fetchUsersByID(ctx, ids)

	resp := make([]AdminConversationResponse, 0, len(convs))
	for _, c := range convs {
		biz := usersByID[c.BusinessID]
		prom := usersByID[c.PromoterID]
		resp = append(resp, AdminConversationResponse{
			ID:                 c.ID.Hex(),
			Business:           toUserSummary(&biz),
			Promoter:           toUserSummary(&prom),
			LastMessageAt:      c.LastMessageAt,
			LastMessagePreview: c.LastMessagePreview,
			CreatedAt:          c.CreatedAt,
		})
	}
	return resp, total, nil
}

func fetchUsersByID(ctx context.Context, ids []bson.ObjectID) map[bson.ObjectID]models.User {
	result := make(map[bson.ObjectID]models.User, len(ids))
	if len(ids) == 0 {
		return result
	}
	cursor, err := database.GetCollection(models.UsersCollection).Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return result
	}
	defer cursor.Close(ctx)
	var us []models.User
	if err := cursor.All(ctx, &us); err != nil {
		return result
	}
	for _, u := range us {
		result[u.ID] = u
	}
	return result
}

// ── Conversation lookup helpers ───────────────────────────────

func loadConversation(ctx context.Context, conversationID string) (*models.Conversation, bson.ObjectID, error) {
	convObjID, err := bson.ObjectIDFromHex(conversationID)
	if err != nil {
		return nil, bson.ObjectID{}, ErrConversationNotFound
	}
	var conv models.Conversation
	if err := database.GetCollection(models.ConversationsCollection).
		FindOne(ctx, bson.M{"_id": convObjID}).Decode(&conv); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, bson.ObjectID{}, ErrConversationNotFound
		}
		return nil, bson.ObjectID{}, err
	}
	return &conv, convObjID, nil
}

// otherParticipant returns the conversation's other participant relative to
// userObjID, or ErrNotParticipant if userObjID isn't in this conversation at all.
func otherParticipant(conv *models.Conversation, userObjID bson.ObjectID) (bson.ObjectID, error) {
	switch userObjID {
	case conv.BusinessID:
		return conv.PromoterID, nil
	case conv.PromoterID:
		return conv.BusinessID, nil
	default:
		return bson.ObjectID{}, ErrNotParticipant
	}
}

// ── Messages ──────────────────────────────────────────────────

// getMessages returns paginated history. When requireParticipant is false
// (the admin oversight path), callerID is ignored and no participant check runs.
func getMessages(ctx context.Context, conversationID, callerID string, requireParticipant bool, page, limit int) ([]MessageResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	conv, convObjID, err := loadConversation(ctx, conversationID)
	if err != nil {
		return nil, 0, err
	}

	if requireParticipant {
		callerObjID, err := bson.ObjectIDFromHex(callerID)
		if err != nil {
			return nil, 0, ErrUserNotFound
		}
		if _, err := otherParticipant(conv, callerObjID); err != nil {
			return nil, 0, err
		}
	}

	col := database.GetCollection(models.MessagesCollection)
	filter := bson.M{"conversationId": convObjID}
	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * limit)
	cursor, err := col.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "createdAt", Value: -1}}).
			SetSkip(skip).
			SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var msgs []models.Message
	if err := cursor.All(ctx, &msgs); err != nil {
		return nil, 0, err
	}

	resp := make([]MessageResponse, 0, len(msgs))
	for i := range msgs {
		resp = append(resp, toMessageResponse(&msgs[i]))
	}
	return resp, total, nil
}

// sendMessage persists a message and returns it plus the other participant's
// ID, so the handler can push a live "chat_message" event to them.
func sendMessage(ctx context.Context, conversationID, senderID, body string) (*MessageResponse, string, error) {
	conv, convObjID, err := loadConversation(ctx, conversationID)
	if err != nil {
		return nil, "", err
	}
	senderObjID, err := bson.ObjectIDFromHex(senderID)
	if err != nil {
		return nil, "", ErrUserNotFound
	}
	otherID, err := otherParticipant(conv, senderObjID)
	if err != nil {
		return nil, "", err
	}

	now := time.Now().UTC()
	msg := models.Message{
		ConversationID: convObjID,
		SenderID:       senderObjID,
		Body:           body,
		CreatedAt:      now,
	}
	result, err := database.GetCollection(models.MessagesCollection).InsertOne(ctx, msg)
	if err != nil {
		return nil, "", err
	}
	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		msg.ID = oid
	}

	_, _ = database.GetCollection(models.ConversationsCollection).UpdateOne(ctx,
		bson.M{"_id": convObjID},
		bson.M{"$set": bson.M{
			"lastMessageAt":      now,
			"lastMessagePreview": truncatePreview(body, 140),
		}},
	)

	resp := toMessageResponse(&msg)
	return &resp, otherID.Hex(), nil
}

func truncatePreview(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// ── Read receipts / typing ────────────────────────────────────

// markRead updates the caller's last-read timestamp and returns the other
// participant's ID so the handler can push a "read_receipt" event to them.
func markRead(ctx context.Context, conversationID, userID string) (string, error) {
	conv, convObjID, err := loadConversation(ctx, conversationID)
	if err != nil {
		return "", err
	}
	userObjID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	var field string
	switch userObjID {
	case conv.BusinessID:
		field = "businessLastReadAt"
	case conv.PromoterID:
		field = "promoterLastReadAt"
	default:
		return "", ErrNotParticipant
	}
	otherID, _ := otherParticipant(conv, userObjID)

	_, err = database.GetCollection(models.ConversationsCollection).UpdateOne(ctx,
		bson.M{"_id": convObjID},
		bson.M{"$set": bson.M{field: time.Now().UTC()}},
	)
	if err != nil {
		return "", err
	}
	return otherID.Hex(), nil
}

// verifyParticipant is used by the typing endpoint, which has nothing to
// persist — just confirms the caller belongs to this conversation and
// returns who to notify.
func verifyParticipant(ctx context.Context, conversationID, userID string) (string, error) {
	conv, _, err := loadConversation(ctx, conversationID)
	if err != nil {
		return "", err
	}
	userObjID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return "", ErrUserNotFound
	}
	otherID, err := otherParticipant(conv, userObjID)
	if err != nil {
		return "", err
	}
	return otherID.Hex(), nil
}

// ── Helpers ──────────────────────────────────────────────────

func pages(total int64, limit int) int64 {
	return int64(math.Ceil(float64(total) / float64(limit)))
}
