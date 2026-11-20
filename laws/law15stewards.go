package laws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"time"

	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	"example.com/second-brain/internal/api"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/proposals"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-15, LAW-16 and LAW-19 for delegated space changes (SB-1201), through the real
// handler and shipped policies: nobody approves their own proposal; a proposal is approved
// only with its exact hash and only against the manifest it was made from, and the result
// is that manifest; and no approved change touches another space or the barriers, changes
// a template, or lowers a label.

const stewardBase = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint }
agents: [ask]
spaces:
  finance:
    template: finance
    members: [group:finance]
    stewards: [group:finance-leads]
    scope: { docs: ["/Finance/**"] }
  eng:
    template: engineering
    members: [group:eng]
    stewards: [group:eng-leads]
    scope: { docs: ["/Engineering/**"] }
barriers:
  - [group:deal-team, group:eng]
`

type appliedManifest struct{ t manifest.Tenant }

func (a *appliedManifest) Applied(context.Context) (manifest.Tenant, string, error) {
	return a.t, "c1", nil
}

// StewardRule is the steward rule the handler uses (manifest.StewardChange in the real system).
type StewardRule func(t manifest.Tenant, space string, proposed json.RawMessage) (manifest.Tenant, []manifest.Diag, error)

type stewardCell struct {
	h       api.Stewards
	applied *appliedManifest
	mux     *http.ServeMux
}

func newStewardCell(ev policy.Evaluator, rule StewardRule) (*stewardCell, error) {
	t, ds := manifest.Check([]byte(stewardBase))
	if len(ds) > 0 {
		return nil, fmt.Errorf("base manifest: %v", ds)
	}
	c := &stewardCell{applied: &appliedManifest{t: t}, mux: http.NewServeMux()}
	n := 0
	c.h = api.Stewards{Tenant: "acme-logistics", Policy: policy.Kernel{Engine: ev, Recorder: okRecorder{}}, Manifests: c.applied,
		Audit:     &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_law" }, Store: auditmem.New(), Now: time.Now},
		Proposals: &proposals.MemoryStore{}, NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }, Now: time.Now, Change: rule}
	c.h.Register(c.mux)
	return c, nil
}

func (c *stewardCell) call(user string, groups []string, method, path string, body any) (int, map[string]any) {
	s, _ := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", user, groups)
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(method, path, bytes.NewReader(b)).WithContext(tenancy.WithScope(context.Background(), s))
	w := httptest.NewRecorder()
	c.mux.ServeHTTP(w, r)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

type person struct {
	id     string
	groups []string
}

func law15Stewards(r *rand.Rand, ev policy.Evaluator, rule StewardRule) error {
	c, err := newStewardCell(ev, rule)
	if err != nil {
		return err
	}
	people := []person{
		{"usr_lead", []string{"finance-leads"}},
		{"usr_both", []string{"finance-leads", "sb-operators"}}, // a steward who is also an operator
		{"usr_op", []string{"sb-operators"}},
		{"usr_out", []string{"eng"}},
	}
	type prop struct{ id, hash, requester string }
	var props []prop
	for i := 0; i < 16; i++ {
		p := people[r.Intn(len(people))]
		before := c.applied.t
		switch r.Intn(6) {
		case 0, 1: // propose a change to finance
			sp := map[string]any{"template": "finance", "members": []string{"group:finance"}, "stewards": []string{"group:finance-leads"},
				"scope": map[string][]string{"docs": {"/Finance/**"}}}
			switch r.Intn(5) {
			case 0:
				sp["members"] = []string{"group:finance", "group:execs"}
			case 1:
				sp["label"] = "internal" // lowering a restricted space
			case 2:
				sp["template"] = "custom"
			case 3:
				sp["barriers"] = [][]string{} // not a space field
			default:
				sp["retention_days"] = 90
			}
			code, out := c.call(p.id, p.groups, "POST", "/v1/spaces/finance/proposals", map[string]any{"space": sp})
			if code == 200 {
				props = append(props, prop{out["id"].(string), out["hash"].(string), p.id})
			}
		case 2, 3, 4: // decide on one
			if len(props) == 0 {
				continue
			}
			pr := props[len(props)-1] // mostly the newest, which is most likely still current
			if r.Intn(3) == 0 {
				pr = props[r.Intn(len(props))]
			}
			hash := pr.hash
			if r.Intn(4) == 0 {
				hash = "0000"
			}
			code, out := c.call(p.id, p.groups, "POST", "/v1/admin/proposals/"+pr.id+"/approve", map[string]any{"hash": hash})
			if code != 200 {
				continue
			}
			if p.id == pr.requester {
				return fmt.Errorf("%s approved their own proposal %s", p.id, pr.id)
			}
			if hash != pr.hash {
				return fmt.Errorf("proposal %s was approved with a hash that isn't its own", pr.id)
			}
			text, _ := out["manifest"].(string)
			next, ds := manifest.Check([]byte(text))
			if len(ds) > 0 {
				return fmt.Errorf("an approved manifest doesn't check: %v", ds)
			}
			if proposals.Hash(next) != pr.hash {
				return fmt.Errorf("the approved manifest isn't the one proposed")
			}
			if err := withinSpace(before, next, "finance"); err != nil {
				return fmt.Errorf("approved %s: %w", pr.id, err)
			}
			stewardApprovals++
			c.applied.t = next // applied
		default: // someone else changes the manifest (an operator's own diff)
			t := c.applied.t
			t.RetentionDays = 300 + r.Intn(60)
			c.applied.t, _ = manifest.Check(manifest.Format(t))
		}
	}
	return nil
}

// withinSpace: next differs from before only in space, which kept its template and stewards
// and didn't lower its label.
func withinSpace(before, next manifest.Tenant, space string) error {
	a, b := before, next
	a.Spaces, b.Spaces = nil, nil
	if !reflect.DeepEqual(a, b) {
		return fmt.Errorf("it changes more than space %s", space)
	}
	for k, v := range before.Spaces {
		if k != space && !reflect.DeepEqual(next.Spaces[k], v) {
			return fmt.Errorf("it changes space %s", k)
		}
	}
	o, n := before.Spaces[space], next.Spaces[space]
	rank := map[string]int{"public": 0, "internal": 1, "confidential": 2, "restricted": 3}
	switch {
	case n.Template != o.Template:
		return fmt.Errorf("it changes the template from %s to %s", o.Template, n.Template)
	case rank[n.Label] < rank[o.Label]:
		return fmt.Errorf("it lowers the label from %s to %s", o.Label, n.Label)
	case len(n.Stewards) == 0:
		return fmt.Errorf("it leaves no stewards")
	}
	return nil
}

// stewardApprovals counts approvals in one run, so the self-test can show the law test
// isn't vacuous: real proposals do get approved.
var stewardApprovals int
