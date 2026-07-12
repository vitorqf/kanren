package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/vitor/kanren/internal/card"
	"github.com/vitor/kanren/internal/store"
)

// newBoard inits a board with two cards and returns an httptest server + dir.
func newBoard(t *testing.T) (*httptest.Server, string, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	if err := store.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, err := store.Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := s.Add("first"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := s.Add("second"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	ts := httptest.NewServer(Handler(s))
	t.Cleanup(ts.Close)
	return ts, dir, s
}

// TestBoardRendersColumns: GET / lists the board's columns and cards (WEB-01).
func TestBoardRendersColumns(t *testing.T) {
	t.Parallel()
	ts, _, _ := newBoard(t)
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body := readAll(t, resp)
	for _, want := range []string{"todo", "doing", "done", "first", "second"} {
		if !strings.Contains(body, want) {
			t.Errorf("board missing %q:\n%s", want, body)
		}
	}
}

// TestListCardsJSON: GET /api/cards returns the cards as JSON (WEB-01 data).
func TestListCardsJSON(t *testing.T) {
	t.Parallel()
	ts, _, _ := newBoard(t)
	resp, err := http.Get(ts.URL + "/api/cards")
	if err != nil {
		t.Fatalf("GET /api/cards: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var cards []card.Card
	if err := json.NewDecoder(resp.Body).Decode(&cards); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(cards) != 2 {
		t.Errorf("got %d cards, want 2", len(cards))
	}
}

// TestMoveCardPersistsLikeCLI: POST move updates status AND writes the same file
// bytes a CLI mv would, leaving the body intact (WEB-02, WEB-04).
func TestMoveCardPersistsLikeCLI(t *testing.T) {
	t.Parallel()
	ts, dir, _ := newBoard(t)

	cardPath := filepath.Join(dir, "cards", "0001-first.md")
	bodyBefore := parseBody(t, cardPath)

	resp := postMove(t, ts, 1, "doing")
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("move status = %d, want 200", resp.StatusCode)
	}
	var updated card.Card
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updated.Status != "doing" {
		t.Errorf("returned status = %q, want doing", updated.Status)
	}

	// File on disk reflects the move via the shared store, body unchanged.
	onDisk := parseCard(t, cardPath)
	if onDisk.Status != "doing" {
		t.Errorf("file status = %q, want doing (WEB-04)", onDisk.Status)
	}
	if onDisk.Body != bodyBefore {
		t.Errorf("move altered body: got %q want %q", onDisk.Body, bodyBefore)
	}
}

// TestMoveInvalidStatus: an invalid target column is rejected (WEB-02 error
// path).
func TestMoveInvalidStatus(t *testing.T) {
	t.Parallel()
	ts, _, _ := newBoard(t)
	resp := postMove(t, ts, 1, "bogus")
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 400 {
		t.Errorf("invalid status = %d, want 4xx", resp.StatusCode)
	}
}

// TestMoveBadID: a non-numeric id path segment is a 400 (WEB-02 error path).
func TestMoveBadID(t *testing.T) {
	t.Parallel()
	ts, _, _ := newBoard(t)
	resp, err := http.Post(ts.URL+"/api/cards/abc/move", "application/json",
		strings.NewReader(`{"status":"doing"}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad id = %d, want 400", resp.StatusCode)
	}
}

// --- helpers ---

func postMove(t *testing.T, ts *httptest.Server, id int, status string) *http.Response {
	t.Helper()
	url := ts.URL + "/api/cards/" + strconv.Itoa(id) + "/move"
	resp, err := http.Post(url, "application/json",
		strings.NewReader(`{"status":"`+status+`"}`))
	if err != nil {
		t.Fatalf("post move: %v", err)
	}
	return resp
}

func readAll(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func parseCard(t *testing.T, path string) card.Card {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	c, err := card.Parse(data)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return c
}

func parseBody(t *testing.T, path string) string { return parseCard(t, path).Body }
