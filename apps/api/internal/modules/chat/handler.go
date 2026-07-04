package chat

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
	"github.com/pulse/api/internal/services/ws"
	"github.com/pulse/api/internal/utils"
)

func errStatus(err error) int {
	switch {
	case errors.Is(err, ErrUserNotFound), errors.Is(err, ErrConversationNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrInvalidRecipient), errors.Is(err, ErrUserSuspended):
		return http.StatusBadRequest
	case errors.Is(err, ErrNotParticipant):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

// POST /api/conversations
func handleStartConversation(c *gin.Context) {
	var req StartConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	conv, err := startOrGetConversation(c.Request.Context(),
		middleware.GetUserID(c), middleware.GetUserRole(c), req.RecipientID)
	if err != nil {
		utils.Fail(c, errStatus(err), err.Error())
		return
	}
	utils.OK(c, http.StatusCreated, "Conversation ready", conv)
}

// GET /api/conversations
func handleListConversations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	convs, total, err := listConversations(c.Request.Context(), middleware.GetUserID(c), middleware.GetUserRole(c), page, limit)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch conversations")
		return
	}

	limit = clampLimit(limit, 50, 20)
	utils.OKWithMeta(c, http.StatusOK, "", convs, ListMeta{
		Total: total, Page: clampPage(page), Limit: limit, Pages: pages(total, limit),
	})
}

// GET /api/conversations/:id
func handleGetConversation(c *gin.Context) {
	conv, err := getConversation(c.Request.Context(), c.Param("id"), middleware.GetUserID(c), middleware.GetUserRole(c))
	if err != nil {
		utils.Fail(c, errStatus(err), err.Error())
		return
	}
	utils.OK(c, http.StatusOK, "", conv)
}

// GET /api/conversations/:id/messages
func handleGetMessages(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	msgs, total, err := getMessages(c.Request.Context(), c.Param("id"), middleware.GetUserID(c), true, page, limit)
	if err != nil {
		utils.Fail(c, errStatus(err), err.Error())
		return
	}

	limit = clampLimit(limit, 100, 50)
	utils.OKWithMeta(c, http.StatusOK, "", msgs, ListMeta{
		Total: total, Page: clampPage(page), Limit: limit, Pages: pages(total, limit),
	})
}

// POST /api/conversations/:id/messages
func handleSendMessage(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	msg, otherPartyID, err := sendMessage(c.Request.Context(), c.Param("id"), middleware.GetUserID(c), req.Body)
	if err != nil {
		utils.Fail(c, errStatus(err), err.Error())
		return
	}

	go ws.Global.Push(otherPartyID, ws.Envelope{Type: "chat_message", Data: msg})

	utils.OK(c, http.StatusCreated, "Message sent", msg)
}

// POST /api/conversations/:id/read
func handleMarkRead(c *gin.Context) {
	userID := middleware.GetUserID(c)
	conversationID := c.Param("id")

	otherPartyID, err := markRead(c.Request.Context(), conversationID, userID)
	if err != nil {
		utils.Fail(c, errStatus(err), err.Error())
		return
	}

	go ws.Global.Push(otherPartyID, ws.Envelope{Type: "read_receipt", Data: gin.H{
		"conversationId": conversationID,
		"userId":         userID,
		"readAt":         time.Now().UTC(),
	}})

	utils.OK(c, http.StatusOK, "Marked as read", nil)
}

// POST /api/conversations/:id/typing
func handleTyping(c *gin.Context) {
	userID := middleware.GetUserID(c)
	conversationID := c.Param("id")

	otherPartyID, err := verifyParticipant(c.Request.Context(), conversationID, userID)
	if err != nil {
		utils.Fail(c, errStatus(err), err.Error())
		return
	}

	go ws.Global.Push(otherPartyID, ws.Envelope{Type: "typing", Data: gin.H{
		"conversationId": conversationID,
		"userId":         userID,
	}})

	utils.OK(c, http.StatusOK, "", nil)
}

// GET /api/admin/conversations
func handleAdminListConversations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	convs, total, err := listAllConversations(c.Request.Context(), page, limit)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch conversations")
		return
	}

	limit = clampLimit(limit, 50, 20)
	utils.OKWithMeta(c, http.StatusOK, "", convs, ListMeta{
		Total: total, Page: clampPage(page), Limit: limit, Pages: pages(total, limit),
	})
}

// GET /api/admin/conversations/:id/messages
func handleAdminGetMessages(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	msgs, total, err := getMessages(c.Request.Context(), c.Param("id"), "", false, page, limit)
	if err != nil {
		utils.Fail(c, errStatus(err), err.Error())
		return
	}

	limit = clampLimit(limit, 100, 50)
	utils.OKWithMeta(c, http.StatusOK, "", msgs, ListMeta{
		Total: total, Page: clampPage(page), Limit: limit, Pages: pages(total, limit),
	})
}

func clampPage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func clampLimit(limit, max, def int) int {
	if limit < 1 || limit > max {
		return def
	}
	return limit
}
