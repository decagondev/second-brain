package laws

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/connectors/folder"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-30 A brief about a person draws only on what they published, and they can read it
// (SB-4807; sb-laws(7)): the 1:1 brief a manager gets for a report holds only goals the
// report owns and their open action items, each cited to a document both can open; nothing
// from the report's asks, memory or routines; and the report's copy is the same text. Random
// goals and action items in documents both, only the manager, or only the report can open,
// beside private asks, memory and routines with unique markers, through the real rhythms
// service, steward queues, folder connector, ingest and index.

// RealBriefs is the real system. Its field is the self-test's fault.
type RealBriefs struct {
	// LeakAsk adds the report's latest ask to their action items, as a brief that reads
	// their asks would.
	LeakAsk bool
}

// leakingActions lists the report's latest private ask as one of their action items.
type leakingActions struct {
	ask.ActionStore
	private *tracemem.Store
	doc     string
}

func (l leakingActions) ActionsOf(ctx context.Context, tenant, user string) ([]ask.Action, error) {
	as, err := l.ActionStore.ActionsOf(ctx, tenant, user)
	if err != nil {
		return nil, err
	}
	qs, ok, err := l.private.Questions(ctx, tenant, user, "trc_0")
	if err != nil || !ok || len(qs) == 0 {
		return as, err
	}
	return append(as, ask.Action{ID: "act_leak", Tenant: tenant, Space: "support", Doc: l.doc, Text: qs[len(qs)-1], User: user}), nil
}

const briefTenant = "acme-logistics"

// Briefs is the rhythms service over a folder share at dir, with the report's private
// records in private and their action items in actions.
func (b RealBriefs) Briefs(dir string, private *tracemem.Store, now func() time.Time) (ask.Rhythms, func() error, error) {
	c, err := folder.Open(dir, "")
	if err != nil {
		return ask.Rhythms{}, nil, err
	}
	sp := ports.Space{ID: spaces.ID("support"), Tenant: briefTenant, Name: "support", Members: []string{"group:support"},
		Stewards: []string{"group:support-leads"}, Label: "internal", Scope: map[string][]string{"docs": {"/Support/**"}}}
	all := func(context.Context) ([]ports.Space, error) { return []ports.Space{sp}, nil }
	idx := memindex.New()
	p := ingest.Pipeline{Tenant: briefTenant, Release: "rel_test", Connector: c, Embedder: hashing.Embedder{}, Index: idx, Spaces: all}
	var cur ports.Cursor
	if _, err := p.Sync(context.Background(), cur); err != nil {
		_ = c.Close()
		return ask.Rhythms{}, nil, err
	}
	n := 0
	allow := policy.Kernel{Engine: allowEngine{}, Recorder: okRecorder{}} // the space policy isn't under test here
	a := ask.Agent{Tenant: briefTenant, Release: "rel_test", Index: idx,
		Auth: retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": c}, Policy: allow,
			Spaces: func(ctx context.Context) ([]ports.Space, [][2]string, error) {
				s, err := all(ctx)
				return s, nil, err
			}},
		Store: private, Killed: func(context.Context) bool { return false },
		Emails: func(_ context.Context, u string) ([]string, error) {
			return []string{strings.TrimPrefix(u, "usr_") + "@acme.example"}, nil
		},
		NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }, Now: now}
	x := ask.Stewards{Agent: a, Store: tracemem.New(), Audit: &keepRows{}, Policy: allow, Bus: directBus{c},
		Readers: func(context.Context, string) ([]string, error) {
			return []string{"ann@acme.example", "mia@acme.example"}, nil
		},
		Owner: func(_ context.Context, email string) (string, bool, error) {
			local, _, _ := strings.Cut(email, "@")
			return "usr_" + local, local == "ann" || local == "mia", nil
		},
		NewID: a.NewID, Now: now}
	var actions ask.ActionStore = private
	if b.LeakAsk {
		actions = leakingActions{private, private, "docs:Support/both/standup.md"}
	}
	iss := tenancy.NewIssuer(briefTenant)
	r := ask.Rhythms{Stewards: x, Legwork: ask.Legwork{Agent: a, Stewards: x, Actions: actions, NewID: a.NewID, Now: now}, Store: tracemem.New(),
		Defs: func(string) map[string]ask.RhythmDef {
			return map[string]ask.RhythmDef{ask.OneOnOne: {Subject: "person", Steps: []ask.RhythmStepDef{
				{Day: 0, Kind: "draft", Ask: "Prepare this week's 1:1", Approver: "manager", Agent: ask.AgentOneOnOne}}}}
		},
		Spaces: func() []string { return []string{"support"} },
		ScopeIn: func(_ context.Context, user, space string) (tenancy.Scope, error) {
			return iss.IssueIn(briefTenant, user, []string{"support"}, "spc_"+space)
		},
		Manager: func(_ context.Context, user string) (string, error) {
			if user == "usr_ann" {
				return "usr_mia", nil
			}
			return "", nil
		},
	}
	return r, c.Close, nil
}

var briefCite = regexp.MustCompile(`\[(docs:[^\]]+)\]`)

// law30Docs writes the share: goals and transcripts in folders both, only the manager, or
// only the report can open. It returns the goal titles and action texts the brief must
// hold, and the ones it must not.
func law30Docs(r *rand.Rand, share string, private *tracemem.Store, now time.Time) (want, never []string, err error) {
	acl := map[string]string{"both": "read ann@acme.example\nread mia@acme.example\nowner mia@acme.example\n",
		"mgr": "read mia@acme.example\nowner mia@acme.example\n", "rep": "read ann@acme.example\nowner ann@acme.example\n"}
	write := func(rel, text string) error {
		p := filepath.Join(share, "Support", rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		return os.WriteFile(p, []byte(text), 0o600)
	}
	for f, a := range acl {
		if err := write(f+"/.acl", a); err != nil {
			return nil, nil, err
		}
	}
	if err := write("both/standup.md", "Stand-up notes.\n"); err != nil {
		return nil, nil, err
	}
	folders := []string{"both", "mgr", "rep"}
	statuses := []string{"on track", "at risk", "off track"}
	for i := 0; i < 1+r.Intn(6); i++ {
		f, owner := folders[r.Intn(3)], []string{"ann", "sam"}[r.Intn(2)]
		title := fmt.Sprintf("goal-%d-%s-%s", i, f, owner) // a goal's title is its file's name
		text := fmt.Sprintf("---\ntype: goal\nowner: %s@acme.example\nquarter: 2026-Q4\nstatus: %s\n---\n\n# %s\n\nWhat we do this quarter.\n", owner, statuses[r.Intn(3)], title)
		if err := write(f+"/"+title+".md", text); err != nil {
			return nil, nil, err
		}
		if f == "both" && owner == "ann" {
			want = append(want, title)
		} else {
			never = append(never, title)
		}
	}
	for i := 0; i < r.Intn(5); i++ {
		f, user := folders[r.Intn(3)], []string{"usr_ann", "usr_sam"}[r.Intn(2)]
		doc := fmt.Sprintf("%s/transcript-%d.md", f, i)
		if err := write(doc, "Meeting transcript.\n"); err != nil {
			return nil, nil, err
		}
		a := ask.Action{ID: fmt.Sprintf("act_%d", i), Tenant: briefTenant, Space: "support", Doc: "docs:Support/" + doc,
			Text: fmt.Sprintf("Action %d %s %s", i, f, user), User: user, Owner: strings.TrimPrefix(user, "usr_") + "@acme.example", At: now}
		done := r.Intn(4) == 0
		if done {
			t := now
			a.DoneAt = &t
		}
		if err := private.PutAction(context.Background(), a); err != nil {
			return nil, nil, err
		}
		if f == "both" && user == "usr_ann" && !done {
			want = append(want, a.Text)
		} else {
			never = append(never, a.Text)
		}
	}
	// The report's private records, each with a marker that must never appear.
	ctx := context.Background()
	for i := 0; i < 1+r.Intn(3); i++ {
		m := fmt.Sprintf("PRIVATE-ASK-%d-%d", i, r.Intn(1e6))
		if err := private.PutTrace(ctx, ask.Trace{ID: fmt.Sprintf("trc_%d", i), Tenant: briefTenant, User: "usr_ann", Space: "spc_support", At: now,
			Response: ask.Response{Trace: fmt.Sprintf("trc_%d", i), Answer: ask.Answer{Claims: []answer.Claim{{Text: m}}}}}); err != nil {
			return nil, nil, err
		}
		if err := private.AppendQuestion(ctx, briefTenant, "usr_ann", fmt.Sprintf("trc_%d", i), m); err != nil {
			return nil, nil, err
		}
		r2 := fmt.Sprintf("PRIVATE-ROUTINE-%d-%d", i, r.Intn(1e6))
		if err := private.PutSaved(ctx, ask.SavedQuestion{ID: fmt.Sprintf("sav_%d", i), Tenant: briefTenant, User: "usr_ann", Question: r2, Space: "support", At: now}); err != nil {
			return nil, nil, err
		}
		never = append(never, m, r2)
	}
	mem := fmt.Sprintf("PRIVATE-MEMORY-%d", r.Intn(1e6))
	if err := private.PutMemory(ctx, ask.Memory{Tenant: briefTenant, User: "usr_ann", Lines: []ask.MemoryLine{{ID: "mem_1", Text: mem, At: now}}, At: now}); err != nil {
		return nil, nil, err
	}
	return want, append(never, mem, "PRIVATE-"), nil
}

func law30(r *rand.Rand, dir string, sys RealBriefs) error {
	ctx := context.Background()
	share, err := os.MkdirTemp(dir, "share")
	if err != nil {
		return err
	}
	clock := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	private := tracemem.New()
	want, never, err := law30Docs(r, share, private, clock)
	if err != nil {
		return err
	}
	x, closer, err := sys.Briefs(share, private, func() time.Time { return clock })
	if err != nil {
		return err
	}
	defer func() { _ = closer() }()
	iss := tenancy.NewIssuer(briefTenant)
	mia, _ := iss.IssueIn(briefTenant, "usr_mia", []string{"support"}, "spc_support")
	ann, _ := iss.IssueIn(briefTenant, "usr_ann", []string{"support"}, "spc_support")
	v, err := x.Start(ctx, mia, "support", ask.RhythmStart{Rhythm: ask.OneOnOne, Subject: "ann@acme.example"})
	if err != nil {
		return fmt.Errorf("start: %v", err)
	}
	briefs := func() (string, string, error) {
		qs, err := x.Stewards.Store.StewardQuestionsBetween(ctx, briefTenant, time.Time{}, clock.Add(time.Hour))
		if err != nil {
			return "", "", err
		}
		var mgr, rep string
		for _, q := range qs {
			if q.Rhythm == nil || q.Rhythm.Run != v.ID || q.Thread == nil {
				continue
			}
			switch q.Space {
			case ask.PersonKey("usr_mia"):
				mgr = q.Thread.Text
			case ask.PersonKey("usr_ann"):
				rep = q.Thread.Text
			default:
				return "", "", fmt.Errorf("a 1:1 brief waits in %s, not the manager's or the report's queue", q.Space)
			}
		}
		if mgr == "" || mgr != rep {
			return "", "", fmt.Errorf("the report's copy differs from the manager's brief (LAW-30):\nmanager: %q\nreport:  %q", mgr, rep)
		}
		return mgr, rep, nil
	}
	brief, _, err := briefs()
	if err != nil {
		return err
	}
	for _, w := range want {
		if !strings.Contains(brief, w) {
			return fmt.Errorf("the brief leaves out %q, which the report published and both can open:\n%s", w, brief)
		}
	}
	for _, n := range never {
		if strings.Contains(brief, n) {
			return fmt.Errorf("the brief holds %q, which isn't the report's own published work both can open (LAW-30):\n%s", n, brief)
		}
	}
	for _, m := range briefCite.FindAllStringSubmatch(brief, -1) {
		if !strings.HasPrefix(m[1], "docs:Support/both/") {
			return fmt.Errorf("the brief cites %s, which one of them can't open (LAW-30)", m[1])
		}
	}
	if _, err := x.AddLine(ctx, mia, v.ID, "The manager's own line"); err == nil {
		return fmt.Errorf("the manager added a line to the report's brief")
	}
	line := fmt.Sprintf("I'd like to talk about %d things", r.Intn(9))
	if _, err := x.AddLine(ctx, ann, v.ID, line); err != nil {
		return fmt.Errorf("the report can't add a line: %v", err)
	}
	after, _, err := briefs()
	if err != nil {
		return err
	}
	if !strings.HasSuffix(after, "- From them: "+line) {
		return fmt.Errorf("the report's line isn't in both copies: %q", after)
	}
	return nil
}
