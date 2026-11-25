package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-25 A memory is its person's own (SB-4303; sb-laws(7)): a memory holds only what its
// person wrote, never text from a document they can open, and it is sent to the model only
// with that person's own asks to the ask agent. The real agent over the memory index, a mock
// source with random viewers, the memory store, and a model that records every prompt.
// People write lines of their own and paste passages; then everyone asks, and an eval probe
// runs as each of them.

// MemorySystem is the memory store the agent reads a person's lines from.
type MemorySystem interface {
	Store(base ask.MemoryStore) ask.MemoryStore
}

// RealMemory is the cell's: the store as it is.
type RealMemory struct{}

// Store implements MemorySystem.
func (RealMemory) Store(base ask.MemoryStore) ask.MemoryStore { return base }

// sharedMemory is the selftest's mutation: everyone's lines in everyone's memory.
type sharedMemory struct{}

func (sharedMemory) Store(base ask.MemoryStore) ask.MemoryStore { return &shared{base: base} }

type shared struct {
	base  ask.MemoryStore
	users []string
}

func (s *shared) MemoryOf(ctx context.Context, tenant, user string) (ask.Memory, bool, error) {
	out := ask.Memory{Tenant: tenant, User: user}
	for _, u := range s.users {
		m, _, err := s.base.MemoryOf(ctx, tenant, u)
		if err != nil {
			return out, false, err
		}
		out.Lines = append(out.Lines, m.Lines...)
	}
	return out, len(out.Lines) > 0, nil
}

func (s *shared) PutMemory(ctx context.Context, m ask.Memory) error {
	if !contains(s.users, m.User) {
		s.users = append(s.users, m.User)
	}
	return s.base.PutMemory(ctx, m)
}

var memoryWords = []string{"quokka", "ledger", "carrier", "invoice", "harbour", "pallet", "refund", "tariff", "freight", "audit", "budget", "vendor"}

func law25(r *rand.Rand, sys MemorySystem) error {
	ctx := context.Background()
	const tenant = "acme-logistics"
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	users := []string{"usr_1", "usr_2", "usr_3"}
	email := func(u string) string { return strings.TrimPrefix(u, "usr_") + "@acme.example" }
	src := mock.NewSource("docs")
	passages := map[string]string{}
	viewers := map[string][]string{}
	for i := 0; i < 3+r.Intn(4); i++ {
		id := fmt.Sprintf("d%d", i)
		var ws []string
		for j := 0; j < 14+r.Intn(10); j++ {
			ws = append(ws, memoryWords[r.Intn(len(memoryWords))])
		}
		passages[id] = fmt.Sprintf("Doc %d says: %s.", i, strings.Join(ws, " "))
		for _, u := range users {
			if r.Intn(2) == 0 {
				viewers[id] = append(viewers[id], email(u))
			}
		}
		if len(viewers[id]) == 0 {
			viewers[id] = []string{email(users[r.Intn(len(users))])}
		}
		src.Put(id, "Doc "+id, passages[id], viewers[id]...)
	}
	idx := memindex.New()
	p := ingest.Pipeline{Tenant: tenant, Release: "rel_law", Connector: mock.New(src, false), Embedder: hashing.Embedder{},
		Index: idx, Now: func() time.Time { return now }}
	if _, err := p.Sync(ctx, ""); err != nil {
		return err
	}
	model := &promptModel{}
	ids := 0
	a := ask.Agent{Tenant: tenant, Release: "rel_law", Index: idx.Bind(tenant), Store: tracemem.New(), Model: model, Embedder: hashing.Embedder{},
		Auth:   retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": mock.New(src, false)}, Policy: policy.Kernel{Engine: askPolicies, Recorder: okRecorder{}}},
		Emails: func(_ context.Context, u string) ([]string, error) { return []string{email(u)}, nil },
		Killed: func(context.Context) bool { return false }, Now: func() time.Time { return now },
		NewID:  func(p string) string { ids++; return fmt.Sprintf("%s%d", p, ids) },
		Memory: sys.Store(tracemem.New()), Audit: &recordingAudit{}}
	iss := tenancy.NewIssuer(tenant)
	scopes := map[string]tenancy.Scope{}
	tokens := map[string]string{}
	for _, u := range users {
		s, err := iss.Issue(tenant, u, nil)
		if err != nil {
			return err
		}
		scopes[u] = s
		tokens[u] = fmt.Sprintf("MEM%x", r.Int63())
		if _, err := a.Remember(ctx, s, "I am "+u+" and my note is "+tokens[u]); err != nil {
			return fmt.Errorf("%s's own line was refused: %v", u, err)
		}
		// A passage pasted from a document they can open is refused.
		for id, text := range passages {
			if !contains(viewers[id], email(u)) || r.Intn(2) == 0 {
				continue
			}
			line := text
			if r.Intn(2) == 0 { // or nine words of it in a row
				ws := strings.Fields(text)
				start := r.Intn(len(ws) - 9)
				line = "Remember: " + strings.Join(ws[start:start+9], " ")
			}
			if len(line) > ask.MaxMemoryLine {
				line = line[:ask.MaxMemoryLine]
			}
			if _, err := a.Remember(ctx, s, line); !errors.Is(err, ask.ErrFromDocument) {
				return fmt.Errorf("LAW-25: %s kept a passage of %s they can open as a memory (err %v)", u, id, err)
			}
		}
	}
	for _, u := range users {
		for _, probe := range []bool{false, true} {
			model.prompts = nil
			var err error
			if probe {
				_, err = a.Probe(ctx, scopes[u], "What do the docs say about quokka freight?")
			} else {
				_, err = a.Ask(ctx, scopes[u], ask.Request{Question: "What do the docs say about quokka freight?"})
			}
			if err != nil {
				return err
			}
			sent := strings.Join(model.prompts, "\n")
			if sent == "" {
				continue // nothing they can open: no model call
			}
			for _, o := range users {
				if o != u && strings.Contains(sent, tokens[o]) {
					return fmt.Errorf("LAW-25: %s's ask carried %s's memory", u, o)
				}
			}
			if probe && strings.Contains(sent, tokens[u]) {
				return fmt.Errorf("LAW-25: an eval probe as %s carried their memory", u)
			}
			if !probe && !strings.Contains(sent, tokens[u]) {
				return fmt.Errorf("%s's own ask didn't carry their memory", u)
			}
		}
	}
	return nil
}
