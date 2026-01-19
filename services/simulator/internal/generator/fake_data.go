// Package generator provides fake data generation for EDA-Lab simulations.
package generator

import (
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Common French first names
var prenoms = []string{
	"Jean", "Pierre", "Marie", "Anne", "Michel", "Françoise", "Philippe", "Isabelle",
	"Jacques", "Catherine", "Bernard", "Nathalie", "Patrick", "Christine", "Nicolas",
	"Sophie", "Laurent", "Sandrine", "Alain", "Valérie", "Christophe", "Céline",
	"Thierry", "Stéphanie", "Eric", "Véronique", "Olivier", "Sylvie", "David", "Julie",
	"Marc", "Aurélie", "Guillaume", "Caroline", "Julien", "Emilie", "Thomas", "Claire",
	"Sébastien", "Laure", "Antoine", "Camille", "Mathieu", "Pauline", "Alexandre", "Marine",
	"Benjamin", "Lucie", "Romain", "Emma", "Maxime", "Léa", "Hugo", "Chloé",
}

// Common French last names
var noms = []string{
	"Martin", "Bernard", "Thomas", "Petit", "Robert", "Richard", "Durand", "Dubois",
	"Moreau", "Laurent", "Simon", "Michel", "Lefebvre", "Leroy", "Roux", "David",
	"Bertrand", "Morel", "Fournier", "Girard", "Bonnet", "Dupont", "Lambert", "Fontaine",
	"Rousseau", "Vincent", "Muller", "Lefevre", "Faure", "Andre", "Mercier", "Blanc",
	"Guerin", "Boyer", "Garnier", "Chevalier", "Francois", "Legrand", "Gauthier", "Garcia",
	"Perrin", "Robin", "Clement", "Morin", "Nicolas", "Henry", "Roussel", "Mathieu",
	"Gautier", "Masson", "Marchand", "Duval", "Denis", "Dumont", "Marie", "Lemaire",
}

// French bank codes (BIC)
var bankCodes = []string{
	"30002", // Crédit Lyonnais
	"30003", // Société Générale
	"30004", // BNP Paribas
	"30006", // CIC
	"30007", // Natixis
	"12506", // Crédit Agricole
	"10207", // Crédit du Nord
	"17515", // La Banque Postale
	"14505", // CM-CIC
	"18706", // HSBC France
}

// FakeDataGenerator generates realistic fake data for banking simulations.
type FakeDataGenerator struct {
	rng *rand.Rand
}

// NewFakeDataGenerator creates a new generator with the given seed.
// Use time.Now().UnixNano() for random behavior, or a fixed seed for reproducibility.
func NewFakeDataGenerator(seed int64) *FakeDataGenerator {
	return &FakeDataGenerator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// NewRandomFakeDataGenerator creates a new generator with a random seed.
func NewRandomFakeDataGenerator() *FakeDataGenerator {
	return NewFakeDataGenerator(time.Now().UnixNano())
}

// GenerateClientID generates a unique client identifier.
func (g *FakeDataGenerator) GenerateClientID() string {
	return fmt.Sprintf("CLI-%s", uuid.New().String()[:8])
}

// GenerateCompteID generates a unique account identifier.
func (g *FakeDataGenerator) GenerateCompteID() string {
	return fmt.Sprintf("CPT-%s", uuid.New().String()[:12])
}

// GenerateNom generates a random French last name.
func (g *FakeDataGenerator) GenerateNom() string {
	return noms[g.rng.Intn(len(noms))]
}

// GeneratePrenom generates a random French first name.
func (g *FakeDataGenerator) GeneratePrenom() string {
	return prenoms[g.rng.Intn(len(prenoms))]
}

// GenerateNomComplet generates a full name (first + last).
func (g *FakeDataGenerator) GenerateNomComplet() string {
	return g.GeneratePrenom() + " " + g.GenerateNom()
}

// GenerateMontant generates a random amount between min and max.
func (g *FakeDataGenerator) GenerateMontant(min, max float64) decimal.Decimal {
	// Generate a random value in the range
	value := min + g.rng.Float64()*(max-min)
	// Round to 2 decimal places
	return decimal.NewFromFloat(value).Round(2)
}

// GenerateSoldeInitial generates a realistic initial balance for a new account.
// Most accounts start with a modest deposit, with some having larger amounts.
func (g *FakeDataGenerator) GenerateSoldeInitial() decimal.Decimal {
	// 60% chance of small deposit (10-500€)
	// 30% chance of medium deposit (500-5000€)
	// 10% chance of large deposit (5000-50000€)
	r := g.rng.Float64()
	switch {
	case r < 0.6:
		return g.GenerateMontant(10, 500)
	case r < 0.9:
		return g.GenerateMontant(500, 5000)
	default:
		return g.GenerateMontant(5000, 50000)
	}
}

// GenerateIBAN generates a valid French IBAN.
// Format: FR + 2 check digits + 5 bank code + 5 branch code + 11 account number + 2 RIB key
func (g *FakeDataGenerator) GenerateIBAN() string {
	// Bank code (5 digits)
	bankCode := bankCodes[g.rng.Intn(len(bankCodes))]

	// Branch code (5 digits)
	branchCode := fmt.Sprintf("%05d", g.rng.Intn(100000))

	// Account number (11 alphanumeric)
	accountNum := g.generateAccountNumber()

	// Calculate RIB key
	ribKey := g.calculateRIBKey(bankCode, branchCode, accountNum)

	// Build BBAN (Basic Bank Account Number)
	bban := bankCode + branchCode + accountNum + ribKey

	// Calculate IBAN check digits
	checkDigits := g.calculateIBANCheckDigits("FR", bban)

	return "FR" + checkDigits + bban
}

// generateAccountNumber generates an 11-character account number.
func (g *FakeDataGenerator) generateAccountNumber() string {
	chars := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var result strings.Builder
	for i := 0; i < 11; i++ {
		result.WriteByte(chars[g.rng.Intn(len(chars))])
	}
	return result.String()
}

// calculateRIBKey calculates the French RIB key (clé RIB).
func (g *FakeDataGenerator) calculateRIBKey(bankCode, branchCode, accountNum string) string {
	// Convert letters to numbers for calculation
	numericAccount := g.convertLettersToNumbers(accountNum)

	// Parse components
	bank, _ := new(big.Int).SetString(bankCode, 10)
	branch, _ := new(big.Int).SetString(branchCode, 10)
	account, _ := new(big.Int).SetString(numericAccount, 10)

	// Key = 97 - ((89 * bank + 15 * branch + 3 * account) mod 97)
	bigVal := new(big.Int)
	bigVal.Mul(big.NewInt(89), bank)
	bigVal.Add(bigVal, new(big.Int).Mul(big.NewInt(15), branch))
	bigVal.Add(bigVal, new(big.Int).Mul(big.NewInt(3), account))
	bigVal.Mod(bigVal, big.NewInt(97))

	key := 97 - bigVal.Int64()
	return fmt.Sprintf("%02d", key)
}

// convertLettersToNumbers converts letters to their numeric equivalent for RIB calculation.
// A-I = 1-9, J-R = 1-9, S-Z = 2-9
func (g *FakeDataGenerator) convertLettersToNumbers(s string) string {
	var result strings.Builder
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result.WriteRune(c)
		} else if c >= 'A' && c <= 'I' {
			result.WriteRune('1' + (c - 'A'))
		} else if c >= 'J' && c <= 'R' {
			result.WriteRune('1' + (c - 'J'))
		} else if c >= 'S' && c <= 'Z' {
			result.WriteRune('2' + (c - 'S'))
		}
	}
	return result.String()
}

// calculateIBANCheckDigits calculates the IBAN check digits.
func (g *FakeDataGenerator) calculateIBANCheckDigits(countryCode, bban string) string {
	// Move country code and 00 to end
	rearranged := bban + g.countryCodeToDigits(countryCode) + "00"

	// Calculate mod 97
	num := new(big.Int)
	num.SetString(rearranged, 10)
	remainder := new(big.Int).Mod(num, big.NewInt(97))

	// Check digits = 98 - remainder
	checkDigits := 98 - remainder.Int64()
	return fmt.Sprintf("%02d", checkDigits)
}

// countryCodeToDigits converts country code letters to digits (A=10, B=11, etc.).
func (g *FakeDataGenerator) countryCodeToDigits(code string) string {
	var result strings.Builder
	for _, c := range code {
		if c >= 'A' && c <= 'Z' {
			result.WriteString(fmt.Sprintf("%d", c-'A'+10))
		}
	}
	return result.String()
}

// GenerateReference generates a transaction reference.
func (g *FakeDataGenerator) GenerateReference() string {
	return fmt.Sprintf("REF%d%06d", time.Now().Unix()%1000000, g.rng.Intn(1000000))
}

// GenerateMotifVirement generates a transfer reason.
func (g *FakeDataGenerator) GenerateMotifVirement() string {
	motifs := []string{
		"Loyer mensuel",
		"Remboursement",
		"Facture",
		"Virement familial",
		"Achat en ligne",
		"Abonnement",
		"Salaire",
		"Epargne mensuelle",
		"Frais divers",
		"Paiement prestation",
	}
	return motifs[g.rng.Intn(len(motifs))]
}

// GenerateEmail generates an email address based on a name.
func (g *FakeDataGenerator) GenerateEmail(prenom, nom string) string {
	domains := []string{"gmail.com", "orange.fr", "free.fr", "sfr.fr", "outlook.fr", "yahoo.fr"}
	formats := []string{
		"%s.%s@%s",
		"%s_%s@%s",
		"%s%s@%s",
		"%s.%s%d@%s",
	}

	prenom = strings.ToLower(strings.ReplaceAll(prenom, " ", ""))
	nom = strings.ToLower(strings.ReplaceAll(nom, " ", ""))
	domain := domains[g.rng.Intn(len(domains))]
	format := formats[g.rng.Intn(len(formats))]

	if strings.Contains(format, "%d") {
		return fmt.Sprintf(format, prenom, nom, g.rng.Intn(100), domain)
	}
	return fmt.Sprintf(format, prenom, nom, domain)
}

// GeneratePhoneNumber generates a French mobile phone number.
func (g *FakeDataGenerator) GeneratePhoneNumber() string {
	prefixes := []string{"06", "07"}
	prefix := prefixes[g.rng.Intn(len(prefixes))]
	return fmt.Sprintf("+33%s%08d", prefix[1:], g.rng.Intn(100000000))
}
