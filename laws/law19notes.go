package laws

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/connectors/folder"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/toolbus"
)

// LAW-19 and LAW-21 for notes (SB-2004): a note's readers are the space's members when it
// was published. When the space narrows, the stewards' review list shows the note with
// the number of its readers who are no longer members, never who; republishing it narrowed
// takes it off the list, its ACL file then names no one it didn't before, and no one who
// left. The real folder connector, ingest pipeline, index, review list and note publish.

// NoteNarrowing is the system under test: it publishes and republishes notes.
type NoteNarrowing interface {
	// Stewards is the service over a share in dir, with members now as the space's readers.
	Stewards(dir string, members func() []string) (ask.Stewards, *folder.Connector, func() error, error)
}

// RealNoteNarrowing is the real one.
type RealNoteNarrowing struct{}

// directBus runs note-publish straight on the connector: the bus is LAW-04's, not this law's.
type directBus struct{ c *folder.Connector }

func (b directBus) Invoke(ctx context.Context, _ tenancy.Scope, call toolbus.Call) (approvals.Action, error) {
	rc, err := b.c.Act(ctx, ports.Action{Target: call.Target, Payload: call.Payload}, ports.IdempotencyKey(fmt.Sprint(time.Now().UnixNano())))
	if err != nil {
		return approvals.Action{}, err
	}
	return approvals.Action{ID: "tc_1", State: approvals.Ran, Receipt: rc.TargetID}, nil
}

// Stewards implements NoteNarrowing.
func (RealNoteNarrowing) Stewards(dir string, members func() []string) (ask.Stewards, *folder.Connector, func() error, error) {
	c, err := folder.Open(dir, "")
	if err != nil {
		return ask.Stewards{}, nil, nil, err
	}
	if err := c.EnableNotes(dir); err != nil {
		return ask.Stewards{}, nil, nil, err
	}
	idx := memindex.New()
	p := ingest.Pipeline{Tenant: "acme-logistics", Release: "rel_test", Connector: c, Embedder: hashing.Embedder{}, Index: idx}
	var cur ports.Cursor
	sync := func() error {
		next, err := p.Sync(context.Background(), cur)
		cur = next
		return err
	}
	n := 0
	allow := policy.Kernel{Engine: allowEngine{}, Recorder: okRecorder{}} // stewardship isn't under test here
	a := ask.Agent{Tenant: "acme-logistics", Release: "rel_test", Index: idx,
		Auth:   retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": c}, Policy: allow},
		Store:  tracemem.New(),
		Killed: func(context.Context) bool { return false },
		Emails: func(_ context.Context, u string) ([]string, error) {
			return []string{strings.TrimPrefix(u, "usr_") + "@acme.example"}, nil
		},
		NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }, Now: time.Now}
	x := ask.Stewards{Agent: a, Store: tracemem.New(), Audit: &keepRows{}, Policy: allow,
		Facts:   func(string) ([]string, string, bool) { return []string{"group:support-leads"}, "internal", true },
		Bus:     directBus{c},
		Readers: func(context.Context, string) ([]string, error) { return members(), nil },
		NewID:   a.NewID, Now: time.Now}
	return x, c, sync, nil
}

func law19Notes(r *rand.Rand, dir string, sys NoteNarrowing) error {
	ctx := context.Background()
	share, err := os.MkdirTemp(dir, "share")
	if err != nil {
		return err
	}
	names := []string{"ann", "bob", "cat", "dan", "eve", "fay", "gus", "hal"}
	r.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	before := names[:2+r.Intn(4)]
	members := append([]string(nil), before...)
	mail := func(ns []string) []string {
		var out []string
		for _, n := range ns {
			out = append(out, n+"@acme.example")
		}
		return out
	}
	x, c, sync, err := sys.Stewards(share, func() []string { return mail(members) })
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	iss := tenancy.NewIssuer("acme-logistics")
	asker, _ := iss.IssueIn("acme-logistics", "usr_"+before[0], []string{"support"}, "spc_support")
	steward, _ := iss.Issue("acme-logistics", "usr_sam", []string{"support-leads"})
	sq, err := sendOne(ctx, x, asker)
	if err != nil {
		return err
	}
	if _, err := x.Reply(ctx, steward, "support", sq.ID, "Over 500 needs a lead.", nil); err != nil {
		return err
	}
	publish := func(title string) error {
		pv, err := x.Note(ctx, steward, "support", sq.ID, ask.NoteRequest{Title: title})
		if err != nil {
			return err
		}
		if _, err := x.Note(ctx, steward, "support", sq.ID, ask.NoteRequest{Title: title, Confirm: &pv.Narrower}); err != nil {
			return err
		}
		return sync()
	}
	if err := publish("Refunds over 500"); err != nil {
		return fmt.Errorf("the note wasn't published: %v", err)
	}
	// The space changes: some members leave, someone may join.
	left := r.Intn(len(before))
	members = append([]string(nil), before[left:]...)
	joined := ""
	if r.Intn(2) == 0 {
		joined = names[len(names)-1]
		members = append(members, joined)
	}
	res, err := x.Review(ctx, steward, "support", tracemem.New(), 0)
	if err != nil {
		return err
	}
	wide := 0
	for _, it := range res.Items {
		if it.Wider > 0 {
			wide++
			if it.Wider != left || it.Question != sq.ID {
				return fmt.Errorf("the note is listed with %d readers to remove for question %q; %d left the space", it.Wider, it.Question, left)
			}
		}
	}
	if (left > 0) != (wide == 1) || wide > 1 {
		return fmt.Errorf("%d members left and the review list shows %d notes read wider than the space (LAW-19)", left, wide)
	}
	body, _ := json.Marshal(res)
	for _, n := range before[:left] {
		if strings.Contains(string(body), n+"@") {
			return fmt.Errorf("the review list names %s, who left the space", n)
		}
	}
	if left == 0 {
		return nil
	}
	if err := publish(""); err != nil { // republished narrowed, under its own title
		return fmt.Errorf("the note couldn't be republished narrowed: %v", err)
	}
	if res, err = x.Review(ctx, steward, "support", tracemem.New(), 0); err != nil {
		return err
	}
	for _, it := range res.Items {
		if it.Wider > 0 {
			return fmt.Errorf("the note republished narrowed is still on the list with %d readers to remove", it.Wider)
		}
	}
	path := "notes/support/" + sq.ID + ".md"
	var allowed []string
	for _, n := range names {
		v, err := c.ACL(ctx, ports.UserRef{Email: n + "@acme.example"}, []ports.SourceRef{{Kind: "docs", ID: path}})
		if err != nil {
			return err
		}
		if v[0] == ports.VerdictAllow {
			allowed = append(allowed, n)
		}
	}
	want := append([]string(nil), before[left:]...)
	sort.Strings(want)
	sort.Strings(allowed)
	if strings.Join(allowed, ",") != strings.Join(want, ",") {
		return fmt.Errorf("after republishing, the note is readable by %v; want the members who could read it and still can, %v (joined: %q; LAW-21)", allowed, want, joined)
	}
	return nil
}
