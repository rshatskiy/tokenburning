package view

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/store"
)

func TestParsePeriod(t *testing.T) {
	if d := parsePeriodDays("7d"); d != 7 {
		t.Fatalf("7d → %d, want 7", d)
	}
	if d := parsePeriodDays("all"); d != 0 {
		t.Fatalf("all → %d, want 0", d)
	}
	if d := parsePeriodDays("garbage"); d != 30 {
		t.Fatalf("default → %d, want 30", d)
	}
}

func TestBuildSummary(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now().UTC()
	if err := db.Insert([]model.Event{{
		EventID: "x", Tool: model.ToolClaudeCode, TS: now, Model: "claude-opus-4-7",
		BillingMode: model.BillingFlatEquivalent,
		Cost:        model.Cost{Amount: 9, Currency: "USD", Basis: model.BasisActual},
		Tokens:      model.Tokens{Input: 100}, SessionID: "s", ProjectKey: "/p",
	}}); err != nil {
		t.Fatal(err)
	}
	sum, err := BuildSummary(db, "30d")
	if err != nil {
		t.Fatalf("BuildSummary: %v", err)
	}
	if sum.KPIs.Cost != 9 {
		t.Fatalf("KPI cost = %v, want 9", sum.KPIs.Cost)
	}
	if len(sum.ByTool) != 1 || sum.ByTool[0].Tool != "claude_code" {
		t.Fatalf("byTool неверно: %+v", sum.ByTool)
	}
	if sum.Period != "30d" {
		t.Fatalf("period = %q", sum.Period)
	}
}
