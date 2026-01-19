package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// TxFunc is a function that executes within a transaction.
type TxFunc func(ctx context.Context, tx pgx.Tx) error

// WithTransaction executes a function within a database transaction.
// It handles begin, commit, and rollback automatically.
// If the function returns an error or panics, the transaction is rolled back.
func (p *DBPool) WithTransaction(ctx context.Context, fn TxFunc) (err error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Handle panics
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(ctx)
			panic(r) // Re-panic after rollback
		}
	}()

	// Execute the function
	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithTransactionOptions executes a function within a transaction with custom options.
func (p *DBPool) WithTransactionOptions(ctx context.Context, txOptions pgx.TxOptions, fn TxFunc) (err error) {
	tx, err := p.pool.BeginTx(ctx, txOptions)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Handle panics
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(ctx)
			panic(r)
		}
	}()

	// Execute the function
	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	// Commit
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ReadOnlyTransaction executes a function within a read-only transaction.
func (p *DBPool) ReadOnlyTransaction(ctx context.Context, fn TxFunc) error {
	return p.WithTransactionOptions(ctx, pgx.TxOptions{
		AccessMode: pgx.ReadOnly,
	}, fn)
}

// SerializableTransaction executes a function within a serializable transaction.
func (p *DBPool) SerializableTransaction(ctx context.Context, fn TxFunc) error {
	return p.WithTransactionOptions(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	}, fn)
}

// RepeatableReadTransaction executes a function within a repeatable read transaction.
func (p *DBPool) RepeatableReadTransaction(ctx context.Context, fn TxFunc) error {
	return p.WithTransactionOptions(ctx, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead,
	}, fn)
}

// Querier is an interface for database query execution.
// Both pgxpool.Pool and pgx.Tx implement this interface.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgx.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

// TxQuerier wraps a transaction to implement Querier.
type TxQuerier struct {
	Tx pgx.Tx
}

// Exec executes a query.
func (q *TxQuerier) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgx.CommandTag, error) {
	return q.Tx.Exec(ctx, sql, arguments...)
}

// Query executes a query that returns rows.
func (q *TxQuerier) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return q.Tx.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns a single row.
func (q *TxQuerier) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return q.Tx.QueryRow(ctx, sql, args...)
}

// PoolQuerier wraps a pool to implement Querier.
type PoolQuerier struct {
	Pool *pgxpool.Pool
}

// Exec executes a query.
func (q *PoolQuerier) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgx.CommandTag, error) {
	return q.Pool.Exec(ctx, sql, arguments...)
}

// Query executes a query that returns rows.
func (q *PoolQuerier) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return q.Pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns a single row.
func (q *PoolQuerier) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return q.Pool.QueryRow(ctx, sql, args...)
}
