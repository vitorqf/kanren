package card

import (
	"strings"
	"testing"
)

// TestSlugify: filename-safe slugs, unicode/punctuation collapsed, empty
// falls back (CARD-01).
func TestSlugify(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Fix bug":               "fix-bug",
		"Fix   auth  expiry!!!": "fix-auth-expiry",
		"  trim me  ":           "trim-me",
		"CamelCase Title":       "camelcase-title",
		"under_score/slash":     "under-score-slash",
		"café münchen":          "caf-m-nchen",
		"":                      "card",
		"!!!":                   "card",
		"v2.0-release":          "v2-0-release",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestFilename: zero-padded id + slug + .md (CARD-01).
func TestFilename(t *testing.T) {
	t.Parallel()
	cases := []struct {
		id    int
		title string
		want  string
	}{
		{12, "Fix bug", "0012-fix-bug.md"},
		{1, "t", "0001-t.md"},
		{9999, "big", "9999-big.md"},
		{12345, "overflow", "12345-overflow.md"},
		{7, "", "0007-card.md"},
	}
	for _, c := range cases {
		if got := Filename(c.id, c.title); got != c.want {
			t.Errorf("Filename(%d, %q) = %q, want %q", c.id, c.title, got, c.want)
		}
	}
}

// TestRoundTrip proves Parse(Marshal(c)) == c for valid cards (CARD-02).
func TestRoundTrip(t *testing.T) {
	t.Parallel()
	cases := map[string]Card{
		"full": {
			ID: 12, Title: "Fix auth token expiry", Status: "doing",
			Tags: []string{"bug", "urgent"}, Assignee: "vitorqf",
			Created: "2026-07-12", Order: 2,
			Body: "# Fix auth token expiry\n\nCheck uses `<` not `<=`.\n",
		},
		"minimal": {
			ID: 1, Title: "t", Status: "todo",
		},
		"empty body": {
			ID: 3, Title: "no body", Status: "done", Body: "",
		},
		"body with dashes": {
			ID: 4, Title: "has fence", Status: "todo",
			Body: "intro\n\n---\n\nsection after a horizontal rule\n",
		},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			b, err := in.Marshal()
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			got, err := Parse(b)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got.ID != in.ID || got.Title != in.Title || got.Status != in.Status ||
				got.Assignee != in.Assignee || got.Created != in.Created || got.Order != in.Order {
				t.Errorf("scalar mismatch:\n got  %+v\n want %+v", got, in)
			}
			if strings.Join(got.Tags, ",") != strings.Join(in.Tags, ",") {
				t.Errorf("tags mismatch: got %v want %v", got.Tags, in.Tags)
			}
			if got.Body != in.Body {
				t.Errorf("body mismatch:\n got  %q\n want %q", got.Body, in.Body)
			}
		})
	}
}

// TestParseOnlyFirstBlock: a "---" line inside the body is NOT treated as
// frontmatter — only the first fenced block is (CARD-03 edge case).
func TestParseOnlyFirstBlock(t *testing.T) {
	t.Parallel()
	src := "---\nid: 7\ntitle: t\nstatus: todo\n---\nbefore\n---\nafter\n"
	c, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if c.ID != 7 {
		t.Errorf("id: got %d want 7", c.ID)
	}
	if c.Body != "before\n---\nafter\n" {
		t.Errorf("body: got %q want %q", c.Body, "before\n---\nafter\n")
	}
}

// TestParseMalformed: bad input returns a named error and never panics (CARD-03).
func TestParseMalformed(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"no opening fence": "id: 1\ntitle: t\n",
		"no closing fence": "---\nid: 1\ntitle: t\n",
		"invalid yaml":     "---\nid: [unterminated\n---\nbody\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(src))
			if err == nil {
				t.Fatalf("expected error for %s, got nil", name)
			}
			if !strings.HasPrefix(err.Error(), "card:") {
				t.Errorf("error should be namespaced 'card:': %v", err)
			}
		})
	}
}
