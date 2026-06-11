// Package insights превращает собранные данные в действия: «почему дорого»
// и «что исправить», а не просто цифры. Все правила детерминированные и
// честные: оценки помечаются как оценки.
package insights

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/rshatskiy/tokenburning/internal/store"
)

// Insight — один сигнал. Kind + Data структурированы, чтобы фронт мог
// локализовать текст; Text — готовая русская строка для CLI.
type Insight struct {
	Kind     string         `json:"kind"`     // cache_drop | expensive_session | unpriced_model | claude_md_big | mcp_many
	Severity string         `json:"severity"` // warn | info
	Data     map[string]any `json:"data"`
	Text     string         `json:"text"`
}

// Build считает инсайты по БД и локальному окружению (~/.claude).
func Build(db *store.DB, home string, now time.Time) []Insight {
	var out []Insight
	out = append(out, cacheDrops(db, now)...)
	out = append(out, expensiveSessions(db, now)...)
	out = append(out, unpricedModels(db, now)...)
	out = append(out, claudeSetup(home)...)
	return out
}

// cacheDrops: кэш-хит проекта упал ≥15 п.п. неделя к неделе при заметном объёме.
func cacheDrops(db *store.DB, now time.Time) []Insight {
	const minTokens = 1_000_000
	cur, err := db.ProjectCacheRates(store.Filter{Since: now.AddDate(0, 0, -7)})
	if err != nil {
		return nil
	}
	prev, err := db.ProjectCacheRates(store.Filter{Since: now.AddDate(0, 0, -14), Until: now.AddDate(0, 0, -7)})
	if err != nil {
		return nil
	}
	rate := func(r store.ProjectCacheRate) (float64, bool) {
		total := r.Input + r.CacheRead
		if total < minTokens {
			return 0, false
		}
		return float64(r.CacheRead) / float64(total) * 100, true
	}
	prevIdx := map[string]store.ProjectCacheRate{}
	for _, p := range prev {
		prevIdx[p.Project] = p
	}
	var out []Insight
	for _, c := range cur {
		p, ok := prevIdx[c.Project]
		if !ok {
			continue
		}
		curR, ok1 := rate(c)
		prevR, ok2 := rate(p)
		if !ok1 || !ok2 || prevR-curR < 15 {
			continue
		}
		proj := c.Project
		if proj == "" {
			proj = "(без проекта)"
		}
		out = append(out, Insight{
			Kind: "cache_drop", Severity: "warn",
			Data: map[string]any{"project": proj, "fromPct": int(prevR), "toPct": int(curR)},
			Text: fmt.Sprintf("кэш-хит в %s упал с %d%% до %d%% за неделю. Что сделать: сравните, что изменилось в начале контекста (набор MCP, системные правила, CLAUDE.md) — стабильный префикс вернёт кэш", filepath.Base(proj), int(prevR), int(curR)),
		})
	}
	return out
}

// expensiveSessions: сессии последней недели дороже max($10, 4×медианы за 30 дней).
func expensiveSessions(db *store.DB, now time.Time) []Insight {
	all, err := db.SessionCosts(store.Filter{Since: now.AddDate(0, 0, -30)})
	if err != nil || len(all) < 5 {
		return nil
	}
	costs := make([]float64, 0, len(all))
	for _, s := range all {
		if s.Cost > 0 {
			costs = append(costs, s.Cost)
		}
	}
	if len(costs) < 5 {
		return nil
	}
	sort.Float64s(costs)
	median := costs[len(costs)/2]
	threshold := median * 4
	if threshold < 10 {
		threshold = 10
	}
	week, err := db.SessionCosts(store.Filter{Since: now.AddDate(0, 0, -7)})
	if err != nil {
		return nil
	}
	sort.Slice(week, func(i, j int) bool { return week[i].Cost > week[j].Cost })
	var out []Insight
	for _, s := range week {
		if s.Cost < threshold || len(out) >= 3 {
			break
		}
		out = append(out, Insight{
			Kind: "expensive_session", Severity: "warn",
			Data: map[string]any{"session": s.SessionID, "tool": s.Tool, "cost": s.Cost, "median": median},
			Text: fmt.Sprintf("сессия %.8s (%s) стоила $%.2f — в %.0f раз дороже вашей медианы $%.2f. Что сделать: дробите длинные сессии — после большой задачи начинайте новую, дороже всего хвост с разросшимся контекстом", s.SessionID, s.Tool, s.Cost, s.Cost/median, median),
		})
	}
	return out
}

// unpricedModels: заметный объём токенов оценён в $0 — прайс не знает модель.
func unpricedModels(db *store.DB, now time.Time) []Insight {
	rows, err := db.FilteredByModel(store.Filter{Since: now.AddDate(0, 0, -30)})
	if err != nil {
		return nil
	}
	var out []Insight
	for _, m := range rows {
		if m.CostAmount == 0 && m.Tokens > 1_000_000 && m.Model != "<synthetic>" && m.Model != "unknown" {
			out = append(out, Insight{
				Kind: "unpriced_model", Severity: "info",
				Data: map[string]any{"model": m.Model, "tokens": m.Tokens},
				Text: fmt.Sprintf("модель %q не оценена ($0 при %d ток.) — задайте соответствие: tokenburning alias %q <каноническое-имя>", m.Model, m.Tokens, m.Model),
			})
		}
	}
	return out
}

// claudeSetup: локальная гигиена ~/.claude — раздутый CLAUDE.md и много MCP.
func claudeSetup(home string) []Insight {
	var out []Insight
	if fi, err := os.Stat(filepath.Join(home, ".claude", "CLAUDE.md")); err == nil && fi.Size() > 20*1024 {
		kb := fi.Size() / 1024
		out = append(out, Insight{
			Kind: "claude_md_big", Severity: "info",
			Data: map[string]any{"kb": kb, "estTokens": fi.Size() / 4},
			Text: fmt.Sprintf("CLAUDE.md занимает %d КБ (~%d ток.) и входит в каждый запрос — сократите или разнесите по скиллам", kb, fi.Size()/4),
		})
	}
	if b, err := os.ReadFile(filepath.Join(home, ".claude.json")); err == nil {
		var doc struct {
			MCP map[string]any `json:"mcpServers"`
		}
		if json.Unmarshal(b, &doc) == nil && len(doc.MCP) >= 6 {
			out = append(out, Insight{
				Kind: "mcp_many", Severity: "info",
				Data: map[string]any{"count": len(doc.MCP)},
				Text: fmt.Sprintf("подключено %d MCP-серверов — каждый добавляет схемы инструментов в контекст; отключите неиспользуемые", len(doc.MCP)),
			})
		}
	}
	return out
}
