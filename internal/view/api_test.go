package view

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/store"
)

func TestPeriodSince(t *testing.T) {
	now := time.Date(2026, 6, 11, 15, 30, 0, 0, time.Local)
	if got := periodSince("today", now); got != time.Date(2026, 6, 11, 0, 0, 0, 0, time.Local) {
		t.Fatalf("today → %v", got)
	}
	if got := periodSince("month", now); got != time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local) {
		t.Fatalf("month → %v", got)
	}
	if got := periodSince("all", now); !got.IsZero() {
		t.Fatalf("all → %v, want zero", got)
	}
	if got := periodSince("garbage", now); got != periodSince("30d", now) {
		t.Fatalf("default должен совпадать с 30d: %v", got)
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
