//go:build e2e
// +build e2e

// ============================================================================
// MVP End-to-End Tests
// ============================================================================
// These tests verify the complete MVP flow from event generation to persistence.
// Run with: go test -v -tags=e2e ./tests/e2e/...
// Prerequisites:
//   - make infra-up
//   - make services-up
// ============================================================================

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	gatewayURL         = "http://localhost:8080"
	simulatorURL       = "http://localhost:8081"
	bancaireURL        = "http://localhost:8082"
	postgresConnString = "postgres://edalab:edalab_password@localhost:5432/edalab"
)

// ============================================================================
// Helper Functions
// ============================================================================

func httpClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}

func postJSON(url string, body interface{}) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return httpClient().Do(req)
}

func getJSON(url string, result interface{}) error {
	resp, err := httpClient().Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func waitForService(url string, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		resp, err := httpClient().Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("service at %s not available after %v", url, maxWait)
}

// ============================================================================
// Service Health Tests
// ============================================================================

func TestServices_AllHealthy(t *testing.T) {
	t.Log("Testing all services are healthy...")

	services := map[string]string{
		"Gateway":   gatewayURL + "/api/v1/health",
		"Simulator": simulatorURL + "/health",
		"Bancaire":  bancaireURL + "/health",
	}

	for name, url := range services {
		t.Run(name, func(t *testing.T) {
			err := waitForService(url, 30*time.Second)
			require.NoError(t, err, "Service %s is not healthy", name)
			t.Logf("%s is healthy", name)
		})
	}
}

// ============================================================================
// Simulation Flow Tests
// ============================================================================

func TestMVP_SimulationStartStop(t *testing.T) {
	t.Log("Testing simulation start/stop flow...")

	// Wait for services
	err := waitForService(simulatorURL+"/health", 30*time.Second)
	require.NoError(t, err, "Simulator not available")

	// Start simulation
	startReq := map[string]interface{}{
		"scenario":    "default",
		"rate":        5,
		"duration_ms": 5000, // 5 seconds
	}

	resp, err := postJSON(simulatorURL+"/api/v1/simulation/start", startReq)
	require.NoError(t, err, "Failed to start simulation")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Start simulation failed")

	var startResult map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&startResult)
	require.NoError(t, err)
	t.Logf("Simulation started: %v", startResult)

	// Check status
	time.Sleep(2 * time.Second)

	var status map[string]interface{}
	err = getJSON(simulatorURL+"/api/v1/simulation/status", &status)
	require.NoError(t, err, "Failed to get status")
	t.Logf("Simulation status: %v", status)

	// Wait for simulation to complete
	time.Sleep(5 * time.Second)

	// Check final status
	err = getJSON(simulatorURL+"/api/v1/simulation/status", &status)
	require.NoError(t, err)

	eventsProduced, ok := status["events_produced"].(float64)
	if ok {
		t.Logf("Events produced: %.0f", eventsProduced)
		assert.GreaterOrEqual(t, int(eventsProduced), 20, "Expected at least 20 events")
	}
}

func TestMVP_ProduceEvents(t *testing.T) {
	t.Log("Testing manual event production...")

	err := waitForService(simulatorURL+"/health", 30*time.Second)
	require.NoError(t, err)

	// Produce 5 CompteOuvert events
	produceReq := map[string]interface{}{
		"event_type": "CompteOuvert",
		"count":      5,
	}

	resp, err := postJSON(simulatorURL+"/api/v1/events/produce", produceReq)
	require.NoError(t, err, "Failed to produce events")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Produce events failed")

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	eventIDs, ok := result["event_ids"].([]interface{})
	if ok {
		assert.Len(t, eventIDs, 5, "Expected 5 event IDs")
		t.Logf("Produced events: %v", eventIDs)
	}
}

// ============================================================================
// Full Flow Tests
// ============================================================================

func TestMVP_FullFlow_CompteOuvert(t *testing.T) {
	t.Log("Testing full flow: Simulator -> Kafka -> Bancaire -> PostgreSQL...")

	// Wait for all services
	services := []string{
		simulatorURL + "/health",
		bancaireURL + "/health",
	}
	for _, url := range services {
		err := waitForService(url, 30*time.Second)
		require.NoError(t, err, "Service not available: %s", url)
	}

	// Connect to PostgreSQL
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, postgresConnString)
	require.NoError(t, err, "Failed to connect to PostgreSQL")
	defer pool.Close()

	// Get initial count
	var initialCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM bancaire.comptes").Scan(&initialCount)
	require.NoError(t, err)
	t.Logf("Initial compte count: %d", initialCount)

	// Produce events
	produceReq := map[string]interface{}{
		"event_type": "CompteOuvert",
		"count":      10,
	}

	resp, err := postJSON(simulatorURL+"/api/v1/events/produce", produceReq)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Wait for Bancaire to process events
	t.Log("Waiting for events to be processed...")
	time.Sleep(10 * time.Second)

	// Check PostgreSQL for new comptes
	var finalCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM bancaire.comptes").Scan(&finalCount)
	require.NoError(t, err)
	t.Logf("Final compte count: %d", finalCount)

	newComptes := finalCount - initialCount
	t.Logf("New comptes created: %d", newComptes)

	// Allow some margin for timing
	assert.GreaterOrEqual(t, newComptes, 5, "Expected at least 5 new comptes")
}

func TestMVP_FullFlow_Simulation(t *testing.T) {
	t.Log("Testing full simulation flow with rate control...")

	// Wait for services
	err := waitForService(simulatorURL+"/health", 30*time.Second)
	require.NoError(t, err)
	err = waitForService(bancaireURL+"/health", 30*time.Second)
	require.NoError(t, err)

	// Connect to PostgreSQL
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, postgresConnString)
	require.NoError(t, err)
	defer pool.Close()

	var initialCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM bancaire.comptes").Scan(&initialCount)
	require.NoError(t, err)

	// Start simulation: 10 events/sec for 10 seconds = ~100 events
	startReq := map[string]interface{}{
		"scenario":    "default",
		"rate":        10,
		"duration_ms": 10000,
	}

	resp, err := postJSON(simulatorURL+"/api/v1/simulation/start", startReq)
	require.NoError(t, err)
	resp.Body.Close()

	// Wait for simulation to complete plus processing time
	t.Log("Running simulation for 10 seconds...")
	time.Sleep(15 * time.Second)

	// Get simulation status
	var status map[string]interface{}
	err = getJSON(simulatorURL+"/api/v1/simulation/status", &status)
	require.NoError(t, err)
	t.Logf("Final status: %v", status)

	// Check PostgreSQL
	var finalCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM bancaire.comptes").Scan(&finalCount)
	require.NoError(t, err)

	newComptes := finalCount - initialCount
	t.Logf("New comptes from simulation: %d", newComptes)

	// We should have created some comptes (allow margin for event type distribution)
	assert.GreaterOrEqual(t, newComptes, 10, "Expected at least 10 comptes from simulation")
}

// ============================================================================
// API Tests
// ============================================================================

func TestMVP_BancaireAPI_GetComptes(t *testing.T) {
	t.Log("Testing Bancaire API...")

	err := waitForService(bancaireURL+"/health", 30*time.Second)
	require.NoError(t, err)

	// Get comptes by client ID (may not find any, but API should work)
	resp, err := httpClient().Get(bancaireURL + "/api/v1/clients/test-client/comptes")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 200 (empty list) or 404
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, resp.StatusCode)
	t.Logf("API response status: %d", resp.StatusCode)
}

// ============================================================================
// Gateway Tests
// ============================================================================

func TestMVP_Gateway_ProxyToSimulator(t *testing.T) {
	t.Log("Testing Gateway proxy to Simulator...")

	err := waitForService(gatewayURL+"/api/v1/health", 30*time.Second)
	require.NoError(t, err)

	// Get simulation status through gateway
	resp, err := httpClient().Get(gatewayURL + "/api/v1/simulation/status")
	if err != nil {
		t.Skipf("Gateway proxy not configured: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Gateway proxy response: %d", resp.StatusCode)
}

// ============================================================================
// Chaos Tests
// ============================================================================

func TestMVP_Chaos_IdempotentProcessing(t *testing.T) {
	t.Log("Testing idempotent event processing...")

	err := waitForService(bancaireURL+"/health", 30*time.Second)
	require.NoError(t, err)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, postgresConnString)
	require.NoError(t, err)
	defer pool.Close()

	// Produce same event type multiple times
	// The Bancaire service should handle idempotency
	for i := 0; i < 3; i++ {
		produceReq := map[string]interface{}{
			"event_type": "CompteOuvert",
			"count":      1,
		}
		resp, err := postJSON(simulatorURL+"/api/v1/events/produce", produceReq)
		if err == nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	time.Sleep(5 * time.Second)

	// Check that processed_events table has entries
	var processedCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM bancaire.processed_events").Scan(&processedCount)
	if err == nil {
		t.Logf("Processed events tracked: %d", processedCount)
		assert.GreaterOrEqual(t, processedCount, 1, "Should have processed events")
	}
}

// ============================================================================
// Performance Tests
// ============================================================================

func TestMVP_Performance_Throughput(t *testing.T) {
	t.Log("Testing throughput performance...")

	err := waitForService(simulatorURL+"/health", 30*time.Second)
	require.NoError(t, err)

	// Run high-throughput simulation
	startReq := map[string]interface{}{
		"scenario":    "default",
		"rate":        50, // 50 events/sec
		"duration_ms": 5000,
	}

	startTime := time.Now()

	resp, err := postJSON(simulatorURL+"/api/v1/simulation/start", startReq)
	require.NoError(t, err)
	resp.Body.Close()

	time.Sleep(6 * time.Second)

	var status map[string]interface{}
	err = getJSON(simulatorURL+"/api/v1/simulation/status", &status)
	require.NoError(t, err)

	duration := time.Since(startTime)

	if eventsProduced, ok := status["events_produced"].(float64); ok {
		actualRate := eventsProduced / duration.Seconds()
		t.Logf("Target rate: 50/s, Actual: %.2f/s, Events: %.0f", actualRate, eventsProduced)
		assert.GreaterOrEqual(t, eventsProduced, float64(100), "Expected at least 100 events")
	}
}
