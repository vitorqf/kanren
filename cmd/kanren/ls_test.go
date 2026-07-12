package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vitor/kanren/internal/card"
)

// seedCards adds three cards and tags/assigns them via a second store write,
// mirroring the store-level seed so CLI query tests have data.
func seedCLI(t *testing.T) {
	t.Helper()
	initBoard(t)
	execCLI(t, "add", "alpha") // #1
	execCLI(t, "add", "beta")  // #2
	execCLI(t, "add", "gamma") // #3
	// Move #1 to doing so filters have something to discriminate.
	if code, _, errs := execCLI(t, "mv", "1", "doing"); code != 0 {
		t.Fatalf("mv: %d %s", code, errs)
	}
}

// TestLsStatusFilter: --status returns only that column (QRY-01).
func TestLsStatusFilter(t *testing.T) {
	seedCLI(t)
	code, out, _ := execCLI(t, "ls", "--status", "doing")
	if code != 0 {
		t.Fatalf("ls: %d", code)
	}
	if !strings.Contains(out, "alpha") {
		t.Errorf("doing should contain alpha:\n%s", out)
	}
	if strings.Contains(out, "beta") || strings.Contains(out, "gamma") {
		t.Errorf("doing filter leaked todo cards:\n%s", out)
	}
}

// TestLsJSON: --json emits a valid parseable array of the matching cards
// (QRY-02).
func TestLsJSON(t *testing.T) {
	seedCLI(t)
	code, out, _ := execCLI(t, "ls", "--json")
	if code != 0 {
		t.Fatalf("ls --json: %d", code)
	}
	var cards []card.Card
	if err := json.Unmarshal([]byte(out), &cards); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if len(cards) != 3 {
		t.Errorf("json length = %d, want 3", len(cards))
	}
}

// TestLsJSONEmptyIsArray: no match under --json prints "[]" and exits 0
// (QRY-03).
func TestLsJSONEmptyIsArray(t *testing.T) {
	seedCLI(t)
	code, out, _ := execCLI(t, "ls", "--json", "--tag", "nonexistent")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	var cards []card.Card
	if err := json.Unmarshal([]byte(out), &cards); err != nil {
		t.Fatalf("empty output not valid JSON array: %v\n%q", err, out)
	}
	if len(cards) != 0 {
		t.Errorf("want empty array, got %d cards", len(cards))
	}
	if !strings.Contains(out, "[]") {
		t.Errorf("empty result should render []: %q", out)
	}
}

// TestLsUnknownFilterNoError: an unknown status/tag returns empty, not error
// (QRY-04).
func TestLsUnknownFilterNoError(t *testing.T) {
	seedCLI(t)
	if code, _, _ := execCLI(t, "ls", "--status", "bogus"); code != 0 {
		t.Errorf("unknown status should exit 0, got %d", code)
	}
	if code, _, _ := execCLI(t, "ls", "--tag", "ghost"); code != 0 {
		t.Errorf("unknown tag should exit 0, got %d", code)
	}
}
