package view

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/rshatskiy/tokenburning/internal/store"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return NewServer(db, "secret-token")
}

func TestAPIRequiresToken(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("GET", "http://127.0.0.1/api/summary?period=30d", nil)
	req.Host = "127.0.0.1"
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("без токена код %d, want 401", rr.Code)
	}
	// с токеном — 200
	req2 := httptest.NewRequest("GET", "http://127.0.0.1/api/summary?period=30d&t=secret-token", nil)
	req2.Host = "127.0.0.1"
	rr2 := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("с токеном код %d, want 200", rr2.Code)
	}
}

func TestRejectsForeignHost(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("GET", "http://evil.example/api/summary?t=secret-token", nil)
	req.Host = "evil.example" // не в allowlist
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("чужой Host код %d, want 403 (анти-rebinding)", rr.Code)
	}
}

func TestAssetsEmbedded(t *testing.T) {
	for _, name := range []string{"assets/index.html", "assets/app.js", "assets/style.css", "assets/favicon.svg"} {
		b, err := assetsFS.ReadFile(name)
		if err != nil || len(b) == 0 {
			t.Fatalf("ассет %s пуст или отсутствует: %v", name, err)
		}
	}
}
