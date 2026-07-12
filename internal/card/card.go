// Package card defines the on-disk card model: one markdown file per card
// with YAML frontmatter. It is a pure codec — no filesystem I/O lives here
// (see .specs/STATE.md AD-001). See .specs/features/core/spec.md (CARD-*).
package card

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/goccy/go-yaml"
)

// delim is the frontmatter fence.
const delim = "---"

// Card is the parsed representation of a single card file.
// Frontmatter fields are declared in the order they serialize to disk;
// goccy/go-yaml preserves struct field order on Marshal, which keeps card
// edits to minimal, deterministic git diffs (AD-002).
type Card struct {
	ID       int      `yaml:"id"`
	Title    string   `yaml:"title"`
	Status   string   `yaml:"status"`
	Tags     []string `yaml:"tags,omitempty"`
	Assignee string   `yaml:"assignee,omitempty"`
	Created  string   `yaml:"created,omitempty"`
	Order    float64  `yaml:"order,omitempty"`
	Body     string   `yaml:"-"`
}

// Parse decodes a card file's bytes into a Card. It reads only the first
// frontmatter block (the opening "---" line through the first following "---"
// line); any "---" lines inside the body are left untouched (CARD-03 edge).
//
// It returns an error naming the reason on malformed input and never panics.
func Parse(data []byte) (Card, error) {
	var c Card

	rest, ok := bytes.CutPrefix(data, []byte(delim+"\n"))
	if !ok {
		return c, errors.New("card: missing opening frontmatter fence '---'")
	}

	// Closing fence: the first "\n---" that begins a line, either followed by
	// a newline (body present) or at end of input (no body).
	front, body, ok := splitFrontmatter(rest)
	if !ok {
		return c, errors.New("card: missing closing frontmatter fence '---'")
	}

	if err := yaml.Unmarshal(front, &c); err != nil {
		return c, fmt.Errorf("card: invalid frontmatter YAML: %w", err)
	}
	c.Body = body
	return c, nil
}

// splitFrontmatter returns the YAML bytes (without fences) and the body string.
// rest is the input with the opening "---\n" already removed.
func splitFrontmatter(rest []byte) (front []byte, body string, ok bool) {
	// Case 1: closing fence mid-document -> "\n---\n" separates front and body.
	if i := bytes.Index(rest, []byte("\n"+delim+"\n")); i >= 0 {
		return rest[:i+1], string(rest[i+len("\n"+delim+"\n"):]), true
	}
	// Case 2: closing fence is the final line with no trailing newline.
	if bytes.Equal(rest, []byte(delim)) {
		return nil, "", true
	}
	if front, found := bytes.CutSuffix(rest, []byte("\n"+delim)); found {
		return append(front, '\n'), "", true
	}
	return nil, "", false
}

// Marshal serializes a Card back to file bytes: frontmatter fenced by "---",
// then the body verbatim. Parse(Marshal(c)) == c for any valid Card.
func (c Card) Marshal() ([]byte, error) {
	front, err := yaml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("card: marshal frontmatter: %w", err)
	}
	var buf bytes.Buffer
	buf.WriteString(delim + "\n")
	buf.Write(front) // goccy output already ends with "\n"
	buf.WriteString(delim + "\n")
	buf.WriteString(c.Body)
	return buf.Bytes(), nil
}

// Slugify converts a title into a filename-safe slug. Full implementation
// lands in T2; kept as a placeholder so the scaffold test stays green.
func Slugify(title string) string {
	return title
}
