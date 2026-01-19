package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/edalab/services/simulator/internal/simulation"
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

func newTestHandler() (*Handler, *MockProducer) {
	producer := &MockProducer{}
	manager := simulation.NewManager(producer, nil, nil, "test-simulator")
	handler := NewHandler(manager, nil, nil, "test-simulator")
	return handler, producer
}

func TestHandleHealth(t *testing.T) {
	handler, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", resp.Status)
	}
	if resp.Service != "test-simulator" {
		t.Errorf("expected service 'test-simulator', got '%s'", resp.Service)
	}
	if resp.Simulation != "idle" {
		t.Errorf("expected simulation 'idle', got '%s'", resp.Simulation)
	}
}

func TestHandleStartSimulation_Success(t *testing.T) {
	handler, _ := newTestHandler()

	body := StartRequest{
		Scenario: "test",
		Rate:     5,
		Duration: 1,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation/start", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp StartResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "running" {
		t.Errorf("expected status 'running', got '%s'", resp.Status)
	}
	if resp.SimulationID == "" {
		t.Error("expected non-empty simulation ID")
	}

	// Wait for simulation to stop (duration is 1 second)
	time.Sleep(2 * time.Second)
}

func TestHandleStartSimulation_AlreadyRunning(t *testing.T) {
	handler, _ := newTestHandler()

	// Start first simulation (no duration = infinite)
	body := StartRequest{
		Scenario: "test",
		Rate:     1,
		Duration: 0,
	}
	bodyBytes, _ := json.Marshal(body)

	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/simulation/start", bytes.NewReader(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	handler.Router().ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first start failed: %s", rec1.Body.String())
	}

	// Try to start second simulation
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/simulation/start", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handler.Router().ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, rec2.Code)
	}

	// Stop the simulation
	stopReq := httptest.NewRequest(http.MethodPost, "/api/v1/simulation/stop", nil)
	stopRec := httptest.NewRecorder()
	handler.Router().ServeHTTP(stopRec, stopReq)
}

func TestHandleStopSimulation_Success(t *testing.T) {
	handler, _ := newTestHandler()

	// Start simulation
	startBody := StartRequest{Rate: 5, Duration: 0}
	bodyBytes, _ := json.Marshal(startBody)
	startReq := httptest.NewRequest(http.MethodPost, "/api/v1/simulation/start", bytes.NewReader(bodyBytes))
	startReq.Header.Set("Content-Type", "application/json")
	startRec := httptest.NewRecorder()
	handler.Router().ServeHTTP(startRec, startReq)

	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %s", startRec.Body.String())
	}

	// Wait a bit for some events to be produced
	time.Sleep(500 * time.Millisecond)

	// Stop simulation
	stopReq := httptest.NewRequest(http.MethodPost, "/api/v1/simulation/stop", nil)
	stopRec := httptest.NewRecorder()
	handler.Router().ServeHTTP(stopRec, stopReq)

	if stopRec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, stopRec.Code, stopRec.Body.String())
	}

	var resp StopResponse
	if err := json.NewDecoder(stopRec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "stopped" {
		t.Errorf("expected status 'stopped', got '%s'", resp.Status)
	}
}

func TestHandleStopSimulation_NotRunning(t *testing.T) {
	handler, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation/stop", nil)
	rec := httptest.NewRecorder()
	handler.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestHandleGetStatus_Running(t *testing.T) {
	handler, _ := newTestHandler()

	// Start simulation
	startBody := StartRequest{Rate: 5, Duration: 0, Scenario: "test-scenario"}
	bodyBytes, _ := json.Marshal(startBody)
	startReq := httptest.NewRequest(http.MethodPost, "/api/v1/simulation/start", bytes.NewReader(bodyBytes))
	startReq.Header.Set("Content-Type", "application/json")
	startRec := httptest.NewRecorder()
	handler.Router().ServeHTTP(startRec, startReq)

	// Get status
	req := httptest.NewRequest(http.MethodGet, "/api/v1/simulation/status", nil)
	rec := httptest.NewRecorder()
	handler.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var status simulation.SimulationStatus
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if status.Status != simulation.StatusRunning {
		t.Errorf("expected status 'running', got '%s'", status.Status)
	}
	if status.Scenario != "test-scenario" {
		t.Errorf("expected scenario 'test-scenario', got '%s'", status.Scenario)
	}
	if status.RateRequested != 5 {
		t.Errorf("expected rate 5, got %d", status.RateRequested)
	}

	// Stop simulation
	stopReq := httptest.NewRequest(http.MethodPost, "/api/v1/simulation/stop", nil)
	stopRec := httptest.NewRecorder()
	handler.Router().ServeHTTP(stopRec, stopReq)
}

func TestHandleGetStatus_Stopped(t *testing.T) {
	handler, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/simulation/status", nil)
	rec := httptest.NewRecorder()
	handler.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var status simulation.SimulationStatus
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if status.Status != simulation.StatusStopped {
		t.Errorf("expected status 'stopped', got '%s'", status.Status)
	}
}

func TestHandleProduceEvents_Success(t *testing.T) {
	handler, producer := newTestHandler()

	body := ProduceRequest{
		EventType: "CompteOuvert",
		Count:     3,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/produce", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp ProduceResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.EventsProduced != 3 {
		t.Errorf("expected 3 events produced, got %d", resp.EventsProduced)
	}
	if len(resp.EventIDs) != 3 {
		t.Errorf("expected 3 event IDs, got %d", len(resp.EventIDs))
	}
	if len(producer.messages) != 3 {
		t.Errorf("expected 3 messages in producer, got %d", len(producer.messages))
	}
}

func TestHandleGetEventTypes(t *testing.T) {
	handler, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/types", nil)
	rec := httptest.NewRecorder()
	handler.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp EventTypesResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.EventTypes) == 0 {
		t.Error("expected non-empty event types list")
	}

	// Check that CompteOuvert is in the list
	found := false
	for _, et := range resp.EventTypes {
		if et == "CompteOuvert" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected CompteOuvert in event types list")
	}
}
