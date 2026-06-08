package cli

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/store"
)

func TestPushSendsBearerToken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".tokenburning"), 0o755); err != nil {
		t.Fatal(err)
	}
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer srv.Close()
	root := NewRootCmd()
	root.SetArgs([]string{"push", "--breadth", "--to", srv.URL, "--token", "secret-tok"})
	if err := root.Execute(); err != nil {
		t.Fatalf("push: %v", err)
	}
	if gotAuth != "Bearer secret-tok" {
		t.Fatalf("Authorization = %q, want 'Bearer secret-tok'", gotAuth)
	}
}

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

func TestPushDryRunPrintsSafePayload(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dbDir := filepath.Join(home, ".tokenburning")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(filepath.Join(dbDir, "tokenburning.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Insert([]model.Event{{
		EventID: "x", Tool: model.ToolClaudeCode, TS: time.Now().UTC(), Model: "claude-opus-4-8",
		BillingMode: model.BillingFlatEquivalent,
		Cost:        model.Cost{Amount: 3, Currency: "USD", Basis: model.BasisActual},
		Tokens:      model.Tokens{Input: 50}, SessionID: "s", ProjectKey: "/Users/me/secret-proj",
	}}); err != nil {
		t.Fatal(err)
	}
	db.Close()

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"push", "--breadth", "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("push --dry-run: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "schemaVersion") {
		t.Fatalf("нет payload в выводе dry-run:\n%s", s)
	}
	if strings.Contains(s, "secret-proj") || strings.Contains(s, "/Users/") {
		t.Fatalf("dry-run слил проект:\n%s", s)
	}
}
