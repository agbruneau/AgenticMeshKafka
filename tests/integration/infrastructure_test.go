//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	kafkaBootstrapServers = "localhost:9092"
	schemaRegistryURL     = "http://localhost:8081"
	postgresConnString    = "postgres://edalab:edalab_password@localhost:5432/edalab?sslmode=disable"
)

func TestKafkaConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a test topic
	testTopic := fmt.Sprintf("test-topic-%d", time.Now().UnixNano())

	conn, err := kafka.DialLeader(ctx, "tcp", kafkaBootstrapServers, testTopic, 0)
	require.NoError(t, err, "Failed to connect to Kafka")
	defer conn.Close()

	// Create topic using the controller connection
	controllerConn, err := kafka.Dial("tcp", kafkaBootstrapServers)
	require.NoError(t, err, "Failed to dial Kafka")
	defer controllerConn.Close()

	controller, err := controllerConn.Controller()
	require.NoError(t, err, "Failed to get controller")

	controllerAddr := net.JoinHostPort(controller.Host, fmt.Sprintf("%d", controller.Port))
	topicConn, err := kafka.Dial("tcp", controllerAddr)
	require.NoError(t, err, "Failed to connect to controller")
	defer topicConn.Close()

	err = topicConn.CreateTopics(kafka.TopicConfig{
		Topic:             testTopic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	require.NoError(t, err, "Failed to create test topic")

	// Clean up: delete the test topic at the end
	defer func() {
		_ = topicConn.DeleteTopics(testTopic)
	}()

	// Create writer (producer)
	writer := &kafka.Writer{
		Addr:         kafka.TCP(kafkaBootstrapServers),
		Topic:        testTopic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
	}
	defer writer.Close()

	// Produce a test message
	testMessage := "Hello Kafka from EDA-Lab!"
	err = writer.WriteMessages(ctx, kafka.Message{
		Value: []byte(testMessage),
	})
	require.NoError(t, err, "Failed to produce message")

	// Create reader (consumer)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kafkaBootstrapServers},
		Topic:          testTopic,
		GroupID:        fmt.Sprintf("test-group-%d", time.Now().UnixNano()),
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        1 * time.Second,
		StartOffset:    kafka.FirstOffset,
		CommitInterval: 0,
	})
	defer reader.Close()

	// Read message
	readCtx, readCancel := context.WithTimeout(ctx, 10*time.Second)
	defer readCancel()

	msg, err := reader.ReadMessage(readCtx)
	require.NoError(t, err, "Failed to read message")

	assert.Equal(t, testMessage, string(msg.Value), "Message content mismatch")

	t.Log("Kafka connection test passed!")
}

func TestSchemaRegistryConnection(t *testing.T) {
	// Test Schema Registry is reachable
	resp, err := http.Get(schemaRegistryURL + "/subjects")
	require.NoError(t, err, "Failed to connect to Schema Registry")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Schema Registry returned non-OK status")

	// Register a test schema
	testSchema := `{
		"type": "record",
		"name": "InfraTestEvent",
		"namespace": "com.edalab.test",
		"fields": [
			{"name": "id", "type": "string"},
			{"name": "timestamp", "type": "long"}
		]
	}`

	subject := fmt.Sprintf("infra-test-%d-value", time.Now().UnixNano())
	payload := map[string]string{"schema": testSchema}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/subjects/%s/versions", schemaRegistryURL, subject), strings.NewReader(string(payloadBytes)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err = client.Do(req)
	require.NoError(t, err, "Failed to register schema")
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Schema registration failed: %s", string(body))

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	require.NoError(t, err)

	schemaID, ok := result["id"]
	require.True(t, ok, "Schema ID not returned")
	t.Logf("Schema registered with ID: %v", schemaID)

	// Retrieve schema by ID
	resp, err = http.Get(fmt.Sprintf("%s/schemas/ids/%v", schemaRegistryURL, schemaID))
	require.NoError(t, err, "Failed to retrieve schema")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Failed to retrieve schema by ID")

	t.Log("Schema Registry connection test passed!")
}

func TestPostgreSQLConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to PostgreSQL
	conn, err := pgx.Connect(ctx, postgresConnString)
	require.NoError(t, err, "Failed to connect to PostgreSQL")
	defer conn.Close(ctx)

	// Ping the database
	err = conn.Ping(ctx)
	require.NoError(t, err, "Failed to ping PostgreSQL")

	// Query the health check table
	var status string
	err = conn.QueryRow(ctx, "SELECT status FROM bancaire.health_check LIMIT 1").Scan(&status)
	require.NoError(t, err, "Failed to query health_check table")

	assert.Equal(t, "healthy", status, "Unexpected health check status")

	t.Log("PostgreSQL connection test passed!")
}
