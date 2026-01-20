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

// Producer defines the interface for Kafka message production
type Producer interface {
	Produce(ctx context.Context, topic string, key string, value interface{}) error
	ProduceWithHeaders(ctx context.Context, topic string, key string, value interface{}, headers map[string]string) error
	Close() error
}

// ProducerConfig holds configuration for the Kafka producer
type ProducerConfig struct {
	BootstrapServers string
	SchemaRegistryURL string
	Acks             string
	Retries          int
	RetryBackoffMs   int
}

// AvroProducer implements Producer with Avro serialization
type AvroProducer struct {
	producer     *kafka.Producer
	schemaClient *srclient.SchemaRegistryClient
	schemas      map[string]*srclient.Schema
	schemasMu    sync.RWMutex
	config       ProducerConfig
}

// NewAvroProducer creates a new Avro-enabled Kafka producer
func NewAvroProducer(config ProducerConfig) (*AvroProducer, error) {
	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers": config.BootstrapServers,
		"acks":              config.Acks,
		"retries":           config.Retries,
		"retry.backoff.ms":  config.RetryBackoffMs,
	}

	// Set defaults
	if config.Acks == "" {
		kafkaConfig.SetKey("acks", "all")
	}
	if config.Retries == 0 {
		kafkaConfig.SetKey("retries", 3)
	}
	if config.RetryBackoffMs == 0 {
		kafkaConfig.SetKey("retry.backoff.ms", 100)
	}

	producer, err := kafka.NewProducer(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	schemaClient := srclient.CreateSchemaRegistryClient(config.SchemaRegistryURL)

	return &AvroProducer{
		producer:     producer,
		schemaClient: schemaClient,
		schemas:      make(map[string]*srclient.Schema),
		config:       config,
	}, nil
}

// Produce sends a message to the specified topic
func (p *AvroProducer) Produce(ctx context.Context, topic string, key string, value interface{}) error {
	return p.ProduceWithHeaders(ctx, topic, key, value, nil)
}

// ProduceWithHeaders sends a message with headers to the specified topic
func (p *AvroProducer) ProduceWithHeaders(ctx context.Context, topic string, key string, value interface{}, headers map[string]string) error {
	// Get or fetch schema
	schema, err := p.getSchema(topic)
	if err != nil {
		return fmt.Errorf("failed to get schema for topic %s: %w", topic, err)
	}

	// Parse Avro schema
	avroSchema, err := avro.Parse(schema.Schema())
	if err != nil {
		return fmt.Errorf("failed to parse Avro schema: %w", err)
	}

	// Serialize value to Avro
	valueBytes, err := avro.Marshal(avroSchema, value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// Prepend schema ID (Confluent wire format)
	// Format: [magic byte (0)] [4-byte schema ID] [Avro data]
	schemaIDBytes := make([]byte, 5)
	schemaIDBytes[0] = 0 // Magic byte
	binary.BigEndian.PutUint32(schemaIDBytes[1:5], uint32(schema.ID()))

	payload := append(schemaIDBytes, valueBytes...)

	// Build Kafka headers
	var kafkaHeaders []kafka.Header
	for k, v := range headers {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}

	// Create message
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:     []byte(key),
		Value:   payload,
		Headers: kafkaHeaders,
	}

	// Produce with delivery report
	deliveryChan := make(chan kafka.Event, 1)
	if err := p.producer.Produce(msg, deliveryChan); err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	// Wait for delivery or timeout
	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-deliveryChan:
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			return fmt.Errorf("delivery failed: %w", m.TopicPartition.Error)
		}
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("produce timeout")
	}
}

// getSchema retrieves the schema from cache or fetches from Schema Registry
func (p *AvroProducer) getSchema(topic string) (*srclient.Schema, error) {
	subject := topic + "-value"

	// Check cache
	p.schemasMu.RLock()
	if schema, ok := p.schemas[subject]; ok {
		p.schemasMu.RUnlock()
		return schema, nil
	}
	p.schemasMu.RUnlock()

	// Fetch from Schema Registry
	p.schemasMu.Lock()
	defer p.schemasMu.Unlock()

	// Double-check after acquiring write lock
	if schema, ok := p.schemas[subject]; ok {
		return schema, nil
	}

	schema, err := p.schemaClient.GetLatestSchema(subject)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema for subject %s: %w", subject, err)
	}

	p.schemas[subject] = schema
	return schema, nil
}

// Close closes the producer
func (p *AvroProducer) Close() error {
	p.producer.Flush(5000)
	p.producer.Close()
	return nil
}

// Flush waits for all messages to be delivered
func (p *AvroProducer) Flush(timeoutMs int) int {
	return p.producer.Flush(timeoutMs)
}
