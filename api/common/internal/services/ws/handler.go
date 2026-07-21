package ws

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pulse/api/internal/config"
	"github.com/pulse/api/internal/middleware"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// Non-browser clients (native apps, wscat, server-to-server) don't
			// send an Origin header at all — nothing to check against.
			return true
		}
		return config.IsAllowedOrigin(origin)
	},
}

// HandleUpgrade upgrades an authenticated HTTP request to a WebSocket
// connection and registers it with the global Hub. Mounted at GET /api/ws
// behind middleware.RequireAuthWS() (not the header-only RequireAuth(), since
// a browser WebSocket handshake can't set custom headers).
func HandleUpgrade(c *gin.Context) {
	userID := middleware.GetUserID(c)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}

	client := &Client{
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, sendBufferSize),
	}
	Global.Register(client)

	go client.writePump()
	go client.readPump()

	Global.Push(userID, Envelope{Type: "connected", Data: gin.H{}})
}

// readPump drains incoming frames (control frames + any client-sent messages)
// and detects disconnects. Chat/notification traffic is server→client only
// today, so this loop mainly exists to keep the read deadline honored and to
// notice when the client goes away.
func (c *Client) readPump() {
	defer func() {
		Global.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

// writePump drains the per-client send channel and writes frames, and sends
// a native WS ping every ~25s — this replaces the old SSE hub's synthetic
// heartbeat comment frame; browsers reply to WS pings with pongs automatically
// at the protocol level, no client-side JS needed.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case data, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
