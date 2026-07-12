// Command kanren is a plain-text, git-backed kanban tool.
// Cards are per-file markdown; the CLI and a local web board edit the same files.
package main

import (
	"fmt"
	"os"
)

// version is overridden at build time via -ldflags.
var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Printf("kanren %s\n", version)
		return
	}
	fmt.Fprintln(os.Stderr, "kanren: no commands yet — see .specs/features/core/spec.md")
	os.Exit(0)
}
