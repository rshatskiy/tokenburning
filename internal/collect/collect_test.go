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
