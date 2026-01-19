package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromEnv_Success(t *testing.T) {
	// Set environment variables
	os.Setenv("KAFKA_BOOTSTRAP_SERVERS", "test-kafka:9092")
	os.Setenv("SCHEMA_REGISTRY_URL", "http://test-schema:8081")
	os.Setenv("POSTGRES_HOST", "test-postgres")
	os.Setenv("POSTGRES_PORT", "5433")
	os.Setenv("SERVICE_NAME", "test-service")
	defer func() {
		os.Unsetenv("KAFKA_BOOTSTRAP_SERVERS")
		os.Unsetenv("SCHEMA_REGISTRY_URL")
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("POSTGRES_PORT")
		os.Unsetenv("SERVICE_NAME")
	}()

	config, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if config.Kafka.BootstrapServers != "test-kafka:9092" {
		t.Errorf("Kafka.BootstrapServers = %v, want %v", config.Kafka.BootstrapServers, "test-kafka:9092")
	}
	if config.Schema.URL != "http://test-schema:8081" {
		t.Errorf("Schema.URL = %v, want %v", config.Schema.URL, "http://test-schema:8081")
	}
	if config.Postgres.Host != "test-postgres" {
		t.Errorf("Postgres.Host = %v, want %v", config.Postgres.Host, "test-postgres")
	}
	if config.Postgres.Port != 5433 {
		t.Errorf("Postgres.Port = %v, want %v", config.Postgres.Port, 5433)
	}
	if config.Service.Name != "test-service" {
		t.Errorf("Service.Name = %v, want %v", config.Service.Name, "test-service")
	}
}

func TestLoadFromEnv_Defaults(t *testing.T) {
	// Clear any existing environment variables
	os.Unsetenv("KAFKA_BOOTSTRAP_SERVERS")
	os.Unsetenv("POSTGRES_PORT")

	config, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	// Check defaults are applied
	if config.Kafka.BootstrapServers != "localhost:9092" {
		t.Errorf("Kafka.BootstrapServers = %v, want %v", config.Kafka.BootstrapServers, "localhost:9092")
	}
	if config.Kafka.AutoOffsetReset != "earliest" {
		t.Errorf("Kafka.AutoOffsetReset = %v, want %v", config.Kafka.AutoOffsetReset, "earliest")
	}
	if config.Postgres.Port != 5432 {
		t.Errorf("Postgres.Port = %v, want %v", config.Postgres.Port, 5432)
	}
	if config.Service.Port != 8080 {
		t.Errorf("Service.Port = %v, want %v", config.Service.Port, 8080)
	}
	if config.Logging.Level != "INFO" {
		t.Errorf("Logging.Level = %v, want %v", config.Logging.Level, "INFO")
	}
}

func TestLoadFromFile_Success(t *testing.T) {
	// Create a temporary config file
	content := `
kafka:
  bootstrap_servers: "file-kafka:9092"
  group_id: "file-group"
  auto_offset_reset: "latest"

schema:
  url: "http://file-schema:8081"

postgres:
  host: "file-postgres"
  port: 5432
  database: "filedb"
  user: "fileuser"
  password: "filepass"

service:
  name: "file-service"
  port: 8081
  metrics_port: 9091

logging:
  level: "DEBUG"
  format: "text"
`
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := LoadFromFile(configFile)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if config.Kafka.BootstrapServers != "file-kafka:9092" {
		t.Errorf("Kafka.BootstrapServers = %v, want %v", config.Kafka.BootstrapServers, "file-kafka:9092")
	}
	if config.Kafka.GroupID != "file-group" {
		t.Errorf("Kafka.GroupID = %v, want %v", config.Kafka.GroupID, "file-group")
	}
	if config.Kafka.AutoOffsetReset != "latest" {
		t.Errorf("Kafka.AutoOffsetReset = %v, want %v", config.Kafka.AutoOffsetReset, "latest")
	}
	if config.Postgres.Database != "filedb" {
		t.Errorf("Postgres.Database = %v, want %v", config.Postgres.Database, "filedb")
	}
	if config.Service.Name != "file-service" {
		t.Errorf("Service.Name = %v, want %v", config.Service.Name, "file-service")
	}
	if config.Service.Port != 8081 {
		t.Errorf("Service.Port = %v, want %v", config.Service.Port, 8081)
	}
	if config.Logging.Level != "DEBUG" {
		t.Errorf("Logging.Level = %v, want %v", config.Logging.Level, "DEBUG")
	}
}

func TestLoadFromFile_FileNotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("LoadFromFile() expected error for non-existent file, got nil")
	}
}

func TestLoadFromFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(configFile, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := LoadFromFile(configFile)
	if err == nil {
		t.Error("LoadFromFile() expected error for invalid YAML, got nil")
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
		},
		Service: ServiceConfig{
			Name: "test-service",
			Port: 8080,
		},
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestValidate_MissingKafkaBootstrap(t *testing.T) {
	config := &Config{
		Kafka: KafkaConfig{
			BootstrapServers: "",
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
		Service: ServiceConfig{
			Name: "test-service",
			Port: 8080,
		},
	}

	err := config.Validate()
	if err != ErrMissingKafkaBootstrap {
		t.Errorf("Validate() error = %v, want %v", err, ErrMissingKafkaBootstrap)
	}
}

func TestValidate_MissingSchemaURL(t *testing.T) {
	config := &Config{
		Kafka: KafkaConfig{
			BootstrapServers: "localhost:9092",
		},
		Schema: SchemaConfig{
			URL: "",
		},
		Postgres: PostgresConfig{
			Host:     "localhost",
			Port:     5432,
			Database: "testdb",
			User:     "testuser",
		},
		Service: ServiceConfig{
			Name: "test-service",
			Port: 8080,
		},
	}

	err := config.Validate()
	if err != ErrMissingSchemaURL {
		t.Errorf("Validate() error = %v, want %v", err, ErrMissingSchemaURL)
	}
}

func TestValidate_MissingPostgresHost(t *testing.T) {
	config := &Config{
		Kafka: KafkaConfig{
			BootstrapServers: "localhost:9092",
		},
		Schema: SchemaConfig{
			URL: "http://localhost:8081",
		},
		Postgres: PostgresConfig{
			Host:     "",
			Port:     5432,
			Database: "testdb",
			User:     "testuser",
		},
		Service: ServiceConfig{
			Name: "test-service",
			Port: 8080,
		},
	}

	err := config.Validate()
	if err != ErrMissingPostgresHost {
		t.Errorf("Validate() error = %v, want %v", err, ErrMissingPostgresHost)
	}
}

func TestValidate_InvalidPostgresPort(t *testing.T) {
	config := &Config{
		Kafka: KafkaConfig{
			BootstrapServers: "localhost:9092",
		},
		Schema: SchemaConfig{
			URL: "http://localhost:8081",
		},
		Postgres: PostgresConfig{
			Host:     "localhost",
			Port:     0,
			Database: "testdb",
			User:     "testuser",
		},
		Service: ServiceConfig{
			Name: "test-service",
			Port: 8080,
		},
	}

	err := config.Validate()
	if err != ErrInvalidPostgresPort {
		t.Errorf("Validate() error = %v, want %v", err, ErrInvalidPostgresPort)
	}
}

func TestValidate_MissingServiceName(t *testing.T) {
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
		},
		Service: ServiceConfig{
			Name: "",
			Port: 8080,
		},
	}

	err := config.Validate()
	if err != ErrMissingServiceName {
		t.Errorf("Validate() error = %v, want %v", err, ErrMissingServiceName)
	}
}

func TestValidate_InvalidServicePort(t *testing.T) {
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
		},
		Service: ServiceConfig{
			Name: "test-service",
			Port: 0,
		},
	}

	err := config.Validate()
	if err != ErrInvalidServicePort {
		t.Errorf("Validate() error = %v, want %v", err, ErrInvalidServicePort)
	}
}

func TestPostgresConfig_ConnectionString(t *testing.T) {
	config := PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		User:     "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}

	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
	result := config.ConnectionString()

	if result != expected {
		t.Errorf("ConnectionString() = %v, want %v", result, expected)
	}
}

func TestPostgresConfig_ConnectionString_DefaultSSL(t *testing.T) {
	config := PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		User:     "testuser",
		Password: "testpass",
	}

	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
	result := config.ConnectionString()

	if result != expected {
		t.Errorf("ConnectionString() = %v, want %v", result, expected)
	}
}

func TestLoad_FromConfigFile(t *testing.T) {
	// Create a temporary config file
	content := `
kafka:
  bootstrap_servers: "env-kafka:9092"
schema:
  url: "http://env-schema:8081"
postgres:
  host: "env-postgres"
  port: 5432
  database: "envdb"
  user: "envuser"
  password: "envpass"
service:
  name: "env-service"
  port: 8080
`
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	os.Setenv("CONFIG_FILE", configFile)
	defer os.Unsetenv("CONFIG_FILE")

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if config.Kafka.BootstrapServers != "env-kafka:9092" {
		t.Errorf("Kafka.BootstrapServers = %v, want %v", config.Kafka.BootstrapServers, "env-kafka:9092")
	}
}

func TestLoad_FromEnvWhenNoConfigFile(t *testing.T) {
	os.Unsetenv("CONFIG_FILE")
	os.Setenv("KAFKA_BOOTSTRAP_SERVERS", "direct-kafka:9092")
	defer os.Unsetenv("KAFKA_BOOTSTRAP_SERVERS")

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if config.Kafka.BootstrapServers != "direct-kafka:9092" {
		t.Errorf("Kafka.BootstrapServers = %v, want %v", config.Kafka.BootstrapServers, "direct-kafka:9092")
	}
}
