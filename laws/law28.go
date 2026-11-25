package laws

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/toolbus"
)

// LAW-28 An agent's writing says so (SB-4706; sb-laws(7)): an answer, in the API and in
// chat, says which agent and version wrote it; an agent draft in a steward queue carries its
// mark; and a note published from a draft keeps generated_by and approved_by in its front
// matter and on every chunk the index holds of it, so every cite of it shows the mark, until
// a person rewrites it: a note none of whose draft lines are left carries none. The real
// folder connector, ingest, index, ask agent, agent draft, reply and note publish.

// MarkSystem is the system under test: the note-publishing stewards, with a bus that may
// change what is published.
type MarkSystem interface {
	Stewards(dir string, members func() []string) (x ask.Stewards, sync, closer func() error, err error)
}

// RealMarks is the real system: RealNoteNarrowing's stewards with a model that cites.
type RealMarks struct {
	// Strip, for the self-test, takes the mark out of every note on its way to the source.
	Strip bool
}

// strippingBus is a bus that drops generated_by from a note's Markdown (the self-test).
type strippingBus struct{ next ask.NoteBus }

func (b strippingBus) Invoke(ctx context.Context, s tenancy.Scope, call toolbus.Call) (approvals.Action, error) {
	var p ask.NotePayload
	if err := json.Unmarshal(call.Payload, &p); err == nil {
		var keep []string
		for _, l := range strings.Split(p.Markdown, "\n") {
			if !strings.HasPrefix(l, "generated_by:") {
				keep = append(keep, l)
			}
		}
		p.Markdown = strings.Join(keep, "\n")
		call.Payload, _ = json.Marshal(p)
	}
	return b.next.Invoke(ctx, s, call)
}

// Stewards implements MarkSystem.
func (m RealMarks) Stewards(dir string, members func() []string) (ask.Stewards, func() error, func() error, error) {
	x, c, sync, err := RealNoteNarrowing{}.Stewards(dir, members)
	if err != nil {
		return ask.Stewards{}, nil, nil, err
	}
	x.Agent.Model = &citingModel{}
	if m.Strip {
		x.Bus = strippingBus{next: x.Bus}
	}
	return x, sync, c.Close, nil
}

func law28(r *rand.Rand, dir string, sys MarkSystem) error {
	ctx := context.Background()
	share, err := os.MkdirTemp(dir, "share")
	if err != nil {
		return err
	}
	topics := []string{"refunds", "returns", "shipping", "invoices", "carriers", "claims"}
	n := 1 + r.Intn(3)
	for i := 0; i < n; i++ {
		p := filepath.Join(share, topics[i]+".md")
		if err := os.WriteFile(p, []byte("# "+topics[i]+"\n\nHow we handle "+topics[i]+" for acme-logistics.\n"), 0o600); err != nil {
			return err
		}
		if err := os.WriteFile(p+".acl", []byte("read domain:acme.example\nowner ann@acme.example\n"), 0o600); err != nil {
			return err
		}
	}
	members := []string{"ann@acme.example", "sam@acme.example"}
	x, sync, closer, err := sys.Stewards(share, func() []string { return members })
	if err != nil {
		return err
	}
	defer func() { _ = closer() }()
	if err := sync(); err != nil {
		return err
	}
	iss := tenancy.NewIssuer("acme-logistics")
	asker, _ := iss.IssueIn("acme-logistics", "usr_ann", []string{"support"}, "spc_support")
	steward, _ := iss.IssueIn("acme-logistics", "usr_sam", []string{"support-leads"}, "spc_support")

	// An answer says who wrote it, in the API and in chat.
	resp, err := x.Agent.Ask(ctx, asker, ask.Request{Question: "how do we handle " + topics[r.Intn(n)]})
	if err != nil {
		return err
	}
	if resp.Agent == "" || resp.Release == "" {
		return fmt.Errorf("an answer carries agent %q and release %q", resp.Agent, resp.Release)
	}
	if line := ask.MarkLine(resp.Agent, resp.Release); !strings.Contains(slackapp.Render("q", resp).Text, line) {
		return fmt.Errorf("the chat answer doesn't say %q", line)
	}

	// An agent's draft carries its mark in the queue.
	sq, err := x.AgentDraft(ctx, steward, ask.DraftBrief, "This week", ask.Request{Question: "what happened with " + topics[0]})
	if err != nil {
		return fmt.Errorf("the agent draft: %v", err)
	}
	items, err := x.Queue(ctx, steward, "support")
	if err != nil {
		return err
	}
	marked := false
	for _, it := range items {
		if it.ID == sq.ID && it.Thread != nil && it.Thread.Mark != nil && it.Thread.Mark.Agent == "ask" && it.Thread.Mark.Version == x.Agent.Release {
			marked = true
		}
	}
	if !marked {
		return fmt.Errorf("the agent draft %s is in the queue without its mark", sq.ID)
	}

	// The steward replies with the draft as it is, edited, or rewritten, and publishes it.
	draft := sq.Thread.Text
	mode := r.Intn(3)
	reply := draft
	switch mode {
	case 1:
		reply = draft + "\nAnd a line of the steward's own."
	case 2:
		reply = "The steward's own account of " + topics[0] + "."
	}
	if _, err := x.Reply(ctx, steward, "support", sq.ID, reply, nil); err != nil {
		return err
	}
	pv, err := x.Note(ctx, steward, "support", sq.ID, ask.NoteRequest{Title: "This week in support"})
	if err != nil {
		return err
	}
	if _, err := x.Note(ctx, steward, "support", sq.ID, ask.NoteRequest{Title: "This week in support", Confirm: &pv.Narrower}); err != nil {
		return err
	}
	if err := sync(); err != nil {
		return err
	}
	all, ok := x.Agent.Index.(interface {
		All(context.Context, string) ([]ports.Chunk, error)
	})
	if !ok {
		return fmt.Errorf("the index can't list its chunks")
	}
	chunks, err := all.All(ctx, "acme-logistics")
	if err != nil {
		return err
	}
	rewritten := mode == 2
	want := ask.GeneratedBy("ask", x.Agent.Release)
	seen := 0
	for _, c := range chunks {
		if !strings.Contains(c.SourceID, sq.ID) {
			if c.GeneratedBy != "" {
				return fmt.Errorf("%s, which a person wrote, carries the mark %q", c.Cite, c.GeneratedBy)
			}
			continue
		}
		seen++
		switch {
		case rewritten && c.GeneratedBy != "":
			return fmt.Errorf("a note the steward rewrote still says %q wrote it", c.GeneratedBy)
		case !rewritten && (c.GeneratedBy != want || c.ApprovedBy != "sam@acme.example" || c.Edited != (mode == 1)):
			return fmt.Errorf("a note published from a draft (mode %d) is indexed with generated_by %q, approved_by %q, edited %v; want %q, sam@acme.example, %v (LAW-28)",
				mode, c.GeneratedBy, c.ApprovedBy, c.Edited, want, mode == 1)
		case !rewritten && ask.CiteMark(c.GeneratedBy, c.ApprovedBy, c.Edited) == "":
			return fmt.Errorf("a cite of %s shows no mark", c.Cite)
		}
	}
	if seen == 0 {
		return fmt.Errorf("the note %s wasn't indexed", sq.ID)
	}
	return nil
}
