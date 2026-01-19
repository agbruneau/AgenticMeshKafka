// Package main is the entry point for the Bancaire service.
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
	"github.com/edalab/pkg/observability"
	"github.com/edalab/services/bancaire/internal/api"
	"github.com/edalab/services/bancaire/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

const serviceName = "bancaire"

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
	logger.Info("starting bancaire service",
		slog.String("version", "1.0.0"),
		slog.Int("port", cfg.Service.Port),
		slog.Int("metrics_port", cfg.Service.MetricsPort),
	)

	// Initialize tracing (if enabled)
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
		logger.Warn("failed to initialize tracer", slog.Any("error", err))
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			tp.Shutdown(shutdownCtx)
		}()
	}

	// Initialize metrics
	metrics := observability.NewMetrics("edalab")
	logger.Info("metrics initialized")

	// Create database pool
	dbConnStr := cfg.Postgres.ConnectionString()
	pool, err := pgxpool.New(context.Background(), dbConnStr)
	if err != nil {
		logger.Error("failed to create database pool", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	// Test database connection
	if err := pool.Ping(context.Background()); err != nil {
		logger.Error("failed to ping database", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("database connection established",
		slog.String("host", cfg.Postgres.Host),
		slog.String("database", cfg.Postgres.Database),
	)

	// Create repository
	repo := repository.NewPostgresCompteRepository(pool)
	logger.Info("repository initialized")

	// Create API handler
	apiHandler := api.NewHandler(repo, logger, metrics, serviceName)

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
		logger.Info("starting HTTP server", slog.String("addr", httpAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	go func() {
		logger.Info("starting metrics server", slog.String("addr", metricsAddr))
		if err := metricsServer.Start(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("metrics server error: %w", err)
		}
	}()

	// Note: Kafka consumer would be added here in a full implementation
	// For MVP, events can be processed via HTTP endpoints or a separate consumer goroutine
	logger.Info("bancaire service ready",
		slog.String("http_addr", httpAddr),
		slog.String("metrics_addr", metricsAddr),
	)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logger.Error("server error", slog.Any("error", err))
	case sig := <-sigChan:
		logger.Info("received shutdown signal", slog.String("signal", sig.String()))
	}

	// Graceful shutdown
	logger.Info("initiating graceful shutdown")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", slog.Any("error", err))
	}

	if err := metricsServer.Shutdown(10 * time.Second); err != nil {
		logger.Error("metrics server shutdown error", slog.Any("error", err))
	}

	logger.Info("bancaire service stopped")
}
