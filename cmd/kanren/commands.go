package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	"github.com/vitor/kanren/internal/card"
	"github.com/vitor/kanren/internal/store"
)

// boardDir is the current directory; a board lives wherever kanren is run.
const boardDir = "."

// openStore opens the board or prints why it could not, returning an exit code
// alongside. Callers return that code directly on failure.
func openStore(stderr io.Writer) (*store.Store, int) {
	s, err := store.Open(boardDir)
	if err != nil {
		fmt.Fprintf(stderr, "kanren: %v\n", err)
		return nil, 1
	}
	for _, w := range s.Warnings() {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}
	return s, 0
}

func cmdInit(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: kanren init")
		return 2
	}
	if err := store.Init(boardDir); err != nil {
		fmt.Fprintf(stderr, "kanren: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "initialized board (.kanren.yml, cards/)")
	return 0
}

func cmdAdd(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || args[0] == "" {
		fmt.Fprintln(stderr, `usage: kanren add "<title>"`)
		return 2
	}
	s, code := openStore(stderr)
	if code != 0 {
		return code
	}
	c, err := s.Add(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "kanren: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "#%d  %s\n", c.ID, card.Filename(c.ID, c.Title))
	return 0
}

func cmdMv(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "usage: kanren mv <id> <status>")
		return 2
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "kanren: invalid id %q\n", args[0])
		return 2
	}
	s, code := openStore(stderr)
	if code != 0 {
		return code
	}
	if err := s.Move(id, args[1]); err != nil {
		fmt.Fprintf(stderr, "kanren: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "moved #%d to %s\n", id, args[1])
	return 0
}

func cmdEdit(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: kanren edit <id>")
		return 2
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "kanren: invalid id %q\n", args[0])
		return 2
	}
	s, code := openStore(stderr)
	if code != 0 {
		return code
	}
	c, err := s.Get(id)
	if err != nil {
		fmt.Fprintf(stderr, "kanren: %v\n", err)
		return 1
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		fmt.Fprintln(stderr, "kanren: $EDITOR is not set")
		return 1
	}
	cmd := exec.Command(editor, s.Path(c))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(stderr, "kanren: editor exited with error: %v\n", err)
		return 1
	}
	// Re-validate the edited file so a broken save is reported, not swallowed.
	if _, err := store.Open(boardDir); err != nil {
		fmt.Fprintf(stderr, "kanren: card no longer parses after edit: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "edited #%d\n", id)
	return 0
}
