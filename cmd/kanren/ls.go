package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/vitorqf/kanren/internal/card"
	"github.com/vitorqf/kanren/internal/store"
)

// cmdLs lists cards, optionally filtered. With --json it emits a machine-
// readable array; otherwise cards are grouped by column (QRY-01/02/03/04).
func cmdLs(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ls", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var f store.Filter
	asJSON := fs.Bool("json", false, "emit JSON array")
	fs.StringVar(&f.Status, "status", "", "filter by column/status")
	fs.StringVar(&f.Tag, "tag", "", "filter by tag")
	fs.StringVar(&f.Assignee, "assignee", "", "filter by assignee")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	s, code := openStore(stderr)
	if code != 0 {
		return code
	}

	cards := s.List(f)

	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		// Encode a guaranteed non-nil slice so no match prints "[]", not "null".
		if cards == nil {
			cards = []card.Card{}
		}
		if err := enc.Encode(cards); err != nil {
			fmt.Fprintf(stderr, "kanren: %v\n", err)
			return 1
		}
		return 0
	}

	// Grouped human view. When a status filter is set, show only that group.
	for _, col := range s.Columns() {
		if f.Status != "" && col != f.Status {
			continue
		}
		group := filterByColumn(cards, col)
		fmt.Fprintf(stdout, "\n%s (%d)\n", col, len(group))
		for _, c := range group {
			fmt.Fprintf(stdout, "  #%-3d %s\n", c.ID, c.Title)
		}
	}
	if f.Status == "" {
		if mis := s.Misfiled(); len(mis) > 0 {
			fmt.Fprintf(stdout, "\n⚠ misfiled (%d)\n", len(mis))
			for _, c := range mis {
				fmt.Fprintf(stdout, "  #%-3d %s  [status: %s]\n", c.ID, c.Title, c.Status)
			}
		}
	}
	return 0
}

func filterByColumn(cards []card.Card, col string) []card.Card {
	var out []card.Card
	for _, c := range cards {
		if c.Status == col {
			out = append(out, c)
		}
	}
	return out
}
