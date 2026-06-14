package docs

import (
	"strings"
	"testing"
)

func TestAllocateID_FreshToken(t *testing.T) {
	id, err := AllocateID("SPEC", map[string]bool{})
	if err != nil {
		t.Fatalf("AllocateID: %v", err)
	}
	if !strings.HasPrefix(id, "SPEC-") || TokenOf(id) == "" {
		t.Fatalf("id = %q, want SPEC-<token>", id)
	}
}

func TestAllocateID_SkipsUsedTokens(t *testing.T) {
	// Mark the first wordlist pair used; allocation must return a different token.
	first := adjectives[0] + "-" + nouns[0]
	id, err := AllocateID("PLAN", map[string]bool{first: true})
	if err != nil {
		t.Fatalf("AllocateID: %v", err)
	}
	if TokenOf(id) == first {
		t.Fatalf("allocated the used token %q", first)
	}
}

func TestAllocateID_UnknownRole(t *testing.T) {
	if _, err := AllocateID("BOGUS", map[string]bool{}); err == nil {
		t.Fatal("unknown role must error")
	}
}

func TestUsedTokens_StripsRolePrefix(t *testing.T) {
	// SPEC-x and PLAN-x are the same initiative → both reduce to the token x,
	// so the seen-set has one entry and that token won't be re-allocated.
	used := UsedTokens([]string{"SPEC-amber-finch", "PLAN-amber-finch", "DEC-onyx-ridge"})
	if !used["amber-finch"] || !used["onyx-ridge"] {
		t.Fatalf("used = %v", used)
	}
	if len(used) != 2 {
		t.Fatalf("SPEC-x and PLAN-x must collapse to one token; used = %v", used)
	}
}

func TestRoleToken_SiblingDerivation(t *testing.T) {
	// The PLAN of a known SPEC is derivable without allocation.
	if got := RoleToken("PLAN", TokenOf("SPEC-amber-finch")); got != "PLAN-amber-finch" {
		t.Fatalf("sibling = %q, want PLAN-amber-finch", got)
	}
}

func TestAllocateID_UniqueAcrossManyAllocations(t *testing.T) {
	used := map[string]bool{}
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		id, err := AllocateID("REF", used)
		if err != nil {
			t.Fatalf("alloc %d: %v", i, err)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q at %d", id, i)
		}
		seen[id] = true
		used[TokenOf(id)] = true
	}
}
