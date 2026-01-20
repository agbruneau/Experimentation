package simulation

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/edalab/pkg/kafka"
	"github.com/edalab/pkg/observability"
	"github.com/edalab/services/simulator/internal/generator"
	"github.com/google/uuid"
)

// Status represents the simulation status
type Status struct {
	ID             string    `json:"id"`
	State          string    `json:"state"` // "running", "stopped"
	EventsProduced int64     `json:"events_produced"`
	StartedAt      time.Time `json:"started_at,omitempty"`
	StoppedAt      time.Time `json:"stopped_at,omitempty"`
	Rate           int       `json:"rate"`
	Duration       int       `json:"duration"`
	ActualRate     float64   `json:"actual_rate"`
}

// Config holds simulation configuration
type Config struct {
	Scenario string `json:"scenario"`
	Rate     int    `json:"rate"`     // events per second
	Duration int    `json:"duration"` // seconds, 0 = infinite
}

// Manager orchestrates event generation
type Manager struct {
	producer  kafka.Producer
	generator *generator.EventGenerator
	logger    *slog.Logger

	status   atomic.Value // *Status
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

// NewManager creates a new simulation manager
func NewManager(producer kafka.Producer, logger *slog.Logger) *Manager {
	m := &Manager{
		producer:  producer,
		generator: generator.NewEventGenerator(producer),
		logger:    logger,
	}

	// Initialize status
	m.status.Store(&Status{
		State: "stopped",
	})

	return m
}

// Start begins the simulation
func (m *Manager) Start(ctx context.Context, config Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("simulation already running")
	}

	// Set defaults
	if config.Rate <= 0 {
		config.Rate = 10
	}

	// Create cancellable context
	simCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	// Initialize status
	status := &Status{
		ID:        uuid.New().String(),
		State:     "running",
		StartedAt: time.Now(),
		Rate:      config.Rate,
		Duration:  config.Duration,
	}
	m.status.Store(status)
	m.running = true

	// Update metrics
	observability.SimulationStatus.WithLabelValues("simulator").Set(1)

	m.logger.Info("Starting simulation",
		slog.String("simulation_id", status.ID),
		slog.Int("rate", config.Rate),
		slog.Int("duration", config.Duration),
	)

	// Start generation goroutine
	m.wg.Add(1)
	go m.runSimulation(simCtx, config, status)

	return nil
}

// Stop stops the simulation
func (m *Manager) Stop() (*Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil, fmt.Errorf("simulation not running")
	}

	// Cancel the context
	if m.cancel != nil {
		m.cancel()
	}

	// Wait for goroutine to finish
	m.wg.Wait()

	// Update status
	status := m.status.Load().(*Status)
	status.State = "stopped"
	status.StoppedAt = time.Now()
	m.status.Store(status)
	m.running = false

	// Update metrics
	observability.SimulationStatus.WithLabelValues("simulator").Set(0)

	m.logger.Info("Simulation stopped",
		slog.String("simulation_id", status.ID),
		slog.Int64("events_produced", status.EventsProduced),
	)

	return status, nil
}

// Status returns the current simulation status
func (m *Manager) Status() *Status {
	return m.status.Load().(*Status)
}

// IsRunning returns whether a simulation is currently running
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// runSimulation runs the event generation loop
func (m *Manager) runSimulation(ctx context.Context, config Config, status *Status) {
	defer m.wg.Done()
	defer func() {
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()

	// Calculate interval between events
	interval := time.Second / time.Duration(config.Rate)

	// Set up duration timer if specified
	var durationTimer <-chan time.Time
	if config.Duration > 0 {
		durationTimer = time.After(time.Duration(config.Duration) * time.Second)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	startTime := time.Now()
	var eventsProduced int64

	for {
		select {
		case <-ctx.Done():
			return

		case <-durationTimer:
			m.logger.Info("Simulation duration reached, stopping")
			return

		case <-ticker.C:
			// Generate random event
			_, err := m.generator.GenerateRandomEvent(ctx)
			if err != nil {
				m.logger.Error("Failed to generate event", slog.String("error", err.Error()))
				observability.ProcessingErrors.WithLabelValues("simulator", "generation_error").Inc()
				continue
			}

			// Update counters
			eventsProduced++
			atomic.AddInt64(&status.EventsProduced, 1)

			// Update actual rate
			elapsed := time.Since(startTime).Seconds()
			if elapsed > 0 {
				status.ActualRate = float64(eventsProduced) / elapsed
			}

			// Update metrics
			observability.EventsGenerated.WithLabelValues("random").Inc()
		}
	}
}

// ProduceEvent produces a single event of the specified type
func (m *Manager) ProduceEvent(ctx context.Context, eventType string, count int) ([]string, error) {
	eventIDs := make([]string, 0, count)

	for i := 0; i < count; i++ {
		var eventID string
		var err error

		switch eventType {
		case "CompteOuvert":
			event, genErr := m.generator.GenerateCompteOuvert(ctx)
			if genErr != nil {
				err = genErr
			} else {
				eventID = event.EventID
			}
		case "DepotEffectue":
			event, genErr := m.generator.GenerateDepotEffectue(ctx, "")
			if genErr != nil {
				err = genErr
			} else {
				eventID = event.EventID
			}
		case "RetraitEffectue":
			event, genErr := m.generator.GenerateRetraitEffectue(ctx, "")
			if genErr != nil {
				err = genErr
			} else {
				eventID = event.EventID
			}
		case "VirementEmis":
			event, genErr := m.generator.GenerateVirementEmis(ctx, "", "")
			if genErr != nil {
				err = genErr
			} else {
				eventID = event.EventID
			}
		default:
			return eventIDs, fmt.Errorf("unknown event type: %s", eventType)
		}

		if err != nil {
			return eventIDs, err
		}

		eventIDs = append(eventIDs, eventID)
		observability.EventsGenerated.WithLabelValues(eventType).Inc()
	}

	return eventIDs, nil
}
