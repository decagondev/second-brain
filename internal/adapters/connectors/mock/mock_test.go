package mock

import (
	"context"
	"fmt"
	"testing"

	"example.com/second-brain/internal/core/connectorkit"
	"example.com/second-brain/internal/core/ports"
)

type sandbox struct {
	src      *Source
	c        *Connector
	writable bool
}

func newSandbox(writable bool) *sandbox {
	src := NewSource("docs")
	src.Put("d0", "Bonus policy", "text", "ann@acme.example")
	for i := 1; i < 200; i++ {
		src.Put(fmt.Sprintf("d%d", i), "doc", "text", "ann@acme.example")
	}
	return &sandbox{src: src, c: New(src, writable), writable: writable}
}

func (s *sandbox) Connector() ports.Connector { return s.c }
func (s *sandbox) Doc() ports.SourceRef       { return ports.SourceRef{Kind: "docs", ID: "d0"} }
func (s *sandbox) Allowed() ports.UserRef {
	return ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}
}
func (s *sandbox) Denied() ports.UserRef {
	return ports.UserRef{ID: "usr_bob", Email: "bob@acme.example"}
}
func (s *sandbox) Refs(n int) []ports.SourceRef {
	out := make([]ports.SourceRef, n)
	for i := range out {
		out[i] = ports.SourceRef{Kind: "docs", ID: fmt.Sprintf("d%d", i)}
	}
	return out
}
func (s *sandbox) InjectACLFault(kind string) { s.src.InjectFault(Fault(kind)) }
func (s *sandbox) Edit(ref ports.SourceRef)   { s.src.Put(ref.ID, "Bonus policy", "new text", nil...) }
func (s *sandbox) ChangePermissions(ref ports.SourceRef) {
	s.src.SetViewers(ref.ID, "ann@acme.example", "cy@acme.example")
}
func (s *sandbox) Action() (ports.Action, bool) {
	return ports.Action{Target: "tickets:OPS-1", Payload: []byte(`{"comment":"x"}`)}, s.writable
}
func (s *sandbox) Effects() int { return s.src.Effects() }

func TestKit(t *testing.T) {
	t.Run("read-only", func(t *testing.T) { connectorkit.Run(t, newSandbox(false)) })
	t.Run("writable", func(t *testing.T) { connectorkit.Run(t, newSandbox(true)) })
}

func TestNegotiation(t *testing.T) {
	if err := ports.Negotiate(old{New(NewSource("docs"), false)}); err == nil {
		t.Fatal("a connector for another contract version must be refused")
	}
}

type old struct{ *Connector }

func (old) APIVersion() int { return ports.ConnectorAPIVersion + 1 }

// Every behaviour the mock offers, asked for on demand (sb-connector(3), MOCK CONNECTOR).
func TestFaultMatrix(t *testing.T) {
	ann := ports.UserRef{Email: "ann@acme.example"}
	ref := []ports.SourceRef{{Kind: "docs", ID: "d"}}
	verdict := func(src *Source) ports.Verdict {
		vs, err := New(src, false).ACL(context.Background(), ann, ref)
		if err != nil || len(vs) != 1 {
			return ports.VerdictUnknown
		}
		return vs[0]
	}
	fresh := func(viewers ...string) *Source { s := NewSource("docs"); s.Put("d", "t", "x", viewers...); return s }
	cases := []struct {
		name  string
		setup func() *Source
		want  ports.Verdict
	}{
		{"allow", func() *Source { return fresh("ann@acme.example") }, ports.VerdictAllow},
		{"deny", func() *Source { return fresh("bob@acme.example") }, ports.VerdictDeny},
		{"scripted unknown", func() *Source {
			s := fresh("ann@acme.example")
			s.ScriptVerdict("ann@acme.example", "d", ports.VerdictUnknown)
			return s
		}, ports.VerdictUnknown},
		{"500", func() *Source { s := fresh("ann@acme.example"); s.QueueFaults(Fault500); return s }, ports.VerdictUnknown},
		{"timeout", func() *Source { s := fresh("ann@acme.example"); s.QueueFaults(FaultTimeout); return s }, ports.VerdictUnknown},
		{"404", func() *Source { s := fresh("ann@acme.example"); s.QueueFaults(FaultNotFound); return s }, ports.VerdictUnknown},
		{"deleted", func() *Source { s := fresh("ann@acme.example"); s.Delete("d"); return s }, ports.VerdictUnknown},
	}
	for _, c := range cases {
		if got := verdict(c.setup()); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
	// A queue applies one fault per call, then the truth again.
	s := fresh("ann@acme.example")
	s.QueueFaults(Fault500, FaultNotFound)
	for i, want := range []ports.Verdict{ports.VerdictUnknown, ports.VerdictUnknown, ports.VerdictAllow} {
		if got := verdict(s); got != want {
			t.Errorf("call %d: %v, want %v", i+1, got, want)
		}
	}
	// No receipt: an error, and no effect.
	s = fresh()
	s.IssueIDs(false)
	if r, err := New(s, true).Act(context.Background(), ports.Action{Target: "t"}, "k"); err == nil || r.TargetID != "" || s.Effects() != 0 {
		t.Fatalf("no-receipt target: %v %v effects %d", r, err, s.Effects())
	}
}
