package websocket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Client represents a WebSocket client.
type Client struct {
	id     string
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	logger *slog.Logger
}

// ClientMessage represents a message from a client.
type ClientMessage struct {
	Action string `json:"action"` // subscribe, unsubscribe
	Topic  string `json:"topic"`
}

// NewClient creates a new client from an HTTP connection.
func NewClient(hub *Hub, conn *websocket.Conn, logger *slog.Logger) *Client {
	return &Client{
		id:     uuid.New().String()[:8],
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		logger: logger,
	}
}

// ServeWs handles websocket requests from clients.
func ServeWs(hub *Hub, logger *slog.Logger, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("failed to upgrade connection", slog.Any("error", err))
		return
	}

	client := NewClient(hub, conn, logger)
	hub.Register(client)

	// Start client goroutines
	go client.writePump()
	go client.readPump()

	// Send welcome message
	welcome := NewMessage("connected", "", map[string]string{
		"client_id": client.id,
		"message":   "Connected to EDA-Lab WebSocket",
	})
	data, _ := json.Marshal(welcome)
	client.send <- data
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Debug("websocket read error",
					slog.String("client_id", c.id),
					slog.Any("error", err),
				)
			}
			break
		}

		// Parse client message
		var msg ClientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.logger.Debug("invalid message format",
				slog.String("client_id", c.id),
				slog.String("message", string(message)),
			)
			continue
		}

		// Handle actions
		switch msg.Action {
		case "subscribe":
			if msg.Topic != "" {
				c.hub.Subscribe(c, msg.Topic)
				c.sendAck("subscribed", msg.Topic)
			}
		case "unsubscribe":
			if msg.Topic != "" {
				c.hub.Unsubscribe(c, msg.Topic)
				c.sendAck("unsubscribed", msg.Topic)
			}
		case "ping":
			c.sendAck("pong", "")
		default:
			c.logger.Debug("unknown action",
				slog.String("client_id", c.id),
				slog.String("action", msg.Action),
			)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
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

// sendAck sends an acknowledgment message to the client.
func (c *Client) sendAck(action, topic string) {
	ack := NewMessage(action, topic, nil)
	data, _ := json.Marshal(ack)
	select {
	case c.send <- data:
	default:
	}
}
