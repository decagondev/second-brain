package laws

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-26 A stored result is checked when read (SB-4201; sb-laws(7)): a research report or a
// meeting brief, stored when it ran, shows its reader only cites they can open when it is
// shown. Random documents and ACLs; a report cites them as its owner could when it ran,
// grants are revoked (or go Unknown) before the read, and the report is shown again.

// StoredSystem shows claims stored earlier to a reader, now.
type StoredSystem interface {
	// Load indexes docs, each with its viewers (user ids); unknown docs answer Unknown.
	Load(docs map[string][]string, unknown map[string]bool) error
	// Show is the stored claims as user may see them now.
	Show(user string, claims []answer.Claim) ([]answer.Claim, error)
}

// NewStoredSystem builds the system under test: the real ask.Agent's Recheck, through
// retrieve.Allowed with the shipped policies.
var NewStoredSystem = func() StoredSystem { return &storedCell{} }

type storedCell struct {
	agent   ask.Agent
	docs    map[string][]string
	unknown map[string]bool
	stale   bool // the mutation: shown as stored
}

type storedACL struct{ c *storedCell }

func (storedACL) APIVersion() int { return ports.ConnectorAPIVersion }
func (storedACL) Fetch(context.Context, ports.SourceRef) (ports.Document, error) {
	return ports.Document{}, fmt.Errorf("not used")
}
func (a storedACL) ACL(_ context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs))
	for i, r := range refs {
		switch {
		case a.c.unknown[r.ID]:
			out[i] = ports.VerdictUnknown
		case contains(a.c.docs[r.ID], u.ID):
			out[i] = ports.VerdictAllow
		default:
			out[i] = ports.VerdictDeny
		}
	}
	return out, nil
}
func (storedACL) Changes(context.Context, ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	return nil, "", nil
}
func (storedACL) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, fmt.Errorf("ENOTSUP")
}

func (c *storedCell) Load(docs map[string][]string, unknown map[string]bool) error {
	ctx := context.Background()
	ev, err := LoadPolicies()
	if err != nil {
		return err
	}
	c.docs, c.unknown = docs, unknown
	idx := memindex.New()
	for id := range docs {
		ch := ports.Chunk{ID: "chk_" + id, Tenant: spaceTenant, Cite: "docs:" + id + "@1", Kind: "docs", SourceID: id, Version: 1,
			Text: "the brain knows " + id, ACLPrefilter: []string{"anyone"}, Label: "internal"}
		if err := idx.Replace(ctx, spaceTenant, ports.SourceRef{Kind: "docs", ID: id}, 1, []ports.Chunk{ch}); err != nil {
			return err
		}
	}
	c.agent = ask.Agent{Tenant: spaceTenant, Release: "rel_law", Index: idx,
		Auth: retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": storedACL{c}},
			Policy: policy.Kernel{Engine: ev, Recorder: okRecorder{}}}}
	return nil
}

func (c *storedCell) Show(user string, claims []answer.Claim) ([]answer.Claim, error) {
	if c.stale {
		return claims, nil
	}
	s, err := tenancy.NewIssuer(spaceTenant).IssueIn(spaceTenant, user, []string{"staff"}, spaces.Company)
	if err != nil {
		return nil, err
	}
	out, _, err := c.agent.Recheck(context.Background(), s, claims)
	return out, err
}

func law26(r *rand.Rand, sys StoredSystem) error {
	users := []string{"usr_1", "usr_2", "usr_3"}
	docs, unknown := map[string][]string{}, map[string]bool{}
	var ids []string
	for i := 0; i < 3+r.Intn(5); i++ {
		id := fmt.Sprintf("d%d", i)
		ids = append(ids, id)
		docs[id] = []string{"usr_1"} // the owner could open every one when it ran
		for _, u := range users[1:] {
			if r.Intn(2) == 0 {
				docs[id] = append(docs[id], u)
			}
		}
	}
	// The report, as it ran: each claim cites one to three documents.
	var claims []answer.Claim
	for i := 0; i < 2+r.Intn(4); i++ {
		cl := answer.Claim{Text: fmt.Sprintf("Finding: %d", i)}
		for j := 0; j < 1+r.Intn(3); j++ {
			cl.Cites = append(cl.Cites, answer.Cite{Cite: "docs:" + ids[r.Intn(len(ids))] + "@1"})
		}
		claims = append(claims, cl)
	}
	// Before the read: grants revoked, and some documents' ACLs can't be read.
	for _, id := range ids {
		var keep []string
		for _, u := range docs[id] {
			if r.Intn(3) != 0 {
				keep = append(keep, u)
			}
		}
		docs[id] = keep
		unknown[id] = r.Intn(6) == 0
	}
	if err := sys.Load(docs, unknown); err != nil {
		return err
	}
	for _, u := range users {
		shown, err := sys.Show(u, claims)
		if err != nil {
			return err
		}
		for _, cl := range shown {
			if !cl.Uncited && len(cl.Cites) == 0 {
				return fmt.Errorf("%s was shown a cited claim with no cite left: %q", u, cl.Text)
			}
			for _, ci := range cl.Cites {
				id := strings.TrimSuffix(strings.TrimPrefix(ci.Cite, "docs:"), "@1")
				if unknown[id] || !contains(docs[id], u) {
					return fmt.Errorf("%s was shown %s, which they can't open now (unknown %v)", u, ci.Cite, unknown[id])
				}
			}
		}
	}
	return nil
}
