package view

import (
	"os"
	"time"

	"github.com/rshatskiy/tokenburning/internal/aggregate"
	"github.com/rshatskiy/tokenburning/internal/insights"
	"github.com/rshatskiy/tokenburning/internal/pricing"
	"github.com/rshatskiy/tokenburning/internal/store"
)

// parsePeriodDays: "7d"/"30d"/"90d" → число дней; "all" → 0; иначе 30.
func parsePeriodDays(p string) int {
	switch p {
	case "7d":
		return 7
	case "30d":
		return 30
	case "90d":
		return 90
	case "all":
		return 0
	default:
		return 30
	}
}

type Summary struct {
	Period         string                 `json:"period"`
	KPIs           store.KPI              `json:"kpis"`
	CostOverTime   []store.DayCost        `json:"costOverTime"`
	ByTool         []store.ToolSummary    `json:"byTool"`
	ByModel        []store.ModelSummary   `json:"byModel"`
	TopProjects    []store.ProjectSummary `json:"topProjects"`
	Activity       []store.DaySessions    `json:"activity"`
	SessionsByTool []store.ToolSessions   `json:"sessionsByTool"`
	CacheSavings   float64                `json:"cacheSavings"`
	Plan           *PlanInfo              `json:"plan,omitempty"`
	Insights       []insights.Insight     `json:"insights,omitempty"`
	Currency       *CurrencyInfo          `json:"currency,omitempty"`
}

// CurrencyInfo — валюта отображения для фронта (данные всегда в USD).
type CurrencyInfo struct {
	Code   string  `json:"code"`
	Rate   float64 `json:"rate"`
	Symbol string  `json:"symbol"`
}

// PlanInfo — «извлечено из подписки»: API-эквивалент с начала месяца против цены плана.
type PlanInfo struct {
	MonthlyUSD float64 `json:"monthlyUsd"`
	MTDCost    float64 `json:"mtdCost"`
	Multiplier float64 `json:"multiplier"`
}

// attachInsights добавляет в сводку сигналы optimize (best-effort).
func attachInsights(s *Summary, db *store.DB) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	s.Insights = insights.Build(db, home, time.Now())
}

// attachPlan дополняет сводку метрикой подписки (no-op при usd<=0).
func attachPlan(s *Summary, db *store.DB, usd float64) {
	if usd <= 0 {
		return
	}
	now := time.Now()
	mtd, err := db.CostTotal(store.Filter{Since: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)})
	if err != nil {
		return
	}
	s.Plan = &PlanInfo{MonthlyUSD: usd, MTDCost: mtd, Multiplier: mtd / usd}
}

// BuildSummary собирает все агрегаты за период в один объект для фронта.
func BuildSummary(db *store.DB, period string) (Summary, error) {
	// Окно: «days календарных суток, сегодня включительно» в локальном поясе —
	// привязано к локальной полуночи, не к текущему времени суток (см. aggregate.SinceForDays).
	since := aggregate.SinceForDays(parsePeriodDays(period), time.Now())
	var s Summary
	var err error
	s.Period = period
	if s.KPIs, err = db.KPITotals(since); err != nil {
		return s, err
	}
	if s.CostOverTime, err = db.CostOverTime(since); err != nil {
		return s, err
	}
	if s.ByTool, err = db.SummaryByTool(since); err != nil {
		return s, err
	}
	if s.ByModel, err = db.SummaryByModel(since); err != nil {
		return s, err
	}
	if s.TopProjects, err = db.SummaryByProject(since); err != nil {
		return s, err
	}
	if s.Activity, err = db.ActivityByDay(since); err != nil {
		return s, err
	}
	if s.SessionsByTool, err = db.SessionStatsByTool(since); err != nil {
		return s, err
	}
	if len(s.TopProjects) > 8 {
		s.TopProjects = s.TopProjects[:8]
	}
	// Оценка экономии на кэше: cache-read токены против цены свежего input.
	if cat, perr := pricing.LoadEmbedded(); perr == nil {
		if crs, cerr := db.CacheReadByModel(since); cerr == nil {
			for _, m := range crs {
				s.CacheSavings += cat.CacheSavings(m.Model, m.CacheRead)
			}
		}
	}
	return s, nil
}
