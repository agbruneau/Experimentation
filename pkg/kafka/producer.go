// Package kafka provides Kafka producer and consumer with Avro serialization support.
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

// Producer defines the interface for producing messages to Kafka.
type Producer interface {
	// Produce sends a message to the specified topic.
	Produce(ctx context.Context, topic string, key string, value interface{}) error

	// ProduceWithHeaders sends a message with custom headers.
	ProduceWithHeaders(ctx context.Context, topic string, key string, value interface{}, headers map[string]string) error

	// ProduceRaw sends a raw byte message without Avro serialization.
	ProduceRaw(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error

	// Flush waits for all messages to be delivered.
	Flush(timeoutMs int) int

	// Close closes the producer.
	Close()
}

// ProducerConfig holds configuration for the Kafka producer.
type ProducerConfig struct {
	BootstrapServers  string
	SchemaRegistryURL string
	Acks              string
	EnableIdempotence bool
	MaxInFlight       int
	LingerMs          int
	BatchSize         int
	CompressionType   string
}

// DefaultProducerConfig returns a producer config with sensible defaults.
func DefaultProducerConfig(bootstrapServers, schemaRegistryURL string) ProducerConfig {
	return ProducerConfig{
		BootstrapServers:  bootstrapServers,
		SchemaRegistryURL: schemaRegistryURL,
		Acks:              "all",
		EnableIdempotence: true,
		MaxInFlight:       5,
		LingerMs:          5,
		BatchSize:         16384,
		CompressionType:   "snappy",
	}
}

// AvroProducer implements Producer with Avro serialization.
type AvroProducer struct {
	producer     *kafka.Producer
	schemaClient *srclient.SchemaRegistryClient
	schemaCache  map[string]*srclient.Schema
	avroCache    map[int]avro.Schema
	mu           sync.RWMutex
}

// NewAvroProducer creates a new Avro-enabled Kafka producer.
func NewAvroProducer(config ProducerConfig) (*AvroProducer, error) {
	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers":   config.BootstrapServers,
		"acks":                config.Acks,
		"enable.idempotence":  config.EnableIdempotence,
		"max.in.flight":       config.MaxInFlight,
		"linger.ms":           config.LingerMs,
		"batch.size":          config.BatchSize,
		"compression.type":    config.CompressionType,
		"retries":             3,
		"retry.backoff.ms":    100,
		"request.timeout.ms":  30000,
		"delivery.timeout.ms": 120000,
	}

	producer, err := kafka.NewProducer(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	schemaClient := srclient.CreateSchemaRegistryClient(config.SchemaRegistryURL)

	return &AvroProducer{
		producer:     producer,
		schemaClient: schemaClient,
		schemaCache:  make(map[string]*srclient.Schema),
		avroCache:    make(map[int]avro.Schema),
	}, nil
}

// Produce sends a message to the specified topic with Avro serialization.
func (p *AvroProducer) Produce(ctx context.Context, topic string, key string, value interface{}) error {
	return p.ProduceWithHeaders(ctx, topic, key, value, nil)
}

// ProduceWithHeaders sends a message with custom headers and Avro serialization.
func (p *AvroProducer) ProduceWithHeaders(ctx context.Context, topic string, key string, value interface{}, headers map[string]string) error {
	// Get or fetch schema for the topic
	subject := topic + "-value"
	schema, err := p.getSchema(subject)
	if err != nil {
		return fmt.Errorf("failed to get schema for %s: %w", subject, err)
	}

	// Get Avro schema
	avroSchema, err := p.getAvroSchema(schema)
	if err != nil {
		return fmt.Errorf("failed to parse Avro schema: %w", err)
	}

	// Serialize value with Avro
	valueBytes, err := p.serializeAvro(schema.ID(), avroSchema, value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	return p.ProduceRaw(ctx, topic, []byte(key), valueBytes, headers)
}

// ProduceRaw sends a raw byte message without Avro serialization.
func (p *AvroProducer) ProduceRaw(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error {
	// Build Kafka headers
	var kafkaHeaders []kafka.Header
	for k, v := range headers {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}

	// Create delivery channel
	deliveryChan := make(chan kafka.Event, 1)

	// Produce message
	err := p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:     key,
		Value:   value,
		Headers: kafkaHeaders,
	}, deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	// Wait for delivery with context
	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-deliveryChan:
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			return fmt.Errorf("delivery failed: %w", m.TopicPartition.Error)
		}
		return nil
	}
}

// Flush waits for all messages to be delivered.
func (p *AvroProducer) Flush(timeoutMs int) int {
	return p.producer.Flush(timeoutMs)
}

// Close closes the producer and releases resources.
func (p *AvroProducer) Close() {
	p.producer.Flush(10000)
	p.producer.Close()
}

// getSchema retrieves a schema from cache or Schema Registry.
func (p *AvroProducer) getSchema(subject string) (*srclient.Schema, error) {
	p.mu.RLock()
	if schema, ok := p.schemaCache[subject]; ok {
		p.mu.RUnlock()
		return schema, nil
	}
	p.mu.RUnlock()

	// Fetch from Schema Registry
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if schema, ok := p.schemaCache[subject]; ok {
		return schema, nil
	}

	schema, err := p.schemaClient.GetLatestSchema(subject)
	if err != nil {
		return nil, err
	}

	p.schemaCache[subject] = schema
	return schema, nil
}

// getAvroSchema parses and caches the Avro schema.
func (p *AvroProducer) getAvroSchema(schema *srclient.Schema) (avro.Schema, error) {
	p.mu.RLock()
	if avroSchema, ok := p.avroCache[schema.ID()]; ok {
		p.mu.RUnlock()
		return avroSchema, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check
	if avroSchema, ok := p.avroCache[schema.ID()]; ok {
		return avroSchema, nil
	}

	avroSchema, err := avro.Parse(schema.Schema())
	if err != nil {
		return nil, err
	}

	p.avroCache[schema.ID()] = avroSchema
	return avroSchema, nil
}

// serializeAvro serializes a value using Avro with the Confluent wire format.
// Wire format: [0] magic byte + [1-4] schema ID (big endian) + [5+] avro payload
func (p *AvroProducer) serializeAvro(schemaID int, schema avro.Schema, value interface{}) ([]byte, error) {
	// Serialize value to Avro
	avroBytes, err := avro.Marshal(schema, value)
	if err != nil {
		return nil, err
	}

	// Build wire format message
	// Magic byte (0) + schema ID (4 bytes big endian) + payload
	result := make([]byte, 5+len(avroBytes))
	result[0] = 0 // Magic byte
	binary.BigEndian.PutUint32(result[1:5], uint32(schemaID))
	copy(result[5:], avroBytes)

	return result, nil
}

// Events returns the producer's events channel for monitoring.
func (p *AvroProducer) Events() chan kafka.Event {
	return p.producer.Events()
}

// ProduceAsync produces a message asynchronously without waiting for delivery.
func (p *AvroProducer) ProduceAsync(topic string, key string, value interface{}, headers map[string]string) error {
	// Get or fetch schema for the topic
	subject := topic + "-value"
	schema, err := p.getSchema(subject)
	if err != nil {
		return fmt.Errorf("failed to get schema for %s: %w", subject, err)
	}

	// Get Avro schema
	avroSchema, err := p.getAvroSchema(schema)
	if err != nil {
		return fmt.Errorf("failed to parse Avro schema: %w", err)
	}

	// Serialize value with Avro
	valueBytes, err := p.serializeAvro(schema.ID(), avroSchema, value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// Build Kafka headers
	var kafkaHeaders []kafka.Header
	for k, v := range headers {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}

	// Produce without waiting
	return p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:     []byte(key),
		Value:   valueBytes,
		Headers: kafkaHeaders,
	}, nil)
}

// SimpleProducer is a producer without Avro serialization.
type SimpleProducer struct {
	producer *kafka.Producer
}

// NewSimpleProducer creates a simple Kafka producer without Schema Registry.
func NewSimpleProducer(bootstrapServers string) (*SimpleProducer, error) {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
		"acks":              "all",
	})
	if err != nil {
		return nil, err
	}

	return &SimpleProducer{producer: producer}, nil
}

// Produce sends a message synchronously.
func (p *SimpleProducer) Produce(ctx context.Context, topic string, key string, value interface{}) error {
	valueBytes, ok := value.([]byte)
	if !ok {
		if str, ok := value.(string); ok {
			valueBytes = []byte(str)
		} else {
			return fmt.Errorf("value must be []byte or string for SimpleProducer")
		}
	}
	return p.ProduceRaw(ctx, topic, []byte(key), valueBytes, nil)
}

// ProduceWithHeaders sends a message with headers.
func (p *SimpleProducer) ProduceWithHeaders(ctx context.Context, topic string, key string, value interface{}, headers map[string]string) error {
	valueBytes, ok := value.([]byte)
	if !ok {
		if str, ok := value.(string); ok {
			valueBytes = []byte(str)
		} else {
			return fmt.Errorf("value must be []byte or string for SimpleProducer")
		}
	}
	return p.ProduceRaw(ctx, topic, []byte(key), valueBytes, headers)
}

// ProduceRaw sends raw bytes.
func (p *SimpleProducer) ProduceRaw(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error {
	var kafkaHeaders []kafka.Header
	for k, v := range headers {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{Key: k, Value: []byte(v)})
	}

	deliveryChan := make(chan kafka.Event, 1)
	err := p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            key,
		Value:          value,
		Headers:        kafkaHeaders,
	}, deliveryChan)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-deliveryChan:
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			return m.TopicPartition.Error
		}
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("delivery timeout")
	}
}

// Flush flushes pending messages.
func (p *SimpleProducer) Flush(timeoutMs int) int {
	return p.producer.Flush(timeoutMs)
}

// Close closes the producer.
func (p *SimpleProducer) Close() {
	p.producer.Flush(10000)
	p.producer.Close()
}
