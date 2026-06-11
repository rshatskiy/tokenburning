package gemini

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/platform"
)

func TestCollectSingleJSON(t *testing.T) {
	a := New()
	var events []model.Event
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { t.Fatalf("неожиданный карантин: %v / %s", err, raw) }

	src := adapter.Source{Path: "testdata/fixtures/v1/8c1d2e3f4a5b/chats/session-f47ac10b.json"}
	if _, err := a.Collect(src, adapter.Cursor{}, emit, quar); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	// 2 события: m-002 и сообщение без id; пустой usage и сообщение
	// без модели событий не дают.
	if len(events) != 2 {
		t.Fatalf("эмитнуто %d событий, ожидалось 2", len(events))
	}

	e := events[0]
	if e.EventID != "m-002" {
		t.Errorf("EventID = %q, want m-002", e.EventID)
	}
	if e.Tool != model.ToolGemini {
		t.Errorf("Tool = %q", e.Tool)
	}
	if e.Model != "gemini-2.5-pro" {
		t.Errorf("Model = %q", e.Model)
	}
	// input у Gemini включает cached: 5200 - 4100 = 1100 свежих.
	if e.Tokens.Input != 1100 || e.Tokens.Output != 310 || e.Tokens.CacheRead != 4100 || e.Tokens.Reasoning != 128 {
		t.Errorf("Tokens неверны: %+v", e.Tokens)
	}
	if e.SessionID != "f47ac10b-58cc-4372-a567-0e02b2c3d479" {
		t.Errorf("SessionID = %q", e.SessionID)
	}
	if e.ProjectKey != "8c1d2e3f4a5b" {
		t.Errorf("ProjectKey = %q, want 8c1d2e3f4a5b (каталог project)", e.ProjectKey)
	}
	if e.BillingMode != model.BillingFlatEquivalent {
		t.Errorf("BillingMode = %q", e.BillingMode)
	}
	if e.TS.UTC().Format("2006-01-02T15:04:05.000Z") != "2026-06-10T09:15:09.844Z" {
		t.Errorf("TS = %v", e.TS)
	}
	if len(e.ExtraRaw) == 0 {
		t.Error("ExtraRaw пуст — сырьё не сохранено для бэкфилла")
	}

	// Сообщение без id получает стабильный фоллбэк <имя-файла>#<index>.
	e2 := events[1]
	if e2.EventID != "session-f47ac10b.json#4" {
		t.Errorf("фоллбэк EventID = %q, want session-f47ac10b.json#4", e2.EventID)
	}
	if e2.Model != "gemini-2.5-flash" {
		t.Errorf("Model = %q", e2.Model)
	}
	// cached отсутствует → input целиком свежий.
	if e2.Tokens.Input != 800 || e2.Tokens.Output != 95 || e2.Tokens.CacheRead != 0 {
		t.Errorf("Tokens неверны: %+v", e2.Tokens)
	}
}

func TestCollectJSONL(t *testing.T) {
	a := New()
	var events []model.Event
	var quarantined int
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { quarantined++ }

	src := adapter.Source{Path: "testdata/fixtures/v1/8c1d2e3f4a5b/chats/session-a1b2c3d4.jsonl"}
	if _, err := a.Collect(src, adapter.Cursor{}, emit, quar); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("эмитнуто %d событий, ожидалось 2", len(events))
	}
	if quarantined != 1 {
		t.Fatalf("в карантине %d, ожидалось 1 (не-JSON строка)", quarantined)
	}

	if events[0].EventID != "m-102" || events[0].Model != "gemini-3-flash-preview" {
		t.Errorf("событие #1: %q / %q", events[0].EventID, events[0].Model)
	}
	if events[0].Tokens.Input != 600 || events[0].Tokens.CacheRead != 900 || events[0].Tokens.Reasoning != 64 {
		t.Errorf("Tokens #1 неверны: %+v", events[0].Tokens)
	}
	if events[0].SessionID != "a1b2c3d4-e5f6-4789-a012-b345c678d901" {
		t.Errorf("SessionID = %q (заголовок JSONL не подхвачен)", events[0].SessionID)
	}

	// Полный кэш-хит: input == cached → свежих 0, в минус не уходим.
	if events[1].Tokens.Input != 0 || events[1].Tokens.CacheRead != 2400 || events[1].Tokens.Output != 510 {
		t.Errorf("Tokens #2 неверны: %+v", events[1].Tokens)
	}
}

func TestCollectQuarantinesBrokenMessages(t *testing.T) {
	a := New()
	var events []model.Event
	var quarantined int
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { quarantined++ }

	src := adapter.Source{Path: "testdata/fixtures/v1/8c1d2e3f4a5b/chats/session-broken.json"}
	if _, err := a.Collect(src, adapter.Cursor{}, emit, quar); err != nil {
		t.Fatalf("Collect не должен падать на битых записях: %v", err)
	}
	if len(events) != 1 || events[0].EventID != "b-001" {
		t.Fatalf("ожидалось одно событие b-001, получено %+v", events)
	}
	// b-002 (битый timestamp) + b-003 (tokens не объект).
	if quarantined != 2 {
		t.Fatalf("в карантине %d, ожидалось 2", quarantined)
	}
}

func TestDiscover(t *testing.T) {
	a := New()
	paths := platform.Paths{GeminiTmp: filepath.Join("testdata", "fixtures", "v1")}
	sources, err := a.Discover(paths)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(sources) != 3 {
		t.Fatalf("найдено %d источников, ожидалось 3: %+v", len(sources), sources)
	}
}

func TestDiscoverIgnoresJunk(t *testing.T) {
	// project без chats/, файл не-JSON и файл в корне не должны мешать.
	root := t.TempDir()
	chats := filepath.Join(root, "proj-a", "chats")
	if err := os.MkdirAll(chats, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "proj-b"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, dir := range map[string]string{
		"session-x.json":  chats,
		"session-y.jsonl": chats,
		"notes.txt":       chats,
		"stray.json":      root, // файл вместо каталога project
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	a := New()
	sources, err := a.Discover(platform.Paths{GeminiTmp: root})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(sources) != 2 {
		t.Fatalf("найдено %d источников, ожидалось 2: %+v", len(sources), sources)
	}
}

func TestDiscoverEmptyDir(t *testing.T) {
	a := New()
	// Несуществующий каталог — не ошибка: Gemini CLI может быть не установлен.
	paths := platform.Paths{GeminiTmp: filepath.Join(t.TempDir(), "нет-такого")}
	sources, err := a.Discover(paths)
	if err != nil {
		t.Fatalf("Discover не должен падать на отсутствующем каталоге: %v", err)
	}
	if len(sources) != 0 {
		t.Fatalf("найдено %d источников, ожидалось 0", len(sources))
	}
}
