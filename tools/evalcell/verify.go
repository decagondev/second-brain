package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/verify"
)

// target is a local cell as a verify.Target (sb-tenant(1), How verify decides). The
// world's foreign_ids stand in for the registry's other-tenant document ids.
type target struct {
	c       *cell
	foreign []string
	// leaky is a mutation for the verify tests: the cell answers with a citation the asker
	// can't open.
	leaky bool
}

func (t *target) Users(context.Context) ([]string, error) {
	var out []string
	for _, u := range t.c.w.Users {
		out = append(out, u.ID)
	}
	sort.Strings(out)
	return out, nil
}

func (t *target) Questions(context.Context) ([]string, error) {
	var out []string
	for _, d := range t.c.w.Documents {
		out = append(out, d.Title)
	}
	return out, nil
}

func (t *target) Ask(ctx context.Context, user, q string) (verify.Answer, error) {
	i, ok := t.c.users[user]
	if !ok {
		return verify.Answer{}, fmt.Errorf("unknown user")
	}
	s, err := tenancy.NewIssuer(t.c.w.Tenant).Issue(t.c.w.Tenant, user, t.c.w.Users[i].Groups)
	if err != nil {
		return verify.Answer{}, err
	}
	resp, err := t.c.agent.Ask(ctx, s, ask.Request{Question: q})
	if err != nil {
		return verify.Answer{}, err
	}
	tr, err := t.c.agent.Trace(ctx, s, resp.Trace)
	if err != nil {
		return verify.Answer{}, err
	}
	a := verify.Answer{LiveChecked: map[string]bool{}}
	for _, d := range tr.Decisions {
		if d.Reason == "allowed" {
			a.LiveChecked[docKey(d.Cite)] = true
		}
	}
	for _, cl := range resp.Answer.Claims {
		switch {
		case cl.Uncited && len(cl.Cites) == 0:
			a.Uncited++
		case !cl.Uncited && len(cl.Cites) > 0:
			a.Cited++
		default:
			a.Unlabelled++
		}
		for _, ci := range cl.Cites {
			a.Cites = append(a.Cites, verify.Cite{Doc: docKey(ci.Cite), Connector: "docs"})
		}
	}
	if t.leaky && len(t.c.w.Documents) > 0 {
		a.Cites = append(a.Cites, verify.Cite{Doc: "docs:" + t.c.w.Documents[len(t.c.w.Documents)-1].ID, Connector: "docs"})
	}
	return a, nil
}

func docKey(cite string) string {
	if i := strings.LastIndex(cite, "@"); i >= 0 {
		return cite[:i]
	}
	return cite
}

func (t *target) Allowed(ctx context.Context, user, doc string) (bool, error) {
	i := t.c.users[user]
	vs, err := t.c.conn.ACL(ctx, ports.UserRef{ID: user, Email: t.c.w.Users[i].Email}, []ports.SourceRef{{Kind: "docs", ID: strings.TrimPrefix(doc, "docs:")}})
	if err != nil || len(vs) != 1 {
		return false, err
	}
	return vs[0] == ports.VerdictAllow, nil
}

func (t *target) ForeignIDs(context.Context) ([]string, error) { return t.foreign, nil }

func (t *target) Timeout(string) (func(), error) {
	t.c.src.QueueFaults(mock.FaultTimeout, mock.FaultTimeout, mock.FaultTimeout, mock.FaultTimeout, mock.FaultTimeout,
		mock.FaultTimeout, mock.FaultTimeout, mock.FaultTimeout, mock.FaultTimeout, mock.FaultTimeout)
	return func() { t.c.src.InjectFault(mock.FaultNone) }, nil
}

func (t *target) Connectors() []string { return []string{"docs"} }

// verifyInput is --verify's stdin: a world, the policies, and other tenants' ids.
type verifyInput struct {
	World      world    `json:"world"`
	Policies   string   `json:"policies"`
	ForeignIDs []string `json:"foreign_ids"`
	Sample     int      `json:"sample"`
	Seed       int64    `json:"seed"`
}

// verifyMain runs sb tenant verify's checks on a local cell and prints the report.
func verifyMain() int {
	var in verifyInput
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
		fmt.Fprintln(os.Stderr, "evalcell --verify: bad input:", err)
		return 2
	}
	rep, err := verifyWorld(context.Background(), in, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "evalcell --verify:", err)
		return 7
	}
	fmt.Fprint(os.Stdout, rep.String())
	if !rep.Passed() {
		return 1
	}
	return 0
}

func verifyWorld(ctx context.Context, in verifyInput, leaky bool) (verify.Report, error) {
	c, err := build(ctx, in.World, "extractive", in.Policies)
	if err != nil {
		return verify.Report{}, err
	}
	return verify.Run(ctx, &target{c: c, foreign: in.ForeignIDs, leaky: leaky}, verify.Options{Sample: in.Sample, Seed: in.Seed}), nil
}
