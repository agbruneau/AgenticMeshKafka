package kafka

import (
	"testing"
)

func TestNewAvroProducer_InvalidConfig(t *testing.T) {
	// Test with empty bootstrap servers
	cfg := ProducerConfig{
		BootstrapServers:  "",
		SchemaRegistryURL: "http://localhost:8081",
	}

	_, err := NewAvroProducer(cfg)
	if err == nil {
		t.Error("NewAvroProducer() should return error for empty bootstrap servers")
	}

	// Test with empty schema registry URL
	cfg = ProducerConfig{
		BootstrapServers:  "localhost:9092",
		SchemaRegistryURL: "",
	}

	_, err = NewAvroProducer(cfg)
	if err == nil {
		t.Error("NewAvroProducer() should return error for empty schema registry URL")
	}
}

func TestProduce_Interface(t *testing.T) {
	// Verify that Producer interface exists with the expected method signature
	var _ Producer = (*AvroProducer)(nil)
}

func TestProducerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ProducerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: ProducerConfig{
				BootstrapServers:  "localhost:9092",
				SchemaRegistryURL: "http://localhost:8081",
			},
			wantErr: false,
		},
		{
			name: "empty bootstrap servers",
			cfg: ProducerConfig{
				BootstrapServers:  "",
				SchemaRegistryURL: "http://localhost:8081",
			},
			wantErr: true,
		},
		{
			name: "empty schema registry URL",
			cfg: ProducerConfig{
				BootstrapServers:  "localhost:9092",
				SchemaRegistryURL: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewAvroProducer_Success(t *testing.T) {
	cfg := ProducerConfig{
		BootstrapServers:  "localhost:9092",
		SchemaRegistryURL: "http://localhost:8081",
	}

	producer, err := NewAvroProducer(cfg)
	if err != nil {
		t.Fatalf("NewAvroProducer() returned error: %v", err)
	}
	defer producer.Close()

	if producer.writer == nil {
		t.Error("NewAvroProducer() writer is nil")
	}
	if producer.schemaClient == nil {
		t.Error("NewAvroProducer() schemaClient is nil")
	}
	if producer.schemaCache == nil {
		t.Error("NewAvroProducer() schemaCache is nil")
	}
}
