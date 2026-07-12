package store

import (
	"fmt"
	"sort"

	"github.com/vitorqf/kanren/internal/card"
)

// Filter selects cards by field. A zero (empty) field matches any value; when
// several fields are set they combine with AND semantics (QRY-01).
type Filter struct {
	Status   string
	Tag      string
	Assignee string
}

// matches reports whether c satisfies every set field of f.
func (f Filter) matches(c card.Card) bool {
	if f.Status != "" && c.Status != f.Status {
		return false
	}
	if f.Assignee != "" && c.Assignee != f.Assignee {
		return false
	}
	if f.Tag != "" {
		found := false
		for _, tag := range c.Tags {
			if tag == f.Tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// validColumn reports whether status is one of the board's columns.
func (s *Store) validColumn(status string) bool {
	for _, col := range s.cfg.Columns {
		if col == status {
			return true
		}
	}
	return false
}

// columnIndex returns the left-to-right position of status, or len(columns) so
// misfiled cards sort after all real columns.
func (s *Store) columnIndex(status string) int {
	for i, col := range s.cfg.Columns {
		if col == status {
			return i
		}
	}
	return len(s.cfg.Columns)
}

// List returns cards matching f, sorted by column order, then Order, then id.
// An empty filter returns every card. No match returns an empty slice, never an
// error (QRY-01, QRY-03, QRY-04).
func (s *Store) List(f Filter) []card.Card {
	var out []card.Card
	for _, c := range s.cards {
		if f.matches(c) {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		ci, cj := s.columnIndex(out[i].Status), s.columnIndex(out[j].Status)
		if ci != cj {
			return ci < cj
		}
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Misfiled returns cards whose status is not one of the board's columns. Such
// cards are surfaced, not silently reassigned (CARD-04).
func (s *Store) Misfiled() []card.Card {
	var out []card.Card
	for _, c := range s.cards {
		if !s.validColumn(c.Status) {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Move changes a card's status (and appends it to the end of the target
// column) without touching its body. It errors, writing nothing, when the id is
// unknown/ambiguous or the target status is not a board column (CLI-02, CLI-04).
func (s *Store) Move(id int, status string) error {
	c, err := s.Get(id)
	if err != nil {
		return err
	}
	if !s.validColumn(status) {
		return fmt.Errorf("store: %q is not a column (have %v)", status, s.cfg.Columns)
	}
	c.Status = status
	c.Order = s.nextOrder(status)
	return s.Save(c)
}

// nextOrder returns one past the highest Order among cards in the given column.
func (s *Store) nextOrder(status string) float64 {
	highest := 0.0
	for _, c := range s.cards {
		if c.Status == status && c.Order > highest {
			highest = c.Order
		}
	}
	return highest + 1
}
