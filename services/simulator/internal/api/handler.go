// Package api provides REST API handlers for the Simulator service.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/edalab/pkg/observability"
	"github.com/edalab/services/simulator/internal/simulation"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Handler handles HTTP requests for the Simulator API.
type Handler struct {
	manager *simulation.Manager
	logger  *slog.Logger
	metrics *observability.Metrics
	service string
}

// NewHandler creates a new API handler.
func NewHandler(
	manager *simulation.Manager,
	logger *slog.Logger,
	metrics *observability.Metrics,
	service string,
) *Handler {
	return &Handler{
		manager: manager,
		logger:  logger,
		metrics: metrics,
		service: service,
	}
}

// Router returns the HTTP router for the API.
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(h.metricsMiddleware)

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		// Health check
		r.Get("/health", h.handleHealth)

		// Simulation endpoints
		r.Route("/simulation", func(r chi.Router) {
			r.Post("/start", h.handleStartSimulation)
			r.Post("/stop", h.handleStopSimulation)
			r.Get("/status", h.handleGetStatus)
		})

		// Event production endpoints
		r.Route("/events", func(r chi.Router) {
			r.Post("/produce", h.handleProduceEvents)
			r.Get("/types", h.handleGetEventTypes)
		})
	})

	return r
}

// metricsMiddleware records HTTP request metrics.
func (h *Handler) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		if h.metrics != nil {
			duration := time.Since(start)
			status := http.StatusText(ww.Status())
			h.metrics.RecordHTTPRequest(h.service, r.Method, r.URL.Path, status, duration)
		}
	})
}

// StartRequest represents a request to start a simulation.
type StartRequest struct {
	Scenario   string   `json:"scenario"`
	Rate       int      `json:"rate"`
	Duration   int      `json:"duration"` // seconds, 0 = infinite
	EventTypes []string `json:"event_types,omitempty"`
}

// StartResponse represents the response to starting a simulation.
type StartResponse struct {
	SimulationID string `json:"simulation_id"`
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
}

// handleStartSimulation handles POST /api/v1/simulation/start
func (h *Handler) handleStartSimulation(w http.ResponseWriter, r *http.Request) {
	var req StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Build config
	config := simulation.Config{
		Scenario:   req.Scenario,
		Rate:       req.Rate,
		Duration:   time.Duration(req.Duration) * time.Second,
		EventTypes: req.EventTypes,
	}

	if config.Scenario == "" {
		config.Scenario = "default"
	}
	if config.Rate <= 0 {
		config.Rate = 10
	}
	if len(config.EventTypes) == 0 {
		config.EventTypes = []string{"CompteOuvert"}
	}

	// Start simulation
	if err := h.manager.Start(r.Context(), config); err != nil {
		h.writeError(w, http.StatusConflict, err.Error())
		return
	}

	status := h.manager.Status()
	h.writeJSON(w, http.StatusOK, StartResponse{
		SimulationID: status.ID,
		Status:       string(status.Status),
		Message:      "simulation started",
	})
}

// StopResponse represents the response to stopping a simulation.
type StopResponse struct {
	Status         string  `json:"status"`
	EventsProduced int64   `json:"events_produced"`
	Duration       float64 `json:"duration_seconds"`
	Message        string  `json:"message,omitempty"`
}

// handleStopSimulation handles POST /api/v1/simulation/stop
func (h *Handler) handleStopSimulation(w http.ResponseWriter, r *http.Request) {
	status, err := h.manager.Stop()
	if err != nil {
		h.writeError(w, http.StatusConflict, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, StopResponse{
		Status:         string(status.Status),
		EventsProduced: status.EventsProduced,
		Duration:       status.Duration,
		Message:        "simulation stopped",
	})
}

// handleGetStatus handles GET /api/v1/simulation/status
func (h *Handler) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.manager.Status()
	h.writeJSON(w, http.StatusOK, status)
}

// ProduceRequest represents a request to produce events.
type ProduceRequest struct {
	EventType string `json:"event_type"`
	Count     int    `json:"count"`
}

// ProduceResponse represents the response to producing events.
type ProduceResponse struct {
	EventsProduced int      `json:"events_produced"`
	EventIDs       []string `json:"event_ids"`
}

// handleProduceEvents handles POST /api/v1/events/produce
func (h *Handler) handleProduceEvents(w http.ResponseWriter, r *http.Request) {
	var req ProduceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.EventType == "" {
		req.EventType = "CompteOuvert"
	}
	if req.Count <= 0 {
		req.Count = 1
	}
	if req.Count > 1000 {
		req.Count = 1000 // Limit batch size
	}

	eventIDs, err := h.manager.ProduceEvents(r.Context(), req.EventType, req.Count)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, ProduceResponse{
		EventsProduced: len(eventIDs),
		EventIDs:       eventIDs,
	})
}

// EventTypesResponse represents the list of supported event types.
type EventTypesResponse struct {
	EventTypes []string `json:"event_types"`
}

// handleGetEventTypes handles GET /api/v1/events/types
func (h *Handler) handleGetEventTypes(w http.ResponseWriter, r *http.Request) {
	types := h.manager.SupportedEventTypes()
	h.writeJSON(w, http.StatusOK, EventTypesResponse{
		EventTypes: types,
	})
}

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status     string `json:"status"`
	Service    string `json:"service"`
	Simulation string `json:"simulation"`
	Timestamp  string `json:"timestamp"`
}

// handleHealth handles GET /api/v1/health
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	simStatus := "idle"
	if h.manager.IsRunning() {
		simStatus = "running"
	}

	h.writeJSON(w, http.StatusOK, HealthResponse{
		Status:     "healthy",
		Service:    h.service,
		Simulation: simStatus,
		Timestamp:  time.Now().Format(time.RFC3339),
	})
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// writeJSON writes a JSON response.
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response",
			slog.Any("error", err),
		)
	}
}

// writeError writes an error response.
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}
