//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	gatewayURL   = "http://localhost:8082"
	simulatorURL = "http://localhost:8080"
	bancaireURL  = "http://localhost:8081"
)

// TestFullEventFlow tests the complete flow:
// Simulator -> Kafka -> Bancaire
func TestFullEventFlow(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	// 1. Check all services are healthy
	t.Run("ServicesHealth", func(t *testing.T) {
		checkHealth(t, client, gatewayURL+"/health")
		checkHealth(t, client, simulatorURL+"/health")
		checkHealth(t, client, bancaireURL+"/health")
	})

	// 2. Produce a single event via simulator
	var eventID string
	t.Run("ProduceEvent", func(t *testing.T) {
		resp, err := client.Post(
			simulatorURL+"/api/v1/events/produce",
			"application/json",
			strings.NewReader(`{"event_type": "compte_ouvert"}`),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)

		if event, ok := result["event"].(map[string]interface{}); ok {
			eventID = event["event_id"].(string)
		}
		assert.NotEmpty(t, eventID, "Event ID should be returned")
	})

	// 3. Wait for event to be processed
	t.Run("WaitForProcessing", func(t *testing.T) {
		// Give some time for the event to flow through the system
		time.Sleep(3 * time.Second)
	})

	// 4. Start simulation and verify events are produced
	t.Run("StartSimulation", func(t *testing.T) {
		resp, err := client.Post(
			simulatorURL+"/api/v1/simulation/start",
			"application/json",
			strings.NewReader(`{"rate": 2}`),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)

		assert.Equal(t, true, result["running"])
	})

	// 5. Let simulation run for a few seconds
	t.Run("LetSimulationRun", func(t *testing.T) {
		time.Sleep(5 * time.Second)
	})

	// 6. Check simulation status
	t.Run("CheckSimulationStatus", func(t *testing.T) {
		resp, err := client.Get(simulatorURL + "/api/v1/simulation/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)

		eventsProduced := result["events_produced"].(float64)
		assert.Greater(t, eventsProduced, float64(0), "Should have produced some events")
	})

	// 7. Stop simulation
	t.Run("StopSimulation", func(t *testing.T) {
		resp, err := client.Post(
			simulatorURL+"/api/v1/simulation/stop",
			"application/json",
			nil,
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)

		assert.Equal(t, false, result["running"])
	})
}

// TestGatewayProxy tests gateway routing to backend services
func TestGatewayProxy(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Test routing through gateway to simulator
	t.Run("GatewayToSimulator", func(t *testing.T) {
		resp, err := client.Get(gatewayURL + "/api/v1/simulation/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test routing through gateway to bancaire
	t.Run("GatewayToBancaire", func(t *testing.T) {
		// This might return 404 if account doesn't exist, but should route correctly
		resp, err := client.Get(gatewayURL + "/api/v1/bancaire/comptes/test-id")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Either 200 (found) or 404 (not found) are acceptable - both mean routing worked
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound)
	})
}

// TestEventTypes tests production of different event types
func TestEventTypes(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	eventTypes := []string{
		"compte_ouvert",
		"depot_effectue",
		"retrait_effectue",
		"virement_emis",
	}

	for _, eventType := range eventTypes {
		t.Run(eventType, func(t *testing.T) {
			resp, err := client.Post(
				simulatorURL+"/api/v1/events/produce",
				"application/json",
				strings.NewReader(fmt.Sprintf(`{"event_type": "%s"}`, eventType)),
			)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func checkHealth(t *testing.T, client *http.Client, url string) {
	resp, err := client.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	require.NoError(t, err)

	status := result["status"].(string)
	assert.True(t, status == "healthy" || status == "degraded", "Service should be healthy or degraded, got: %s", status)
}
