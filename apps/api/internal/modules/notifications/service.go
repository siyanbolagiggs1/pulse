package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"github.com/pulse/api/internal/services/sse"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrNotFound = errors.New("notification not found")

// ── Send ─────────────────────────────────────────────────────

// Send persists a notification and pushes it to the user's live SSE stream.
// Designed to be called as a goroutine — it never blocks the caller.
func Send(ctx context.Context, userID bson.ObjectID, nType models.NotificationType, title, message string, metadata map[string]interface{}) {
	now := time.Now().UTC()
	n := models.Notification{
		UserID:    userID,
		Type:      nType,
		Title:     title,
		Message:   message,
		IsRead:    false,
		Metadata:  metadata,
		CreatedAt: now,
	}

	result, err := database.GetCollection(models.NotificationsCollection).InsertOne(ctx, n)
	if err != nil {
		return
	}
	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		n.ID = oid
	}

	data, err := json.Marshal(toNotificationResponse(&n))
	if err != nil {
		return
	}
	sse.Global.Push(userID.Hex(), string(data))
}

// ── List ─────────────────────────────────────────────────────

func getNotifications(ctx context.Context, userID string, page, limit int) ([]models.Notification, int64, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, 0, ErrNotFound
	}

	col := database.GetCollection(models.NotificationsCollection)
	filter := bson.M{"userId": objID}

	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	unread, _ := col.CountDocuments(ctx, bson.M{"userId": objID, "isRead": false})

	skip := int64((page - 1) * limit)
	cursor, err := col.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "createdAt", Value: -1}}).
			SetSkip(skip).
			SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	var notifs []models.Notification
	if err := cursor.All(ctx, &notifs); err != nil {
		return nil, 0, 0, err
	}

	return notifs, total, unread, nil
}

// ── Mark read ─────────────────────────────────────────────────

func markAsRead(ctx context.Context, userID, notifID string) error {
	userObjID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return ErrNotFound
	}
	nObjID, err := bson.ObjectIDFromHex(notifID)
	if err != nil {
		return ErrNotFound
	}

	result, err := database.GetCollection(models.NotificationsCollection).UpdateOne(ctx,
		bson.M{"_id": nObjID, "userId": userObjID},
		bson.M{"$set": bson.M{"isRead": true}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func markAllAsRead(ctx context.Context, userID string) error {
	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return ErrNotFound
	}

	_, err = database.GetCollection(models.NotificationsCollection).UpdateMany(ctx,
		bson.M{"userId": objID, "isRead": false},
		bson.M{"$set": bson.M{"isRead": true}},
	)
	return err
}

// ── Helpers ──────────────────────────────────────────────────

func pages(total int64, limit int) int64 {
	return int64(math.Ceil(float64(total) / float64(limit)))
}

var _ = mongo.ErrNoDocuments // keep mongo import alive
