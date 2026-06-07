package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunScanProducesSummary(t *testing.T) {
	home := t.TempDir()
	projDir := filepath.Join(home, ".claude", "projects", "-proj")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"assistant","requestId":"req_X","sessionId":"s","cwd":"/p","timestamp":"2026-06-07T10:00:00.000Z","message":{"id":"m","model":"claude-opus-4-7","usage":{"input_tokens":1000000,"output_tokens":1000000}}}` + "\n"
	if err := os.WriteFile(filepath.Join(projDir, "a.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))

	dbPath := filepath.Join(t.TempDir(), "lens.db")
	out, err := runScan(dbPath)
	if err != nil {
		t.Fatalf("runScan: %v", err)
	}
	if !contains(out, "claude-opus-4-7") || !contains(out, "90.0") {
		t.Fatalf("в выводе нет ожидаемой модели/стоимости 90.0:\n%s", out)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestRescanAfterRewriteNoDuplicateCounts(t *testing.T) {
	home := t.TempDir()
	projDir := filepath.Join(home, ".claude", "projects", "-proj")
	os.MkdirAll(projDir, 0o755)
	logPath := filepath.Join(projDir, "a.jsonl")
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	dbPath := filepath.Join(t.TempDir(), "lens.db")

	ev := func(id string) string {
		return `{"type":"assistant","requestId":"` + id + `","sessionId":"s","cwd":"/p","timestamp":"2026-06-07T10:00:00.000Z","message":{"id":"m","model":"claude-opus-4-7","usage":{"input_tokens":1000000,"output_tokens":0}}}` + "\n"
	}

	os.WriteFile(logPath, []byte(ev("req_1")+ev("req_2")), 0o644)
	if _, err := runScan(dbPath); err != nil {
		t.Fatal(err)
	}
	// «Компактизация»: файл переписан, req_1 исчез, добавился req_3.
	os.WriteFile(logPath, []byte(ev("req_2")+ev("req_3")), 0o644)
	out, err := runScan(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// req_1 (15 USD) + req_2 + req_3 = 3 события, по 15 USD = 45.0; req_2 не задвоен.
	if !contains(out, "45.0") {
		t.Fatalf("ожидалось 3 уникальных события (45.0 USD), идемпотентность нарушена:\n%s", out)
	}
}
