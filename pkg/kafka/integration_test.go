//go:build integration

package kafka

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

const (
	testBootstrapServers  = "localhost:9092"
	testSchemaRegistryURL = "http://localhost:8081"
)

func TestProducerConsumerRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Generate unique topic and group for this test
	testID := fmt.Sprintf("test-%d", time.Now().UnixNano())
	topic := fmt.Sprintf("integration-test-%s", testID)
	groupID := fmt.Sprintf("test-group-%s", testID)

	// Create producer
	producerCfg := ProducerConfig{
		BootstrapServers:  testBootstrapServers,
		SchemaRegistryURL: testSchemaRegistryURL,
	}

	producer, err := NewAvroProducer(producerCfg)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	// Create consumer
	consumerCfg := ConsumerConfig{
		BootstrapServers:  testBootstrapServers,
		SchemaRegistryURL: testSchemaRegistryURL,
		GroupID:           groupID,
		Topics:            []string{topic},
		AutoOffsetReset:   "earliest",
	}

	consumer, err := NewAvroConsumer(consumerCfg)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Test data
	const numMessages = 10
	testSchema := `{
		"type": "record",
		"name": "TestEvent",
		"namespace": "com.edalab.test",
		"fields": [
			{"name": "id", "type": "int"},
			{"name": "message", "type": "string"}
		]
	}`

	// Produce messages
	for i := 0; i < numMessages; i++ {
		event := map[string]interface{}{
			"id":      i,
			"message": fmt.Sprintf("Test message %d", i),
		}
		key := []byte(fmt.Sprintf("key-%d", i))

		err := producer.ProduceWithSchema(ctx, topic, key, event, testSchema)
		if err != nil {
			t.Fatalf("Failed to produce message %d: %v", i, err)
		}
	}

	t.Logf("Produced %d messages to topic %s", numMessages, topic)

	// Consume messages
	var received []*Message
	var mu sync.Mutex

	consumeCtx, consumeCancel := context.WithTimeout(ctx, 20*time.Second)
	defer consumeCancel()

	go func() {
		err := consumer.Consume(consumeCtx, func(ctx context.Context, msg *Message) error {
			mu.Lock()
			received = append(received, msg)
			count := len(received)
			mu.Unlock()

			t.Logf("Received message %d: key=%s, schemaID=%d", count, string(msg.Key), msg.SchemaID)

			if count >= numMessages {
				consumeCancel()
			}
			return nil
		})

		if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			t.Logf("Consumer error (may be expected): %v", err)
		}
	}()

	// Wait for all messages to be received
	<-consumeCtx.Done()

	mu.Lock()
	receivedCount := len(received)
	mu.Unlock()

	if receivedCount != numMessages {
		t.Errorf("Expected %d messages, received %d", numMessages, receivedCount)
	}

	// Verify messages
	for i, msg := range received {
		if msg.Topic != topic {
			t.Errorf("Message %d: expected topic %s, got %s", i, topic, msg.Topic)
		}
		if msg.SchemaID <= 0 {
			t.Errorf("Message %d: invalid schema ID %d", i, msg.SchemaID)
		}
		if len(msg.Value) == 0 {
			t.Errorf("Message %d: empty value", i)
		}
	}
}

func TestParseWireFormat(t *testing.T) {
	tests := []struct {
		name          string
		data          []byte
		wantSchemaID  int
		wantPayload   []byte
		wantErr       bool
	}{
		{
			name:          "valid wire format",
			data:          []byte{0, 0, 0, 0, 1, 'h', 'e', 'l', 'l', 'o'},
			wantSchemaID:  1,
			wantPayload:   []byte("hello"),
			wantErr:       false,
		},
		{
			name:          "schema ID 256",
			data:          []byte{0, 0, 0, 1, 0, 'd', 'a', 't', 'a'},
			wantSchemaID:  256,
			wantPayload:   []byte("data"),
			wantErr:       false,
		},
		{
			name:    "too short",
			data:    []byte{0, 0, 0},
			wantErr: true,
		},
		{
			name:    "invalid magic byte",
			data:    []byte{1, 0, 0, 0, 1, 'h', 'e', 'l', 'l', 'o'},
			wantErr: true,
		},
		{
			name:         "empty payload",
			data:         []byte{0, 0, 0, 0, 5},
			wantSchemaID: 5,
			wantPayload:  []byte{},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaID, payload, err := parseWireFormat(tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseWireFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if schemaID != tt.wantSchemaID {
					t.Errorf("parseWireFormat() schemaID = %v, want %v", schemaID, tt.wantSchemaID)
				}
				if string(payload) != string(tt.wantPayload) {
					t.Errorf("parseWireFormat() payload = %v, want %v", payload, tt.wantPayload)
				}
			}
		})
	}
}
