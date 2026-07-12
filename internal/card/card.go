// Package card defines the on-disk card model: one markdown file per card
// with YAML frontmatter. See .specs/features/core/spec.md (CARD-*).
package card

// Card is the parsed representation of a single card file.
// Fields map to YAML frontmatter; Body is the markdown after it.
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

// Slugify is a placeholder for filename slug generation (CARD-01).
// Real implementation lands with the card-model task.
func Slugify(title string) string {
	return title
}
