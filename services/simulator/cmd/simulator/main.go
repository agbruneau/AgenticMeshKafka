// Package main is the entry point for the Simulator service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edalab/pkg/config"
	"github.com/edalab/pkg/kafka"
	"github.com/edalab/pkg/observability"
	"github.com/edalab/services/simulator/internal/api"
	"github.com/edalab/services/simulator/internal/simulation"
)

const serviceName = "simulator"

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override service name
	cfg.Service.Name = serviceName

	// Initialize logger
	logger := observability.InitLogger(observability.LoggerConfig{
		Level:       cfg.Logging.Level,
		Format:      cfg.Logging.Format,
		ServiceName: serviceName,
	})
	logger.Info("starting simulator service",
		slog.String("version", "1.0.0"),
		slog.Int("port", cfg.Service.Port),
		slog.Int("metrics_port", cfg.Service.MetricsPort),
	)

	// Initialize tracing
	tracerCfg := observability.TracerConfig{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		Environment:    "development",
		Endpoint:       cfg.Tracing.JaegerEndpoint,
		SampleRate:     cfg.Tracing.SampleRate,
		Enabled:        cfg.Tracing.Enabled,
	}
	tp, err := observability.InitTracer(context.Background(), tracerCfg)
	if err != nil {
		logger.Warn("failed to initialize tracer",
			slog.Any("error", err),
		)
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tp.Shutdown(shutdownCtx); err != nil {
				logger.Error("failed to shutdown tracer",
					slog.Any("error", err),
				)
			}
		}()
	}

	// Initialize metrics
	metrics := observability.NewMetrics("edalab")
	logger.Info("metrics initialized")

	// Create Kafka producer
	producerConfig := kafka.DefaultProducerConfig(
		cfg.Kafka.BootstrapServers,
		cfg.Schema.URL,
	)
	// Use JSON serialization temporarily while debugging Avro issues
	producerConfig.UseJSON = true
	producer, err := kafka.NewAvroProducer(producerConfig)
	if err != nil {
		logger.Error("failed to create Kafka producer",
			slog.Any("error", err),
		)
		os.Exit(1)
	}
	defer producer.Close()
	logger.Info("Kafka producer initialized",
		slog.String("bootstrap_servers", cfg.Kafka.BootstrapServers),
		slog.String("schema_registry", cfg.Schema.URL),
	)

	// Create simulation manager
	simManager := simulation.NewManager(producer, logger, metrics, serviceName)
	logger.Info("simulation manager initialized",
		slog.Any("supported_events", simManager.SupportedEventTypes()),
	)

	// Create API handler
	apiHandler := api.NewHandler(simManager, logger, metrics, serviceName)

	// Create HTTP server
	httpAddr := fmt.Sprintf(":%d", cfg.Service.Port)
	httpServer := &http.Server{
		Addr:         httpAddr,
		Handler:      apiHandler.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Create metrics server
	metricsAddr := fmt.Sprintf(":%d", cfg.Service.MetricsPort)
	metricsServer := observability.NewMetricsServer(metricsAddr)

	// Start servers
	errChan := make(chan error, 2)

	go func() {
		logger.Info("starting HTTP server",
			slog.String("addr", httpAddr),
		)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	go func() {
		logger.Info("starting metrics server",
			slog.String("addr", metricsAddr),
		)
		if err := metricsServer.Start(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("metrics server error: %w", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logger.Error("server error",
			slog.Any("error", err),
		)
	case sig := <-sigChan:
		logger.Info("received shutdown signal",
			slog.String("signal", sig.String()),
		)
	}

	// Graceful shutdown
	logger.Info("initiating graceful shutdown")

	// Stop simulation if running
	if simManager.IsRunning() {
		logger.Info("stopping active simulation")
		if _, err := simManager.Stop(); err != nil {
			logger.Error("failed to stop simulation",
				slog.Any("error", err),
			)
		}
	}

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error",
			slog.Any("error", err),
		)
	}

	// Shutdown metrics server
	if err := metricsServer.Shutdown(10 * time.Second); err != nil {
		logger.Error("metrics server shutdown error",
			slog.Any("error", err),
		)
	}

	// Flush producer
	logger.Info("flushing Kafka producer")
	remaining := producer.Flush(10000)
	if remaining > 0 {
		logger.Warn("some messages were not delivered",
			slog.Int("remaining", remaining),
		)
	}

	logger.Info("simulator service stopped")
}
