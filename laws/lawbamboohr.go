package laws

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"example.com/second-brain/internal/adapters/connectors/bamboohr"
	"example.com/second-brain/internal/adapters/connectors/bamboohr/bamboohrtest"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/obs"
)

// The BambooHR connector under the laws (SB-1104). LAW-03: a directory BambooHR can't
// serve makes every verdict Unknown. LAW-07 and LAW-20: through the real pipeline and a
// restricted People space, a sensitive field is read from BambooHR, and indexed, only if
// the space allow-lists it, and then the record is restricted.

func newBamboo() (*bamboohrtest.Fake, *bamboohr.Connector) {
	f := bamboohrtest.New()
	c := &bamboohr.Connector{Base: f.Base(), HTTP: f.Server.Client(), Redact: obs.NewRedactor(),
		Creds: func(context.Context) (string, error) { return f.CredsJSON(), nil }}
	return f, c
}

// NewBambooWorld is the connector-level world: access means the file is shared with all
// employees; "script" makes the first ref an id BambooHR can't place.
var NewBambooWorld = func() ConnectorWorld {
	f, c := newBamboo()
	f.PutEmployee("1", map[string]string{"displayName": "Ann", "workEmail": "ann@acme.example"}, true)
	return &bambooWorld{f: f, c: c}
}

type bambooWorld struct {
	f *bamboohrtest.Fake
	c *bamboohr.Connector
	n int
}

func (w *bambooWorld) ACL(access []bool, fault string) ([]string, error) {
	w.n++
	refs := make([]ports.SourceRef, len(access))
	for i, ok := range access {
		id := fmt.Sprintf("f%d_%d", w.n, i)
		w.f.PutFile(id, "Policy", "Policies", "text", ok)
		refs[i] = ports.SourceRef{Kind: "hris", ID: "files/" + id}
	}
	switch fault {
	case "script":
		refs[0] = ports.SourceRef{Kind: "hris", ID: "payroll/" + refs[0].ID} // an id BambooHR can't place
	case "":
	default:
		w.f.FailNext(fault)
	}
	vs, err := w.c.ACL(context.Background(), ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}, refs)
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.String()
	}
	return out, err
}

func (w *bambooWorld) Act(key string, _ bool) (string, error) {
	r, err := w.c.Act(context.Background(), ports.Action{Target: "hris:x"}, ports.IdempotencyKey(key))
	return r.TargetID, err
}

func (w *bambooWorld) Effects() int { return 0 }

// AllowHook gives the connector the sensitive fields to read, from the spaces' allow-lists
// (the identity in the real system).
type AllowHook func(spaceAllowed []string) func() []string

func allowFromSpaces(a []string) func() []string { return func() []string { return a } }

func law07BambooHR(r *rand.Rand, hook AllowHook) error {
	ctx := context.Background()
	f, c := newBamboo()
	sensitive := []string{"payRate", "ssn", "dateOfBirth", "homePhone"}
	var allowed []string
	for _, s := range sensitive {
		if r.Intn(3) == 0 {
			allowed = append(allowed, s)
		}
	}
	c.Allowed = hook(allowed)
	all := []ports.Space{{ID: spaces.ID("people"), Name: "people", Label: "restricted",
		Scope: map[string][]string{"hris": {"employees/**"}}, Fields: map[string][]string{"hris": allowed}}}
	idx := memindex.New()
	p := ingest.Pipeline{Tenant: "acme-logistics", Release: "rel_test", Connector: c, Embedder: hashing.Embedder{}, Index: idx,
		Spaces: func(context.Context) ([]ports.Space, error) { return all, nil }}
	for i := 0; i < 2+r.Intn(4); i++ {
		fields := map[string]string{"displayName": fmt.Sprintf("Emp %d", i), "department": "Ops",
			"workEmail": fmt.Sprintf("e%d@acme.example", i)}
		for _, s := range sensitive {
			fields[s] = fmt.Sprintf("SENS-%s-%d-%x", s, i, r.Int63())
		}
		f.PutEmployee(fmt.Sprint(i), fields, true)
	}
	if _, err := p.Sync(ctx, ""); err != nil {
		return err
	}
	for _, s := range sensitive {
		if f.Asked[s] > 0 && !contains(allowed, s) {
			return fmt.Errorf("BambooHR was asked for %s, which no space allow-lists (allowed %v)", s, allowed)
		}
	}
	chunks, err := idx.All(ctx, "acme-logistics")
	if err != nil {
		return err
	}
	for _, ch := range chunks {
		for _, s := range sensitive {
			if strings.Contains(ch.Text, "SENS-"+s+"-") && !contains(allowed, s) {
				return fmt.Errorf("%s holds %s, which no space allow-lists", ch.SourceID, s)
			}
		}
		if len(allowed) > 0 && ch.Label != "restricted" {
			return fmt.Errorf("%s holds allow-listed sensitive fields and is labelled %q", ch.SourceID, ch.Label)
		}
	}
	return nil
}

// law01BambooHR: the asker's access follows BambooHR now: someone made inactive, or a file
// unshared, is refused at the next check, never on the strength of an earlier answer.
func law01BambooHR(r *rand.Rand) error {
	f, c := newBamboo()
	f.PutFile("f1", "Handbook", "Policies", "Holidays are 25 days.", true)
	ref := []ports.SourceRef{{Kind: "hris", ID: "files/f1"}}
	active, shared := true, true
	f.PutEmployee("1", map[string]string{"displayName": "Ann", "workEmail": "ann@acme.example"}, active)
	for i := 0; i < 6; i++ {
		switch r.Intn(3) {
		case 0:
			active = !active
			f.PutEmployee("1", map[string]string{"displayName": "Ann", "workEmail": "ann@acme.example"}, active)
		case 1:
			if shared {
				shared = false
				f.Unshare("f1")
			}
		}
		vs, err := c.ACL(context.Background(), ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}, ref)
		if err != nil {
			return err
		}
		want := map[bool]ports.Verdict{true: ports.VerdictAllow, false: ports.VerdictDeny}[active && shared]
		if vs[0] != want {
			return fmt.Errorf("active %v, shared %v: the check said %v", active, shared, vs[0])
		}
	}
	return nil
}
