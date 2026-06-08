package aggregate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rshatskiy/tokenburning/internal/store"
)

type ToolAgg struct {
	Tool   string  `json:"tool"`
	Events int64   `json:"events"`
	Tokens int64   `json:"tokens"`
	Cost   float64 `json:"cost"`
}

type ModelAgg struct {
	Model  string  `json:"model"`
	Events int64   `json:"events"`
	Cost   float64 `json:"cost"`
}

type DayAgg struct {
	Date string  `json:"date"` // день (UTC) — тайминги огрублены до дня
	Cost float64 `json:"cost"`
}

type BreadthAgg struct {
	TotalCost   float64    `json:"totalCost"`
	TotalTokens int64      `json:"totalTokens"`
	ActiveDays  int        `json:"activeDays"`
	ByTool      []ToolAgg  `json:"byTool"`
	ByModel     []ModelAgg `json:"byModel"`
	CostByDay   []DayAgg   `json:"costByDay"`
}

type ToolSessionAgg struct {
	Tool              string  `json:"tool"`
	MedianDurationMin float64 `json:"medianDurationMin"`
	MedianTokens      float64 `json:"medianTokens"`
	MedianIterations  float64 `json:"medianIterations"`
}

type DepthAgg struct {
	Sessions []ToolSessionAgg `json:"sessions"`
}

// Payload — производный агрегат одного человека. Без сырого, без project_key,
// без точных таймингов. Сервер применяет когортное подавление (другой спек).
type Payload struct {
	SchemaVersion string      `json:"schemaVersion"`
	Period        string      `json:"period"`
	Categories    []string    `json:"categories"`
	Breadth       *BreadthAgg `json:"breadth,omitempty"`
	Depth         *DepthAgg   `json:"depth,omitempty"`
}

// periodSince конвертирует период (7d/30d/90d/all) в момент since (zero = всё).
func periodSince(period string) time.Time {
	days := map[string]int{"7d": 7, "30d": 30, "90d": 90}[period]
	if period == "all" || days == 0 {
		return time.Time{}
	}
	return time.Now().UTC().AddDate(0, 0, -days)
}

func has(cats []string, c string) bool {
	for _, x := range cats {
		if x == c {
			return true
		}
	}
	return false
}

// knownModelPrefixes — публичные семейства моделей; всё прочее (включая кастомные
// приватные алиасы) обобщается в "other", чтобы не раскрывать необычные имена.
var knownModelPrefixes = []string{
	"claude-", "claude_", "gpt-", "gpt5", "o1", "o3", "o4", "gemini", "llama",
	"mistral", "deepseek", "grok", "qwen", "sonnet", "opus", "haiku",
}

// normalizeModel оставляет только распознаваемые публичные имена; пустые/unknown/
// synthetic и любые нераспознанные строки → "other".
func normalizeModel(m string) string {
	if m == "" || m == "unknown" || m == "<synthetic>" {
		return "other"
	}
	lm := strings.ToLower(m)
	for _, p := range knownModelPrefixes {
		if strings.HasPrefix(lm, p) {
			return m
		}
	}
	return "other"
}

// Push отправляет payload на endpoint с Bearer-токеном. Только производное.
func Push(p Payload, endpoint, token string) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint+"/v1/aggregates", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	c := &http.Client{Timeout: 30 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("push: сервер ответил %s", resp.Status)
	}
	return nil
}

// Build собирает payload по выбранным категориям. project_key и точные ts не включаются.
func Build(db *store.DB, categories []string, period string) (Payload, error) {
	since := periodSince(period)
	p := Payload{SchemaVersion: "1", Period: period, Categories: categories}

	if has(categories, "breadth") {
		k, err := db.KPITotals(since)
		if err != nil {
			return p, err
		}
		tools, err := db.SummaryByTool(since)
		if err != nil {
			return p, err
		}
		models, err := db.SummaryByModel(since)
		if err != nil {
			return p, err
		}
		cot, err := db.CostOverTime(since)
		if err != nil {
			return p, err
		}
		b := &BreadthAgg{TotalCost: k.Cost, TotalTokens: k.Tokens, ActiveDays: k.ActiveDays}
		for _, t := range tools {
			b.ByTool = append(b.ByTool, ToolAgg{Tool: t.Tool, Events: t.Events, Tokens: t.Tokens, Cost: t.CostAmount})
		}
		// модели с merge редких в "other"
		idx := map[string]int{}
		for _, m := range models {
			name := normalizeModel(m.Model)
			if i, ok := idx[name]; ok {
				b.ByModel[i].Cost += m.CostAmount
				b.ByModel[i].Events += m.Events
			} else {
				idx[name] = len(b.ByModel)
				b.ByModel = append(b.ByModel, ModelAgg{Model: name, Cost: m.CostAmount, Events: m.Events})
			}
		}
		for _, d := range cot {
			b.CostByDay = append(b.CostByDay, DayAgg{Date: d.Date, Cost: d.Cost})
		}
		p.Breadth = b
	}

	if has(categories, "depth") {
		st, err := db.SessionStatsByTool(since)
		if err != nil {
			return p, err
		}
		d := &DepthAgg{}
		for _, ts := range st {
			d.Sessions = append(d.Sessions, ToolSessionAgg{
				Tool:              ts.Tool,
				MedianDurationMin: ts.Stats.MedianDurationMin,
				MedianTokens:      ts.Stats.MedianTokens,
				MedianIterations:  ts.Stats.MedianIterations,
			})
		}
		p.Depth = d
	}

	return p, nil
}
