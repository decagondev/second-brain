package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// W16 Find (SB-1701 to SB-1703) under the retrieval laws.
//
// LAW-01 and LAW-03 for search: the same world as an answer, through ask.Agent.Search. A
// document is returned only if the source allows it now and was asked; Unknown, a timeout
// or a missing document is never returned.
//
// LAW-01 and LAW-12 for owners: an answer and a search name an owner only on a document
// the asker was allowed, and only an owner who is one active directory user.
//
// LAW-02 and LAW-07 for shares: a share from another tenant is never opened, and the audit
// row of a share holds the question's hash, never its text.

// searchSystem is the system the search laws run: the real agent over the world. Its
// mutations are for the self-tests.
type searchSystem struct {
	unknownAsAllow bool // the live check treats Unknown as Allow
	anyOwner       bool // the owner lookup ignores whether the user is active
}

func scopeOf(u User) (tenancy.Scope, error) {
	return tenancy.NewIssuer(u.Tenant).Issue(u.Tenant, u.ID, u.Groups)
}

// lawOwners gives each document an owner and each owner an active flag.
type lawOwners struct {
	of     map[string]string // doc id -> owner email
	active map[string]bool
}

func genOwners(r *rand.Rand, w *World) lawOwners {
	o := lawOwners{of: map[string]string{}, active: map[string]bool{}}
	for i := 0; i < 3; i++ {
		e := fmt.Sprintf("owner%d@acme-logistics.example", i)
		o.active[e] = r.Intn(2) == 0
	}
	for _, d := range w.Docs {
		if r.Intn(4) > 0 {
			o.of[d.ID] = fmt.Sprintf("owner%d@acme-logistics.example", r.Intn(3))
		}
	}
	return o
}

func (s searchSystem) agent(ctx context.Context, w *World, tenant string, o lawOwners) (ask.Agent, error) {
	a, err := askSystem{w: w, unknownAsAllow: s.unknownAsAllow}.agent(ctx, w, tenant)
	if err != nil {
		return a, err
	}
	all, err := a.Index.(ports.Index).All(ctx, tenant)
	if err != nil {
		return a, err
	}
	for _, c := range all {
		c.Owner = o.of[c.SourceID]
		c.Title = "Title of " + c.SourceID
		ref := ports.SourceRef{Kind: c.Kind, ID: c.SourceID}
		if err := a.Index.(ports.Index).Replace(ctx, tenant, ref, c.Version, []ports.Chunk{c}); err != nil {
			return a, err
		}
	}
	a.Owner = func(_ context.Context, email string) (answer.Person, bool) {
		if !o.active[email] && !s.anyOwner {
			return answer.Person{}, false
		}
		return answer.Person{Email: email, Title: "Lead"}, true
	}
	return a, nil
}

// law01Search: every document a search returns is one the source allows the asker now.
func law01Search(r *rand.Rand, s searchSystem) error {
	w := Gen(r)
	o := genOwners(r, w)
	ctx := context.Background()
	for _, u := range w.Users {
		scope, err := scopeOf(u)
		if err != nil {
			return err
		}
		a, err := s.agent(ctx, w, u.Tenant, o)
		if err != nil {
			return err
		}
		res, err := a.Search(ctx, scope, ask.SearchRequest{Query: "anything"})
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, x := range res.Results {
			d := docOf(x.Cite)
			if seen[d] {
				return fmt.Errorf("%s was shown %s twice", u.ID, d)
			}
			seen[d] = true
			if !w.truthAllow(u.ID, d) {
				return fmt.Errorf("%s was shown %s, which the source does not allow now (LAW-01, LAW-03)", u.ID, d)
			}
			if !w.asked[u.ID+"|"+d] {
				return fmt.Errorf("%s was shown %s without asking the source", u.ID, d)
			}
			if x.Owner != nil && (!o.active[x.Owner.Email] || o.of[d] != x.Owner.Email) {
				return fmt.Errorf("%s was shown owner %s of %s, who isn't its one active owner (LAW-12)", u.ID, x.Owner.Email, d)
			}
		}
		tr, err := a.Trace(ctx, scope, res.Trace)
		if err != nil {
			return fmt.Errorf("a search left no trace: %v", err)
		}
		if tr.QuerySHA256 == "" || strings.Contains(fmt.Sprint(tr), "anything") {
			return fmt.Errorf("a search trace must keep the query's hash, never its text (LAW-07)")
		}
	}
	return nil
}

// law12Owners: an answer's owners and people are active owners of documents it cites,
// and every cited document is one the asker may open.
func law12Owners(r *rand.Rand, s searchSystem) error {
	w := Gen(r)
	o := genOwners(r, w)
	ctx := context.Background()
	for _, u := range w.Users {
		scope, err := scopeOf(u)
		if err != nil {
			return err
		}
		a, err := s.agent(ctx, w, u.Tenant, o)
		if err != nil {
			return err
		}
		resp, err := a.Ask(ctx, scope, ask.Request{Question: "anything"})
		if err != nil {
			return err
		}
		owned := map[string]bool{}
		for _, c := range resp.Answer.Claims {
			for _, ci := range c.Cites {
				d := docOf(ci.Cite)
				if !w.truthAllow(u.ID, d) {
					return fmt.Errorf("%s was cited %s, which the source does not allow now", u.ID, d)
				}
				if ci.Owner == nil {
					continue
				}
				if !o.active[ci.Owner.Email] || o.of[d] != ci.Owner.Email {
					return fmt.Errorf("%s was told to ask %s about %s, who isn't its one active owner (LAW-12)", u.ID, ci.Owner.Email, d)
				}
				owned[ci.Owner.Email] = true
			}
		}
		if len(resp.People) > ask.MaxPeople {
			return fmt.Errorf("%d people suggested, more than %d", len(resp.People), ask.MaxPeople)
		}
		for _, p := range resp.People {
			if !owned[p.Email] {
				return fmt.Errorf("%s was told to ask %s, who owns nothing the answer cites (LAW-01)", u.ID, p.Email)
			}
		}
	}
	return nil
}

// shareSystem is the share service each tenant's cell runs. sharedStore puts every tenant
// in one store and drops the tenant from lookups (a self-test mutation).
type shareSystem struct {
	sharedStore bool
}

type keepRows struct{ rows []audit.Event }

func (k *keepRows) Write(_ context.Context, e audit.Event) (audit.Row, error) {
	k.rows = append(k.rows, e)
	return audit.Row{}, nil
}

type tenantBlindStore struct{ *tracemem.Store }

// GetShare finds a share by id alone and files it under whichever tenant asked: a store
// not keyed by tenant.
func (t tenantBlindStore) GetShare(ctx context.Context, tenant string, id string) (ask.Share, bool, error) {
	for _, tn := range []string{"acme-logistics", "beta-freight", "gamma-parts"} {
		if sh, ok, err := t.Store.GetShare(ctx, tn, id); ok || err != nil {
			sh.Tenant = tenant
			return sh, ok, err
		}
	}
	return ask.Share{}, false, nil
}

func law02Shares(r *rand.Rand, s shareSystem) error {
	w := Gen(r)
	ctx := context.Background()
	now := time.Unix(1_800_000_000, 0)
	shared := tracemem.New()
	rows := map[string]*keepRows{}
	svc := map[string]ask.Questions{}
	n := 0
	for _, u := range w.Users {
		if _, ok := svc[u.Tenant]; ok {
			continue
		}
		var st ask.QuestionStore = tracemem.New()
		if s.sharedStore {
			st = tenantBlindStore{shared}
		}
		rows[u.Tenant] = &keepRows{}
		svc[u.Tenant] = ask.Questions{Tenant: u.Tenant, Store: st, Audit: rows[u.Tenant], Now: func() time.Time { return now },
			NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }}
	}
	type made struct {
		tenant, id, text string
	}
	var shares []made
	for i, u := range w.Users {
		scope, err := scopeOf(u)
		if err != nil {
			return err
		}
		text := fmt.Sprintf("where is the refund policy %d %d", i, r.Intn(1000))
		sh, err := svc[u.Tenant].Share(ctx, scope, text)
		if err != nil {
			return err
		}
		shares = append(shares, made{u.Tenant, sh.ID, text})
	}
	for _, u := range w.Users {
		scope, err := scopeOf(u)
		if err != nil {
			return err
		}
		for _, m := range shares {
			sh, err := svc[u.Tenant].Open(ctx, scope, m.id)
			switch {
			case m.tenant != u.Tenant && err == nil:
				return fmt.Errorf("%s (tenant %s) opened a share of tenant %s (LAW-02)", u.ID, u.Tenant, m.tenant)
			case m.tenant != u.Tenant && !errors.Is(err, ask.ErrNoShare):
				return fmt.Errorf("another tenant's share gave %v, not EPERM", err)
			case m.tenant == u.Tenant && err != nil:
				return fmt.Errorf("%s couldn't open a share of their own tenant: %v", u.ID, err)
			case m.tenant == u.Tenant && sh.Question != m.text:
				return fmt.Errorf("a share opened with another question")
			}
		}
	}
	for t, k := range rows {
		for _, e := range k.rows {
			for _, m := range shares {
				if strings.Contains(fmt.Sprint(e), m.text) {
					return fmt.Errorf("tenant %s's audit row holds a shared question's text (LAW-07)", t)
				}
			}
			if e.PayloadSHA256 == "" {
				return fmt.Errorf("a share's audit row has no question hash")
			}
		}
	}
	// An expired share is never opened.
	later := svc[w.Users[0].Tenant]
	later.Now = func() time.Time { return now.Add(ask.ShareTTL) }
	scope, err := scopeOf(w.Users[0])
	if err != nil {
		return err
	}
	if _, err := later.Open(ctx, scope, shares[0].id); !errors.Is(err, ask.ErrNoShare) {
		return fmt.Errorf("an expired share was opened (%v)", err)
	}
	return nil
}
