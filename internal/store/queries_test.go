package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/rshatskiy/tokenburning/internal/model"
)

func openTmp(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func ev(id, sess, proj string, ts time.Time, cost float64, inTok, cacheRead int64) model.Event {
	return model.Event{
		EventID: id, Tool: model.ToolClaudeCode, TS: ts, Model: "claude-opus-4-7",
		BillingMode: model.BillingFlatEquivalent,
		Cost:        model.Cost{Amount: cost, Currency: "USD", Basis: model.BasisActual, PricingVersion: "v"},
		Tokens:      model.Tokens{Input: inTok, CacheRead: cacheRead}, SessionID: sess, ProjectKey: proj,
	}
}

func TestPercentile(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if m := percentile(xs, 50); m < 5 || m > 6 {
		t.Fatalf("median = %v, want ~5.5", m)
	}
	if p := percentile(xs, 90); p < 9 {
		t.Fatalf("p90 = %v, want >=9", p)
	}
	if percentile(nil, 50) != 0 {
		t.Fatal("percentile(nil) should be 0")
	}
	if percentile([]float64{7}, 50) != 7 {
		t.Fatal("single element percentile should return that element")
	}
	if percentile([]float64{1, 3}, 100) != 3 {
		t.Fatal("p100 of {1,3} should be 3")
	}
	if percentile([]float64{1, 3}, 0) != 1 {
		t.Fatal("p0 of {1,3} should be 1")
	}
}

func TestSummaryByProjectAndActivity(t *testing.T) {
	db := openTmp(t)
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	since := base.Add(-24 * time.Hour)
	if err := db.Insert([]model.Event{
		ev("a", "s1", "/p1", base, 10, 1, 0),
		ev("b", "s2", "/p1", base.Add(24*time.Hour), 5, 1, 0),
		ev("c", "s3", "/p2", base, 20, 1, 0),
	}); err != nil {
		t.Fatal(err)
	}
	pr, err := db.SummaryByProject(since)
	if err != nil {
		t.Fatalf("SummaryByProject: %v", err)
	}
	if len(pr) != 2 {
		t.Fatalf("want 2 projects, got %d", len(pr))
	}
	if pr[0].Project != "/p2" {
		t.Fatalf("первый проект должен быть /p2 (cost 20 > 15), got %s", pr[0].Project)
	}
	var p1 *ProjectSummary
	for i := range pr {
		if pr[i].Project == "/p1" {
			p1 = &pr[i]
		}
	}
	if p1 == nil || p1.Cost != 15 || p1.Sessions != 2 {
		t.Fatalf("/p1 summary неверна: %+v", p1)
	}
	act, err := db.ActivityByDay(since)
	if err != nil {
		t.Fatalf("ActivityByDay: %v", err)
	}
	if len(act) != 2 {
		t.Fatalf("дней активности %d, want 2", len(act))
	}
}

func TestSessionStats(t *testing.T) {
	db := openTmp(t)
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	since := base.Add(-48 * time.Hour)
	var evs []model.Event
	// сессия s1: 3 события, длительность 60 мин, стоимость 30
	evs = append(evs,
		ev("a1", "s1", "/p1", base, 10, 100, 0),
		ev("a2", "s1", "/p1", base.Add(30*time.Minute), 10, 100, 0),
		ev("a3", "s1", "/p1", base.Add(60*time.Minute), 10, 100, 0))
	// сессия s2: 1 событие (как Codex), длительность 0, стоимость 5
	evs = append(evs, ev("b1", "s2", "/p2", base, 5, 50, 0))
	// сессия s3: 2 события, длительность 10 мин, стоимость 2
	evs = append(evs,
		ev("c1", "s3", "/p3", base, 1, 10, 0),
		ev("c2", "s3", "/p3", base.Add(10*time.Minute), 1, 10, 0))
	if err := db.Insert(evs); err != nil {
		t.Fatal(err)
	}
	s, err := db.SessionStats(since)
	if err != nil {
		t.Fatalf("SessionStats: %v", err)
	}
	// длительность считается по s1(60) и s3(10) → медиана 35
	if s.MedianDurationMin < 34 || s.MedianDurationMin > 36 {
		t.Fatalf("MedianDurationMin = %v, want ~35", s.MedianDurationMin)
	}
	// в scatter — все 3 сессии
	if len(s.Scatter) != 3 {
		t.Fatalf("scatter точек %d, want 3", len(s.Scatter))
	}
	// самая дорогая сессия s1 должна быть среди flagged
	if len(s.Flagged) == 0 || s.Flagged[0].Cost != 30 {
		t.Fatalf("flagged[0] должна быть s1 ($30): %+v", s.Flagged)
	}
	outliers := 0
	for _, p := range s.Scatter {
		if p.Outlier {
			outliers++
		}
	}
	if outliers != 1 {
		t.Fatalf("ожидался 1 выброс (дорогая длинная сессия s1), получено %d", outliers)
	}
}

func TestSessionStatsEmpty(t *testing.T) {
	db := openTmp(t)
	s, err := db.SessionStats(time.Unix(0, 0))
	if err != nil {
		t.Fatalf("SessionStats: %v", err)
	}
	if len(s.Scatter) != 0 || len(s.Flagged) != 0 {
		t.Fatalf("пустая БД не должна давать сессий: %+v", s)
	}
}

func TestKPITotalsAndCostOverTime(t *testing.T) {
	db := openTmp(t)
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	must := func(e error) {
		if e != nil {
			t.Fatal(e)
		}
	}
	must(db.Insert([]model.Event{
		ev("a", "s1", "/p1", base, 10, 1000, 4000),
		ev("b", "s1", "/p1", base.Add(time.Hour), 5, 500, 0),
		ev("c", "s2", "/p2", base.Add(24*time.Hour), 20, 2000, 0),
	}))
	since := base.Add(-24 * time.Hour)
	k, err := db.KPITotals(since)
	if err != nil {
		t.Fatalf("KPITotals: %v", err)
	}
	if k.Cost != 35 {
		t.Fatalf("Cost = %v, want 35", k.Cost)
	}
	if k.Sessions != 2 {
		t.Fatalf("Sessions = %d, want 2", k.Sessions)
	}
	if k.ActiveDays != 2 {
		t.Fatalf("ActiveDays = %d, want 2", k.ActiveDays)
	}
	cot, err := db.CostOverTime(since)
	if err != nil {
		t.Fatalf("CostOverTime: %v", err)
	}
	if len(cot) != 2 {
		t.Fatalf("buckets = %d, want 2 (два дня)", len(cot))
	}
	if cot[0].Cost != 15 {
		t.Fatalf("day1 cost = %v, want 15", cot[0].Cost)
	}
}
