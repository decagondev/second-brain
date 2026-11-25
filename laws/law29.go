package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/toolbus"
)

// LAW-29 A rhythm only drafts, and stops with its subject (SB-4801; sb-laws(7)): every step
// a rhythm offers waits, unanswered, in the queue of the person or stewards it names, and
// nothing is published or sent on its own; once its subject leaves the space, the next read
// shows it stopped, and no step is offered after that. A random rhythm of up to 12 steps
// over up to 120 days, a clock that jumps, and the subject leaving at a random moment, with
// the real rhythms service, steward queues, folder connector and index.

// RhythmSystem is the system under test: the rhythms service over stewards, with the
// members of the space now.
type RhythmSystem interface {
	Rhythms(dir string, members func() []string, now func() time.Time) (x ask.Rhythms, published func() int, closer func() error, err error)
}

// RealRhythms is the real system. Its fields are the self-tests' faults.
type RealRhythms struct {
	// IgnoreLeave keeps a person who left a member, as a rhythm that never checks would.
	IgnoreLeave bool
	// AutoReply answers each step as it is offered, as a rhythm that acts on its own would.
	AutoReply bool
}

// countingBus counts every note a stewards' service publishes.
type countingBus struct {
	next ask.NoteBus
	n    *int
}

func (b countingBus) Invoke(ctx context.Context, s tenancy.Scope, call toolbus.Call) (approvals.Action, error) {
	*b.n++
	return b.next.Invoke(ctx, s, call)
}

// autoReplying answers every rhythm step as it is put.
type autoReplying struct{ *tracemem.Store }

func (a autoReplying) PutStewardQuestion(ctx context.Context, q ask.StewardQuestion) error {
	if q.Rhythm != nil && q.Reply == nil {
		q.Reply = &ask.StewardReply{By: "rhythm", Text: "Sent.", At: time.Now()}
	}
	return a.Store.PutStewardQuestion(ctx, q)
}

// Rhythms implements RhythmSystem.
func (s RealRhythms) Rhythms(dir string, members func() []string, now func() time.Time) (ask.Rhythms, func() int, func() error, error) {
	x, c, sync, err := RealNoteNarrowing{}.Stewards(dir, members)
	if err != nil {
		return ask.Rhythms{}, nil, nil, err
	}
	if err := sync(); err != nil {
		return ask.Rhythms{}, nil, nil, err
	}
	x.Agent.Model = &citingModel{}
	published := 0
	x.Bus = countingBus{x.Bus, &published}
	x.Now, x.Agent.Now = now, now
	x.StartItems = func(string) []string { return []string{"docs:/"} }
	store := tracemem.New()
	x.Store = store
	if s.AutoReply {
		x.Store = autoReplying{store}
	}
	iss := tenancy.NewIssuer("acme-logistics")
	isMember := func(user string) bool {
		for _, m := range members() {
			if "usr_"+m == user || m == user {
				return true
			}
		}
		return s.IgnoreLeave
	}
	r := ask.Rhythms{Stewards: x, Store: tracemem.New(),
		Defs:   func(string) map[string]ask.RhythmDef { return map[string]ask.RhythmDef{"law": lawRhythm} },
		Spaces: func() []string { return []string{"support"} },
		ScopeIn: func(_ context.Context, user, space string) (tenancy.Scope, error) {
			if !isMember(user) {
				return tenancy.Scope{}, errors.New("not a member")
			}
			return iss.IssueIn("acme-logistics", user, []string{"support"}, "spc_"+space)
		},
		Manager: func(context.Context, string) (string, error) { return "usr_mia", nil },
	}
	return r, func() int { return published }, c.Close, nil
}

// lawRhythm is the rhythm each run draws; law29 sets it.
var lawRhythm ask.RhythmDef

func randomRhythm(r *rand.Rand) ask.RhythmDef {
	kinds := []string{"draft", "routine", "checkin", "request"}
	approvers := []string{"subject", "starter", "manager", "steward"}
	d := ask.RhythmDef{Subject: "person"}
	day := 0
	for i := 0; i < 1+r.Intn(12); i++ {
		day += r.Intn(1 + (120-day)/3)
		st := ask.RhythmStepDef{Day: day, Kind: kinds[r.Intn(len(kinds))], Ask: fmt.Sprintf("Step %d", i+1), Approver: approvers[r.Intn(len(approvers))]}
		if st.Kind == "draft" && r.Intn(2) == 0 {
			st.Agent = "start-here"
		}
		d.Steps = append(d.Steps, st)
	}
	return d
}

func law29(r *rand.Rand, dir string, sys RhythmSystem) error {
	ctx := context.Background()
	share, err := os.MkdirTemp(dir, "share")
	if err != nil {
		return err
	}
	for _, topic := range []string{"welcome", "refunds", "returns"} {
		p := filepath.Join(share, topic+".md")
		if err := os.WriteFile(p, []byte("# "+topic+"\n\nHow "+topic+" work at acme-logistics.\n"), 0o600); err != nil {
			return err
		}
		if err := os.WriteFile(p+".acl", []byte("read domain:acme.example\nowner sam@acme.example\n"), 0o600); err != nil {
			return err
		}
	}
	members := []string{"ann", "sam", "mia"}
	clock := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	lawRhythm = randomRhythm(r)
	x, published, closer, err := sys.Rhythms(share, func() []string { return members }, func() time.Time { return clock })
	if err != nil {
		return err
	}
	defer func() { _ = closer() }()
	steward, _ := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", "usr_sam", []string{"support-leads"})
	x.Stewards.Owner = func(_ context.Context, email string) (string, bool, error) {
		return "usr_ann", email == "ann@acme.example", nil
	}
	v, err := x.Start(ctx, steward, "support", ask.RhythmStart{Rhythm: "law", Subject: "ann@acme.example"})
	if err != nil {
		return fmt.Errorf("start: %v", err)
	}
	items := func() ([]ask.StewardQuestion, error) {
		return x.Stewards.Store.StewardQuestionsBetween(ctx, "acme-logistics", time.Time{}, clock.Add(time.Hour))
	}
	leaveAt := r.Intn(8)
	left, offeredAtLeave := false, -1
	for round := 0; round < 8; round++ {
		if round == leaveAt {
			members = []string{"sam", "mia"}
			left = true
			vs, err := x.For(ctx, steward, "support")
			if err != nil {
				return err
			}
			if len(vs) != 1 || vs[0].State != ask.RhythmStopped {
				return fmt.Errorf("the subject left and the next read shows the rhythm %+v, not stopped (LAW-29)", vs)
			}
			all, err := items()
			if err != nil {
				return err
			}
			offeredAtLeave = len(all)
		}
		clock = clock.Add(time.Duration(r.Intn(31)) * 24 * time.Hour)
		if _, err := x.Tick(ctx); err != nil {
			return fmt.Errorf("tick: %v", err)
		}
		all, err := items()
		if err != nil {
			return err
		}
		if left && len(all) != offeredAtLeave {
			return fmt.Errorf("%d steps were offered after the subject left (LAW-29)", len(all)-offeredAtLeave)
		}
		for _, q := range all {
			if q.Rhythm == nil || q.Rhythm.Run != v.ID {
				return fmt.Errorf("an item %s not of the rhythm appeared", q.ID)
			}
			if q.Reply != nil || q.Thread != nil && q.Thread.Discarded {
				return fmt.Errorf("step day %d was answered without its approver: %+v (LAW-29)", q.Rhythm.Day, q.Reply)
			}
			if q.Rhythm.Step < 0 {
				continue
			}
			want := "spc_support"
			switch lawRhythm.Steps[q.Rhythm.Step].Approver {
			case "subject":
				want = ask.PersonKey("usr_ann")
			case "starter":
				want = ask.PersonKey("usr_sam")
			case "manager":
				want = ask.PersonKey("usr_mia")
			}
			if q.Space != want {
				return fmt.Errorf("a %s step waits in %s, not its approver's %s (LAW-29)", lawRhythm.Steps[q.Rhythm.Step].Approver, q.Space, want)
			}
		}
		if published() != 0 {
			return fmt.Errorf("a rhythm published %d notes on its own (LAW-29)", published())
		}
	}
	return nil
}
