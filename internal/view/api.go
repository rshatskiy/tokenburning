package view

import (
	"time"

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
	Period       string                   `json:"period"`
	KPIs         store.KPI                `json:"kpis"`
	CostOverTime []store.DayCost          `json:"costOverTime"`
	ByTool       []store.ToolSummary      `json:"byTool"`
	ByModel      []store.ModelSummary     `json:"byModel"`
	TopProjects  []store.ProjectSummary   `json:"topProjects"`
	Activity     []store.DaySessions      `json:"activity"`
	Sessions     store.SessionStatsResult `json:"sessions"`
}

// BuildSummary собирает все агрегаты за период в один объект для фронта.
func BuildSummary(db *store.DB, period string) (Summary, error) {
	days := parsePeriodDays(period)
	var since time.Time // нулевое время = «всё»
	if days > 0 {
		since = time.Now().UTC().AddDate(0, 0, -days)
	}
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
	if s.Sessions, err = db.SessionStats(since); err != nil {
		return s, err
	}
	if len(s.TopProjects) > 8 {
		s.TopProjects = s.TopProjects[:8]
	}
	return s, nil
}
