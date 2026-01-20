package kafka

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/riferrei/srclient"
	"github.com/segmentio/kafka-go"
)

// ProducerConfig holds the configuration for the Avro producer
type ProducerConfig struct {
	BootstrapServers  string
	SchemaRegistryURL string
	Topic             string
}

// Validate checks if the producer configuration is valid
func (c *ProducerConfig) Validate() error {
	if c.BootstrapServers == "" {
		return fmt.Errorf("bootstrap_servers is required")
	}
	if c.SchemaRegistryURL == "" {
		return fmt.Errorf("schema_registry_url is required")
	}
	return nil
}

// Producer defines the interface for producing messages
type Producer interface {
	Produce(ctx context.Context, topic string, key []byte, value interface{}, schemaID int) error
	Close() error
}

// AvroProducer implements the Producer interface with Avro serialization
type AvroProducer struct {
	config        ProducerConfig
	writer        *kafka.Writer
	schemaClient  *srclient.SchemaRegistryClient
	schemaCache   map[string]int
	schemaCacheMu sync.RWMutex
}

// NewAvroProducer creates a new Avro producer
func NewAvroProducer(cfg ProducerConfig) (*AvroProducer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.BootstrapServers),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireAll,
	}

	schemaClient := srclient.CreateSchemaRegistryClient(cfg.SchemaRegistryURL)

	return &AvroProducer{
		config:       cfg,
		writer:       writer,
		schemaClient: schemaClient,
		schemaCache:  make(map[string]int),
	}, nil
}

// Produce sends a message to Kafka with Avro serialization
func (p *AvroProducer) Produce(ctx context.Context, topic string, key []byte, value interface{}, schemaID int) error {
	// Serialize value to JSON (for MVP, we use JSON-compatible Avro)
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// Create wire format: magic byte (0) + 4-byte schema ID + payload
	wireFormat := make([]byte, 5+len(valueBytes))
	wireFormat[0] = 0 // Magic byte
	binary.BigEndian.PutUint32(wireFormat[1:5], uint32(schemaID))
	copy(wireFormat[5:], valueBytes)

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   key,
		Value: wireFormat,
	})
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	return nil
}

// ProduceWithSchema sends a message to Kafka, registering the schema if needed
func (p *AvroProducer) ProduceWithSchema(ctx context.Context, topic string, key []byte, value interface{}, schema string) error {
	schemaID, err := p.getOrRegisterSchema(topic+"-value", schema)
	if err != nil {
		return fmt.Errorf("failed to get schema ID: %w", err)
	}

	return p.Produce(ctx, topic, key, value, schemaID)
}

// getOrRegisterSchema returns the schema ID, registering if necessary
func (p *AvroProducer) getOrRegisterSchema(subject string, schema string) (int, error) {
	p.schemaCacheMu.RLock()
	if id, ok := p.schemaCache[subject]; ok {
		p.schemaCacheMu.RUnlock()
		return id, nil
	}
	p.schemaCacheMu.RUnlock()

	// Register or get existing schema
	schemaObj, err := p.schemaClient.CreateSchema(subject, schema, srclient.Avro)
	if err != nil {
		return 0, fmt.Errorf("failed to register schema: %w", err)
	}

	p.schemaCacheMu.Lock()
	p.schemaCache[subject] = schemaObj.ID()
	p.schemaCacheMu.Unlock()

	return schemaObj.ID(), nil
}

// Close closes the producer
func (p *AvroProducer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
