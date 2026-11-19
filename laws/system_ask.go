package laws

import (
	"context"
	"fmt"

	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// The real ask pipeline (SB-301 to SB-303) under the retrieval-level laws: one index per
// tenant, bound to it; the world's source answers the live ACL; the cedar base policies
// run in the kernel. Tool calls (Call) are still pending until SB-405 and SB-408.

func init() {
	NewSystem = func(w *World) System { return askSystem{w: w} }
}

type askSystem struct {
	pending
	w *World
	// Mutations for the self-tests: index other tenants' documents in this cell, or
	// treat an Unknown live answer as Allow.
	indexOthers, unknownAsAllow bool
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
	// The asker's cell has its own index, holding only its tenant's documents.
	idx := memindex.New().Bind(u.Tenant)
	for _, d := range w.Docs {
		if d.Tenant != u.Tenant && !s.indexOthers {
			continue
		}
		ref := ports.SourceRef{Kind: "docs", ID: d.ID}
		c := ports.Chunk{ID: "chk_" + d.ID, Tenant: d.Tenant, Cite: fmt.Sprintf("%s@%d", d.ID, d.Version), Kind: "docs",
			SourceID: d.ID, Version: d.Version, Text: "anything " + d.ID, ACLPrefilter: d.Prefilter}
		if s.indexOthers && d.Tenant != u.Tenant {
			// The mutation files another tenant's chunk under this cell, with a prefilter
			// built wrongly; only the tenant boundary itself is left to stop it.
			c.Tenant, c.ACLPrefilter = u.Tenant, []string{"anyone"}
		}
		if err := idx.Replace(ctx, u.Tenant, ref, d.Version, []ports.Chunk{c}); err != nil {
			return Answer{}, err
		}
	}
	auth := retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": worldConnector{w, s.unknownAsAllow}},
		Policy: policy.Kernel{Engine: askPolicies, Recorder: okRecorder{}}}
	res, err := retrieve.Context(ctx, scope, ports.UserRef{ID: u.ID}, nil, idx, auth, ports.Query{Text: question})
	if err != nil {
		return Answer{}, err
	}
	var a Answer
	seen := map[string]bool{}
	for _, c := range res.Context {
		if !seen[c.SourceID] {
			seen[c.SourceID] = true
			a.Context = append(a.Context, c.SourceID)
			a.Citations = append(a.Citations, Citation{Doc: c.SourceID, Version: c.Version})
		}
	}
	return a, nil
}
