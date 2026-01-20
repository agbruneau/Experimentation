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
	"github.com/edalab/pkg/events"
	"github.com/edalab/pkg/kafka"
	"github.com/edalab/pkg/observability"
	"github.com/edalab/services/gateway/internal/api"
	"github.com/edalab/services/gateway/internal/proxy"
	"github.com/edalab/services/gateway/internal/streaming"
	"github.com/edalab/services/gateway/internal/websocket"
)

func main() {
	// Load configuration
	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("Failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Override service name and ports
	cfg.Service.Name = "gateway"
	if cfg.Service.Port == 0 {
		cfg.Service.Port = 8082
	}
	if cfg.Service.MetricsPort == 0 {
		cfg.Service.MetricsPort = 9092
	}
	if cfg.Kafka.GroupID == "" {
		cfg.Kafka.GroupID = "gateway-group"
	}

	// Get backend URLs from environment
	simulatorURL := getEnv("SIMULATOR_URL", "http://localhost:8080")
	bancaireURL := getEnv("BANCAIRE_URL", "http://localhost:8081")

	// Initialize logger
	logger := observability.InitLogger(cfg.Service.Name, cfg.Service.LogLevel)
	logger.Info("Starting gateway service",
		slog.String("kafka", cfg.Kafka.BootstrapServers),
		slog.Int("port", cfg.Service.Port),
		slog.String("simulator", simulatorURL),
		slog.String("bancaire", bancaireURL),
	)

	// Register metrics
	observability.RegisterMetrics()

	// Create WebSocket hub
	hub := websocket.NewHub(logger)
	go hub.Run()

	// Create service proxy
	serviceProxy := proxy.NewServiceProxy(simulatorURL, bancaireURL, logger)

	// Create Kafka consumer for streaming
	consumer, err := kafka.NewAvroConsumer(kafka.ConsumerConfig{
		BootstrapServers:  cfg.Kafka.BootstrapServers,
		SchemaRegistryURL: cfg.Schema.URL,
		GroupID:           cfg.Kafka.GroupID,
		AutoOffsetReset:   cfg.Kafka.AutoOffsetReset,
		EnableAutoCommit:  true,
	})
	if err != nil {
		logger.Error("Failed to create Kafka consumer", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer consumer.Close()

	// Create Kafka streamer
	topics := []string{
		events.TopicCompteOuvert,
		events.TopicDepotEffectue,
		events.TopicRetraitEffectue,
		events.TopicVirementEmis,
	}
	streamer := streaming.NewKafkaStreamer(consumer, hub, topics, logger)

	// Create router
	router := api.NewRouter(serviceProxy, hub, logger)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Service.Port),
		Handler:      router.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  120 * time.Second, // Longer for WebSocket
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

	// Start Kafka streamer
	go func() {
		logger.Info("Starting Kafka streamer")
		if err := streamer.Start(ctx); err != nil && err != context.Canceled {
			errChan <- fmt.Errorf("Kafka streamer error: %w", err)
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
	cancel() // Cancel context to stop streamer

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown servers
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", slog.String("error", err.Error()))
	}
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Metrics server shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("Gateway service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
