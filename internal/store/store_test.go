package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeCard is a test helper: drops a raw card file into a board's cards dir.
func writeCard(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, "cards", name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// TestInitCreatesBoard: Init writes config + cards dir with default columns
// (INIT-01).
func TestInitCreatesBoard(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ConfigName)); err != nil {
		t.Errorf("config not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "cards")); err != nil {
		t.Errorf("cards dir not created: %v", err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open after Init: %v", err)
	}
	want := []string{"todo", "doing", "done"}
	if got := s.Columns(); len(got) != 3 || got[0] != want[0] || got[2] != want[2] {
		t.Errorf("columns = %v, want %v", got, want)
	}
}

// TestInitRefusesExisting: Init on a board that already exists changes nothing
// (INIT-02).
func TestInitRefusesExisting(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatalf("first Init: %v", err)
	}
	before, _ := os.ReadFile(filepath.Join(dir, ConfigName))
	if err := Init(dir); err == nil {
		t.Fatal("expected Init to refuse existing board, got nil")
	}
	after, _ := os.ReadFile(filepath.Join(dir, ConfigName))
	if string(before) != string(after) {
		t.Error("Init overwrote existing config")
	}
}

// TestOpenMissingConfig: Open without a board errors and names init (spec:
// ".kanren.yml missing -> run kanren init").
func TestOpenMissingConfig(t *testing.T) {
	t.Parallel()
	_, err := Open(t.TempDir())
	if err == nil {
		t.Fatal("expected error opening dir with no board")
	}
	if !strings.Contains(err.Error(), "kanren init") {
		t.Errorf("error should mention `kanren init`: %v", err)
	}
}

// TestIndexSkipsMalformed: a malformed card is skipped with a warning while
// valid cards still load (CARD-03).
func TestIndexSkipsMalformed(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	writeCard(t, dir, "0001-good.md", "---\nid: 1\ntitle: good\nstatus: todo\n---\nbody\n")
	writeCard(t, dir, "0002-bad.md", "no frontmatter here\n")

	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if s.Count() != 1 {
		t.Errorf("Count = %d, want 1 (bad card skipped)", s.Count())
	}
	if len(s.Warnings()) != 1 {
		t.Errorf("Warnings = %v, want 1", s.Warnings())
	}
}

// TestIndexIgnoresNonMarkdown: non-.md files and empty boards are fine (edges).
func TestIndexIgnoresNonMarkdown(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	writeCard(t, dir, "notes.txt", "ignore me")
	writeCard(t, dir, "0001-c.md", "---\nid: 1\ntitle: c\nstatus: todo\n---\n")

	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if s.Count() != 1 {
		t.Errorf("Count = %d, want 1 (txt ignored)", s.Count())
	}
	if len(s.Warnings()) != 0 {
		t.Errorf("non-.md should not warn: %v", s.Warnings())
	}
}

// TestEmptyBoard: Open on a freshly init'd board with no cards is not an error.
func TestEmptyBoard(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open empty board: %v", err)
	}
	if s.Count() != 0 {
		t.Errorf("Count = %d, want 0", s.Count())
	}
}
