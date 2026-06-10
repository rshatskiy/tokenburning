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
	if cost.PricingVersion == "" || cost.PricingVersion != c.Version {
		t.Fatalf("PricingVersion = %s, want версию каталога %s", cost.PricingVersion, c.Version)
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

// Точность прайса: текущая топ-модель обязана быть в каталоге.
func TestFable5Priced(t *testing.T) {
	c, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	got := c.Cost("claude-fable-5", model.Tokens{Input: 1_000_000, Output: 1_000_000})
	if got.Basis != model.BasisActual {
		t.Fatalf("claude-fable-5 неизвестна каталогу (basis=%s)", got.Basis)
	}
	if got.Amount != 60.0 { // $10 input + $50 output
		t.Fatalf("Amount = %v, ожидалось 60.0", got.Amount)
	}
}

// Нормализация имён: датированные id, суффикс "[1m]" и голые алиасы из реальных
// логов Claude Code не должны давать $0.
func TestCostNameNormalization(t *testing.T) {
	c, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	tk := model.Tokens{Input: 1_000_000}
	cases := map[string]float64{
		"claude-opus-4-5-20251101": 5.0,  // датированный id → префикс claude-opus-4-5
		"claude-fable-5[1m]":       10.0, // суффикс контекстного окна
		"opus":                     5.0,  // голый алиас из логов
		"sonnet":                   3.0,
		"haiku":                    1.0,
	}
	for name, want := range cases {
		got := c.Cost(name, tk)
		if got.Basis != model.BasisActual || got.Amount != want {
			t.Errorf("Cost(%q) = %v (basis=%s), ожидалось %v/actual", name, got.Amount, got.Basis, want)
		}
	}
	if got := c.Cost("totally-unknown-model", tk); got.Basis != model.BasisEstimated || got.Amount != 0 {
		t.Errorf("неизвестная модель должна давать estimated/0, получено %+v", got)
	}
}
