//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/edalab/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPostgresConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("edalab_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Minute),
		),
	)
	require.NoError(t, err)
	defer pgContainer.Terminate(ctx)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create database pool
	pool, err := database.NewPool(ctx, database.Config{
		DSN:             connStr,
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	})
	require.NoError(t, err)
	defer pool.Close()

	// Test connection
	err = pool.Ping(ctx)
	require.NoError(t, err)
}

func TestPostgresSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("edalab_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Minute),
		),
	)
	require.NoError(t, err)
	defer pgContainer.Terminate(ctx)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create database pool
	pool, err := database.NewPool(ctx, database.Config{
		DSN:             connStr,
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	})
	require.NoError(t, err)
	defer pool.Close()

	// Create schema
	initSQL := `
		CREATE SCHEMA IF NOT EXISTS bancaire;
		CREATE SCHEMA IF NOT EXISTS events;

		CREATE TABLE IF NOT EXISTS bancaire.comptes (
			id VARCHAR(36) PRIMARY KEY,
			client_id VARCHAR(36) NOT NULL,
			type_compte VARCHAR(20) NOT NULL,
			solde DECIMAL(15, 2) NOT NULL DEFAULT 0,
			devise VARCHAR(3) NOT NULL DEFAULT 'EUR',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS events.processed_events (
			event_id VARCHAR(36) PRIMARY KEY,
			topic VARCHAR(100) NOT NULL,
			processed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`

	err = pool.WithTx(ctx, func(tx database.Tx) error {
		_, err := tx.Exec(ctx, initSQL)
		return err
	})
	require.NoError(t, err)

	// Verify tables exist
	var tableCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema IN ('bancaire', 'events')
	`).Scan(&tableCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, tableCount, 2)
}

func TestIdempotency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("edalab_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Minute),
		),
	)
	require.NoError(t, err)
	defer pgContainer.Terminate(ctx)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create database pool
	pool, err := database.NewPool(ctx, database.Config{
		DSN:             connStr,
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	})
	require.NoError(t, err)
	defer pool.Close()

	// Create idempotency table
	_, err = pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS events;
		CREATE TABLE IF NOT EXISTS events.processed_events (
			event_id VARCHAR(36) PRIMARY KEY,
			topic VARCHAR(100) NOT NULL,
			processed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	eventID := "test-event-123"
	topic := "test.topic"

	// First insert should succeed
	_, err = pool.Exec(ctx, `
		INSERT INTO events.processed_events (event_id, topic)
		VALUES ($1, $2)
		ON CONFLICT (event_id) DO NOTHING
	`, eventID, topic)
	require.NoError(t, err)

	// Verify event was inserted
	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM events.processed_events WHERE event_id = $1`, eventID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Second insert should be idempotent (no error, no duplicate)
	_, err = pool.Exec(ctx, `
		INSERT INTO events.processed_events (event_id, topic)
		VALUES ($1, $2)
		ON CONFLICT (event_id) DO NOTHING
	`, eventID, topic)
	require.NoError(t, err)

	// Still only one record
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM events.processed_events WHERE event_id = $1`, eventID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
