package config

import (
	"os"
	"testing"
)

func TestLoadFromEnv_Success(t *testing.T) {
	// Set environment variables
	os.Setenv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092")
	os.Setenv("KAFKA_AUTO_OFFSET_RESET", "earliest")
	os.Setenv("SCHEMA_REGISTRY_URL", "http://localhost:8081")
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")
	os.Setenv("POSTGRES_DB", "edalab")
	os.Setenv("POSTGRES_USER", "edalab")
	os.Setenv("POSTGRES_PASSWORD", "edalab_password")
	os.Setenv("SERVICE_NAME", "test-service")
	os.Setenv("SERVICE_PORT", "8080")
	defer func() {
		os.Unsetenv("KAFKA_BOOTSTRAP_SERVERS")
		os.Unsetenv("KAFKA_AUTO_OFFSET_RESET")
		os.Unsetenv("SCHEMA_REGISTRY_URL")
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("POSTGRES_PORT")
		os.Unsetenv("POSTGRES_DB")
		os.Unsetenv("POSTGRES_USER")
		os.Unsetenv("POSTGRES_PASSWORD")
		os.Unsetenv("SERVICE_NAME")
		os.Unsetenv("SERVICE_PORT")
	}()

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() returned error: %v", err)
	}

	if cfg.Kafka.BootstrapServers != "localhost:9092" {
		t.Errorf("Kafka.BootstrapServers = %q, want %q", cfg.Kafka.BootstrapServers, "localhost:9092")
	}
	if cfg.Kafka.AutoOffsetReset != "earliest" {
		t.Errorf("Kafka.AutoOffsetReset = %q, want %q", cfg.Kafka.AutoOffsetReset, "earliest")
	}
	if cfg.Schema.URL != "http://localhost:8081" {
		t.Errorf("Schema.URL = %q, want %q", cfg.Schema.URL, "http://localhost:8081")
	}
	if cfg.Postgres.Host != "localhost" {
		t.Errorf("Postgres.Host = %q, want %q", cfg.Postgres.Host, "localhost")
	}
	if cfg.Postgres.Port != 5432 {
		t.Errorf("Postgres.Port = %d, want %d", cfg.Postgres.Port, 5432)
	}
	if cfg.Postgres.Database != "edalab" {
		t.Errorf("Postgres.Database = %q, want %q", cfg.Postgres.Database, "edalab")
	}
	if cfg.Service.Name != "test-service" {
		t.Errorf("Service.Name = %q, want %q", cfg.Service.Name, "test-service")
	}
	if cfg.Service.Port != 8080 {
		t.Errorf("Service.Port = %d, want %d", cfg.Service.Port, 8080)
	}
}

func TestLoadFromEnv_MissingRequired(t *testing.T) {
	// Clear all environment variables
	os.Unsetenv("KAFKA_BOOTSTRAP_SERVERS")
	os.Unsetenv("SCHEMA_REGISTRY_URL")
	os.Unsetenv("POSTGRES_HOST")

	_, err := LoadFromEnv()
	if err == nil {
		t.Error("LoadFromEnv() should return error when required variables are missing")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Kafka: KafkaConfig{
			BootstrapServers: "localhost:9092",
			AutoOffsetReset:  "earliest",
		},
		Schema: SchemaConfig{
			URL: "http://localhost:8081",
		},
		Postgres: PostgresConfig{
			Host:     "localhost",
			Port:     5432,
			Database: "edalab",
			User:     "edalab",
			Password: "password",
		},
		Service: ServiceConfig{
			Name: "test-service",
			Port: 8080,
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() returned error for valid config: %v", err)
	}
}

func TestValidate_InvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
	}{
		{
			name: "empty kafka bootstrap servers",
			cfg: &Config{
				Kafka: KafkaConfig{
					BootstrapServers: "",
				},
				Schema:   SchemaConfig{URL: "http://localhost:8081"},
				Postgres: PostgresConfig{Host: "localhost", Port: 5432, Database: "db", User: "user", Password: "pass"},
				Service:  ServiceConfig{Name: "svc", Port: 8080},
			},
		},
		{
			name: "invalid port 0",
			cfg: &Config{
				Kafka:    KafkaConfig{BootstrapServers: "localhost:9092"},
				Schema:   SchemaConfig{URL: "http://localhost:8081"},
				Postgres: PostgresConfig{Host: "localhost", Port: 0, Database: "db", User: "user", Password: "pass"},
				Service:  ServiceConfig{Name: "svc", Port: 8080},
			},
		},
		{
			name: "empty schema url",
			cfg: &Config{
				Kafka:    KafkaConfig{BootstrapServers: "localhost:9092"},
				Schema:   SchemaConfig{URL: ""},
				Postgres: PostgresConfig{Host: "localhost", Port: 5432, Database: "db", User: "user", Password: "pass"},
				Service:  ServiceConfig{Name: "svc", Port: 8080},
			},
		},
		{
			name: "service port 0",
			cfg: &Config{
				Kafka:    KafkaConfig{BootstrapServers: "localhost:9092"},
				Schema:   SchemaConfig{URL: "http://localhost:8081"},
				Postgres: PostgresConfig{Host: "localhost", Port: 5432, Database: "db", User: "user", Password: "pass"},
				Service:  ServiceConfig{Name: "svc", Port: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if err == nil {
				t.Error("Validate() should return error for invalid config")
			}
		})
	}
}
