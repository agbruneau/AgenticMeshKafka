// Package simulation provides simulation management for EDA-Lab.
package simulation

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/edalab/pkg/events"
	"github.com/edalab/pkg/kafka"
	"github.com/edalab/pkg/observability"
	"github.com/edalab/services/simulator/internal/generator"
	"github.com/google/uuid"
)

// Status represents the current status of a simulation.
type Status string

const (
	StatusStopped Status = "stopped"
	StatusRunning Status = "running"
	StatusPaused  Status = "paused"
)

// SimulationStatus holds the current state of a simulation.
type SimulationStatus struct {
	ID             string    `json:"id"`
	Status         Status    `json:"status"`
	Scenario       string    `json:"scenario"`
	EventsProduced int64     `json:"events_produced"`
	EventsFailed   int64     `json:"events_failed"`
	StartedAt      time.Time `json:"started_at,omitempty"`
	StoppedAt      time.Time `json:"stopped_at,omitempty"`
	Duration       float64   `json:"duration_seconds"`
	RateRequested  int       `json:"rate_requested"`
	RateActual     float64   `json:"rate_actual"`
	LastEventAt    time.Time `json:"last_event_at,omitempty"`
}

// Config holds configuration for a simulation.
type Config struct {
	Scenario   string        `json:"scenario"`
	Rate       int           `json:"rate"`       // Events per second
	Duration   time.Duration `json:"duration"`   // 0 = infinite
	EventTypes []string      `json:"event_types"` // Event types to generate
}

// DefaultConfig returns a default simulation configuration.
func DefaultConfig() Config {
	return Config{
		Scenario:   "default",
		Rate:       10,
		Duration:   0,
		EventTypes: []string{"CompteOuvert"},
	}
}

// Manager orchestrates event generation simulations.
type Manager struct {
	producer   kafka.Producer
	logger     *slog.Logger
	metrics    *observability.Metrics
	service    string
	generators map[string]generator.EventGenerator
	factory    *generator.GeneratorFactory

	// State
	mu       sync.RWMutex
	status   *SimulationStatus
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	produced atomic.Int64
	failed   atomic.Int64
}

// NewManager creates a new simulation manager.
func NewManager(
	producer kafka.Producer,
	logger *slog.Logger,
	metrics *observability.Metrics,
	service string,
) *Manager {
	factory := generator.NewGeneratorFactory(producer, logger, metrics, service, time.Now().UnixNano())

	return &Manager{
		producer:   producer,
		logger:     logger,
		metrics:    metrics,
		service:    service,
		generators: factory.CreateAll(),
		factory:    factory,
		status: &SimulationStatus{
			Status: StatusStopped,
		},
	}
}

// Start begins a new simulation.
func (m *Manager) Start(ctx context.Context, config Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already running
	if m.status.Status == StatusRunning {
		return fmt.Errorf("simulation already running with ID: %s", m.status.ID)
	}

	// Validate config
	if config.Rate <= 0 {
		config.Rate = 10
	}
	if len(config.EventTypes) == 0 {
		config.EventTypes = []string{"CompteOuvert"}
	}

	// Create simulation context with background (not tied to HTTP request)
	simCtx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	// Initialize status
	m.status = &SimulationStatus{
		ID:            uuid.New().String(),
		Status:        StatusRunning,
		Scenario:      config.Scenario,
		StartedAt:     time.Now(),
		RateRequested: config.Rate,
	}
	m.produced.Store(0)
	m.failed.Store(0)

	// Update metrics
	if m.metrics != nil {
		m.metrics.SetActiveSimulations(m.service, 1)
	}

	// Start generation goroutine
	m.wg.Add(1)
	go m.runSimulation(simCtx, config)

	m.logger.Info("simulation started",
		slog.String("simulation_id", m.status.ID),
		slog.String("scenario", config.Scenario),
		slog.Int("rate", config.Rate),
		slog.Duration("duration", config.Duration),
		slog.Any("event_types", config.EventTypes),
	)

	return nil
}

// runSimulation is the main simulation loop.
func (m *Manager) runSimulation(ctx context.Context, config Config) {
	defer m.wg.Done()
	defer func() {
		m.mu.Lock()
		m.status.Status = StatusStopped
		m.status.StoppedAt = time.Now()
		m.status.Duration = m.status.StoppedAt.Sub(m.status.StartedAt).Seconds()
		m.status.EventsProduced = m.produced.Load()
		m.status.EventsFailed = m.failed.Load()
		if m.metrics != nil {
			m.metrics.SetActiveSimulations(m.service, 0)
		}
		m.mu.Unlock()
	}()

	// Calculate interval between events
	interval := time.Second / time.Duration(config.Rate)

	// Create ticker for rate control
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Create duration timer if specified
	var durationTimer <-chan time.Time
	if config.Duration > 0 {
		durationTimer = time.After(config.Duration)
	}

	// Event type index for round-robin
	eventTypeIndex := 0

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("simulation stopped by context")
			return

		case <-durationTimer:
			m.logger.Info("simulation completed (duration reached)",
				slog.Int64("events_produced", m.produced.Load()),
			)
			return

		case <-ticker.C:
			// Select event type (round-robin)
			eventType := config.EventTypes[eventTypeIndex]
			eventTypeIndex = (eventTypeIndex + 1) % len(config.EventTypes)

			// Get generator
			gen, ok := m.generators[eventType]
			if !ok {
				m.logger.Warn("unknown event type",
					slog.String("event_type", eventType),
				)
				continue
			}

			// Generate event
			event, err := gen.Generate(ctx)
			if err != nil {
				m.failed.Add(1)
				m.logger.Error("failed to generate event",
					slog.String("event_type", eventType),
					slog.Any("error", err),
				)
				continue
			}

			m.produced.Add(1)

			// Update last event time
			m.mu.Lock()
			m.status.LastEventAt = time.Now()
			m.status.EventsProduced = m.produced.Load()
			m.status.EventsFailed = m.failed.Load()
			// Calculate actual rate
			elapsed := time.Since(m.status.StartedAt).Seconds()
			if elapsed > 0 {
				m.status.RateActual = float64(m.produced.Load()) / elapsed
			}
			m.mu.Unlock()

			_ = event // Event is already logged by generator
		}
	}
}

// Stop stops the current simulation.
func (m *Manager) Stop() (*SimulationStatus, error) {
	m.mu.Lock()
	if m.status.Status != StatusRunning {
		m.mu.Unlock()
		return nil, fmt.Errorf("no simulation is running")
	}

	// Cancel context
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Unlock()

	// Wait for goroutine to finish
	m.wg.Wait()

	m.logger.Info("simulation stopped",
		slog.String("simulation_id", m.status.ID),
		slog.Int64("events_produced", m.status.EventsProduced),
		slog.Float64("duration_seconds", m.status.Duration),
	)

	return m.Status(), nil
}

// Status returns the current simulation status.
func (m *Manager) Status() *SimulationStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy
	status := *m.status

	// Update dynamic fields if running
	if status.Status == StatusRunning {
		status.EventsProduced = m.produced.Load()
		status.EventsFailed = m.failed.Load()
		status.Duration = time.Since(status.StartedAt).Seconds()
		if status.Duration > 0 {
			status.RateActual = float64(status.EventsProduced) / status.Duration
		}
	}

	return &status
}

// IsRunning returns true if a simulation is currently running.
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status.Status == StatusRunning
}

// ProduceEvents produces events on demand (outside of simulation).
func (m *Manager) ProduceEvents(ctx context.Context, eventType string, count int) ([]string, error) {
	gen, ok := m.generators[eventType]
	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	eventIDs := make([]string, 0, count)

	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			return eventIDs, ctx.Err()
		default:
		}

		event, err := gen.Generate(ctx)
		if err != nil {
			m.logger.Error("failed to produce event",
				slog.String("event_type", eventType),
				slog.Int("index", i),
				slog.Any("error", err),
			)
			continue
		}

		// Extract event ID
		switch e := event.(type) {
		case *events.CompteOuvert:
			eventIDs = append(eventIDs, e.EventID)
		case *events.DepotEffectue:
			eventIDs = append(eventIDs, e.EventID)
		case *events.VirementEmis:
			eventIDs = append(eventIDs, e.EventID)
		}
	}

	m.logger.Info("events produced on demand",
		slog.String("event_type", eventType),
		slog.Int("requested", count),
		slog.Int("produced", len(eventIDs)),
	)

	return eventIDs, nil
}

// SupportedEventTypes returns the list of supported event types.
func (m *Manager) SupportedEventTypes() []string {
	types := make([]string, 0, len(m.generators))
	for t := range m.generators {
		types = append(types, t)
	}
	return types
}
