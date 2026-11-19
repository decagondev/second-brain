package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/connectors/gdrive"
	"example.com/second-brain/internal/adapters/connectors/gdrive/drivetest"
	"example.com/second-brain/internal/adapters/connectors/linear"
	"example.com/second-brain/internal/adapters/connectors/linear/lineartest"
	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/connectors/slack"
	"example.com/second-brain/internal/adapters/connectors/slack/slacktest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/obs"
)

// Connector-level laws (SB-201): what a connector itself must guarantee, checked on the
// fault-injecting mock. LAW-03: under any fault, ACL is never Allow; with no fault it
// tells the truth. LAW-05: Act returns a target-issued receipt or an error, and the
// same key twice has one effect.

// ConnectorWorld drives one connector over a source the law controls.
type ConnectorWorld interface {
	// ACL asks for user's verdicts on n documents; access[i] is the truth; fault is
	// "", "500", "timeout", "404", or "script" (the first ref's verdict is scripted Unknown).
	ACL(access []bool, fault string) (verdicts []string, err error)
	// Act runs action key against a target that does or doesn't issue ids; it returns
	// the receipt id ("" if none) and the error.
	Act(key string, targetIssuesID bool) (receipt string, err error)
	// Effects is how many changes the target has applied.
	Effects() int
}

// NewConnectorWorld builds the system under test: the mock source and its writable
// connector (SB-201).
var NewConnectorWorld = func() ConnectorWorld {
	src := mock.NewSource("tickets")
	return &connWorld{src: src, c: mock.New(src, true)}
}

type connWorld struct {
	src *mock.Source
	c   *mock.Connector
	n   int
}

func (w *connWorld) ACL(access []bool, fault string) ([]string, error) {
	w.n++
	refs := make([]ports.SourceRef, len(access))
	for i, ok := range access {
		id := fmt.Sprintf("d%d_%d", w.n, i)
		if ok {
			w.src.Put(id, "t", "x", "ann@acme.example")
		} else {
			w.src.Put(id, "t", "x", "bob@acme.example")
		}
		refs[i] = ports.SourceRef{Kind: "tickets", ID: id}
	}
	if fault == "script" { // an unmapped permission model on the first ref only
		w.src.ScriptVerdict("ann@acme.example", refs[0].ID, ports.VerdictUnknown)
	} else {
		w.src.InjectFault(mock.Fault(fault))
	}
	vs, err := w.c.ACL(context.Background(), ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}, refs)
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.String()
	}
	return out, err
}

func (w *connWorld) Act(key string, issues bool) (string, error) {
	w.src.IssueIDs(issues)
	r, err := w.c.Act(context.Background(), ports.Action{Target: "tickets:OPS-1", Payload: []byte(`{}`)}, ports.IdempotencyKey(key))
	return r.TargetID, err
}

func (w *connWorld) Effects() int { return w.src.Effects() }

func law03Connector(r *rand.Rand, w ConnectorWorld) error {
	for i := 0; i < 6; i++ {
		access := make([]bool, 1+r.Intn(8))
		for j := range access {
			access[j] = r.Intn(2) == 0
		}
		fault := []string{"", "", "500", "timeout", "404", "script"}[r.Intn(6)]
		if fault == "timeout" && r.Intn(4) != 0 {
			fault = "500" // keep the slow case rare; it waits out the 800 ms deadline
		}
		got, err := w.ACL(access, fault)
		if err != nil {
			return err
		}
		if len(got) != len(access) {
			return fmt.Errorf("asked %d refs, got %d verdicts", len(access), len(got))
		}
		for j, v := range got {
			want := "deny"
			if access[j] {
				want = "allow"
			}
			if (fault != "" && fault != "script") || (fault == "script" && j == 0) {
				want = "unknown"
			}
			if v != want {
				return fmt.Errorf("fault %q, ref %d (access %v): verdict %s, want %s", fault, j, access[j], v, want)
			}
		}
	}
	return nil
}

func law05Connector(r *rand.Rand, w ConnectorWorld) error {
	receipts := map[string]string{}
	effects := 0
	for i := 0; i < 8; i++ {
		key := fmt.Sprintf("apr_%d", r.Intn(4))
		issues := r.Intn(3) != 0
		got, err := w.Act(key, issues)
		if errors.Is(err, ErrPending) {
			return err
		}
		prev, seen := receipts[key]
		switch {
		case err == nil && got == "":
			return fmt.Errorf("Act %s succeeded without a receipt", key)
		case err == nil && seen && got != prev:
			return fmt.Errorf("Act %s twice gave receipts %s and %s", key, prev, got)
		case err == nil && !seen:
			receipts[key] = got
			effects++
		}
		if w.Effects() != effects {
			return fmt.Errorf("after %d distinct successful keys the target shows %d effects", effects, w.Effects())
		}
	}
	return nil
}

// NewDriveWorld is the same connector-level world over the Google Drive connector
// (SB-203) and its recorded fake. Drive is read-only, so only ACL is exercised; a
// "script" fault trashes the first file, which the source can no longer answer for.
var NewDriveWorld = func() ConnectorWorld {
	f := drivetest.New()
	sa := &gdrive.ServiceAccount{Key: func(context.Context) (string, error) { return f.KeyJSON(), nil },
		HTTP: f.Server.Client(), Redact: obs.NewRedactor(), Now: time.Now}
	return &driveWorld{f: f, c: &gdrive.Connector{Base: f.Base(), HTTP: f.Server.Client(), Tokens: sa}}
}

type driveWorld struct {
	f *drivetest.Fake
	c *gdrive.Connector
	n int
}

func (w *driveWorld) ACL(access []bool, fault string) ([]string, error) {
	w.n++
	refs := make([]ports.SourceRef, len(access))
	for i, ok := range access {
		id := fmt.Sprintf("f%d_%d", w.n, i)
		viewer := "bob@acme.example"
		if ok {
			viewer = "ann@acme.example"
		}
		w.f.Put(id, "t", "application/vnd.google-apps.document", "x", viewer)
		refs[i] = ports.SourceRef{Kind: "docs", ID: id}
	}
	switch fault {
	case "script":
		w.f.Trash(refs[0].ID)
	case "":
	default:
		for range refs {
			w.f.FailNext(fault)
		}
	}
	vs, err := w.c.ACL(context.Background(), ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}, refs)
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.String()
	}
	return out, err
}

func (w *driveWorld) Act(string, bool) (string, error) { return "", errors.New("ENOTSUP") }
func (w *driveWorld) Effects() int                     { return 0 }

// NewSlackWorld is the connector-level world over the Slack connector (SB-204) and its
// fake. Access means the asker is in the private channel; "script" deletes the first
// message, which the source can then no longer answer for.
var NewSlackWorld = func() ConnectorWorld {
	f := slacktest.New()
	f.AddUser("U1", "ann@acme.example", false)
	f.AddUser("U2", "bob@acme.example", false)
	f.AddChannel("CIN", "in", true, true, "U1")
	f.AddChannel("COUT", "out", true, true, "U2")
	c := &slack.Connector{Base: f.Base(), HTTP: f.Server.Client(), Redact: obs.NewRedactor(), Inbox: &slack.MemoryInbox{},
		Creds: func(context.Context) (string, error) { return f.CredsJSON(), nil }}
	return &slackWorld{f: f, c: c}
}

type slackWorld struct {
	f *slacktest.Fake
	c *slack.Connector
}

func (w *slackWorld) ACL(access []bool, fault string) ([]string, error) {
	refs := make([]ports.SourceRef, len(access))
	var first [2]string
	for i, ok := range access {
		ch := "COUT"
		if ok {
			ch = "CIN"
		}
		ts, _ := w.f.Post(ch, "U2", "", "x")
		if i == 0 {
			first = [2]string{ch, ts}
		}
		refs[i] = ports.SourceRef{Kind: "chat", ID: ch + "/" + ts + "/" + ts}
	}
	switch fault {
	case "script":
		w.f.Delete(first[0], first[1])
	case "404": // the source can't find any channel in the batch: one fault per distinct channel
		seen := map[string]bool{}
		for _, r := range refs {
			ch := strings.SplitN(r.ID, "/", 2)[0]
			if !seen[ch] {
				seen[ch] = true
				w.f.FailNext("404")
			}
		}
	case "":
	default:
		w.f.FailNext(fault)
	}
	vs, err := w.c.ACL(context.Background(), ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}, refs)
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.String()
	}
	return out, err
}

func (w *slackWorld) Act(string, bool) (string, error) { return "", errors.New("ENOTSUP") }
func (w *slackWorld) Effects() int                     { return 0 }

// NewLinearWorld is the connector-level world over the Linear connector (SB-211) and its
// fake. Access means the asker is in the private team; "script" archives the first issue.
var NewLinearWorld = func() ConnectorWorld {
	f := lineartest.New()
	f.AddUser("U1", "ann@acme.example", false)
	f.AddUser("U2", "bob@acme.example", false)
	f.AddTeam("TIN", "IN", true, "U1")
	f.AddTeam("TOUT", "OUT", true, "U2")
	f.AddTeam("TOPS", "OPS", false, "U1")
	f.Put("iops", "TOPS", "Disk full", "the target of LAW-05's comments")
	c := &linear.Connector{Endpoint: f.Endpoint(), HTTP: f.Server.Client(), Redact: obs.NewRedactor(),
		Key: func(context.Context) (string, error) { return f.Key, nil }}
	return &linearWorld{f: f, c: c}
}

type linearWorld struct {
	f *lineartest.Fake
	c *linear.Connector
	n int
}

func (w *linearWorld) ACL(access []bool, fault string) ([]string, error) {
	w.n++
	refs := make([]ports.SourceRef, len(access))
	for i, ok := range access {
		team := "TOUT"
		if ok {
			team = "TIN"
		}
		id := fmt.Sprintf("i%d_%d", w.n, i)
		w.f.Put(id, team, "t", "x")
		refs[i] = ports.SourceRef{Kind: "tickets", ID: id}
	}
	switch fault {
	case "script":
		w.f.Archive(refs[0].ID)
	case "404": // every issue lookup fails: one fault per ref
		for range refs {
			w.f.FailNext("404")
		}
	case "":
	default:
		w.f.FailNext(fault)
	}
	vs, err := w.c.ACL(context.Background(), ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}, refs)
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.String()
	}
	return out, err
}

// Act posts a comment (SB-410). A target that "doesn't issue ids" answers without one:
// half the time the comment was made (the connector must read it back), half the time
// nothing was made (the connector must fail, never claim a receipt).
func (w *linearWorld) Act(key string, issues bool) (string, error) {
	if !issues {
		w.n++
		w.f.FailNext([]string{"noid", "noop"}[w.n%2])
	}
	r, err := w.c.Act(context.Background(), ports.Action{Target: "tickets:OPS-1", Payload: []byte(`{"comment":"law-05"}`)}, ports.IdempotencyKey(key))
	return r.TargetID, err
}
func (w *linearWorld) Effects() int { return len(w.f.Comments) }
