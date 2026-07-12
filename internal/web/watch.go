package web

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// hub fans a single "cards changed" signal out to every connected SSE client.
type hub struct {
	mu      sync.Mutex
	clients map[chan struct{}]bool
}

func newHub() *hub { return &hub{clients: map[chan struct{}]bool{}} }

func (h *hub) add() chan struct{} {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.clients[ch] = true
	h.mu.Unlock()
	return ch
}

func (h *hub) remove(ch chan struct{}) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
}

// broadcast wakes every client. The buffered channels + default make it
// non-blocking: a client already holding a pending signal is simply left as is.
func (h *hub) broadcast() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// watch reloads the store and broadcasts whenever the cards directory changes.
// It uses fsnotify, falling back to a 2s poll if a watcher cannot be created
// (WEB-03 mitigation). It runs until the process exits.
func (s *Server) watch() {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("kanren: file watching unavailable, polling every 2s: %v", err)
		s.poll()
		return
	}
	if err := w.Add(s.cardsDir); err != nil {
		log.Printf("kanren: cannot watch %s, polling every 2s: %v", s.cardsDir, err)
		_ = w.Close()
		s.poll()
		return
	}
	for {
		select {
		case _, ok := <-w.Events:
			if !ok {
				return
			}
			s.reloadAndNotify()
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			log.Printf("kanren: watch error: %v", err)
		}
	}
}

// poll is the fallback when fsnotify is unavailable.
func (s *Server) poll() {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for range t.C {
		s.reloadAndNotify()
	}
}

// reloadAndNotify re-reads the board and signals SSE clients to refresh.
func (s *Server) reloadAndNotify() {
	s.mu.Lock()
	err := s.store.Reload()
	s.mu.Unlock()
	if err != nil {
		log.Printf("kanren: reload failed: %v", err)
		return
	}
	s.hub.broadcast()
}

// events is the SSE endpoint. Each connected board receives a "reload" event
// whenever a card file changes on disk (WEB-03).
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := s.hub.add()
	defer s.hub.remove(ch)

	// Prompt clients to establish the stream cleanly.
	fmt.Fprint(w, "retry: 2000\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			fmt.Fprint(w, "data: reload\n\n")
			flusher.Flush()
		}
	}
}
