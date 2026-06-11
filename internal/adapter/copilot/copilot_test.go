package copilot

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/platform"
)

func fixturePaths() platform.Paths {
	return platform.Paths{
		CopilotSessions:   filepath.Join("testdata", "fixtures", "cli"),
		VSCodeStorageDirs: []string{filepath.Join("testdata", "fixtures", "vscode", "User")},
	}
}

func TestDiscoverFindsBothSources(t *testing.T) {
	a := New()
	srcs, err := a.Discover(fixturePaths())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	// 2 CLI-сессии + 1 транскрипт VS Code
	if len(srcs) != 3 {
		t.Fatalf("найдено %d источников, ожидалось 3: %+v", len(srcs), srcs)
	}
}

func TestDiscoverMissingDirsIsNotError(t *testing.T) {
	a := New()
	missing := filepath.Join(t.TempDir(), "nope")
	srcs, err := a.Discover(platform.Paths{
		CopilotSessions:   filepath.Join(missing, "session-state"),
		VSCodeStorageDirs: []string{filepath.Join(missing, "Code", "User")},
	})
	if err != nil {
		t.Fatalf("отсутствие каталогов не должно быть ошибкой: %v", err)
	}
	if len(srcs) != 0 {
		t.Fatalf("ожидалось 0 источников, получено %d", len(srcs))
	}
}

func TestCollectLegacyCLISession(t *testing.T) {
	a := New()
	var events []model.Event
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { t.Fatalf("неожиданный карантин: %v", err) }

	src := adapter.Source{Path: filepath.Join("testdata", "fixtures", "cli", "sess-abc-123", "events.jsonl")}
	cur, err := a.Collect(src, adapter.Cursor{}, emit, quar)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if cur != (adapter.Cursor{}) {
		t.Errorf("курсор должен быть пустым, получено %+v", cur)
	}
	// m2 пропущен (outputTokens=0) — остаются m1 и m3
	if len(events) != 2 {
		t.Fatalf("эмитнуто %d событий, ожидалось 2", len(events))
	}

	e := events[0]
	if e.EventID != "copilot:sess-abc-123:m1" {
		t.Errorf("EventID = %q", e.EventID)
	}
	if e.Tool != model.ToolCopilot {
		t.Errorf("Tool = %q", e.Tool)
	}
	if e.Model != "gpt-5" {
		t.Errorf("Model = %q, want gpt-5 (из session.model_change)", e.Model)
	}
	// CLI-формат: input локально не пишется, output — реальный из файла
	if e.Tokens.Input != 0 || e.Tokens.Output != 123 {
		t.Errorf("Tokens = %+v, want Input=0 Output=123", e.Tokens)
	}
	if e.Tokens.CacheRead != 0 || e.Tokens.Cache1h != 0 || e.Tokens.Cache5m != 0 {
		t.Errorf("кэш-полей быть не должно: %+v", e.Tokens)
	}
	if e.SessionID != "sess-abc-123" {
		t.Errorf("SessionID = %q", e.SessionID)
	}
	if e.ProjectKey != "/Users/dev/myproj" {
		t.Errorf("ProjectKey = %q, want /Users/dev/myproj (из workspace.yaml)", e.ProjectKey)
	}
	wantTS := time.Date(2026, 6, 1, 10, 0, 5, 0, time.UTC)
	if !e.TS.Equal(wantTS) {
		t.Errorf("TS = %v, want %v", e.TS, wantTS)
	}

	e2 := events[1]
	if e2.EventID != "copilot:sess-abc-123:m3" {
		t.Errorf("EventID = %q", e2.EventID)
	}
	if e2.Model != "claude-sonnet-4-5" {
		t.Errorf("Model = %q, want claude-sonnet-4-5 (явный data.model)", e2.Model)
	}
	if e2.Tokens.Output != 50 {
		t.Errorf("Output = %d, want 50", e2.Tokens.Output)
	}
}

func TestCollectVSCodeTranscriptEstimatesTokens(t *testing.T) {
	a := New()
	var events []model.Event
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { t.Fatalf("неожиданный карантин: %v", err) }

	src := adapter.Source{Path: filepath.Join(
		"testdata", "fixtures", "vscode", "User", "workspaceStorage", "a1b2c3",
		"GitHub.copilot-chat", "transcripts", "0198c3a0-1111-2222-3333-444455556666.jsonl")}
	if _, err := a.Collect(src, adapter.Cursor{}, emit, quar); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	// t2 пропущен (пустая заглушка стриминга) — остаются t1 и t3
	if len(events) != 2 {
		t.Fatalf("эмитнуто %d событий, ожидалось 2", len(events))
	}

	e := events[0]
	if e.EventID != "copilot:0198c3a0-1111-2222-3333-444455556666:t1" {
		t.Errorf("EventID = %q", e.EventID)
	}
	// Модель выведена из префиксов toolu_* (два хита) → Anthropic auto
	if e.Model != "copilot-anthropic-auto" {
		t.Errorf("Model = %q, want copilot-anthropic-auto", e.Model)
	}
	// Оценка по длине: user "Fix the bug in parser" (21) → 6,
	// content "Fixed the parser bug." (21) → 6, reasoning "Thinking..." (11) → 3
	if e.Tokens.Input != 6 || e.Tokens.Output != 6 || e.Tokens.Reasoning != 3 {
		t.Errorf("Tokens = %+v, want Input=6 Output=6 Reasoning=3", e.Tokens)
	}
	if e.Tokens.CacheRead != 0 || e.Tokens.Cache1h != 0 || e.Tokens.Cache5m != 0 {
		t.Errorf("кэш-полей быть не должно: %+v", e.Tokens)
	}
	if e.SessionID != "0198c3a0-1111-2222-3333-444455556666" {
		t.Errorf("SessionID = %q", e.SessionID)
	}
	if e.ProjectKey != "/Users/dev/my project" {
		t.Errorf("ProjectKey = %q, want /Users/dev/my project (из workspace.json)", e.ProjectKey)
	}
	wantTS := time.Date(2026, 6, 3, 12, 0, 5, 0, time.UTC)
	if !e.TS.Equal(wantTS) {
		t.Errorf("TS = %v, want %v", e.TS, wantTS)
	}

	// t3: outputTokens задан в файле → используем его; user уже израсходован → Input 0;
	// toolRequests-строка (битый формат) не роняет разбор.
	e2 := events[1]
	if e2.EventID != "copilot:0198c3a0-1111-2222-3333-444455556666:t3" {
		t.Errorf("EventID = %q", e2.EventID)
	}
	if e2.Tokens.Input != 0 || e2.Tokens.Output != 7 || e2.Tokens.Reasoning != 0 {
		t.Errorf("Tokens = %+v, want Input=0 Output=7 Reasoning=0", e2.Tokens)
	}
}

func TestCollectQuarantinesBrokenLineStillEmits(t *testing.T) {
	a := New()
	var events []model.Event
	var quarantined int
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { quarantined++ }

	src := adapter.Source{Path: filepath.Join("testdata", "fixtures", "cli", "broken-sess", "events.jsonl")}
	if _, err := a.Collect(src, adapter.Cursor{}, emit, quar); err != nil {
		t.Fatalf("Collect не должен падать на битой строке: %v", err)
	}
	if quarantined != 1 {
		t.Errorf("в карантине %d записей, ожидалась 1", quarantined)
	}
	if len(events) != 1 {
		t.Fatalf("эмитнуто %d событий, ожидалось 1", len(events))
	}
	if events[0].Model != "gpt-4o" || events[0].Tokens.Output != 10 {
		t.Errorf("событие после битой строки разобрано неверно: %+v", events[0])
	}
	// ProjectKey пуст: у broken-sess нет workspace.yaml — это не ошибка
	if events[0].ProjectKey != "" {
		t.Errorf("ProjectKey = %q, want пусто", events[0].ProjectKey)
	}
}

func TestCollectIdempotentEventIDs(t *testing.T) {
	// Повторный сбор того же файла даёт те же EventID — дедуп выше по стеку.
	a := New()
	collect := func() []string {
		var ids []string
		emit := func(e model.Event) { ids = append(ids, e.EventID) }
		quar := func(raw []byte, err error) {}
		src := adapter.Source{Path: filepath.Join("testdata", "fixtures", "cli", "sess-abc-123", "events.jsonl")}
		if _, err := a.Collect(src, adapter.Cursor{}, emit, quar); err != nil {
			t.Fatalf("Collect: %v", err)
		}
		return ids
	}
	first, second := collect(), collect()
	if len(first) != len(second) {
		t.Fatalf("разное число событий: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("EventID нестабилен: %q vs %q", first[i], second[i])
		}
	}
}

func TestCapabilities(t *testing.T) {
	caps := New().Capabilities()
	if caps.HasTokens != model.FidelityPartial {
		t.Errorf("HasTokens = %q, want partial (оценка)", caps.HasTokens)
	}
	if caps.HasCache {
		t.Error("HasCache должен быть false — источник кэш не отдаёт")
	}
	if !caps.HasSessions {
		t.Error("HasSessions должен быть true")
	}
	if New().Name() != model.ToolCopilot {
		t.Errorf("Name = %q", New().Name())
	}
}
