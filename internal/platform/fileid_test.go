package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStatSameFileSameID(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.log")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	id1, err := Stat(p)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	id2, err := Stat(p)
	if err != nil {
		t.Fatalf("Stat (second): %v", err)
	}
	if id1 != id2 {
		t.Fatalf("FileID не стабилен: %v != %v", id1, id2)
	}
	if id1.IsZero() {
		t.Fatal("FileID пустой для существующего файла")
	}

	p2 := filepath.Join(dir, "b.log")
	if err := os.WriteFile(p2, []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
	id3, err := Stat(p2)
	if err != nil {
		t.Fatalf("Stat p2: %v", err)
	}
	if id1 == id3 {
		t.Fatal("FileID одинаков для разных файлов")
	}
}

func TestDetectHonorsClaudeConfigDir(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "/custom")
	paths, err := Detect()
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if paths.ClaudeCodeProjects != filepath.Join("/custom", "projects") {
		t.Fatalf("ClaudeCodeProjects = %q, want /custom/projects", paths.ClaudeCodeProjects)
	}
}

func TestDetectCodexHomeOverride(t *testing.T) {
	t.Setenv("CODEX_HOME", "/custom-codex")
	p, err := Detect()
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if p.CodexSessions != filepath.Join("/custom-codex", "sessions") {
		t.Fatalf("CodexSessions = %q, want /custom-codex/sessions", p.CodexSessions)
	}
}

func TestDetectCursorStorageNonEmpty(t *testing.T) {
	p, err := Detect()
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if p.CursorStorage == "" {
		t.Fatal("CursorStorage пуст — не определён путь Cursor для этой ОС")
	}
}
