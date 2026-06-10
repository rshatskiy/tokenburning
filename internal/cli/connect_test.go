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
	t.Setenv("USERPROFILE", home) // Windows: os.UserHomeDir читает USERPROFILE
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

func TestRequireHTTPS(t *testing.T) {
	for _, ok := range []string{"https://tokenburning.ru", "http://localhost:8080", "http://127.0.0.1:9999"} {
		if err := requireHTTPS(ok); err != nil {
			t.Errorf("requireHTTPS(%q) = %v, ожидалось nil", ok, err)
		}
	}
	for _, bad := range []string{"http://tokenburning.ru", "http://192.168.1.10:8080", "ftp://x"} {
		if err := requireHTTPS(bad); err == nil {
			t.Errorf("requireHTTPS(%q) = nil, ожидалась ошибка", bad)
		}
	}
}

func TestConnectRequiresToAndToken(t *testing.T) {
	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--breadth"})
	if err := root.Execute(); err == nil {
		t.Fatal("connect без --to/--token должен падать")
	}
}
