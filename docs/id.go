package docs

import (
	"fmt"
	"strings"
)

// Reference-ID system (SPEC-cedar-harbor §5): an ID is `<ROLE>-<token>` where
// the token is a speakable mnemonic word-pair (`amber-finch`). The TOKEN
// identifies the initiative; the ROLE prefix is the artifact facet — so
// `SPEC-amber-finch` and `PLAN-amber-finch` are the same work. Uniqueness is by
// TOKEN (across roles); the registry dedups against tokens already used, so the
// token needs no UUID-strength randomness — it stays short and speakable.

// Roles is the controlled set of ID role prefixes.
var Roles = map[string]bool{
	"SPEC": true, "PLAN": true, "DEC": true, "REF": true, "RES": true, "RUN": true, "NOTE": true,
}

// adjectives + nouns compose the token wordlist (adjective-noun). Curated for
// speakability + non-ambiguity; word-pairs give len(adjectives)*len(nouns)
// tokens, expandable by extending either list.
var (
	adjectives = []string{
		"amber", "onyx", "cedar", "slate", "ivory", "cobalt", "crimson", "jade",
		"umber", "azure", "russet", "sable", "teal", "ochre", "indigo", "pewter",
		"garnet", "hazel", "flint", "basalt", "copper", "marble", "saffron", "olive",
	}
	nouns = []string{
		"finch", "kestrel", "harbor", "falcon", "heron", "sparrow", "raven", "marten",
		"otter", "lynx", "hollow", "ridge", "delta", "fjord", "mesa", "vale",
		"reef", "cove", "spire", "grove", "thicket", "meadow", "summit", "current",
	}
)

// RoleToken composes an ID from a role + token. It does not allocate or dedup;
// use it to derive the sibling artifact of a known initiative (e.g. the PLAN of
// a SPEC: RoleToken("PLAN", TokenOf("SPEC-amber-finch"))).
func RoleToken(role, token string) string { return role + "-" + token }

// TokenOf returns the initiative token of an ID (everything after the role
// prefix), or "" if the ID has no `<ROLE>-` prefix.
func TokenOf(id string) string {
	i := strings.IndexByte(id, '-')
	if i < 0 {
		return ""
	}
	return id[i+1:]
}

// UsedTokens reduces a set of existing IDs to the set of initiative tokens in
// use — the dedup seen-set (derived from the cairn repo's `id:` frontmatter).
func UsedTokens(ids []string) map[string]bool {
	used := make(map[string]bool, len(ids))
	for _, id := range ids {
		if tok := TokenOf(id); tok != "" {
			used[tok] = true
		}
	}
	return used
}

// AllocateID mints a fresh `<role>-<token>` whose token is not already used.
// Deterministic (first unused pair in wordlist order) — uniqueness, not
// unpredictability, is the goal (the registry enforces it; §5.3). Returns an
// error if the role is unknown or the wordlist is exhausted.
func AllocateID(role string, usedTokens map[string]bool) (string, error) {
	if !Roles[role] {
		return "", fmt.Errorf("docs: unknown role %q (want one of SPEC/PLAN/DEC/REF/RES/RUN/NOTE)", role)
	}
	for _, a := range adjectives {
		for _, n := range nouns {
			tok := a + "-" + n
			if !usedTokens[tok] {
				return RoleToken(role, tok), nil
			}
		}
	}
	return "", fmt.Errorf("docs: token wordlist exhausted (%d pairs all used)", len(adjectives)*len(nouns))
}
