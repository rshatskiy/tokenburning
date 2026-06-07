package claudecode

import (
	"testing"

	"github.com/lens/lens/internal/adapter"
	"github.com/lens/lens/internal/model"
)

func TestCollectEmitsOnlyAssistantEvents(t *testing.T) {
	a := New()
	var events []model.Event
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { t.Fatalf("неожиданный карантин: %v / %s", err, raw) }

	src := adapter.Source{Path: "testdata/fixtures/v1/sample.jsonl"}
	if _, err := a.Collect(src, adapter.Cursor{}, emit, quar); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("эмитнуто %d событий, ожидалось 1 (только assistant)", len(events))
	}
	e := events[0]
	if e.EventID != "req_TEST001" {
		t.Errorf("EventID = %q, want req_TEST001", e.EventID)
	}
	if e.Tool != model.ToolClaudeCode {
		t.Errorf("Tool = %q", e.Tool)
	}
	if e.Model != "claude-opus-4-7" {
		t.Errorf("Model = %q", e.Model)
	}
	if e.Tokens.Input != 6 || e.Tokens.Output != 204 || e.Tokens.CacheRead != 18140 || e.Tokens.Cache1h != 20012 || e.Tokens.Cache5m != 0 {
		t.Errorf("Tokens неверны: %+v", e.Tokens)
	}
	if e.SessionID != "sess-1" {
		t.Errorf("SessionID = %q", e.SessionID)
	}
	if len(e.ExtraRaw) == 0 {
		t.Error("ExtraRaw пуст — сырьё не сохранено для бэкфилла")
	}
}

func TestCollectQuarantinesBrokenLine(t *testing.T) {
	a := New()
	var events []model.Event
	var quarantined int
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { quarantined++ }

	src := adapter.Source{Path: "testdata/fixtures/v1/broken.jsonl"}
	if _, err := a.Collect(src, adapter.Cursor{}, emit, quar); err != nil {
		t.Fatalf("Collect не должен падать на битой строке: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("эмитнуто %d событий, ожидалось 1 (валидная строка)", len(events))
	}
	if quarantined != 1 {
		t.Fatalf("в карантине %d, ожидалось 1 (битая строка)", quarantined)
	}
}

func TestCollectCacheCreationFallback(t *testing.T) {
	a := New()
	var events []model.Event
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { t.Fatalf("неожиданный карантин: %v", err) }
	src := adapter.Source{Path: "testdata/fixtures/v1/cache_fallback.jsonl"}
	if _, err := a.Collect(src, adapter.Cursor{}, emit, quar); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("эмитнуто %d, ожидалось 1", len(events))
	}
	if events[0].Tokens.Cache5m != 500 || events[0].Tokens.Cache1h != 0 {
		t.Fatalf("fallback кэша неверен: Cache5m=%d Cache1h=%d, ожидалось 500/0",
			events[0].Tokens.Cache5m, events[0].Tokens.Cache1h)
	}
}

func TestCollectQuarantinesEmptyRequestID(t *testing.T) {
	a := New()
	var events []model.Event
	var quarantined int
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { quarantined++ }
	src := adapter.Source{Path: "testdata/fixtures/v1/empty_reqid.jsonl"}
	if _, err := a.Collect(src, adapter.Cursor{}, emit, quar); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("эмитнуто %d, ожидалось 0 (пустой requestId)", len(events))
	}
	if quarantined != 1 {
		t.Fatalf("в карантине %d, ожидалось 1", quarantined)
	}
}
