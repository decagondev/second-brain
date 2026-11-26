package ask_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

type poster struct{ got []string }

func (p *poster) Post(_ context.Context, channel, text string) (string, error) {
	p.got = append(p.got, channel+"|"+text)
	return "m1", nil
}

// SB-5304: a draft comes from the person's own answer, with only cites they can open; a
// message goes to one of the space's channels, signed, after its audit row.
func TestDraftTheNextAction(t *testing.T) {
	ctx := context.Background()
	a, _ := setup(t, &model{reply: `{"claims":[{"text":"Limit is 500.","cites":["C1"]},{"text":"Ask finance.","cites":[]}]}`})
	a.Emails = func(_ context.Context, u string) ([]string, error) { return []string{u + "@acme.example"}, nil }
	alice := scope(t, "alice")
	resp, err := a.Ask(ctx, alice, ask.Request{Question: "refund limit?"})
	if err != nil {
		t.Fatal(err)
	}
	d, err := a.DraftAction(ctx, alice, resp.Trace, ask.ActionTicket)
	if err != nil || !strings.Contains(d.Text, "- Limit is 500.") || strings.Contains(d.Text, "Ask finance") || len(d.Cites) != 1 {
		t.Fatalf("draft %+v %v", d, err)
	}
	if _, err := a.DraftAction(ctx, scope(t, "bob"), resp.Trace, ask.ActionTicket); err == nil {
		t.Fatal("someone else drafted from alice's answer")
	}
	if _, err := a.DraftAction(ctx, alice, resp.Trace, "email"); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("an unknown kind: %v", err)
	}
	inOps, err := tenancy.NewIssuer(tenant).Issue(tenant, "alice", []string{"ops"})
	if err != nil {
		t.Fatal(err)
	}
	a.Auth.Spaces = func(context.Context) ([]ports.Space, [][2]string, error) {
		return []ports.Space{{Name: "ops", Tenant: tenant, Members: []string{"group:ops"}}}, nil, nil
	}
	rows := &rows{}
	n := 0
	x := ask.Stewards{Agent: a, Store: &stq{m: map[string]ask.StewardQuestion{}}, Audit: rows,
		Policy:  policy.Kernel{Engine: allowAll{}, Recorder: rec{}},
		Facts:   func(string) ([]string, string, bool) { return []string{"group:ops"}, "", true },
		Members: func(context.Context, string) ([]string, error) { return []string{"alice"}, nil },
		NewID:   func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }, Now: time.Now}
	p := &poster{}
	channels := func(string) []string { return []string{"#ops"} }
	if _, err := x.SendMessage(ctx, alice, "ops", "#ops", d.Text, p, channels); !errors.Is(err, ask.ErrNotMember) {
		t.Fatalf("a non-member sent: %v", err)
	}
	if _, err := x.SendMessage(ctx, inOps, "ops", "#random", d.Text, p, channels); !errors.Is(err, ask.ErrInvalid) || len(p.got) != 0 {
		t.Fatalf("a channel that isn't the space's: %v %v", err, p.got)
	}
	m, err := x.SendMessage(ctx, inOps, "ops", "#ops", d.Text, p, channels)
	if err != nil || m.ID != "m1" || len(p.got) != 1 || !strings.HasPrefix(p.got[0], "#ops|Sent by alice@acme.example via Second Brain:\n- Limit is 500.") {
		t.Fatalf("send %+v %v %v", m, err, p.got)
	}
}
