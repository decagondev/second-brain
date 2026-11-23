package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// W16 Onboard and hand over (SB-1706, SB-1707) under the laws.
//
// LAW-01 and LAW-03 for start lists: an item is shown only if the source allows the member
// now; every other item is counted, never named. LAW-01 and LAW-12 for handover: a brief
// cites only the person's documents the asker may open, counts the rest, and is refused to
// anyone policy doesn't relate to the person.

func law01StartHere(r *rand.Rand, s searchSystem) error {
	w := Gen(r)
	ctx := context.Background()
	for _, u := range w.Users {
		var items []string
		for _, d := range w.Docs {
			if d.Tenant == u.Tenant && r.Intn(2) == 0 && len(items) < 10 {
				items = append(items, "docs:"+d.ID)
			}
		}
		a, err := s.agent(ctx, w, u.Tenant, lawOwners{})
		if err != nil {
			return err
		}
		scope, _ := scopeOf(u)
		res, err := a.StartHere(ctx, scope, items)
		if err != nil {
			return err
		}
		for _, it := range res.Items {
			if d := docOf(it.Cite); !w.truthAllow(u.ID, d) {
				return fmt.Errorf("%s was shown start item %s, which the source doesn't allow now (LAW-01, LAW-03)", u.ID, d)
			}
		}
		if len(res.Items)+res.Hidden != len(items) {
			return fmt.Errorf("%d shown and %d hidden of %d items", len(res.Items), res.Hidden, len(items))
		}
	}
	return nil
}

func law12Handover(r *rand.Rand, s searchSystem) error {
	w := Gen(r)
	o := genOwners(r, w)
	ctx := context.Background()
	k := policy.Kernel{Engine: askPolicies, Recorder: okRecorder{}}
	for _, u := range w.Users {
		a, err := s.agent(ctx, w, u.Tenant, o)
		if err != nil {
			return err
		}
		scope, _ := scopeOf(u)
		person := fmt.Sprintf("owner%d", r.Intn(3))
		email := person + "@acme-logistics.example"
		related := r.Intn(2) == 0
		rel := func(context.Context, tenancy.Scope, string) ([]string, error) {
			if related {
				return []string{[]string{"self", "manager", "steward"}[r.Intn(3)]}, nil
			}
			return []string{"colleague"}, nil
		}
		emails := func(context.Context, string) ([]string, error) { return []string{email}, nil }
		h, err := a.Handover(ctx, scope, person, emails, rel, k)
		if !related {
			if !errors.Is(err, ask.ErrNoPerson) {
				return fmt.Errorf("%s got a handover brief with no relation to the person (%v)", u.ID, err)
			}
			continue
		}
		if err != nil {
			return err
		}
		owned, allowed := 0, 0
		for _, d := range w.Docs {
			if d.Tenant == u.Tenant && o.of[d.ID] == email {
				owned++
				if w.truthAllow(u.ID, d.ID) {
					allowed++
				}
			}
		}
		for _, c := range h.Answer.Claims {
			if c.Uncited && len(h.Answer.Claims) > 1 {
				return fmt.Errorf("a handover brief kept an uncited line (LAW-14)")
			}
			for _, ci := range c.Cites {
				d := docOf(ci.Cite)
				if !w.truthAllow(u.ID, d) {
					return fmt.Errorf("%s's brief cites %s, which the source doesn't allow them (LAW-01)", u.ID, d)
				}
				if o.of[d] != email {
					return fmt.Errorf("%s's brief of %s cites %s, which isn't theirs", u.ID, person, d)
				}
			}
		}
		if h.Hidden != owned-allowed {
			return fmt.Errorf("brief hides %d of %d owned documents, want %d", h.Hidden, owned, owned-allowed)
		}
	}
	return nil
}

// law01HandoverPeople (SB-2106): the people search lists exactly the active candidates who
// match and whom policy lets the asker brief now; a colleague with no relation is never
// listed, whatever is typed. allowAll is a mutation that skips the policy decision.
func law01HandoverPeople(r *rand.Rand, s searchSystem, allowAll bool) error {
	w := Gen(r)
	ctx := context.Background()
	k := policy.Kernel{Engine: askPolicies, Recorder: okRecorder{}}
	if allowAll {
		k = policy.Kernel{Engine: allowEngine{}, Recorder: okRecorder{}}
	}
	for _, u := range w.Users {
		a, err := s.agent(ctx, w, u.Tenant, lawOwners{})
		if err != nil {
			return err
		}
		scope, _ := scopeOf(u)
		rels := map[string][]string{}
		var cands []ports.User
		want := map[string]bool{}
		for i := 0; i < 2+r.Intn(8); i++ {
			id := fmt.Sprintf("usr_p%d", i)
			name := []string{"dana", "dan", "ann", "bob"}[r.Intn(4)] + fmt.Sprint(i)
			active := r.Intn(4) > 0
			rel := []string{"self", "manager", "steward", "colleague", "colleague"}[r.Intn(5)]
			rels[id] = []string{rel}
			cands = append(cands, ports.User{ID: id, Tenant: u.Tenant, UserName: name, Emails: []string{name + "@acme.example"}, Active: active})
			if active && rel != "colleague" && strings.Contains(name, "dan") {
				want[id] = true
			}
		}
		got, err := a.HandoverPeople(ctx, scope, "DAN", func(context.Context, tenancy.Scope) ([]ports.User, error) { return cands, nil },
			func(_ context.Context, _ tenancy.Scope, p string) ([]string, error) { return rels[p], nil }, k)
		if err != nil {
			return err
		}
		for _, p := range got {
			if !want[p.ID] {
				return fmt.Errorf("%s's search lists %s (%v), whom they may not brief or who doesn't match (LAW-01)", u.ID, p.ID, rels[p.ID])
			}
			delete(want, p.ID)
		}
		if len(want) > 0 {
			return fmt.Errorf("%s's search left out people they may brief: %v", u.ID, want)
		}
	}
	return nil
}
