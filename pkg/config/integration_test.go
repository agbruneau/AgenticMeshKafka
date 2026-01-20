//go:build integration

package config

import (
	"testing"
)

func TestLoadLocalConfig(t *testing.T) {
	cfg, err := LoadFromYAML("../../config/local.yaml")
	if err != nil {
		t.Fatalf("LoadFromYAML() returned error: %v", err)
	}

	if cfg.Kafka.BootstrapServers != "localhost:9092" {
		t.Errorf("Kafka.BootstrapServers = %q, want %q", cfg.Kafka.BootstrapServers, "localhost:9092")
	}
	if cfg.Schema.URL != "http://localhost:8081" {
		t.Errorf("Schema.URL = %q, want %q", cfg.Schema.URL, "http://localhost:8081")
	}
	if cfg.Postgres.Host != "localhost" {
		t.Errorf("Postgres.Host = %q, want %q", cfg.Postgres.Host, "localhost")
	}
}

func TestLoadDockerConfig(t *testing.T) {
	cfg, err := LoadFromYAML("../../config/docker.yaml")
	if err != nil {
		t.Fatalf("LoadFromYAML() returned error: %v", err)
	}

	if cfg.Kafka.BootstrapServers != "kafka:29092" {
		t.Errorf("Kafka.BootstrapServers = %q, want %q", cfg.Kafka.BootstrapServers, "kafka:29092")
	}
	if cfg.Schema.URL != "http://schema-registry:8081" {
		t.Errorf("Schema.URL = %q, want %q", cfg.Schema.URL, "http://schema-registry:8081")
	}
	if cfg.Postgres.Host != "postgres" {
		t.Errorf("Postgres.Host = %q, want %q", cfg.Postgres.Host, "postgres")
	}
}
