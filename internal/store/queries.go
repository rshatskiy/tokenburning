package store

import (
	"sort"
	"time"
)

// percentile возвращает p-й перцентиль (0..100) методом ближайшего ранга (линейная интерполяция).
func percentile(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	if len(s) == 1 {
		return s[0]
	}
	rank := p / 100 * float64(len(s)-1)
	lo := int(rank)
	frac := rank - float64(lo)
	if lo+1 >= len(s) {
		return s[len(s)-1]
	}
	return s[lo]*(1-frac) + s[lo+1]*frac
}

type KPI struct {
	Cost            float64  `json:"cost"`
	Tokens          int64    `json:"tokens"`
	CacheReadTokens int64    `json:"cacheReadTokens"`
	ActiveDays      int      `json:"activeDays"`
	Sessions        int      `json:"sessions"`
	Tools           []string `json:"tools"`
}

// KPITotals считает сводные KPI по событиям начиная с since.
func (d *DB) KPITotals(since time.Time) (KPI, error) {
	var k KPI
	// Дни/сессии считаются по ЛОКАЛЬНОЙ дате (date(ts,'unixepoch','localtime')) —
	// процесс дашборда работает на машине пользователя, так что это его пояс.
	// COUNT(DISTINCT session_id) исключает события без session_id (NULL) — намеренно.
	row := d.db.QueryRow(`SELECT
        COALESCE(SUM(cost_amount),0),
        COALESCE(SUM(MAX(tok_total, tok_input+tok_output+tok_cache_read+tok_cache_1h+tok_cache_5m+tok_reasoning)),0),
        COALESCE(SUM(tok_cache_read),0),
        COUNT(DISTINCT date(ts,'unixepoch','localtime')),
        COUNT(DISTINCT session_id)
        FROM events WHERE ts >= ?`, since.Unix())
	if err := row.Scan(&k.Cost, &k.Tokens, &k.CacheReadTokens, &k.ActiveDays, &k.Sessions); err != nil {
		return k, err
	}
	rows, err := d.db.Query(`SELECT DISTINCT tool FROM events WHERE ts >= ? ORDER BY tool`, since.Unix())
	if err != nil {
		return k, err
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return k, err
		}
		k.Tools = append(k.Tools, t)
	}
	return k, rows.Err()
}

type DayCost struct {
	Date string  `json:"date"`
	Cost float64 `json:"cost"`
}

// CostOverTime — стоимость по дням (локальный пояс) начиная с since.
func (d *DB) CostOverTime(since time.Time) ([]DayCost, error) {
	rows, err := d.db.Query(`SELECT date(ts,'unixepoch','localtime'), COALESCE(SUM(cost_amount),0)
        FROM events WHERE ts >= ? GROUP BY date(ts,'unixepoch','localtime') ORDER BY 1`, since.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DayCost
	for rows.Next() {
		var d DayCost
		if err := rows.Scan(&d.Date, &d.Cost); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

type ProjectSummary struct {
	Project  string  `json:"project"`
	Cost     float64 `json:"cost"`
	Sessions int     `json:"sessions"`
	Events   int     `json:"events"`
}

// SummaryByProject — агрегат по project_key начиная с since, по убыванию стоимости.
func (d *DB) SummaryByProject(since time.Time) ([]ProjectSummary, error) {
	rows, err := d.db.Query(`SELECT COALESCE(project_key,'(нет)'),
        COALESCE(SUM(cost_amount),0), COUNT(DISTINCT session_id), COUNT(*)
        FROM events WHERE ts >= ? GROUP BY project_key ORDER BY 2 DESC`, since.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProjectSummary
	for rows.Next() {
		var p ProjectSummary
		if err := rows.Scan(&p.Project, &p.Cost, &p.Sessions, &p.Events); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type DaySessions struct {
	Date     string `json:"date"`
	Sessions int    `json:"sessions"`
}

// ActivityByDay — число сессий по дням (локальный пояс) начиная с since.
func (d *DB) ActivityByDay(since time.Time) ([]DaySessions, error) {
	rows, err := d.db.Query(`SELECT date(ts,'unixepoch','localtime'), COUNT(DISTINCT session_id)
        FROM events WHERE ts >= ? GROUP BY date(ts,'unixepoch','localtime') ORDER BY 1`, since.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DaySessions
	for rows.Next() {
		var d DaySessions
		if err := rows.Scan(&d.Date, &d.Sessions); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

type SessionPoint struct {
	Project     string  `json:"project"`
	DurationMin float64 `json:"durationMin"`
	Iterations  int     `json:"iterations"`
	Cost        float64 `json:"cost"`
	Tokens      int64   `json:"tokens"`
	Outlier     bool    `json:"outlier"`
}

type SessionStatsResult struct {
	MedianDurationMin float64        `json:"medianDurationMin"`
	P90DurationMin    float64        `json:"p90DurationMin"`
	MedianTokens      float64        `json:"medianTokens"`
	P90Tokens         float64        `json:"p90Tokens"`
	MedianIterations  float64        `json:"medianIterations"`
	P90Iterations     float64        `json:"p90Iterations"`
	MedianCost        float64        `json:"medianCost"`
	P90Cost           float64        `json:"p90Cost"`
	Scatter           []SessionPoint `json:"scatter"`
	Flagged           []SessionPoint `json:"flagged"`
}

// computeSessionStats считает медианы/p90 и выбросы для набора сессий одного инструмента.
// Длительность/итерации — только по многособытийным сессиям (iters>=2); токены/стоимость — по всем.
func computeSessionStats(points []SessionPoint) SessionStatsResult {
	var res SessionStatsResult
	var durMulti, iterMulti, tokAll, costAll []float64
	for _, p := range points {
		tokAll = append(tokAll, float64(p.Tokens))
		costAll = append(costAll, p.Cost)
		if p.Iterations >= 2 {
			durMulti = append(durMulti, p.DurationMin)
			iterMulti = append(iterMulti, float64(p.Iterations))
		}
	}
	res.MedianDurationMin, res.P90DurationMin = percentile(durMulti, 50), percentile(durMulti, 90)
	res.MedianIterations, res.P90Iterations = percentile(iterMulti, 50), percentile(iterMulti, 90)
	res.MedianTokens, res.P90Tokens = percentile(tokAll, 50), percentile(tokAll, 90)
	res.MedianCost, res.P90Cost = percentile(costAll, 50), percentile(costAll, 90)
	hasDur := len(durMulti) > 0
	for i := range points {
		costOut := res.P90Cost > 0 && points[i].Cost >= res.P90Cost
		durOut := hasDur && res.P90DurationMin > 0 && points[i].DurationMin >= res.P90DurationMin
		points[i].Outlier = costOut || durOut
	}
	res.Scatter = points
	var flagged []SessionPoint
	for _, p := range points {
		if p.Iterations >= 2 && p.Cost > 0 && p.DurationMin > 0 &&
			p.DurationMin >= res.MedianDurationMin && p.Cost >= res.MedianCost {
			flagged = append(flagged, p)
		}
	}
	sort.Slice(flagged, func(i, j int) bool { return flagged[i].Cost > flagged[j].Cost })
	if len(flagged) > 3 {
		flagged = flagged[:3]
	}
	res.Flagged = flagged
	return res
}

type ToolSessions struct {
	Tool  string             `json:"tool"`
	Stats SessionStatsResult `json:"stats"`
}

// SessionStatsByTool группирует сессии по инструменту и считает аналитику отдельно
// для каждого. Длительность сессии — АКТИВНОЕ время: сумма промежутков между соседними
// событиями, где промежуток <= 30 мин (большие паузы = простой, не считаются).
func (d *DB) SessionStatsByTool(since time.Time) ([]ToolSessions, error) {
	rows, err := d.db.Query(`
WITH ev AS (
  SELECT tool, session_id,
    COALESCE(project_key,'(нет)') AS proj,
    cost_amount AS cost,
    MAX(tok_total, tok_input+tok_output+tok_cache_read+tok_cache_1h+tok_cache_5m+tok_reasoning) AS toks,
    ts - LAG(ts) OVER (PARTITION BY session_id ORDER BY ts) AS gap
  FROM events WHERE ts >= ?
)
SELECT tool, MIN(proj),
  SUM(CASE WHEN gap > 0 AND gap <= 1800 THEN gap ELSE 0 END)/60.0,
  SUM(toks), SUM(cost), COUNT(*)
FROM ev GROUP BY tool, session_id`, since.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byTool := map[string][]SessionPoint{}
	var order []string
	for rows.Next() {
		var tool, proj string
		var activeMin, cost float64
		var tokens, iters int64
		if err := rows.Scan(&tool, &proj, &activeMin, &tokens, &cost, &iters); err != nil {
			return nil, err
		}
		if _, ok := byTool[tool]; !ok {
			order = append(order, tool)
		}
		byTool[tool] = append(byTool[tool], SessionPoint{
			Project: proj, DurationMin: activeMin, Iterations: int(iters), Cost: cost, Tokens: tokens,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// claude_code первым (самый богатый), остальные по алфавиту
	sort.SliceStable(order, func(i, j int) bool {
		if order[i] == "claude_code" {
			return true
		}
		if order[j] == "claude_code" {
			return false
		}
		return order[i] < order[j]
	})
	var out []ToolSessions
	for _, t := range order {
		out = append(out, ToolSessions{Tool: t, Stats: computeSessionStats(byTool[t])})
	}
	return out, nil
}
