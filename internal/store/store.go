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

	"github.com/goccy/go-yaml"
	"github.com/vitor/kanren/internal/card"
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
	warnings []string          // malformed/skipped files, reported not fatal
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

	s := &Store{dir: dir, cfg: cfg, cards: map[int]card.Card{}}
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
		s.cards[c.ID] = c
	}
	return nil
}

// Columns returns the board's columns in left-to-right order.
func (s *Store) Columns() []string { return s.cfg.Columns }

// Warnings returns non-fatal problems found while indexing (e.g. malformed
// card files that were skipped).
func (s *Store) Warnings() []string { return s.warnings }

// Count returns the number of indexed cards.
func (s *Store) Count() int { return len(s.cards) }
