package view

import (
	"encoding/json"
	"io/fs"
	"net"
	"net/http"

	"github.com/rshatskiy/tokenburning/internal/store"
)

type Server struct {
	db    *store.DB
	token string
	mux   *http.ServeMux
}

func NewServer(db *store.DB, token string) *Server {
	s := &Server{db: db, token: token, mux: http.NewServeMux()}
	sub, _ := fs.Sub(assetsFS, "assets")
	s.mux.Handle("/", http.FileServer(http.FS(sub)))
	s.mux.HandleFunc("/api/summary", s.handleSummary)
	return s
}

// Handler оборачивает mux проверкой Host (анти-DNS-rebinding).
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hostAllowed(r.Host) {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		s.mux.ServeHTTP(w, r)
	})
}

// hostAllowed разрешает только loopback-хосты (с любым портом).
func hostAllowed(host string) bool {
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host // нет порта — берём строку целиком как хост
	}
	return h == "127.0.0.1" || h == "localhost" || h == "::1"
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("t") != s.token {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d"
	}
	sum, err := BuildSummary(s.db, period)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sum)
}
