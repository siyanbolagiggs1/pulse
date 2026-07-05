package chat

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pulse/api/internal/config"
	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"github.com/pulse/api/internal/services/ai"
	"github.com/pulse/api/internal/services/ws"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const escalationMessageBody = "Thanks for reaching out — this looks like something an admin should personally look into. I've flagged it and someone from our team will get back to you soon."

const (
	kbSimilarityThreshold = 0.80
	kbMaxMatches          = 3
)

// MaybeRespondAsSupportAI is fire-and-forget-called after a message is sent
// into a conversation. It only acts when the sender is a real user (not the
// support admin/bot itself) and the other participant is the configured
// support admin. It either answers automatically — casual conversation, or
// something matching a previously-learned admin answer — or sends the fixed
// escalation reply and flags the conversation for a human. No-op if no AI
// provider or support admin is configured.
func MaybeRespondAsSupportAI(ctx context.Context, conversationID, senderID, body string) {
	if config.App.GroqAPIKey == "" && config.App.GeminiAPIKey == "" {
		return
	}
	if config.App.SupportAdminEmail == "" {
		return
	}

	conv, _, err := loadConversation(ctx, conversationID)
	if err != nil {
		return
	}
	senderObjID, err := bson.ObjectIDFromHex(senderID)
	if err != nil {
		return
	}

	var supportAdmin models.User
	if err := database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"email": config.App.SupportAdminEmail}).Decode(&supportAdmin); err != nil {
		return
	}
	// Never respond to the support admin's own messages (real replies or bot
	// replies both use supportAdmin.ID as sender).
	if senderObjID == supportAdmin.ID {
		return
	}
	other, err := otherParticipant(conv, senderObjID)
	if err != nil || other != supportAdmin.ID {
		return
	}

	kbContext := buildKnowledgeContext(ctx, body)
	systemPrompt := supportSystemPrompt(kbContext)

	reply, err := ai.Reply(ctx, systemPrompt, body)
	escalate := err != nil || isEscalationSignal(reply)
	if escalate {
		reply = escalationMessageBody
	}

	msg, otherPartyID, err := sendBotMessage(ctx, conv, supportAdmin.ID, reply)
	if err != nil {
		return
	}
	if escalate {
		setNeedsAdminReview(ctx, conv.ID, true)
	}

	ws.Global.Push(otherPartyID, ws.Envelope{Type: "chat_message", Data: msg})
}

func isEscalationSignal(reply string) bool {
	trimmed := strings.ToUpper(strings.Trim(strings.TrimSpace(reply), ".! "))
	return trimmed == "" || trimmed == "ESCALATE"
}

func supportSystemPrompt(kbContext string) string {
	base := `You are Pulse's automated support assistant, replying inside a live conversation with a user on behalf of the support team.

Rules:
- If the message is casual conversation (a greeting, thanks, small talk, or a simple check-in), reply warmly and briefly as Pulse support.
- If the message closely matches one of the previously-answered questions below, answer it the same way in your own words — never mention that you're referencing past answers.
- If it's a real question or issue you have no reliable information about, respond with EXACTLY the single word ESCALATE and nothing else.
- Never invent specifics about a particular user's account, balance, campaign, or submission — escalate those.
- Keep replies short (2-4 sentences), friendly, plain text, no markdown.`

	if kbContext == "" {
		return base + "\n\nPreviously answered questions: (none yet)"
	}
	return base + "\n\nPreviously answered questions:\n" + kbContext
}

// buildKnowledgeContext embeds question and returns a formatted block of the
// closest previously-learned Q&A pairs above the similarity threshold, or ""
// if embeddings aren't configured or nothing matched closely enough.
func buildKnowledgeContext(ctx context.Context, question string) string {
	if config.App.GeminiAPIKey == "" {
		return ""
	}
	embedding, err := ai.Embed(ctx, question)
	if err != nil {
		return ""
	}

	cursor, err := database.GetCollection(models.KnowledgeCollection).Find(ctx, bson.M{})
	if err != nil {
		return ""
	}
	defer cursor.Close(ctx)

	var entries []models.KnowledgeEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return ""
	}

	type scored struct {
		entry models.KnowledgeEntry
		score float64
	}
	var candidates []scored
	for _, e := range entries {
		if len(e.Embedding) == 0 {
			continue
		}
		if s := ai.CosineSimilarity(embedding, e.Embedding); s >= kbSimilarityThreshold {
			candidates = append(candidates, scored{e, s})
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	if len(candidates) > kbMaxMatches {
		candidates = candidates[:kbMaxMatches]
	}

	var sb strings.Builder
	for _, c := range candidates {
		fmt.Fprintf(&sb, "Q: %s\nA: %s\n\n", c.entry.Question, c.entry.Answer)
	}
	return sb.String()
}

// CaptureSupportKnowledge is fire-and-forget-called whenever a real admin
// (via the normal send-message HTTP path — bot replies never go through it)
// sends a message in a conversation. It pairs that reply with the other
// participant's most recent message and stores it for future similarity
// matching, then clears the conversation's escalation flag. No-op if
// embeddings aren't configured or there's no preceding user message to pair.
func CaptureSupportKnowledge(ctx context.Context, conversationID, adminSenderID, answerBody string) {
	if config.App.GeminiAPIKey == "" {
		return
	}

	conv, _, err := loadConversation(ctx, conversationID)
	if err != nil {
		return
	}
	adminObjID, err := bson.ObjectIDFromHex(adminSenderID)
	if err != nil {
		return
	}

	var questionMsg models.Message
	err = database.GetCollection(models.MessagesCollection).FindOne(ctx, bson.M{
		"conversationId": conv.ID,
		"senderId":       bson.M{"$ne": adminObjID},
	}, options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})).Decode(&questionMsg)
	if err != nil {
		return
	}

	embedding, err := ai.Embed(ctx, questionMsg.Body)
	if err != nil {
		return
	}

	entry := models.KnowledgeEntry{
		Question:             questionMsg.Body,
		Answer:               answerBody,
		Embedding:            embedding,
		SourceConversationID: conv.ID,
		CreatedAt:            time.Now().UTC(),
	}
	_, _ = database.GetCollection(models.KnowledgeCollection).InsertOne(ctx, entry)

	setNeedsAdminReview(ctx, conv.ID, false)
}

func setNeedsAdminReview(ctx context.Context, convID bson.ObjectID, needs bool) {
	_, _ = database.GetCollection(models.ConversationsCollection).UpdateOne(ctx,
		bson.M{"_id": convID},
		bson.M{"$set": bson.M{"needsAdminReview": needs}},
	)
}
