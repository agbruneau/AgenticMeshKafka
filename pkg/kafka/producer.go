// Package kafka provides Kafka producer and consumer with Avro serialization support.
package kafka

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hamba/avro/v2"
	"github.com/riferrei/srclient"
	"github.com/twmb/franz-go/pkg/kgo"
)

// Producer defines the interface for producing messages to Kafka.
type Producer interface {
	// Produce sends a message to the specified topic.
	Produce(ctx context.Context, topic string, key string, value interface{}) error

	// ProduceWithHeaders sends a message with custom headers.
	ProduceWithHeaders(ctx context.Context, topic string, key string, value interface{}, headers map[string]string) error

	// ProduceRaw sends a raw byte message without Avro serialization.
	ProduceRaw(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error

	// Flush waits for all messages to be delivered.
	Flush(timeoutMs int) int

	// Close closes the producer.
	Close()
}

// ProducerConfig holds configuration for the Kafka producer.
type ProducerConfig struct {
	BootstrapServers  string
	SchemaRegistryURL string
	Acks              string
	EnableIdempotence bool
	MaxInFlight       int
	LingerMs          int
	BatchSize         int
	CompressionType   string
	UseJSON           bool // Use JSON serialization instead of Avro (for testing)
}

// DefaultProducerConfig returns a producer config with sensible defaults.
func DefaultProducerConfig(bootstrapServers, schemaRegistryURL string) ProducerConfig {
	return ProducerConfig{
		BootstrapServers:  bootstrapServers,
		SchemaRegistryURL: schemaRegistryURL,
		Acks:              "all",
		EnableIdempotence: true,
		MaxInFlight:       5,
		LingerMs:          5,
		BatchSize:         16384,
		CompressionType:   "snappy",
	}
}

// AvroProducer implements Producer with Avro serialization.
type AvroProducer struct {
	client       *kgo.Client
	schemaClient *srclient.SchemaRegistryClient
	schemaCache  map[string]*srclient.Schema
	avroCache    map[int]avro.Schema
	mu           sync.RWMutex
	useJSON      bool // Use JSON serialization instead of Avro
}

// NewAvroProducer creates a new Avro-enabled Kafka producer.
func NewAvroProducer(config ProducerConfig) (*AvroProducer, error) {
	// Parse acks setting
	var acks kgo.Acks
	switch config.Acks {
	case "all", "-1":
		acks = kgo.AllISRAcks()
	case "1":
		acks = kgo.LeaderAck()
	case "0":
		acks = kgo.NoAck()
	default:
		acks = kgo.AllISRAcks()
	}

	// Create franz-go client options
	opts := []kgo.Opt{
		kgo.SeedBrokers(config.BootstrapServers),
		kgo.RequiredAcks(acks),
		kgo.ProducerLinger(time.Duration(config.LingerMs) * time.Millisecond),
		kgo.ProducerBatchMaxBytes(int32(config.BatchSize)),
		kgo.RecordRetries(3),
	}

	// Enable idempotence if requested
	if config.EnableIdempotence {
		opts = append(opts, kgo.RequiredAcks(kgo.AllISRAcks()))
	}

	// Set compression
	switch config.CompressionType {
	case "snappy":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.SnappyCompression()))
	case "gzip":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.GzipCompression()))
	case "lz4":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.Lz4Compression()))
	case "zstd":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.ZstdCompression()))
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client: %w", err)
	}

	schemaClient := srclient.CreateSchemaRegistryClient(config.SchemaRegistryURL)

	return &AvroProducer{
		client:       client,
		schemaClient: schemaClient,
		schemaCache:  make(map[string]*srclient.Schema),
		avroCache:    make(map[int]avro.Schema),
		useJSON:      config.UseJSON,
	}, nil
}

// Produce sends a message to the specified topic with Avro serialization.
func (p *AvroProducer) Produce(ctx context.Context, topic string, key string, value interface{}) error {
	return p.ProduceWithHeaders(ctx, topic, key, value, nil)
}

// ProduceWithHeaders sends a message with custom headers and Avro serialization.
func (p *AvroProducer) ProduceWithHeaders(ctx context.Context, topic string, key string, value interface{}, headers map[string]string) error {
	var valueBytes []byte
	var err error

	if p.useJSON {
		// Use JSON serialization (for testing/debugging)
		valueBytes, err = json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to serialize value to JSON: %w", err)
		}
	} else {
		// Get or fetch schema for the topic
		subject := topic + "-value"
		schema, err := p.getSchema(subject)
		if err != nil {
			return fmt.Errorf("failed to get schema for %s: %w", subject, err)
		}

		// Get Avro schema
		avroSchema, err := p.getAvroSchema(schema)
		if err != nil {
			return fmt.Errorf("failed to parse Avro schema: %w", err)
		}

		// Serialize value with Avro
		valueBytes, err = p.serializeAvro(schema.ID(), avroSchema, value)
		if err != nil {
			return fmt.Errorf("failed to serialize value: %w", err)
		}
	}

	return p.ProduceRaw(ctx, topic, []byte(key), valueBytes, headers)
}

// ProduceRaw sends a raw byte message without Avro serialization.
func (p *AvroProducer) ProduceRaw(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error {
	// Build record headers
	var recordHeaders []kgo.RecordHeader
	for k, v := range headers {
		recordHeaders = append(recordHeaders, kgo.RecordHeader{
			Key:   k,
			Value: []byte(v),
		})
	}

	// Create record
	record := &kgo.Record{
		Topic:   topic,
		Key:     key,
		Value:   value,
		Headers: recordHeaders,
	}

	// Produce synchronously
	results := p.client.ProduceSync(ctx, record)
	if err := results.FirstErr(); err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	return nil
}

// Flush waits for all messages to be delivered.
func (p *AvroProducer) Flush(timeoutMs int) int {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	if err := p.client.Flush(ctx); err != nil {
		return 1 // Return 1 to indicate unflushed messages
	}
	return 0
}

// Close closes the producer and releases resources.
func (p *AvroProducer) Close() {
	p.client.Close()
}

// getSchema retrieves a schema from cache or Schema Registry.
func (p *AvroProducer) getSchema(subject string) (*srclient.Schema, error) {
	p.mu.RLock()
	if schema, ok := p.schemaCache[subject]; ok {
		p.mu.RUnlock()
		return schema, nil
	}
	p.mu.RUnlock()

	// Fetch from Schema Registry
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if schema, ok := p.schemaCache[subject]; ok {
		return schema, nil
	}

	schema, err := p.schemaClient.GetLatestSchema(subject)
	if err != nil {
		return nil, err
	}

	p.schemaCache[subject] = schema
	return schema, nil
}

// getAvroSchema parses and caches the Avro schema.
func (p *AvroProducer) getAvroSchema(schema *srclient.Schema) (avro.Schema, error) {
	p.mu.RLock()
	if avroSchema, ok := p.avroCache[schema.ID()]; ok {
		p.mu.RUnlock()
		return avroSchema, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check
	if avroSchema, ok := p.avroCache[schema.ID()]; ok {
		return avroSchema, nil
	}

	avroSchema, err := avro.Parse(schema.Schema())
	if err != nil {
		return nil, err
	}

	p.avroCache[schema.ID()] = avroSchema
	return avroSchema, nil
}

// serializeAvro serializes a value using Avro with the Confluent wire format.
// Wire format: [0] magic byte + [1-4] schema ID (big endian) + [5+] avro payload
func (p *AvroProducer) serializeAvro(schemaID int, schema avro.Schema, value interface{}) ([]byte, error) {
	// Serialize value to Avro
	avroBytes, err := avro.Marshal(schema, value)
	if err != nil {
		return nil, err
	}

	// Build wire format message
	// Magic byte (0) + schema ID (4 bytes big endian) + payload
	result := make([]byte, 5+len(avroBytes))
	result[0] = 0 // Magic byte
	binary.BigEndian.PutUint32(result[1:5], uint32(schemaID))
	copy(result[5:], avroBytes)

	return result, nil
}

// SimpleProducer is a producer without Avro serialization.
type SimpleProducer struct {
	client *kgo.Client
}

// NewSimpleProducer creates a simple Kafka producer without Schema Registry.
func NewSimpleProducer(bootstrapServers string) (*SimpleProducer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(bootstrapServers),
		kgo.RequiredAcks(kgo.AllISRAcks()),
	)
	if err != nil {
		return nil, err
	}

	return &SimpleProducer{client: client}, nil
}

// Produce sends a message synchronously.
func (p *SimpleProducer) Produce(ctx context.Context, topic string, key string, value interface{}) error {
	valueBytes, ok := value.([]byte)
	if !ok {
		if str, ok := value.(string); ok {
			valueBytes = []byte(str)
		} else {
			return fmt.Errorf("value must be []byte or string for SimpleProducer")
		}
	}
	return p.ProduceRaw(ctx, topic, []byte(key), valueBytes, nil)
}

// ProduceWithHeaders sends a message with headers.
func (p *SimpleProducer) ProduceWithHeaders(ctx context.Context, topic string, key string, value interface{}, headers map[string]string) error {
	valueBytes, ok := value.([]byte)
	if !ok {
		if str, ok := value.(string); ok {
			valueBytes = []byte(str)
		} else {
			return fmt.Errorf("value must be []byte or string for SimpleProducer")
		}
	}
	return p.ProduceRaw(ctx, topic, []byte(key), valueBytes, headers)
}

// ProduceRaw sends raw bytes.
func (p *SimpleProducer) ProduceRaw(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error {
	var recordHeaders []kgo.RecordHeader
	for k, v := range headers {
		recordHeaders = append(recordHeaders, kgo.RecordHeader{Key: k, Value: []byte(v)})
	}

	record := &kgo.Record{
		Topic:   topic,
		Key:     key,
		Value:   value,
		Headers: recordHeaders,
	}

	results := p.client.ProduceSync(ctx, record)
	return results.FirstErr()
}

// Flush flushes pending messages.
func (p *SimpleProducer) Flush(timeoutMs int) int {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	if err := p.client.Flush(ctx); err != nil {
		return 1
	}
	return 0
}

// Close closes the producer.
func (p *SimpleProducer) Close() {
	p.client.Close()
}
