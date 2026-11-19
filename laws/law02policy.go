package laws

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"

	"example.com/second-brain/internal/adapters/policy/cedar"
	"example.com/second-brain/internal/core/policy"
)

// LAW-02 in the policies themselves (SB-110): the platform base policies plus a tenant's
// policies never allow anything across tenants, even when the kernel's own tenant guard
// is bypassed. Every permit must carry the tenant condition.

// LoadPolicies loads policy/platform/*.cedar and examples/policies/*.cedar.
func LoadPolicies() (policy.Evaluator, error) {
	_, here, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(here), "..")
	files := map[string][]byte{}
	for _, glob := range []string{"policy/platform/*.cedar", "examples/policies/*.cedar"} {
		ms, _ := filepath.Glob(filepath.Join(root, glob))
		for _, m := range ms {
			b, err := os.ReadFile(m)
			if err != nil {
				return nil, err
			}
			files[filepath.Base(m)] = b
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no policy files found under %s", root)
	}
	e, err := cedar.Load(files)
	return e, err
}

func law02Policies(r *rand.Rand, ev policy.Evaluator) error {
	tenants := []string{"acme-logistics", "beta-freight"}
	groups := []string{"grp_it_leads", "grp_contractors", "grp_finance", "grp_everyone"}
	type shape struct {
		action string
		res    func(id string) policy.Entity
	}
	shapes := []shape{
		{"read", func(id string) policy.Entity {
			return policy.Entity{Type: "Chunk", ID: id, Attrs: map[string]any{"cite": "docs:" + id + "@1"}}
		}},
		{"read", func(id string) policy.Entity {
			return policy.Entity{Type: "Document", ID: id, Attrs: map[string]any{"source": "docs"}}
		}},
		{"use_agent", func(string) policy.Entity {
			n := []string{"ask", "draft-reply"}[r.Intn(2)]
			return policy.Entity{Type: "Agent", ID: n, Attrs: map[string]any{"name": n, "consequential": false}}
		}},
		{"approve", func(id string) policy.Entity {
			return policy.Entity{Type: "ProposedAction", ID: id, Attrs: map[string]any{"workflow": "ticket-update",
				"requester": policy.Ref{Type: "User", ID: "usr_req"}}}
		}},
		{"admin", func(id string) policy.Entity { return policy.Entity{Type: "Cell", ID: id} }},
	}
	for i := 0; i < 20; i++ {
		pt, rt := tenants[r.Intn(2)], tenants[r.Intn(2)]
		var gs []string
		for _, g := range groups {
			if r.Intn(2) == 0 {
				gs = append(gs, g)
			}
		}
		sh := shapes[r.Intn(len(shapes))]
		res := sh.res(fmt.Sprintf("r%d", i))
		res.Tenant = rt
		req := policy.Request{Principal: policy.Principal{User: "usr_1", Tenant: pt, Groups: gs}, Action: sh.action, Resource: res,
			Context: map[string]any{}}
		got, err := ev.Evaluate(context.Background(), req)
		if err != nil {
			return err
		}
		if got.Decision == policy.Allow && pt != rt {
			return fmt.Errorf("policies allow %s (tenant %s) to %s %s %s of tenant %s: %s", "usr_1", pt, sh.action, res.Type, res.ID, rt, got.Reason)
		}
	}
	return nil
}
