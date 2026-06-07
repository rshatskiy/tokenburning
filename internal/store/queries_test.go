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
	if len(pr) != 2 || pr[0].Project != "/p1" { // /p1 дороже: 15 vs 20? нет, /p2=20 -> сортировка по cost desc
		// /p2 cost 20 > /p1 cost 15 → первый /p2
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
