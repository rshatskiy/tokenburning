package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rshatskiy/tokenburning/internal/config"
)

func TestConnectSavesConfigAndPushes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".tokenburning"), 0o755); err != nil {
		t.Fatal(err)
	}
	var gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--to", srv.URL, "--token", "tok123", "--breadth", "--no-autostart"})
	if err := root.Execute(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if gotAuth != "Bearer tok123" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if !strings.Contains(gotBody, "schemaVersion") {
		t.Fatalf("сервер не получил payload: %q", gotBody)
	}
	// конфиг сохранён
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Push.Enabled || cfg.Push.Endpoint != srv.URL || cfg.Push.Token != "tok123" {
		t.Fatalf("конфиг не сохранён: %+v", cfg.Push)
	}
}

func TestConnectRequiresToAndToken(t *testing.T) {
	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--breadth"})
	if err := root.Execute(); err == nil {
		t.Fatal("connect без --to/--token должен падать")
	}
}
