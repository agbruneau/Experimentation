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

func main() {
	// Load configuration
	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("Failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Override service name
	cfg.Service.Name = "simulator"
	if cfg.Service.Port == 0 {
		cfg.Service.Port = 8080
	}
	if cfg.Service.MetricsPort == 0 {
		cfg.Service.MetricsPort = 9090
	}

	// Initialize logger
	logger := observability.InitLogger(cfg.Service.Name, cfg.Service.LogLevel)
	logger.Info("Starting simulator service",
		slog.String("kafka", cfg.Kafka.BootstrapServers),
		slog.Int("port", cfg.Service.Port),
	)

	// Register metrics
	observability.RegisterMetrics()

	// Create Kafka producer
	producer, err := kafka.NewAvroProducer(kafka.ProducerConfig{
		BootstrapServers:  cfg.Kafka.BootstrapServers,
		SchemaRegistryURL: cfg.Schema.URL,
		Acks:              "all",
		Retries:           3,
		RetryBackoffMs:    100,
	})
	if err != nil {
		logger.Error("Failed to create Kafka producer", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer producer.Close()

	// Create simulation manager
	manager := simulation.NewManager(producer, logger)

	// Create API handler
	handler := api.NewHandler(manager, logger)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Service.Port),
		Handler:      handler.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start metrics server
	metricsServer := observability.NewMetricsServer(cfg.Service.MetricsPort)

	// Start servers in goroutines
	errChan := make(chan error, 2)

	go func() {
		logger.Info("Starting HTTP server", slog.Int("port", cfg.Service.Port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	go func() {
		logger.Info("Starting metrics server", slog.Int("port", cfg.Service.MetricsPort))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("metrics server error: %w", err)
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop simulation if running
	if manager.IsRunning() {
		manager.Stop()
	}

	// Shutdown servers
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", slog.String("error", err.Error()))
	}
	if err := metricsServer.Shutdown(ctx); err != nil {
		logger.Error("Metrics server shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("Simulator service stopped")
}
