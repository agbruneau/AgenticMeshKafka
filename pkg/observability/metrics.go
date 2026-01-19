// Package observability provides metrics, tracing, and logging for EDA-Lab services.
package observability

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for the application.
type Metrics struct {
	// Message metrics
	MessagesProduced *prometheus.CounterVec
	MessagesConsumed *prometheus.CounterVec
	MessagesFailed   *prometheus.CounterVec

	// Latency metrics
	MessageLatency     *prometheus.HistogramVec
	ProcessingLatency  *prometheus.HistogramVec
	DatabaseLatency    *prometheus.HistogramVec

	// Error metrics
	ProcessingErrors *prometheus.CounterVec

	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	// Custom metrics
	ActiveSimulations *prometheus.GaugeVec
	EventsInFlight    *prometheus.GaugeVec
}

var (
	defaultMetrics     *Metrics
	defaultMetricsOnce sync.Once
)

// NewMetrics creates a new Metrics instance with the given namespace.
// Uses sync.Once to ensure metrics are only registered once.
func NewMetrics(namespace string) *Metrics {
	defaultMetricsOnce.Do(func() {
		defaultMetrics = createMetrics(namespace)
	})
	return defaultMetrics
}

// createMetrics actually creates the metrics (called only once).
func createMetrics(namespace string) *Metrics {
	return &Metrics{
		MessagesProduced: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_produced_total",
				Help:      "Total number of messages produced to Kafka",
			},
			[]string{"service", "topic"},
		),

		MessagesConsumed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_consumed_total",
				Help:      "Total number of messages consumed from Kafka",
			},
			[]string{"service", "topic"},
		),

		MessagesFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_failed_total",
				Help:      "Total number of messages that failed to process",
			},
			[]string{"service", "topic", "error_type"},
		),

		MessageLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "message_latency_seconds",
				Help:      "Latency of message delivery in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"service", "topic"},
		),

		ProcessingLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "processing_latency_seconds",
				Help:      "Latency of message processing in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"service", "event_type"},
		),

		DatabaseLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "database_latency_seconds",
				Help:      "Latency of database operations in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			},
			[]string{"service", "operation"},
		),

		ProcessingErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "processing_errors_total",
				Help:      "Total number of processing errors",
			},
			[]string{"service", "error_type"},
		),

		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"service", "method", "endpoint", "status"},
		),

		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "Duration of HTTP requests in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"service", "method", "endpoint"},
		),

		ActiveSimulations: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_simulations",
				Help:      "Number of active simulations",
			},
			[]string{"service"},
		),

		EventsInFlight: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "events_in_flight",
				Help:      "Number of events currently being processed",
			},
			[]string{"service"},
		),
	}
}

// RecordMessageProduced records a produced message.
func (m *Metrics) RecordMessageProduced(service, topic string) {
	m.MessagesProduced.WithLabelValues(service, topic).Inc()
}

// RecordMessageConsumed records a consumed message.
func (m *Metrics) RecordMessageConsumed(service, topic string) {
	m.MessagesConsumed.WithLabelValues(service, topic).Inc()
}

// RecordMessageFailed records a failed message.
func (m *Metrics) RecordMessageFailed(service, topic, errorType string) {
	m.MessagesFailed.WithLabelValues(service, topic, errorType).Inc()
}

// RecordMessageLatency records message latency.
func (m *Metrics) RecordMessageLatency(service, topic string, duration time.Duration) {
	m.MessageLatency.WithLabelValues(service, topic).Observe(duration.Seconds())
}

// RecordProcessingLatency records processing latency.
func (m *Metrics) RecordProcessingLatency(service, eventType string, duration time.Duration) {
	m.ProcessingLatency.WithLabelValues(service, eventType).Observe(duration.Seconds())
}

// RecordDatabaseLatency records database operation latency.
func (m *Metrics) RecordDatabaseLatency(service, operation string, duration time.Duration) {
	m.DatabaseLatency.WithLabelValues(service, operation).Observe(duration.Seconds())
}

// RecordProcessingError records a processing error.
func (m *Metrics) RecordProcessingError(service, errorType string) {
	m.ProcessingErrors.WithLabelValues(service, errorType).Inc()
}

// RecordHTTPRequest records an HTTP request.
func (m *Metrics) RecordHTTPRequest(service, method, endpoint, status string, duration time.Duration) {
	m.HTTPRequestsTotal.WithLabelValues(service, method, endpoint, status).Inc()
	m.HTTPRequestDuration.WithLabelValues(service, method, endpoint).Observe(duration.Seconds())
}

// SetActiveSimulations sets the number of active simulations.
func (m *Metrics) SetActiveSimulations(service string, count float64) {
	m.ActiveSimulations.WithLabelValues(service).Set(count)
}

// SetEventsInFlight sets the number of events in flight.
func (m *Metrics) SetEventsInFlight(service string, count float64) {
	m.EventsInFlight.WithLabelValues(service).Set(count)
}

// Timer is a helper for measuring operation duration.
type Timer struct {
	start time.Time
}

// NewTimer creates a new timer.
func NewTimer() *Timer {
	return &Timer{start: time.Now()}
}

// Elapsed returns the elapsed time.
func (t *Timer) Elapsed() time.Duration {
	return time.Since(t.start)
}

// ObserveProcessing observes processing latency and stops the timer.
func (t *Timer) ObserveProcessing(m *Metrics, service, eventType string) {
	m.RecordProcessingLatency(service, eventType, t.Elapsed())
}

// ObserveDatabase observes database latency and stops the timer.
func (t *Timer) ObserveDatabase(m *Metrics, service, operation string) {
	m.RecordDatabaseLatency(service, operation, t.Elapsed())
}

// MetricsServer creates an HTTP server for Prometheus metrics.
type MetricsServer struct {
	server *http.Server
}

// NewMetricsServer creates a new metrics server on the specified port.
func NewMetricsServer(addr string) *MetricsServer {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return &MetricsServer{
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

// Start starts the metrics server.
func (s *MetricsServer) Start() error {
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the metrics server.
func (s *MetricsServer) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// Handler returns the metrics HTTP handler.
func Handler() http.Handler {
	return promhttp.Handler()
}
