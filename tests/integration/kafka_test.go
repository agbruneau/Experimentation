//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/edalab/pkg/events"
	"github.com/edalab/pkg/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kafkacontainer "github.com/testcontainers/testcontainers-go/modules/kafka"
)

func TestKafkaProducerConsumer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start Kafka container
	kafkaContainer, err := kafkacontainer.RunContainer(ctx,
		kafkacontainer.WithClusterID("test-cluster"),
	)
	require.NoError(t, err)
	defer kafkaContainer.Terminate(ctx)

	brokers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err)

	// Create producer
	producer, err := kafka.NewAvroProducer(kafka.ProducerConfig{
		BootstrapServers:  brokers[0],
		SchemaRegistryURL: "mock://", // Use mock schema registry for tests
	})
	require.NoError(t, err)
	defer producer.Close()

	// Create consumer
	consumer, err := kafka.NewAvroConsumer(kafka.ConsumerConfig{
		BootstrapServers:  brokers[0],
		SchemaRegistryURL: "mock://",
		GroupID:           "test-group",
		AutoOffsetReset:   "earliest",
		EnableAutoCommit:  true,
	})
	require.NoError(t, err)
	defer consumer.Close()

	// Test event
	event := &events.CompteOuvert{
		EventID:      "test-123",
		Timestamp:    time.Now().Format(time.RFC3339),
		CompteID:     "compte-001",
		ClientID:     "client-001",
		TypeCompte:   "courant",
		SoldeInitial: "1000.00",
		Devise:       "EUR",
	}

	// Produce event
	err = producer.Produce(events.TopicCompteOuvert, event.EventID, event)
	require.NoError(t, err)

	// Subscribe to topic
	err = consumer.Subscribe([]string{events.TopicCompteOuvert})
	require.NoError(t, err)

	// Consume event (with timeout)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var received *events.CompteOuvert
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for event")
		default:
			msg, err := consumer.Poll(100)
			if err != nil {
				continue
			}
			if msg != nil && msg.Topic == events.TopicCompteOuvert {
				var evt events.CompteOuvert
				if err := consumer.Deserialize(msg, &evt); err == nil {
					received = &evt
					goto done
				}
			}
		}
	}

done:
	assert.NotNil(t, received)
	assert.Equal(t, event.EventID, received.EventID)
	assert.Equal(t, event.CompteID, received.CompteID)
	assert.Equal(t, event.ClientID, received.ClientID)
}

func TestKafkaTopicCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start Kafka container
	kafkaContainer, err := kafkacontainer.RunContainer(ctx,
		kafkacontainer.WithClusterID("test-cluster-2"),
	)
	require.NoError(t, err)
	defer kafkaContainer.Terminate(ctx)

	brokers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err)

	// Create admin client to verify topics
	producer, err := kafka.NewAvroProducer(kafka.ProducerConfig{
		BootstrapServers:  brokers[0],
		SchemaRegistryURL: "mock://",
	})
	require.NoError(t, err)
	defer producer.Close()

	// Produce to a topic (auto-creates it)
	event := &events.DepotEffectue{
		EventID:   "test-depot-123",
		Timestamp: time.Now().Format(time.RFC3339),
		CompteID:  "compte-001",
		Montant:   "500.00",
		Devise:    "EUR",
		Reference: "DEP-001",
	}

	err = producer.Produce(events.TopicDepotEffectue, event.EventID, event)
	require.NoError(t, err)
}
