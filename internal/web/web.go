// Package web serves the local board UI and JSON API over a store. Every
// mutation goes through the same store the CLI uses, so the board and the CLI
// never diverge (spec WEB-04; AD-001). A single mutex serializes access because
// the store's in-memory index is not safe for concurrent use.
package web

import (
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/vitor/kanren/internal/store"
)

// Server adapts a store to HTTP. Hold mu around every store call.
type Server struct {
	mu    sync.Mutex
	store *store.Store
}

// Handler builds the HTTP routes for a board. Exposed for tests via httptest.
func Handler(s *store.Store) http.Handler {
	srv := &Server{store: s}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", srv.board)
	mux.HandleFunc("GET /api/cards", srv.listCards)
	mux.HandleFunc("POST /api/cards/{id}/move", srv.moveCard)
	return mux
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

// board renders a minimal server-side board. T9 replaces this with embedded
// assets, SortableJS drag-drop, and an impeccable visual pass.
func (s *Server) board(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	cols := s.store.Columns()
	byCol := map[string][]string{}
	for _, col := range cols {
		for _, c := range s.store.List(store.Filter{Status: col}) {
			byCol[col] = append(byCol[col], c.Title)
		}
	}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, "<!doctype html><meta charset=utf-8><title>kanren</title><h1>kanren</h1>")
	for _, col := range cols {
		fmt.Fprintf(w, "<section><h2>%s</h2><ul>", html.EscapeString(col))
		for _, title := range byCol[col] {
			fmt.Fprintf(w, "<li>%s</li>", html.EscapeString(title))
		}
		fmt.Fprint(w, "</ul></section>")
	}
}
