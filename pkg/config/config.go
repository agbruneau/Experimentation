package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for EDA-Lab services
type Config struct {
	Kafka    KafkaConfig    `yaml:"kafka"`
	Schema   SchemaConfig   `yaml:"schema"`
	Postgres PostgresConfig `yaml:"postgres"`
	Service  ServiceConfig  `yaml:"service"`
}

// KafkaConfig holds Kafka connection configuration
type KafkaConfig struct {
	BootstrapServers string `yaml:"bootstrap_servers"`
	GroupID          string `yaml:"group_id"`
	AutoOffsetReset  string `yaml:"auto_offset_reset"`
}

// SchemaConfig holds Schema Registry configuration
type SchemaConfig struct {
	URL string `yaml:"url"`
}

// PostgresConfig holds PostgreSQL connection configuration
type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// ServiceConfig holds service-specific configuration
type ServiceConfig struct {
	Name        string `yaml:"name"`
	Port        int    `yaml:"port"`
	MetricsPort int    `yaml:"metrics_port"`
	LogLevel    string `yaml:"log_level"`
}

// ConnectionString returns the PostgreSQL connection string
func (c PostgresConfig) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.User, c.Password, c.Host, c.Port, c.Database)
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	config := &Config{
		Kafka: KafkaConfig{
			BootstrapServers: getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
			GroupID:          getEnv("KAFKA_GROUP_ID", ""),
			AutoOffsetReset:  getEnv("KAFKA_AUTO_OFFSET_RESET", "earliest"),
		},
		Schema: SchemaConfig{
			URL: getEnv("SCHEMA_REGISTRY_URL", "http://localhost:8081"),
		},
		Postgres: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnvInt("POSTGRES_PORT", 5432),
			Database: getEnv("POSTGRES_DB", "edalab"),
			User:     getEnv("POSTGRES_USER", "edalab"),
			Password: getEnv("POSTGRES_PASSWORD", "edalab_password"),
		},
		Service: ServiceConfig{
			Name:        getEnv("SERVICE_NAME", "unknown"),
			Port:        getEnvInt("SERVICE_PORT", 8080),
			MetricsPort: getEnvInt("METRICS_PORT", 9090),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
		},
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	if config.Kafka.AutoOffsetReset == "" {
		config.Kafka.AutoOffsetReset = "earliest"
	}
	if config.Service.LogLevel == "" {
		config.Service.LogLevel = "info"
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Kafka.BootstrapServers == "" {
		return fmt.Errorf("kafka.bootstrap_servers is required")
	}
	if c.Schema.URL == "" {
		return fmt.Errorf("schema.url is required")
	}
	if c.Postgres.Host == "" {
		return fmt.Errorf("postgres.host is required")
	}
	if c.Postgres.Port <= 0 {
		return fmt.Errorf("postgres.port must be positive")
	}
	if c.Postgres.Database == "" {
		return fmt.Errorf("postgres.database is required")
	}
	if c.Postgres.User == "" {
		return fmt.Errorf("postgres.user is required")
	}
	return nil
}

// getEnv returns environment variable value or default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns environment variable as int or default
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
