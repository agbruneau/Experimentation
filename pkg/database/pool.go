package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBPool wraps pgxpool.Pool with additional functionality
type DBPool struct {
	pool *pgxpool.Pool
}

// Config holds database connection configuration
type Config struct {
	Host            string
	Port            int
	Database        string
	User            string
	Password        string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// ConnectionString returns the PostgreSQL connection string
func (c Config) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.User, c.Password, c.Host, c.Port, c.Database)
}

// NewDBPool creates a new database connection pool
func NewDBPool(ctx context.Context, config Config) (*DBPool, error) {
	connString := config.ConnectionString()

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Set pool configuration
	if config.MaxConns > 0 {
		poolConfig.MaxConns = config.MaxConns
	} else {
		poolConfig.MaxConns = 10
	}

	if config.MinConns > 0 {
		poolConfig.MinConns = config.MinConns
	} else {
		poolConfig.MinConns = 2
	}

	if config.MaxConnLifetime > 0 {
		poolConfig.MaxConnLifetime = config.MaxConnLifetime
	} else {
		poolConfig.MaxConnLifetime = time.Hour
	}

	if config.MaxConnIdleTime > 0 {
		poolConfig.MaxConnIdleTime = config.MaxConnIdleTime
	} else {
		poolConfig.MaxConnIdleTime = 30 * time.Minute
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DBPool{pool: pool}, nil
}

// NewDBPoolFromString creates a new database connection pool from a connection string
func NewDBPoolFromString(ctx context.Context, connString string) (*DBPool, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DBPool{pool: pool}, nil
}

// Close closes the database connection pool
func (p *DBPool) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}

// Pool returns the underlying pgxpool.Pool
func (p *DBPool) Pool() *pgxpool.Pool {
	return p.pool
}

// HealthCheck performs a health check on the database connection
func (p *DBPool) HealthCheck(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Stat returns pool statistics
func (p *DBPool) Stat() *pgxpool.Stat {
	return p.pool.Stat()
}

// TxFunc is a function that runs within a transaction
type TxFunc func(ctx context.Context, tx pgx.Tx) error

// WithTransaction executes a function within a database transaction
func (p *DBPool) WithTransaction(ctx context.Context, fn TxFunc) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(ctx)
			panic(r) // re-throw panic after rollback
		}
	}()

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("failed to rollback transaction: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Exec executes a query without returning rows
func (p *DBPool) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := p.pool.Exec(ctx, sql, args...)
	return err
}

// QueryRow executes a query that returns a single row
func (p *DBPool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}

// Query executes a query that returns rows
func (p *DBPool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}
