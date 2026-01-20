package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Compte represents a bank account
type Compte struct {
	ID         string          `json:"id"`
	ClientID   string          `json:"client_id"`
	TypeCompte string          `json:"type_compte"`
	Devise     string          `json:"devise"`
	Solde      decimal.Decimal `json:"solde"`
	Statut     string          `json:"statut"` // ACTIF, FERME, BLOQUE
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// Transaction represents a bank transaction
type Transaction struct {
	ID        string          `json:"id"`
	CompteID  string          `json:"compte_id"`
	Type      string          `json:"type"` // DEPOT, RETRAIT, VIREMENT_ENTRANT, VIREMENT_SORTANT
	Montant   decimal.Decimal `json:"montant"`
	Reference string          `json:"reference"`
	CreatedAt time.Time       `json:"created_at"`
}

// Account statuses
const (
	StatutActif  = "ACTIF"
	StatutFerme  = "FERME"
	StatutBloque = "BLOQUE"
)

// Transaction types
const (
	TypeDepot            = "DEPOT"
	TypeRetrait          = "RETRAIT"
	TypeVirementEntrant  = "VIREMENT_ENTRANT"
	TypeVirementSortant  = "VIREMENT_SORTANT"
	TypeOuvertureCompte  = "OUVERTURE_COMPTE"
)
