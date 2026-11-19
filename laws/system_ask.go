package laws

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	memindex "example.com/second-brain/internal/adapters/index/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// The real ask pipeline (SB-301 to SB-303) under the retrieval-level laws: one index per
// tenant, bound to it; the world's source answers the live ACL; the cedar base policies
// run in the kernel, and the answer goes through the ask agent and composer (SB-306), with a
// stand-in model that cites everything it is given and tries a spoofed label. Tool calls
// (Call) are still pending until SB-405 and SB-408.

func init() {
	NewSystem = func(w *World) System { return askSystem{w: w} }
}

type askSystem struct {
	pending
	w *World
	// Mutations for the self-tests: index other tenants' documents in this cell, or
	// treat an Unknown live answer as Allow.
	indexOthers, unknownAsAllow bool
	// citeCandidates: a composer that cites every retrieved candidate, refused or not.
	citeCandidates bool
}

// worldConnector answers ACL from the world's live verdicts and records that it asked.
type worldConnector struct {
	w              *World
	unknownAsAllow bool
}

func (worldConnector) APIVersion() int { return ports.ConnectorAPIVersion }
func (worldConnector) Fetch(context.Context, ports.SourceRef) (ports.Document, error) {
	return ports.Document{}, fmt.Errorf("not used")
}
func (c worldConnector) ACL(_ context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs))
	for i, r := range refs {
		out[i] = c.w.ACL(u.ID, r.ID)
		if c.unknownAsAllow && out[i] == ports.VerdictUnknown {
			out[i] = ports.VerdictAllow
		}
	}
	return out, nil
}
func (worldConnector) Changes(context.Context, ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	return nil, "", nil
}
func (worldConnector) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, fmt.Errorf("ENOTSUP")
}

var askPolicies, askPoliciesErr = LoadPolicies()

func (s askSystem) Ask(w *World, userID, question string) (Answer, error) {
	if askPoliciesErr != nil {
		return Answer{}, askPoliciesErr
	}
	ctx := context.Background()
	u := w.user(userID)
	scope, err := tenancy.NewIssuer(u.Tenant).Issue(u.Tenant, u.ID, u.Groups)
	if err != nil {
		return Answer{}, err
	}
	agent, err := s.agent(ctx, w, u.Tenant)
	if err != nil {
		return Answer{}, err
	}
	resp, err := agent.Ask(ctx, scope, ask.Request{Question: question})
	if err != nil {
		return Answer{}, err
	}
	tr, err := agent.Trace(ctx, scope, resp.Trace)
	if err != nil {
		return Answer{}, err
	}
	var a Answer
	for _, d := range tr.Decisions {
		if d.Verdict == "allow" {
			a.Context = append(a.Context, docOf(d.Cite))
		}
	}
	seen := map[string]bool{}
	for _, c := range resp.Answer.Claims {
		cc := ComposedClaim{Text: c.Text, Uncited: c.Uncited}
		for _, ci := range c.Cites {
			cc.Cites = append(cc.Cites, ci.Cite)
			doc := docOf(ci.Cite)
			if !seen[doc] {
				seen[doc] = true
				v, _ := strconv.Atoi(ci.Cite[strings.LastIndex(ci.Cite, "@")+1:])
				a.Citations = append(a.Citations, Citation{Doc: doc, Version: v})
			}
		}
		a.Claims = append(a.Claims, cc)
	}
	if s.citeCandidates {
		for _, d := range tr.Decisions {
			a.Claims = append(a.Claims, ComposedClaim{Text: "candidate", Cites: []string{d.Cite}})
		}
	}
	return a, nil
}

func docOf(cite string) string {
	if i := strings.LastIndex(cite, "@"); i >= 0 {
		return cite[:i]
	}
	return cite
}

// citingModel stands in for a model provider: it cites every data block it was given,
// one claim each, plus a claim citing a label that was never in the context (a spoof the
// composer must strip) and a claim citing nothing.
type citingModel struct{}

var dataLabel = regexp.MustCompile(`<<data cite="(C[0-9]+) `)

func (*citingModel) Complete(_ context.Context, req ports.ModelRequest) (ports.ModelResponse, error) {
	type claim struct {
		Text  string   `json:"text"`
		Cites []string `json:"cites"`
	}
	var cs []claim
	last := req.Messages[len(req.Messages)-1].Content
	for _, m := range dataLabel.FindAllStringSubmatch(last, -1) {
		cs = append(cs, claim{Text: "from " + m[1], Cites: []string{m[1]}})
	}
	cs = append(cs, claim{Text: "spoofed", Cites: []string{"C999"}}, claim{Text: "general", Cites: nil})
	b, err := json.Marshal(map[string]any{"claims": cs})
	return ports.ModelResponse{Text: string(b)}, err
}

// agent builds the cell's ask agent for one tenant: its own index, holding only its
// tenant's documents (unless mutated), the world's source as the live check, and the
// cedar base policies.
func (s askSystem) agent(ctx context.Context, w *World, tenant string) (ask.Agent, error) {
	idx := memindex.New().Bind(tenant)
	for _, d := range w.Docs {
		if d.Tenant != tenant && !s.indexOthers {
			continue
		}
		ref := ports.SourceRef{Kind: "docs", ID: d.ID}
		c := ports.Chunk{ID: "chk_" + d.ID, Tenant: d.Tenant, Cite: fmt.Sprintf("%s@%d", d.ID, d.Version), Kind: "docs",
			SourceID: d.ID, Version: d.Version, Text: "anything " + d.ID, ACLPrefilter: d.Prefilter}
		if s.indexOthers && d.Tenant != tenant {
			// The mutation files another tenant's chunk under this cell, with a prefilter
			// built wrongly; only the tenant boundary itself is left to stop it.
			c.Tenant, c.ACLPrefilter = tenant, []string{"anyone"}
		}
		if err := idx.Replace(ctx, tenant, ref, d.Version, []ports.Chunk{c}); err != nil {
			return ask.Agent{}, err
		}
	}
	auth := retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": worldConnector{w, s.unknownAsAllow}},
		Policy: policy.Kernel{Engine: askPolicies, Recorder: okRecorder{}}}
	n := 0
	agent := ask.Agent{Tenant: tenant, Release: "rel_law", Index: idx, Auth: auth,
		Model:  &citingModel{},
		Store:  tracemem.New(),
		Killed: func(context.Context) bool { return false },
		NewID:  func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) },
		Now:    time.Now}
	return agent, nil
}
