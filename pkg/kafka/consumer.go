package kafka

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/hamba/avro/v2"
	"github.com/riferrei/srclient"
	"github.com/twmb/franz-go/pkg/kgo"
)

// Message represents a consumed Kafka message.
type Message struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       string
	Value     interface{}
	RawValue  []byte
	Headers   map[string]string
	Timestamp time.Time
	SchemaID  int
}

// MessageHandler is a function that processes a consumed message.
type MessageHandler func(ctx context.Context, msg *Message) error

// Consumer defines the interface for consuming messages from Kafka.
type Consumer interface {
	// Subscribe subscribes to the given topics.
	Subscribe(topics []string) error

	// Consume starts consuming messages and calls the handler for each.
	Consume(ctx context.Context, handler MessageHandler) error

	// ConsumeOnce consumes a single message with timeout.
	ConsumeOnce(ctx context.Context, timeout time.Duration) (*Message, error)

	// Commit commits the current offsets.
	Commit() error

	// Close closes the consumer.
	Close() error
}

// ConsumerConfig holds configuration for the Kafka consumer.
type ConsumerConfig struct {
	BootstrapServers  string
	SchemaRegistryURL string
	GroupID           string
	AutoOffsetReset   string
	EnableAutoCommit  bool
	SessionTimeoutMs  int
	HeartbeatMs       int
	MaxPollInterval   int
}

// DefaultConsumerConfig returns a consumer config with sensible defaults.
func DefaultConsumerConfig(bootstrapServers, schemaRegistryURL, groupID string) ConsumerConfig {
	return ConsumerConfig{
		BootstrapServers:  bootstrapServers,
		SchemaRegistryURL: schemaRegistryURL,
		GroupID:           groupID,
		AutoOffsetReset:   "earliest",
		EnableAutoCommit:  true,
		SessionTimeoutMs:  30000,
		HeartbeatMs:       10000,
		MaxPollInterval:   300000,
	}
}

// AvroConsumer implements Consumer with Avro deserialization.
type AvroConsumer struct {
	client       *kgo.Client
	schemaClient *srclient.SchemaRegistryClient
	avroCache    map[int]avro.Schema
	mu           sync.RWMutex
	topics       []string
	autoCommit   bool
}

// NewAvroConsumer creates a new Avro-enabled Kafka consumer.
func NewAvroConsumer(config ConsumerConfig) (*AvroConsumer, error) {
	// Determine start offset
	var startOffset kgo.Offset
	switch config.AutoOffsetReset {
	case "earliest":
		startOffset = kgo.NewOffset().AtStart()
	case "latest":
		startOffset = kgo.NewOffset().AtEnd()
	default:
		startOffset = kgo.NewOffset().AtStart()
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(config.BootstrapServers),
		kgo.ConsumerGroup(config.GroupID),
		kgo.ConsumeResetOffset(startOffset),
		kgo.SessionTimeout(time.Duration(config.SessionTimeoutMs) * time.Millisecond),
		kgo.HeartbeatInterval(time.Duration(config.HeartbeatMs) * time.Millisecond),
	}

	if config.EnableAutoCommit {
		opts = append(opts, kgo.AutoCommitInterval(5*time.Second))
	} else {
		opts = append(opts, kgo.DisableAutoCommit())
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	schemaClient := srclient.CreateSchemaRegistryClient(config.SchemaRegistryURL)

	return &AvroConsumer{
		client:       client,
		schemaClient: schemaClient,
		avroCache:    make(map[int]avro.Schema),
		autoCommit:   config.EnableAutoCommit,
	}, nil
}

// Subscribe subscribes to the given topics.
func (c *AvroConsumer) Subscribe(topics []string) error {
	c.topics = topics
	c.client.AddConsumeTopics(topics...)
	return nil
}

// Consume starts consuming messages and calls the handler for each.
// This method blocks until the context is cancelled or an unrecoverable error occurs.
func (c *AvroConsumer) Consume(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fetches := c.client.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				for _, err := range errs {
					if err.Err == context.Canceled || err.Err == context.DeadlineExceeded {
						return err.Err
					}
					fmt.Printf("Fetch error: topic=%s partition=%d error=%v\n", err.Topic, err.Partition, err.Err)
				}
			}

			fetches.EachRecord(func(record *kgo.Record) {
				message, err := c.deserializeRecord(record)
				if err != nil {
					fmt.Printf("Failed to deserialize message: %v\n", err)
					return
				}

				if err := handler(ctx, message); err != nil {
					fmt.Printf("Handler error: %v\n", err)
				}
			})
		}
	}
}

// ConsumeOnce consumes a single message with timeout.
func (c *AvroConsumer) ConsumeOnce(ctx context.Context, timeout time.Duration) (*Message, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for message")
		default:
			fetches := c.client.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				for _, err := range errs {
					if err.Err == context.Canceled || err.Err == context.DeadlineExceeded {
						return nil, fmt.Errorf("timeout waiting for message")
					}
				}
				continue
			}

			var firstMessage *Message
			fetches.EachRecord(func(record *kgo.Record) {
				if firstMessage != nil {
					return
				}
				msg, err := c.deserializeRecord(record)
				if err == nil {
					firstMessage = msg
				}
			})

			if firstMessage != nil {
				return firstMessage, nil
			}
		}
	}
}

// Commit commits the current offsets.
func (c *AvroConsumer) Commit() error {
	return c.client.CommitUncommittedOffsets(context.Background())
}

// Close closes the consumer.
func (c *AvroConsumer) Close() error {
	c.client.Close()
	return nil
}

// deserializeRecord converts a franz-go record to our Message type.
func (c *AvroConsumer) deserializeRecord(record *kgo.Record) (*Message, error) {
	message := &Message{
		Topic:     record.Topic,
		Partition: record.Partition,
		Offset:    record.Offset,
		Key:       string(record.Key),
		RawValue:  record.Value,
		Headers:   make(map[string]string),
		Timestamp: record.Timestamp,
	}

	// Parse headers
	for _, header := range record.Headers {
		message.Headers[header.Key] = string(header.Value)
	}

	// Try to deserialize as Avro
	if len(record.Value) > 5 && record.Value[0] == 0 {
		// Has Confluent wire format
		schemaID := int(binary.BigEndian.Uint32(record.Value[1:5]))
		message.SchemaID = schemaID

		schema, err := c.getAvroSchema(schemaID)
		if err != nil {
			// Could not get schema, return raw value
			message.Value = record.Value
			return message, nil
		}

		// Deserialize Avro payload
		avroPayload := record.Value[5:]
		var value interface{}
		if err := avro.Unmarshal(schema, avroPayload, &value); err != nil {
			// Deserialization failed, return raw value
			message.Value = record.Value
			return message, nil
		}

		message.Value = value
	} else {
		// Not Avro format, return raw value
		message.Value = record.Value
	}

	return message, nil
}

// DeserializeInto deserializes the message value into the provided struct.
func (c *AvroConsumer) DeserializeInto(msg *Message, target interface{}) error {
	if msg.SchemaID == 0 {
		return fmt.Errorf("message does not have a schema ID")
	}

	schema, err := c.getAvroSchema(msg.SchemaID)
	if err != nil {
		return fmt.Errorf("failed to get schema: %w", err)
	}

	// Get Avro payload (skip wire format header)
	if len(msg.RawValue) < 5 {
		return fmt.Errorf("invalid message format")
	}
	avroPayload := msg.RawValue[5:]

	return avro.Unmarshal(schema, avroPayload, target)
}

// getAvroSchema retrieves and caches an Avro schema by ID.
func (c *AvroConsumer) getAvroSchema(schemaID int) (avro.Schema, error) {
	c.mu.RLock()
	if schema, ok := c.avroCache[schemaID]; ok {
		c.mu.RUnlock()
		return schema, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check
	if schema, ok := c.avroCache[schemaID]; ok {
		return schema, nil
	}

	// Fetch from Schema Registry
	srcSchema, err := c.schemaClient.GetSchema(schemaID)
	if err != nil {
		return nil, err
	}

	avroSchema, err := avro.Parse(srcSchema.Schema())
	if err != nil {
		return nil, err
	}

	c.avroCache[schemaID] = avroSchema
	return avroSchema, nil
}

// SimpleConsumer is a consumer without Avro deserialization.
type SimpleConsumer struct {
	client *kgo.Client
}

// NewSimpleConsumer creates a simple Kafka consumer without Schema Registry.
func NewSimpleConsumer(bootstrapServers, groupID string) (*SimpleConsumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(bootstrapServers),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		return nil, err
	}

	return &SimpleConsumer{client: client}, nil
}

// Subscribe subscribes to topics.
func (c *SimpleConsumer) Subscribe(topics []string) error {
	c.client.AddConsumeTopics(topics...)
	return nil
}

// Consume starts consuming messages.
func (c *SimpleConsumer) Consume(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fetches := c.client.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				for _, err := range errs {
					if err.Err == context.Canceled || err.Err == context.DeadlineExceeded {
						return err.Err
					}
				}
				continue
			}

			fetches.EachRecord(func(record *kgo.Record) {
				message := &Message{
					Topic:     record.Topic,
					Partition: record.Partition,
					Offset:    record.Offset,
					Key:       string(record.Key),
					Value:     record.Value,
					RawValue:  record.Value,
					Headers:   make(map[string]string),
					Timestamp: record.Timestamp,
				}

				for _, h := range record.Headers {
					message.Headers[h.Key] = string(h.Value)
				}

				if err := handler(ctx, message); err != nil {
					fmt.Printf("Handler error: %v\n", err)
				}
			})
		}
	}
}

// ConsumeOnce consumes a single message.
func (c *SimpleConsumer) ConsumeOnce(ctx context.Context, timeout time.Duration) (*Message, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for message")
		default:
			fetches := c.client.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				continue
			}

			var firstMessage *Message
			fetches.EachRecord(func(record *kgo.Record) {
				if firstMessage != nil {
					return
				}
				firstMessage = &Message{
					Topic:     record.Topic,
					Partition: record.Partition,
					Offset:    record.Offset,
					Key:       string(record.Key),
					Value:     record.Value,
					RawValue:  record.Value,
					Headers:   make(map[string]string),
					Timestamp: record.Timestamp,
				}
				for _, h := range record.Headers {
					firstMessage.Headers[h.Key] = string(h.Value)
				}
			})

			if firstMessage != nil {
				return firstMessage, nil
			}
		}
	}
}

// Commit commits offsets.
func (c *SimpleConsumer) Commit() error {
	return c.client.CommitUncommittedOffsets(context.Background())
}

// Close closes the consumer.
func (c *SimpleConsumer) Close() error {
	c.client.Close()
	return nil
}
