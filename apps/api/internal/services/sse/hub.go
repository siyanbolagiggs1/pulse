package sse

import "sync"

// Hub maintains the set of active SSE client connections, keyed by userID.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]chan string
}

// Global is the process-wide SSE hub. Initialised once at startup.
var Global = &Hub{
	clients: make(map[string]chan string),
}

// Register creates a buffered channel for the given user and returns it.
// Replaces any previous channel the user may have had (e.g. reconnect).
func (h *Hub) Register(userID string) chan string {
	ch := make(chan string, 32)
	h.mu.Lock()
	if old, ok := h.clients[userID]; ok {
		close(old)
	}
	h.clients[userID] = ch
	h.mu.Unlock()
	return ch
}

// Unregister removes the user's channel and closes it.
func (h *Hub) Unregister(userID string) {
	h.mu.Lock()
	if ch, ok := h.clients[userID]; ok {
		delete(h.clients, userID)
		close(ch)
	}
	h.mu.Unlock()
}

// Push sends a JSON payload to the user's SSE stream.
// If the user is not connected, or their channel is full, the event is dropped.
func (h *Hub) Push(userID, data string) {
	h.mu.RLock()
	ch, ok := h.clients[userID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case ch <- data:
	default:
		// client is slow or backlogged; drop rather than block
	}
}
