package cline

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/platform"
)

// Основной сценарий: одно событие на каждую api_req_started с реальными
// токенами; нулевые пропускаются, битые — в карантин, индексы стабильны.
func TestCollectEmitsApiReqEvents(t *testing.T) {
	a := New()
	var events []model.Event
	var quarantined int
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { quarantined++ }

	src := adapter.Source{Path: "testdata/fixtures/v1/tasks/1717000000000"}
	cur, err := a.Collect(src, adapter.Cursor{}, emit, quar)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if cur != (adapter.Cursor{}) {
		t.Errorf("Cursor должен быть пустым, получен %+v", cur)
	}
	if len(events) != 2 {
		t.Fatalf("эмитнуто %d событий, ожидалось 2 (нулевые и битые api_req_started не в счёт)", len(events))
	}
	if quarantined != 1 {
		t.Fatalf("в карантине %d, ожидалось 1 (битый text внутри api_req_started)", quarantined)
	}

	e := events[0]
	if e.EventID != "1717000000000:0" {
		t.Errorf("EventID = %q, want 1717000000000:0", e.EventID)
	}
	if e.Tool != model.ToolCline {
		t.Errorf("Tool = %q", e.Tool)
	}
	if e.Model != "claude-sonnet-4-5" {
		t.Errorf("Model = %q, want claude-sonnet-4-5 (из <model> в истории, без префикса провайдера)", e.Model)
	}
	if e.BillingMode != model.BillingAPIUsage {
		t.Errorf("BillingMode = %q", e.BillingMode)
	}
	if e.Tokens.Input != 1200 || e.Tokens.Output != 350 || e.Tokens.CacheRead != 4500 || e.Tokens.Cache5m != 800 {
		t.Errorf("Tokens неверны: %+v", e.Tokens)
	}
	if want := time.UnixMilli(1717000000500).UTC(); !e.TS.Equal(want) {
		t.Errorf("TS = %v, want %v", e.TS, want)
	}
	if e.SessionID != "1717000000000" {
		t.Errorf("SessionID = %q, want task id", e.SessionID)
	}
	if e.ProjectKey != "/Users/dev/myproject" {
		t.Errorf("ProjectKey = %q", e.ProjectKey)
	}
	if len(e.ExtraRaw) == 0 {
		t.Error("ExtraRaw пуст — сырьё не сохранено")
	}

	// Второе событие: индекс 3, а не 1 — пропущенные/битые записи
	// тоже занимают индекс, иначе EventID последующих поплывут.
	e2 := events[1]
	if e2.EventID != "1717000000000:3" {
		t.Errorf("EventID #2 = %q, want 1717000000000:3", e2.EventID)
	}
	if e2.Tokens.Input != 20 || e2.Tokens.Output != 7 || e2.Tokens.CacheRead != 0 || e2.Tokens.Cache5m != 0 {
		t.Errorf("Tokens #2 неверны: %+v", e2.Tokens)
	}
}

// Идемпотентность: повторный Collect даёт те же EventID.
func TestCollectIdempotentEventIDs(t *testing.T) {
	a := New()
	src := adapter.Source{Path: "testdata/fixtures/v1/tasks/1717000000000"}
	collect := func() []string {
		var ids []string
		emit := func(e model.Event) { ids = append(ids, e.EventID) }
		if _, err := a.Collect(src, adapter.Cursor{}, emit, func([]byte, error) {}); err != nil {
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
			t.Errorf("EventID #%d нестабилен: %q vs %q", i, first[i], second[i])
		}
	}
}

// Целиком битый ui_messages.json — карантин, не ошибка и не паника.
func TestCollectQuarantinesBrokenFile(t *testing.T) {
	a := New()
	var events []model.Event
	var quarantined int
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { quarantined++ }

	src := adapter.Source{Path: "testdata/fixtures/v1/broken/tasks/1718000000000"}
	if _, err := a.Collect(src, adapter.Cursor{}, emit, quar); err != nil {
		t.Fatalf("Collect не должен падать на битом файле: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("эмитнуто %d событий, ожидалось 0", len(events))
	}
	if quarantined != 1 {
		t.Fatalf("в карантине %d, ожидалось 1 (весь файл)", quarantined)
	}
}

// Без api_conversation_history.json и без model в записи — модель "unknown".
func TestCollectModelUnknownWithoutHistory(t *testing.T) {
	taskDir := filepath.Join(t.TempDir(), "1719000000000")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ui := `[{"ts":1719000000100,"type":"say","say":"api_req_started","text":"{\"tokensIn\":5,\"tokensOut\":9}"}]`
	if err := os.WriteFile(filepath.Join(taskDir, "ui_messages.json"), []byte(ui), 0o644); err != nil {
		t.Fatal(err)
	}

	a := New()
	var events []model.Event
	emit := func(e model.Event) { events = append(events, e) }
	quar := func(raw []byte, err error) { t.Fatalf("неожиданный карантин: %v / %s", err, raw) }
	if _, err := a.Collect(adapter.Source{Path: taskDir}, adapter.Cursor{}, emit, quar); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("эмитнуто %d, ожидалось 1", len(events))
	}
	if events[0].Model != "unknown" {
		t.Errorf("Model = %q, want unknown", events[0].Model)
	}
}

// Поле model прямо в api_req_started имеет приоритет над историей.
func TestCollectModelFromApiReqEntry(t *testing.T) {
	taskDir := filepath.Join(t.TempDir(), "1719100000000")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ui := `[{"ts":1719100000100,"type":"say","say":"api_req_started","text":"{\"tokensIn\":5,\"tokensOut\":9,\"model\":\"gpt-5.2-codex\"}"}]`
	if err := os.WriteFile(filepath.Join(taskDir, "ui_messages.json"), []byte(ui), 0o644); err != nil {
		t.Fatal(err)
	}

	a := New()
	var events []model.Event
	emit := func(e model.Event) { events = append(events, e) }
	if _, err := a.Collect(adapter.Source{Path: taskDir}, adapter.Cursor{}, emit, func([]byte, error) {}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(events) != 1 || events[0].Model != "gpt-5.2-codex" {
		t.Fatalf("ожидалось одно событие с моделью gpt-5.2-codex, получено %+v", events)
	}
}

// Discover: находит задачи всех трёх расширений в VSCodeStorageDirs и
// standalone ~/.cline/data, дедуплицирует один task id между корнями
// (побеждает свежайший ui_messages.json).
func TestDiscoverFindsAllRootsAndDedupes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)        // os.UserHomeDir → наш временный $HOME
	t.Setenv("USERPROFILE", home) // Windows читает USERPROFILE

	userDir := filepath.Join(t.TempDir(), "Code", "User")

	mkTask := func(root, taskID string, mtime time.Time) string {
		dir := filepath.Join(root, "tasks", taskID)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(dir, "ui_messages.json")
		if err := os.WriteFile(p, []byte("[]"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mtime, mtime); err != nil {
			t.Fatal(err)
		}
		return dir
	}

	base := time.Now().Add(-time.Hour)
	clineRoot := filepath.Join(userDir, "globalStorage", "saoudrizwan.claude-dev")
	rooRoot := filepath.Join(userDir, "globalStorage", "rooveterinaryinc.roo-cline")
	kiloRoot := filepath.Join(userDir, "globalStorage", "kilocode.kilo-code")
	standaloneRoot := filepath.Join(home, ".cline", "data")

	mkTask(clineRoot, "100", base)
	rooDir := mkTask(rooRoot, "200", base)
	kiloDir := mkTask(kiloRoot, "300", base)
	// task 100 продублирован в standalone-корне со свежим mtime — он и победит
	dupDir := mkTask(standaloneRoot, "100", base.Add(30*time.Minute))
	// каталог задачи без ui_messages.json игнорируется
	if err := os.MkdirAll(filepath.Join(clineRoot, "tasks", "400"), 0o755); err != nil {
		t.Fatal(err)
	}

	a := New()
	srcs, err := a.Discover(platform.Paths{VSCodeStorageDirs: []string{userDir}})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	got := make(map[string]bool, len(srcs))
	for _, s := range srcs {
		got[s.Path] = true
	}
	want := []string{dupDir, rooDir, kiloDir}
	if len(srcs) != len(want) {
		t.Fatalf("найдено %d источников %v, ожидалось %d: %v", len(srcs), got, len(want), want)
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("источник %s не найден, есть: %v", w, got)
		}
	}
}

// Пустые пути → пусто, без ошибок (на чужой машине расширений может не быть).
func TestDiscoverEmptyPaths(t *testing.T) {
	emptyHome := t.TempDir()
	t.Setenv("HOME", emptyHome)
	t.Setenv("USERPROFILE", emptyHome)
	a := New()
	srcs, err := a.Discover(platform.Paths{})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(srcs) != 0 {
		t.Fatalf("ожидалось 0 источников, найдено %d", len(srcs))
	}
}
