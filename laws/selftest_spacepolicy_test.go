package laws

import (
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/policy/cedar"
)

// LAW-19's policy test catches a cell where a space's allow skips the source's live ACL.
func TestSelf_LAW19PolicyMutationIsCaught(t *testing.T) {
	mk := func() SpaceSystem {
		s := NewSpacePolicySystem().(*spacePolicy)
		s.sourceLast = false
		return s
	}
	if !violates(runs, func(r *rand.Rand) error { return law19(r, mk()) }) {
		t.Error("LAW-19 (policy): a space allow that skips the source ACL was not caught")
	}
}

// The shipped space policies already hold LAW-20 and LAW-22 at the policy layer, and agree
// with the reference on every ask. Their law tests stay on the cell's ask path (SB-1006,
// SB-1203); this is evidence for those tickets, not a substitute.
func TestSelf_SpacePolicyHoldsLAW20LAW22AndMatchesTheReference(t *testing.T) {
	for _, law := range []func(*rand.Rand, SpaceSystem) error{law20, law22} {
		if violates(runs, func(r *rand.Rand) error { return law(r, NewSpacePolicySystem()) }) {
			t.Fatal("the space policies break LAW-20 or LAW-22")
		}
	}
	for seed := int64(0); seed < 30; seed++ {
		sp := genSpaces(rand.New(rand.NewSource(seed)))
		ref, pol := &refSpaces{}, NewSpacePolicySystem()
		_ = ref.Load(sp)
		_ = pol.Load(sp)
		for _, a := range asks(sp) {
			want, wr, _ := ref.Ask(a[0], a[1])
			got, gr, _ := pol.Ask(a[0], a[1])
			if wr != gr || len(want) != len(got) || !sameSet(want, got) {
				t.Fatalf("seed %d, %s in %s: policy %v (refused %v), reference %v (refused %v)", seed, a[0], a[1], got, gr, want, wr)
			}
		}
	}
}

func TestLAW03_MissingSpaceFactsRefuse(t *testing.T) {
	ev, err := LoadPolicies()
	if err != nil {
		t.Fatal(err)
	}
	Check(t, "LAW-03", 40, func(r *rand.Rand) error { return law03SpaceFacts(r, ev) })
}

// LAW-03's space-facts test catches the space policies without their missing-fact rule.
func TestSelf_LAW03SpaceFactsMutationIsCaught(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(here), "..")
	files := map[string][]byte{}
	for _, f := range []string{"policy/platform/base.cedar", "policy/platform/spaces.cedar"} {
		b, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatal(err)
		}
		files[filepath.Base(f)] = b
	}
	s := string(files["spaces.cedar"])
	i := strings.Index(s, "// 0.")
	j := strings.Index(s, "// 1.")
	if i < 0 || j < i {
		t.Fatal("spaces.cedar no longer has rule 0 before rule 1")
	}
	files["spaces.cedar"] = []byte(s[:i] + s[j:])
	ev, err := cedar.Load(files)
	if err != nil {
		t.Fatal(err)
	}
	if !violates(40, func(r *rand.Rand) error { return law03SpaceFacts(r, ev) }) {
		t.Error("LAW-03 (space facts): policies without the missing-fact rule were not caught")
	}
}

// LAW-20 on the ask path catches retrieval without its space step: the live ACL alone
// lets a viewer read restricted content of a space they aren't in.
func TestSelf_LAW20AskMutationIsCaught(t *testing.T) {
	mk := func() SpaceSystem { a := newAskSpaces(); a.skipSpaces = true; return a }
	if !violates(runs, func(r *rand.Rand) error { return law20(r, mk()) }) {
		t.Error("LAW-20 (ask): retrieval without the space step was not caught")
	}
}

// The ask path agrees with the reference cell on every ask.
func TestSelf_AskSpacesMatchTheReference(t *testing.T) {
	for seed := int64(0); seed < 30; seed++ {
		sp := genSpaces(rand.New(rand.NewSource(seed)))
		ref, sys := &refSpaces{}, NewSpaceSystem()
		_ = ref.Load(sp)
		if err := sys.Load(sp); err != nil {
			t.Fatal(err)
		}
		for _, a := range asks(sp) {
			want, wr, _ := ref.Ask(a[0], a[1])
			got, gr, err := sys.Ask(a[0], a[1])
			if err != nil || wr != gr || !sameSet(want, got) {
				t.Fatalf("seed %d, %s in %s: ask %v (refused %v, %v), reference %v (refused %v)", seed, a[0], a[1], got, gr, err, want, wr)
			}
		}
	}
}

func TestLAW19_AsksOnlyNarrow(t *testing.T) {
	Check(t, "LAW-19", runs, func(r *rand.Rand) error { return law19(r, NewSpaceSystem()) })
}
