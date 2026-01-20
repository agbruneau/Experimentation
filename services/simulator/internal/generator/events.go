package generator

import (
	"context"
	"time"

	"github.com/edalab/pkg/events"
	"github.com/edalab/pkg/kafka"
	"github.com/shopspring/decimal"
)

// EventGenerator generates banking events
type EventGenerator struct {
	fakeData *FakeDataGenerator
	producer kafka.Producer
}

// NewEventGenerator creates a new event generator
func NewEventGenerator(producer kafka.Producer) *EventGenerator {
	return &EventGenerator{
		fakeData: NewFakeDataGeneratorRandom(),
		producer: producer,
	}
}

// NewEventGeneratorWithSeed creates a generator with a specific seed for testing
func NewEventGeneratorWithSeed(producer kafka.Producer, seed int64) *EventGenerator {
	return &EventGenerator{
		fakeData: NewFakeDataGenerator(seed),
		producer: producer,
	}
}

// GenerateCompteOuvert generates and produces a CompteOuvert event
func (g *EventGenerator) GenerateCompteOuvert(ctx context.Context) (*events.CompteOuvert, error) {
	event := &events.CompteOuvert{
		EventID:      g.fakeData.GenerateEventID(),
		Timestamp:    time.Now(),
		CompteID:     g.fakeData.GenerateCompteID(),
		ClientID:     g.fakeData.GenerateClientID(),
		TypeCompte:   g.randomTypeCompte(),
		Devise:       "EUR",
		SoldeInitial: g.fakeData.GenerateMontant(0, 10000),
		Metadata: map[string]string{
			"source": "simulator",
			"nom":    g.fakeData.GenerateNom(),
			"prenom": g.fakeData.GeneratePrenom(),
		},
	}

	if err := g.producer.Produce(ctx, events.TopicCompteOuvert, event.CompteID, event); err != nil {
		return nil, err
	}

	return event, nil
}

// GenerateDepotEffectue generates and produces a DepotEffectue event
func (g *EventGenerator) GenerateDepotEffectue(ctx context.Context, compteID string) (*events.DepotEffectue, error) {
	if compteID == "" {
		compteID = g.fakeData.GenerateCompteID()
	}

	event := &events.DepotEffectue{
		EventID:   g.fakeData.GenerateEventID(),
		Timestamp: time.Now(),
		CompteID:  compteID,
		Montant:   g.fakeData.GenerateMontant(10, 5000),
		Devise:    "EUR",
		Reference: g.fakeData.GenerateReference(),
		Canal:     g.randomCanal(),
		Metadata: map[string]string{
			"source": "simulator",
		},
	}

	if err := g.producer.Produce(ctx, events.TopicDepotEffectue, event.CompteID, event); err != nil {
		return nil, err
	}

	return event, nil
}

// GenerateRetraitEffectue generates and produces a RetraitEffectue event
func (g *EventGenerator) GenerateRetraitEffectue(ctx context.Context, compteID string) (*events.RetraitEffectue, error) {
	if compteID == "" {
		compteID = g.fakeData.GenerateCompteID()
	}

	event := &events.RetraitEffectue{
		EventID:   g.fakeData.GenerateEventID(),
		Timestamp: time.Now(),
		CompteID:  compteID,
		Montant:   g.fakeData.GenerateMontant(10, 500),
		Devise:    "EUR",
		Reference: g.fakeData.GenerateReference(),
		Canal:     g.randomCanalRetrait(),
		Metadata: map[string]string{
			"source": "simulator",
		},
	}

	if err := g.producer.Produce(ctx, events.TopicRetraitEffectue, event.CompteID, event); err != nil {
		return nil, err
	}

	return event, nil
}

// GenerateVirementEmis generates and produces a VirementEmis event
func (g *EventGenerator) GenerateVirementEmis(ctx context.Context, compteSourceID, compteDestID string) (*events.VirementEmis, error) {
	if compteSourceID == "" {
		compteSourceID = g.fakeData.GenerateCompteID()
	}
	if compteDestID == "" {
		compteDestID = g.fakeData.GenerateCompteID()
	}

	event := &events.VirementEmis{
		EventID:             g.fakeData.GenerateEventID(),
		Timestamp:           time.Now(),
		CompteSourceID:      compteSourceID,
		CompteDestinationID: compteDestID,
		Montant:             g.fakeData.GenerateMontant(10, 2000),
		Devise:              "EUR",
		Motif:               g.fakeData.GenerateMotif(),
		Reference:           g.fakeData.GenerateReference(),
		Statut:              events.StatutVirementInitie,
		Metadata: map[string]string{
			"source": "simulator",
		},
	}

	if err := g.producer.Produce(ctx, events.TopicVirementEmis, event.CompteSourceID, event); err != nil {
		return nil, err
	}

	return event, nil
}

// GenerateRandomEvent generates a random event type
func (g *EventGenerator) GenerateRandomEvent(ctx context.Context) (interface{}, error) {
	eventTypes := []string{"CompteOuvert", "DepotEffectue", "RetraitEffectue", "VirementEmis"}
	weights := []int{30, 40, 20, 10} // Weighted distribution

	// Calculate total weight
	total := 0
	for _, w := range weights {
		total += w
	}

	// Random selection based on weights
	r := g.fakeData.RandomInt(0, total)
	cumulative := 0
	selectedIndex := 0
	for i, w := range weights {
		cumulative += w
		if r < cumulative {
			selectedIndex = i
			break
		}
	}

	switch eventTypes[selectedIndex] {
	case "CompteOuvert":
		return g.GenerateCompteOuvert(ctx)
	case "DepotEffectue":
		return g.GenerateDepotEffectue(ctx, "")
	case "RetraitEffectue":
		return g.GenerateRetraitEffectue(ctx, "")
	case "VirementEmis":
		return g.GenerateVirementEmis(ctx, "", "")
	default:
		return g.GenerateCompteOuvert(ctx)
	}
}

// Helper functions for random enum values
func (g *EventGenerator) randomTypeCompte() events.TypeCompte {
	types := []events.TypeCompte{
		events.TypeCompteCourant,
		events.TypeCompteEpargne,
		events.TypeCompteJoint,
	}
	return types[g.fakeData.RandomInt(0, len(types))]
}

func (g *EventGenerator) randomCanal() events.Canal {
	canaux := []events.Canal{
		events.CanalGuichet,
		events.CanalVirement,
		events.CanalCheque,
		events.CanalCarte,
		events.CanalEnLigne,
	}
	return canaux[g.fakeData.RandomInt(0, len(canaux))]
}

func (g *EventGenerator) randomCanalRetrait() events.CanalRetrait {
	canaux := []events.CanalRetrait{
		events.CanalRetraitGuichet,
		events.CanalRetraitDAB,
		events.CanalRetraitCarte,
		events.CanalRetraitEnLigne,
	}
	return canaux[g.fakeData.RandomInt(0, len(canaux))]
}

// GenerateBatchCompteOuvert generates multiple CompteOuvert events
func (g *EventGenerator) GenerateBatchCompteOuvert(ctx context.Context, count int) ([]*events.CompteOuvert, error) {
	results := make([]*events.CompteOuvert, 0, count)
	for i := 0; i < count; i++ {
		event, err := g.GenerateCompteOuvert(ctx)
		if err != nil {
			return results, err
		}
		results = append(results, event)

		// Check context cancellation
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}
	}
	return results, nil
}

// Utility function to get zero decimal
func zeroDecimal() decimal.Decimal {
	return decimal.NewFromInt(0)
}
