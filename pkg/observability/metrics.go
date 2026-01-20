package observability

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Message counters
	messagesProduced *prometheus.CounterVec
	messagesConsumed *prometheus.CounterVec

	// Processing metrics
	messageProcessingDuration *prometheus.HistogramVec

	// Error counters
	errorsTotal *prometheus.CounterVec

	// Database metrics
	dbQueriesTotal    *prometheus.CounterVec
	dbQueryDuration   *prometheus.HistogramVec
	dbConnectionsOpen *prometheus.GaugeVec

	registerOnce sync.Once
)

// RegisterMetrics registers all Prometheus metrics
func RegisterMetrics() {
	registerOnce.Do(func() {
		messagesProduced = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "edalab_messages_produced_total",
				Help: "Total number of messages produced",
			},
			[]string{"topic"},
		)

		messagesConsumed = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "edalab_messages_consumed_total",
				Help: "Total number of messages consumed",
			},
			[]string{"topic", "consumer_group"},
		)

		messageProcessingDuration = promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "edalab_message_processing_duration_seconds",
				Help:    "Duration of message processing in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"topic", "status"},
		)

		errorsTotal = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "edalab_errors_total",
				Help: "Total number of errors",
			},
			[]string{"component", "error_type"},
		)

		dbQueriesTotal = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "edalab_db_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"operation", "status"},
		)

		dbQueryDuration = promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "edalab_db_query_duration_seconds",
				Help:    "Duration of database queries in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation"},
		)

		dbConnectionsOpen = promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "edalab_db_connections_open",
				Help: "Number of open database connections",
			},
			[]string{"pool"},
		)
	})
}

// MetricsHandler returns the Prometheus metrics HTTP handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// NewMetricsServer creates a new HTTP server for metrics
func NewMetricsServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", MetricsHandler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}

// RecordMessageProduced increments the messages produced counter
func RecordMessageProduced(topic string) {
	if messagesProduced != nil {
		messagesProduced.WithLabelValues(topic).Inc()
	}
}

// RecordMessageConsumed increments the messages consumed counter
func RecordMessageConsumed(topic, consumerGroup string) {
	if messagesConsumed != nil {
		messagesConsumed.WithLabelValues(topic, consumerGroup).Inc()
	}
}

// RecordError increments the error counter
func RecordError(component, errorType string) {
	if errorsTotal != nil {
		errorsTotal.WithLabelValues(component, errorType).Inc()
	}
}

// RecordDBQuery records a database query metric
func RecordDBQuery(operation, status string, duration time.Duration) {
	if dbQueriesTotal != nil {
		dbQueriesTotal.WithLabelValues(operation, status).Inc()
	}
	if dbQueryDuration != nil {
		dbQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
	}
}

// SetDBConnections sets the number of open database connections
func SetDBConnections(pool string, count float64) {
	if dbConnectionsOpen != nil {
		dbConnectionsOpen.WithLabelValues(pool).Set(count)
	}
}

// ProcessingTimer is a helper for recording processing duration
type ProcessingTimer struct {
	start time.Time
}

// StartProcessingTimer starts a new processing timer
func StartProcessingTimer() *ProcessingTimer {
	return &ProcessingTimer{start: time.Now()}
}

// ObserveDuration records the duration since the timer was started
func (t *ProcessingTimer) ObserveDuration(topic, status string) {
	if messageProcessingDuration != nil {
		duration := time.Since(t.start).Seconds()
		messageProcessingDuration.WithLabelValues(topic, status).Observe(duration)
	}
}
