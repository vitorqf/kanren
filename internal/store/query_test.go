package store

import (
	"bytes"
	"os"
	"testing"

	"github.com/vitor/kanren/internal/card"
)

// seed adds cards with tags/assignee for query tests.
func seed(t *testing.T, s *Store) {
	t.Helper()
	specs := []struct {
		title, status, assignee string
		tags                    []string
	}{
		{"a", "doing", "vitor", []string{"bug", "urgent"}},
		{"b", "doing", "ana", []string{"urgent"}},
		{"c", "todo", "vitor", []string{"chore"}},
	}
	for _, sp := range specs {
		c, err := s.Add(sp.title)
		if err != nil {
			t.Fatalf("Add: %v", err)
		}
		c.Status, c.Assignee, c.Tags = sp.status, sp.assignee, sp.tags
		if err := s.Save(c); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}
}

// TestListAndFilter: combined filters use AND semantics (QRY-01).
func TestListAndFilter(t *testing.T) {
	t.Parallel()
	s := openBoard(t)
	seed(t, s)

	got := s.List(Filter{Status: "doing", Tag: "urgent", Assignee: "vitor"})
	if len(got) != 1 || got[0].Title != "a" {
		t.Fatalf("AND filter = %v, want single card 'a'", titles(got))
	}
}

// TestListEmptyFilterReturnsAll: empty filter returns every card (QRY-01).
func TestListEmptyFilterReturnsAll(t *testing.T) {
	t.Parallel()
	s := openBoard(t)
	seed(t, s)
	if got := s.List(Filter{}); len(got) != 3 {
		t.Errorf("empty filter = %d cards, want 3", len(got))
	}
}

// TestListNoMatch: no match returns empty, not error (QRY-03, QRY-04).
func TestListNoMatch(t *testing.T) {
	t.Parallel()
	s := openBoard(t)
	seed(t, s)
	if got := s.List(Filter{Tag: "nonexistent"}); len(got) != 0 {
		t.Errorf("unknown tag = %v, want empty", titles(got))
	}
	if got := s.List(Filter{Status: "bogus"}); len(got) != 0 {
		t.Errorf("unknown status = %v, want empty", titles(got))
	}
}

// TestListSortedByColumnOrder: results order by column left-to-right (todo
// before doing), so 'c' (todo) precedes 'a'/'b' (doing).
func TestListSortedByColumnOrder(t *testing.T) {
	t.Parallel()
	s := openBoard(t)
	seed(t, s)
	got := s.List(Filter{})
	if got[0].Title != "c" {
		t.Errorf("first = %q, want 'c' (todo column is leftmost)", got[0].Title)
	}
}

// TestMoveChangesOnlyStatusAndOrder: Move edits status/order but leaves the
// card body byte-identical (CLI-02).
func TestMoveChangesOnlyStatusAndOrder(t *testing.T) {
	t.Parallel()
	s := openBoard(t)
	c, err := s.Add("keep body")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	c.Body = "# keep body\n\nSome **markdown** that must survive a move.\n"
	if err := s.Save(c); err != nil {
		t.Fatalf("Save: %v", err)
	}
	bodyBefore := extractBody(t, s.Path(c))

	if err := s.Move(c.ID, "done"); err != nil {
		t.Fatalf("Move: %v", err)
	}
	moved, _ := s.Get(c.ID)
	if moved.Status != "done" {
		t.Errorf("status = %q, want done", moved.Status)
	}
	if got := extractBody(t, s.Path(moved)); got != bodyBefore {
		t.Errorf("body changed by move:\n got  %q\n want %q", got, bodyBefore)
	}
}

// TestMoveInvalidStatusNoWrite: an invalid target status errors and writes
// nothing (CLI-04).
func TestMoveInvalidStatusNoWrite(t *testing.T) {
	t.Parallel()
	s := openBoard(t)
	c, err := s.Add("x")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	before, _ := os.ReadFile(s.Path(c))
	if err := s.Move(c.ID, "nope"); err == nil {
		t.Fatal("expected error for invalid status")
	}
	after, _ := os.ReadFile(s.Path(c))
	if !bytes.Equal(before, after) {
		t.Error("Move to invalid status modified the file")
	}
}

// TestMoveUnknownID: moving a missing id errors (CLI-04).
func TestMoveUnknownID(t *testing.T) {
	t.Parallel()
	s := openBoard(t)
	if err := s.Move(42, "done"); err == nil {
		t.Fatal("expected error moving unknown id")
	}
}

// TestMisfiledSurfaced: a card whose status is not a column is reported as
// misfiled, not silently reassigned (CARD-04).
func TestMisfiledSurfaced(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	writeCard(t, dir, "0001-x.md", "---\nid: 1\ntitle: x\nstatus: limbo\n---\n")
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	mis := s.Misfiled()
	if len(mis) != 1 || mis[0].Status != "limbo" {
		t.Fatalf("Misfiled = %v, want the 'limbo' card", mis)
	}
}

func titles(cs []card.Card) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Title
	}
	return out
}

func extractBody(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	c, err := card.Parse(data)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return c.Body
}
