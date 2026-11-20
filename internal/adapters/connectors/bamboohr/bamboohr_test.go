package bamboohr

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/connectors/bamboohr/bamboohrtest"
	"example.com/second-brain/internal/core/connectorkit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/obs"
)

func setup(t *testing.T) (*bamboohrtest.Fake, *Connector) {
	t.Helper()
	f := bamboohrtest.New()
	t.Cleanup(f.Server.Close)
	f.PutEmployee("1", map[string]string{"displayName": "Ann", "workEmail": "ann@acme.example", "department": "Ops", "payRate": "PAY-ANN"}, true)
	f.PutEmployee("2", map[string]string{"displayName": "Bob", "workEmail": "bob@acme.example", "department": "Ops"}, false)
	return f, &Connector{Base: f.Base(), HTTP: f.Server.Client(), Redact: obs.NewRedactor(),
		Creds: func(context.Context) (string, error) { return f.CredsJSON(), nil }}
}

type sandbox struct {
	f *bamboohrtest.Fake
	c *Connector
}

func (s sandbox) Connector() ports.Connector { return s.c }
func (s sandbox) Doc() ports.SourceRef       { return ports.SourceRef{Kind: "hris", ID: "files/f1"} }
func (s sandbox) Allowed() ports.UserRef {
	return ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}
}
func (s sandbox) Denied() ports.UserRef {
	return ports.UserRef{ID: "usr_bob", Email: "bob@acme.example"}
}
func (s sandbox) Refs(n int) []ports.SourceRef {
	out := make([]ports.SourceRef, n)
	for i := range out {
		out[i] = ports.SourceRef{Kind: "hris", ID: fmt.Sprintf("files/f%d", i+1)}
	}
	return out
}
func (s sandbox) InjectACLFault(kind string) { s.f.FailNext(kind) }
func (s sandbox) Edit(ports.SourceRef) {
	s.f.PutFile("f1", "Handbook", "Policies", "Holidays are 25 days, edited.", true)
}
func (s sandbox) ChangePermissions(ports.SourceRef) { s.f.Unshare("f1") }
func (s sandbox) Action() (ports.Action, bool)      { return ports.Action{}, false }
func (s sandbox) Effects() int                      { return 0 }

func TestKit(t *testing.T) {
	f, c := setup(t)
	for i := 1; i <= 200; i++ {
		f.PutFile(fmt.Sprintf("f%d", i), fmt.Sprintf("Policy %d", i), "Policies", "Holidays are 25 days.", true)
	}
	connectorkit.Run(t, sandbox{f, c})
}

// SB-1104: sensitive values are never even read unless a space allow-lists the field, and
// the record carries only declared fields.
func TestSensitiveFieldsAreNotRead(t *testing.T) {
	f, c := setup(t)
	ctx := context.Background()
	d, err := c.Fetch(ctx, ports.SourceRef{Kind: "hris", ID: "employees/1"})
	if err != nil || d.Fields["displayName"] != "Ann" || d.Location != "employees/Ops/1" {
		t.Fatalf("%+v %v", d, err)
	}
	if _, ok := d.Fields["payRate"]; ok || f.Asked["payRate"] != 0 || f.Asked["ssn"] != 0 {
		t.Fatalf("a sensitive field was asked for or returned: %v %v", d.Fields, f.Asked)
	}
	c.Allowed = func() []string { return []string{"payRate"} }
	d, _ = c.Fetch(ctx, ports.SourceRef{Kind: "hris", ID: "employees/1"})
	if d.Fields["payRate"] != "PAY-ANN" || f.Asked["ssn"] != 0 {
		t.Fatalf("allow-listed payRate: %v asked %v", d.Fields, f.Asked)
	}
	fs, ok := c.Fields()
	if !ok || len(fs) != len(Declared) {
		t.Fatal("the connector must declare its fields")
	}
}

func TestEmployeeChanges(t *testing.T) {
	f, c := setup(t)
	ctx := context.Background()
	set, cur, err := c.Changes(ctx, "")
	if err != nil || len(set) != 2 {
		t.Fatalf("%v %v", set, err)
	}
	f.PutEmployee("3", map[string]string{"displayName": "Cy"}, true)
	f.DeleteEmployee("2")
	set, next, err := c.Changes(ctx, cur)
	got := fmt.Sprint(set)
	if err != nil || !strings.Contains(got, "employees/3") || !strings.Contains(got, "{hris employees/2} true") {
		t.Fatalf("changes: %s %v", got, err)
	}
	if rest, _, _ := c.Changes(ctx, next); len(rest) != 0 {
		t.Fatalf("after the latest cursor: %v", rest)
	}
	vs, _ := c.ACL(ctx, ports.UserRef{Email: "ann@acme.example"}, []ports.SourceRef{{Kind: "hris", ID: "employees/2"}, {Kind: "hris", ID: "employees/3"}})
	if vs[0] != ports.VerdictDeny || vs[1] != ports.VerdictAllow {
		t.Fatalf("a deleted record must be denied: %v", vs)
	}
}

// An employee change in the same second as the last sync is still reported, once.
func TestSameSecondChangeIsNotLost(t *testing.T) {
	f, c := setup(t)
	ctx := context.Background()
	_, cur, err := c.Changes(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	f.Frozen = true
	f.PutEmployee("9", map[string]string{"displayName": "Cy", "workEmail": "cy@acme.example", "department": "Ops"}, true)
	set, cur, err := c.Changes(ctx, cur)
	n := 0
	for _, ch := range set {
		if ch.Ref.ID == "employees/9" {
			n++
		}
	}
	if err != nil || n != 1 {
		t.Fatalf("same-second change: %+v %v", set, err)
	}
	set, _, _ = c.Changes(ctx, cur)
	for _, ch := range set {
		if ch.Ref.ID == "employees/9" {
			t.Fatal("reported twice")
		}
	}
}
