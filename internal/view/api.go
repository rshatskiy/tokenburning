package view

import (
	"os"
	"time"

	"github.com/rshatskiy/tokenburning/internal/aggregate"
	"github.com/rshatskiy/tokenburning/internal/insights"
	"github.com/rshatskiy/tokenburning/internal/pricing"
	"github.com/rshatskiy/tokenburning/internal/quality"
	"github.com/rshatskiy/tokenburning/internal/store"
)

// periodSince: начало окна для периода дашборда (локальный пояс).
// today — с местной полуночи; month — с 1-го числа; Nd — календарные сутки;
// all — нулевое время (без границы).
func periodSince(p string, now time.Time) time.Time {
	switch p {
	case "today":
		return aggregate.SinceForDays(1, now)
	case "month":
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	case "7d":
		return aggregate.SinceForDays(7, now)
	case "90d":
		return aggregate.SinceForDays(90, now)
	case "all":
		return time.Time{}
	default: // 30d
		return aggregate.SinceForDays(30, now)
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
	Quality        []quality.ModelQuality `json:"quality,omitempty"`
}

// CurrencyInfo — валюта отображения для фронта (данные всегда в USD).
type CurrencyInfo struct {
	Code   string  `json:"code"`
	Rate   float64 `json:"rate"`
	Symbol string  `json:"symbol"`
}

// PlanInfo — «извлечено из подписки»: API-эквивалент с начала месяца против цены плана.
type PlanInfo struct {
	MonthlyUSD   float64 `json:"monthlyUsd"`
	MTDCost      float64 `json:"mtdCost"`
	Multiplier   float64 `json:"multiplier"`
	ForecastCost float64 `json:"forecastCost,omitempty"` // линейный прогноз на конец месяца
	ForecastX    float64 `json:"forecastX,omitempty"`
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
	p := &PlanInfo{MonthlyUSD: usd, MTDCost: mtd, Multiplier: mtd / usd}
	// линейный прогноз до конца месяца по прошедшей доле месяца
	daysIn := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.Local).Day()
	elapsed := float64(now.Day()-1) + float64(now.Hour())/24
	if elapsed > 0.25 { // в первые часы месяца прогноз — шум
		p.ForecastCost = mtd / elapsed * float64(daysIn)
		p.ForecastX = p.ForecastCost / usd
	}
	s.Plan = p
}

// BuildSummary собирает все агрегаты за период в один объект для фронта.
func BuildSummary(db *store.DB, period string) (Summary, error) {
	now := time.Now()
	since := periodSince(period, now)
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
	if rows, qerr := db.RawToolEvents(store.Filter{Since: since}); qerr == nil {
		s.Quality = quality.Compute(rows)
		if len(s.Quality) > 6 {
			s.Quality = s.Quality[:6]
		}
		// тренд: то же окно непосредственно перед текущим
		if !since.IsZero() {
			win := now.Sub(since)
			if prev, perr := db.RawToolEvents(store.Filter{Since: since.Add(-win), Until: since}); perr == nil {
				prevQ := map[string]float64{}
				for _, q := range quality.Compute(prev) {
					prevQ[q.Model] = q.OneShotPct
				}
				for i := range s.Quality {
					if pv, ok := prevQ[s.Quality[i].Model]; ok {
						d := s.Quality[i].OneShotPct - pv
						s.Quality[i].DeltaPct = &d
					}
				}
			}
		}
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
