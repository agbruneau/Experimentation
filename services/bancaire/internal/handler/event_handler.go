package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/edalab/pkg/events"
	"github.com/edalab/pkg/kafka"
	"github.com/edalab/pkg/observability"
	"github.com/edalab/services/bancaire/internal/domain"
	"github.com/edalab/services/bancaire/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// EventHandler handles incoming Kafka events
type EventHandler struct {
	repo   repository.CompteRepository
	logger *slog.Logger
}

// NewEventHandler creates a new event handler
func NewEventHandler(repo repository.CompteRepository, logger *slog.Logger) *EventHandler {
	return &EventHandler{
		repo:   repo,
		logger: logger,
	}
}

// HandleCompteOuvert handles account opening events
func (h *EventHandler) HandleCompteOuvert(ctx context.Context, event *events.CompteOuvert) error {
	h.logger.Info("Handling CompteOuvert event",
		slog.String("event_id", event.EventID),
		slog.String("compte_id", event.CompteID),
		slog.String("client_id", event.ClientID),
	)

	// Check idempotency
	processed, err := h.repo.EventProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("failed to check event idempotency: %w", err)
	}
	if processed {
		h.logger.Info("Event already processed, skipping",
			slog.String("event_id", event.EventID),
		)
		return nil
	}

	// Create account
	compte := &domain.Compte{
		ID:         event.CompteID,
		ClientID:   event.ClientID,
		TypeCompte: string(event.TypeCompte),
		Devise:     event.Devise,
		Solde:      event.SoldeInitial,
		Statut:     domain.StatutActif,
		CreatedAt:  event.Timestamp,
		UpdatedAt:  event.Timestamp,
	}

	if err := h.repo.Create(ctx, compte); err != nil {
		if err == repository.ErrAlreadyExists {
			h.logger.Info("Account already exists",
				slog.String("compte_id", event.CompteID),
			)
			// Mark as processed even if account exists
			h.repo.MarkEventProcessed(ctx, event.EventID, "CompteOuvert")
			return nil
		}
		return fmt.Errorf("failed to create account: %w", err)
	}

	// Add initial transaction if there's a balance
	if event.SoldeInitial.GreaterThan(decimal.Zero) {
		tx := &domain.Transaction{
			ID:        uuid.New().String(),
			CompteID:  event.CompteID,
			Type:      domain.TypeOuvertureCompte,
			Montant:   event.SoldeInitial,
			Reference: event.EventID,
			CreatedAt: event.Timestamp,
		}
		if err := h.repo.AddTransaction(ctx, tx); err != nil {
			h.logger.Warn("Failed to add initial transaction",
				slog.String("error", err.Error()),
			)
		}
	}

	// Mark event as processed
	if err := h.repo.MarkEventProcessed(ctx, event.EventID, "CompteOuvert"); err != nil {
		h.logger.Warn("Failed to mark event as processed",
			slog.String("error", err.Error()),
		)
	}

	observability.MessagesConsumed.WithLabelValues("bancaire", events.TopicCompteOuvert).Inc()
	return nil
}

// HandleDepotEffectue handles deposit events
func (h *EventHandler) HandleDepotEffectue(ctx context.Context, event *events.DepotEffectue) error {
	h.logger.Info("Handling DepotEffectue event",
		slog.String("event_id", event.EventID),
		slog.String("compte_id", event.CompteID),
	)

	// Check idempotency
	processed, err := h.repo.EventProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("failed to check event idempotency: %w", err)
	}
	if processed {
		h.logger.Info("Event already processed, skipping",
			slog.String("event_id", event.EventID),
		)
		return nil
	}

	// Get current account
	compte, err := h.repo.GetByID(ctx, event.CompteID)
	if err != nil {
		if err == repository.ErrNotFound {
			h.logger.Warn("Account not found for deposit",
				slog.String("compte_id", event.CompteID),
			)
			observability.ProcessingErrors.WithLabelValues("bancaire", "account_not_found").Inc()
			return nil // Don't retry, account doesn't exist
		}
		return fmt.Errorf("failed to get account: %w", err)
	}

	// Update balance
	newSolde := compte.Solde.Add(event.Montant)
	if err := h.repo.UpdateSolde(ctx, event.CompteID, newSolde); err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// Add transaction
	tx := &domain.Transaction{
		ID:        uuid.New().String(),
		CompteID:  event.CompteID,
		Type:      domain.TypeDepot,
		Montant:   event.Montant,
		Reference: event.Reference,
		CreatedAt: event.Timestamp,
	}
	if err := h.repo.AddTransaction(ctx, tx); err != nil {
		h.logger.Warn("Failed to add transaction",
			slog.String("error", err.Error()),
		)
	}

	// Mark event as processed
	h.repo.MarkEventProcessed(ctx, event.EventID, "DepotEffectue")

	observability.MessagesConsumed.WithLabelValues("bancaire", events.TopicDepotEffectue).Inc()
	return nil
}

// HandleRetraitEffectue handles withdrawal events
func (h *EventHandler) HandleRetraitEffectue(ctx context.Context, event *events.RetraitEffectue) error {
	h.logger.Info("Handling RetraitEffectue event",
		slog.String("event_id", event.EventID),
		slog.String("compte_id", event.CompteID),
	)

	// Check idempotency
	processed, err := h.repo.EventProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("failed to check event idempotency: %w", err)
	}
	if processed {
		return nil
	}

	// Get current account
	compte, err := h.repo.GetByID(ctx, event.CompteID)
	if err != nil {
		if err == repository.ErrNotFound {
			h.logger.Warn("Account not found for withdrawal",
				slog.String("compte_id", event.CompteID),
			)
			return nil
		}
		return fmt.Errorf("failed to get account: %w", err)
	}

	// Check sufficient balance
	if compte.Solde.LessThan(event.Montant) {
		h.logger.Warn("Insufficient balance for withdrawal",
			slog.String("compte_id", event.CompteID),
			slog.String("solde", compte.Solde.String()),
			slog.String("montant", event.Montant.String()),
		)
		observability.ProcessingErrors.WithLabelValues("bancaire", "insufficient_balance").Inc()
		// Still mark as processed to avoid retries
		h.repo.MarkEventProcessed(ctx, event.EventID, "RetraitEffectue")
		return nil
	}

	// Update balance
	newSolde := compte.Solde.Sub(event.Montant)
	if err := h.repo.UpdateSolde(ctx, event.CompteID, newSolde); err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// Add transaction
	tx := &domain.Transaction{
		ID:        uuid.New().String(),
		CompteID:  event.CompteID,
		Type:      domain.TypeRetrait,
		Montant:   event.Montant,
		Reference: event.Reference,
		CreatedAt: event.Timestamp,
	}
	h.repo.AddTransaction(ctx, tx)
	h.repo.MarkEventProcessed(ctx, event.EventID, "RetraitEffectue")

	observability.MessagesConsumed.WithLabelValues("bancaire", events.TopicRetraitEffectue).Inc()
	return nil
}

// HandleVirementEmis handles outgoing transfer events
func (h *EventHandler) HandleVirementEmis(ctx context.Context, event *events.VirementEmis) error {
	h.logger.Info("Handling VirementEmis event",
		slog.String("event_id", event.EventID),
		slog.String("compte_source", event.CompteSourceID),
		slog.String("compte_dest", event.CompteDestinationID),
	)

	// Check idempotency
	processed, err := h.repo.EventProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("failed to check event idempotency: %w", err)
	}
	if processed {
		return nil
	}

	// Get source account
	compteSource, err := h.repo.GetByID(ctx, event.CompteSourceID)
	if err != nil {
		if err == repository.ErrNotFound {
			h.logger.Warn("Source account not found",
				slog.String("compte_id", event.CompteSourceID),
			)
			h.repo.MarkEventProcessed(ctx, event.EventID, "VirementEmis")
			return nil
		}
		return fmt.Errorf("failed to get source account: %w", err)
	}

	// Check sufficient balance
	if compteSource.Solde.LessThan(event.Montant) {
		h.logger.Warn("Insufficient balance for transfer",
			slog.String("compte_id", event.CompteSourceID),
		)
		h.repo.MarkEventProcessed(ctx, event.EventID, "VirementEmis")
		return nil
	}

	// Debit source account
	newSolde := compteSource.Solde.Sub(event.Montant)
	if err := h.repo.UpdateSolde(ctx, event.CompteSourceID, newSolde); err != nil {
		return fmt.Errorf("failed to update source balance: %w", err)
	}

	// Add outgoing transaction
	txOut := &domain.Transaction{
		ID:        uuid.New().String(),
		CompteID:  event.CompteSourceID,
		Type:      domain.TypeVirementSortant,
		Montant:   event.Montant,
		Reference: event.Reference,
		CreatedAt: event.Timestamp,
	}
	h.repo.AddTransaction(ctx, txOut)

	// Try to credit destination account if it exists
	compteDest, err := h.repo.GetByID(ctx, event.CompteDestinationID)
	if err == nil && compteDest != nil {
		newSoldeDest := compteDest.Solde.Add(event.Montant)
		h.repo.UpdateSolde(ctx, event.CompteDestinationID, newSoldeDest)

		txIn := &domain.Transaction{
			ID:        uuid.New().String(),
			CompteID:  event.CompteDestinationID,
			Type:      domain.TypeVirementEntrant,
			Montant:   event.Montant,
			Reference: event.Reference,
			CreatedAt: event.Timestamp,
		}
		h.repo.AddTransaction(ctx, txIn)
	}

	h.repo.MarkEventProcessed(ctx, event.EventID, "VirementEmis")
	observability.MessagesConsumed.WithLabelValues("bancaire", events.TopicVirementEmis).Inc()
	return nil
}

// Route routes messages to appropriate handlers
func (h *EventHandler) Route(ctx context.Context, msg *kafka.Message) error {
	start := time.Now()
	defer func() {
		observability.MessageLatency.WithLabelValues("bancaire", msg.Topic).Observe(time.Since(start).Seconds())
	}()

	// Deserialize value to map
	valueMap, ok := msg.Value.(map[string]interface{})
	if !ok {
		h.logger.Error("Invalid message value type",
			slog.String("topic", msg.Topic),
		)
		return fmt.Errorf("invalid message value type")
	}

	// Convert map to JSON then to struct
	jsonBytes, err := json.Marshal(valueMap)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	switch msg.Topic {
	case events.TopicCompteOuvert:
		var event events.CompteOuvert
		if err := json.Unmarshal(jsonBytes, &event); err != nil {
			return fmt.Errorf("failed to unmarshal CompteOuvert: %w", err)
		}
		return h.HandleCompteOuvert(ctx, &event)

	case events.TopicDepotEffectue:
		var event events.DepotEffectue
		if err := json.Unmarshal(jsonBytes, &event); err != nil {
			return fmt.Errorf("failed to unmarshal DepotEffectue: %w", err)
		}
		return h.HandleDepotEffectue(ctx, &event)

	case events.TopicRetraitEffectue:
		var event events.RetraitEffectue
		if err := json.Unmarshal(jsonBytes, &event); err != nil {
			return fmt.Errorf("failed to unmarshal RetraitEffectue: %w", err)
		}
		return h.HandleRetraitEffectue(ctx, &event)

	case events.TopicVirementEmis:
		var event events.VirementEmis
		if err := json.Unmarshal(jsonBytes, &event); err != nil {
			return fmt.Errorf("failed to unmarshal VirementEmis: %w", err)
		}
		return h.HandleVirementEmis(ctx, &event)

	default:
		h.logger.Warn("Unknown topic",
			slog.String("topic", msg.Topic),
		)
		return nil
	}
}
