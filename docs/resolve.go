package docs

import (
	"fmt"
	"strconv"
	"strings"
)

// Reference resolution (SPEC-cedar-harbor §5.4/§5.5): a bare `<id>` resolves to
// the current version; `<id>@vN` addresses a prior version (git history). The
// docs-core layer owns ref PARSING + the id→path index built from the repo;
// fetching a specific @vN's bytes is git-history work the caller (cw docs) does.

// Ref is a parsed document reference.
type Ref struct {
	ID      string // e.g. SPEC-amber-finch
	Version int    // 0 = current (bare id); N = the @vN pin
}

// ParseRef parses `<id>` or `<id>@vN` into a Ref. A bare id yields Version 0
// (current). Errors on an empty id or a malformed @ suffix.
func ParseRef(ref string) (Ref, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return Ref{}, fmt.Errorf("docs: empty reference")
	}
	id, ver, found := strings.Cut(ref, "@")
	if id == "" {
		return Ref{}, fmt.Errorf("docs: reference %q has no id", ref)
	}
	if !found {
		return Ref{ID: id, Version: 0}, nil
	}
	if !strings.HasPrefix(ver, "v") {
		return Ref{}, fmt.Errorf("docs: version suffix %q must be vN (e.g. @v2)", ver)
	}
	n, err := strconv.Atoi(ver[1:])
	if err != nil || n <= 0 {
		return Ref{}, fmt.Errorf("docs: version suffix %q must be a positive vN", ver)
	}
	return Ref{ID: id, Version: n}, nil
}

// DocRef is one document's identity + location, as scanned from the repo.
type DocRef struct {
	ID   string
	Path string
}

// Index maps a document id to its current path. It is derived from the cairn
// repo (scan each doc's `id:` frontmatter → path) and is rebuildable, so it
// holds no state the git tree doesn't (SPEC §8).
type Index map[string]string

// BuildIndex constructs the id→path index from scanned doc refs, erroring on a
// duplicate id (the registry's uniqueness invariant must hold in the tree).
func BuildIndex(refs []DocRef) (Index, error) {
	ix := make(Index, len(refs))
	for _, r := range refs {
		if r.ID == "" {
			continue
		}
		if existing, dup := ix[r.ID]; dup {
			return nil, fmt.Errorf("docs: duplicate id %q at %q and %q", r.ID, existing, r.Path)
		}
		ix[r.ID] = r.Path
	}
	return ix, nil
}

// Resolve returns the current path for an id, or ok=false if unknown.
func (ix Index) Resolve(id string) (string, bool) {
	p, ok := ix[id]
	return p, ok
}
