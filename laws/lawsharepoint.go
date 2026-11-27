package laws

import (
	"context"
	"fmt"
	"math/rand"

	"example.com/second-brain/internal/adapters/msgraph"
	"example.com/second-brain/internal/adapters/msgraph/graphtest"
	"example.com/second-brain/internal/core/ports"
)

// The SharePoint and OneDrive connector under the connector laws (SB-5901, SB-5903): LAW-03
// (anything Graph can't answer is Unknown) and LAW-01 (the live permissions and the
// person's Entra groups now decide, never the copy taken at fetch).

func newSharePoint() (*graphtest.Fake, *msgraph.SharePoint) {
	f := graphtest.New()
	g := &msgraph.Client{Base: f.Base(), Login: f.Login(), HTTP: f.Server.Client(), Creds: func(context.Context) (string, error) { return graphtest.Creds, nil }}
	return f, &msgraph.SharePoint{Graph: g, Drives: []string{"d1"}, Domain: "acme.example", Dir: &msgraph.Directory{Graph: g}}
}

// NewSharePointWorld is the connector-level world: access means the item is shared with ann.
var NewSharePointWorld = func() ConnectorWorld {
	f, sp := newSharePoint()
	return &sharePointWorld{f: f, sp: sp}
}

type sharePointWorld struct {
	f  *graphtest.Fake
	sp *msgraph.SharePoint
	n  int
}

func (w *sharePointWorld) ACL(access []bool, fault string) ([]string, error) {
	w.n++
	names := make([]string, len(access))
	for i, ok := range access {
		names[i] = fmt.Sprintf("r%d-%d.md", w.n, i)
		it := graphtest.Item{ID: names[i], Name: names[i], Content: []byte("x"), Users: []string{"bob@acme.example"}}
		if ok {
			it.Users = []string{"ann@acme.example"}
		}
		w.f.Put("d1", it)
	}
	refs := make([]ports.SourceRef, len(access))
	for i := range names {
		refs[i] = msgraph.ItemRef("d1", names[i])
	}
	switch fault {
	case "script":
		w.f.Delete("d1", names[0])
	case "":
	default:
		w.f.FailCalls(fault, len(refs)) // every permissions call of the batch
	}
	vs, err := w.sp.ACL(context.Background(), ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}, refs)
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.String()
	}
	return out, err
}

func (w *sharePointWorld) Act(key string, _ bool) (string, error) {
	r, err := w.sp.Act(context.Background(), ports.Action{Target: "docs:x"}, ports.IdempotencyKey(key))
	return r.TargetID, err
}

func (w *sharePointWorld) Effects() int { return 0 }

// SharePointCheck asks whether ann may read a document now.
type SharePointCheck func(sp *msgraph.SharePoint, ref ports.SourceRef) ports.Verdict

func liveSharePoint(sp *msgraph.SharePoint, ref ports.SourceRef) ports.Verdict {
	vs, err := sp.ACL(context.Background(), ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}, []ports.SourceRef{ref})
	if err != nil || len(vs) != 1 {
		return ports.VerdictUnknown
	}
	return vs[0]
}

// law01SharePoint: an item shared with a group ann is in through a nested group, then
// shared and unshared at random after the fetch; the verdict always follows Graph now.
func law01SharePoint(r *rand.Rand, check SharePointCheck) error {
	f, sp := newSharePoint()
	f.Mails["g-all"] = "all@acme.example"
	f.Groups["g-all"] = []string{"g:g-fin"}
	f.Groups["g-fin"] = []string{"ann@acme.example"}
	f.Put("d1", graphtest.Item{ID: "plan", Name: "plan.md", Content: []byte("The plan.")})
	ref := msgraph.ItemRef("d1", "plan")
	for i := 0; i < 6; i++ {
		can := r.Intn(2) == 0
		it := graphtest.Item{ID: "plan", Name: "plan.md", Content: []byte("The plan.")}
		if can {
			it.Groups = []string{"g-all"}
		}
		f.Put("d1", it)
		want := map[bool]ports.Verdict{true: ports.VerdictAllow, false: ports.VerdictDeny}[can]
		if got := check(sp, ref); got != want {
			return fmt.Errorf("after Graph set ann's access to %v, the check said %v", can, got)
		}
	}
	return nil
}
