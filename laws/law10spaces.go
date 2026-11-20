package laws

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-10 for a space (SB-1008): within 10 seconds of `sb kill space NAME`, an ask in that
// space is refused, and no answer anywhere, company-wide included, uses a chunk the space
// holds; the other spaces keep working. Unkill brings it back. Real checker, real
// retrieval and policies, on a fake clock.

// SpaceKillHook is how retrieval learns a space is killed (the checker in the real system).
type SpaceKillHook func(c *kill.Checker) func(string) bool

func law10Spaces(r *rand.Rand, hook SpaceKillHook) error {
	ctx := context.Background()
	w := newKillWorld(false)
	ev, err := LoadPolicies()
	if err != nil {
		return err
	}
	all := []ports.Space{
		{ID: spaces.ID("finance"), Tenant: spaceTenant, Name: "finance", Members: []string{"group:staff"}},
		{ID: spaces.ID("eng"), Tenant: spaceTenant, Name: "eng", Members: []string{"group:staff"}},
	}
	idx := memindex.New()
	holders := map[string][]string{"d-fin": {"spc_finance"}, "d-eng": {"spc_eng"}, "d-both": {"spc_eng", "spc_finance"}, "d-none": {}}
	for id, h := range holders {
		c := ports.Chunk{ID: "chk_" + id, Tenant: spaceTenant, Cite: "docs:" + id + "@1", Kind: "docs", SourceID: id, Version: 1,
			Text: "the brain knows " + id, ACLPrefilter: []string{"anyone"}, Label: "internal", Spaces: h}
		if err := idx.Replace(ctx, spaceTenant, ports.SourceRef{Kind: "docs", ID: id}, 1, []ports.Chunk{c}); err != nil {
			return err
		}
	}
	src := &askSpaces{spec: SpaceSpec{Docs: []SpaceDoc{}}}
	for id := range holders {
		src.spec.Docs = append(src.spec.Docs, SpaceDoc{ID: id, Viewers: []string{"usr_1"}})
	}
	auth := retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": viewers{src}},
		Policy:      policy.Kernel{Engine: ev, Recorder: okRecorder{}},
		Spaces:      func(context.Context) ([]ports.Space, [][2]string, error) { return all, nil, nil },
		SpaceKilled: hook(w.checker)}
	w.Advance(3 * time.Second)
	now := 3 * time.Second
	killedAt := map[string]time.Duration{}
	clearedAt := map[string]time.Duration{} // an unkill takes effect within the same bound
	for step := 0; step < 30; step++ {
		switch r.Intn(4) {
		case 0:
			n := []string{"finance", "eng"}[r.Intn(2)]
			if err := w.Kill("space:" + n); err != nil {
				return err
			}
			if _, ok := killedAt[n]; !ok {
				killedAt[n] = now
			}
		case 1:
			for n := range killedAt {
				if err := w.Unkill("space:" + n); err != nil {
					return err
				}
				delete(killedAt, n)
				clearedAt[n] = now
				break
			}
		default:
			d := time.Duration(r.Intn(6000)) * time.Millisecond
			w.Advance(d)
			now += d
		}
		settled := func(n string) bool { at, ok := killedAt[n]; return ok && now-at >= kill.Bound }
		live := func(n string) bool {
			_, killed := killedAt[n]
			at, cleared := clearedAt[n]
			return !killed && (!cleared || now-at >= kill.Bound)
		}
		// An ask in each space.
		for _, n := range []string{"finance", "eng"} {
			in := w.checker.Check(kill.Op{Agent: "ask", Space: n}) == nil
			if settled(n) && in {
				return fmt.Errorf("an ask in killed space %s ran %v after the kill", n, now-killedAt[n])
			}
			if live(n) && !in {
				return fmt.Errorf("an ask in %s, which isn't killed, was refused", n)
			}
		}
		// A company-wide answer.
		scope, err := tenancy.NewIssuer(spaceTenant).IssueIn(spaceTenant, "usr_1", []string{"staff"}, spaces.Company)
		if err != nil {
			return err
		}
		res, err := retrieve.Context(ctx, scope, ports.UserRef{ID: "usr_1"}, nil, idx, auth, ports.Query{Text: "brain", Limit: 100})
		if err != nil {
			return err
		}
		used := map[string]bool{}
		for _, c := range res.Context {
			used[c.SourceID] = true
		}
		for id, hs := range holders {
			allLive, settledHolder := true, false
			for _, h := range hs {
				n := h[len("spc_"):]
				allLive = allLive && live(n)
				settledHolder = settledHolder || settled(n)
			}
			if settledHolder && used[id] {
				return fmt.Errorf("a company-wide answer used %s, held by a killed space, %v after the kill", id, now)
			}
			if allLive && !used[id] {
				return fmt.Errorf("a company-wide answer dropped %s, whose spaces aren't killed (killed: %v)", id, killedAt)
			}
		}
	}
	return nil
}

// LAW-11 with spaces (SB-1008): the space a row was written in is part of the hash, so a
// row moved to another space, or out of one, breaks the chain.

// RowHash is the chain's hash function (audit.HashOf in the real system).
type RowHash func(audit.Row) (string, error)

func law11Spaces(r *rand.Rand, hash RowHash) error {
	ctx := context.Background()
	st := auditmem.New()
	alog := &audit.Log{Tenant: spaceTenant, Release: func() string { return "rel_law" }, Store: st,
		Now: func() time.Time { return time.Unix(1_800_000_000, 0) }}
	for i := 0; i < 3+r.Intn(6); i++ {
		sp := []string{"", "company", "spc_finance", "spc_people"}[r.Intn(4)]
		if _, err := alog.Write(ctx, audit.Event{Actor: "usr_1", Kind: "decision", Subject: fmt.Sprintf("Chunk::\"c%d\"", i),
			State: "allow", Reason: "read: permitted", Space: sp}); err != nil {
			return err
		}
	}
	rows, err := st.Range(ctx, 1, 100)
	if err != nil {
		return err
	}
	verify := func(rs []audit.Row) bool {
		prev := audit.Zero
		for _, row := range rs {
			h, err := hash(row)
			if err != nil || row.Prev != prev || h != row.Hash {
				return false
			}
			prev = row.Hash
		}
		return true
	}
	if !verify(rows) {
		return fmt.Errorf("the chain as written doesn't verify")
	}
	for _, row := range rows {
		if row.Space == "company" {
			return fmt.Errorf("row %d records company as a space; company-wide rows carry none", row.Seq)
		}
	}
	k := r.Intn(len(rows))
	tampered := append([]audit.Row(nil), rows...)
	tampered[k].Space = map[string]string{"": "spc_finance", "spc_finance": "spc_people", "spc_people": ""}[tampered[k].Space]
	if verify(tampered) {
		return fmt.Errorf("moving row %d to space %q left the chain verifying", tampered[k].Seq, tampered[k].Space)
	}
	return nil
}
