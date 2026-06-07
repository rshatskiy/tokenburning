package codex

import (
	"testing"

	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/model"
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
