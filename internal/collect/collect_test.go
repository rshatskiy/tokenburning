package collect

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rshatskiy/tokenburning/internal/platform"
	"github.com/rshatskiy/tokenburning/internal/pricing"
	"github.com/rshatskiy/tokenburning/internal/store"
)

func TestRunCollectsOne(t *testing.T) {
	home := t.TempDir()
	projDir := filepath.Join(home, ".claude", "projects", "-proj")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"assistant","requestId":"req_C1","sessionId":"s","cwd":"/p","timestamp":"2026-06-07T10:00:00.000Z","message":{"id":"m","model":"claude-opus-4-8","usage":{"input_tokens":1000000,"output_tokens":0}}}` + "\n"
	if err := os.WriteFile(filepath.Join(projDir, "a.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	// Isolate all three adapters to temp dirs so we pick up exactly one event.
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows: os.UserHomeDir читает USERPROFILE
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	t.Setenv("CODEX_HOME", filepath.Join(home, "no-codex"))

	paths, err := platform.Detect()
	if err != nil {
		t.Fatal(err)
	}
	cat, err := pricing.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	res, err := Run(db, cat, paths, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Collected != 1 {
		t.Fatalf("ожидалось Collected=1, получено %d", res.Collected)
	}

	rows, err := db.SummaryByModel(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rows {
		if r.Model == "claude-opus-4-8" {
			found = true
		}
	}
	if !found {
		t.Fatalf("модель claude-opus-4-8 не найдена в SummaryByModel: %v", rows)
	}
}

// Инкрементальный Run: неизменённые файлы пропускаются по курсору,
// дописанный хвост даёт только новые события.
func TestRunIncremental(t *testing.T) {
	home := t.TempDir()
	projDir := filepath.Join(home, ".claude", "projects", "-proj")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	line := func(id string) string {
		return `{"type":"assistant","requestId":"` + id + `","sessionId":"s","cwd":"/p","timestamp":"2026-06-07T10:00:00.000Z","message":{"id":"m","model":"claude-opus-4-8","usage":{"input_tokens":10,"output_tokens":5}}}` + "\n"
	}
	file := filepath.Join(projDir, "a.jsonl")
	if err := os.WriteFile(file, []byte(line("req_N1")), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows: os.UserHomeDir читает USERPROFILE
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	t.Setenv("CODEX_HOME", filepath.Join(home, "no-codex"))

	paths, err := platform.Detect()
	if err != nil {
		t.Fatal(err)
	}
	cat, err := pricing.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if res, err := Run(db, cat, paths, nil); err != nil || res.Collected != 1 {
		t.Fatalf("проход #1: res=%+v err=%v, ожидалось Collected=1", res, err)
	}
	// без изменений — файл должен быть пропущен целиком
	res2, err := Run(db, cat, paths, nil)
	if err != nil {
		t.Fatalf("проход #2: %v", err)
	}
	if res2.Collected != 0 {
		t.Fatalf("проход #2: Collected=%d, ожидалось 0 (файл не изменился)", res2.Collected)
	}
	if res2.Skipped == 0 {
		t.Fatalf("проход #2: Skipped=0, файл должен пропускаться по курсору")
	}
	// дописали строку — собирается только она
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(line("req_N2")); err != nil {
		t.Fatal(err)
	}
	f.Close()
	res3, err := Run(db, cat, paths, nil)
	if err != nil {
		t.Fatalf("проход #3: %v", err)
	}
	if res3.Collected != 1 {
		t.Fatalf("проход #3: Collected=%d, ожидалось 1 (только новая строка)", res3.Collected)
	}
	rows, err := db.SummaryByModel(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	var total int64
	for _, r := range rows {
		total += r.Events
	}
	if total != 2 {
		t.Fatalf("в БД %d событий, ожидалось 2", total)
	}
}
