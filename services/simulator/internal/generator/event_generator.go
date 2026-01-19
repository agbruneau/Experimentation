package generator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/edalab/pkg/events"
	"github.com/edalab/pkg/kafka"
	"github.com/edalab/pkg/observability"
	"github.com/shopspring/decimal"
)

// EventGenerator defines the interface for event generators.
type EventGenerator interface {
	Generate(ctx context.Context) (events.Event, error)
	GenerateBatch(ctx context.Context, count int, interval time.Duration) ([]events.Event, error)
	EventType() string
}

// CompteOuvertGenerator generates CompteOuvert events.
type CompteOuvertGenerator struct {
	fakeData *FakeDataGenerator
	producer kafka.Producer
	logger   *slog.Logger
	metrics  *observability.Metrics
	service  string
}

// NewCompteOuvertGenerator creates a new CompteOuvert event generator.
func NewCompteOuvertGenerator(
	fakeData *FakeDataGenerator,
	producer kafka.Producer,
	logger *slog.Logger,
	metrics *observability.Metrics,
	service string,
) *CompteOuvertGenerator {
	return &CompteOuvertGenerator{
		fakeData: fakeData,
		producer: producer,
		logger:   logger,
		metrics:  metrics,
		service:  service,
	}
}

// EventType returns the type of events this generator creates.
func (g *CompteOuvertGenerator) EventType() string {
	return "CompteOuvert"
}

// Generate creates and produces a single CompteOuvert event.
func (g *CompteOuvertGenerator) Generate(ctx context.Context) (events.Event, error) {
	// Generate fake data
	compteID := g.fakeData.GenerateCompteID()
	clientID := g.fakeData.GenerateClientID()
	soldeInitial := g.fakeData.GenerateSoldeInitial()

	// Choose account type with distribution: 70% COURANT, 20% EPARGNE, 10% JOINT
	var typeCompte events.TypeCompte
	r := g.fakeData.rng.Float64()
	switch {
	case r < 0.7:
		typeCompte = events.TypeCompteCourant
	case r < 0.9:
		typeCompte = events.TypeCompteEpargne
	default:
		typeCompte = events.TypeCompteJoint
	}

	// Create event
	event := events.NewCompteOuvert(compteID, clientID, typeCompte, soldeInitial)

	// Add metadata
	event.Metadata = map[string]string{
		"nom_client":     g.fakeData.GenerateNomComplet(),
		"email":          g.fakeData.GenerateEmail(g.fakeData.GeneratePrenom(), g.fakeData.GenerateNom()),
		"telephone":      g.fakeData.GeneratePhoneNumber(),
		"source":         "simulator",
		"simulation_time": time.Now().Format(time.RFC3339),
	}

	// Produce to Kafka
	timer := observability.NewTimer()
	err := g.producer.Produce(ctx, events.TopicCompteOuvert, compteID, event)
	if err != nil {
		if g.metrics != nil {
			g.metrics.RecordMessageFailed(g.service, events.TopicCompteOuvert, "produce_error")
		}
		return nil, fmt.Errorf("failed to produce CompteOuvert event: %w", err)
	}

	// Record metrics
	if g.metrics != nil {
		g.metrics.RecordMessageProduced(g.service, events.TopicCompteOuvert)
		g.metrics.RecordMessageLatency(g.service, events.TopicCompteOuvert, timer.Elapsed())
	}

	// Log
	if g.logger != nil {
		g.logger.Info("CompteOuvert event produced",
			slog.String("event_id", event.EventID),
			slog.String("compte_id", compteID),
			slog.String("client_id", clientID),
			slog.String("type_compte", string(typeCompte)),
			slog.String("solde_initial", soldeInitial.String()),
		)
	}

	return event, nil
}

// GenerateBatch generates multiple CompteOuvert events with the specified interval.
func (g *CompteOuvertGenerator) GenerateBatch(ctx context.Context, count int, interval time.Duration) ([]events.Event, error) {
	generated := make([]events.Event, 0, count)

	for i := 0; i < count; i++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return generated, ctx.Err()
		default:
		}

		event, err := g.Generate(ctx)
		if err != nil {
			g.logger.Error("failed to generate event",
				slog.Int("index", i),
				slog.Any("error", err),
			)
			continue
		}
		generated = append(generated, event)

		// Wait for interval (except for last event)
		if i < count-1 && interval > 0 {
			select {
			case <-ctx.Done():
				return generated, ctx.Err()
			case <-time.After(interval):
			}
		}
	}

	return generated, nil
}

// DepotEffectueGenerator generates DepotEffectue events.
type DepotEffectueGenerator struct {
	fakeData *FakeDataGenerator
	producer kafka.Producer
	logger   *slog.Logger
	metrics  *observability.Metrics
	service  string
}

// NewDepotEffectueGenerator creates a new DepotEffectue event generator.
func NewDepotEffectueGenerator(
	fakeData *FakeDataGenerator,
	producer kafka.Producer,
	logger *slog.Logger,
	metrics *observability.Metrics,
	service string,
) *DepotEffectueGenerator {
	return &DepotEffectueGenerator{
		fakeData: fakeData,
		producer: producer,
		logger:   logger,
		metrics:  metrics,
		service:  service,
	}
}

// EventType returns the type of events this generator creates.
func (g *DepotEffectueGenerator) EventType() string {
	return "DepotEffectue"
}

// GenerateForCompte creates a DepotEffectue event for a specific account.
func (g *DepotEffectueGenerator) GenerateForCompte(ctx context.Context, compteID string) (events.Event, error) {
	// Generate fake data
	montant := g.fakeData.GenerateMontant(10, 5000)
	reference := g.fakeData.GenerateReference()

	// Choose canal with distribution
	var canal events.Canal
	r := g.fakeData.rng.Float64()
	switch {
	case r < 0.4:
		canal = events.CanalVirement
	case r < 0.7:
		canal = events.CanalGuichet
	case r < 0.9:
		canal = events.CanalCarte
	default:
		canal = events.CanalCheque
	}

	// Create event
	event := events.NewDepotEffectue(compteID, montant, canal, reference)

	// Produce to Kafka
	timer := observability.NewTimer()
	err := g.producer.Produce(ctx, events.TopicDepotEffectue, compteID, event)
	if err != nil {
		if g.metrics != nil {
			g.metrics.RecordMessageFailed(g.service, events.TopicDepotEffectue, "produce_error")
		}
		return nil, fmt.Errorf("failed to produce DepotEffectue event: %w", err)
	}

	// Record metrics
	if g.metrics != nil {
		g.metrics.RecordMessageProduced(g.service, events.TopicDepotEffectue)
		g.metrics.RecordMessageLatency(g.service, events.TopicDepotEffectue, timer.Elapsed())
	}

	// Log
	if g.logger != nil {
		g.logger.Info("DepotEffectue event produced",
			slog.String("event_id", event.EventID),
			slog.String("compte_id", compteID),
			slog.String("montant", montant.String()),
			slog.String("canal", string(canal)),
		)
	}

	return event, nil
}

// Generate creates a DepotEffectue event with a random account ID.
func (g *DepotEffectueGenerator) Generate(ctx context.Context) (events.Event, error) {
	return g.GenerateForCompte(ctx, g.fakeData.GenerateCompteID())
}

// GenerateBatch generates multiple DepotEffectue events.
func (g *DepotEffectueGenerator) GenerateBatch(ctx context.Context, count int, interval time.Duration) ([]events.Event, error) {
	generated := make([]events.Event, 0, count)

	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			return generated, ctx.Err()
		default:
		}

		event, err := g.Generate(ctx)
		if err != nil {
			g.logger.Error("failed to generate event",
				slog.Int("index", i),
				slog.Any("error", err),
			)
			continue
		}
		generated = append(generated, event)

		if i < count-1 && interval > 0 {
			select {
			case <-ctx.Done():
				return generated, ctx.Err()
			case <-time.After(interval):
			}
		}
	}

	return generated, nil
}

// VirementEmisGenerator generates VirementEmis events.
type VirementEmisGenerator struct {
	fakeData *FakeDataGenerator
	producer kafka.Producer
	logger   *slog.Logger
	metrics  *observability.Metrics
	service  string
}

// NewVirementEmisGenerator creates a new VirementEmis event generator.
func NewVirementEmisGenerator(
	fakeData *FakeDataGenerator,
	producer kafka.Producer,
	logger *slog.Logger,
	metrics *observability.Metrics,
	service string,
) *VirementEmisGenerator {
	return &VirementEmisGenerator{
		fakeData: fakeData,
		producer: producer,
		logger:   logger,
		metrics:  metrics,
		service:  service,
	}
}

// EventType returns the type of events this generator creates.
func (g *VirementEmisGenerator) EventType() string {
	return "VirementEmis"
}

// GenerateForComptes creates a VirementEmis event between two accounts.
func (g *VirementEmisGenerator) GenerateForComptes(ctx context.Context, sourceID, destID string, montant decimal.Decimal) (events.Event, error) {
	reference := g.fakeData.GenerateReference()
	motif := g.fakeData.GenerateMotifVirement()

	// Create event
	event := events.NewVirementEmis(sourceID, destID, montant, motif, reference)

	// Produce to Kafka
	timer := observability.NewTimer()
	err := g.producer.Produce(ctx, events.TopicVirementEmis, sourceID, event)
	if err != nil {
		if g.metrics != nil {
			g.metrics.RecordMessageFailed(g.service, events.TopicVirementEmis, "produce_error")
		}
		return nil, fmt.Errorf("failed to produce VirementEmis event: %w", err)
	}

	// Record metrics
	if g.metrics != nil {
		g.metrics.RecordMessageProduced(g.service, events.TopicVirementEmis)
		g.metrics.RecordMessageLatency(g.service, events.TopicVirementEmis, timer.Elapsed())
	}

	// Log
	if g.logger != nil {
		g.logger.Info("VirementEmis event produced",
			slog.String("event_id", event.EventID),
			slog.String("source_id", sourceID),
			slog.String("dest_id", destID),
			slog.String("montant", montant.String()),
			slog.String("motif", motif),
		)
	}

	return event, nil
}

// Generate creates a VirementEmis event with random accounts.
func (g *VirementEmisGenerator) Generate(ctx context.Context) (events.Event, error) {
	sourceID := g.fakeData.GenerateCompteID()
	destID := g.fakeData.GenerateCompteID()
	montant := g.fakeData.GenerateMontant(10, 2000)
	return g.GenerateForComptes(ctx, sourceID, destID, montant)
}

// GenerateBatch generates multiple VirementEmis events.
func (g *VirementEmisGenerator) GenerateBatch(ctx context.Context, count int, interval time.Duration) ([]events.Event, error) {
	generated := make([]events.Event, 0, count)

	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			return generated, ctx.Err()
		default:
		}

		event, err := g.Generate(ctx)
		if err != nil {
			g.logger.Error("failed to generate event",
				slog.Int("index", i),
				slog.Any("error", err),
			)
			continue
		}
		generated = append(generated, event)

		if i < count-1 && interval > 0 {
			select {
			case <-ctx.Done():
				return generated, ctx.Err()
			case <-time.After(interval):
			}
		}
	}

	return generated, nil
}

// GeneratorFactory creates event generators by type.
type GeneratorFactory struct {
	fakeData *FakeDataGenerator
	producer kafka.Producer
	logger   *slog.Logger
	metrics  *observability.Metrics
	service  string
}

// NewGeneratorFactory creates a new generator factory.
func NewGeneratorFactory(
	producer kafka.Producer,
	logger *slog.Logger,
	metrics *observability.Metrics,
	service string,
	seed int64,
) *GeneratorFactory {
	return &GeneratorFactory{
		fakeData: NewFakeDataGenerator(seed),
		producer: producer,
		logger:   logger,
		metrics:  metrics,
		service:  service,
	}
}

// Create creates a generator for the specified event type.
func (f *GeneratorFactory) Create(eventType string) (EventGenerator, error) {
	switch eventType {
	case "CompteOuvert":
		return NewCompteOuvertGenerator(f.fakeData, f.producer, f.logger, f.metrics, f.service), nil
	case "DepotEffectue":
		return NewDepotEffectueGenerator(f.fakeData, f.producer, f.logger, f.metrics, f.service), nil
	case "VirementEmis":
		return NewVirementEmisGenerator(f.fakeData, f.producer, f.logger, f.metrics, f.service), nil
	default:
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}
}

// CreateAll creates all available generators.
func (f *GeneratorFactory) CreateAll() map[string]EventGenerator {
	return map[string]EventGenerator{
		"CompteOuvert":  NewCompteOuvertGenerator(f.fakeData, f.producer, f.logger, f.metrics, f.service),
		"DepotEffectue": NewDepotEffectueGenerator(f.fakeData, f.producer, f.logger, f.metrics, f.service),
		"VirementEmis":  NewVirementEmisGenerator(f.fakeData, f.producer, f.logger, f.metrics, f.service),
	}
}
