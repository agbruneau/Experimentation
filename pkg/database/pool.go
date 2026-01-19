// Package database provides PostgreSQL database connection management.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBPool wraps a pgxpool.Pool with additional functionality.
type DBPool struct {
	pool *pgxpool.Pool
}

// PoolConfig holds configuration for the database connection pool.
type PoolConfig struct {
	Host           string
	Port           int
	Database       string
	User           string
	Password       string
	MaxConnections int32
	MinConnections int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	HealthCheckPeriod time.Duration
	SSLMode        string
}

// DefaultPoolConfig returns a pool config with sensible defaults.
func DefaultPoolConfig(host string, port int, database, user, password string) PoolConfig {
	return PoolConfig{
		Host:              host,
		Port:              port,
		Database:          database,
		User:              user,
		Password:          password,
		MaxConnections:    10,
		MinConnections:    2,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   30 * time.Minute,
		HealthCheckPeriod: time.Minute,
		SSLMode:           "disable",
	}
}

// ConnectionString returns the PostgreSQL connection string.
func (c PoolConfig) ConnectionString() string {
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, sslMode,
	)
}

// NewDBPool creates a new database connection pool.
func NewDBPool(ctx context.Context, config PoolConfig) (*DBPool, error) {
	poolConfig, err := pgxpool.ParseConfig(config.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Apply pool configuration
	poolConfig.MaxConns = config.MaxConnections
	poolConfig.MinConns = config.MinConnections
	poolConfig.MaxConnLifetime = config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = config.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = config.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DBPool{pool: pool}, nil
}

// NewDBPoolFromConnString creates a pool from a connection string.
func NewDBPoolFromConnString(ctx context.Context, connString string) (*DBPool, error) {
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

// Pool returns the underlying pgxpool.Pool.
func (p *DBPool) Pool() *pgxpool.Pool {
	return p.pool
}

// Close closes the connection pool.
func (p *DBPool) Close() {
	p.pool.Close()
}

// HealthCheck verifies the database connection is healthy.
func (p *DBPool) HealthCheck(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Stats returns pool statistics.
func (p *DBPool) Stats() *pgxpool.Stat {
	return p.pool.Stat()
}

// Exec executes a query that doesn't return rows.
func (p *DBPool) Exec(ctx context.Context, sql string, args ...interface{}) (int64, error) {
	result, err := p.pool.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// Query executes a query that returns rows.
func (p *DBPool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns a single row.
func (p *DBPool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}

// Begin starts a new transaction.
func (p *DBPool) Begin(ctx context.Context) (pgx.Tx, error) {
	return p.pool.Begin(ctx)
}

// BeginTx starts a new transaction with options.
func (p *DBPool) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return p.pool.BeginTx(ctx, txOptions)
}
