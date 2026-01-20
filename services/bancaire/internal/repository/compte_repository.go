package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/edalab/pkg/database"
	"github.com/edalab/pkg/observability"
	"github.com/edalab/services/bancaire/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// ErrNotFound is returned when a resource is not found
var ErrNotFound = errors.New("not found")

// ErrAlreadyExists is returned when a resource already exists
var ErrAlreadyExists = errors.New("already exists")

// CompteRepository defines the interface for account persistence
type CompteRepository interface {
	Create(ctx context.Context, compte *domain.Compte) error
	GetByID(ctx context.Context, id string) (*domain.Compte, error)
	GetByClientID(ctx context.Context, clientID string) ([]*domain.Compte, error)
	UpdateSolde(ctx context.Context, id string, nouveauSolde decimal.Decimal) error
	AddTransaction(ctx context.Context, tx *domain.Transaction) error
	GetTransactions(ctx context.Context, compteID string, limit int) ([]*domain.Transaction, error)
	EventProcessed(ctx context.Context, eventID string) (bool, error)
	MarkEventProcessed(ctx context.Context, eventID, eventType string) error
}

// PostgresCompteRepository implements CompteRepository using PostgreSQL
type PostgresCompteRepository struct {
	pool *database.DBPool
}

// NewPostgresCompteRepository creates a new PostgreSQL repository
func NewPostgresCompteRepository(pool *database.DBPool) *PostgresCompteRepository {
	return &PostgresCompteRepository{pool: pool}
}

// Create creates a new account
func (r *PostgresCompteRepository) Create(ctx context.Context, compte *domain.Compte) error {
	start := time.Now()
	defer func() {
		observability.DatabaseOperations.WithLabelValues("bancaire", "create_compte").Observe(time.Since(start).Seconds())
	}()

	query := `
		INSERT INTO bancaire.comptes (id, client_id, type_compte, devise, solde, statut, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.pool.Pool().Exec(ctx, query,
		compte.ID,
		compte.ClientID,
		compte.TypeCompte,
		compte.Devise,
		compte.Solde,
		compte.Statut,
		compte.CreatedAt,
		compte.UpdatedAt,
	)

	if err != nil {
		// Check for duplicate key error
		if isPgDuplicateKeyError(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("failed to create compte: %w", err)
	}

	return nil
}

// GetByID retrieves an account by ID
func (r *PostgresCompteRepository) GetByID(ctx context.Context, id string) (*domain.Compte, error) {
	start := time.Now()
	defer func() {
		observability.DatabaseOperations.WithLabelValues("bancaire", "get_compte").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT id, client_id, type_compte, devise, solde, statut, created_at, updated_at
		FROM bancaire.comptes
		WHERE id = $1
	`

	compte := &domain.Compte{}
	err := r.pool.Pool().QueryRow(ctx, query, id).Scan(
		&compte.ID,
		&compte.ClientID,
		&compte.TypeCompte,
		&compte.Devise,
		&compte.Solde,
		&compte.Statut,
		&compte.CreatedAt,
		&compte.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get compte: %w", err)
	}

	return compte, nil
}

// GetByClientID retrieves all accounts for a client
func (r *PostgresCompteRepository) GetByClientID(ctx context.Context, clientID string) ([]*domain.Compte, error) {
	start := time.Now()
	defer func() {
		observability.DatabaseOperations.WithLabelValues("bancaire", "get_comptes_by_client").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT id, client_id, type_compte, devise, solde, statut, created_at, updated_at
		FROM bancaire.comptes
		WHERE client_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Pool().Query(ctx, query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to query comptes: %w", err)
	}
	defer rows.Close()

	var comptes []*domain.Compte
	for rows.Next() {
		compte := &domain.Compte{}
		if err := rows.Scan(
			&compte.ID,
			&compte.ClientID,
			&compte.TypeCompte,
			&compte.Devise,
			&compte.Solde,
			&compte.Statut,
			&compte.CreatedAt,
			&compte.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan compte: %w", err)
		}
		comptes = append(comptes, compte)
	}

	return comptes, rows.Err()
}

// UpdateSolde updates the account balance
func (r *PostgresCompteRepository) UpdateSolde(ctx context.Context, id string, nouveauSolde decimal.Decimal) error {
	start := time.Now()
	defer func() {
		observability.DatabaseOperations.WithLabelValues("bancaire", "update_solde").Observe(time.Since(start).Seconds())
	}()

	query := `
		UPDATE bancaire.comptes
		SET solde = $2, updated_at = $3
		WHERE id = $1
	`

	result, err := r.pool.Pool().Exec(ctx, query, id, nouveauSolde, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update solde: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// AddTransaction adds a transaction record
func (r *PostgresCompteRepository) AddTransaction(ctx context.Context, tx *domain.Transaction) error {
	start := time.Now()
	defer func() {
		observability.DatabaseOperations.WithLabelValues("bancaire", "add_transaction").Observe(time.Since(start).Seconds())
	}()

	query := `
		INSERT INTO bancaire.transactions (id, compte_id, type, montant, reference, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Pool().Exec(ctx, query,
		tx.ID,
		tx.CompteID,
		tx.Type,
		tx.Montant,
		tx.Reference,
		tx.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to add transaction: %w", err)
	}

	return nil
}

// GetTransactions retrieves transactions for an account
func (r *PostgresCompteRepository) GetTransactions(ctx context.Context, compteID string, limit int) ([]*domain.Transaction, error) {
	start := time.Now()
	defer func() {
		observability.DatabaseOperations.WithLabelValues("bancaire", "get_transactions").Observe(time.Since(start).Seconds())
	}()

	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, compte_id, type, montant, reference, created_at
		FROM bancaire.transactions
		WHERE compte_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Pool().Query(ctx, query, compteID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*domain.Transaction
	for rows.Next() {
		tx := &domain.Transaction{}
		if err := rows.Scan(
			&tx.ID,
			&tx.CompteID,
			&tx.Type,
			&tx.Montant,
			&tx.Reference,
			&tx.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	return transactions, rows.Err()
}

// EventProcessed checks if an event has already been processed (idempotency)
func (r *PostgresCompteRepository) EventProcessed(ctx context.Context, eventID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM events.processed_events WHERE event_id = $1)`

	var exists bool
	err := r.pool.Pool().QueryRow(ctx, query, eventID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check event: %w", err)
	}

	return exists, nil
}

// MarkEventProcessed marks an event as processed
func (r *PostgresCompteRepository) MarkEventProcessed(ctx context.Context, eventID, eventType string) error {
	query := `
		INSERT INTO events.processed_events (event_id, event_type, processed_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (event_id) DO NOTHING
	`

	_, err := r.pool.Pool().Exec(ctx, query, eventID, eventType, time.Now())
	if err != nil {
		return fmt.Errorf("failed to mark event processed: %w", err)
	}

	return nil
}

// isPgDuplicateKeyError checks if the error is a PostgreSQL duplicate key error
func isPgDuplicateKeyError(err error) bool {
	return err != nil && (err.Error() == "ERROR: duplicate key value violates unique constraint" ||
		contains(err.Error(), "duplicate key") ||
		contains(err.Error(), "23505"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
