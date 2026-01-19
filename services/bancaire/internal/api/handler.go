// Package api provides REST API handlers for the Bancaire service.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/edalab/pkg/observability"
	"github.com/edalab/services/bancaire/internal/domain"
	"github.com/edalab/services/bancaire/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Handler handles HTTP requests for the Bancaire API.
type Handler struct {
	repo    repository.CompteRepository
	logger  *slog.Logger
	metrics *observability.Metrics
	service string
}

// NewHandler creates a new API handler.
func NewHandler(
	repo repository.CompteRepository,
	logger *slog.Logger,
	metrics *observability.Metrics,
	service string,
) *Handler {
	return &Handler{
		repo:    repo,
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
		r.Get("/health", h.handleHealth)

		r.Route("/comptes", func(r chi.Router) {
			r.Get("/{id}", h.handleGetCompte)
			r.Get("/{id}/transactions", h.handleGetTransactions)
		})

		r.Route("/clients", func(r chi.Router) {
			r.Get("/{clientId}/comptes", h.handleGetComptesByClient)
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

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Database  string `json:"database"`
	Timestamp string `json:"timestamp"`
}

// handleHealth handles GET /api/v1/health
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbStatus := "healthy"
	// Could add actual DB health check here

	h.writeJSON(w, http.StatusOK, HealthResponse{
		Status:    "healthy",
		Service:   h.service,
		Database:  dbStatus,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// CompteResponse represents a compte in API responses.
type CompteResponse struct {
	ID         string `json:"id"`
	ClientID   string `json:"client_id"`
	TypeCompte string `json:"type_compte"`
	Solde      string `json:"solde"`
	Devise     string `json:"devise"`
	Statut     string `json:"statut"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// handleGetCompte handles GET /api/v1/comptes/{id}
func (h *Handler) handleGetCompte(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	compte, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrCompteNotFound {
			h.writeError(w, http.StatusNotFound, "compte not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "failed to get compte")
		return
	}

	h.writeJSON(w, http.StatusOK, compteToResponse(compte))
}

// TransactionResponse represents a transaction in API responses.
type TransactionResponse struct {
	ID             string  `json:"id"`
	CompteID       string  `json:"compte_id"`
	EventID        string  `json:"event_id"`
	Type           string  `json:"type"`
	Montant        string  `json:"montant"`
	Devise         string  `json:"devise"`
	SoldeApres     string  `json:"solde_apres"`
	Reference      string  `json:"reference"`
	Description    string  `json:"description"`
	CompteSourceID *string `json:"compte_source_id,omitempty"`
	CompteDestID   *string `json:"compte_dest_id,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

// TransactionsResponse represents a list of transactions.
type TransactionsResponse struct {
	CompteID     string                 `json:"compte_id"`
	Transactions []*TransactionResponse `json:"transactions"`
	Count        int                    `json:"count"`
}

// handleGetTransactions handles GET /api/v1/comptes/{id}/transactions
func (h *Handler) handleGetTransactions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Parse limit query param
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Check compte exists
	_, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrCompteNotFound {
			h.writeError(w, http.StatusNotFound, "compte not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "failed to get compte")
		return
	}

	transactions, err := h.repo.GetTransactions(r.Context(), id, limit)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to get transactions")
		return
	}

	response := TransactionsResponse{
		CompteID:     id,
		Transactions: make([]*TransactionResponse, len(transactions)),
		Count:        len(transactions),
	}
	for i, tx := range transactions {
		response.Transactions[i] = transactionToResponse(tx)
	}

	h.writeJSON(w, http.StatusOK, response)
}

// ComptesResponse represents a list of comptes.
type ComptesResponse struct {
	ClientID string           `json:"client_id"`
	Comptes  []*CompteResponse `json:"comptes"`
	Count    int              `json:"count"`
}

// handleGetComptesByClient handles GET /api/v1/clients/{clientId}/comptes
func (h *Handler) handleGetComptesByClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "clientId")

	comptes, err := h.repo.GetByClientID(r.Context(), clientID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to get comptes")
		return
	}

	response := ComptesResponse{
		ClientID: clientID,
		Comptes:  make([]*CompteResponse, len(comptes)),
		Count:    len(comptes),
	}
	for i, c := range comptes {
		response.Comptes[i] = compteToResponse(c)
	}

	h.writeJSON(w, http.StatusOK, response)
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
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}

// compteToResponse converts a domain Compte to API response.
func compteToResponse(c *domain.Compte) *CompteResponse {
	return &CompteResponse{
		ID:         c.ID,
		ClientID:   c.ClientID,
		TypeCompte: string(c.TypeCompte),
		Solde:      c.Solde.String(),
		Devise:     c.Devise,
		Statut:     c.Statut,
		CreatedAt:  c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  c.UpdatedAt.Format(time.RFC3339),
	}
}

// transactionToResponse converts a domain Transaction to API response.
func transactionToResponse(tx *domain.Transaction) *TransactionResponse {
	return &TransactionResponse{
		ID:             tx.ID,
		CompteID:       tx.CompteID,
		EventID:        tx.EventID,
		Type:           string(tx.Type),
		Montant:        tx.Montant.String(),
		Devise:         tx.Devise,
		SoldeApres:     tx.SoldeApres.String(),
		Reference:      tx.Reference,
		Description:    tx.Description,
		CompteSourceID: tx.CompteSourceID,
		CompteDestID:   tx.CompteDestID,
		CreatedAt:      tx.CreatedAt.Format(time.RFC3339),
	}
}
