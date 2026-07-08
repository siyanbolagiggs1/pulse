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
const humanRequestedMessageBody = "Of course — I've flagged this conversation so a member of our team can jump in and help you directly."
const aiModeResumedMessageBody = "You're back in AI mode — I'll answer with quick, automated replies again. Just ask for a human anytime if you'd rather wait for our team."

const (
	kbSimilarityThreshold = 0.80
	kbMaxMatches          = 3
)

// humanHandoffPhrases catches common ways of asking for a real person. This
// runs before any AI call — a match escalates immediately and deterministically,
// rather than depending on the model to always honor the system-prompt rule.
var humanHandoffPhrases = []string{
	"talk to a human", "talk to human", "speak to a human", "speak to human",
	"speak with a human", "chat with a human", "connect me to a human",
	"real person", "actual person", "human agent", "human support",
	"live agent", "customer service", "talk to someone", "speak to someone",
	"speak with someone", "talk to support", "speak to support", "speak with support",
	"talk to an admin", "speak to an admin", "speak with an admin",
	"talk to a person", "speak to a person", "need a human", "want a human",
	"get me a human", "representative please", "human being", "human rep",
	"actual human",
}

// wantsHumanHandoff reports whether body is asking to be connected to a
// real person rather than the bot — this always escalates, even for
// questions the bot could otherwise answer.
func wantsHumanHandoff(body string) bool {
	lower := strings.ToLower(body)
	for _, phrase := range humanHandoffPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

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
	// Once a conversation is escalated, the AI stops replying entirely —
	// the thread stays in human mode until an admin answers and
	// CaptureSupportKnowledge clears the flag.
	if conv.NeedsAdminReview {
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

	var reply string
	var escalate bool

	if wantsHumanHandoff(body) {
		reply = humanRequestedMessageBody
		escalate = true
	} else {
		kbContext := buildKnowledgeContext(ctx, body)
		systemPrompt := supportSystemPrompt(kbContext)

		var replyErr error
		reply, replyErr = ai.Reply(ctx, systemPrompt, body)
		escalate = replyErr != nil || isEscalationSignal(reply)
		if escalate {
			reply = escalationMessageBody
		}
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

// ResumeAISupport lets the user manually switch a support conversation back
// to AI mode after it was escalated to a human — the inverse of
// wantsHumanHandoff. Restricted to the caller's actual support thread (the
// other participant must be the configured support admin), same guard
// MaybeRespondAsSupportAI applies before ever generating a reply.
func ResumeAISupport(ctx context.Context, conversationID, userID string) (*MessageResponse, string, error) {
	if config.App.SupportAdminEmail == "" {
		return nil, "", ErrSupportNotConfigured
	}

	conv, _, err := loadConversation(ctx, conversationID)
	if err != nil {
		return nil, "", err
	}
	userObjID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, "", ErrUserNotFound
	}

	var supportAdmin models.User
	if err := database.GetCollection(models.UsersCollection).
		FindOne(ctx, bson.M{"email": config.App.SupportAdminEmail}).Decode(&supportAdmin); err != nil {
		return nil, "", ErrSupportNotConfigured
	}

	other, err := otherParticipant(conv, userObjID)
	if err != nil {
		return nil, "", err
	}
	if other != supportAdmin.ID {
		return nil, "", ErrInvalidRecipient
	}

	setNeedsAdminReview(ctx, conv.ID, false)

	return sendBotMessage(ctx, conv, supportAdmin.ID, aiModeResumedMessageBody)
}

func isEscalationSignal(reply string) bool {
	trimmed := strings.ToUpper(strings.Trim(strings.TrimSpace(reply), ".! "))
	return trimmed == "" || trimmed == "ESCALATE"
}

// pulseAppContext grounds the model in what Pulse actually is, so general
// "what is this / how does it work" questions can be answered directly
// instead of escalating or — worse — the model guessing and getting it
// wrong. Keep this in sync with the "What Is Pulse" / "Core Flow" sections
// of CLAUDE.md if the product model changes.
const pulseAppContext = `About Pulse, the platform you support:
Pulse is a social engagement marketplace. Businesses create repost campaigns (also shown as "adverts") with a budget and a payout rate. Promoters — everyday users — earn money by reposting those campaigns on Instagram or Twitter/X and submitting proof (the post URL plus a screenshot). An admin reviews each submission and approves or rejects it. Pulse takes a platform commission (20% by default) out of each approved payout.

Core flow: a business creates a campaign → a promoter accepts it if they meet the campaign's follower/engagement minimums → the promoter reposts and submits proof → an admin reviews the submission → on approval, the payout lands in the promoter's pending balance, then becomes available after a 48-hour hold → the promoter withdraws it via Stripe Connect. Businesses fund their spending via a wallet topped up through Stripe.

Other concepts: influence score (0-100, based on followers/engagement/account age/track record) scales how much a promoter earns per repost; trust score (starts at 50) reflects a promoter's reliability and drops on rejected or fraudulent submissions.`

func supportSystemPrompt(kbContext string) string {
	base := pulseAppContext + `

You are Pulse's automated support assistant, replying inside a live conversation with a user on behalf of the support team.

Rules:
- Use the "About Pulse" context above to answer general questions about what Pulse is, how it works, its roles, or its flow — answer these directly and confidently, don't escalate them.
- If the user is explicitly asking to speak with a human, a real person, support, or an admin, respond with EXACTLY the single word ESCALATE regardless of whether you could otherwise answer their question.
- If the message is casual conversation (a greeting, thanks, small talk, or a simple check-in), reply warmly and briefly as Pulse support.
- If the message closely matches one of the previously-answered questions below, answer it the same way in your own words — never mention that you're referencing past answers.
- Never invent specifics about a particular user's account, balance, campaign, submission, or live platform numbers (e.g. "how many campaigns are open right now") — you have no access to that data, so escalate those.
- For any other real question or issue you have no reliable information about, respond with EXACTLY the single word ESCALATE and nothing else.
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
