package docs

import "testing"

func TestParseRef(t *testing.T) {
	cases := []struct {
		in      string
		wantID  string
		wantVer int
		wantErr bool
	}{
		{"SPEC-amber-finch", "SPEC-amber-finch", 0, false},
		{"SPEC-amber-finch@v2", "SPEC-amber-finch", 2, false},
		{"  PLAN-onyx-ridge  ", "PLAN-onyx-ridge", 0, false},
		{"", "", 0, true},
		{"@v2", "", 0, true},
		{"SPEC-x@2", "", 0, true},   // missing the v
		{"SPEC-x@v0", "", 0, true},  // version must be positive
		{"SPEC-x@vbad", "", 0, true},
	}
	for _, c := range cases {
		got, err := ParseRef(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseRef(%q): expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseRef(%q): %v", c.in, err)
			continue
		}
		if got.ID != c.wantID || got.Version != c.wantVer {
			t.Errorf("ParseRef(%q) = %+v, want id=%q ver=%d", c.in, got, c.wantID, c.wantVer)
		}
	}
}

func TestBuildIndex_AndResolve(t *testing.T) {
	ix, err := BuildIndex([]DocRef{
		{ID: "SPEC-amber-finch", Path: "model-stack/spec.md"},
		{ID: "PLAN-amber-finch", Path: "model-stack/plan.md"},
	})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if p, ok := ix.Resolve("SPEC-amber-finch"); !ok || p != "model-stack/spec.md" {
		t.Fatalf("resolve SPEC = %q ok=%v", p, ok)
	}
	if _, ok := ix.Resolve("NOPE-x"); ok {
		t.Fatal("resolve of unknown id must be ok=false")
	}
}

func TestBuildIndex_DuplicateID(t *testing.T) {
	_, err := BuildIndex([]DocRef{
		{ID: "SPEC-amber-finch", Path: "a.md"},
		{ID: "SPEC-amber-finch", Path: "b.md"},
	})
	if err == nil {
		t.Fatal("duplicate id must error (uniqueness invariant)")
	}
}
