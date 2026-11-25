package kill

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/audit"
)

type mem struct {
	sw   map[string]Switch
	down bool
}

func (m *mem) Set(_ context.Context, _ string, s Switch) error { m.sw[s.Scope] = s; return nil }
func (m *mem) Clear(_ context.Context, _, scope string) error  { delete(m.sw, scope); return nil }
func (m *mem) List(context.Context, string) ([]Switch, error) {
	if m.down {
		return nil, errors.New("down")
	}
	var out []Switch
	for _, s := range m.sw {
		out = append(out, s)
	}
	return out, nil
}

type rows struct {
	rs   []audit.Row
	fail bool
}

func (r *rows) Append(_ context.Context, b func(int64, string) (audit.Row, error)) (audit.Row, error) {
	if r.fail {
		return audit.Row{}, errors.New("audit down")
	}
	seq, h := int64(0), audit.Zero
	if n := len(r.rs); n > 0 {
		seq, h = r.rs[n-1].Seq, r.rs[n-1].Hash
	}
	x, err := b(seq, h)
	if err == nil {
		r.rs = append(r.rs, x)
	}
	return x, err
}
func (r *rows) Range(context.Context, int64, int) ([]audit.Row, error) { return r.rs, nil }

func setup() (*Control, *Checker, *mem, *rows, *time.Time) {
	now := time.Unix(1_800_000_000, 0)
	clock := func() time.Time { return now }
	m, rs := &mem{sw: map[string]Switch{}}, &rows{}
	log := &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_x" }, Store: rs, Now: clock}
	return &Control{Store: m, Audit: log, Now: clock}, &Checker{Tenant: "acme-logistics", Store: m, Now: clock}, m, rs, &now
}

func TestAuditedFirst(t *testing.T) {
	c, _, m, rs, _ := setup()
	ctx := context.Background()
	if err := c.Kill(ctx, "acme-logistics", "agent:ask", "runaway answers", "usr_op"); err != nil {
		t.Fatal(err)
	}
	if len(rs.rs) != 1 || rs.rs[0].Kind != "kill" || rs.rs[0].State != "killed" || rs.rs[0].Subject != "agent:ask" || rs.rs[0].Reason != "runaway answers" {
		t.Fatalf("audit %+v", rs.rs)
	}
	rs.fail = true
	if err := c.Unkill(ctx, "acme-logistics", "agent:ask", "fixed", "usr_op"); err == nil {
		t.Fatal("unkill went ahead without an audit row")
	}
	if _, still := m.sw["agent:ask"]; !still {
		t.Fatal("the switch changed although its audit row failed")
	}
	for _, bad := range [][3]string{{"agent:Ask!", "r", "u"}, {"connector:email", "r", "u"}, {"tenant", "", "u"}} {
		if err := c.Kill(ctx, "acme-logistics", bad[0], bad[1], bad[2]); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
	if err := c.Kill(ctx, "beta-freight", "tenant", "r", "u"); !errors.Is(err, ErrNoAudit) {
		t.Errorf("another tenant's switch through this tenant's log: %v", err)
	}
}

func TestChecker(t *testing.T) {
	c, ch, m, _, now := setup()
	ctx := context.Background()
	if ch.Check(Op{Agent: "ask"}) == nil || ch.Check(Op{Admin: true}) != nil {
		t.Fatal("before the first load, only admin may go ahead")
	}
	_ = ch.Refresh(ctx)
	if err := ch.Check(Op{Agent: "ask", Connector: "docs", Consequential: true}); err != nil {
		t.Fatal(err)
	}
	_ = c.Kill(ctx, "acme-logistics", "connector:docs", "compromised", "usr_op")
	_ = ch.Refresh(ctx)
	if !errors.Is(ch.Check(Op{Connector: "docs"}), ErrKilled) || ch.Check(Op{Connector: "tickets"}) != nil || !ch.ConnectorKilled("docs") {
		t.Fatal("connector kill")
	}
	_ = c.Kill(ctx, "acme-logistics", "tenant", "incident", "usr_op")
	_ = ch.Refresh(ctx)
	if ch.Check(Op{Agent: "draft-reply"}) == nil || ch.Check(Op{Admin: true}) != nil || !ch.AgentKilled("ask")(ctx) {
		t.Fatal("tenant kill")
	}
	_ = c.Unkill(ctx, "acme-logistics", "tenant", "over", "usr_op")
	_ = c.Unkill(ctx, "acme-logistics", "connector:docs", "over", "usr_op")
	m.down = true
	*now = now.Add(29 * time.Second)
	_ = ch.Refresh(ctx) // fails; the old list (with the kills) stays
	if ch.Check(Op{Agent: "ask"}) == nil {
		t.Fatal("a failed refresh dropped the old switches")
	}
	*now = now.Add(2 * time.Second)
	if err := ch.Check(Op{Consequential: true}); !errors.Is(err, ErrKilled) {
		t.Fatalf("fail-safe after 30s: %v", err)
	}
	m.down = false
	_ = ch.Refresh(ctx)
	if ch.Check(Op{Agent: "ask", Consequential: true}) != nil || len(ch.Killed()) != 0 {
		t.Fatal("recovery")
	}
}

// SB-1008: space scopes are valid, stop only their space, and the drill fires each space.
func TestSpaceScope(t *testing.T) {
	for s, ok := range map[string]bool{"space:finance": true, "space:it-security": true, "space:Finance": false, "space:": false, "space:a": false} {
		if ValidScope(s) != ok {
			t.Errorf("ValidScope(%q) = %v", s, !ok)
		}
	}
	c := &Checker{Tenant: "acme-logistics", Now: func() time.Time { return time.Unix(0, 0) }}
	if !c.SpaceKilled("finance") {
		t.Fatal("before the first load every space counts as killed")
	}
	c.Push([]Switch{{Scope: "space:finance"}})
	if c.Check(Op{Agent: "ask", Space: "finance"}) == nil || c.Check(Op{Agent: "ask", Space: "eng"}) != nil || c.Check(Op{Agent: "ask"}) != nil {
		t.Fatal("a space kill must stop that space only")
	}
	if !c.SpaceKilled("finance") || c.SpaceKilled("eng") {
		t.Fatal("SpaceKilled")
	}
	c.Push([]Switch{{Scope: "tenant"}})
	if !c.SpaceKilled("eng") {
		t.Fatal("a killed tenant kills every space")
	}
	got := DrillScopes([]string{"ask"}, []string{"docs"}, []string{"finance", "people"})
	want := []string{"tenant", "agent:ask", "connector:docs", "space:finance", "space:people"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("DrillScopes = %v", got)
	}
}

// SB-4702: an agent paused in one space stops there only.
func TestAgentPausedInASpace(t *testing.T) {
	for s, ok := range map[string]bool{"agent:case-brief@support": true, "agent:ask@it-security": true, "agent:ask@": false, "agent:ask@Support": false, "agent:@support": false} {
		if ValidScope(s) != ok {
			t.Errorf("ValidScope(%q) = %v", s, !ok)
		}
	}
	c := &Checker{Tenant: "acme-logistics", Now: func() time.Time { return time.Unix(0, 0) }}
	if !c.Paused("case-brief", "support") {
		t.Fatal("before the first load every agent counts as paused")
	}
	c.Push([]Switch{{Scope: PauseScope("case-brief", "support")}, {Scope: PauseScope("case-brief", "sales")}, {Scope: "space:finance"}})
	if !c.Paused("case-brief", "support") || c.Paused("case-brief", "it") || c.Paused("ask", "support") {
		t.Fatal("Paused")
	}
	if c.Check(Op{Agent: "case-brief", Space: "support"}) == nil || c.Check(Op{Agent: "case-brief", Space: "it"}) != nil || c.Check(Op{Agent: "case-brief"}) != nil {
		t.Fatal("a pause must stop that agent in that space only")
	}
	if got := strings.Join(c.PausedIn("case-brief"), ","); got != "sales,support" || len(c.PausedIn("ask")) != 0 {
		t.Fatalf("PausedIn = %s", got)
	}
}
