package docs

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/CarriedWorldUniverse/cwb-client/client"
	"github.com/CarriedWorldUniverse/cwb-client/commonplace"
)

// Committer commits a document's markdown to the source-of-truth store (the
// cairn doc-store repo) at the given repo-relative path. The cwb-client/cairn
// client manages repos/PRs, not file commits, so committing files is git work —
// the `cw docs` CLI provides a git-shell implementation. A nil Committer skips
// the commit (index-only): used by the env-gated live test and any
// already-committed path.
type Committer func(ctx context.Context, path string, md []byte) error

// WriteResult reports what Write did.
type WriteResult struct {
	ID        string
	Path      string
	Entry     commonplace.Entry
	Committed bool
}

var slugStrip = regexp.MustCompile(`[^a-z0-9]+`)

// DocPath is the repo-relative path a doc files to: <project>/<date>-<slug>.md,
// slug derived from the title. The path is convenience; the id is the durable
// handle (the registry resolves id→current path).
func DocPath(fm Frontmatter) string {
	slug := strings.Trim(slugStrip.ReplaceAllString(strings.ToLower(fm.Title), "-"), "-")
	if slug == "" {
		slug = strings.ToLower(fm.ID)
	}
	return fmt.Sprintf("%s/%s-%s.md", fm.Project, fm.Date, slug)
}

// Write publishes a document end to end: validate its frontmatter, optionally
// commit the markdown to the source store (via commit), and Store its body into
// commonplace so it is semantically searchable. The doc MUST already carry a
// valid frontmatter id (allocation against the repo seen-set is the caller's
// job — ids are repo-scoped). visibility is "org" (the shared doc-store
// visibility, SPEC §D1). One call → a versioned doc that is findable.
func Write(ctx context.Context, c client.Doer, commit Committer, raw []byte) (WriteResult, error) {
	fm, body, err := Parse(raw)
	if err != nil {
		return WriteResult{}, err
	}
	res := WriteResult{ID: fm.ID, Path: DocPath(fm)}
	if commit != nil {
		if err := commit(ctx, res.Path, raw); err != nil {
			return WriteResult{}, fmt.Errorf("docs: commit to source store: %w", err)
		}
		res.Committed = true
	}
	e, err := commonplace.Store(ctx, c, commonplace.StoreInput{
		Topic:      fm.ID + " " + fm.Title,
		Content:    string(body),
		Visibility: "org",
		Tags:       fm.Tags,
	})
	if err != nil {
		return WriteResult{}, fmt.Errorf("docs: index into commonplace: %w", err)
	}
	res.Entry = e
	return res, nil
}
