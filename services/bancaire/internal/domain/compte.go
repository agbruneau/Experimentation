// Package domain defines the domain models for the Bancaire service.
package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// TypeCompte represents the type of bank account.
type TypeCompte string

const (
	TypeCompteCourant TypeCompte = "COURANT"
	TypeCompteEpargne TypeCompte = "EPARGNE"
	TypeCompteJoint   TypeCompte = "JOINT"
)

// TypeTransaction represents the type of transaction.
type TypeTransaction string

const (
	TypeTransactionOuverture TypeTransaction = "OUVERTURE"
	TypeTransactionDepot     TypeTransaction = "DEPOT"
	TypeTransactionRetrait   TypeTransaction = "RETRAIT"
	TypeTransactionVirement  TypeTransaction = "VIREMENT"
	TypeTransactionPaiement  TypeTransaction = "PAIEMENT"
)

// Compte represents a bank account.
type Compte struct {
	ID         string          `json:"id" db:"id"`
	ClientID   string          `json:"client_id" db:"client_id"`
	TypeCompte TypeCompte      `json:"type_compte" db:"type_compte"`
	Solde      decimal.Decimal `json:"solde" db:"solde"`
	Devise     string          `json:"devise" db:"devise"`
	Statut     string          `json:"statut" db:"statut"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`
}

// Transaction represents a transaction on an account.
type Transaction struct {
	ID              string            `json:"id" db:"id"`
	CompteID        string            `json:"compte_id" db:"compte_id"`
	EventID         string            `json:"event_id" db:"event_id"`
	Type            TypeTransaction   `json:"type" db:"type"`
	Montant         decimal.Decimal   `json:"montant" db:"montant"`
	Devise          string            `json:"devise" db:"devise"`
	SoldeApres      decimal.Decimal   `json:"solde_apres" db:"solde_apres"`
	Reference       string            `json:"reference" db:"reference"`
	Description     string            `json:"description" db:"description"`
	CompteSourceID  *string           `json:"compte_source_id,omitempty" db:"compte_source_id"`
	CompteDestID    *string           `json:"compte_dest_id,omitempty" db:"compte_dest_id"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
}

// NewCompte creates a new Compte.
func NewCompte(id, clientID string, typeCompte TypeCompte, soldeInitial decimal.Decimal, devise string) *Compte {
	now := time.Now()
	return &Compte{
		ID:         id,
		ClientID:   clientID,
		TypeCompte: typeCompte,
		Solde:      soldeInitial,
		Devise:     devise,
		Statut:     "ACTIF",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// NewTransaction creates a new Transaction.
func NewTransaction(id, compteID, eventID string, txType TypeTransaction, montant, soldeApres decimal.Decimal, devise, reference, description string) *Transaction {
	return &Transaction{
		ID:          id,
		CompteID:    compteID,
		EventID:     eventID,
		Type:        txType,
		Montant:     montant,
		Devise:      devise,
		SoldeApres:  soldeApres,
		Reference:   reference,
		Description: description,
		CreatedAt:   time.Now(),
	}
}

// CanDebit checks if the account can be debited by the given amount.
func (c *Compte) CanDebit(montant decimal.Decimal) bool {
	return c.Solde.GreaterThanOrEqual(montant) && c.Statut == "ACTIF"
}

// Credit adds money to the account.
func (c *Compte) Credit(montant decimal.Decimal) {
	c.Solde = c.Solde.Add(montant)
	c.UpdatedAt = time.Now()
}

// Debit removes money from the account.
func (c *Compte) Debit(montant decimal.Decimal) error {
	if !c.CanDebit(montant) {
		return ErrInsufficientFunds
	}
	c.Solde = c.Solde.Sub(montant)
	c.UpdatedAt = time.Now()
	return nil
}

// Errors
var (
	ErrInsufficientFunds = &DomainError{Code: "INSUFFICIENT_FUNDS", Message: "solde insuffisant"}
	ErrCompteNotFound    = &DomainError{Code: "COMPTE_NOT_FOUND", Message: "compte non trouvé"}
	ErrCompteInactif     = &DomainError{Code: "COMPTE_INACTIF", Message: "compte inactif"}
	ErrDuplicateCompte   = &DomainError{Code: "DUPLICATE_COMPTE", Message: "compte déjà existant"}
)

// DomainError represents a domain-specific error.
type DomainError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *DomainError) Error() string {
	return e.Message
}
