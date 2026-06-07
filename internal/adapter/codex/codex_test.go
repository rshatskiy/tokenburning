package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/platform"
)

func TestCollectAggregatesSession(t *testing.T) {
	a := New()
	var events []model.Event
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { t.Fatalf("неожиданный карантин: %v", err) }
	src := adapter.Source{Path: "testdata/fixtures/v1/rollout-sample.jsonl"}
	if _, err := a.Collect(src, adapter.Cursor{}, emit, quar); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("эмитнуто %d, ожидалось 1 событие на сессию", len(events))
	}
	e := events[0]
	if e.EventID != "sess-xyz" {
		t.Errorf("EventID = %q, want sess-xyz", e.EventID)
	}
	if e.Tool != model.ToolCodex {
		t.Errorf("Tool = %q", e.Tool)
	}
	// last суммируется (дельты): Input 100+5=105, Output 50+3=53, CacheRead 20, Reasoning 10
	if e.Tokens.Input != 105 || e.Tokens.Output != 53 || e.Tokens.CacheRead != 20 || e.Tokens.Reasoning != 10 {
		t.Errorf("разбивка токенов неверна: %+v", e.Tokens)
	}
	// Total — последний кумулятив total_token_usage.total_tokens
	if e.Tokens.Total != 368 {
		t.Errorf("Total = %d, want 368 (кумулятив)", e.Tokens.Total)
	}
	if e.Model != "unknown" {
		t.Errorf("Model = %q, want unknown (локально модели нет)", e.Model)
	}
	if e.ProjectKey != "/Users/dev/proj" {
		t.Errorf("ProjectKey = %q", e.ProjectKey)
	}
}

func TestCollectQuarantinesBrokenLineStillEmits(t *testing.T) {
	a := New()
	var events []model.Event
	var q int
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { q++ }
	src := adapter.Source{Path: "testdata/fixtures/v1/rollout-broken.jsonl"}
	if _, err := a.Collect(src, adapter.Cursor{}, emit, quar); err != nil {
		t.Fatalf("Collect не должен падать: %v", err)
	}
	if len(events) != 1 || q != 1 {
		t.Fatalf("events=%d quarantined=%d, ожидалось 1/1", len(events), q)
	}
}

func TestDiscoverSkipsImportedSessions(t *testing.T) {
	home := t.TempDir()
	sessions := filepath.Join(home, "sessions", "2026", "06", "07")
	if err := os.MkdirAll(sessions, 0o755); err != nil {
		t.Fatal(err)
	}
	keepUUID := "019ea0b9-272b-7ba0-a8bf-0bae3e702ae6"
	impUUID := "019ea0b7-d631-7731-b1b1-242b28b0e3e8"
	for _, u := range []string{keepUUID, impUUID} {
		p := filepath.Join(sessions, "rollout-2026-06-07T11-15-49-"+u+".jsonl")
		if err := os.WriteFile(p, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// манифест импорта помечает impUUID
	manifest := `{"records":[{"source_path":"/x.jsonl","content_sha256":"a","imported_thread_id":"` + impUUID + `","imported_at":1}]}`
	if err := os.WriteFile(filepath.Join(home, "external_agent_session_imports.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	a := New()
	srcs, err := a.Discover(platform.Paths{CodexSessions: filepath.Join(home, "sessions")})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(srcs) != 1 {
		t.Fatalf("ожидался 1 источник (импортированный пропущен), получено %d", len(srcs))
	}
	if !strings.Contains(srcs[0].Path, keepUUID) {
		t.Fatalf("остаться должен не-импортированный rollout, got %s", srcs[0].Path)
	}
}
