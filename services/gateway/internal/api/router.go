package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/edalab/services/gateway/internal/proxy"
	"github.com/edalab/services/gateway/internal/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	ws "github.com/gorilla/websocket"
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// Router handles HTTP routing for the gateway
type Router struct {
	proxy  *proxy.ServiceProxy
	hub    *websocket.Hub
	logger *slog.Logger
}

// NewRouter creates a new router
func NewRouter(proxy *proxy.ServiceProxy, hub *websocket.Hub, logger *slog.Logger) *Router {
	return &Router{
		proxy:  proxy,
		hub:    hub,
		logger: logger,
	}
}

// Handler returns the HTTP handler
func (rt *Router) Handler() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	// CORS
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Health check
	r.Get("/health", rt.healthCheck)

	// WebSocket endpoint
	r.Get("/ws", rt.handleWebSocket)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Simulation routes -> Simulator service
		r.Route("/simulation", func(r chi.Router) {
			r.Post("/start", rt.proxy.ForwardToSimulator)
			r.Post("/stop", rt.proxy.ForwardToSimulator)
			r.Get("/status", rt.proxy.ForwardToSimulator)
		})

		// Events routes -> Simulator service
		r.Route("/events", func(r chi.Router) {
			r.Post("/produce", rt.proxy.ForwardToSimulator)
		})

		// Bancaire routes -> Bancaire service
		r.Route("/bancaire", func(r chi.Router) {
			r.Get("/comptes/{id}", rt.proxy.ForwardToBancaire)
			r.Get("/comptes/{id}/transactions", rt.proxy.ForwardToBancaire)
			r.Get("/clients/{client_id}/comptes", rt.proxy.ForwardToBancaire)
		})
	})

	return r
}

// healthCheck handles health check requests
func (rt *Router) healthCheck(w http.ResponseWriter, r *http.Request) {
	// Check backend services
	backendHealth := rt.proxy.HealthCheck()

	response := map[string]interface{}{
		"status":      "healthy",
		"service":     "gateway",
		"backends":    backendHealth,
		"connections": rt.hub.ClientCount(),
	}

	// If any backend is unhealthy, mark gateway as degraded
	allHealthy := true
	for _, healthy := range backendHealth {
		if !healthy {
			allHealthy = false
			break
		}
	}

	if !allHealthy {
		response["status"] = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWebSocket handles WebSocket connections
func (rt *Router) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		rt.logger.Error("WebSocket upgrade failed", slog.String("error", err.Error()))
		return
	}

	client := websocket.NewClient(rt.hub, conn)
	rt.hub.Register(client)

	// Start client pumps
	go client.WritePump()
	go client.ReadPump()
}
