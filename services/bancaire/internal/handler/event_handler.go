// Package handler provides Kafka event handlers for the Bancaire service.
package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/edalab/pkg/events"
	"github.com/edalab/pkg/observability"
	"github.com/edalab/services/bancaire/internal/domain"
	"github.com/edalab/services/bancaire/internal/repository"
	"github.com/google/uuid"
)

// EventHandler handles Kafka events for the Bancaire domain.
type EventHandler struct {
	repo    repository.CompteRepository
	logger  *slog.Logger
	metrics *observability.Metrics
	service string
}

// NewEventHandler creates a new event handler.
func NewEventHandler(
	repo repository.CompteRepository,
	logger *slog.Logger,
	metrics *observability.Metrics,
	service string,
) *EventHandler {
	return &EventHandler{
		repo:    repo,
		logger:  logger,
		metrics: metrics,
		service: service,
	}
}

// HandleCompteOuvert processes a CompteOuvert event.
func (h *EventHandler) HandleCompteOuvert(ctx context.Context, event *events.CompteOuvert) error {
	timer := observability.NewTimer()
	defer func() {
		if h.metrics != nil {
			h.metrics.RecordProcessingLatency(h.service, "CompteOuvert", timer.Elapsed())
		}
	}()

	// Check idempotency
	processed, err := h.repo.EventProcessed(ctx, event.EventID)
	if err != nil {
		h.logger.Error("failed to check event idempotency",
			slog.String("event_id", event.EventID),
			slog.Any("error", err),
		)
		return err
	}
	if processed {
		h.logger.Info("event already processed (idempotent)",
			slog.String("event_id", event.EventID),
		)
		return nil
	}

	// Create compte
	compte := domain.NewCompte(
		event.CompteID,
		event.ClientID,
		domain.TypeCompte(event.TypeCompte),
		event.SoldeInitial,
		event.Devise,
	)

	err = h.repo.Create(ctx, compte)
	if err != nil {
		if err == domain.ErrDuplicateCompte {
			h.logger.Warn("compte already exists",
				slog.String("compte_id", event.CompteID),
			)
			// Mark as processed anyway for idempotency
			h.repo.MarkEventProcessed(ctx, event.EventID)
			return nil
		}
		h.logger.Error("failed to create compte",
			slog.String("compte_id", event.CompteID),
			slog.Any("error", err),
		)
		return err
	}

	// Add initial transaction if solde > 0
	if event.SoldeInitial.IsPositive() {
		tx := domain.NewTransaction(
			uuid.New().String(),
			event.CompteID,
			event.EventID,
			domain.TypeTransactionOuverture,
			event.SoldeInitial,
			event.SoldeInitial,
			event.Devise,
			"",
			"Ouverture de compte",
		)
		if err := h.repo.AddTransaction(ctx, tx); err != nil {
			h.logger.Error("failed to add opening transaction",
				slog.String("compte_id", event.CompteID),
				slog.Any("error", err),
			)
		}
	}

	// Mark event as processed
	if err := h.repo.MarkEventProcessed(ctx, event.EventID); err != nil {
		h.logger.Error("failed to mark event processed",
			slog.String("event_id", event.EventID),
			slog.Any("error", err),
		)
	}

	h.logger.Info("compte created",
		slog.String("event_id", event.EventID),
		slog.String("compte_id", event.CompteID),
		slog.String("client_id", event.ClientID),
		slog.String("solde_initial", event.SoldeInitial.String()),
	)

	return nil
}

// HandleDepotEffectue processes a DepotEffectue event.
func (h *EventHandler) HandleDepotEffectue(ctx context.Context, event *events.DepotEffectue) error {
	timer := observability.NewTimer()
	defer func() {
		if h.metrics != nil {
			h.metrics.RecordProcessingLatency(h.service, "DepotEffectue", timer.Elapsed())
		}
	}()

	// Check idempotency
	processed, err := h.repo.EventProcessed(ctx, event.EventID)
	if err != nil {
		return err
	}
	if processed {
		h.logger.Info("event already processed (idempotent)",
			slog.String("event_id", event.EventID),
		)
		return nil
	}

	// Get compte
	compte, err := h.repo.GetByID(ctx, event.CompteID)
	if err != nil {
		h.logger.Error("compte not found for depot",
			slog.String("compte_id", event.CompteID),
			slog.Any("error", err),
		)
		return err
	}

	// Credit the account
	compte.Credit(event.Montant)

	// Update solde
	if err := h.repo.UpdateSolde(ctx, compte.ID, compte.Solde); err != nil {
		return err
	}

	// Add transaction
	tx := domain.NewTransaction(
		uuid.New().String(),
		event.CompteID,
		event.EventID,
		domain.TypeTransactionDepot,
		event.Montant,
		compte.Solde,
		event.Devise,
		event.Reference,
		fmt.Sprintf("Dépôt %s", event.Canal),
	)
	if err := h.repo.AddTransaction(ctx, tx); err != nil {
		h.logger.Error("failed to add depot transaction",
			slog.String("compte_id", event.CompteID),
			slog.Any("error", err),
		)
	}

	// Mark event as processed
	h.repo.MarkEventProcessed(ctx, event.EventID)

	h.logger.Info("depot processed",
		slog.String("event_id", event.EventID),
		slog.String("compte_id", event.CompteID),
		slog.String("montant", event.Montant.String()),
		slog.String("new_solde", compte.Solde.String()),
	)

	return nil
}

// HandleRetraitEffectue processes a RetraitEffectue event.
func (h *EventHandler) HandleRetraitEffectue(ctx context.Context, event *events.RetraitEffectue) error {
	timer := observability.NewTimer()
	defer func() {
		if h.metrics != nil {
			h.metrics.RecordProcessingLatency(h.service, "RetraitEffectue", timer.Elapsed())
		}
	}()

	// Check idempotency
	processed, err := h.repo.EventProcessed(ctx, event.EventID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	// Get compte
	compte, err := h.repo.GetByID(ctx, event.CompteID)
	if err != nil {
		return err
	}

	// Debit the account
	if err := compte.Debit(event.Montant); err != nil {
		h.logger.Warn("insufficient funds for retrait",
			slog.String("compte_id", event.CompteID),
			slog.String("montant", event.Montant.String()),
			slog.String("solde", compte.Solde.String()),
		)
		return err
	}

	// Update solde
	if err := h.repo.UpdateSolde(ctx, compte.ID, compte.Solde); err != nil {
		return err
	}

	// Add transaction
	tx := domain.NewTransaction(
		uuid.New().String(),
		event.CompteID,
		event.EventID,
		domain.TypeTransactionRetrait,
		event.Montant.Neg(),
		compte.Solde,
		event.Devise,
		event.Reference,
		fmt.Sprintf("Retrait %s", event.Canal),
	)
	h.repo.AddTransaction(ctx, tx)
	h.repo.MarkEventProcessed(ctx, event.EventID)

	h.logger.Info("retrait processed",
		slog.String("event_id", event.EventID),
		slog.String("compte_id", event.CompteID),
		slog.String("montant", event.Montant.String()),
	)

	return nil
}

// HandleVirementEmis processes a VirementEmis event.
func (h *EventHandler) HandleVirementEmis(ctx context.Context, event *events.VirementEmis) error {
	timer := observability.NewTimer()
	defer func() {
		if h.metrics != nil {
			h.metrics.RecordProcessingLatency(h.service, "VirementEmis", timer.Elapsed())
		}
	}()

	// Check idempotency
	processed, err := h.repo.EventProcessed(ctx, event.EventID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	// Get source compte
	compteSource, err := h.repo.GetByID(ctx, event.CompteSourceID)
	if err != nil {
		h.logger.Error("source compte not found",
			slog.String("compte_source_id", event.CompteSourceID),
		)
		return err
	}

	// Check sufficient funds
	if !compteSource.CanDebit(event.Montant) {
		h.logger.Warn("insufficient funds for virement",
			slog.String("compte_source_id", event.CompteSourceID),
			slog.String("montant", event.Montant.String()),
			slog.String("solde", compteSource.Solde.String()),
		)
		return domain.ErrInsufficientFunds
	}

	// Debit source
	compteSource.Debit(event.Montant)
	if err := h.repo.UpdateSolde(ctx, compteSource.ID, compteSource.Solde); err != nil {
		return err
	}

	// Add debit transaction
	destID := event.CompteDestinationID
	txSource := domain.NewTransaction(
		uuid.New().String(),
		event.CompteSourceID,
		event.EventID,
		domain.TypeTransactionVirement,
		event.Montant.Neg(),
		compteSource.Solde,
		event.Devise,
		event.Reference,
		fmt.Sprintf("Virement vers %s: %s", event.CompteDestinationID, event.Motif),
	)
	txSource.CompteDestID = &destID
	h.repo.AddTransaction(ctx, txSource)

	// Credit destination if it exists locally
	compteDest, err := h.repo.GetByID(ctx, event.CompteDestinationID)
	if err == nil {
		compteDest.Credit(event.Montant)
		h.repo.UpdateSolde(ctx, compteDest.ID, compteDest.Solde)

		sourceID := event.CompteSourceID
		txDest := domain.NewTransaction(
			uuid.New().String(),
			event.CompteDestinationID,
			event.EventID,
			domain.TypeTransactionVirement,
			event.Montant,
			compteDest.Solde,
			event.Devise,
			event.Reference,
			fmt.Sprintf("Virement de %s: %s", event.CompteSourceID, event.Motif),
		)
		txDest.CompteSourceID = &sourceID
		h.repo.AddTransaction(ctx, txDest)
	}

	h.repo.MarkEventProcessed(ctx, event.EventID)

	h.logger.Info("virement processed",
		slog.String("event_id", event.EventID),
		slog.String("compte_source_id", event.CompteSourceID),
		slog.String("compte_dest_id", event.CompteDestinationID),
		slog.String("montant", event.Montant.String()),
	)

	return nil
}
