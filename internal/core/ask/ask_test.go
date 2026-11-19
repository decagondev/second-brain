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
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

const tenant = "acme-logistics"

type allowAll struct{}

func (allowAll) Evaluate(context.Context, policy.Request) (policy.Result, error) {
	return policy.Result{Decision: policy.Allow}, nil
}

type rec struct{}

func (rec) RecordDecision(context.Context, policy.Record) error { return nil }

// index returns every chunk as a candidate (the prefilter is the index's job, tested there).
type index []ports.Chunk

func (ix index) Search(context.Context, tenancy.Scope, []string, ports.Query) ([]ports.Candidate, error) {
	var out []ports.Candidate
	for i, c := range ix {
		out = append(out, ports.Candidate{Chunk: c, KeywordRank: i + 1})
	}
	return out, nil
}

// source answers ACL from a fixed table; anything else is Unknown.
type source map[string]ports.Verdict

func (source) APIVersion() int { return ports.ConnectorAPIVersion }
func (source) Fetch(context.Context, ports.SourceRef) (ports.Document, error) {
	return ports.Document{}, errors.New("unused")
}
func (s source) ACL(_ context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs))
	for i, r := range refs {
		out[i] = s[r.ID+"|"+u.ID]
	}
	return out, nil
}
func (source) Changes(context.Context, ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	return nil, "", nil
}
func (source) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, errors.New("unused")
}

type store struct {
	traces map[string]ask.Trace
	convs  map[string][]string // "user|cv" -> questions
}

func (s *store) PutTrace(_ context.Context, t ask.Trace) error { s.traces[t.ID] = t; return nil }
func (s *store) GetTrace(_ context.Context, _, id string) (ask.Trace, bool, error) {
	t, ok := s.traces[id]
	return t, ok, nil
}
func (s *store) Questions(_ context.Context, _, user, cv string) ([]string, bool, error) {
	q, ok := s.convs[user+"|"+cv]
	return q, ok, nil
}
func (s *store) AppendQuestion(_ context.Context, _, user, cv, q string) error {
	s.convs[user+"|"+cv] = append(s.convs[user+"|"+cv], q)
	return nil
}

type model struct {
	reply string
	err   error
	got   []ports.ModelRequest
}

func (m *model) Complete(_ context.Context, r ports.ModelRequest) (ports.ModelResponse, error) {
	m.got = append(m.got, r)
	return ports.ModelResponse{Text: m.reply}, m.err
}

func scope(t *testing.T, user string) tenancy.Scope {
	s, err := tenancy.NewIssuer(tenant).Issue(tenant, user, nil)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// setup: two docs, refunds (alice may read) and salaries (nobody may).
func setup(t *testing.T, m ports.Model) (ask.Agent, *[]string) {
	var idx index
	for _, d := range []struct{ id, text string }{{"refunds", "refund limit is 500"}, {"salaries", "refund salaries secret"}} {
		idx = append(idx, ports.Chunk{ID: "chk_" + d.id, Tenant: tenant, Kind: "docs", SourceID: d.id, Version: 2, Cite: "docs:" + d.id + "@2", Text: d.text})
	}
	src := source{"refunds|alice": ports.VerdictAllow, "salaries|alice": ports.VerdictDeny}
	var spoofs []string
	n := 0
	a := ask.Agent{Tenant: tenant, Release: "rel_test", Index: idx,
		Auth:   retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": src}, Policy: policy.Kernel{Engine: allowAll{}, Recorder: rec{}}},
		Model:  m,
		Store:  &store{traces: map[string]ask.Trace{}, convs: map[string][]string{}},
		Killed: func(context.Context) bool { return false },
		Spoof:  func(_ context.Context, tr, l string) { spoofs = append(spoofs, tr+" "+l) },
		NewID:  func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) },
		Now:    time.Now}
	return a, &spoofs
}

func TestAskCitesOnlyAllowed(t *testing.T) {
	m := &model{reply: `{"claims":[{"text":"Limit is 500.","cites":["C1","C7"]},{"text":"Ask finance.","cites":[]}]}`}
	a, spoofs := setup(t, m)
	resp, err := a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "refund limit?"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Refused != 1 || resp.Release != "rel_test" || !strings.HasPrefix(resp.Trace, "tr_") || !strings.HasPrefix(resp.Conversation, "cv_") {
		t.Fatalf("%+v", resp)
	}
	c := resp.Answer.Claims
	if len(c) != 2 || len(c[0].Cites) != 1 || c[0].Cites[0].Cite != "docs:refunds@2" || !c[1].Uncited {
		t.Fatalf("claims %+v", c)
	}
	if resp.Notice != nil {
		t.Errorf("notice %v, want none (half uncited is not most)", *resp.Notice)
	}
	if len(*spoofs) != 1 || !strings.HasSuffix((*spoofs)[0], " C7") {
		t.Errorf("spoofs %v", *spoofs)
	}
	if strings.Contains(m.got[0].Messages[0].Content, "salaries") {
		t.Error("a refused document reached the model")
	}
	tr, err := a.Trace(context.Background(), scope(t, "alice"), resp.Trace)
	if err != nil || len(tr.Decisions) != 2 {
		t.Fatalf("trace %+v %v", tr, err)
	}
	if _, err := a.Trace(context.Background(), scope(t, "bob"), resp.Trace); !errors.Is(err, ask.ErrNotFound) {
		t.Errorf("another user read the trace: %v", err)
	}
}

func TestNoContextSkipsTheModel(t *testing.T) {
	m := &model{}
	a, _ := setup(t, m)
	resp, err := a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "salaries", Sources: []string{"tickets"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.got) != 0 || resp.Notice == nil || *resp.Notice != ask.NoticeNoSources || !resp.Answer.Claims[0].Uncited {
		t.Fatalf("%+v (model calls %d)", resp, len(m.got))
	}
	resp, err = a.Ask(context.Background(), scope(t, "carol"), ask.Request{Question: "refund"})
	if err != nil || resp.Notice == nil || *resp.Notice != ask.NoticeNoAccess || len(m.got) != 0 {
		t.Fatalf("carol: %+v %v", resp, err)
	}
}

func TestValidation(t *testing.T) {
	a, _ := setup(t, &model{})
	for _, r := range []ask.Request{{Question: " "}, {Question: strings.Repeat("x", ask.MaxQuestion+1)}, {Question: "q", Sources: []string{"email"}}, {Question: "q", Conversation: "cv_nope"}} {
		if _, err := a.Ask(context.Background(), scope(t, "alice"), r); !errors.Is(err, ask.ErrInvalid) {
			t.Errorf("%+v: got %v, want EINVAL", r, err)
		}
	}
	if _, err := a.Ask(context.Background(), tenancy.Scope{}, ask.Request{Question: "q"}); !errors.Is(err, ask.ErrInvalid) {
		t.Errorf("zero scope: %v", err)
	}
	a.Killed = nil
	if _, err := a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "q"}); !errors.Is(err, ask.ErrKilled) {
		t.Errorf("no kill reader: %v, want EKILLED", err)
	}
}

func TestConversationSendsOnlyEarlierQuestions(t *testing.T) {
	m := &model{reply: `{"claims":[{"text":"Limit is 500.","cites":["C1"]}]}`}
	a, _ := setup(t, m)
	ctx := context.Background()
	r1, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund limit?"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Ask(ctx, scope(t, "bob"), ask.Request{Question: "and?", Conversation: r1.Conversation}); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("bob continued alice's conversation: %v", err)
	}
	r2, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund and since when?", Conversation: r1.Conversation})
	if err != nil || r2.Conversation != r1.Conversation {
		t.Fatal(err)
	}
	msgs := m.got[1].Messages
	if msgs[0].Content != "refund limit?" || strings.Contains(msgs[1].Content, "500") || strings.Contains(msgs[1].Content, "refund limit is") {
		t.Errorf("earlier turn carried more than the question: %+v", msgs[:2])
	}
}

func TestModelDownIsUnavailable(t *testing.T) {
	a, _ := setup(t, &model{err: errors.New("boom")})
	if _, err := a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "refund"}); !errors.Is(err, ask.ErrUnavailable) {
		t.Fatalf("got %v", err)
	}
}

// BUG-0007: the live check must reach the connector with the asker's email, because
// real connectors ask their source by email.
type emailSource struct {
	source
	got *[]ports.UserRef
}

func (e emailSource) ACL(ctx context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	*e.got = append(*e.got, u)
	return e.source.ACL(ctx, u, refs)
}

func TestLiveCheckCarriesTheEmail(t *testing.T) {
	a, _ := setup(t, &model{reply: `{"claims":[]}`})
	var got []ports.UserRef
	a.Auth.Connectors["docs"] = emailSource{source: source{}, got: &got}
	a.Emails = func(context.Context, string) ([]string, error) {
		return []string{"alice@acme.example", "a.l@acme.example"}, nil
	}
	if _, err := a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "refund"}); err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 || got[0].Email != "alice@acme.example" || got[0].ID != "alice" {
		t.Fatalf("ACL asked about %+v, want alice with her email", got)
	}
}
