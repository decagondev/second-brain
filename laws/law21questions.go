package laws

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	actmem "example.com/second-brain/internal/adapters/actions/memory"
	"example.com/second-brain/internal/adapters/connectors/folder"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/toolbus"
)

// W16 Ask and capture (SB-1704, SB-1705) under the laws.
//
// LAW-01: a steward's reply shows the asker only the cites the asker can open now, however
// many the steward added. LAW-21: whether someone works a space's queue is decided by their
// groups at that request, so a steward who leaves the group stops seeing it. LAW-04: a
// question or a reply that can't be audited is never stored. LAW-05: a note is Ran only
// with a receipt that is the hash of the note as it is on the share.

// questionSystem's mutations are for the self-tests.
type questionSystem struct {
	stewardSeesAll bool // the asker is shown every cite the steward gave
	anyoneAnswers  bool // the steward rule ignores the caller's groups
}

type failingAudit struct{}

func (failingAudit) Write(context.Context, audit.Event) (audit.Row, error) {
	return audit.Row{}, errors.New("audit down")
}

// lawStewards is one tenant's service: the world's agent, the cedar base policies, and
// company questions answered by the stewards group.
func (q questionSystem) stewards(ctx context.Context, w *World, tenant, group string) (ask.Stewards, error) {
	a, err := searchSystem{}.agent(ctx, w, tenant, lawOwners{})
	if err != nil {
		return ask.Stewards{}, err
	}
	if q.stewardSeesAll {
		a.Auth.Connectors["docs"] = worldConnector{w, true}
	}
	n := 0
	k := policy.Kernel{Engine: askPolicies, Recorder: okRecorder{}}
	if q.anyoneAnswers {
		k = policy.Kernel{Engine: allowEngine{}, Recorder: okRecorder{}}
	}
	return ask.Stewards{Agent: a, Store: tracemem.New(), Audit: &keepRows{}, Policy: k,
		Facts: func(string) ([]string, string, bool) { return []string{"group:" + group}, "", true },
		NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }, Now: time.Now}, nil
}

type allowEngine struct{}

func (allowEngine) Evaluate(context.Context, policy.Request) (policy.Result, error) {
	return policy.Result{Decision: policy.Allow}, nil
}

// sendOne sends a question as the asker, from an answer of theirs that nothing answered.
func sendOne(ctx context.Context, x ask.Stewards, scope tenancy.Scope) (ask.StewardQuestion, error) {
	n := ask.NoticeNoSources
	tr := ask.Trace{ID: x.NewID("tr_"), Tenant: scope.Tenant(), User: scope.User(), Space: scope.Space(), At: time.Now(),
		Response: ask.Response{Notice: &n, Agent: "ask"}}
	if err := x.Agent.Store.PutTrace(ctx, tr); err != nil {
		return ask.StewardQuestion{}, err
	}
	return x.Send(ctx, scope, "zzqx unanswerable", tr.ID)
}

func law01StewardCites(r *rand.Rand, q questionSystem) error {
	w := Gen(r)
	ctx := context.Background()
	for _, asker := range w.Users {
		var steward User
		for _, u := range w.Users {
			if u.Tenant == asker.Tenant && u.ID != asker.ID {
				steward = u
				break
			}
		}
		if steward.ID == "" || len(steward.Groups) == 0 {
			continue
		}
		x, err := q.stewards(ctx, w, asker.Tenant, steward.Groups[0])
		if err != nil {
			return err
		}
		as, _ := scopeOf(asker)
		st, _ := scopeOf(steward)
		sq, err := sendOne(ctx, x, as)
		if err != nil {
			return err
		}
		var cites []string
		for _, d := range w.Docs {
			if d.Tenant == asker.Tenant && w.truthAllow(steward.ID, d.ID) && len(cites) < ask.MaxReplyCites {
				cites = append(cites, fmt.Sprintf("%s@%d", d.ID, d.Version))
			}
		}
		if _, err := x.Reply(ctx, st, "company", sq.ID, "Ask finance, see these.", cites); err != nil {
			return fmt.Errorf("the steward couldn't cite documents they can open: %v", err)
		}
		mine, err := x.Mine(ctx, as)
		if err != nil {
			return err
		}
		for _, it := range mine {
			if it.Reply == nil {
				return fmt.Errorf("the asker's question shows no reply")
			}
			for _, c := range it.Reply.Cites {
				if d := docOf(c.Cite); !w.truthAllow(asker.ID, d) {
					return fmt.Errorf("%s was shown %s through a steward's reply, which the source doesn't allow them (LAW-01)", asker.ID, d)
				}
			}
			if len(it.Reply.Cites)+it.Reply.Dropped != len(cites) {
				return fmt.Errorf("%d shown and %d dropped of %d cites", len(it.Reply.Cites), it.Reply.Dropped, len(cites))
			}
		}
	}
	return nil
}

func law21StewardQueue(r *rand.Rand, q questionSystem) error {
	w := Gen(r)
	ctx := context.Background()
	u := w.Users[r.Intn(len(w.Users))]
	if len(u.Groups) == 0 {
		return nil
	}
	x, err := q.stewards(ctx, w, u.Tenant, u.Groups[0])
	if err != nil {
		return err
	}
	as, _ := scopeOf(u)
	sq, err := sendOne(ctx, x, as)
	if err != nil {
		return err
	}
	in, _ := tenancy.NewIssuer(u.Tenant).Issue(u.Tenant, u.ID, u.Groups)
	if _, err := x.Queue(ctx, in, "company"); err != nil {
		return fmt.Errorf("a steward was refused their queue: %v", err)
	}
	// The same person, after the directory takes them out of the stewards group.
	out, _ := tenancy.NewIssuer(u.Tenant).Issue(u.Tenant, u.ID, u.Groups[1:])
	if _, err := x.Queue(ctx, out, "company"); !errors.Is(err, ask.ErrNotSteward) {
		return fmt.Errorf("a steward who left the group still sees the queue (LAW-21)")
	}
	if _, err := x.Reply(ctx, out, "company", sq.ID, "late", nil); !errors.Is(err, ask.ErrNotSteward) {
		return fmt.Errorf("a steward who left the group still replies (LAW-21)")
	}
	return nil
}

func law04Questions(r *rand.Rand) error {
	w := Gen(r)
	ctx := context.Background()
	u := w.Users[r.Intn(len(w.Users))]
	if len(u.Groups) == 0 {
		return nil
	}
	x, err := questionSystem{}.stewards(ctx, w, u.Tenant, u.Groups[0])
	if err != nil {
		return err
	}
	x.Audit = failingAudit{}
	as, _ := scopeOf(u)
	if _, err := sendOne(ctx, x, as); err == nil {
		return fmt.Errorf("a question was sent with the audit log down (LAW-04)")
	}
	if got, _ := x.Store.StewardQuestionsOf(ctx, u.Tenant, u.ID); len(got) != 0 {
		return fmt.Errorf("an unaudited question was stored (LAW-04)")
	}
	return nil
}

type lawKill struct{}

func (lawKill) Check(kill.Op) error { return nil }

// law05Notes publishes notes through the real bus and folder connector, some to a path
// already holding another note, and checks Ran only ever comes with the file's hash.
func law05Notes(r *rand.Rand, dir string) error {
	ctx := context.Background()
	share := filepath.Join(dir, fmt.Sprint(r.Int63()))
	if err := os.MkdirAll(share, 0o755); err != nil {
		return err
	}
	c, err := folder.Open(share, "")
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	if err := c.EnableNotes(share); err != nil {
		return err
	}
	k := policy.Kernel{Engine: askPolicies, Recorder: okRecorder{}}
	n := 0
	bus := &toolbus.Bus{Tenant: "acme-logistics", Tools: map[string]toolbus.Tool{ask.NoteTool: {Name: ask.NoteTool, Connector: "docs"}},
		Kill: lawKill{}, Policy: k, Audit: &keepRows{}, Actions: actmem.New(), Connectors: map[string]ports.Connector{"docs": c},
		Now: time.Now, NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }}
	s, _ := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", "usr_s", []string{"support-leads"})
	other, _ := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", "usr_o", []string{"sales"})
	for i := 0; i < 4; i++ {
		target := fmt.Sprintf("notes/support/qs_%d.md", r.Intn(3))
		text := fmt.Sprintf("# Refunds\n\nReply %d.\n", r.Intn(3))
		payload := fmt.Sprintf(`{"markdown":%q,"readers":["ann@acme.example"],"owner":"sam@acme.example","label":"internal"}`, text)
		who, outsider := s, r.Intn(4) == 0
		if outsider {
			who = other
		}
		a, err := bus.Invoke(ctx, who, toolbus.Call{Agent: "steward", Tool: ask.NoteTool, Target: target, Payload: []byte(payload),
			Context: map[string]any{"stewards": []string{"group:support-leads"}, "groups": groupNames(who)}})
		if outsider {
			if err == nil {
				return fmt.Errorf("someone who isn't a steward published a note")
			}
			continue
		}
		if err != nil {
			return err
		}
		if a.State == approvals.Ran {
			b, rerr := os.ReadFile(filepath.Join(share, filepath.FromSlash(target)))
			if rerr != nil || !strings.HasSuffix(a.Receipt, "@sha256:"+sha256hex(b)) {
				return fmt.Errorf("a note was Ran with receipt %q that isn't the file on the share (LAW-05)", a.Receipt)
			}
		} else if a.Receipt != "" {
			return fmt.Errorf("a note that didn't run carries a receipt")
		}
	}
	return nil
}

func groupNames(s tenancy.Scope) []string {
	var out []string
	for _, g := range s.Groups() {
		out = append(out, "group:"+g)
	}
	return out
}

func sha256hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
