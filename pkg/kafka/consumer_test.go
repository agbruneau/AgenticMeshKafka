package kafka

import (
	"context"
	"testing"
)

func TestNewAvroConsumer_InvalidConfig(t *testing.T) {
	// Test with empty bootstrap servers
	cfg := ConsumerConfig{
		BootstrapServers:  "",
		SchemaRegistryURL: "http://localhost:8081",
		GroupID:           "test-group",
		Topics:            []string{"test-topic"},
	}

	_, err := NewAvroConsumer(cfg)
	if err == nil {
		t.Error("NewAvroConsumer() should return error for empty bootstrap servers")
	}

	// Test with empty schema registry URL
	cfg = ConsumerConfig{
		BootstrapServers:  "localhost:9092",
		SchemaRegistryURL: "",
		GroupID:           "test-group",
		Topics:            []string{"test-topic"},
	}

	_, err = NewAvroConsumer(cfg)
	if err == nil {
		t.Error("NewAvroConsumer() should return error for empty schema registry URL")
	}

	// Test with empty group ID
	cfg = ConsumerConfig{
		BootstrapServers:  "localhost:9092",
		SchemaRegistryURL: "http://localhost:8081",
		GroupID:           "",
		Topics:            []string{"test-topic"},
	}

	_, err = NewAvroConsumer(cfg)
	if err == nil {
		t.Error("NewAvroConsumer() should return error for empty group ID")
	}

	// Test with empty topics
	cfg = ConsumerConfig{
		BootstrapServers:  "localhost:9092",
		SchemaRegistryURL: "http://localhost:8081",
		GroupID:           "test-group",
		Topics:            []string{},
	}

	_, err = NewAvroConsumer(cfg)
	if err == nil {
		t.Error("NewAvroConsumer() should return error for empty topics")
	}
}

func TestConsume_Interface(t *testing.T) {
	// Verify that Consumer interface exists with the expected method signature
	var _ Consumer = (*AvroConsumer)(nil)
}

func TestConsumerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ConsumerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: ConsumerConfig{
				BootstrapServers:  "localhost:9092",
				SchemaRegistryURL: "http://localhost:8081",
				GroupID:           "test-group",
				Topics:            []string{"test-topic"},
			},
			wantErr: false,
		},
		{
			name: "empty bootstrap servers",
			cfg: ConsumerConfig{
				BootstrapServers:  "",
				SchemaRegistryURL: "http://localhost:8081",
				GroupID:           "test-group",
				Topics:            []string{"test-topic"},
			},
			wantErr: true,
		},
		{
			name: "empty schema registry URL",
			cfg: ConsumerConfig{
				BootstrapServers:  "localhost:9092",
				SchemaRegistryURL: "",
				GroupID:           "test-group",
				Topics:            []string{"test-topic"},
			},
			wantErr: true,
		},
		{
			name: "empty group ID",
			cfg: ConsumerConfig{
				BootstrapServers:  "localhost:9092",
				SchemaRegistryURL: "http://localhost:8081",
				GroupID:           "",
				Topics:            []string{"test-topic"},
			},
			wantErr: true,
		},
		{
			name: "empty topics",
			cfg: ConsumerConfig{
				BootstrapServers:  "localhost:9092",
				SchemaRegistryURL: "http://localhost:8081",
				GroupID:           "test-group",
				Topics:            []string{},
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

func TestMessageHandler_Signature(t *testing.T) {
	// Test that MessageHandler type has correct signature
	var handler MessageHandler = func(ctx context.Context, msg *Message) error {
		return nil
	}

	// Verify it compiles and can be called
	err := handler(context.Background(), &Message{
		Topic:     "test",
		Partition: 0,
		Offset:    0,
		Key:       []byte("key"),
		Value:     []byte("value"),
		SchemaID:  1,
	})
	if err != nil {
		t.Errorf("MessageHandler returned unexpected error: %v", err)
	}
}
