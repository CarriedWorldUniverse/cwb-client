package docs

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/CarriedWorldUniverse/cwb-client/client"
	"github.com/CarriedWorldUniverse/cwb-client/commonplace"
)

func TestDocPath(t *testing.T) {
	p := DocPath(Frontmatter{Project: "model-stack", Date: "2026-06-14", Title: "Unified Local Model Stack"})
	if p != "model-stack/2026-06-14-unified-local-model-stack.md" {
		t.Fatalf("DocPath = %q", p)
	}
}

// TestWrite_CommitterInvokedWithPathAndBody verifies Write calls the committer
// with the derived path + the full markdown, before indexing. Uses a stub
// committer + (commit-only) skips commonplace by erroring in the committer to
// keep it offline.
func TestWrite_CommitterInvokedWithPathAndBody(t *testing.T) {
	const doc = "---\nid: SPEC-amber-finch\ntitle: T\ntype: spec\nproject: p\nstatus: draft\nversion: 1\ndate: 2026-06-14\nauthor: a\n---\nthe body\n"
	var gotPath string
	var gotMD []byte
	stub := func(_ context.Context, path string, md []byte) error {
		gotPath = path
		gotMD = md
		return errStop // halt before the (offline) commonplace call
	}
	_, err := Write(context.Background(), nil, stub, []byte(doc))
	if err == nil {
		t.Fatal("expected the stop error from the committer")
	}
	if gotPath != "p/2026-06-14-t.md" {
		t.Fatalf("committer path = %q", gotPath)
	}
	if !strings.Contains(string(gotMD), "the body") {
		t.Fatalf("committer md missing body: %q", gotMD)
	}
}

func TestWrite_RejectsInvalidFrontmatter(t *testing.T) {
	_, err := Write(context.Background(), nil, nil, []byte("no frontmatter"))
	if err == nil {
		t.Fatal("Write must reject a doc with no/invalid frontmatter before committing")
	}
}

// errStop is a sentinel used to halt Write after the committer in the unit test.
var errStop = stopErr{}

type stopErr struct{}

func (stopErr) Error() string { return "stop" }

// TestWrite_Live is the env-gated capability test (mirrors cw kb's TestLiveKB):
// against live commonplace as docs-writer, writing a doc makes it surface for a
// DIFFERENTLY-WORDED query. Set CW_DOCS_EDGE + CW_DOCS_TOKEN to run; skips otherwise.
func TestWrite_Live(t *testing.T) {
	edge := os.Getenv("CW_DOCS_EDGE")
	tok := os.Getenv("CW_DOCS_TOKEN")
	if edge == "" || tok == "" {
		t.Skip("set CW_DOCS_EDGE + CW_DOCS_TOKEN to run the live capability test")
	}
	c := client.WithStaticToken(edge, tok)
	ctx := context.Background()
	// Unique marker so the search is unambiguous; body wording differs from the query.
	marker := os.Getenv("CW_DOCS_MARKER")
	if marker == "" {
		marker = "cedarharbor-capability"
	}
	doc := "---\nid: NOTE-" + marker + "\ntitle: Capability probe " + marker +
		"\ntype: note\nproject: doc-store\nstatus: draft\nversion: 1\ndate: 2026-06-14\nauthor: docs-writer\ntags: [probe]\n---\n" +
		"This document proves the cedar-harbor write path: a markdown note indexed into the knowledge pillar (" + marker + ").\n"
	res, err := Write(ctx, c, nil, []byte(doc))
	if err != nil {
		t.Fatalf("Write live: %v", err)
	}
	if res.Entry.ID == "" {
		t.Fatalf("no commonplace entry id: %+v", res)
	}
	// Differently-worded query (semantic, not substring of the body).
	hits, err := commonplace.Search(ctx, c, "central documentation store search "+marker, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	for _, h := range hits {
		if strings.Contains(h.Entry.Topic, marker) {
			return // capability confirmed: written doc surfaced by search
		}
	}
	t.Fatalf("written doc (%s) did not surface in search of %d hits", res.ID, len(hits))
}
