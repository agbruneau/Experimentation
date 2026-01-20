package kafka

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/hamba/avro/v2"
	"github.com/riferrei/srclient"
)

// Message represents a consumed Kafka message
type Message struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       string
	Value     interface{}
	Headers   map[string]string
	Timestamp time.Time
	SchemaID  int
}

// MessageHandler processes consumed messages
type MessageHandler func(ctx context.Context, msg *Message) error

// Consumer defines the interface for Kafka message consumption
type Consumer interface {
	Subscribe(topics []string) error
	Consume(ctx context.Context, handler MessageHandler) error
	Close() error
}

// ConsumerConfig holds configuration for the Kafka consumer
type ConsumerConfig struct {
	BootstrapServers  string
	SchemaRegistryURL string
	GroupID           string
	AutoOffsetReset   string
	EnableAutoCommit  bool
	SessionTimeoutMs  int
}

// AvroConsumer implements Consumer with Avro deserialization
type AvroConsumer struct {
	consumer     *kafka.Consumer
	schemaClient *srclient.SchemaRegistryClient
	config       ConsumerConfig
}

// NewAvroConsumer creates a new Avro-enabled Kafka consumer
func NewAvroConsumer(config ConsumerConfig) (*AvroConsumer, error) {
	if config.GroupID == "" {
		return nil, fmt.Errorf("group_id is required")
	}

	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers":  config.BootstrapServers,
		"group.id":           config.GroupID,
		"auto.offset.reset":  config.AutoOffsetReset,
		"enable.auto.commit": config.EnableAutoCommit,
		"session.timeout.ms": config.SessionTimeoutMs,
	}

	// Set defaults
	if config.AutoOffsetReset == "" {
		kafkaConfig.SetKey("auto.offset.reset", "earliest")
	}
	if config.SessionTimeoutMs == 0 {
		kafkaConfig.SetKey("session.timeout.ms", 6000)
	}

	consumer, err := kafka.NewConsumer(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	schemaClient := srclient.CreateSchemaRegistryClient(config.SchemaRegistryURL)

	return &AvroConsumer{
		consumer:     consumer,
		schemaClient: schemaClient,
		config:       config,
	}, nil
}

// Subscribe subscribes to the specified topics
func (c *AvroConsumer) Subscribe(topics []string) error {
	if err := c.consumer.SubscribeTopics(topics, nil); err != nil {
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}
	return nil
}

// Consume starts consuming messages and calls the handler for each message
func (c *AvroConsumer) Consume(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				// Timeout is expected, continue polling
				if kafkaErr, ok := err.(kafka.Error); ok && kafkaErr.Code() == kafka.ErrTimedOut {
					continue
				}
				return fmt.Errorf("error reading message: %w", err)
			}

			// Parse message
			parsedMsg, err := c.parseMessage(msg)
			if err != nil {
				// Log error but continue processing
				fmt.Printf("Error parsing message: %v\n", err)
				continue
			}

			// Call handler
			if err := handler(ctx, parsedMsg); err != nil {
				// Log error but continue processing
				fmt.Printf("Error handling message: %v\n", err)
				continue
			}

			// Commit offset if auto-commit is disabled
			if !c.config.EnableAutoCommit {
				if _, err := c.consumer.CommitMessage(msg); err != nil {
					fmt.Printf("Error committing message: %v\n", err)
				}
			}
		}
	}
}

// parseMessage parses a Kafka message and deserializes the Avro value
func (c *AvroConsumer) parseMessage(msg *kafka.Message) (*Message, error) {
	// Extract headers
	headers := make(map[string]string)
	for _, h := range msg.Headers {
		headers[h.Key] = string(h.Value)
	}

	// Parse Confluent wire format
	if len(msg.Value) < 5 {
		return nil, fmt.Errorf("message value too short")
	}

	// Check magic byte
	if msg.Value[0] != 0 {
		return nil, fmt.Errorf("invalid magic byte: %d", msg.Value[0])
	}

	// Extract schema ID
	schemaID := int(binary.BigEndian.Uint32(msg.Value[1:5]))

	// Get schema from registry
	schema, err := c.schemaClient.GetSchema(schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema %d: %w", schemaID, err)
	}

	// Parse Avro schema
	avroSchema, err := avro.Parse(schema.Schema())
	if err != nil {
		return nil, fmt.Errorf("failed to parse Avro schema: %w", err)
	}

	// Deserialize value (skip wire format header)
	var value interface{}
	if err := avro.Unmarshal(avroSchema, msg.Value[5:], &value); err != nil {
		return nil, fmt.Errorf("failed to deserialize value: %w", err)
	}

	return &Message{
		Topic:     *msg.TopicPartition.Topic,
		Partition: msg.TopicPartition.Partition,
		Offset:    int64(msg.TopicPartition.Offset),
		Key:       string(msg.Key),
		Value:     value,
		Headers:   headers,
		Timestamp: msg.Timestamp,
		SchemaID:  schemaID,
	}, nil
}

// ConsumeRaw consumes raw messages without Avro deserialization
func (c *AvroConsumer) ConsumeRaw(ctx context.Context, handler func(ctx context.Context, msg *kafka.Message) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				if kafkaErr, ok := err.(kafka.Error); ok && kafkaErr.Code() == kafka.ErrTimedOut {
					continue
				}
				return fmt.Errorf("error reading message: %w", err)
			}

			if err := handler(ctx, msg); err != nil {
				fmt.Printf("Error handling message: %v\n", err)
				continue
			}

			if !c.config.EnableAutoCommit {
				if _, err := c.consumer.CommitMessage(msg); err != nil {
					fmt.Printf("Error committing message: %v\n", err)
				}
			}
		}
	}
}

// Close closes the consumer
func (c *AvroConsumer) Close() error {
	return c.consumer.Close()
}
