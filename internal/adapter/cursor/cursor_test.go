package cursor

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/model"
	_ "modernc.org/sqlite"
)

func writeCursorDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state.vscdb")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE cursorDiskKV (key TEXT PRIMARY KEY, value BLOB)`); err != nil {
		t.Fatal(err)
	}
	rows := []struct{ k, v string }{
		// assistant (type 2) с токенами и моделью
		{"bubbleId:comp1:msgA", `{"type":2,"bubbleId":"msgA","createdAt":"2026-03-19T14:24:57.729Z","tokenCount":{"inputTokens":120,"outputTokens":40},"modelInfo":{"modelName":"default"}}`},
		// user (type 1) — игнорируется
		{"bubbleId:comp1:msgB", `{"type":1,"bubbleId":"msgB","createdAt":"2026-03-19T14:25:00.000Z"}`},
	}
	for _, r := range rows {
		if _, err := db.Exec(`INSERT INTO cursorDiskKV(key,value) VALUES(?,?)`, r.k, r.v); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestCollectEmitsAssistantBubbles(t *testing.T) {
	a := New()
	var events []model.Event
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { t.Fatalf("неожиданный карантин: %v", err) }
	src := adapter.Source{Path: writeCursorDB(t)}
	if _, err := a.Collect(src, adapter.Cursor{}, emit, quar); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("эмитнуто %d, ожидалось 1 (только assistant)", len(events))
	}
	e := events[0]
	if e.EventID != "bubbleId:comp1:msgA" {
		t.Errorf("EventID = %q", e.EventID)
	}
	if e.Tool != model.ToolCursor {
		t.Errorf("Tool = %q", e.Tool)
	}
	if e.Tokens.Input != 120 || e.Tokens.Output != 40 {
		t.Errorf("Tokens = %+v", e.Tokens)
	}
	if e.Model != "default" {
		t.Errorf("Model = %q", e.Model)
	}
	if e.SessionID != "comp1" {
		t.Errorf("SessionID = %q, want comp1", e.SessionID)
	}
}

func TestCollectMissingTableIsEmpty(t *testing.T) {
	// БД без таблицы cursorDiskKV (как у части workspace-storage) — не ошибка, просто пусто.
	path := filepath.Join(t.TempDir(), "empty.vscdb")
	db, _ := sql.Open("sqlite", "file:"+path)
	db.Exec(`CREATE TABLE ItemTable (key TEXT, value BLOB)`)
	db.Close()
	a := New()
	var events []model.Event
	_, err := a.Collect(adapter.Source{Path: path}, adapter.Cursor{}, func(e model.Event) { events = append(events, e) }, func(raw []byte, err error) {})
	if err != nil {
		t.Fatalf("Collect не должен падать на БД без cursorDiskKV: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("эмитнуто %d, ожидалось 0", len(events))
	}
}
