// Command kanren is a plain-text, git-backed kanban tool.
// Cards are per-file markdown; the CLI and a local web board edit the same
// files. See .specs/features/core/spec.md.
package main

import (
	"fmt"
	"io"
	"os"
)

// version is overridden at build time via -ldflags.
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run dispatches a subcommand and returns a process exit code. It takes its
// output writers so tests can drive it without spawning a process.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "kanren %s\n", version)
		return 0
	case "init":
		return cmdInit(rest, stdout, stderr)
	case "add":
		return cmdAdd(rest, stdout, stderr)
	case "ls":
		return cmdLs(rest, stdout, stderr)
	case "mv":
		return cmdMv(rest, stdout, stderr)
	case "edit":
		return cmdEdit(rest, stdout, stderr)
	case "serve":
		return cmdServe(rest, stdout, stderr)
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "kanren: unknown command %q\n", cmd)
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `kanren — plain-text, git-backed kanban

Usage:
  kanren init                 scaffold a board in the current directory
  kanren add "<title>"        create a card in the leftmost column
  kanren ls [filters]         list cards grouped by column
  kanren mv <id> <status>     move a card to another column
  kanren edit <id>            open a card in $EDITOR
  kanren serve [--port N]     open the local board in a browser
  kanren version              print version
`)
}
