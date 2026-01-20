package events

import (
	"time"

	"github.com/shopspring/decimal"
)

// TypeCompte defines the type of bank account
type TypeCompte string

const (
	TypeCompteCourant TypeCompte = "COURANT"
	TypeCompteEpargne TypeCompte = "EPARGNE"
	TypeCompteJoint   TypeCompte = "JOINT"
)

// Canal defines the channel for deposits
type Canal string

const (
	CanalGuichet  Canal = "GUICHET"
	CanalVirement Canal = "VIREMENT"
	CanalCheque   Canal = "CHEQUE"
	CanalCarte    Canal = "CARTE"
	CanalEnLigne  Canal = "EN_LIGNE"
)

// CanalRetrait defines the channel for withdrawals
type CanalRetrait string

const (
	CanalRetraitGuichet CanalRetrait = "GUICHET"
	CanalRetraitDAB     CanalRetrait = "DAB"
	CanalRetraitCarte   CanalRetrait = "CARTE"
	CanalRetraitEnLigne CanalRetrait = "EN_LIGNE"
)

// StatutVirement defines the status of a transfer
type StatutVirement string

const (
	StatutVirementInitie   StatutVirement = "INITIE"
	StatutVirementEnCours  StatutVirement = "EN_COURS"
	StatutVirementComplete StatutVirement = "COMPLETE"
	StatutVirementRejete   StatutVirement = "REJETE"
)

// CompteOuvert represents an account opening event
type CompteOuvert struct {
	EventID      string            `json:"event_id" avro:"event_id"`
	Timestamp    time.Time         `json:"timestamp" avro:"timestamp"`
	CompteID     string            `json:"compte_id" avro:"compte_id"`
	ClientID     string            `json:"client_id" avro:"client_id"`
	TypeCompte   TypeCompte        `json:"type_compte" avro:"type_compte"`
	Devise       string            `json:"devise" avro:"devise"`
	SoldeInitial decimal.Decimal   `json:"solde_initial" avro:"solde_initial"`
	Metadata     map[string]string `json:"metadata,omitempty" avro:"metadata"`
}

// DepotEffectue represents a deposit event
type DepotEffectue struct {
	EventID   string            `json:"event_id" avro:"event_id"`
	Timestamp time.Time         `json:"timestamp" avro:"timestamp"`
	CompteID  string            `json:"compte_id" avro:"compte_id"`
	Montant   decimal.Decimal   `json:"montant" avro:"montant"`
	Devise    string            `json:"devise" avro:"devise"`
	Reference string            `json:"reference" avro:"reference"`
	Canal     Canal             `json:"canal" avro:"canal"`
	Metadata  map[string]string `json:"metadata,omitempty" avro:"metadata"`
}

// RetraitEffectue represents a withdrawal event
type RetraitEffectue struct {
	EventID   string            `json:"event_id" avro:"event_id"`
	Timestamp time.Time         `json:"timestamp" avro:"timestamp"`
	CompteID  string            `json:"compte_id" avro:"compte_id"`
	Montant   decimal.Decimal   `json:"montant" avro:"montant"`
	Devise    string            `json:"devise" avro:"devise"`
	Reference string            `json:"reference" avro:"reference"`
	Canal     CanalRetrait      `json:"canal" avro:"canal"`
	Metadata  map[string]string `json:"metadata,omitempty" avro:"metadata"`
}

// VirementEmis represents an outgoing transfer event
type VirementEmis struct {
	EventID            string            `json:"event_id" avro:"event_id"`
	Timestamp          time.Time         `json:"timestamp" avro:"timestamp"`
	CompteSourceID     string            `json:"compte_source_id" avro:"compte_source_id"`
	CompteDestinationID string           `json:"compte_destination_id" avro:"compte_destination_id"`
	Montant            decimal.Decimal   `json:"montant" avro:"montant"`
	Devise             string            `json:"devise" avro:"devise"`
	Motif              string            `json:"motif,omitempty" avro:"motif"`
	Reference          string            `json:"reference" avro:"reference"`
	Statut             StatutVirement    `json:"statut" avro:"statut"`
	Metadata           map[string]string `json:"metadata,omitempty" avro:"metadata"`
}

// Topic names for bancaire events
const (
	TopicCompteOuvert    = "bancaire.compte.ouvert"
	TopicCompteFerme     = "bancaire.compte.ferme"
	TopicDepotEffectue   = "bancaire.depot.effectue"
	TopicRetraitEffectue = "bancaire.retrait.effectue"
	TopicVirementEmis    = "bancaire.virement.emis"
	TopicVirementRecu    = "bancaire.virement.recu"
)
