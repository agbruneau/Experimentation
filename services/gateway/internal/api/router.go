// Package api provides the HTTP router for the Gateway service.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/edalab/pkg/observability"
	"github.com/edalab/services/gateway/internal/proxy"
	"github.com/edalab/services/gateway/internal/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Router handles HTTP routing for the Gateway.
type Router struct {
	proxy   *proxy.ServiceProxy
	hub     *websocket.Hub
	logger  *slog.Logger
	metrics *observability.Metrics
	service string
}

// NewRouter creates a new router.
func NewRouter(
	proxy *proxy.ServiceProxy,
	hub *websocket.Hub,
	logger *slog.Logger,
	metrics *observability.Metrics,
	service string,
) *Router {
	return &Router{
		proxy:   proxy,
		hub:     hub,
		logger:  logger,
		metrics: metrics,
		service: service,
	}
}

// Handler returns the HTTP handler.
func (rt *Router) Handler() http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(rt.corsMiddleware)
	r.Use(rt.metricsMiddleware)

	// Health check
	r.Get("/api/v1/health", rt.handleHealth)

	// WebSocket endpoint
	r.Get("/ws", rt.handleWebSocket)

	// Proxy to Simulator
	r.Route("/api/v1/simulation", func(r chi.Router) {
		r.HandleFunc("/*", func(w http.ResponseWriter, req *http.Request) {
			path := "/api/v1/simulation" + strings.TrimPrefix(req.URL.Path, "/api/v1/simulation")
			rt.proxy.ForwardToSimulator(w, req, path)
		})
	})

	r.Route("/api/v1/events", func(r chi.Router) {
		r.HandleFunc("/*", func(w http.ResponseWriter, req *http.Request) {
			path := "/api/v1/events" + strings.TrimPrefix(req.URL.Path, "/api/v1/events")
			rt.proxy.ForwardToSimulator(w, req, path)
		})
	})

	// Proxy to Bancaire
	r.Route("/api/v1/bancaire", func(r chi.Router) {
		r.HandleFunc("/*", func(w http.ResponseWriter, req *http.Request) {
			// Transform path: /api/v1/bancaire/comptes/* -> /api/v1/comptes/*
			path := strings.TrimPrefix(req.URL.Path, "/api/v1/bancaire")
			rt.proxy.ForwardToBancaire(w, req, "/api/v1"+path)
		})
	})

	r.Route("/api/v1/comptes", func(r chi.Router) {
		r.HandleFunc("/*", func(w http.ResponseWriter, req *http.Request) {
			path := "/api/v1/comptes" + strings.TrimPrefix(req.URL.Path, "/api/v1/comptes")
			rt.proxy.ForwardToBancaire(w, req, path)
		})
	})

	r.Route("/api/v1/clients", func(r chi.Router) {
		r.HandleFunc("/*", func(w http.ResponseWriter, req *http.Request) {
			path := "/api/v1/clients" + strings.TrimPrefix(req.URL.Path, "/api/v1/clients")
			rt.proxy.ForwardToBancaire(w, req, path)
		})
	})

	return r
}

// corsMiddleware adds CORS headers for web UI access.
func (rt *Router) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// metricsMiddleware records HTTP request metrics.
func (rt *Router) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		if rt.metrics != nil {
			duration := time.Since(start)
			status := http.StatusText(ww.Status())
			rt.metrics.RecordHTTPRequest(rt.service, r.Method, r.URL.Path, status, duration)
		}
	})
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status     string            `json:"status"`
	Service    string            `json:"service"`
	Services   map[string]string `json:"services"`
	WebSocket  WebSocketStatus   `json:"websocket"`
	Timestamp  string            `json:"timestamp"`
}

// WebSocketStatus represents WebSocket hub status.
type WebSocketStatus struct {
	Clients int `json:"clients"`
}

// handleHealth handles the health check endpoint.
func (rt *Router) handleHealth(w http.ResponseWriter, r *http.Request) {
	services := rt.proxy.HealthCheck()

	// Determine overall status
	status := "healthy"
	for _, s := range services {
		if s != "healthy" {
			status = "degraded"
			break
		}
	}

	response := HealthResponse{
		Status:   status,
		Service:  rt.service,
		Services: services,
		WebSocket: WebSocketStatus{
			Clients: rt.hub.ClientCount(),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWebSocket handles WebSocket connections.
func (rt *Router) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	websocket.ServeWs(rt.hub, rt.logger, w, r)
}
