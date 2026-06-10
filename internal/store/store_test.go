package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/rshatskiy/tokenburning/internal/model"
)

func sampleEvent(id string) model.Event {
	return model.Event{
		EventID: id, Tool: model.ToolClaudeCode, TS: time.Now(),
		Model: "claude-opus-4-7", BillingMode: model.BillingFlatEquivalent,
		Cost:   model.Cost{Amount: 1.5, Currency: "USD", Basis: model.BasisActual, PricingVersion: "2026-06-07"},
		Tokens: model.Tokens{Input: 10, Output: 20},
	}
}

// Регрессия first-run: ~/.tokenburning ещё не существует — Open обязан создать
// родительский каталог сам, а не падать с SQLITE_CANTOPEN(14) (как dashboard на свежей машине).
func TestOpenCreatesParentDir(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".tokenburning", "tokenburning.db")
	db, err := Open(p)
	if err != nil {
		t.Fatalf("Open с отсутствующим родительским каталогом: %v", err)
	}
	defer db.Close()
	if err := db.Insert([]model.Event{sampleEvent("first")}); err != nil {
		t.Fatalf("Insert: %v", err)
	}
}

func TestDefaultPath(t *testing.T) {
	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if filepath.Base(p) != "tokenburning.db" || filepath.Base(filepath.Dir(p)) != ".tokenburning" {
		t.Fatalf("неожиданный путь: %s", p)
	}
}

func TestInsertIsIdempotent(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "tokenburning.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	for i := 0; i < 3; i++ { // три раза один и тот же event_id
		if err := db.Insert([]model.Event{sampleEvent("req_A")}); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}
	rows, err := db.SummaryByModel(time.Time{})
	if err != nil {
		t.Fatalf("SummaryByModel: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("строк сводки %d, ожидалось 1", len(rows))
	}
	if rows[0].Events != 1 {
		t.Fatalf("Events = %d, ожидалось 1 (идемпотентность нарушена)", rows[0].Events)
	}
	if rows[0].CostAmount != 1.5 {
		t.Fatalf("CostAmount = %f, ожидалось 1.5", rows[0].CostAmount)
	}

	// Другой event_id должен добавить строку.
	if err := db.Insert([]model.Event{sampleEvent("req_B")}); err != nil {
		t.Fatalf("Insert req_B: %v", err)
	}
	rows, err = db.SummaryByModel(time.Time{})
	if err != nil {
		t.Fatalf("SummaryByModel: %v", err)
	}
	if rows[0].Events != 2 {
		t.Fatalf("Events = %d, ожидалось 2 после добавления другого event_id", rows[0].Events)
	}
}

func TestSummaryByTool(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "tokenburning.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	e1 := sampleEvent("a")
	e2 := sampleEvent("b")
	e2.Tool = model.ToolCodex
	e2.Tokens = model.Tokens{Input: 100, Output: 50}
	if err := db.Insert([]model.Event{e1, e2}); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	rows, err := db.SummaryByTool(time.Time{})
	if err != nil {
		t.Fatalf("SummaryByTool: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("строк %d, ожидалось 2", len(rows))
	}
	var codex *ToolSummary
	for i := range rows {
		if rows[i].Tool == string(model.ToolCodex) {
			codex = &rows[i]
		}
	}
	if codex == nil || codex.Events != 1 || codex.Tokens != 150 {
		t.Fatalf("codex summary неверна: %+v", codex)
	}
}
