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
	id2, _ := Stat(p)
	if id1 != id2 {
		t.Fatalf("FileID не стабилен: %v != %v", id1, id2)
	}
	if id1.IsZero() {
		t.Fatal("FileID пустой для существующего файла")
	}
}
