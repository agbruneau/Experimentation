//go:build integration
// +build integration

package kafka

import (
	"context"
	"fmt"
	"testing"
	"time"
)

const (
	testBootstrapServers  = "localhost:9092"
	testSchemaRegistryURL = "http://localhost:8081"
)

func TestSimpleProducerConsumer(t *testing.T) {
	ctx := context.Background()
	topic := fmt.Sprintf("test-simple-%d", time.Now().UnixNano())
	groupID := fmt.Sprintf("test-group-%d", time.Now().UnixNano())

	// Create producer
	producer, err := NewSimpleProducer(testBootstrapServers)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	// Produce message
	testMessage := "Hello, Kafka!"
	err = producer.Produce(ctx, topic, "test-key", testMessage)
	if err != nil {
		t.Fatalf("Failed to produce message: %v", err)
	}
	t.Logf("Produced message to topic %s", topic)

	// Create consumer
	consumer, err := NewSimpleConsumer(testBootstrapServers, groupID)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Subscribe
	err = consumer.Subscribe([]string{topic})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Consume message
	msg, err := consumer.ConsumeOnce(ctx, 30*time.Second)
	if err != nil {
		t.Fatalf("Failed to consume message: %v", err)
	}

	// Verify
	if string(msg.RawValue) != testMessage {
		t.Errorf("Message content mismatch: got %s, want %s", string(msg.RawValue), testMessage)
	}
	if msg.Key != "test-key" {
		t.Errorf("Message key mismatch: got %s, want %s", msg.Key, "test-key")
	}
	t.Logf("Consumed message: %s", string(msg.RawValue))
}

func TestProducerWithHeaders(t *testing.T) {
	ctx := context.Background()
	topic := fmt.Sprintf("test-headers-%d", time.Now().UnixNano())
	groupID := fmt.Sprintf("test-group-%d", time.Now().UnixNano())

	// Create producer
	producer, err := NewSimpleProducer(testBootstrapServers)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	// Produce with headers
	headers := map[string]string{
		"trace-id":    "abc123",
		"event-type":  "test-event",
		"correlation": "xyz789",
	}
	err = producer.ProduceWithHeaders(ctx, topic, "key", "value", headers)
	if err != nil {
		t.Fatalf("Failed to produce with headers: %v", err)
	}

	// Create consumer
	consumer, err := NewSimpleConsumer(testBootstrapServers, groupID)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	err = consumer.Subscribe([]string{topic})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Consume and verify headers
	msg, err := consumer.ConsumeOnce(ctx, 30*time.Second)
	if err != nil {
		t.Fatalf("Failed to consume: %v", err)
	}

	for k, v := range headers {
		if msg.Headers[k] != v {
			t.Errorf("Header %s mismatch: got %s, want %s", k, msg.Headers[k], v)
		}
	}
	t.Logf("All headers verified")
}

func TestMultipleMessages(t *testing.T) {
	ctx := context.Background()
	topic := fmt.Sprintf("test-multiple-%d", time.Now().UnixNano())
	groupID := fmt.Sprintf("test-group-%d", time.Now().UnixNano())
	messageCount := 10

	// Create producer
	producer, err := NewSimpleProducer(testBootstrapServers)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	// Produce multiple messages
	for i := 0; i < messageCount; i++ {
		msg := fmt.Sprintf("Message %d", i)
		err = producer.Produce(ctx, topic, fmt.Sprintf("key-%d", i), msg)
		if err != nil {
			t.Fatalf("Failed to produce message %d: %v", i, err)
		}
	}
	t.Logf("Produced %d messages", messageCount)

	// Create consumer
	consumer, err := NewSimpleConsumer(testBootstrapServers, groupID)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	err = consumer.Subscribe([]string{topic})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Consume all messages
	received := 0
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for received < messageCount {
		msg, err := consumer.ConsumeOnce(ctx, 5*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			continue
		}
		received++
		t.Logf("Received: %s", string(msg.RawValue))
	}

	if received != messageCount {
		t.Errorf("Received %d messages, want %d", received, messageCount)
	}
}

func TestConsumerHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	topic := fmt.Sprintf("test-handler-%d", time.Now().UnixNano())
	groupID := fmt.Sprintf("test-group-%d", time.Now().UnixNano())

	// Create producer
	producer, err := NewSimpleProducer(testBootstrapServers)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	// Produce message
	err = producer.Produce(ctx, topic, "key", "test message for handler")
	if err != nil {
		t.Fatalf("Failed to produce: %v", err)
	}

	// Create consumer
	consumer, err := NewSimpleConsumer(testBootstrapServers, groupID)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	err = consumer.Subscribe([]string{topic})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Consume with handler
	receivedChan := make(chan *Message, 1)
	handlerCtx, handlerCancel := context.WithCancel(ctx)

	handler := func(ctx context.Context, msg *Message) error {
		receivedChan <- msg
		handlerCancel() // Stop after first message
		return nil
	}

	go func() {
		consumer.Consume(handlerCtx, handler)
	}()

	select {
	case msg := <-receivedChan:
		if string(msg.RawValue) != "test message for handler" {
			t.Errorf("Unexpected message: %s", string(msg.RawValue))
		}
		t.Log("Handler received message successfully")
	case <-ctx.Done():
		t.Fatal("Timeout waiting for handler")
	}
}
