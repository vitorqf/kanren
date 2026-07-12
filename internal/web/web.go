// Package web serves the local board UI and JSON API over a store. Every
// mutation goes through the same store the CLI uses, so the board and the CLI
// never diverge (spec WEB-04; AD-001). A single mutex serializes access because
// the store's in-memory index is not safe for concurrent use.
package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/vitorqf/kanren/internal/store"
)

// assetsFS holds the board UI (HTML, CSS, JS, vendored SortableJS), compiled
// into the binary so kanren ships as a single file with no external requests.
//
//go:embed assets
var assetsFS embed.FS

// Server adapts a store to HTTP. Hold mu around every store call.
type Server struct {
	mu       sync.Mutex
	store    *store.Store
	hub      *hub
	cardsDir string
}

// Handler builds the HTTP routes for a board and starts watching the cards
// directory for live reload. Exposed for tests via httptest.
func Handler(s *store.Store) http.Handler {
	srv := &Server{store: s, hub: newHub(), cardsDir: s.CardsDirPath()}
	go srv.watch()

	mux := http.NewServeMux()

	// Serve embedded UI assets under /static/ (files live in assets/).
	static, _ := fs.Sub(assetsFS, "assets")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))

	mux.HandleFunc("GET /{$}", srv.index)
	mux.HandleFunc("GET /api/columns", srv.listColumns)
	mux.HandleFunc("GET /api/cards", srv.listCards)
	mux.HandleFunc("POST /api/cards", srv.createCard)
	mux.HandleFunc("POST /api/cards/{id}/move", srv.moveCard)
	mux.HandleFunc("GET /events", srv.events)
	return mux
}

// createCard adds a card from the web UI so the board is usable without the
// CLI. Body is JSON {"title":"...","status":"..."}; status is optional and
// defaults to the leftmost column. Reuses store.Add/Move (WEB-04: identical
// files to the CLI).
func (s *Server) createCard(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Title) == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	c, err := s.store.Add(body.Title)
	if err == nil && body.Status != "" {
		if mErr := s.store.Move(c.ID, body.Status); mErr == nil {
			c, _ = s.store.Get(c.ID)
		}
	}
	s.mu.Unlock()

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(c)
}

// index serves the single-page board shell; data arrives via the JSON API.
func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFileFS(w, r, assetsFS, "assets/index.html")
}

// listColumns returns the board's columns in order (drives UI layout).
func (s *Server) listColumns(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	cols := s.store.Columns()
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cols)
}

// Serve starts the board server on port, returning a clear error if the port is
// already in use (edge case).
func Serve(s *store.Store, port int) error {
	addr := fmt.Sprintf("localhost:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("web: cannot listen on %s: %w", addr, err)
	}
	fmt.Printf("kanren board on http://%s\n", addr)
	return http.Serve(ln, Handler(s))
}

// listCards returns all cards as JSON (WEB-01 data source, reused by the UI).
func (s *Server) listCards(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	cards := s.store.List(store.Filter{})
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	// Never emit null for an empty board.
	if cards == nil {
		_, _ = w.Write([]byte("[]"))
		return
	}
	_ = json.NewEncoder(w).Encode(cards)
}

// moveCard applies a status change requested as JSON {"status":"..."} and
// returns the updated card. It reuses store.Move, so the persisted file is
// identical to what a CLI `kanren mv` would produce (WEB-02, WEB-04).
func (s *Server) moveCard(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	err = s.store.Move(id, body.Status)
	var card any
	if err == nil {
		card, _ = s.store.Get(id)
	}
	s.mu.Unlock()

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(card)
}

