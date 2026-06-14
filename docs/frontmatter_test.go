package docs

import (
	"strings"
	"testing"
)

// validDoc mirrors a real doc-store document (the T1 contract).
const validDoc = `---
id: SPEC-amber-finch
title: Test Doc
type: spec
project: references
status: draft
version: 1
date: 2026-06-14
author: docs-writer
related: [NEX-642, PLAN-amber-finch]
tags: [a, b]
---
# Body
content here
`

func TestParse_Valid(t *testing.T) {
	fm, body, err := Parse([]byte(validDoc))
	if err != nil {
		t.Fatalf("Parse valid: %v", err)
	}
	if fm.ID != "SPEC-amber-finch" || fm.Type != "spec" || fm.Project != "references" ||
		fm.Status != "draft" || fm.Version != 1 || fm.Author != "docs-writer" || fm.Title != "Test Doc" {
		t.Fatalf("frontmatter mismatch: %+v", fm)
	}
	if len(fm.Related) != 2 || fm.Related[0] != "NEX-642" || len(fm.Tags) != 2 {
		t.Fatalf("related/tags: %v / %v", fm.Related, fm.Tags)
	}
	if !strings.HasPrefix(string(body), "# Body") {
		t.Fatalf("body = %q", string(body))
	}
}

func TestParse_InvalidNamesOffendingField(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(string) string
		wantField string
	}{
		{"missing id", func(s string) string { return strings.Replace(s, "id: SPEC-amber-finch\n", "", 1) }, "id"},
		{"missing title", func(s string) string { return strings.Replace(s, "title: Test Doc\n", "", 1) }, "title"},
		{"missing type", func(s string) string { return strings.Replace(s, "type: spec\n", "", 1) }, "type"},
		{"missing project", func(s string) string { return strings.Replace(s, "project: references\n", "", 1) }, "project"},
		{"missing status", func(s string) string { return strings.Replace(s, "status: draft\n", "", 1) }, "status"},
		{"missing version", func(s string) string { return strings.Replace(s, "version: 1\n", "", 1) }, "version"},
		{"missing date", func(s string) string { return strings.Replace(s, "date: 2026-06-14\n", "", 1) }, "date"},
		{"missing author", func(s string) string { return strings.Replace(s, "author: docs-writer\n", "", 1) }, "author"},
		{"bad type", func(s string) string { return strings.Replace(s, "type: spec", "type: bogus", 1) }, "type"},
		{"bad status", func(s string) string { return strings.Replace(s, "status: draft", "status: bogus", 1) }, "status"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := Parse([]byte(c.mutate(validDoc)))
			if err == nil {
				t.Fatalf("%s: expected a validation error", c.name)
			}
			if !strings.Contains(err.Error(), c.wantField) {
				t.Fatalf("%s: error %q should name field %q", c.name, err.Error(), c.wantField)
			}
		})
	}
}

func TestParse_NoFrontmatter(t *testing.T) {
	if _, _, err := Parse([]byte("just a body, no fence")); err == nil {
		t.Fatal("expected error when frontmatter fence is absent")
	}
}

func TestParse_UnterminatedFrontmatter(t *testing.T) {
	if _, _, err := Parse([]byte("---\nid: X\ntitle: y\n(no closing fence)")); err == nil {
		t.Fatal("expected error when the closing --- fence is missing")
	}
}
