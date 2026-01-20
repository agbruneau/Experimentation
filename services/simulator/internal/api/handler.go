package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/edalab/services/simulator/internal/simulation"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Handler handles HTTP requests for the simulator API
type Handler struct {
	manager *simulation.Manager
	logger  *slog.Logger
}

// NewHandler creates a new API handler
func NewHandler(manager *simulation.Manager, logger *slog.Logger) *Handler {
	return &Handler{
		manager: manager,
		logger:  logger,
	}
}

// Router returns the HTTP router
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * 1000 * 1000 * 1000)) // 60 seconds

	// CORS
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Routes
	r.Get("/health", h.healthCheck)
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/simulation", func(r chi.Router) {
			r.Post("/start", h.startSimulation)
			r.Post("/stop", h.stopSimulation)
			r.Get("/status", h.getStatus)
		})
		r.Route("/events", func(r chi.Router) {
			r.Post("/produce", h.produceEvents)
		})
	})

	return r
}

// healthCheck handles health check requests
func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	status := h.manager.Status()
	response := map[string]interface{}{
		"status":     "healthy",
		"service":    "simulator",
		"simulation": status.State,
	}
	writeJSON(w, http.StatusOK, response)
}

// StartRequest represents a start simulation request
type StartRequest struct {
	Scenario string `json:"scenario"`
	Rate     int    `json:"rate"`
	Duration int    `json:"duration"`
}

// startSimulation handles starting a simulation
func (h *Handler) startSimulation(w http.ResponseWriter, r *http.Request) {
	var req StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	config := simulation.Config{
		Scenario: req.Scenario,
		Rate:     req.Rate,
		Duration: req.Duration,
	}

	if err := h.manager.Start(r.Context(), config); err != nil {
		h.logger.Error("Failed to start simulation", slog.String("error", err.Error()))
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	status := h.manager.Status()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"simulation_id": status.ID,
		"status":        status.State,
		"rate":          status.Rate,
		"duration":      status.Duration,
	})
}

// stopSimulation handles stopping a simulation
func (h *Handler) stopSimulation(w http.ResponseWriter, r *http.Request) {
	status, err := h.manager.Stop()
	if err != nil {
		h.logger.Error("Failed to stop simulation", slog.String("error", err.Error()))
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":          "stopped",
		"events_produced": status.EventsProduced,
		"actual_rate":     status.ActualRate,
	})
}

// getStatus handles getting simulation status
func (h *Handler) getStatus(w http.ResponseWriter, r *http.Request) {
	status := h.manager.Status()
	writeJSON(w, http.StatusOK, status)
}

// ProduceRequest represents a produce events request
type ProduceRequest struct {
	EventType string `json:"event_type"`
	Count     int    `json:"count"`
}

// produceEvents handles producing individual events
func (h *Handler) produceEvents(w http.ResponseWriter, r *http.Request) {
	var req ProduceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Count <= 0 {
		req.Count = 1
	}
	if req.Count > 100 {
		req.Count = 100 // Limit batch size
	}

	eventIDs, err := h.manager.ProduceEvent(r.Context(), req.EventType, req.Count)
	if err != nil {
		h.logger.Error("Failed to produce events",
			slog.String("error", err.Error()),
			slog.String("event_type", req.EventType),
		)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events_produced": len(eventIDs),
		"event_ids":       eventIDs,
	})
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
