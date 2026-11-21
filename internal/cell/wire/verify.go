package wire

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync/atomic"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/verify"
)

// cellTarget is the running cell as the verifier sees it (sb-tenant(1), How verify
// decides; SB-1508): its directory's users, questions from its index, its own ask path,
// and a live ACL check at the source. It asks through a copy of the agent whose
// connectors can be made to time out, so LAW-03's injected timeout never touches anyone
// else's answers.
type cellTarget struct {
	c       *Cell
	foreign []string
	agent   ask.Agent
	down    map[string]*atomic.Bool
}

var _ verify.Target = (*cellTarget)(nil)

// errTimeout is the injected connector timeout.
var errTimeout = errors.New("verify: injected timeout")

type switchable struct {
	ports.Connector
	down *atomic.Bool
}

func (s switchable) Fetch(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
	if s.down.Load() {
		return ports.Document{}, errTimeout
	}
	return s.Connector.Fetch(ctx, ref)
}

func (s switchable) ACL(ctx context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	if s.down.Load() {
		return nil, errTimeout
	}
	return s.Connector.ACL(ctx, u, refs)
}

// VerifyTarget is the verifier's target over the parts the cell runs now, with the
// control plane's other-tenant document ids (ids only).
func (c *Cell) VerifyTarget(foreign []string) verify.Target {
	p := c.Parts()
	t := &cellTarget{c: c, foreign: foreign, agent: p.Agent, down: map[string]*atomic.Bool{}}
	conns := map[string]ports.Connector{}
	for k, conn := range p.Connectors {
		t.down[k] = &atomic.Bool{}
		conns[k] = switchable{Connector: conn, down: t.down[k]}
	}
	t.agent.Auth.Connectors = conns
	return t
}

func (t *cellTarget) tenant() string { return t.c.Config.Tenant }

// Users are the directory's active users.
func (t *cellTarget) Users(ctx context.Context) ([]string, error) {
	return t.c.Directory.ActiveUserIDs(ctx, t.tenant())
}

// Questions are the opening words of each indexed document's first chunk: the verifier
// asks about what is there, and prints none of it.
func (t *cellTarget) Questions(ctx context.Context) ([]string, error) {
	chunks, err := t.c.Index.All(ctx, t.tenant())
	if err != nil {
		return nil, err
	}
	first := map[string]ports.Chunk{}
	for _, ch := range chunks {
		k := ch.Kind + ":" + ch.SourceID
		if cur, ok := first[k]; !ok || ch.Start < cur.Start {
			first[k] = ch
		}
	}
	keys := make([]string, 0, len(first))
	for k := range first {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []string
	for _, k := range keys {
		words := strings.Fields(first[k].Text)
		if len(words) > 8 {
			words = words[:8]
		}
		if len(words) > 0 {
			out = append(out, strings.Join(words, " "))
		}
	}
	return out, nil
}

func docKey(cite string) string {
	if i := strings.LastIndex(cite, "@"); i >= 0 {
		return cite[:i]
	}
	return cite
}

// Ask asks as user, through the cell's own ask path with the user's directory groups.
func (t *cellTarget) Ask(ctx context.Context, user, q string) (verify.Answer, error) {
	groups, err := t.c.Directory.GroupsOf(ctx, t.tenant(), user)
	if err != nil {
		return verify.Answer{}, err
	}
	s, err := t.c.Auth.Issuer.Issue(t.tenant(), user, groups)
	if err != nil {
		return verify.Answer{}, err
	}
	resp, err := t.agent.Ask(ctx, s, ask.Request{Question: q})
	if err != nil {
		return verify.Answer{}, err
	}
	tr, err := t.agent.Trace(ctx, s, resp.Trace)
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
			doc := docKey(ci.Cite)
			kind, _, _ := strings.Cut(doc, ":")
			a.Cites = append(a.Cites, verify.Cite{Doc: doc, Connector: kind})
		}
	}
	return a, nil
}

// Allowed asks the source, live, whether user may open doc (KIND:ID) now.
func (t *cellTarget) Allowed(ctx context.Context, user, doc string) (bool, error) {
	kind, id, ok := strings.Cut(doc, ":")
	conn, found := t.c.Parts().Connectors[kind]
	if !ok || !found {
		return false, nil
	}
	u, known, err := t.c.Directory.User(ctx, t.tenant(), user)
	if err != nil || !known || len(u.Emails) == 0 {
		return false, err
	}
	vs, err := conn.ACL(ctx, ports.UserRef{ID: user, Email: u.Emails[0]}, []ports.SourceRef{{Kind: kind, ID: id}})
	if err != nil || len(vs) != 1 {
		return false, err
	}
	return vs[0] == ports.VerdictAllow, nil
}

// ForeignIDs are the control plane's, as the request carried them.
func (t *cellTarget) ForeignIDs(context.Context) ([]string, error) { return t.foreign, nil }

// Timeout makes one connector time out for the verifier's own asks until restore.
func (t *cellTarget) Timeout(connector string) (func(), error) {
	d, ok := t.down[connector]
	if !ok {
		return nil, errors.New("verify: no such connector")
	}
	d.Store(true)
	return func() { d.Store(false) }, nil
}

// Connectors are the cell's connector kinds.
func (t *cellTarget) Connectors() []string {
	out := make([]string, 0, len(t.down))
	for k := range t.down {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
