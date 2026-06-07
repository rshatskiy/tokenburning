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
