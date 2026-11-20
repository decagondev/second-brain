package reviews

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/manifest"
)

type roster map[string][]string

func (r roster) Members(_ context.Context, g string) ([]string, error) {
	if g == "group:broken" {
		return nil, errors.New("down")
	}
	return r[g], nil
}

func tenant() manifest.Tenant {
	return manifest.Tenant{Spaces: map[string]manifest.Space{
		"finance": {Label: "restricted", Members: []string{"group:finance", "group:execs"}, Stewards: []string{"group:leads"}},
		"eng":     {Label: "internal", Members: []string{"group:eng"}},
	}}
}

func TestQuarter(t *testing.T) {
	for in, want := range map[string]string{"2027-01-01": "2027-Q1", "2027-03-31": "2027-Q1", "2027-04-01": "2027-Q2", "2027-12-31": "2027-Q4"} {
		d, _ := time.Parse("2006-01-02", in)
		if got := Quarter(d); got != want {
			t.Errorf("Quarter(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestRunOpensRestrictedOnceAndDecides(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2027, 4, 1, 5, 0, 0, 0, time.UTC)
	ro := roster{"group:finance": {"usr_a", "usr_b"}, "group:execs": {"usr_a", "usr_c"}}
	last := FromAudit([]audit.Row{{Actor: "usr_a", Space: "spc_finance", At: "2027-03-02T10:00:00Z"}, {Actor: "usr_a", Space: "spc_finance", At: "2027-03-01T10:00:00Z"}})
	st := &MemoryStore{}
	n := 0
	id := func(p string) string { n++; return p + string(rune('0'+n)) }
	got, err := Run(ctx, tenant(), now, ro, last, st, id)
	if err != nil || len(got) != 1 || got[0].Space != "finance" || got[0].Quarter != "2027-Q2" {
		t.Fatalf("Run = %+v, %v", got, err)
	}
	r := got[0]
	if len(r.Items) != 3 || r.Items[0].User != "usr_a" || len(r.Items[0].Via) != 2 || r.Items[0].LastUse != "2027-03-02T10:00:00Z" || r.Items[1].LastUse != Never {
		t.Fatalf("items = %+v", r.Items)
	}
	if again, _ := Run(ctx, tenant(), now.Add(time.Hour), ro, last, st, id); len(again) != 0 {
		t.Fatal("a second run in the quarter opened another review")
	}
	if _, err := Decide(r, "usr_a", Certify, "", "usr_a", now); !errors.Is(err, ErrSelf) {
		t.Fatalf("self-decision: %v", err)
	}
	if _, err := Decide(r, "usr_b", Remove, " ", "usr_x", now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("removal without a reason: %v", err)
	}
	r2, err := Decide(r, "usr_b", Remove, "left", "usr_x", now)
	if err != nil || r.Items[1].Decision != "" {
		t.Fatalf("Decide changed its input or failed: %v", err)
	}
	if _, err := Decide(r2, "usr_b", Certify, "", "usr_x", now); !errors.Is(err, ErrDecided) {
		t.Fatalf("re-decision: %v", err)
	}
	r2, _ = Decide(r2, "usr_a", Certify, "", "usr_x", now)
	if r2.State != Open || !Overdue(r2, now.Add(Window)) || Overdue(r2, now.Add(Window-time.Second)) {
		t.Fatal("overdue bound is wrong")
	}
	r2, _ = Decide(r2, "usr_c", Certify, "", "usr_x", now)
	if r2.State != Complete || Overdue(r2, now.Add(100*Window)) {
		t.Fatal("a fully decided review isn't complete")
	}
	if _, err := Decide(r2, "usr_c", Certify, "", "usr_x", now); !errors.Is(err, ErrClosed) {
		t.Fatalf("decision on a complete review: %v", err)
	}
}

func TestOpenRefusesUnreadableGroup(t *testing.T) {
	tn := tenant()
	sp := tn.Spaces["finance"]
	sp.Members = append(sp.Members, "group:broken")
	tn.Spaces["finance"] = sp
	if got, err := OpenAll(context.Background(), tn, time.Now(), roster{}, FromAudit(nil), nil, func(string) string { return "x" }); err == nil || got != nil {
		t.Fatal("an unreadable group opened a review")
	}
}

func TestEvidenceVerifies(t *testing.T) {
	r := Review{ID: "rev_1", Space: "finance", State: Complete, Items: []Item{{User: "usr_a", Via: []string{"group:x"}, Decision: Certify, AuditSeq: 7}}, Requests: []Request{}}
	doc, err := Evidence(r)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := VerifyEvidence(doc); err != nil || got.Items[0].AuditSeq != 7 {
		t.Fatalf("VerifyEvidence = %+v, %v", got, err)
	}
	tampered := []byte(strings.Replace(string(doc), `"certify"`, `"remove"`, 1))
	if _, err := VerifyEvidence(tampered); err == nil {
		t.Fatal("tampered evidence verified")
	}
}

type gauge map[string]float64

func (g gauge) Set(_, _ string, l map[string]string, v float64) { g[l["space"]] = v }

func TestWatch(t *testing.T) {
	now := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	rs := []Review{{ID: "rev_1", Space: "finance", State: Open, Opened: now.Format(time.RFC3339)},
		{ID: "rev_2", Space: "people", State: Complete, Opened: now.Format(time.RFC3339)}}
	g := gauge{}
	if a := Watch(rs, now.Add(15*24*time.Hour), g); len(a) != 1 || a[0].Space != "finance" || a[0].Severity != "sev2" || g["finance"] != 1 || g["people"] != 0 {
		t.Fatalf("Watch = %+v, gauge %v", a, g)
	}
}

func TestRequestsDoesNotTouchInput(t *testing.T) {
	tn := tenant()
	tn.Groups = map[string][]string{"group:finance": {"ana@acme.example", "bo@acme.example"}}
	r := Review{Items: []Item{{User: "usr_x", Via: []string{"group:execs"}, Decision: Remove}}}
	next, changed, reqs := Requests(tn, r)
	if changed || len(reqs) != 1 || reqs[0].Kind != "idp" || len(next.Groups["group:finance"]) != 2 || len(tn.Groups["group:finance"]) != 2 {
		t.Fatalf("Requests = %v %v", changed, reqs)
	}
}
