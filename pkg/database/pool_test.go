//go:build integration
// +build integration

package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

const testConnString = "postgres://edalab:edalab_password@localhost:5432/edalab"

func TestNewDBPool_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewDBPoolFromConnString(ctx, testConnString)
	if err != nil {
		t.Fatalf("NewDBPoolFromConnString() error = %v", err)
	}
	defer pool.Close()

	// Verify connection works
	var result int
	err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("Query error = %v", err)
	}
	if result != 1 {
		t.Errorf("Query result = %v, want 1", result)
	}
}

func TestNewDBPool_InvalidConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := NewDBPoolFromConnString(ctx, "postgres://invalid:invalid@localhost:9999/invalid")
	if err == nil {
		t.Error("NewDBPoolFromConnString() expected error for invalid config, got nil")
	}
}

func TestHealthCheck_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewDBPoolFromConnString(ctx, testConnString)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	err = pool.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
}

func TestWithTransaction_Commit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewDBPoolFromConnString(ctx, testConnString)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Create test table
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx_commit (
			id SERIAL PRIMARY KEY,
			value TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
	defer pool.Exec(ctx, "DROP TABLE IF EXISTS test_tx_commit")

	// Execute transaction
	err = pool.WithTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO test_tx_commit (value) VALUES ($1)", "test-value")
		return err
	})
	if err != nil {
		t.Fatalf("WithTransaction() error = %v", err)
	}

	// Verify data was committed
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM test_tx_commit WHERE value = 'test-value'").Scan(&count)
	if err != nil {
		t.Fatalf("Query error = %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row, got %d", count)
	}
}

func TestWithTransaction_Rollback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewDBPoolFromConnString(ctx, testConnString)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Create test table
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx_rollback (
			id SERIAL PRIMARY KEY,
			value TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
	defer pool.Exec(ctx, "DROP TABLE IF EXISTS test_tx_rollback")

	// Execute transaction that returns an error
	testErr := errors.New("intentional error")
	err = pool.WithTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO test_tx_rollback (value) VALUES ($1)", "rollback-value")
		if err != nil {
			return err
		}
		return testErr
	})

	if err != testErr {
		t.Errorf("WithTransaction() error = %v, want %v", err, testErr)
	}

	// Verify data was NOT committed
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM test_tx_rollback WHERE value = 'rollback-value'").Scan(&count)
	if err != nil {
		t.Fatalf("Query error = %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 rows (rollback), got %d", count)
	}
}

func TestWithTransaction_Panic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewDBPoolFromConnString(ctx, testConnString)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Create test table
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx_panic (
			id SERIAL PRIMARY KEY,
			value TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
	defer pool.Exec(ctx, "DROP TABLE IF EXISTS test_tx_panic")

	// Execute transaction that panics
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic to be re-raised")
		}
	}()

	pool.WithTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO test_tx_panic (value) VALUES ($1)", "panic-value")
		if err != nil {
			return err
		}
		panic("intentional panic")
	})
}

func TestExec(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewDBPoolFromConnString(ctx, testConnString)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Create and use test table
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_exec (
			id SERIAL PRIMARY KEY,
			value TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
	defer pool.Exec(ctx, "DROP TABLE IF EXISTS test_exec")

	// Insert
	rowsAffected, err := pool.Exec(ctx, "INSERT INTO test_exec (value) VALUES ($1), ($2)", "val1", "val2")
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if rowsAffected != 2 {
		t.Errorf("Exec() rowsAffected = %d, want 2", rowsAffected)
	}
}

func TestQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewDBPoolFromConnString(ctx, testConnString)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Create and populate test table
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_query (
			id SERIAL PRIMARY KEY,
			value TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
	defer pool.Exec(ctx, "DROP TABLE IF EXISTS test_query")

	_, err = pool.Exec(ctx, "INSERT INTO test_query (value) VALUES ($1), ($2), ($3)", "a", "b", "c")
	if err != nil {
		t.Fatalf("Insert error = %v", err)
	}

	// Query
	rows, err := pool.Query(ctx, "SELECT value FROM test_query ORDER BY id")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("Scan error = %v", err)
		}
		values = append(values, v)
	}

	if len(values) != 3 {
		t.Errorf("Query() returned %d rows, want 3", len(values))
	}
}

func TestStats(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewDBPoolFromConnString(ctx, testConnString)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	stats := pool.Stats()
	if stats == nil {
		t.Error("Stats() returned nil")
	}

	t.Logf("Pool stats: TotalConns=%d, IdleConns=%d, AcquiredConns=%d",
		stats.TotalConns(), stats.IdleConns(), stats.AcquiredConns())
}
