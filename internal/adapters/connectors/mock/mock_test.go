package mock

import (
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
