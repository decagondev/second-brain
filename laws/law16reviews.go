package laws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"time"

	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	"example.com/second-brain/internal/api"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/groupsync"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/proposals"
	"example.com/second-brain/internal/core/reviews"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-16 for quarterly access reviews (SB-1202), through the real handler and policies: a
// review lists exactly the people who can use each restricted space; deciding changes
// nothing live, and a complete review's removals all become change requests (a manifest
// proposal an operator must still approve, or an IdP request); nobody decides on their own
// access; a review is overdue exactly when open 14 days; and the evidence verifies and
// points at each decision's audit row.

const reviewBase = `tenant: delta-tools
channel: stable
idp: identity-platform://sb-idp-delta/delta-tools-x7k2
region: eu
connectors: { docs: sharepoint }
agents: [ask]
groups:
  group:finance: [ana@delta-tools.example, bo@delta-tools.example, cy@delta-tools.example]
  group:people: [di@delta-tools.example, ed@delta-tools.example]
  group:finance-leads: [lea@delta-tools.example, ana@delta-tools.example]
  group:hr-leads: [hal@delta-tools.example]
spaces:
  finance:
    template: finance
    members: [group:finance, group:execs]
    stewards: [group:finance-leads]
    scope: { docs: ["/Finance/**"] }
  people:
    template: people
    members: [group:people, group:execs]
    stewards: [group:hr-leads]
    scope: { docs: ["/People/**"] }
  eng:
    template: engineering
    members: [group:eng]
    stewards: [group:eng-leads]
    scope: { docs: ["/Engineering/**"] }
`

// ReviewSystem is the review rules under test (package reviews in the real system).
type ReviewSystem struct {
	Open     func(ctx context.Context, t manifest.Tenant, now time.Time, roster reviews.Roster, last reviews.LastUse, have func(string, string) bool, newID func(string) string) ([]reviews.Review, error)
	Requests func(t manifest.Tenant, r reviews.Review) (manifest.Tenant, bool, []reviews.Request)
	Overdue  func(r reviews.Review, now time.Time) bool
}

// RealReviews is the shipped rules.
var RealReviews = ReviewSystem{Open: reviews.OpenAll, Requests: reviews.Requests, Overdue: reviews.Overdue}

type mapRoster struct {
	m    map[string][]string
	fail string
}

func (m mapRoster) Members(_ context.Context, g string) ([]string, error) {
	if g == m.fail {
		return nil, errors.New("directory unavailable")
	}
	return append([]string(nil), m.m[g]...), nil
}

func law16Reviews(r *rand.Rand, sys ReviewSystem) error {
	ctx := context.Background()
	base, ds := manifest.Check([]byte(reviewBase))
	if len(ds) > 0 {
		return fmt.Errorf("base manifest: %v", ds)
	}
	now := time.Date(2027, 1, 1, 5, 0, 0, 0, time.UTC)
	uid := groupsync.UserID
	// The directory: manifest groups as groupsync makes them, plus IdP groups.
	roster := mapRoster{m: map[string][]string{}}
	for g, es := range base.Groups {
		for _, e := range es {
			roster.m[g] = append(roster.m[g], uid(e))
		}
	}
	execs := []string{uid("ana@delta-tools.example"), "usr_exec1", "usr_exec2"}[:1+r.Intn(3)]
	roster.m["group:execs"] = execs
	roster.m["group:eng"] = []string{"usr_eng1"}
	last := map[string]time.Time{}
	lastUse := func(space, user string) (time.Time, bool) { t, ok := last[space+"/"+user]; return t, ok }
	for _, u := range []string{uid("ana@delta-tools.example"), uid("di@delta-tools.example"), "usr_exec1"} {
		if r.Intn(2) == 0 {
			last[[]string{"finance", "people"}[r.Intn(2)]+"/"+u] = now.Add(-time.Duration(1+r.Intn(2000)) * time.Hour)
		}
	}
	n := 0
	newID := func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }

	// A group that can't be read opens nothing.
	if r.Intn(4) == 0 {
		bad := roster
		bad.fail = []string{"group:finance", "group:execs", "group:people"}[r.Intn(3)]
		if got, err := sys.Open(ctx, base, now, bad, lastUse, nil, newID); err == nil || len(got) > 0 {
			return fmt.Errorf("an unreadable group %s still opened %d reviews", bad.fail, len(got))
		}
	}
	opened, err := sys.Open(ctx, base, now, roster, lastUse, nil, newID)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	// Exactly the restricted spaces, exactly their people.
	var spaces []string
	for _, rv := range opened {
		spaces = append(spaces, rv.Space)
	}
	sort.Strings(spaces)
	if !reflect.DeepEqual(spaces, []string{"finance", "people"}) {
		return fmt.Errorf("reviews opened for %v, want the restricted spaces [finance people]", spaces)
	}
	for _, rv := range opened {
		want := map[string][]string{}
		for _, g := range base.Spaces[rv.Space].Members {
			for _, u := range roster.m[g] {
				want[u] = append(want[u], g)
			}
		}
		got := map[string][]string{}
		for _, it := range rv.Items {
			if _, twice := got[it.User]; twice {
				return fmt.Errorf("review of %s lists %s twice", rv.Space, it.User)
			}
			got[it.User] = it.Via
			wl := reviews.Never
			if t, ok := lastUse(rv.Space, it.User); ok {
				wl = t.Format(time.RFC3339)
			}
			if it.LastUse != wl {
				return fmt.Errorf("review of %s: %s last used %s, listed as %s", rv.Space, it.User, wl, it.LastUse)
			}
		}
		if !reflect.DeepEqual(got, want) {
			return fmt.Errorf("review of %s lists %v, but these can use it: %v", rv.Space, got, want)
		}
	}

	// Decisions, through the handler.
	applied := &appliedManifest{t: base}
	alog := &audit.Log{Tenant: "delta-tools", Release: func() string { return "rel_law" }, Store: auditmem.New(), Now: time.Now}
	store, props := &reviews.MemoryStore{}, &proposals.MemoryStore{}
	for _, rv := range opened {
		_ = store.Put(ctx, rv)
	}
	ev, err := LoadPolicies()
	if err != nil {
		return err
	}
	h := api.Reviews{Tenant: "delta-tools", Policy: policy.Kernel{Engine: ev, Recorder: okRecorder{}}, Manifests: applied, Audit: alog,
		Reviews: store, Proposals: props, NewID: newID, Now: func() time.Time { return now }, Requests: sys.Requests}
	mux := http.NewServeMux()
	h.Register(mux)
	call := func(user string, groups []string, method, path string, body any) (int, []byte) {
		s, _ := tenancy.NewIssuer("delta-tools").Issue("delta-tools", user, groups)
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(method, path, bytes.NewReader(b)).WithContext(tenancy.WithScope(ctx, s))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w.Code, w.Body.Bytes()
	}
	reviewers := map[string][]person{
		"finance": {{uid("lea@delta-tools.example"), []string{"finance-leads"}}, {uid("ana@delta-tools.example"), []string{"finance-leads", "finance"}}},
		"people":  {{uid("hal@delta-tools.example"), []string{"hr-leads"}}},
	}
	outsider := person{"usr_out", []string{"finance", "people"}}
	baseHash := proposals.Hash(base)
	for _, rv := range opened {
		for step := 0; step < 3*len(rv.Items)+4; step++ {
			cur, _, _ := store.Get(ctx, rv.ID)
			if cur.State != reviews.Open {
				break
			}
			it := cur.Items[r.Intn(len(cur.Items))]
			who := reviewers[rv.Space][r.Intn(len(reviewers[rv.Space]))]
			if r.Intn(6) == 0 {
				who = outsider
			}
			dec, reason := reviews.Certify, ""
			if r.Intn(2) == 0 {
				dec, reason = reviews.Remove, "left the team"
			}
			code, _ := call(who.id, who.groups, "POST", "/v1/spaces/"+rv.Space+"/reviews/"+rv.ID+"/decide",
				map[string]string{"user": it.User, "decision": dec, "reason": reason})
			if code == 200 {
				switch {
				case who.id == outsider.id:
					return fmt.Errorf("someone who isn't a steward of %s decided on %s", rv.Space, it.User)
				case who.id == it.User:
					return fmt.Errorf("%s decided on their own access to %s", who.id, rv.Space)
				case it.Decision != "":
					return fmt.Errorf("%s's decision in %s was changed after it was made", it.User, rv.Space)
				}
			}
			if proposals.Hash(applied.t) != baseHash {
				return errors.New("a review decision changed the applied manifest")
			}
		}
		// Anyone left undecided is certified by a steward who isn't them, so the review completes.
		cur, _, _ := store.Get(ctx, rv.ID)
		for _, it := range cur.Items {
			if it.Decision != "" {
				continue
			}
			for _, who := range reviewers[rv.Space] {
				if who.id != it.User {
					call(who.id, who.groups, "POST", "/v1/spaces/"+rv.Space+"/reviews/"+rv.ID+"/decide",
						map[string]string{"user": it.User, "decision": reviews.Certify})
					break
				}
			}
		}
		done, _, _ := store.Get(ctx, rv.ID)
		if done.State != reviews.Complete {
			return fmt.Errorf("review of %s isn't complete after every person was decided", rv.Space)
		}
		if err := checkRemovals(ctx, base, done, props); err != nil {
			return fmt.Errorf("review of %s: %w", rv.Space, err)
		}
		// The evidence.
		code, doc := call("usr_op", []string{"sb-operators"}, "GET", "/v1/admin/reviews/"+rv.ID+"/evidence", nil)
		if code != 200 {
			return fmt.Errorf("evidence for %s: HTTP %d", rv.ID, code)
		}
		ev, err := reviews.VerifyEvidence(doc)
		if err != nil {
			return fmt.Errorf("evidence for %s: %w", rv.ID, err)
		}
		for _, it := range ev.Items {
			rows, err := alog.Store.Range(ctx, it.AuditSeq, 1)
			if it.AuditSeq == 0 || err != nil || len(rows) != 1 || rows[0].Seq != it.AuditSeq || rows[0].Kind != "access" ||
				rows[0].Subject != rv.ID+":"+it.User || rows[0].State != it.Decision || rows[0].Space != "spc_"+rv.Space || rows[0].Actor != it.By {
				return fmt.Errorf("evidence for %s: %s's decision doesn't point at its audit row (seq %d)", rv.ID, it.User, it.AuditSeq)
			}
		}
	}

	// Overdue exactly when open 14 days.
	for i := 0; i < 8; i++ {
		rv := reviews.Clone(opened[r.Intn(len(opened))])
		if r.Intn(3) == 0 {
			rv.State = reviews.Complete
		}
		at := now.Add(time.Duration(r.Intn(30*24)) * time.Hour)
		if r.Intn(3) == 0 {
			at = now.Add(reviews.Window - time.Duration(r.Intn(2))*time.Second)
		}
		want := rv.State == reviews.Open && !at.Before(now.Add(14*24*time.Hour))
		if got := sys.Overdue(rv, at); got != want {
			return fmt.Errorf("review opened %s, state %s: overdue at %s is %v, want %v", rv.Opened, rv.State, at.Format(time.RFC3339), got, want)
		}
	}
	return nil
}

// checkRemovals: every removal is requested for every group that gives the access; manifest
// groups go to one open proposal taking exactly those people out; nothing is approved yet.
func checkRemovals(ctx context.Context, base manifest.Tenant, rv reviews.Review, props *proposals.MemoryStore) error {
	want := map[string]bool{}
	expect := map[string][]string{}
	for g, es := range base.Groups {
		expect[g] = append([]string(nil), es...)
	}
	listed := false
	for _, it := range rv.Items {
		if it.Decision != reviews.Remove {
			continue
		}
		for _, g := range it.Via {
			kind := "idp"
			if _, ok := base.Groups[g]; ok {
				kind, listed = "manifest", true
				var kept []string
				for _, e := range expect[g] {
					if groupsync.UserID(e) != it.User {
						kept = append(kept, e)
					}
				}
				expect[g] = kept
			}
			want[kind+" "+it.User+" "+g] = true
		}
	}
	got := map[string]bool{}
	pid := ""
	for _, q := range rv.Requests {
		got[q.Kind+" "+q.User+" "+q.Group] = true
		if q.Kind == "manifest" {
			pid = q.Proposal
		}
	}
	if !reflect.DeepEqual(got, want) {
		return fmt.Errorf("removals requested %v, want %v", sortedSet(got), sortedSet(want))
	}
	if !listed {
		return nil
	}
	p, found, _ := props.Get(ctx, pid)
	if !found {
		return fmt.Errorf("manifest removals have no proposal (%q)", pid)
	}
	if p.State != proposals.Open {
		return fmt.Errorf("the removals' proposal is %s, not waiting for an operator", p.State)
	}
	next, ds := manifest.Check([]byte(p.Manifest))
	if len(ds) > 0 {
		return fmt.Errorf("the removals' proposal doesn't check: %v", ds)
	}
	wantT := base
	wantT.Groups = map[string][]string{}
	for g, es := range expect {
		wantT.Groups[g] = es
	}
	if w, _ := manifest.Check(manifest.Format(wantT)); proposals.Hash(w) != proposals.Hash(next) || p.Base != proposals.Hash(base) {
		return fmt.Errorf("the removals' proposal isn't exactly the removals:\n%s", p.Manifest)
	}
	reviewProposals++
	return nil
}

// reviewProposals counts removal proposals checked in one run, so the law test can show it
// isn't vacuous.
var reviewProposals int

func sortedSet(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
