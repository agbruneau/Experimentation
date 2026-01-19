//go:build integration
// +build integration

// ============================================================================
// Infrastructure Integration Tests
// ============================================================================
// These tests verify that the infrastructure components are working correctly.
// Run with: go test -v -tags=integration ./tests/integration/...
// Prerequisites: make infra-up
// ============================================================================

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	kafkaBootstrapServers = "localhost:9092"
	schemaRegistryURL     = "http://localhost:8081"
	postgresConnString    = "postgres://edalab:edalab_password@localhost:5432/edalab"
)

// ============================================================================
// Kafka Tests
// ============================================================================

func TestKafkaConnection(t *testing.T) {
	t.Log("Testing Kafka connection...")

	// Create admin client
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": kafkaBootstrapServers,
	})
	require.NoError(t, err, "Failed to create Kafka admin client")
	defer adminClient.Close()

	// Get cluster metadata
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metadata, err := adminClient.GetMetadata(nil, true, int(10*time.Second/time.Millisecond))
	require.NoError(t, err, "Failed to get Kafka metadata")

	assert.NotEmpty(t, metadata.Brokers, "No brokers found")
	t.Logf("Connected to Kafka cluster with %d broker(s)", len(metadata.Brokers))

	for _, broker := range metadata.Brokers {
		t.Logf("  Broker: %s:%d (ID: %d)", broker.Host, broker.Port, broker.ID)
	}
}

func TestKafkaProduceConsume(t *testing.T) {
	t.Log("Testing Kafka produce/consume...")

	testTopic := fmt.Sprintf("test-topic-%d", time.Now().UnixNano())
	testMessage := fmt.Sprintf("Test message %d", time.Now().UnixNano())

	// Create the test topic
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": kafkaBootstrapServers,
	})
	require.NoError(t, err)
	defer adminClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := adminClient.CreateTopics(ctx, []kafka.TopicSpecification{
		{
			Topic:             testTopic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	})
	require.NoError(t, err)
	require.Equal(t, kafka.ErrNoError, results[0].Error.Code(), "Failed to create topic: %v", results[0].Error)

	// Cleanup: delete topic after test
	defer func() {
		adminClient.DeleteTopics(ctx, []string{testTopic})
	}()

	t.Logf("Created test topic: %s", testTopic)

	// Create producer
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": kafkaBootstrapServers,
		"acks":              "all",
	})
	require.NoError(t, err, "Failed to create producer")
	defer producer.Close()

	// Produce message
	deliveryChan := make(chan kafka.Event)
	err = producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &testTopic, Partition: kafka.PartitionAny},
		Value:          []byte(testMessage),
		Key:            []byte("test-key"),
	}, deliveryChan)
	require.NoError(t, err, "Failed to produce message")

	// Wait for delivery
	select {
	case e := <-deliveryChan:
		m := e.(*kafka.Message)
		require.Nil(t, m.TopicPartition.Error, "Delivery failed: %v", m.TopicPartition.Error)
		t.Logf("Message delivered to %s [%d] at offset %v",
			*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for delivery")
	}

	// Create consumer
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": kafkaBootstrapServers,
		"group.id":          fmt.Sprintf("test-group-%d", time.Now().UnixNano()),
		"auto.offset.reset": "earliest",
	})
	require.NoError(t, err, "Failed to create consumer")
	defer consumer.Close()

	// Subscribe and consume
	err = consumer.Subscribe(testTopic, nil)
	require.NoError(t, err, "Failed to subscribe")

	// Poll for message
	var receivedMessage string
	for i := 0; i < 30; i++ {
		msg, err := consumer.ReadMessage(time.Second)
		if err != nil {
			if err.(kafka.Error).Code() == kafka.ErrTimedOut {
				continue
			}
			t.Fatalf("Error reading message: %v", err)
		}
		receivedMessage = string(msg.Value)
		t.Logf("Received message: %s", receivedMessage)
		break
	}

	assert.Equal(t, testMessage, receivedMessage, "Message content mismatch")
}

// ============================================================================
// Schema Registry Tests
// ============================================================================

func TestSchemaRegistryConnection(t *testing.T) {
	t.Log("Testing Schema Registry connection...")

	resp, err := http.Get(schemaRegistryURL + "/subjects")
	require.NoError(t, err, "Failed to connect to Schema Registry")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	t.Logf("Schema Registry subjects: %s", string(body))
}

func TestSchemaRegistryRegisterSchema(t *testing.T) {
	t.Log("Testing Schema Registry schema registration...")

	testSubject := fmt.Sprintf("test-subject-%d", time.Now().UnixNano())

	// Avro schema
	schema := map[string]interface{}{
		"type":      "record",
		"name":      "TestEvent",
		"namespace": "com.edalab.test",
		"fields": []map[string]interface{}{
			{"name": "id", "type": "string"},
			{"name": "timestamp", "type": "long"},
			{"name": "message", "type": "string"},
		},
	}

	schemaJSON, err := json.Marshal(schema)
	require.NoError(t, err)

	// Register schema
	payload := map[string]string{
		"schema": string(schemaJSON),
	}
	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/subjects/%s/versions", schemaRegistryURL, testSubject),
		bytes.NewBuffer(payloadJSON))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err, "Failed to register schema")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Failed to register schema: %s", string(body))

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	require.NoError(t, err)

	schemaID, ok := result["id"].(float64)
	require.True(t, ok, "Schema ID not found in response")
	t.Logf("Schema registered with ID: %d", int(schemaID))

	// Retrieve schema by ID
	resp2, err := http.Get(fmt.Sprintf("%s/schemas/ids/%d", schemaRegistryURL, int(schemaID)))
	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode, "Failed to retrieve schema")
	t.Log("Schema retrieved successfully")

	// Cleanup: delete subject
	req3, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/subjects/%s", schemaRegistryURL, testSubject), nil)
	client.Do(req3)

	req4, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/subjects/%s?permanent=true", schemaRegistryURL, testSubject), nil)
	client.Do(req4)
}

// ============================================================================
// PostgreSQL Tests
// ============================================================================

func TestPostgreSQLConnection(t *testing.T) {
	t.Log("Testing PostgreSQL connection...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, postgresConnString)
	require.NoError(t, err, "Failed to connect to PostgreSQL")
	defer pool.Close()

	// Test simple query
	var result int
	err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	require.NoError(t, err, "Failed to execute query")
	assert.Equal(t, 1, result, "Unexpected query result")

	t.Log("PostgreSQL connection successful")
}

func TestPostgreSQLBancaireSchema(t *testing.T) {
	t.Log("Testing PostgreSQL bancaire schema...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, postgresConnString)
	require.NoError(t, err)
	defer pool.Close()

	// Check schema exists
	var schemaExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.schemata
			WHERE schema_name = 'bancaire'
		)
	`).Scan(&schemaExists)
	require.NoError(t, err)
	assert.True(t, schemaExists, "Schema 'bancaire' does not exist")

	// Check health_check table
	var healthStatus string
	err = pool.QueryRow(ctx, `
		SELECT status FROM bancaire.health_check LIMIT 1
	`).Scan(&healthStatus)
	require.NoError(t, err)
	assert.NotEmpty(t, healthStatus, "Health check status is empty")
	t.Logf("Health check status: %s", healthStatus)

	// Check comptes table exists
	var comptesExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'bancaire' AND table_name = 'comptes'
		)
	`).Scan(&comptesExists)
	require.NoError(t, err)
	assert.True(t, comptesExists, "Table 'bancaire.comptes' does not exist")

	// Check transactions table exists
	var transactionsExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'bancaire' AND table_name = 'transactions'
		)
	`).Scan(&transactionsExists)
	require.NoError(t, err)
	assert.True(t, transactionsExists, "Table 'bancaire.transactions' does not exist")

	t.Log("All bancaire schema tables verified")
}

func TestPostgreSQLCRUD(t *testing.T) {
	t.Log("Testing PostgreSQL CRUD operations...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, postgresConnString)
	require.NoError(t, err)
	defer pool.Close()

	testID := fmt.Sprintf("test-%d", time.Now().UnixNano())
	testClientID := fmt.Sprintf("client-%d", time.Now().UnixNano())

	// INSERT
	_, err = pool.Exec(ctx, `
		INSERT INTO bancaire.comptes (id, client_id, type_compte, devise, solde, statut)
		VALUES ($1, $2, 'COURANT', 'EUR', 1000.00, 'ACTIF')
	`, testID, testClientID)
	require.NoError(t, err, "Failed to insert compte")
	t.Log("INSERT successful")

	// SELECT
	var solde float64
	err = pool.QueryRow(ctx, `
		SELECT solde FROM bancaire.comptes WHERE id = $1
	`, testID).Scan(&solde)
	require.NoError(t, err, "Failed to select compte")
	assert.Equal(t, 1000.00, solde, "Unexpected solde")
	t.Log("SELECT successful")

	// UPDATE
	_, err = pool.Exec(ctx, `
		UPDATE bancaire.comptes SET solde = 1500.00 WHERE id = $1
	`, testID)
	require.NoError(t, err, "Failed to update compte")

	err = pool.QueryRow(ctx, `
		SELECT solde FROM bancaire.comptes WHERE id = $1
	`, testID).Scan(&solde)
	require.NoError(t, err)
	assert.Equal(t, 1500.00, solde, "Update failed")
	t.Log("UPDATE successful")

	// DELETE
	_, err = pool.Exec(ctx, `
		DELETE FROM bancaire.comptes WHERE id = $1
	`, testID)
	require.NoError(t, err, "Failed to delete compte")

	var exists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM bancaire.comptes WHERE id = $1)
	`, testID).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists, "Delete failed")
	t.Log("DELETE successful")
}
