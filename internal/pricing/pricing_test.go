package pricing

import (
	"math"
	"testing"

	"github.com/lens/lens/internal/model"
)

func TestCostKnownModelUsesActualBasis(t *testing.T) {
	c, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	// 1M input + 1M output по opus: 15 + 75 = 90 USD
	cost := c.Cost("claude-opus-4-7", model.Tokens{Input: 1_000_000, Output: 1_000_000})
	if math.Abs(cost.Amount-90.0) > 1e-6 {
		t.Fatalf("Amount = %f, want 90.0", cost.Amount)
	}
	if cost.Basis != model.BasisActual {
		t.Fatalf("Basis = %s, want actual", cost.Basis)
	}
	if cost.PricingVersion != "2026-06-07" {
		t.Fatalf("PricingVersion = %s, want 2026-06-07", cost.PricingVersion)
	}
}

func TestCostUnknownModelIsEstimatedZero(t *testing.T) {
	c, _ := LoadEmbedded()
	cost := c.Cost("gpt-9-ultra", model.Tokens{Input: 1000})
	if cost.Basis != model.BasisEstimated {
		t.Fatalf("Basis = %s, want estimated", cost.Basis)
	}
}
