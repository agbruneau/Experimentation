// Package repository provides data access for the Bancaire service.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/edalab/services/bancaire/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// CompteRepository defines the interface for compte data access.
type CompteRepository interface {
	Create(ctx context.Context, compte *domain.Compte) error
	GetByID(ctx context.Context, id string) (*domain.Compte, error)
	GetByClientID(ctx context.Context, clientID string) ([]*domain.Compte, error)
	UpdateSolde(ctx context.Context, id string, solde decimal.Decimal) error
	Exists(ctx context.Context, id string) (bool, error)
	AddTransaction(ctx context.Context, tx *domain.Transaction) error
	GetTransactions(ctx context.Context, compteID string, limit int) ([]*domain.Transaction, error)
	EventProcessed(ctx context.Context, eventID string) (bool, error)
	MarkEventProcessed(ctx context.Context, eventID string) error
}

// PostgresCompteRepository implements CompteRepository using PostgreSQL.
type PostgresCompteRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresCompteRepository creates a new PostgreSQL repository.
func NewPostgresCompteRepository(pool *pgxpool.Pool) *PostgresCompteRepository {
	return &PostgresCompteRepository{pool: pool}
}

// Create inserts a new compte into the database.
func (r *PostgresCompteRepository) Create(ctx context.Context, compte *domain.Compte) error {
	query := `
		INSERT INTO bancaire.comptes (id, client_id, type_compte, solde, devise, statut, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		compte.ID,
		compte.ClientID,
		compte.TypeCompte,
		compte.Solde,
		compte.Devise,
		compte.Statut,
		compte.CreatedAt,
		compte.UpdatedAt,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return domain.ErrDuplicateCompte
		}
		return fmt.Errorf("failed to create compte: %w", err)
	}
	return nil
}

// GetByID retrieves a compte by its ID.
func (r *PostgresCompteRepository) GetByID(ctx context.Context, id string) (*domain.Compte, error) {
	query := `
		SELECT id, client_id, type_compte, solde, devise, statut, created_at, updated_at
		FROM bancaire.comptes
		WHERE id = $1
	`
	compte := &domain.Compte{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&compte.ID,
		&compte.ClientID,
		&compte.TypeCompte,
		&compte.Solde,
		&compte.Devise,
		&compte.Statut,
		&compte.CreatedAt,
		&compte.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCompteNotFound
		}
		return nil, fmt.Errorf("failed to get compte: %w", err)
	}
	return compte, nil
}

// GetByClientID retrieves all comptes for a client.
func (r *PostgresCompteRepository) GetByClientID(ctx context.Context, clientID string) ([]*domain.Compte, error) {
	query := `
		SELECT id, client_id, type_compte, solde, devise, statut, created_at, updated_at
		FROM bancaire.comptes
		WHERE client_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to query comptes: %w", err)
	}
	defer rows.Close()

	var comptes []*domain.Compte
	for rows.Next() {
		compte := &domain.Compte{}
		err := rows.Scan(
			&compte.ID,
			&compte.ClientID,
			&compte.TypeCompte,
			&compte.Solde,
			&compte.Devise,
			&compte.Statut,
			&compte.CreatedAt,
			&compte.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan compte: %w", err)
		}
		comptes = append(comptes, compte)
	}
	return comptes, nil
}

// UpdateSolde updates the balance of a compte.
func (r *PostgresCompteRepository) UpdateSolde(ctx context.Context, id string, solde decimal.Decimal) error {
	query := `
		UPDATE bancaire.comptes
		SET solde = $2, updated_at = NOW()
		WHERE id = $1
	`
	result, err := r.pool.Exec(ctx, query, id, solde)
	if err != nil {
		return fmt.Errorf("failed to update solde: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrCompteNotFound
	}
	return nil
}

// Exists checks if a compte exists.
func (r *PostgresCompteRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM bancaire.comptes WHERE id = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return exists, nil
}

// AddTransaction inserts a new transaction.
func (r *PostgresCompteRepository) AddTransaction(ctx context.Context, tx *domain.Transaction) error {
	query := `
		INSERT INTO bancaire.transactions
		(id, compte_id, event_id, type, montant, devise, solde_apres, reference, description, compte_source_id, compte_dest_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.pool.Exec(ctx, query,
		tx.ID,
		tx.CompteID,
		tx.EventID,
		tx.Type,
		tx.Montant,
		tx.Devise,
		tx.SoldeApres,
		tx.Reference,
		tx.Description,
		tx.CompteSourceID,
		tx.CompteDestID,
		tx.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to add transaction: %w", err)
	}
	return nil
}

// GetTransactions retrieves transactions for a compte.
func (r *PostgresCompteRepository) GetTransactions(ctx context.Context, compteID string, limit int) ([]*domain.Transaction, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, compte_id, event_id, type, montant, devise, solde_apres, reference, description, compte_source_id, compte_dest_id, created_at
		FROM bancaire.transactions
		WHERE compte_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, compteID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*domain.Transaction
	for rows.Next() {
		tx := &domain.Transaction{}
		err := rows.Scan(
			&tx.ID,
			&tx.CompteID,
			&tx.EventID,
			&tx.Type,
			&tx.Montant,
			&tx.Devise,
			&tx.SoldeApres,
			&tx.Reference,
			&tx.Description,
			&tx.CompteSourceID,
			&tx.CompteDestID,
			&tx.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}
	return transactions, nil
}

// EventProcessed checks if an event has already been processed (idempotency).
func (r *PostgresCompteRepository) EventProcessed(ctx context.Context, eventID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM bancaire.processed_events WHERE event_id = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, eventID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check event: %w", err)
	}
	return exists, nil
}

// MarkEventProcessed marks an event as processed.
func (r *PostgresCompteRepository) MarkEventProcessed(ctx context.Context, eventID string) error {
	query := `INSERT INTO bancaire.processed_events (event_id, processed_at) VALUES ($1, NOW()) ON CONFLICT DO NOTHING`
	_, err := r.pool.Exec(ctx, query, eventID)
	if err != nil {
		return fmt.Errorf("failed to mark event processed: %w", err)
	}
	return nil
}

// isDuplicateKeyError checks if the error is a duplicate key violation.
func isDuplicateKeyError(err error) bool {
	return err != nil && (errors.Is(err, pgx.ErrNoRows) == false &&
		(fmt.Sprintf("%v", err) == "ERROR: duplicate key value violates unique constraint" ||
		 len(fmt.Sprintf("%v", err)) > 0 && fmt.Sprintf("%v", err)[:5] == "ERROR"))
}
