package kafka

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/hamba/avro/v2"
	"github.com/riferrei/srclient"
)

// Message represents a consumed Kafka message.
type Message struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       string
	Value     interface{}
	RawValue  []byte
	Headers   map[string]string
	Timestamp time.Time
	SchemaID  int
}

// MessageHandler is a function that processes a consumed message.
type MessageHandler func(ctx context.Context, msg *Message) error

// Consumer defines the interface for consuming messages from Kafka.
type Consumer interface {
	// Subscribe subscribes to the given topics.
	Subscribe(topics []string) error

	// Consume starts consuming messages and calls the handler for each.
	Consume(ctx context.Context, handler MessageHandler) error

	// ConsumeOnce consumes a single message with timeout.
	ConsumeOnce(ctx context.Context, timeout time.Duration) (*Message, error)

	// Commit commits the current offsets.
	Commit() error

	// Close closes the consumer.
	Close() error
}

// ConsumerConfig holds configuration for the Kafka consumer.
type ConsumerConfig struct {
	BootstrapServers  string
	SchemaRegistryURL string
	GroupID           string
	AutoOffsetReset   string
	EnableAutoCommit  bool
	SessionTimeoutMs  int
	HeartbeatMs       int
	MaxPollInterval   int
}

// DefaultConsumerConfig returns a consumer config with sensible defaults.
func DefaultConsumerConfig(bootstrapServers, schemaRegistryURL, groupID string) ConsumerConfig {
	return ConsumerConfig{
		BootstrapServers:  bootstrapServers,
		SchemaRegistryURL: schemaRegistryURL,
		GroupID:           groupID,
		AutoOffsetReset:   "earliest",
		EnableAutoCommit:  true,
		SessionTimeoutMs:  30000,
		HeartbeatMs:       10000,
		MaxPollInterval:   300000,
	}
}

// AvroConsumer implements Consumer with Avro deserialization.
type AvroConsumer struct {
	consumer     *kafka.Consumer
	schemaClient *srclient.SchemaRegistryClient
	avroCache    map[int]avro.Schema
	mu           sync.RWMutex
	running      bool
}

// NewAvroConsumer creates a new Avro-enabled Kafka consumer.
func NewAvroConsumer(config ConsumerConfig) (*AvroConsumer, error) {
	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers":        config.BootstrapServers,
		"group.id":                 config.GroupID,
		"auto.offset.reset":        config.AutoOffsetReset,
		"enable.auto.commit":       config.EnableAutoCommit,
		"session.timeout.ms":       config.SessionTimeoutMs,
		"heartbeat.interval.ms":    config.HeartbeatMs,
		"max.poll.interval.ms":     config.MaxPollInterval,
		"enable.partition.eof":     false,
		"go.events.channel.enable": false,
	}

	consumer, err := kafka.NewConsumer(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	schemaClient := srclient.CreateSchemaRegistryClient(config.SchemaRegistryURL)

	return &AvroConsumer{
		consumer:     consumer,
		schemaClient: schemaClient,
		avroCache:    make(map[int]avro.Schema),
	}, nil
}

// Subscribe subscribes to the given topics.
func (c *AvroConsumer) Subscribe(topics []string) error {
	return c.consumer.SubscribeTopics(topics, nil)
}

// Consume starts consuming messages and calls the handler for each.
// This method blocks until the context is cancelled or an unrecoverable error occurs.
func (c *AvroConsumer) Consume(ctx context.Context, handler MessageHandler) error {
	c.running = true
	defer func() { c.running = false }()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.consumer.ReadMessage(time.Second)
			if err != nil {
				// Timeout is not an error, just continue
				if kafkaErr, ok := err.(kafka.Error); ok {
					if kafkaErr.Code() == kafka.ErrTimedOut {
						continue
					}
					// Handle other Kafka errors
					if kafkaErr.IsFatal() {
						return fmt.Errorf("fatal Kafka error: %w", err)
					}
				}
				continue
			}

			// Deserialize message
			message, err := c.deserializeMessage(msg)
			if err != nil {
				// Log error but continue consuming
				fmt.Printf("Failed to deserialize message: %v\n", err)
				continue
			}

			// Call handler
			if err := handler(ctx, message); err != nil {
				// Log error but continue consuming
				fmt.Printf("Handler error: %v\n", err)
			}
		}
	}
}

// ConsumeOnce consumes a single message with timeout.
func (c *AvroConsumer) ConsumeOnce(ctx context.Context, timeout time.Duration) (*Message, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return nil, fmt.Errorf("timeout waiting for message")
			}

			pollTimeout := remaining
			if pollTimeout > time.Second {
				pollTimeout = time.Second
			}

			msg, err := c.consumer.ReadMessage(pollTimeout)
			if err != nil {
				if kafkaErr, ok := err.(kafka.Error); ok {
					if kafkaErr.Code() == kafka.ErrTimedOut {
						continue
					}
				}
				return nil, err
			}

			return c.deserializeMessage(msg)
		}
	}

	return nil, fmt.Errorf("timeout waiting for message")
}

// Commit commits the current offsets.
func (c *AvroConsumer) Commit() error {
	_, err := c.consumer.Commit()
	return err
}

// Close closes the consumer.
func (c *AvroConsumer) Close() error {
	return c.consumer.Close()
}

// deserializeMessage converts a Kafka message to our Message type.
func (c *AvroConsumer) deserializeMessage(msg *kafka.Message) (*Message, error) {
	message := &Message{
		Topic:     *msg.TopicPartition.Topic,
		Partition: msg.TopicPartition.Partition,
		Offset:    int64(msg.TopicPartition.Offset),
		Key:       string(msg.Key),
		RawValue:  msg.Value,
		Headers:   make(map[string]string),
		Timestamp: msg.Timestamp,
	}

	// Parse headers
	for _, header := range msg.Headers {
		message.Headers[header.Key] = string(header.Value)
	}

	// Try to deserialize as Avro
	if len(msg.Value) > 5 && msg.Value[0] == 0 {
		// Has Confluent wire format
		schemaID := int(binary.BigEndian.Uint32(msg.Value[1:5]))
		message.SchemaID = schemaID

		schema, err := c.getAvroSchema(schemaID)
		if err != nil {
			// Could not get schema, return raw value
			message.Value = msg.Value
			return message, nil
		}

		// Deserialize Avro payload
		avroPayload := msg.Value[5:]
		var value interface{}
		if err := avro.Unmarshal(schema, avroPayload, &value); err != nil {
			// Deserialization failed, return raw value
			message.Value = msg.Value
			return message, nil
		}

		message.Value = value
	} else {
		// Not Avro format, return raw value
		message.Value = msg.Value
	}

	return message, nil
}

// DeserializeInto deserializes the message value into the provided struct.
func (c *AvroConsumer) DeserializeInto(msg *Message, target interface{}) error {
	if msg.SchemaID == 0 {
		return fmt.Errorf("message does not have a schema ID")
	}

	schema, err := c.getAvroSchema(msg.SchemaID)
	if err != nil {
		return fmt.Errorf("failed to get schema: %w", err)
	}

	// Get Avro payload (skip wire format header)
	if len(msg.RawValue) < 5 {
		return fmt.Errorf("invalid message format")
	}
	avroPayload := msg.RawValue[5:]

	return avro.Unmarshal(schema, avroPayload, target)
}

// getAvroSchema retrieves and caches an Avro schema by ID.
func (c *AvroConsumer) getAvroSchema(schemaID int) (avro.Schema, error) {
	c.mu.RLock()
	if schema, ok := c.avroCache[schemaID]; ok {
		c.mu.RUnlock()
		return schema, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check
	if schema, ok := c.avroCache[schemaID]; ok {
		return schema, nil
	}

	// Fetch from Schema Registry
	srcSchema, err := c.schemaClient.GetSchema(schemaID)
	if err != nil {
		return nil, err
	}

	avroSchema, err := avro.Parse(srcSchema.Schema())
	if err != nil {
		return nil, err
	}

	c.avroCache[schemaID] = avroSchema
	return avroSchema, nil
}

// SimpleConsumer is a consumer without Avro deserialization.
type SimpleConsumer struct {
	consumer *kafka.Consumer
}

// NewSimpleConsumer creates a simple Kafka consumer without Schema Registry.
func NewSimpleConsumer(bootstrapServers, groupID string) (*SimpleConsumer, error) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
		"group.id":          groupID,
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		return nil, err
	}

	return &SimpleConsumer{consumer: consumer}, nil
}

// Subscribe subscribes to topics.
func (c *SimpleConsumer) Subscribe(topics []string) error {
	return c.consumer.SubscribeTopics(topics, nil)
}

// Consume starts consuming messages.
func (c *SimpleConsumer) Consume(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.consumer.ReadMessage(time.Second)
			if err != nil {
				if kafkaErr, ok := err.(kafka.Error); ok && kafkaErr.Code() == kafka.ErrTimedOut {
					continue
				}
				continue
			}

			message := &Message{
				Topic:     *msg.TopicPartition.Topic,
				Partition: msg.TopicPartition.Partition,
				Offset:    int64(msg.TopicPartition.Offset),
				Key:       string(msg.Key),
				Value:     msg.Value,
				RawValue:  msg.Value,
				Headers:   make(map[string]string),
				Timestamp: msg.Timestamp,
			}

			for _, h := range msg.Headers {
				message.Headers[h.Key] = string(h.Value)
			}

			if err := handler(ctx, message); err != nil {
				fmt.Printf("Handler error: %v\n", err)
			}
		}
	}
}

// ConsumeOnce consumes a single message.
func (c *SimpleConsumer) ConsumeOnce(ctx context.Context, timeout time.Duration) (*Message, error) {
	msg, err := c.consumer.ReadMessage(timeout)
	if err != nil {
		return nil, err
	}

	message := &Message{
		Topic:     *msg.TopicPartition.Topic,
		Partition: msg.TopicPartition.Partition,
		Offset:    int64(msg.TopicPartition.Offset),
		Key:       string(msg.Key),
		Value:     msg.Value,
		RawValue:  msg.Value,
		Headers:   make(map[string]string),
		Timestamp: msg.Timestamp,
	}

	for _, h := range msg.Headers {
		message.Headers[h.Key] = string(h.Value)
	}

	return message, nil
}

// Commit commits offsets.
func (c *SimpleConsumer) Commit() error {
	_, err := c.consumer.Commit()
	return err
}

// Close closes the consumer.
func (c *SimpleConsumer) Close() error {
	return c.consumer.Close()
}
