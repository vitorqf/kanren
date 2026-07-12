// Package store is the single owner of all filesystem I/O for a board
// (see .specs/STATE.md AD-001). The CLI and web server are adapters over it;
// no other package reads or writes card files. See spec.md (CARD-*/CLI-*/QRY-*).
package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/vitorqf/kanren/internal/card"
)

// ConfigName is the board config file at the board root.
const ConfigName = ".kanren.yml"

// Config is the board configuration read from ConfigName.
type Config struct {
	Columns  []string `yaml:"columns"`
	CardsDir string   `yaml:"cards_dir"`
}

// defaultConfig is written by Init.
func defaultConfig() Config {
	return Config{Columns: []string{"todo", "doing", "done"}, CardsDir: "cards"}
}

// Store is an opened board: its config plus an in-memory index of cards.
type Store struct {
	dir      string
	cfg      Config
	cards    map[int]card.Card // id -> card
	files    map[int]string    // id -> filename (basename) it was loaded from
	dups     map[int][]string  // id -> filenames, when >1 file claims the same id
	warnings []string          // malformed/skipped files, reported not fatal
}

// Exists reports whether dir already contains a kanren board.
func Exists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ConfigName))
	return err == nil
}

// Init scaffolds a new board in dir: writes ConfigName with default columns and
// creates the cards directory. It refuses if a board config already exists,
// changing nothing (INIT-01, INIT-02).
func Init(dir string) error {
	cfgPath := filepath.Join(dir, ConfigName)
	if _, err := os.Stat(cfgPath); err == nil {
		return fmt.Errorf("store: board already initialized (%s exists)", ConfigName)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("store: checking for existing board: %w", err)
	}

	cfg := defaultConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("store: marshal default config: %w", err)
	}
	if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
		return fmt.Errorf("store: write config: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, cfg.CardsDir), 0o755); err != nil {
		return fmt.Errorf("store: create cards dir: %w", err)
	}
	return nil
}

// Open loads the board at dir: reads config (erroring with a hint to run init
// when absent) and indexes every card file. Malformed files are skipped and
// recorded as warnings rather than aborting the load (CARD-03); non-.md files
// are ignored.
func Open(dir string) (*Store, error) {
	cfgPath := filepath.Join(dir, ConfigName)
	data, err := os.ReadFile(cfgPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("store: no board here (missing %s) — run `kanren init`", ConfigName)
	} else if err != nil {
		return nil, fmt.Errorf("store: read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("store: invalid %s: %w", ConfigName, err)
	}

	s := &Store{
		dir:   dir,
		cfg:   cfg,
		cards: map[int]card.Card{},
		files: map[int]string{},
		dups:  map[int][]string{},
	}
	if err := s.index(); err != nil {
		return nil, err
	}
	return s, nil
}

// index reads every .md file in the cards dir into the in-memory map.
func (s *Store) index() error {
	cardsDir := filepath.Join(s.dir, s.cfg.CardsDir)
	entries, err := os.ReadDir(cardsDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil // empty board: no cards dir yet is not fatal
	} else if err != nil {
		return fmt.Errorf("store: read cards dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(cardsDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			s.warnings = append(s.warnings, fmt.Sprintf("%s: %v", e.Name(), err))
			continue
		}
		c, err := card.Parse(data)
		if err != nil {
			s.warnings = append(s.warnings, fmt.Sprintf("%s: %v", e.Name(), err))
			continue
		}
		if prev, seen := s.files[c.ID]; seen {
			// Second (or later) file claiming an existing id: record every
			// filename involved so id-based ops can refuse ambiguously.
			if len(s.dups[c.ID]) == 0 {
				s.dups[c.ID] = []string{prev}
			}
			s.dups[c.ID] = append(s.dups[c.ID], e.Name())
			s.warnings = append(s.warnings,
				fmt.Sprintf("duplicate id %d in %v", c.ID, s.dups[c.ID]))
			continue
		}
		s.cards[c.ID] = c
		s.files[c.ID] = e.Name()
	}
	return nil
}

// Columns returns the board's columns in left-to-right order.
func (s *Store) Columns() []string { return s.cfg.Columns }

// CardsDirPath returns the absolute path of the directory holding card files.
func (s *Store) CardsDirPath() string {
	return filepath.Join(s.dir, s.cfg.CardsDir)
}

// Reload re-reads every card file from disk, replacing the in-memory index. Use
// it after external changes (a CLI edit, a git pull) so the board reflects them.
func (s *Store) Reload() error {
	s.cards = map[int]card.Card{}
	s.files = map[int]string{}
	s.dups = map[int][]string{}
	s.warnings = nil
	return s.index()
}

// Warnings returns non-fatal problems found while indexing (e.g. malformed
// card files that were skipped).
func (s *Store) Warnings() []string { return s.warnings }

// Count returns the number of indexed cards.
func (s *Store) Count() int { return len(s.cards) }

// nextID returns one past the highest indexed id (ids start at 1).
func (s *Store) nextID() int {
	highest := 0
	for id := range s.cards {
		if id > highest {
			highest = id
		}
	}
	return highest + 1
}

// Add creates a new card titled title in the leftmost column, assigns it the
// next free id and today's date, writes it to disk, and returns it (CLI-01).
func (s *Store) Add(title string) (card.Card, error) {
	if len(s.cfg.Columns) == 0 {
		return card.Card{}, fmt.Errorf("store: board has no columns")
	}
	c := card.Card{
		ID:      s.nextID(),
		Title:   title,
		Status:  s.cfg.Columns[0],
		Created: time.Now().Format("2006-01-02"),
	}
	if err := s.Save(c); err != nil {
		return card.Card{}, err
	}
	return c, nil
}

// Get returns the card with the given id. It refuses when the id is claimed by
// more than one file on disk, naming the offending files (duplicate-id edge).
func (s *Store) Get(id int) (card.Card, error) {
	if files := s.dups[id]; len(files) > 0 {
		return card.Card{}, fmt.Errorf("store: id %d is ambiguous, claimed by %v", id, files)
	}
	c, ok := s.cards[id]
	if !ok {
		return card.Card{}, fmt.Errorf("store: no card with id %d", id)
	}
	return c, nil
}

// Path returns the absolute file path a card serializes to.
func (s *Store) Path(c card.Card) string {
	return filepath.Join(s.dir, s.cfg.CardsDir, card.Filename(c.ID, c.Title))
}

// Save writes c to disk and updates the in-memory index. The filename derives
// from the card's id and title, so the written file round-trips through
// card.Parse identically. When a title change renames the file, the old file is
// removed so no orphan is left behind.
func (s *Store) Save(c card.Card) error {
	data, err := c.Marshal()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(s.dir, s.cfg.CardsDir), 0o755); err != nil {
		return fmt.Errorf("store: create cards dir: %w", err)
	}
	newName := card.Filename(c.ID, c.Title)
	if err := os.WriteFile(s.Path(c), data, 0o644); err != nil {
		return fmt.Errorf("store: write card %d: %w", c.ID, err)
	}
	if old, ok := s.files[c.ID]; ok && old != newName {
		_ = os.Remove(filepath.Join(s.dir, s.cfg.CardsDir, old))
	}
	s.cards[c.ID] = c
	s.files[c.ID] = newName
	return nil
}

// Update applies edited fields to a card and persists it, keeping id, status,
// and order intact. It is the single entry point for editing card content
// (title, body, tags, assignee) from any adapter.
func (s *Store) Update(id int, title, body string, tags []string, assignee string) (card.Card, error) {
	c, err := s.Get(id)
	if err != nil {
		return card.Card{}, err
	}
	if strings.TrimSpace(title) != "" {
		c.Title = title
	}
	c.Body = body
	c.Tags = tags
	c.Assignee = assignee
	if err := s.Save(c); err != nil {
		return card.Card{}, err
	}
	return c, nil
}
