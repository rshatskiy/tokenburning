package pricing

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/lens/lens/internal/model"
)

//go:embed prices.json
var embeddedPrices []byte

type modelPrice struct {
	Input     float64 `json:"input"`
	Output    float64 `json:"output"`
	CacheRead float64 `json:"cache_read"`
	Cache5m   float64 `json:"cache_5m"`
	Cache1h   float64 `json:"cache_1h"`
}

type Catalog struct {
	Version  string                `json:"version"`
	Currency string                `json:"currency"`
	Models   map[string]modelPrice `json:"models"`
}

// LoadEmbedded читает вшитый fallback-снапшот цен. Сетевых вызовов нет.
func LoadEmbedded() (*Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(embeddedPrices, &c); err != nil {
		return nil, err
	}
	if c.Currency == "" {
		return nil, fmt.Errorf("pricing: missing currency in embedded snapshot")
	}
	return &c, nil
}

// Cost считает стоимость по разбивке токенов. Неизвестная модель → estimated с нулём.
func (c *Catalog) Cost(modelName string, tk model.Tokens) model.Cost {
	p, ok := c.Models[modelName]
	if !ok {
		return model.Cost{Amount: 0, Currency: c.Currency, Basis: model.BasisEstimated, PricingVersion: c.Version}
	}
	const M = 1_000_000.0
	amount := float64(tk.Input)/M*p.Input +
		float64(tk.Output)/M*p.Output +
		float64(tk.CacheRead)/M*p.CacheRead +
		float64(tk.Cache5m)/M*p.Cache5m +
		float64(tk.Cache1h)/M*p.Cache1h
	return model.Cost{Amount: amount, Currency: c.Currency, Basis: model.BasisActual, PricingVersion: c.Version}
}
