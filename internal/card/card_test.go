package card

import "testing"

// TestSlugifyPlaceholder is a trivial passing test so CI is green on scaffold
// (CI-03). Replaced by real round-trip parse tests with the card-model task.
func TestSlugifyPlaceholder(t *testing.T) {
	if got := Slugify("fix bug"); got == "" {
		t.Fatal("Slugify returned empty")
	}
}
