package pricing

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LiteLLM публикует актуальные цены всех моделей; мы кэшируем их сутки и
// накладываем поверх встроенного снапшота. Сеть — best-effort: офлайн или
// TOKENBURNING_OFFLINE=1 → работаем на встроенном прайсе.
const liteLLMURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

const liveTTL = 24 * time.Hour

func liveCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tokenburning", "prices-live.json"), nil
}

// liteLLMEntry — поля LiteLLM, которые нам нужны (цены за токен).
type liteLLMEntry struct {
	Input     float64 `json:"input_cost_per_token"`
	Output    float64 `json:"output_cost_per_token"`
	CacheRead float64 `json:"cache_read_input_token_cost"`
	CacheCr   float64 `json:"cache_creation_input_token_cost"`
}

// RefreshLive обновляет локальный кэш цен, если он старше суток. Сетевая
// ошибка не фатальна — вернётся в следующий раз.
func RefreshLive(timeout time.Duration) error {
	if os.Getenv("TOKENBURNING_OFFLINE") != "" {
		return nil
	}
	p, err := liveCachePath()
	if err != nil {
		return err
	}
	if fi, err := os.Stat(p); err == nil && time.Since(fi.ModTime()) < liveTTL {
		return nil // кэш свежий
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(liteLLMURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	var raw map[string]liteLLMEntry
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return err
	}
	models := map[string]modelPrice{}
	for name, e := range raw {
		if e.Input <= 0 && e.Output <= 0 {
			continue
		}
		mp := modelPrice{
			Input:     e.Input * 1e6,
			Output:    e.Output * 1e6,
			CacheRead: e.CacheRead * 1e6,
			Cache5m:   e.CacheCr * 1e6,
		}
		if mp.Cache5m > 0 {
			mp.Cache1h = mp.Cache5m * 1.6 // 2×input против 1.25×input — стандартное соотношение TTL
		}
		// "vendor/model" → дублируем и под голым именем, если оно свободно
		models[name] = mp
		if i := strings.LastIndexByte(name, '/'); i >= 0 {
			bare := name[i+1:]
			if _, busy := models[bare]; !busy && bare != "" {
				models[bare] = mp
			}
		}
	}
	if len(models) < 50 {
		return nil // подозрительно мало — не затираем кэш мусором
	}
	b, err := json.Marshal(Catalog{Version: "litellm-" + time.Now().UTC().Format("2006-01-02"), Currency: "USD", Models: models})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// LoadEffective возвращает рабочий прайс: встроенный снапшот, поверх — живой
// кэш LiteLLM (если есть), поверх — пользовательские алиасы моделей.
func LoadEffective(aliases map[string]string) (*Catalog, error) {
	c, err := LoadEmbedded()
	if err != nil {
		return nil, err
	}
	if p, perr := liveCachePath(); perr == nil {
		if b, rerr := os.ReadFile(p); rerr == nil {
			var live Catalog
			if json.Unmarshal(b, &live) == nil && len(live.Models) > 0 {
				for k, v := range live.Models {
					if _, ours := c.Models[k]; !ours { // наш снапшот точнее для Claude — не затираем
						c.Models[k] = v
					}
				}
				c.Version = c.Version + "+" + live.Version
			}
		}
	}
	c.aliases = aliases
	return c, nil
}
