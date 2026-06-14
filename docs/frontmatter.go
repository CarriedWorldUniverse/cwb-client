// Package docs is the shared docs-core for cedar-harbor (SPEC-cedar-harbor): the
// single place the frontmatter contract, reference-ID rules, and cairn +
// commonplace glue live. `cw docs` (CLI) and the `docs` MCP are thin wrappers
// over it, so the rules exist in exactly one place.
package docs

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Frontmatter is the enforced metadata every doc-store document carries. It is
// the code form of the contract documented in the doc-store repo's
// references/frontmatter.md.
type Frontmatter struct {
	ID           string   `yaml:"id"`
	Title        string   `yaml:"title"`
	Type         string   `yaml:"type"`
	Project      string   `yaml:"project"`
	Status       string   `yaml:"status"`
	Version      int      `yaml:"version"`
	Date         string   `yaml:"date"`
	Author       string   `yaml:"author"`
	Related      []string `yaml:"related,omitempty"`
	Tags         []string `yaml:"tags,omitempty"`
	SupersededBy string   `yaml:"superseded_by,omitempty"`
}

// AllowedTypes / AllowedStatuses are the contract's controlled vocabularies.
var (
	AllowedTypes = map[string]bool{
		"spec": true, "plan": true, "decision": true, "reference": true,
		"research": true, "runbook": true, "note": true,
	}
	AllowedStatuses = map[string]bool{
		"draft": true, "active": true, "approved": true, "superseded": true,
	}
)

// Parse splits the YAML frontmatter from the markdown body, unmarshals it, and
// validates it against the contract. It returns the metadata + the remaining
// body, or a validation error that names the offending field.
func Parse(raw []byte) (Frontmatter, []byte, error) {
	var fm Frontmatter
	body, err := split(raw, &fm)
	if err != nil {
		return Frontmatter{}, nil, err
	}
	if err := fm.Validate(); err != nil {
		return Frontmatter{}, nil, err
	}
	return fm, body, nil
}

// split extracts the `---`-fenced YAML block at the head of raw into out and
// returns the body after the closing fence.
func split(raw []byte, out *Frontmatter) ([]byte, error) {
	if !bytes.HasPrefix(raw, []byte("---\n")) && !bytes.HasPrefix(raw, []byte("---\r\n")) {
		return nil, fmt.Errorf("docs: no frontmatter (document must start with a --- fence)")
	}
	rest := bytes.TrimLeft(raw[len("---"):], "\r\n")
	idx := bytes.Index(rest, []byte("\n---"))
	if idx < 0 {
		return nil, fmt.Errorf("docs: unterminated frontmatter (no closing --- fence)")
	}
	yamlBlock := rest[:idx]
	body := bytes.TrimLeft(rest[idx+len("\n---"):], "\r\n")
	if err := yaml.Unmarshal(yamlBlock, out); err != nil {
		return nil, fmt.Errorf("docs: frontmatter yaml: %w", err)
	}
	return body, nil
}

// Validate enforces the contract: every required field present, version a
// positive integer, type + status within their vocabularies. The error names
// the offending field so a caller (lint, write_doc) can point at it.
func (fm Frontmatter) Validate() error {
	for _, f := range []struct{ name, val string }{
		{"id", fm.ID}, {"title", fm.Title}, {"type", fm.Type}, {"project", fm.Project},
		{"status", fm.Status}, {"date", fm.Date}, {"author", fm.Author},
	} {
		if f.val == "" {
			return fmt.Errorf("docs: frontmatter missing required field %q", f.name)
		}
	}
	if fm.Version <= 0 {
		return fmt.Errorf("docs: frontmatter field \"version\" must be a positive integer, got %d", fm.Version)
	}
	if !AllowedTypes[fm.Type] {
		return fmt.Errorf("docs: frontmatter field \"type\" = %q not in {spec,plan,decision,reference,research,runbook,note}", fm.Type)
	}
	if !AllowedStatuses[fm.Status] {
		return fmt.Errorf("docs: frontmatter field \"status\" = %q not in {draft,active,approved,superseded}", fm.Status)
	}
	return nil
}
