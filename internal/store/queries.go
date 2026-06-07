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
	Cost            float64
	Tokens          int64
	CacheReadTokens int64
	ActiveDays      int
	Sessions        int
	Tools           []string
}

// KPITotals считает сводные KPI по событиям начиная с since.
func (d *DB) KPITotals(since time.Time) (KPI, error) {
	var k KPI
	// Дни/сессии считаются по UTC-дате (date(ts,'unixepoch')); COUNT(DISTINCT session_id)
	// исключает события без session_id (NULL) — это намеренно: считаем только известные сессии.
	row := d.db.QueryRow(`SELECT
        COALESCE(SUM(cost_amount),0),
        COALESCE(SUM(tok_input+tok_output+tok_cache_read+tok_cache_1h+tok_cache_5m+tok_reasoning),0),
        COALESCE(SUM(tok_cache_read),0),
        COUNT(DISTINCT date(ts,'unixepoch')),
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
	Date string
	Cost float64
}

// CostOverTime — стоимость по дням (UTC) начиная с since.
func (d *DB) CostOverTime(since time.Time) ([]DayCost, error) {
	rows, err := d.db.Query(`SELECT date(ts,'unixepoch'), COALESCE(SUM(cost_amount),0)
        FROM events WHERE ts >= ? GROUP BY date(ts,'unixepoch') ORDER BY 1`, since.Unix())
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
	Project  string
	Cost     float64
	Sessions int
	Events   int
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
	Date     string
	Sessions int
}

// ActivityByDay — число сессий по дням (UTC) начиная с since.
func (d *DB) ActivityByDay(since time.Time) ([]DaySessions, error) {
	rows, err := d.db.Query(`SELECT date(ts,'unixepoch'), COUNT(DISTINCT session_id)
        FROM events WHERE ts >= ? GROUP BY date(ts,'unixepoch') ORDER BY 1`, since.Unix())
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
