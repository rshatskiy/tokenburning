package daemon

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunOnceCollects(t *testing.T) {
	home := t.TempDir()
	projDir := filepath.Join(home, ".claude", "projects", "-proj")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"assistant","requestId":"req_D1","sessionId":"s","cwd":"/p","timestamp":"2026-06-07T10:00:00.000Z","message":{"id":"m","model":"claude-opus-4-8","usage":{"input_tokens":1000000,"output_tokens":0}}}` + "\n"
	if err := os.WriteFile(filepath.Join(projDir, "a.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	// Isolate all adapters to temp dirs so we pick up exactly one event.
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	t.Setenv("CODEX_HOME", filepath.Join(home, "no-codex"))

	dbPath := filepath.Join(t.TempDir(), "daemon_test.db")
	res, err := RunOnce(Options{DBPath: dbPath})
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if res.Collected != 1 {
		t.Fatalf("ожидалось Collected=1, получено %d", res.Collected)
	}
}

func TestRunCancels(t *testing.T) {
	// Prepare a minimal env so RunOnce succeeds on the first pass.
	home := t.TempDir()
	projDir := filepath.Join(home, ".claude", "projects", "-proj")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"assistant","requestId":"req_D2","sessionId":"s","cwd":"/p","timestamp":"2026-06-07T10:00:00.000Z","message":{"id":"m","model":"claude-opus-4-8","usage":{"input_tokens":100,"output_tokens":0}}}` + "\n"
	if err := os.WriteFile(filepath.Join(projDir, "b.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	// Isolate all adapters to temp dirs.
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	t.Setenv("CODEX_HOME", filepath.Join(home, "no-codex"))

	dbPath := filepath.Join(t.TempDir(), "daemon_cancel_test.db")
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Options{
			DBPath:   dbPath,
			Interval: time.Hour, // long interval so we block on ticker
		})
	}()

	// Give time for the first RunOnce to finish, then cancel.
	time.Sleep(300 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancellation within 2s")
	}
}
