package ask

import (
	"context"
	"sort"
	"sync"

	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

// Suspect passages (sb-ingest(7), Suspect passages; SB-5101): a passage that reads like
// instructions to the model is kept out of every prompt, unless its space monitors only,
// and its cite is recorded. Never its text.

// Suspect modes of a space (sb-space(5), suspect).
const (
	SuspectSkip    = "skip"
	SuspectMonitor = "monitor"
)

// skips collects the cites of one answer's suspect passages, its consultations' too.
type skips struct {
	mu      sync.Mutex
	cites   map[string]bool
	skipped map[string]bool
}

type skipsKey struct{}

// withSkips gives ctx a collector, or keeps the one it has: a consultation is part of the
// asker's answer.
func withSkips(ctx context.Context) (context.Context, *skips) {
	if sk, ok := ctx.Value(skipsKey{}).(*skips); ok {
		return ctx, sk
	}
	sk := &skips{cites: map[string]bool{}, skipped: map[string]bool{}}
	return context.WithValue(ctx, skipsKey{}, sk), sk
}

// Cites are the recorded suspect cites, sorted.
func (sk *skips) Cites() []string {
	sk.mu.Lock()
	defer sk.mu.Unlock()
	out := make([]string, 0, len(sk.cites))
	for c := range sk.cites {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// Skipped is how many passages were kept out of a prompt, each once.
func (sk *skips) Skipped() int {
	sk.mu.Lock()
	defer sk.mu.Unlock()
	return len(sk.skipped)
}

// suspectMode is space's mode, by key: skip unless the space says monitor.
func (a Agent) suspectMode(space string) string {
	if a.Suspect != nil && a.Suspect(space) == SuspectMonitor {
		return SuspectMonitor
	}
	return SuspectSkip
}

// screen is chunks without their suspect passages (or all of them, monitored) for the
// space ctx asks in, each suspect cite recorded in ctx's collector.
func (a Agent) screen(ctx context.Context, chunks []ports.Chunk) []ports.Chunk {
	return a.screenIn(ctx, spaceOf(ctx), chunks)
}

// screenIn is screen for space, by key. Screening twice records each cite once.
func (a Agent) screenIn(ctx context.Context, space string, chunks []ports.Chunk) []ports.Chunk {
	sk, _ := ctx.Value(skipsKey{}).(*skips)
	monitor := a.suspectMode(space) == SuspectMonitor
	out := make([]ports.Chunk, 0, len(chunks))
	for _, c := range chunks {
		if !ingest.Suspect(c.Text, c.Flags) {
			out = append(out, c)
			continue
		}
		if sk != nil {
			sk.mu.Lock()
			sk.cites[c.Cite] = true
			if !monitor {
				sk.skipped[c.Cite] = true
			}
			sk.mu.Unlock()
		}
		if monitor {
			out = append(out, c)
		}
	}
	return out
}

// GuardEvent is one suspect passage of a kept answer (SB-5103): its cite, space, the agent
// that answered and whether it was skipped or monitored. Never the asker or the text.
type GuardEvent struct {
	Cite, Space, Agent, State string
}

// Guard states of a suspect passage.
const (
	GuardSkipped   = "skipped"
	GuardMonitored = "monitored"
)

// guard records cites as guardrail events of an answer in space by agent.
func (a Agent) guard(ctx context.Context, space, agent string, cites []string) {
	if a.Guard == nil {
		return
	}
	state := GuardSkipped
	if a.suspectMode(space) == SuspectMonitor {
		state = GuardMonitored
	}
	for _, c := range cites {
		a.Guard(ctx, GuardEvent{Cite: c, Space: space, Agent: agent, State: state})
	}
}
