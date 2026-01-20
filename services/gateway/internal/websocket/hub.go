package websocket

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/edalab/pkg/observability"
	"github.com/gorilla/websocket"
)

// Message represents a WebSocket message
type Message struct {
	Type      string      `json:"type"`      // "event", "status", "error"
	Topic     string      `json:"topic,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// ClientMessage represents a message from client
type ClientMessage struct {
	Action string   `json:"action"` // "subscribe", "unsubscribe"
	Topics []string `json:"topics"`
}

// Client represents a WebSocket client connection
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	topics map[string]bool
	mu     sync.RWMutex
}

// Hub manages all WebSocket clients
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *slog.Logger
}

// NewHub creates a new WebSocket hub
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			observability.ActiveConnections.WithLabelValues("gateway", "websocket").Inc()
			h.logger.Info("Client connected", slog.Int("total_clients", len(h.clients)))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			observability.ActiveConnections.WithLabelValues("gateway", "websocket").Dec()
			h.logger.Info("Client disconnected", slog.Int("total_clients", len(h.clients)))

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// Register registers a new client
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message to all clients
func (h *Hub) Broadcast(msg *Message) {
	msg.Timestamp = time.Now()
	select {
	case h.broadcast <- msg:
	default:
		h.logger.Warn("Broadcast channel full, dropping message")
	}
}

// BroadcastToTopic sends a message to clients subscribed to a topic
func (h *Hub) BroadcastToTopic(topic string, msg *Message) {
	msg.Topic = topic
	msg.Timestamp = time.Now()
	select {
	case h.broadcast <- msg:
	default:
		h.logger.Warn("Broadcast channel full, dropping message")
	}
}

// broadcastMessage sends a message to appropriate clients
func (h *Hub) broadcastMessage(msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("Failed to marshal message", slog.String("error", err.Error()))
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		// Check if client is subscribed to this topic
		if msg.Topic != "" {
			client.mu.RLock()
			subscribed := client.topics[msg.Topic] || client.topics["*"]
			client.mu.RUnlock()
			if !subscribed {
				continue
			}
		}

		select {
		case client.send <- data:
		default:
			// Client buffer full, close connection
			go func(c *Client) {
				h.unregister <- c
			}(client)
		}
	}
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		topics: make(map[string]bool),
	}
}

// Subscribe subscribes the client to topics
func (c *Client) Subscribe(topics []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, topic := range topics {
		c.topics[topic] = true
	}
}

// Unsubscribe unsubscribes the client from topics
func (c *Client) Unsubscribe(topics []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, topic := range topics {
		delete(c.topics, topic)
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.Warn("WebSocket error", slog.String("error", err.Error()))
			}
			break
		}

		// Parse client message
		var clientMsg ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			continue
		}

		switch clientMsg.Action {
		case "subscribe":
			c.Subscribe(clientMsg.Topics)
			c.hub.logger.Debug("Client subscribed",
				slog.Any("topics", clientMsg.Topics),
			)
		case "unsubscribe":
			c.Unsubscribe(clientMsg.Topics)
			c.hub.logger.Debug("Client unsubscribed",
				slog.Any("topics", clientMsg.Topics),
			)
		}
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
