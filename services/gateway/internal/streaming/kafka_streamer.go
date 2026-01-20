package streaming

import (
	"context"
	"log/slog"

	"github.com/edalab/pkg/kafka"
	"github.com/edalab/pkg/observability"
	"github.com/edalab/services/gateway/internal/websocket"
)

// KafkaStreamer streams Kafka events to WebSocket clients
type KafkaStreamer struct {
	consumer *kafka.AvroConsumer
	hub      *websocket.Hub
	topics   []string
	logger   *slog.Logger
}

// NewKafkaStreamer creates a new Kafka streamer
func NewKafkaStreamer(consumer *kafka.AvroConsumer, hub *websocket.Hub, topics []string, logger *slog.Logger) *KafkaStreamer {
	return &KafkaStreamer{
		consumer: consumer,
		hub:      hub,
		topics:   topics,
		logger:   logger,
	}
}

// Start starts consuming and streaming events
func (s *KafkaStreamer) Start(ctx context.Context) error {
	if err := s.consumer.Subscribe(s.topics); err != nil {
		return err
	}

	s.logger.Info("Started Kafka streamer",
		slog.Any("topics", s.topics),
	)

	return s.consumer.Consume(ctx, s.handleMessage)
}

// handleMessage handles incoming Kafka messages
func (s *KafkaStreamer) handleMessage(ctx context.Context, msg *kafka.Message) error {
	s.logger.Debug("Received Kafka message",
		slog.String("topic", msg.Topic),
		slog.String("key", msg.Key),
	)

	// Create WebSocket message
	wsMsg := &websocket.Message{
		Type:    "event",
		Topic:   msg.Topic,
		Payload: msg.Value,
	}

	// Broadcast to WebSocket clients
	s.hub.BroadcastToTopic(msg.Topic, wsMsg)

	// Update metrics
	observability.MessagesConsumed.WithLabelValues("gateway", msg.Topic).Inc()

	return nil
}

// Stop stops the streamer
func (s *KafkaStreamer) Stop() {
	s.consumer.Close()
	s.logger.Info("Stopped Kafka streamer")
}
