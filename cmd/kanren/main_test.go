package main

import (
	"bytes"
	"strings"
	"testing"
)

// exec runs the CLI in an isolated temp dir and returns exit code + streams.
func execCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = run(args, &out, &errb)
	return code, out.String(), errb.String()
}

// initBoard chdir's into a fresh board for a test.
func initBoard(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	if code, _, errs := execCLI(t, "init"); code != 0 {
		t.Fatalf("init failed: code %d, %s", code, errs)
	}
}

// TestVersion: version prints and exits 0.
func TestVersion(t *testing.T) {
	code, out, _ := execCLI(t, "version")
	if code != 0 || !strings.Contains(out, "kanren") {
		t.Errorf("version: code %d out %q", code, out)
	}
}

// TestUnknownCommand: unknown command exits nonzero with usage.
func TestUnknownCommand(t *testing.T) {
	code, _, errs := execCLI(t, "frobnicate")
	if code == 0 {
		t.Error("unknown command should exit nonzero")
	}
	if !strings.Contains(errs, "unknown command") {
		t.Errorf("stderr = %q, want 'unknown command'", errs)
	}
}

// TestAddThenLs: add creates a card, ls shows it under the leftmost column
// (CLI-01, CLI-03).
func TestAddThenLs(t *testing.T) {
	initBoard(t)
	code, out, errs := execCLI(t, "add", "fix the bug")
	if code != 0 {
		t.Fatalf("add: code %d, %s", code, errs)
	}
	if !strings.Contains(out, "#1") {
		t.Errorf("add output = %q, want id #1", out)
	}
	code, out, _ = execCLI(t, "ls")
	if code != 0 {
		t.Fatalf("ls: code %d", code)
	}
	if !strings.Contains(out, "todo (1)") || !strings.Contains(out, "fix the bug") {
		t.Errorf("ls output missing card:\n%s", out)
	}
}

// TestMvChangesColumn: mv moves a card; ls reflects the new column (CLI-02).
func TestMvChangesColumn(t *testing.T) {
	initBoard(t)
	execCLI(t, "add", "task")
	code, _, errs := execCLI(t, "mv", "1", "doing")
	if code != 0 {
		t.Fatalf("mv: code %d, %s", code, errs)
	}
	_, out, _ := execCLI(t, "ls")
	if !strings.Contains(out, "doing (1)") || !strings.Contains(out, "todo (0)") {
		t.Errorf("ls after mv:\n%s", out)
	}
}

// TestMvBadInput: invalid id and invalid status both exit nonzero (CLI-04).
func TestMvBadInput(t *testing.T) {
	initBoard(t)
	execCLI(t, "add", "task")

	if code, _, _ := execCLI(t, "mv", "notanumber", "doing"); code == 0 {
		t.Error("non-numeric id should exit nonzero")
	}
	if code, _, _ := execCLI(t, "mv", "1", "bogus"); code == 0 {
		t.Error("invalid status should exit nonzero")
	}
	if code, _, _ := execCLI(t, "mv", "99", "doing"); code == 0 {
		t.Error("unknown id should exit nonzero")
	}
}

// TestCommandsNeedBoard: running add outside a board errors with the init hint.
func TestCommandsNeedBoard(t *testing.T) {
	t.Chdir(t.TempDir())
	code, _, errs := execCLI(t, "add", "x")
	if code == 0 {
		t.Error("add without a board should fail")
	}
	if !strings.Contains(errs, "kanren init") {
		t.Errorf("stderr should hint init: %q", errs)
	}
}
