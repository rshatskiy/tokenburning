package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestIsOlder(t *testing.T) {
	cases := []struct {
		cur, latest string
		want        bool
	}{
		{"v0.1.0", "v0.2.0", true},
		{"0.1.0", "0.1.1", true},
		{"v0.2.0", "v0.2.0", false},
		{"v0.3.0", "v0.2.0", false},
		{"v1.0.0", "v0.9.9", false},
		{"dev", "v0.2.0", true},      // непарсимая текущая → предлагаем обновление
		{"v0.2.0", "garbage", false}, // непарсимый latest → не предлагаем
	}
	for _, c := range cases {
		if got := IsOlder(c.cur, c.latest); got != c.want {
			t.Errorf("IsOlder(%q,%q)=%v want %v", c.cur, c.latest, got, c.want)
		}
	}
}

func TestChecksumFor(t *testing.T) {
	sums := "abc123  tokenburning_0.2.0_linux_amd64.tar.gz\ndef456  tokenburning_0.2.0_darwin_arm64.tar.gz\n"
	if got := checksumFor(sums, "tokenburning_0.2.0_darwin_arm64.tar.gz"); got != "def456" {
		t.Errorf("checksumFor = %q", got)
	}
	if got := checksumFor(sums, "missing.tar.gz"); got != "" {
		t.Errorf("checksumFor(missing) = %q, want empty", got)
	}
}

func TestExtractBinaryTarGz(t *testing.T) {
	want := []byte("BINARYBYTES")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	// добавим лишний файл + сам бинарь
	for _, f := range []struct {
		name string
		data []byte
	}{{"README.md", []byte("x")}, {"tokenburning", want}} {
		_ = tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0o755, Size: int64(len(f.data))})
		_, _ = tw.Write(f.data)
	}
	tw.Close()
	gz.Close()

	got, err := extractBinary(buf.Bytes(), false)
	if err != nil {
		t.Fatalf("extractBinary: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("extracted = %q, want %q", got, want)
	}
}

func TestReplaceFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "tokenburning")
	if err := os.WriteFile(target, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(target, []byte("NEWVERSION")); err != nil {
		t.Fatalf("replaceFile: %v", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "NEWVERSION" {
		t.Fatalf("content = %q", got)
	}
	info, _ := os.Stat(target)
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("binary not executable: %v", info.Mode())
	}
	// в каталоге не должно остаться временных файлов
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("leftover temp files: %v", entries)
	}
}
