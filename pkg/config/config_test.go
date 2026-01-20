package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromEnv_Success(t *testing.T) {
	// Set required environment variables
	os.Setenv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092")
	os.Setenv("SCHEMA_REGISTRY_URL", "http://localhost:8081")
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")
	os.Setenv("POSTGRES_DB", "testdb")
	os.Setenv("POSTGRES_USER", "testuser")
	os.Setenv("POSTGRES_PASSWORD", "testpass")
	defer func() {
		os.Unsetenv("KAFKA_BOOTSTRAP_SERVERS")
		os.Unsetenv("SCHEMA_REGISTRY_URL")
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("POSTGRES_PORT")
		os.Unsetenv("POSTGRES_DB")
		os.Unsetenv("POSTGRES_USER")
		os.Unsetenv("POSTGRES_PASSWORD")
	}()

	config, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}

	if config.Kafka.BootstrapServers != "localhost:9092" {
		t.Errorf("expected bootstrap servers localhost:9092, got %s", config.Kafka.BootstrapServers)
	}
	if config.Postgres.Database != "testdb" {
		t.Errorf("expected database testdb, got %s", config.Postgres.Database)
	}
}

func TestLoadFromEnv_Defaults(t *testing.T) {
	// Clear environment
	os.Clearenv()

	config, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}

	// Check defaults
	if config.Kafka.AutoOffsetReset != "earliest" {
		t.Errorf("expected auto_offset_reset earliest, got %s", config.Kafka.AutoOffsetReset)
	}
	if config.Service.LogLevel != "info" {
		t.Errorf("expected log_level info, got %s", config.Service.LogLevel)
	}
}

func TestLoadFromFile_Success(t *testing.T) {
	// Create a temporary config file
	content := `
kafka:
  bootstrap_servers: "kafka:9092"
  auto_offset_reset: "latest"
schema:
  url: "http://schema-registry:8081"
postgres:
  host: "postgres"
  port: 5432
  database: "edalab"
  user: "edalab"
  password: "password123"
service:
  name: "test-service"
  port: 8080
  log_level: "debug"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if config.Kafka.BootstrapServers != "kafka:9092" {
		t.Errorf("expected bootstrap servers kafka:9092, got %s", config.Kafka.BootstrapServers)
	}
	if config.Kafka.AutoOffsetReset != "latest" {
		t.Errorf("expected auto_offset_reset latest, got %s", config.Kafka.AutoOffsetReset)
	}
	if config.Service.Name != "test-service" {
		t.Errorf("expected service name test-service, got %s", config.Service.Name)
	}
}

func TestLoadFromFile_FileNotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	config := &Config{
		Kafka: KafkaConfig{
			BootstrapServers: "localhost:9092",
		},
		Schema: SchemaConfig{
			URL: "http://localhost:8081",
		},
		Postgres: PostgresConfig{
			Host:     "localhost",
			Port:     5432,
			Database: "testdb",
			User:     "testuser",
			Password: "testpass",
		},
	}

	if err := config.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}
}

func TestValidate_InvalidConfig_MissingKafka(t *testing.T) {
	config := &Config{
		Kafka: KafkaConfig{
			BootstrapServers: "", // Missing
		},
		Schema: SchemaConfig{
			URL: "http://localhost:8081",
		},
		Postgres: PostgresConfig{
			Host:     "localhost",
			Port:     5432,
			Database: "testdb",
			User:     "testuser",
		},
	}

	if err := config.Validate(); err == nil {
		t.Error("expected error for missing kafka bootstrap servers")
	}
}

func TestValidate_InvalidConfig_InvalidPort(t *testing.T) {
	config := &Config{
		Kafka: KafkaConfig{
			BootstrapServers: "localhost:9092",
		},
		Schema: SchemaConfig{
			URL: "http://localhost:8081",
		},
		Postgres: PostgresConfig{
			Host:     "localhost",
			Port:     -1, // Invalid
			Database: "testdb",
			User:     "testuser",
		},
	}

	if err := config.Validate(); err == nil {
		t.Error("expected error for invalid port")
	}
}

func TestPostgresConfig_ConnectionString(t *testing.T) {
	config := PostgresConfig{
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
