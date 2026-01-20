package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterMetrics_Success(t *testing.T) {
	// Test that RegisterMetrics does not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RegisterMetrics() panicked: %v", r)
		}
	}()

	RegisterMetrics()
}

func TestMetricsServer_Endpoint(t *testing.T) {
	// Ensure metrics are registered first
	RegisterMetrics()

	// Create test server with metrics handler
	handler := MetricsHandler()
	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check that response contains Prometheus metrics
	body := rr.Body.String()
	if !strings.Contains(body, "go_") && !strings.Contains(body, "process_") {
		t.Error("Metrics endpoint should return Prometheus metrics")
	}
}

func TestMessagesProducedCounter(t *testing.T) {
	RegisterMetrics()

	// Test incrementing the counter
	RecordMessageProduced("test-topic")

	// No panic means success for basic functionality
}

func TestMessagesConsumedCounter(t *testing.T) {
	RegisterMetrics()

	RecordMessageConsumed("test-topic", "test-group")
}

func TestMessageProcessingDuration(t *testing.T) {
	RegisterMetrics()

	// Test recording duration
	timer := StartProcessingTimer()
	timer.ObserveDuration("test-topic", "success")
}

func TestErrorsCounter(t *testing.T) {
	RegisterMetrics()

	RecordError("producer", "connection_failed")
}
