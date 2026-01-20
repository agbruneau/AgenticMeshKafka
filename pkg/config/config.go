package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// KafkaConfig holds Kafka connection settings
type KafkaConfig struct {
	BootstrapServers string `yaml:"bootstrap_servers"`
	AutoOffsetReset  string `yaml:"auto_offset_reset"`
}

// SchemaConfig holds Schema Registry settings
type SchemaConfig struct {
	URL string `yaml:"url"`
}

// PostgresConfig holds PostgreSQL connection settings
type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// ServiceConfig holds service-specific settings
type ServiceConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

// Config is the root configuration structure
type Config struct {
	Kafka    KafkaConfig    `yaml:"kafka"`
	Schema   SchemaConfig   `yaml:"schema"`
	Postgres PostgresConfig `yaml:"postgres"`
	Service  ServiceConfig  `yaml:"service"`
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	cfg := &Config{}

	// Kafka config
	cfg.Kafka.BootstrapServers = os.Getenv("KAFKA_BOOTSTRAP_SERVERS")
	if cfg.Kafka.BootstrapServers == "" {
		return nil, fmt.Errorf("KAFKA_BOOTSTRAP_SERVERS is required")
	}
	cfg.Kafka.AutoOffsetReset = os.Getenv("KAFKA_AUTO_OFFSET_RESET")
	if cfg.Kafka.AutoOffsetReset == "" {
		cfg.Kafka.AutoOffsetReset = "earliest"
	}

	// Schema Registry config
	cfg.Schema.URL = os.Getenv("SCHEMA_REGISTRY_URL")
	if cfg.Schema.URL == "" {
		return nil, fmt.Errorf("SCHEMA_REGISTRY_URL is required")
	}

	// PostgreSQL config
	cfg.Postgres.Host = os.Getenv("POSTGRES_HOST")
	if cfg.Postgres.Host == "" {
		return nil, fmt.Errorf("POSTGRES_HOST is required")
	}
	portStr := os.Getenv("POSTGRES_PORT")
	if portStr == "" {
		cfg.Postgres.Port = 5432
	} else {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("POSTGRES_PORT must be a valid integer: %w", err)
		}
		cfg.Postgres.Port = port
	}
	cfg.Postgres.Database = os.Getenv("POSTGRES_DB")
	if cfg.Postgres.Database == "" {
		return nil, fmt.Errorf("POSTGRES_DB is required")
	}
	cfg.Postgres.User = os.Getenv("POSTGRES_USER")
	if cfg.Postgres.User == "" {
		return nil, fmt.Errorf("POSTGRES_USER is required")
	}
	cfg.Postgres.Password = os.Getenv("POSTGRES_PASSWORD")
	if cfg.Postgres.Password == "" {
		return nil, fmt.Errorf("POSTGRES_PASSWORD is required")
	}

	// Service config
	cfg.Service.Name = os.Getenv("SERVICE_NAME")
	if cfg.Service.Name == "" {
		cfg.Service.Name = "unknown"
	}
	servicePortStr := os.Getenv("SERVICE_PORT")
	if servicePortStr == "" {
		cfg.Service.Port = 8080
	} else {
		port, err := strconv.Atoi(servicePortStr)
		if err != nil {
			return nil, fmt.Errorf("SERVICE_PORT must be a valid integer: %w", err)
		}
		cfg.Service.Port = port
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
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
	if c.Postgres.Port <= 0 || c.Postgres.Port > 65535 {
		return fmt.Errorf("postgres.port must be between 1 and 65535")
	}
	if c.Postgres.Database == "" {
		return fmt.Errorf("postgres.database is required")
	}
	if c.Postgres.User == "" {
		return fmt.Errorf("postgres.user is required")
	}
	if c.Service.Port <= 0 || c.Service.Port > 65535 {
		return fmt.Errorf("service.port must be between 1 and 65535")
	}
	return nil
}

// LoadFromYAML loads configuration from a YAML file
func LoadFromYAML(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}
