package generator

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/shopspring/decimal"
)

// Common French first names
var firstNames = []string{
	"Jean", "Pierre", "Marie", "Anne", "Paul", "Jacques", "Michel", "François",
	"Louis", "Philippe", "Sophie", "Catherine", "Isabelle", "Nathalie", "Claire",
	"Thomas", "Nicolas", "Alexandre", "Julien", "Maxime", "Emma", "Léa", "Chloé",
}

// Common French last names
var lastNames = []string{
	"Martin", "Bernard", "Dubois", "Thomas", "Robert", "Richard", "Petit", "Durand",
	"Leroy", "Moreau", "Simon", "Laurent", "Lefebvre", "Michel", "Garcia", "David",
	"Bertrand", "Roux", "Vincent", "Fournier", "Morel", "Girard", "André", "Mercier",
}

// FakeDataGenerator generates realistic fake data
type FakeDataGenerator struct {
	rng *rand.Rand
}

// NewFakeDataGenerator creates a new fake data generator
func NewFakeDataGenerator(seed int64) *FakeDataGenerator {
	return &FakeDataGenerator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// NewFakeDataGeneratorRandom creates a generator with random seed
func NewFakeDataGeneratorRandom() *FakeDataGenerator {
	return NewFakeDataGenerator(time.Now().UnixNano())
}

// GenerateClientID generates a unique client ID
func (g *FakeDataGenerator) GenerateClientID() string {
	return fmt.Sprintf("CLI-%08d", g.rng.Intn(100000000))
}

// GenerateCompteID generates a unique account ID
func (g *FakeDataGenerator) GenerateCompteID() string {
	return fmt.Sprintf("CPT-%08d", g.rng.Intn(100000000))
}

// GenerateEventID generates a UUID-like event ID
func (g *FakeDataGenerator) GenerateEventID() string {
	b := make([]byte, 16)
	g.rng.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// GenerateNom generates a random last name
func (g *FakeDataGenerator) GenerateNom() string {
	return lastNames[g.rng.Intn(len(lastNames))]
}

// GeneratePrenom generates a random first name
func (g *FakeDataGenerator) GeneratePrenom() string {
	return firstNames[g.rng.Intn(len(firstNames))]
}

// GenerateMontant generates a random amount in the specified range
func (g *FakeDataGenerator) GenerateMontant(min, max float64) decimal.Decimal {
	amount := min + g.rng.Float64()*(max-min)
	return decimal.NewFromFloat(amount).Round(2)
}

// GenerateIBAN generates a valid French IBAN
func (g *FakeDataGenerator) GenerateIBAN() string {
	// French IBAN format: FR + 2 check digits + 5 bank code + 5 branch code + 11 account + 2 key
	bankCode := fmt.Sprintf("%05d", g.rng.Intn(100000))
	branchCode := fmt.Sprintf("%05d", g.rng.Intn(100000))
	accountNum := fmt.Sprintf("%011d", g.rng.Int63n(100000000000))
	ribKey := fmt.Sprintf("%02d", g.rng.Intn(100))

	// BBAN without check digits
	bban := bankCode + branchCode + accountNum + ribKey

	// Calculate check digits (simplified - not full MOD-97 calculation)
	checkDigits := 97 - (g.rng.Intn(85) + 10)

	return fmt.Sprintf("FR%02d%s", checkDigits, bban)
}

// GenerateReference generates a unique reference
func (g *FakeDataGenerator) GenerateReference() string {
	return fmt.Sprintf("REF-%s-%06d",
		time.Now().Format("20060102"),
		g.rng.Intn(1000000))
}

// GenerateMotif generates a random transfer motif
func (g *FakeDataGenerator) GenerateMotif() string {
	motifs := []string{
		"Virement mensuel",
		"Paiement facture",
		"Remboursement",
		"Transfert épargne",
		"Paiement loyer",
		"Salaire",
		"Prime",
		"Don familial",
	}
	return motifs[g.rng.Intn(len(motifs))]
}

// RandomChoice returns a random element from a slice
func (g *FakeDataGenerator) RandomChoice(choices []string) string {
	return choices[g.rng.Intn(len(choices))]
}

// RandomInt returns a random int in range [min, max)
func (g *FakeDataGenerator) RandomInt(min, max int) int {
	return min + g.rng.Intn(max-min)
}

// RandomFloat returns a random float64 in range [min, max)
func (g *FakeDataGenerator) RandomFloat(min, max float64) float64 {
	return min + g.rng.Float64()*(max-min)
}
