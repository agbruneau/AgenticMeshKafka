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
	"github.com/edalab/pkg/database"
	"github.com/edalab/pkg/events"
	"github.com/edalab/pkg/kafka"
	"github.com/edalab/pkg/observability"
	"github.com/edalab/services/bancaire/internal/api"
	"github.com/edalab/services/bancaire/internal/handler"
	"github.com/edalab/services/bancaire/internal/repository"
)

func main() {
	// Load configuration
	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("Failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Override service name
	cfg.Service.Name = "bancaire"
	if cfg.Service.Port == 0 {
		cfg.Service.Port = 8083
	}
	if cfg.Service.MetricsPort == 0 {
		cfg.Service.MetricsPort = 9083
	}
	if cfg.Kafka.GroupID == "" {
		cfg.Kafka.GroupID = "bancaire-group"
	}

	// Initialize logger
	logger := observability.InitLogger(cfg.Service.Name, cfg.Service.LogLevel)
	logger.Info("Starting bancaire service",
		slog.String("kafka", cfg.Kafka.BootstrapServers),
		slog.String("postgres", cfg.Postgres.Host),
		slog.Int("port", cfg.Service.Port),
	)

	// Register metrics
	observability.RegisterMetrics()

	// Create database pool
	ctx := context.Background()
	dbPool, err := database.NewDBPool(ctx, database.Config{
		Host:     cfg.Postgres.Host,
		Port:     cfg.Postgres.Port,
		Database: cfg.Postgres.Database,
		User:     cfg.Postgres.User,
		Password: cfg.Postgres.Password,
		MaxConns: 10,
		MinConns: 2,
	})
	if err != nil {
		logger.Error("Failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbPool.Close()
	logger.Info("Connected to PostgreSQL")

	// Create repository
	repo := repository.NewPostgresCompteRepository(dbPool)

	// Create Kafka consumer
	consumer, err := kafka.NewAvroConsumer(kafka.ConsumerConfig{
		BootstrapServers:  cfg.Kafka.BootstrapServers,
		SchemaRegistryURL: cfg.Schema.URL,
		GroupID:           cfg.Kafka.GroupID,
		AutoOffsetReset:   cfg.Kafka.AutoOffsetReset,
		EnableAutoCommit:  false,
	})
	if err != nil {
		logger.Error("Failed to create Kafka consumer", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer consumer.Close()

	// Subscribe to topics
	topics := []string{
		events.TopicCompteOuvert,
		events.TopicDepotEffectue,
		events.TopicRetraitEffectue,
		events.TopicVirementEmis,
	}
	if err := consumer.Subscribe(topics); err != nil {
		logger.Error("Failed to subscribe to topics", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("Subscribed to Kafka topics", slog.Any("topics", topics))

	// Create event handler
	eventHandler := handler.NewEventHandler(repo, logger)

	// Create API handler
	apiHandler := api.NewHandler(repo, logger)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Service.Port),
		Handler:      apiHandler.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start metrics server
	metricsServer := observability.NewMetricsServer(cfg.Service.MetricsPort)

	// Create context for shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Error channel
	errChan := make(chan error, 3)

	// Start HTTP server
	go func() {
		logger.Info("Starting HTTP server", slog.Int("port", cfg.Service.Port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Start metrics server
	go func() {
		logger.Info("Starting metrics server", slog.Int("port", cfg.Service.MetricsPort))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("metrics server error: %w", err)
		}
	}()

	// Start Kafka consumer
	go func() {
		logger.Info("Starting Kafka consumer")
		if err := consumer.Consume(ctx, eventHandler.Route); err != nil && err != context.Canceled {
			errChan <- fmt.Errorf("Kafka consumer error: %w", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info("Shutting down...")
	case err := <-errChan:
		logger.Error("Server error", slog.String("error", err.Error()))
	}

	// Graceful shutdown
	cancel() // Cancel context to stop consumer

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown servers
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", slog.String("error", err.Error()))
	}
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Metrics server shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("Bancaire service stopped")
}
