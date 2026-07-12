package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitor/kanren/internal/card"
)

// openBoard is a test helper: init + open a fresh board.
func openBoard(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

// TestAddAssignsIDAndLeftmostColumn: Add allocates a unique incrementing id and
// places the card in the leftmost column (CLI-01).
func TestAddAssignsIDAndLeftmostColumn(t *testing.T) {
	t.Parallel()
	s := openBoard(t)

	a, err := s.Add("first")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	b, err := s.Add("second")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if a.ID != 1 || b.ID != 2 {
		t.Errorf("ids = %d,%d, want 1,2", a.ID, b.ID)
	}
	if a.Status != "todo" {
		t.Errorf("status = %q, want leftmost column todo", a.Status)
	}
	if a.Created == "" {
		t.Error("Created not set")
	}
}

// TestAddPersistsRoundTrippable: the file Add writes parses back identically
// (CLI-01 + CARD-02 integration).
func TestAddPersistsRoundTrippable(t *testing.T) {
	t.Parallel()
	s := openBoard(t)
	c, err := s.Add("Fix the bug")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	data, err := os.ReadFile(s.Path(c))
	if err != nil {
		t.Fatalf("read written card: %v", err)
	}
	got, err := card.Parse(data)
	if err != nil {
		t.Fatalf("parse written card: %v", err)
	}
	if got.ID != c.ID || got.Title != c.Title || got.Status != c.Status {
		t.Errorf("round-trip mismatch: got %+v want %+v", got, c)
	}
	if filepath.Base(s.Path(c)) != "0001-fix-the-bug.md" {
		t.Errorf("filename = %q, want 0001-fix-the-bug.md", filepath.Base(s.Path(c)))
	}
}

// TestGetMissing: Get on an unknown id errors (CLI-04 support).
func TestGetMissing(t *testing.T) {
	t.Parallel()
	s := openBoard(t)
	if _, err := s.Get(99); err == nil {
		t.Fatal("expected error for missing id")
	}
}

// TestGetDuplicateIDRefuses: two files with the same id make id-based ops refuse
// ambiguously, naming both files (duplicate-id edge case).
func TestGetDuplicateIDRefuses(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	writeCard(t, dir, "0001-a.md", "---\nid: 1\ntitle: a\nstatus: todo\n---\n")
	writeCard(t, dir, "0001-b.md", "---\nid: 1\ntitle: b\nstatus: todo\n---\n")

	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	_, err = s.Get(1)
	if err == nil {
		t.Fatal("expected ambiguous-id error")
	}
	for _, name := range []string{"0001-a.md", "0001-b.md"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error should name %s: %v", name, err)
		}
	}
}
