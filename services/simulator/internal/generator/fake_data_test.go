package generator

import (
	"regexp"
	"testing"

	"github.com/shopspring/decimal"
)

func TestNewFakeDataGenerator(t *testing.T) {
	gen := NewFakeDataGenerator(12345)
	if gen == nil {
		t.Fatal("expected non-nil generator")
	}
	if gen.rng == nil {
		t.Fatal("expected non-nil random source")
	}
}

func TestGenerateClientID_UniqueFormat(t *testing.T) {
	gen := NewFakeDataGenerator(12345)

	// Generate multiple IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := gen.GenerateClientID()

		// Check format: CLI-xxxxxxxx
		if len(id) != 12 {
			t.Errorf("expected client ID length 12, got %d: %s", len(id), id)
		}
		if id[:4] != "CLI-" {
			t.Errorf("expected client ID to start with 'CLI-', got: %s", id)
		}

		// Check uniqueness
		if ids[id] {
			t.Errorf("duplicate client ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestGenerateCompteID(t *testing.T) {
	gen := NewFakeDataGenerator(12345)

	id := gen.GenerateCompteID()
	if len(id) != 16 {
		t.Errorf("expected compte ID length 16, got %d: %s", len(id), id)
	}
	if id[:4] != "CPT-" {
		t.Errorf("expected compte ID to start with 'CPT-', got: %s", id)
	}
}

func TestGenerateNom(t *testing.T) {
	gen := NewFakeDataGenerator(12345)

	nom := gen.GenerateNom()
	if nom == "" {
		t.Error("expected non-empty nom")
	}

	// Verify it's from the list
	found := false
	for _, n := range noms {
		if n == nom {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("generated nom '%s' not in known list", nom)
	}
}

func TestGeneratePrenom(t *testing.T) {
	gen := NewFakeDataGenerator(12345)

	prenom := gen.GeneratePrenom()
	if prenom == "" {
		t.Error("expected non-empty prenom")
	}

	// Verify it's from the list
	found := false
	for _, p := range prenoms {
		if p == prenom {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("generated prenom '%s' not in known list", prenom)
	}
}

func TestGenerateMontant_InRange(t *testing.T) {
	gen := NewFakeDataGenerator(12345)

	tests := []struct {
		min, max float64
	}{
		{0, 100},
		{100, 1000},
		{1000, 10000},
		{0.01, 0.99},
	}

	for _, tc := range tests {
		for i := 0; i < 100; i++ {
			montant := gen.GenerateMontant(tc.min, tc.max)
			minDec := decimal.NewFromFloat(tc.min)
			maxDec := decimal.NewFromFloat(tc.max)

			if montant.LessThan(minDec) || montant.GreaterThan(maxDec) {
				t.Errorf("montant %s not in range [%f, %f]", montant, tc.min, tc.max)
			}

			// Check it has at most 2 decimal places
			str := montant.String()
			parts := regexp.MustCompile(`\.(\d+)$`).FindStringSubmatch(str)
			if len(parts) > 1 && len(parts[1]) > 2 {
				t.Errorf("montant %s has more than 2 decimal places", montant)
			}
		}
	}
}

func TestGenerateIBAN_ValidFormat(t *testing.T) {
	gen := NewFakeDataGenerator(12345)

	// French IBAN format: FR + 2 check + 23 BBAN = 27 chars
	ibanRegex := regexp.MustCompile(`^FR\d{2}\d{23}$`)

	for i := 0; i < 100; i++ {
		iban := gen.GenerateIBAN()

		if len(iban) != 27 {
			t.Errorf("expected IBAN length 27, got %d: %s", len(iban), iban)
		}

		if !ibanRegex.MatchString(iban) {
			t.Errorf("IBAN does not match expected format: %s", iban)
		}
	}
}

func TestGenerateIBAN_ValidChecksum(t *testing.T) {
	gen := NewFakeDataGenerator(12345)

	for i := 0; i < 50; i++ {
		iban := gen.GenerateIBAN()

		// Validate IBAN checksum
		if !validateIBANChecksum(iban) {
			t.Errorf("invalid IBAN checksum: %s", iban)
		}
	}
}

// validateIBANChecksum validates the IBAN check digits using the mod 97 algorithm.
func validateIBANChecksum(iban string) bool {
	// Move first 4 characters to end
	rearranged := iban[4:] + iban[:4]

	// Convert letters to digits (A=10, B=11, etc.)
	var numeric string
	for _, c := range rearranged {
		if c >= 'A' && c <= 'Z' {
			numeric += string('0' + (c-'A')/10)
			numeric += string('0' + (c-'A')%10 + 10%10)
		} else {
			numeric += string(c)
		}
	}

	// Calculate mod 97 (handle large numbers by processing in chunks)
	remainder := 0
	for _, c := range numeric {
		remainder = (remainder*10 + int(c-'0')) % 97
	}

	return remainder == 1
}

func TestGenerateReference(t *testing.T) {
	gen := NewFakeDataGenerator(12345)

	ref := gen.GenerateReference()
	if ref == "" {
		t.Error("expected non-empty reference")
	}
	if ref[:3] != "REF" {
		t.Errorf("expected reference to start with 'REF', got: %s", ref)
	}
}

func TestDeterministicWithSameSeed(t *testing.T) {
	gen1 := NewFakeDataGenerator(42)
	gen2 := NewFakeDataGenerator(42)

	// Generate same sequence
	for i := 0; i < 10; i++ {
		nom1 := gen1.GenerateNom()
		nom2 := gen2.GenerateNom()
		if nom1 != nom2 {
			t.Errorf("different noms with same seed: %s vs %s", nom1, nom2)
		}

		prenom1 := gen1.GeneratePrenom()
		prenom2 := gen2.GeneratePrenom()
		if prenom1 != prenom2 {
			t.Errorf("different prenoms with same seed: %s vs %s", prenom1, prenom2)
		}

		montant1 := gen1.GenerateMontant(0, 1000)
		montant2 := gen2.GenerateMontant(0, 1000)
		if !montant1.Equal(montant2) {
			t.Errorf("different montants with same seed: %s vs %s", montant1, montant2)
		}
	}
}

func TestGenerateSoldeInitial(t *testing.T) {
	gen := NewFakeDataGenerator(12345)

	// Generate many values and check distribution
	smallCount, mediumCount, largeCount := 0, 0, 0
	iterations := 1000

	for i := 0; i < iterations; i++ {
		solde := gen.GenerateSoldeInitial()

		// Verify it's positive
		if solde.LessThanOrEqual(decimal.Zero) {
			t.Errorf("expected positive solde, got: %s", solde)
		}

		// Categorize
		val, _ := solde.Float64()
		switch {
		case val < 500:
			smallCount++
		case val < 5000:
			mediumCount++
		default:
			largeCount++
		}
	}

	// Verify rough distribution (with some tolerance)
	smallPct := float64(smallCount) / float64(iterations)
	mediumPct := float64(mediumCount) / float64(iterations)
	largePct := float64(largeCount) / float64(iterations)

	if smallPct < 0.4 || smallPct > 0.8 {
		t.Errorf("small balance percentage %.2f outside expected range [0.4, 0.8]", smallPct)
	}
	if mediumPct < 0.15 || mediumPct > 0.45 {
		t.Errorf("medium balance percentage %.2f outside expected range [0.15, 0.45]", mediumPct)
	}
	if largePct < 0.02 || largePct > 0.2 {
		t.Errorf("large balance percentage %.2f outside expected range [0.02, 0.2]", largePct)
	}
}

func TestGenerateEmail(t *testing.T) {
	gen := NewFakeDataGenerator(12345)

	email := gen.GenerateEmail("Jean", "Dupont")
	if email == "" {
		t.Error("expected non-empty email")
	}

	// Check basic email format
	emailRegex := regexp.MustCompile(`^[a-z0-9._]+@[a-z.]+$`)
	if !emailRegex.MatchString(email) {
		t.Errorf("invalid email format: %s", email)
	}
}

func TestGeneratePhoneNumber(t *testing.T) {
	gen := NewFakeDataGenerator(12345)

	phone := gen.GeneratePhoneNumber()

	// French mobile format: +33X XXXXXXXX
	phoneRegex := regexp.MustCompile(`^\+33[67]\d{8}$`)
	if !phoneRegex.MatchString(phone) {
		t.Errorf("invalid phone format: %s", phone)
	}
}
