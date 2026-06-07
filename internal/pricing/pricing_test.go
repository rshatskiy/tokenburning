package pricing

import (
	"math"
	"testing"

	"github.com/rshatskiy/tokenburning/internal/model"
)

func TestCostKnownModelUsesActualBasis(t *testing.T) {
	c, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	// 1M input + 1M output по opus: 5 + 25 = 30 USD
	cost := c.Cost("claude-opus-4-8", model.Tokens{Input: 1_000_000, Output: 1_000_000})
	if math.Abs(cost.Amount-30.0) > 1e-6 {
		t.Fatalf("Amount = %f, want 30.0", cost.Amount)
	}
	if cost.Basis != model.BasisActual {
		t.Fatalf("Basis = %s, want actual", cost.Basis)
	}
	if cost.PricingVersion != "2026-06-07b" {
		t.Fatalf("PricingVersion = %s, want 2026-06-07b", cost.PricingVersion)
	}
}

func TestCostUnknownModelIsEstimatedZero(t *testing.T) {
	c, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	cost := c.Cost("gpt-9-ultra", model.Tokens{Input: 1000})
	if cost.Basis != model.BasisEstimated {
		t.Fatalf("Basis = %s, want estimated", cost.Basis)
	}
}

func TestCostIncludesCacheTokens(t *testing.T) {
	c, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	// 1M cache_read по sonnet (0.3 USD/M) = 0.3 USD
	cost := c.Cost("claude-sonnet-4-6", model.Tokens{CacheRead: 1_000_000})
	if math.Abs(cost.Amount-0.3) > 1e-6 {
		t.Fatalf("Amount = %f, want 0.3", cost.Amount)
	}
}
