package cli

import (
	"os"
	"path/filepath"
	"strings"
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
	if !strings.Contains(out, "claude-opus-4-7") || !strings.Contains(out, "90.0") {
		t.Fatalf("в выводе нет ожидаемой модели/стоимости 90.0:\n%s", out)
	}
}

func TestRunScanPrintsPerToolSection(t *testing.T) {
	home := t.TempDir()
	projDir := filepath.Join(home, ".claude", "projects", "-proj")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"assistant","requestId":"req_T","sessionId":"s","cwd":"/p","timestamp":"2026-06-07T10:00:00.000Z","message":{"id":"m","model":"claude-opus-4-7","usage":{"input_tokens":1000000,"output_tokens":0}}}` + "\n"
	if err := os.WriteFile(filepath.Join(projDir, "a.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	t.Setenv("CODEX_HOME", filepath.Join(home, "no-codex"))
	dbPath := filepath.Join(t.TempDir(), "lens.db")
	out, err := runScan(dbPath)
	if err != nil {
		t.Fatalf("runScan: %v", err)
	}
	if !strings.Contains(out, "TOOL") || !strings.Contains(out, "claude_code") {
		t.Fatalf("нет секции TOOL/claude_code:\n%s", out)
	}
}

func TestRescanAfterRewriteNoDuplicateCounts(t *testing.T) {
	home := t.TempDir()
	projDir := filepath.Join(home, ".claude", "projects", "-proj")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(projDir, "a.jsonl")
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	dbPath := filepath.Join(t.TempDir(), "lens.db")

	ev := func(id string) string {
		return `{"type":"assistant","requestId":"` + id + `","sessionId":"s","cwd":"/p","timestamp":"2026-06-07T10:00:00.000Z","message":{"id":"m","model":"claude-opus-4-7","usage":{"input_tokens":1000000,"output_tokens":0}}}` + "\n"
	}

	if err := os.WriteFile(logPath, []byte(ev("req_1")+ev("req_2")), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runScan(dbPath); err != nil {
		t.Fatal(err)
	}
	// «Компактизация»: файл переписан, req_1 исчез, добавился req_3.
	if err := os.WriteFile(logPath, []byte(ev("req_2")+ev("req_3")), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runScan(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// req_1 (15 USD) + req_2 + req_3 = 3 события, по 15 USD = 45.0; req_2 не задвоен.
	if !strings.Contains(out, "45.0") {
		t.Fatalf("ожидалось 3 уникальных события (45.0 USD), идемпотентность нарушена:\n%s", out)
	}
	// Колонка EVENTS=3 для opus (формат %8d): ищем "3" перед стоимостью 45.0 в той же строке.
	if !strings.Contains(out, "claude-opus-4-7") {
		t.Fatalf("нет строки opus:\n%s", out)
	}
	for _, ln := range strings.Split(out, "\n") {
		if strings.Contains(ln, "claude-opus-4-7") {
			if !strings.Contains(ln, "3") || !strings.Contains(ln, "45.0") {
				t.Fatalf("строка opus должна иметь EVENTS=3 и COST=45.0: %q", ln)
			}
		}
	}
}
