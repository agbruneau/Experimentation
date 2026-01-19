package simulation

import (
	"context"
	"testing"
	"time"
)

// MockProducer implements kafka.Producer for testing
type MockProducer struct {
	messages []interface{}
}

func (m *MockProducer) Produce(ctx context.Context, topic string, key string, value interface{}) error {
	m.messages = append(m.messages, value)
	return nil
}

func (m *MockProducer) ProduceWithHeaders(ctx context.Context, topic string, key string, value interface{}, headers map[string]string) error {
	m.messages = append(m.messages, value)
	return nil
}

func (m *MockProducer) ProduceRaw(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error {
	m.messages = append(m.messages, value)
	return nil
}

func (m *MockProducer) Flush(timeoutMs int) int {
	return 0
}

func (m *MockProducer) Close() {}

func TestNewManager(t *testing.T) {
	producer := &MockProducer{}
	manager := NewManager(producer, nil, nil, "test-service")

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}

	if manager.IsRunning() {
		t.Error("expected simulation not running initially")
	}

	status := manager.Status()
	if status.Status != StatusStopped {
		t.Errorf("expected status 'stopped', got '%s'", status.Status)
	}
}

func TestManager_Start_Success(t *testing.T) {
	producer := &MockProducer{}
	manager := NewManager(producer, nil, nil, "test-service")

	config := Config{
		Scenario:   "test",
		Rate:       10,
		Duration:   1 * time.Second,
		EventTypes: []string{"CompteOuvert"},
	}

	err := manager.Start(context.Background(), config)
	if err != nil {
		t.Fatalf("failed to start simulation: %v", err)
	}

	if !manager.IsRunning() {
		t.Error("expected simulation to be running")
	}

	status := manager.Status()
	if status.Status != StatusRunning {
		t.Errorf("expected status 'running', got '%s'", status.Status)
	}
	if status.Scenario != "test" {
		t.Errorf("expected scenario 'test', got '%s'", status.Scenario)
	}
	if status.RateRequested != 10 {
		t.Errorf("expected rate 10, got %d", status.RateRequested)
	}

	// Wait for simulation to complete (duration is 1 second)
	time.Sleep(2 * time.Second)

	if manager.IsRunning() {
		t.Error("expected simulation to be stopped after duration")
	}

	// Check that some events were produced
	if len(producer.messages) == 0 {
		t.Error("expected some messages to be produced")
	}
}

func TestManager_Start_AlreadyRunning(t *testing.T) {
	producer := &MockProducer{}
	manager := NewManager(producer, nil, nil, "test-service")

	config := Config{
		Rate:     1,
		Duration: 0, // infinite
	}

	// Start first simulation
	err := manager.Start(context.Background(), config)
	if err != nil {
		t.Fatalf("failed to start first simulation: %v", err)
	}

	// Try to start second simulation
	err = manager.Start(context.Background(), config)
	if err == nil {
		t.Error("expected error when starting second simulation")
	}

	// Clean up
	manager.Stop()
}

func TestManager_Stop_Success(t *testing.T) {
	producer := &MockProducer{}
	manager := NewManager(producer, nil, nil, "test-service")

	config := Config{
		Rate:     5,
		Duration: 0, // infinite
	}

	err := manager.Start(context.Background(), config)
	if err != nil {
		t.Fatalf("failed to start simulation: %v", err)
	}

	// Wait a bit
	time.Sleep(500 * time.Millisecond)

	status, err := manager.Stop()
	if err != nil {
		t.Fatalf("failed to stop simulation: %v", err)
	}

	if status.Status != StatusStopped {
		t.Errorf("expected status 'stopped', got '%s'", status.Status)
	}
	if status.EventsProduced == 0 {
		t.Error("expected some events to be produced")
	}
	if status.Duration == 0 {
		t.Error("expected non-zero duration")
	}

	if manager.IsRunning() {
		t.Error("expected simulation not running after stop")
	}
}

func TestManager_Stop_NotRunning(t *testing.T) {
	producer := &MockProducer{}
	manager := NewManager(producer, nil, nil, "test-service")

	_, err := manager.Stop()
	if err == nil {
		t.Error("expected error when stopping non-running simulation")
	}
}

func TestManager_AutoStopAfterDuration(t *testing.T) {
	producer := &MockProducer{}
	manager := NewManager(producer, nil, nil, "test-service")

	config := Config{
		Rate:     20,
		Duration: 500 * time.Millisecond,
	}

	err := manager.Start(context.Background(), config)
	if err != nil {
		t.Fatalf("failed to start simulation: %v", err)
	}

	// Should be running initially
	if !manager.IsRunning() {
		t.Error("expected simulation to be running")
	}

	// Wait for duration + buffer
	time.Sleep(1 * time.Second)

	// Should be stopped
	if manager.IsRunning() {
		t.Error("expected simulation to be stopped after duration")
	}

	status := manager.Status()
	if status.Status != StatusStopped {
		t.Errorf("expected status 'stopped', got '%s'", status.Status)
	}

	// Should have produced approximately 10 events (20/sec * 0.5 sec)
	// Allow some variance due to timing
	if status.EventsProduced < 5 || status.EventsProduced > 20 {
		t.Errorf("expected ~10 events, got %d", status.EventsProduced)
	}
}

func TestManager_RateControl(t *testing.T) {
	producer := &MockProducer{}
	manager := NewManager(producer, nil, nil, "test-service")

	config := Config{
		Rate:     10, // 10 events per second
		Duration: 2 * time.Second,
	}

	err := manager.Start(context.Background(), config)
	if err != nil {
		t.Fatalf("failed to start simulation: %v", err)
	}

	// Wait for completion
	time.Sleep(3 * time.Second)

	status := manager.Status()

	// Should have produced approximately 20 events (10/sec * 2 sec)
	// Allow some variance
	if status.EventsProduced < 15 || status.EventsProduced > 25 {
		t.Errorf("expected ~20 events, got %d", status.EventsProduced)
	}

	// Check actual rate
	if status.RateActual < 8 || status.RateActual > 12 {
		t.Errorf("expected rate ~10, got %.2f", status.RateActual)
	}
}

func TestManager_ProduceEvents(t *testing.T) {
	producer := &MockProducer{}
	manager := NewManager(producer, nil, nil, "test-service")

	eventIDs, err := manager.ProduceEvents(context.Background(), "CompteOuvert", 5)
	if err != nil {
		t.Fatalf("failed to produce events: %v", err)
	}

	if len(eventIDs) != 5 {
		t.Errorf("expected 5 event IDs, got %d", len(eventIDs))
	}

	if len(producer.messages) != 5 {
		t.Errorf("expected 5 messages in producer, got %d", len(producer.messages))
	}
}

func TestManager_ProduceEvents_UnknownType(t *testing.T) {
	producer := &MockProducer{}
	manager := NewManager(producer, nil, nil, "test-service")

	_, err := manager.ProduceEvents(context.Background(), "UnknownEvent", 1)
	if err == nil {
		t.Error("expected error for unknown event type")
	}
}

func TestManager_SupportedEventTypes(t *testing.T) {
	producer := &MockProducer{}
	manager := NewManager(producer, nil, nil, "test-service")

	types := manager.SupportedEventTypes()

	if len(types) == 0 {
		t.Error("expected non-empty event types list")
	}

	// Check that expected types are present
	hasCompteOuvert := false
	hasDepotEffectue := false
	hasVirementEmis := false

	for _, t := range types {
		switch t {
		case "CompteOuvert":
			hasCompteOuvert = true
		case "DepotEffectue":
			hasDepotEffectue = true
		case "VirementEmis":
			hasVirementEmis = true
		}
	}

	if !hasCompteOuvert {
		t.Error("expected CompteOuvert in supported types")
	}
	if !hasDepotEffectue {
		t.Error("expected DepotEffectue in supported types")
	}
	if !hasVirementEmis {
		t.Error("expected VirementEmis in supported types")
	}
}

func TestManager_ContextCancellation(t *testing.T) {
	producer := &MockProducer{}
	manager := NewManager(producer, nil, nil, "test-service")

	ctx, cancel := context.WithCancel(context.Background())

	config := Config{
		Rate:     10,
		Duration: 0, // infinite
	}

	err := manager.Start(ctx, config)
	if err != nil {
		t.Fatalf("failed to start simulation: %v", err)
	}

	// Wait a bit
	time.Sleep(200 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for simulation to stop
	time.Sleep(200 * time.Millisecond)

	if manager.IsRunning() {
		t.Error("expected simulation to stop on context cancellation")
	}
}
