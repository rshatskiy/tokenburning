package claudecode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/model"
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

// Инкрементальность: Collect с курсором читает только дописанный хвост,
// а возвращённый Offset двигается ровно по завершённым ('\n') строкам.
func TestCollectIncrementalOffset(t *testing.T) {
	line := func(id string) string {
		return `{"type":"assistant","requestId":"` + id + `","sessionId":"s","cwd":"/p","timestamp":"2026-06-07T10:00:00.000Z","message":{"model":"m","usage":{"input_tokens":1,"output_tokens":2}}}` + "\n"
	}
	p := filepath.Join(t.TempDir(), "inc.jsonl")
	if err := os.WriteFile(p, []byte(line("req_I1")+line("req_I2")), 0o644); err != nil {
		t.Fatal(err)
	}
	a := New()
	var events []model.Event
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { t.Fatalf("карантин: %v / %s", err, raw) }
	src := adapter.Source{Path: p}

	cur, err := a.Collect(src, adapter.Cursor{}, emit, quar)
	if err != nil {
		t.Fatalf("Collect #1: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("проход #1: событий %d, ожидалось 2", len(events))
	}
	want := int64(len(line("req_I1")) + len(line("req_I2")))
	if cur.Offset != want {
		t.Fatalf("Offset = %d, ожидалось %d", cur.Offset, want)
	}

	// дописываем третью строку и продолжаем с курсора
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(line("req_I3")); err != nil {
		t.Fatal(err)
	}
	f.Close()

	events = nil
	cur2, err := a.Collect(src, cur, emit, quar)
	if err != nil {
		t.Fatalf("Collect #2: %v", err)
	}
	if len(events) != 1 || events[0].EventID != "req_I3" {
		t.Fatalf("проход #2: ожидалось одно событие req_I3, получено %+v", events)
	}
	if cur2.Offset != want+int64(len(line("req_I3"))) {
		t.Fatalf("Offset #2 = %d, ожидалось %d", cur2.Offset, want+int64(len(line("req_I3"))))
	}

	// незавершённая (без '\n') строка обрабатывается, но offset через неё не шагает
	if err := os.WriteFile(p, []byte(line("req_I1")+`{"partial`), 0o644); err != nil {
		t.Fatal(err)
	}
	events = nil
	var quarantined int
	cur3, err := a.Collect(src, adapter.Cursor{}, emit, func([]byte, error) { quarantined++ })
	if err != nil {
		t.Fatalf("Collect #3: %v", err)
	}
	if cur3.Offset != int64(len(line("req_I1"))) {
		t.Fatalf("Offset #3 = %d, ожидалось %d (хвост без \\n не двигает курсор)", cur3.Offset, len(line("req_I1")))
	}
	if quarantined != 1 {
		t.Fatalf("незавершённый хвост должен попасть в карантин, quarantined=%d", quarantined)
	}
}
