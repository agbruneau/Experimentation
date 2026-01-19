// Package events defines the event types for EDA-Lab.
package events

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TypeCompte represents the type of bank account.
type TypeCompte string

const (
	TypeCompteCourant TypeCompte = "COURANT"
	TypeCompteEpargne TypeCompte = "EPARGNE"
	TypeCompteJoint   TypeCompte = "JOINT"
)

// Canal represents the channel through which a transaction was made.
type Canal string

const (
	CanalGuichet  Canal = "GUICHET"
	CanalVirement Canal = "VIREMENT"
	CanalCheque   Canal = "CHEQUE"
	CanalCarte    Canal = "CARTE"
)

// StatutVirement represents the status of a transfer.
type StatutVirement string

const (
	StatutVirementInitie   StatutVirement = "INITIE"
	StatutVirementEnCours  StatutVirement = "EN_COURS"
	StatutVirementComplete StatutVirement = "COMPLETE"
	StatutVirementRejete   StatutVirement = "REJETE"
)

// CompteOuvert represents an account opening event.
type CompteOuvert struct {
	EventID      string            `avro:"event_id" json:"event_id"`
	Timestamp    time.Time         `avro:"timestamp" json:"timestamp"`
	CompteID     string            `avro:"compte_id" json:"compte_id"`
	ClientID     string            `avro:"client_id" json:"client_id"`
	TypeCompte   TypeCompte        `avro:"type_compte" json:"type_compte"`
	Devise       string            `avro:"devise" json:"devise"`
	SoldeInitial decimal.Decimal   `avro:"solde_initial" json:"solde_initial"`
	Metadata     map[string]string `avro:"metadata" json:"metadata,omitempty"`
}

// NewCompteOuvert creates a new CompteOuvert event.
func NewCompteOuvert(compteID, clientID string, typeCompte TypeCompte, soldeInitial decimal.Decimal) *CompteOuvert {
	return &CompteOuvert{
		EventID:      uuid.New().String(),
		Timestamp:    time.Now(),
		CompteID:     compteID,
		ClientID:     clientID,
		TypeCompte:   typeCompte,
		Devise:       "EUR",
		SoldeInitial: soldeInitial,
	}
}

// CompteFerme represents an account closure event.
type CompteFerme struct {
	EventID    string            `avro:"event_id" json:"event_id"`
	Timestamp  time.Time         `avro:"timestamp" json:"timestamp"`
	CompteID   string            `avro:"compte_id" json:"compte_id"`
	ClientID   string            `avro:"client_id" json:"client_id"`
	SoldeFinal decimal.Decimal   `avro:"solde_final" json:"solde_final"`
	Motif      string            `avro:"motif" json:"motif"`
	Metadata   map[string]string `avro:"metadata" json:"metadata,omitempty"`
}

// NewCompteFerme creates a new CompteFerme event.
func NewCompteFerme(compteID, clientID string, soldeFinal decimal.Decimal, motif string) *CompteFerme {
	return &CompteFerme{
		EventID:    uuid.New().String(),
		Timestamp:  time.Now(),
		CompteID:   compteID,
		ClientID:   clientID,
		SoldeFinal: soldeFinal,
		Motif:      motif,
	}
}

// DepotEffectue represents a deposit event.
type DepotEffectue struct {
	EventID   string            `avro:"event_id" json:"event_id"`
	Timestamp time.Time         `avro:"timestamp" json:"timestamp"`
	CompteID  string            `avro:"compte_id" json:"compte_id"`
	Montant   decimal.Decimal   `avro:"montant" json:"montant"`
	Devise    string            `avro:"devise" json:"devise"`
	Reference string            `avro:"reference" json:"reference"`
	Canal     Canal             `avro:"canal" json:"canal"`
	Metadata  map[string]string `avro:"metadata" json:"metadata,omitempty"`
}

// NewDepotEffectue creates a new DepotEffectue event.
func NewDepotEffectue(compteID string, montant decimal.Decimal, canal Canal, reference string) *DepotEffectue {
	return &DepotEffectue{
		EventID:   uuid.New().String(),
		Timestamp: time.Now(),
		CompteID:  compteID,
		Montant:   montant,
		Devise:    "EUR",
		Reference: reference,
		Canal:     canal,
	}
}

// RetraitEffectue represents a withdrawal event.
type RetraitEffectue struct {
	EventID   string            `avro:"event_id" json:"event_id"`
	Timestamp time.Time         `avro:"timestamp" json:"timestamp"`
	CompteID  string            `avro:"compte_id" json:"compte_id"`
	Montant   decimal.Decimal   `avro:"montant" json:"montant"`
	Devise    string            `avro:"devise" json:"devise"`
	Reference string            `avro:"reference" json:"reference"`
	Canal     Canal             `avro:"canal" json:"canal"`
	Metadata  map[string]string `avro:"metadata" json:"metadata,omitempty"`
}

// NewRetraitEffectue creates a new RetraitEffectue event.
func NewRetraitEffectue(compteID string, montant decimal.Decimal, canal Canal, reference string) *RetraitEffectue {
	return &RetraitEffectue{
		EventID:   uuid.New().String(),
		Timestamp: time.Now(),
		CompteID:  compteID,
		Montant:   montant,
		Devise:    "EUR",
		Reference: reference,
		Canal:     canal,
	}
}

// VirementEmis represents an outgoing transfer event.
type VirementEmis struct {
	EventID            string            `avro:"event_id" json:"event_id"`
	Timestamp          time.Time         `avro:"timestamp" json:"timestamp"`
	CompteSourceID     string            `avro:"compte_source_id" json:"compte_source_id"`
	CompteDestinationID string           `avro:"compte_destination_id" json:"compte_destination_id"`
	Montant            decimal.Decimal   `avro:"montant" json:"montant"`
	Devise             string            `avro:"devise" json:"devise"`
	Motif              string            `avro:"motif" json:"motif"`
	Reference          string            `avro:"reference" json:"reference"`
	Statut             StatutVirement    `avro:"statut" json:"statut"`
	Metadata           map[string]string `avro:"metadata" json:"metadata,omitempty"`
}

// NewVirementEmis creates a new VirementEmis event.
func NewVirementEmis(compteSourceID, compteDestinationID string, montant decimal.Decimal, motif, reference string) *VirementEmis {
	return &VirementEmis{
		EventID:             uuid.New().String(),
		Timestamp:           time.Now(),
		CompteSourceID:      compteSourceID,
		CompteDestinationID: compteDestinationID,
		Montant:             montant,
		Devise:              "EUR",
		Motif:               motif,
		Reference:           reference,
		Statut:              StatutVirementInitie,
	}
}

// VirementRecu represents an incoming transfer event.
type VirementRecu struct {
	EventID            string            `avro:"event_id" json:"event_id"`
	Timestamp          time.Time         `avro:"timestamp" json:"timestamp"`
	CompteSourceID     string            `avro:"compte_source_id" json:"compte_source_id"`
	CompteDestinationID string           `avro:"compte_destination_id" json:"compte_destination_id"`
	Montant            decimal.Decimal   `avro:"montant" json:"montant"`
	Devise             string            `avro:"devise" json:"devise"`
	Motif              string            `avro:"motif" json:"motif"`
	Reference          string            `avro:"reference" json:"reference"`
	Metadata           map[string]string `avro:"metadata" json:"metadata,omitempty"`
}

// NewVirementRecu creates a new VirementRecu event.
func NewVirementRecu(compteSourceID, compteDestinationID string, montant decimal.Decimal, motif, reference string) *VirementRecu {
	return &VirementRecu{
		EventID:             uuid.New().String(),
		Timestamp:           time.Now(),
		CompteSourceID:      compteSourceID,
		CompteDestinationID: compteDestinationID,
		Montant:             montant,
		Devise:              "EUR",
		Motif:               motif,
		Reference:           reference,
	}
}

// PaiementPrimeEffectue represents an insurance premium payment event.
type PaiementPrimeEffectue struct {
	EventID      string            `avro:"event_id" json:"event_id"`
	Timestamp    time.Time         `avro:"timestamp" json:"timestamp"`
	CompteID     string            `avro:"compte_id" json:"compte_id"`
	ContratID    string            `avro:"contrat_id" json:"contrat_id"`
	Montant      decimal.Decimal   `avro:"montant" json:"montant"`
	Devise       string            `avro:"devise" json:"devise"`
	Reference    string            `avro:"reference" json:"reference"`
	TypeContrat  string            `avro:"type_contrat" json:"type_contrat"`
	Metadata     map[string]string `avro:"metadata" json:"metadata,omitempty"`
}

// NewPaiementPrimeEffectue creates a new PaiementPrimeEffectue event.
func NewPaiementPrimeEffectue(compteID, contratID string, montant decimal.Decimal, typeContrat, reference string) *PaiementPrimeEffectue {
	return &PaiementPrimeEffectue{
		EventID:     uuid.New().String(),
		Timestamp:   time.Now(),
		CompteID:    compteID,
		ContratID:   contratID,
		Montant:     montant,
		Devise:      "EUR",
		Reference:   reference,
		TypeContrat: typeContrat,
	}
}

// Topic constants for Kafka topics.
const (
	TopicCompteOuvert         = "bancaire.compte.ouvert"
	TopicCompteFerme          = "bancaire.compte.ferme"
	TopicDepotEffectue        = "bancaire.depot.effectue"
	TopicRetraitEffectue      = "bancaire.retrait.effectue"
	TopicVirementEmis         = "bancaire.virement.emis"
	TopicVirementRecu         = "bancaire.virement.recu"
	TopicPaiementPrimeEffectue = "bancaire.paiement-prime.effectue"
	TopicSystemDLQ            = "system.dlq"
)

// EventType returns the event type name.
func (e *CompteOuvert) EventType() string         { return "CompteOuvert" }
func (e *CompteFerme) EventType() string          { return "CompteFerme" }
func (e *DepotEffectue) EventType() string        { return "DepotEffectue" }
func (e *RetraitEffectue) EventType() string      { return "RetraitEffectue" }
func (e *VirementEmis) EventType() string         { return "VirementEmis" }
func (e *VirementRecu) EventType() string         { return "VirementRecu" }
func (e *PaiementPrimeEffectue) EventType() string { return "PaiementPrimeEffectue" }

// Topic returns the Kafka topic for the event.
func (e *CompteOuvert) Topic() string         { return TopicCompteOuvert }
func (e *CompteFerme) Topic() string          { return TopicCompteFerme }
func (e *DepotEffectue) Topic() string        { return TopicDepotEffectue }
func (e *RetraitEffectue) Topic() string      { return TopicRetraitEffectue }
func (e *VirementEmis) Topic() string         { return TopicVirementEmis }
func (e *VirementRecu) Topic() string         { return TopicVirementRecu }
func (e *PaiementPrimeEffectue) Topic() string { return TopicPaiementPrimeEffectue }

// Event is the interface for all events.
type Event interface {
	EventType() string
	Topic() string
}
