package chat

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/pulse/api/internal/config"
	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"github.com/pulse/api/internal/services/ws"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidRecipient     = errors.New("you cannot start a conversation with a user of that role")
	ErrUserSuspended        = errors.New("cannot start a conversation with a suspended user")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrNotParticipant       = errors.New("you are not a participant in this conversation")
)

const welcomeMessageBody = "Welcome to Pulse! If you ever run into issues, have feedback, or just want to flag something, feel free to message us right here — we read every message."

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

	conv, err := getOrCreateConversation(ctx, callerObjID, recipientObjID)
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

// canonicalOrder returns (a, b) such that a's hex ObjectID string sorts
// lexicographically <= b's, so the same two users always produce the same
// (UserAID, UserBID) pair regardless of who initiates — required for the
// unique (userAId, userBId) index to prevent duplicate conversations.
func canonicalOrder(id1, id2 bson.ObjectID) (bson.ObjectID, bson.ObjectID) {
	if id1.Hex() <= id2.Hex() {
		return id1, id2
	}
	return id2, id1
}

// getOrCreateConversation is the shared get-or-create used by both the HTTP
// start-conversation path and the system-initiated welcome message path
// (which has no "caller role" to check via isValidPair).
func getOrCreateConversation(ctx context.Context, userID1, userID2 bson.ObjectID) (*models.Conversation, error) {
	userAID, userBID := canonicalOrder(userID1, userID2)

	col := database.GetCollection(models.ConversationsCollection)
	now := time.Now().UTC()

	var conv models.Conversation
	err := col.FindOneAndUpdate(ctx,
		bson.M{"userAId": userAID, "userBId": userBID},
		bson.M{"$setOnInsert": bson.M{
			"userAId":            userAID,
			"userBId":            userBID,
			"lastMessageAt":      time.Time{},
			"lastMessagePreview": "",
			"userALastReadAt":    time.Time{},
			"userBLastReadAt":    time.Time{},
			"createdAt":          now,
		}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&conv)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// allowedPairs enumerates every valid (role, role) combination for starting a
// conversation, keyed order-independently. business<->business, promoter<->
// promoter, and admin<->admin are all intentionally absent (rejected).
var allowedPairs = map[[2]string]bool{
	{string(models.RoleBusiness), string(models.RolePromoter)}: true,
	{string(models.RolePromoter), string(models.RoleBusiness)}: true,
	{string(models.RoleAdmin), string(models.RoleBusiness)}:    true,
	{string(models.RoleBusiness), string(models.RoleAdmin)}:    true,
	{string(models.RoleAdmin), string(models.RolePromoter)}:    true,
	{string(models.RolePromoter), string(models.RoleAdmin)}:    true,
}

func isValidPair(callerRole, recipientRole string) bool {
	return allowedPairs[[2]string{callerRole, recipientRole}]
}

// ── List conversations ────────────────────────────────────────

func listConversations(ctx context.Context, userID string, page, limit int) ([]ConversationResponse, int64, error) {
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

	filter := bson.M{"$or": bson.A{
		bson.M{"userAId": objID},
		bson.M{"userBId": objID},
	}}

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
	for i := range convs {
		if other, err := otherParticipant(&convs[i], objID); err == nil {
			otherIDs = append(otherIDs, other)
		}
	}
	usersByID := fetchUsersByID(ctx, otherIDs)

	msgCol := database.GetCollection(models.MessagesCollection)
	resp := make([]ConversationResponse, 0, len(convs))
	for _, c := range convs {
		otherID, err := otherParticipant(&c, objID)
		if err != nil {
			continue
		}
		lastRead := myLastReadAt(&c, objID)

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
func getConversation(ctx context.Context, conversationID, userID string) (*ConversationResponse, error) {
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

	lastRead := myLastReadAt(conv, userObjID)
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
		ids = append(ids, c.UserAID, c.UserBID)
	}
	usersByID := fetchUsersByID(ctx, ids)

	resp := make([]AdminConversationResponse, 0, len(convs))
	for _, c := range convs {
		a := usersByID[c.UserAID]
		b := usersByID[c.UserBID]
		resp = append(resp, AdminConversationResponse{
			ID:                 c.ID.Hex(),
			ParticipantA:       toUserSummary(&a),
			ParticipantB:       toUserSummary(&b),
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
	case conv.UserAID:
		return conv.UserBID, nil
	case conv.UserBID:
		return conv.UserAID, nil
	default:
		return bson.ObjectID{}, ErrNotParticipant
	}
}

// myLastReadAt returns the caller's own last-read timestamp for this
// conversation (mirrors otherParticipant's field-matching pattern).
func myLastReadAt(conv *models.Conversation, userObjID bson.ObjectID) time.Time {
	if userObjID == conv.UserBID {
		return conv.UserBLastReadAt
	}
	return conv.UserALastReadAt
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
	case conv.UserAID:
		field = "userALastReadAt"
	case conv.UserBID:
		field = "userBLastReadAt"
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

// ── Support welcome message ────────────────────────────────────

// SendWelcomeMessage sends a one-time welcome message from the designated
// support admin (config.App.SupportAdminEmail) to the given user. Safe to
// call more than once for the same user — a no-op if they already have any
// message in their conversation with support (whether from a previous
// welcome, or because they messaged support first). Never blocks or panics;
// callers (registration hook, backfill) are expected to only log failures.
func SendWelcomeMessage(ctx context.Context, recipientUserID string) (sent bool, err error) {
	if config.App.SupportAdminEmail == "" {
		return false, nil
	}

	var supportAdmin models.User
	if err := database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"email": config.App.SupportAdminEmail}).Decode(&supportAdmin); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, fmt.Errorf("support admin account %q not found", config.App.SupportAdminEmail)
		}
		return false, err
	}
	if supportAdmin.Role != models.RoleAdmin {
		return false, fmt.Errorf("configured SUPPORT_ADMIN_EMAIL %q is not an admin account", config.App.SupportAdminEmail)
	}

	recipientObjID, err := bson.ObjectIDFromHex(recipientUserID)
	if err != nil {
		return false, ErrUserNotFound
	}
	if recipientObjID == supportAdmin.ID {
		return false, nil
	}

	conv, err := getOrCreateConversation(ctx, supportAdmin.ID, recipientObjID)
	if err != nil {
		return false, err
	}

	existingCount, err := database.GetCollection(models.MessagesCollection).
		CountDocuments(ctx, bson.M{"conversationId": conv.ID})
	if err != nil {
		return false, err
	}
	if existingCount > 0 {
		return false, nil
	}

	msg, otherPartyID, err := sendMessage(ctx, conv.ID.Hex(), supportAdmin.ID.Hex(), welcomeMessageBody)
	if err != nil {
		return false, err
	}

	ws.Global.Push(otherPartyID, ws.Envelope{Type: "chat_message", Data: msg})

	return true, nil
}

// broadcastWelcomeMessages sends the welcome message to every business/
// promoter user who doesn't already have one (via SendWelcomeMessage's own
// idempotency check). Safe to call repeatedly — e.g. re-running to catch
// stragglers doesn't create duplicate welcomes.
func broadcastWelcomeMessages(ctx context.Context) (sent int, skipped int, err error) {
	cursor, err := database.GetCollection(models.UsersCollection).Find(ctx, bson.M{
		"role": bson.M{"$in": bson.A{string(models.RoleBusiness), string(models.RolePromoter)}},
	})
	if err != nil {
		return 0, 0, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return 0, 0, err
	}

	for _, u := range users {
		wasSent, sendErr := SendWelcomeMessage(ctx, u.ID.Hex())
		if sendErr != nil {
			skipped++
			continue
		}
		if wasSent {
			sent++
		} else {
			skipped++
		}
	}
	return sent, skipped, nil
}

// ── Helpers ──────────────────────────────────────────────────

func pages(total int64, limit int) int64 {
	return int64(math.Ceil(float64(total) / float64(limit)))
}
