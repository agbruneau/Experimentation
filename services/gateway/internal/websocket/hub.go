// Package websocket provides WebSocket functionality for real-time event streaming.
package websocket

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

// Message represents a WebSocket message.
type Message struct {
	Type      string          `json:"type"`
	Topic     string          `json:"topic,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp string          `json:"timestamp"`
}

// NewMessage creates a new message.
func NewMessage(msgType, topic string, data interface{}) *Message {
	var rawData json.RawMessage
	if data != nil {
		rawData, _ = json.Marshal(data)
	}
	return &Message{
		Type:      msgType,
		Topic:     topic,
		Data:      rawData,
		Timestamp: time.Now().Format(time.RFC3339Nano),
	}
}

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Client subscriptions by topic
	subscriptions map[string]map[*Client]bool

	// Inbound messages from clients
	broadcast chan *Message

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Subscribe requests
	subscribe chan *subscribeRequest

	// Unsubscribe requests
	unsubscribe chan *subscribeRequest

	logger *slog.Logger
	mu     sync.RWMutex
}

type subscribeRequest struct {
	client *Client
	topic  string
}

// NewHub creates a new Hub.
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		subscriptions: make(map[string]map[*Client]bool),
		broadcast:     make(chan *Message, 256),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		subscribe:     make(chan *subscribeRequest, 64),
		unsubscribe:   make(chan *subscribeRequest, 64),
		logger:        logger,
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Debug("client registered",
				slog.String("client_id", client.id),
			)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				// Remove from all subscriptions
				for topic, clients := range h.subscriptions {
					delete(clients, client)
					if len(clients) == 0 {
						delete(h.subscriptions, topic)
					}
				}
			}
			h.mu.Unlock()
			h.logger.Debug("client unregistered",
				slog.String("client_id", client.id),
			)

		case req := <-h.subscribe:
			h.mu.Lock()
			if h.subscriptions[req.topic] == nil {
				h.subscriptions[req.topic] = make(map[*Client]bool)
			}
			h.subscriptions[req.topic][req.client] = true
			h.mu.Unlock()
			h.logger.Debug("client subscribed",
				slog.String("client_id", req.client.id),
				slog.String("topic", req.topic),
			)

		case req := <-h.unsubscribe:
			h.mu.Lock()
			if clients, ok := h.subscriptions[req.topic]; ok {
				delete(clients, req.client)
				if len(clients) == 0 {
					delete(h.subscriptions, req.topic)
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// broadcastMessage sends a message to appropriate clients.
func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		h.logger.Error("failed to marshal message", slog.Any("error", err))
		return
	}

	if message.Topic != "" {
		// Send to subscribed clients only
		if clients, ok := h.subscriptions[message.Topic]; ok {
			for client := range clients {
				select {
				case client.send <- data:
				default:
					// Client buffer full, skip
				}
			}
		}

		// Also send to clients subscribed to wildcard
		if clients, ok := h.subscriptions["*"]; ok {
			for client := range clients {
				select {
				case client.send <- data:
				default:
				}
			}
		}
	} else {
		// Broadcast to all clients
		for client := range h.clients {
			select {
			case client.send <- data:
			default:
			}
		}
	}
}

// Broadcast sends a message to all clients.
func (h *Hub) Broadcast(message *Message) {
	h.broadcast <- message
}

// BroadcastToTopic sends a message to clients subscribed to a topic.
func (h *Hub) BroadcastToTopic(topic string, data interface{}) {
	message := NewMessage("event", topic, data)
	h.broadcast <- message
}

// Register registers a client with the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Subscribe subscribes a client to a topic.
func (h *Hub) Subscribe(client *Client, topic string) {
	h.subscribe <- &subscribeRequest{client: client, topic: topic}
}

// Unsubscribe unsubscribes a client from a topic.
func (h *Hub) Unsubscribe(client *Client, topic string) {
	h.unsubscribe <- &subscribeRequest{client: client, topic: topic}
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// TopicSubscribers returns the number of subscribers for a topic.
func (h *Hub) TopicSubscribers(topic string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.subscriptions[topic]; ok {
		return len(clients)
	}
	return 0
}
