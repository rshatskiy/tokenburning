package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPushDryRunRequiresCategory(t *testing.T) {
	root := NewRootCmd()
	root.SetArgs([]string{"push", "--dry-run"})
	if err := root.Execute(); err == nil {
		t.Fatal("ожидалась ошибка: без категории push должен падать")
	}
}

func TestPushToTestServer(t *testing.T) {
	// фейковая БД в HOME
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".tokenburning"), 0o755); err != nil {
		t.Fatal(err)
	}
	// создаём пустую БД через scan-путь невозможно тут; пушим пустые агрегаты — это ок
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		w.WriteHeader(200)
	}))
	defer srv.Close()
	root := NewRootCmd()
	root.SetArgs([]string{"push", "--breadth", "--to", srv.URL})
	if err := root.Execute(); err != nil {
		t.Fatalf("push: %v", err)
	}
	if !strings.Contains(got, "schemaVersion") {
		t.Fatalf("сервер не получил payload: %q", got)
	}
}
