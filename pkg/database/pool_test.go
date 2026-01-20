package database

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func getTestConfig() Config {
	return Config{
		Host:     getEnv("TEST_POSTGRES_HOST", "localhost"),
		Port:     5432,
		Database: getEnv("TEST_POSTGRES_DB", "edalab"),
		User:     getEnv("TEST_POSTGRES_USER", "edalab"),
		Password: getEnv("TEST_POSTGRES_PASSWORD", "edalab_password"),
		MaxConns: 5,
		MinConns: 1,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func skipIfNoPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	config := getTestConfig()
	pool, err := NewDBPool(ctx, config)
	if err != nil {
		t.Skipf("Skipping test: PostgreSQL not available: %v", err)
	}
	pool.Close()
}

func TestNewDBPool_Success(t *testing.T) {
	skipIfNoPostgres(t)

	ctx := context.Background()
	config := getTestConfig()

	pool, err := NewDBPool(ctx, config)
	if err != nil {
		t.Fatalf("NewDBPool failed: %v", err)
	}
	defer pool.Close()

	if pool.Pool() == nil {
		t.Error("expected non-nil pool")
	}
}

func TestNewDBPool_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	config := Config{
		Host:     "nonexistent.host.local",
		Port:     5432,
		Database: "testdb",
		User:     "testuser",
		Password: "testpass",
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := NewDBPool(ctx, config)
	if err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestHealthCheck_Success(t *testing.T) {
	skipIfNoPostgres(t)

	ctx := context.Background()
	config := getTestConfig()

	pool, err := NewDBPool(ctx, config)
	if err != nil {
		t.Fatalf("NewDBPool failed: %v", err)
	}
	defer pool.Close()

	if err := pool.HealthCheck(ctx); err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}

func TestWithTransaction_Commit(t *testing.T) {
	skipIfNoPostgres(t)

	ctx := context.Background()
	config := getTestConfig()

	pool, err := NewDBPool(ctx, config)
	if err != nil {
		t.Fatalf("NewDBPool failed: %v", err)
	}
	defer pool.Close()

	// Create a test table
	_, err = pool.Pool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx (
			id SERIAL PRIMARY KEY,
			value TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}
	defer pool.Pool().Exec(ctx, "DROP TABLE IF EXISTS test_tx")

	// Test successful transaction
	err = pool.WithTransaction(ctx, func(ctx context.Context, tx interface{}) error {
		// Using type assertion to access Exec method
		pgxTx := tx.(interface {
			Exec(ctx context.Context, sql string, args ...interface{}) (interface{}, error)
		})
		_, err := pgxTx.Exec(ctx, "INSERT INTO test_tx (value) VALUES ($1)", "test_value")
		return err
	})

	if err != nil {
		t.Errorf("WithTransaction failed: %v", err)
	}

	// Verify the data was committed
	var count int
	err = pool.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM test_tx WHERE value = $1", "test_value").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}

func TestWithTransaction_Rollback(t *testing.T) {
	skipIfNoPostgres(t)

	ctx := context.Background()
	config := getTestConfig()

	pool, err := NewDBPool(ctx, config)
	if err != nil {
		t.Fatalf("NewDBPool failed: %v", err)
	}
	defer pool.Close()

	// Create a test table
	_, err = pool.Pool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx_rollback (
			id SERIAL PRIMARY KEY,
			value TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}
	defer pool.Pool().Exec(ctx, "DROP TABLE IF EXISTS test_tx_rollback")

	// Test transaction rollback
	testErr := errors.New("intentional error")
	err = pool.WithTransaction(ctx, func(ctx context.Context, tx interface{}) error {
		pgxTx := tx.(interface {
			Exec(ctx context.Context, sql string, args ...interface{}) (interface{}, error)
		})
		_, err := pgxTx.Exec(ctx, "INSERT INTO test_tx_rollback (value) VALUES ($1)", "rollback_value")
		if err != nil {
			return err
		}
		return testErr // Return error to trigger rollback
	})

	if !errors.Is(err, testErr) {
		t.Errorf("expected testErr, got: %v", err)
	}

	// Verify the data was rolled back
	var count int
	err = pool.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM test_tx_rollback WHERE value = $1", "rollback_value").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows after rollback, got %d", count)
	}
}

func TestConnectionString(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		User:     "testuser",
		Password: "testpass",
	}

	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
	if config.ConnectionString() != expected {
		t.Errorf("expected %s, got %s", expected, config.ConnectionString())
	}
}
