package laws

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/policy/cedar"
	"example.com/second-brain/internal/core/manifest"
)

func TestLAW15_StewardsNeverApproveTheirOwn(t *testing.T) {
	ev, err := LoadPolicies()
	if err != nil {
		t.Fatal(err)
	}
	stewardApprovals = 0
	Check(t, "LAW-15", runs, func(r *rand.Rand) error { return law15Stewards(r, ev, manifest.StewardChange) })
	if stewardApprovals == 0 {
		t.Fatal("no proposal was ever approved: the law test checked nothing")
	}
}

// The steward law test catches policies without the self-approval forbid, and a steward
// rule that lets any change through.
func TestSelf_StewardMutationsAreCaught(t *testing.T) {
	files := map[string][]byte{}
	for _, f := range []string{"base.cedar", "spaces.cedar"} {
		b, err := os.ReadFile(filepath.Join(repoRoot(), "policy", "platform", f))
		if err != nil {
			t.Fatal(err)
		}
		files[f] = b
	}
	base := string(files["base.cedar"])
	forbid := "forbid (principal, action == Action::\"approve_space\", resource is SpaceProposal)\nwhen { resource.requester == principal };"
	if !strings.Contains(base, forbid) {
		t.Fatal("base.cedar no longer has the self-approval forbid")
	}
	files["base.cedar"] = []byte(strings.Replace(base, forbid, "", 1))
	selfish, err := cedar.Load(files)
	if err != nil {
		t.Fatal(err)
	}
	if !violates(runs, func(r *rand.Rand) error { return law15Stewards(r, selfish, manifest.StewardChange) }) {
		t.Error("LAW-15 (stewards): policies that let a steward approve their own proposal were not caught")
	}
	ev, err := LoadPolicies()
	if err != nil {
		t.Fatal(err)
	}
	anything := func(cur manifest.Tenant, space string, raw json.RawMessage) (manifest.Tenant, []manifest.Diag, error) {
		var sp manifest.Space
		if err := json.Unmarshal(raw, &sp); err != nil {
			return manifest.Tenant{}, nil, err
		}
		out, ds := manifest.WithSpace(cur, space, sp) // the checks of sb tenant check, none of a steward's
		return out, ds, nil
	}
	if !violates(runs, func(r *rand.Rand) error { return law15Stewards(r, ev, anything) }) {
		t.Error("LAW-19 (stewards): a rule that lets a steward lower a label or change a template was not caught")
	}
}
