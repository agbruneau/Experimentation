// Package config provides configuration management for EDA-Lab services.
// It supports loading configuration from environment variables and YAML files.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete configuration for an EDA-Lab service.
type Config struct {
	Kafka    KafkaConfig    `yaml:"kafka"`
	Schema   SchemaConfig   `yaml:"schema"`
	Postgres PostgresConfig `yaml:"postgres"`
	Service  ServiceConfig  `yaml:"service"`
	Logging  LoggingConfig  `yaml:"logging"`
	Tracing  TracingConfig  `yaml:"tracing"`
}

// KafkaConfig holds Kafka broker configuration.
type KafkaConfig struct {
	BootstrapServers string `yaml:"bootstrap_servers"`
	GroupID          string `yaml:"group_id"`
	AutoOffsetReset  string `yaml:"auto_offset_reset"`
	EnableAutoCommit bool   `yaml:"enable_auto_commit"`
	SessionTimeoutMs int    `yaml:"session_timeout_ms"`
}

// SchemaConfig holds Schema Registry configuration.
type SchemaConfig struct {
	URL           string `yaml:"url"`
	CacheCapacity int    `yaml:"cache_capacity"`
}

// PostgresConfig holds PostgreSQL connection configuration.
type PostgresConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	Database       string `yaml:"database"`
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	MaxConnections int    `yaml:"max_connections"`
	MinConnections int    `yaml:"min_connections"`
	SSLMode        string `yaml:"ssl_mode"`
}

// ServiceConfig holds service-specific configuration.
type ServiceConfig struct {
	Name        string `yaml:"name"`
	Port        int    `yaml:"port"`
	MetricsPort int    `yaml:"metrics_port"`
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// TracingConfig holds distributed tracing configuration.
type TracingConfig struct {
	Enabled        bool    `yaml:"enabled"`
	JaegerEndpoint string  `yaml:"jaeger_endpoint"`
	SampleRate     float64 `yaml:"sample_rate"`
}

// ConnectionString returns a PostgreSQL connection string.
func (p *PostgresConfig) ConnectionString() string {
	sslMode := p.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.Database, sslMode,
	)
}

// ConnectionStringWithPool returns a PostgreSQL connection string with pool settings.
func (p *PostgresConfig) ConnectionStringWithPool() string {
	base := p.ConnectionString()
	if p.MaxConnections > 0 {
		base += fmt.Sprintf("&pool_max_conns=%d", p.MaxConnections)
	}
	if p.MinConnections > 0 {
		base += fmt.Sprintf("&pool_min_conns=%d", p.MinConnections)
	}
	return base
}

// Validation errors
var (
	ErrMissingKafkaBootstrap   = errors.New("kafka.bootstrap_servers is required")
	ErrMissingSchemaURL        = errors.New("schema.url is required")
	ErrMissingPostgresHost     = errors.New("postgres.host is required")
	ErrMissingPostgresDatabase = errors.New("postgres.database is required")
	ErrMissingPostgresUser     = errors.New("postgres.user is required")
	ErrInvalidPostgresPort     = errors.New("postgres.port must be between 1 and 65535")
	ErrMissingServiceName      = errors.New("service.name is required")
	ErrInvalidServicePort      = errors.New("service.port must be between 1 and 65535")
)

// Validate checks that all required configuration fields are set and valid.
func (c *Config) Validate() error {
	// Kafka validation
	if c.Kafka.BootstrapServers == "" {
		return ErrMissingKafkaBootstrap
	}

	// Schema Registry validation
	if c.Schema.URL == "" {
		return ErrMissingSchemaURL
	}

	// PostgreSQL validation
	if c.Postgres.Host == "" {
		return ErrMissingPostgresHost
	}
	if c.Postgres.Database == "" {
		return ErrMissingPostgresDatabase
	}
	if c.Postgres.User == "" {
		return ErrMissingPostgresUser
	}
	if c.Postgres.Port < 1 || c.Postgres.Port > 65535 {
		return ErrInvalidPostgresPort
	}

	// Service validation
	if c.Service.Name == "" {
		return ErrMissingServiceName
	}
	if c.Service.Port < 1 || c.Service.Port > 65535 {
		return ErrInvalidServicePort
	}

	return nil
}

// LoadFromFile loads configuration from a YAML file.
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	applyDefaults(config)

	return config, nil
}

// LoadFromEnv loads configuration from environment variables.
func LoadFromEnv() (*Config, error) {
	config := &Config{
		Kafka: KafkaConfig{
			BootstrapServers: getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
			GroupID:          getEnv("KAFKA_GROUP_ID", "edalab-default"),
			AutoOffsetReset:  getEnv("KAFKA_AUTO_OFFSET_RESET", "earliest"),
			EnableAutoCommit: getEnvBool("KAFKA_ENABLE_AUTO_COMMIT", true),
			SessionTimeoutMs: getEnvInt("KAFKA_SESSION_TIMEOUT_MS", 30000),
		},
		Schema: SchemaConfig{
			URL:           getEnv("SCHEMA_REGISTRY_URL", "http://localhost:8081"),
			CacheCapacity: getEnvInt("SCHEMA_CACHE_CAPACITY", 100),
		},
		Postgres: PostgresConfig{
			Host:           getEnv("POSTGRES_HOST", "localhost"),
			Port:           getEnvInt("POSTGRES_PORT", 5432),
			Database:       getEnv("POSTGRES_DB", "edalab"),
			User:           getEnv("POSTGRES_USER", "edalab"),
			Password:       getEnv("POSTGRES_PASSWORD", "edalab_password"),
			MaxConnections: getEnvInt("POSTGRES_MAX_CONNECTIONS", 10),
			MinConnections: getEnvInt("POSTGRES_MIN_CONNECTIONS", 2),
			SSLMode:        getEnv("POSTGRES_SSL_MODE", "disable"),
		},
		Service: ServiceConfig{
			Name:        getEnv("SERVICE_NAME", "edalab-service"),
			Port:        getEnvInt("SERVICE_PORT", 8080),
			MetricsPort: getEnvInt("SERVICE_METRICS_PORT", 9090),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "INFO"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Tracing: TracingConfig{
			Enabled:        getEnvBool("TRACING_ENABLED", false),
			JaegerEndpoint: getEnv("JAEGER_ENDPOINT", "http://localhost:14268/api/traces"),
			SampleRate:     getEnvFloat("TRACING_SAMPLE_RATE", 1.0),
		},
	}

	return config, nil
}

// Load loads configuration from file if CONFIG_FILE env is set, otherwise from environment.
func Load() (*Config, error) {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile != "" {
		return LoadFromFile(configFile)
	}
	return LoadFromEnv()
}

// MustLoad loads configuration and panics on error.
func MustLoad() *Config {
	config, err := Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load configuration: %v", err))
	}
	if err := config.Validate(); err != nil {
		panic(fmt.Sprintf("invalid configuration: %v", err))
	}
	return config
}

// applyDefaults sets default values for optional fields.
func applyDefaults(config *Config) {
	// Kafka defaults
	if config.Kafka.AutoOffsetReset == "" {
		config.Kafka.AutoOffsetReset = "earliest"
	}
	if config.Kafka.SessionTimeoutMs == 0 {
		config.Kafka.SessionTimeoutMs = 30000
	}

	// Schema defaults
	if config.Schema.CacheCapacity == 0 {
		config.Schema.CacheCapacity = 100
	}

	// Postgres defaults
	if config.Postgres.Port == 0 {
		config.Postgres.Port = 5432
	}
	if config.Postgres.MaxConnections == 0 {
		config.Postgres.MaxConnections = 10
	}
	if config.Postgres.MinConnections == 0 {
		config.Postgres.MinConnections = 2
	}

	// Service defaults
	if config.Service.Port == 0 {
		config.Service.Port = 8080
	}
	if config.Service.MetricsPort == 0 {
		config.Service.MetricsPort = 9090
	}

	// Logging defaults
	if config.Logging.Level == "" {
		config.Logging.Level = "INFO"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}

	// Tracing defaults
	if config.Tracing.SampleRate == 0 {
		config.Tracing.SampleRate = 1.0
	}
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
