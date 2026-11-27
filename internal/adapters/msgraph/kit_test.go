package msgraph

import (
	"fmt"
	"testing"

	"example.com/second-brain/internal/adapters/msgraph/graphtest"
	"example.com/second-brain/internal/core/connectorkit"
	"example.com/second-brain/internal/core/ports"
)

// kitSandbox is the connector contract kit's sandbox over the recorded Graph (SB-5901).
type kitSandbox struct {
	f  *graphtest.Fake
	sp *SharePoint
}

func (s kitSandbox) Connector() ports.Connector { return s.sp }
func (s kitSandbox) Doc() ports.SourceRef {
	return ports.SourceRef{Kind: "docs", ID: spID("d1", "bonus")}
}
func (s kitSandbox) Allowed() ports.UserRef {
	return ports.UserRef{ID: "usr_a", Email: "ann@acme.example"}
}
func (s kitSandbox) Denied() ports.UserRef {
	return ports.UserRef{ID: "usr_b", Email: "bob@acme.example"}
}
func (s kitSandbox) Refs(n int) []ports.SourceRef {
	out := make([]ports.SourceRef, n)
	for i := range out {
		out[i] = ports.SourceRef{Kind: "docs", ID: spID("d1", fmt.Sprintf("f%d", i%200))}
	}
	return out
}
func (s kitSandbox) InjectACLFault(kind string) { s.f.FailNext(kind) }
func (s kitSandbox) Edit(ref ports.SourceRef) {
	s.f.Put("d1", graphtest.Item{ID: "bonus", Name: "bonus.md", Path: "/HR", Content: []byte("edited text"), Users: []string{"ann@acme.example"}})
}
func (s kitSandbox) ChangePermissions(ref ports.SourceRef) {
	s.f.Share("d1", "bonus", "ann@acme.example", "cy@acme.example")
}
func (s kitSandbox) Action() (ports.Action, bool) { return ports.Action{}, false }
func (s kitSandbox) Effects() int                 { return 0 }

func TestContractKit(t *testing.T) {
	f, g := world(t)
	f.Put("d1", graphtest.Item{ID: "bonus", Name: "bonus.md", Path: "/HR", Content: []byte("Bonus policy"), Users: []string{"ann@acme.example"}})
	for i := 0; i < 200; i++ {
		f.Put("d1", graphtest.Item{ID: fmt.Sprintf("f%d", i), Name: fmt.Sprintf("doc-%d.md", i), Content: []byte("text"), Users: []string{"ann@acme.example"}})
	}
	connectorkit.Run(t, kitSandbox{f: f, sp: &SharePoint{Graph: g, Drives: []string{"d1"}, Domain: "acme.example"}})
}
