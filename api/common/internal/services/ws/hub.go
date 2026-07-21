// Package ws provides the real-time push transport for Pulse, carrying both
// account notifications and chat messages over a single WebSocket connection
// per client. It replaces the old internal/services/sse package, which only
// supported one connection per user (a second browser tab silently killed
// the first tab's stream) and no bidirectional traffic.
package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Envelope is the single JSON shape carried over the socket for every event
// type: "connected", "notification", "chat_message", "typing", "read_receipt".
type Envelope struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

const (
	sendBufferSize = 32
	pingInterval   = 25 * time.Second
	pongWait       = 60 * time.Second
	writeWait      = 10 * time.Second
)

// Client wraps one live WebSocket connection for one user. A user may have
// several Clients at once (multiple tabs/devices) — Push fans out to all of them.
type Client struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
}

// Hub tracks every live connection, grouped by userID so Push can fan out to
// every tab/device a user currently has open.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}
}

var Global = &Hub{clients: make(map[string]map[*Client]struct{})}

// Register adds a connection to the hub. Unlike the old SSE hub, this never
// closes an existing connection for the same user — multiple tabs/devices
// coexist independently.
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[c.userID] == nil {
		h.clients[c.userID] = make(map[*Client]struct{})
	}
	h.clients[c.userID][c] = struct{}{}
}

// Unregister removes a connection. Safe to call more than once for the same
// client (e.g. from both the read pump and write pump exit paths).
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.clients[c.userID]; ok {
		if _, present := conns[c]; present {
			delete(conns, c)
			close(c.send)
		}
		if len(conns) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

// Push sends an envelope to every live connection for the given user. Delivery
// is best-effort and non-blocking — if a connection's send buffer is full
// (slow/stuck client), the event is silently dropped for that connection only.
// If the user isn't connected at all, this is a no-op (the caller is expected
// to have already persisted whatever the push represents, e.g. a notification
// or chat message, so nothing is lost — just not delivered live).
func (h *Hub) Push(userID string, envelope Envelope) {
	data, err := json.Marshal(envelope)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients[userID] {
		select {
		case c.send <- data:
		default:
		}
	}
}
