package kafka

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/riferrei/srclient"
	"github.com/segmentio/kafka-go"
)

// ConsumerConfig holds the configuration for the Avro consumer
type ConsumerConfig struct {
	BootstrapServers  string
	SchemaRegistryURL string
	GroupID           string
	Topics            []string
	AutoOffsetReset   string // "earliest" or "latest"
}

// Validate checks if the consumer configuration is valid
func (c *ConsumerConfig) Validate() error {
	if c.BootstrapServers == "" {
		return fmt.Errorf("bootstrap_servers is required")
	}
	if c.SchemaRegistryURL == "" {
		return fmt.Errorf("schema_registry_url is required")
	}
	if c.GroupID == "" {
		return fmt.Errorf("group_id is required")
	}
	if len(c.Topics) == 0 {
		return fmt.Errorf("at least one topic is required")
	}
	return nil
}

// Message represents a consumed Kafka message with decoded metadata
type Message struct {
	Topic     string
	Partition int
	Offset    int64
	Key       []byte
	Value     []byte
	SchemaID  int
	Headers   map[string]string
}

// MessageHandler is a function type for processing consumed messages
type MessageHandler func(ctx context.Context, msg *Message) error

// Consumer defines the interface for consuming messages
type Consumer interface {
	Subscribe(topics []string) error
	Consume(ctx context.Context, handler MessageHandler) error
	Close() error
}

// AvroConsumer implements the Consumer interface with Avro deserialization
type AvroConsumer struct {
	config       ConsumerConfig
	reader       *kafka.Reader
	schemaClient *srclient.SchemaRegistryClient
}

// NewAvroConsumer creates a new Avro consumer
func NewAvroConsumer(cfg ConsumerConfig) (*AvroConsumer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	startOffset := kafka.FirstOffset
	if cfg.AutoOffsetReset == "latest" {
		startOffset = kafka.LastOffset
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{cfg.BootstrapServers},
		GroupID:     cfg.GroupID,
		GroupTopics: cfg.Topics,
		StartOffset: startOffset,
	})

	schemaClient := srclient.CreateSchemaRegistryClient(cfg.SchemaRegistryURL)

	return &AvroConsumer{
		config:       cfg,
		reader:       reader,
		schemaClient: schemaClient,
	}, nil
}

// Subscribe subscribes to the specified topics
func (c *AvroConsumer) Subscribe(topics []string) error {
	// kafka-go handles subscription through ReaderConfig.GroupTopics
	// This method exists for interface compatibility
	return nil
}

// Consume starts consuming messages and calls the handler for each message
func (c *AvroConsumer) Consume(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return fmt.Errorf("failed to fetch message: %w", err)
			}

			// Parse wire format to extract schema ID
			schemaID, payload, err := parseWireFormat(msg.Value)
			if err != nil {
				return fmt.Errorf("failed to parse wire format: %w", err)
			}

			// Convert headers
			headers := make(map[string]string)
			for _, h := range msg.Headers {
				headers[h.Key] = string(h.Value)
			}

			// Create message struct
			message := &Message{
				Topic:     msg.Topic,
				Partition: msg.Partition,
				Offset:    msg.Offset,
				Key:       msg.Key,
				Value:     payload,
				SchemaID:  schemaID,
				Headers:   headers,
			}

			// Call handler
			if err := handler(ctx, message); err != nil {
				return fmt.Errorf("handler error: %w", err)
			}

			// Commit offset
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				return fmt.Errorf("failed to commit message: %w", err)
			}
		}
	}
}

// ConsumeWithDecoding consumes messages and decodes them using the schema registry
func (c *AvroConsumer) ConsumeWithDecoding(ctx context.Context, handler func(ctx context.Context, msg *Message, decoded map[string]interface{}) error) error {
	return c.Consume(ctx, func(ctx context.Context, msg *Message) error {
		// Decode JSON payload
		var decoded map[string]interface{}
		if err := json.Unmarshal(msg.Value, &decoded); err != nil {
			return fmt.Errorf("failed to decode message: %w", err)
		}
		return handler(ctx, msg, decoded)
	})
}

// Close closes the consumer
func (c *AvroConsumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

// parseWireFormat extracts the schema ID and payload from the Confluent wire format
// Wire format: [magic byte (0)] [4-byte schema ID] [payload]
func parseWireFormat(data []byte) (int, []byte, error) {
	if len(data) < 5 {
		return 0, nil, fmt.Errorf("message too short: expected at least 5 bytes, got %d", len(data))
	}

	if data[0] != 0 {
		return 0, nil, fmt.Errorf("invalid magic byte: expected 0, got %d", data[0])
	}

	schemaID := int(binary.BigEndian.Uint32(data[1:5]))
	payload := data[5:]

	return schemaID, payload, nil
}
