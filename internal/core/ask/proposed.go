package ask

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/tenancy"
)

// Memory over MCP (sb-ask(3), Memory I control; SB-5505): an MCP client reads the person's
// own lines and may propose one; a proposal is never sent with an ask until the person
// confirms it in the web UI.

// MaxProposed is the most proposals waiting at once.
const MaxProposed = 10

// InboxMemory is a proposed memory line waiting for its person (SB-5505).
const InboxMemory = "memory"

// ProposeMemory keeps a line an MCP client proposes for the caller, unconfirmed. The same
// checks as Remember apply, so a line that repeats a document is refused.
func (a Agent) ProposeMemory(ctx context.Context, s tenancy.Scope, text string) (MemoryLine, error) {
	return a.addLine(ctx, s, text, true)
}

// addLine keeps one line the caller wrote, or one proposed for them. A line that repeats a
// passage they can open is refused, and so is any line when that can't be checked (LAW-03,
// LAW-25).
func (a Agent) addLine(ctx context.Context, s tenancy.Scope, text string, proposed bool) (MemoryLine, error) {
	if err := a.memoryScope(s); err != nil {
		return MemoryLine{}, err
	}
	if !a.memoryOn() {
		return MemoryLine{}, ErrMemoryOff
	}
	line := strings.TrimSpace(text)
	if line == "" || utf8.RuneCountInString(line) > MaxMemoryLine || strings.ContainsAny(line, "\r\n") {
		return MemoryLine{}, fmt.Errorf("%w: a line is 1 to %d characters on one line", ErrInvalid, MaxMemoryLine)
	}
	m, _, err := a.Memory.MemoryOf(ctx, a.Tenant, s.User())
	if err != nil {
		return MemoryLine{}, fmt.Errorf("%w: memory: %v", ErrUnavailable, err)
	}
	switch {
	case proposed && len(m.Proposed) >= MaxProposed:
		return MemoryLine{}, fmt.Errorf("%w: %d proposals are waiting; confirm or dismiss them first", ErrInvalid, MaxProposed)
	case !proposed && len(m.Lines) >= MaxMemoryLines:
		return MemoryLine{}, fmt.Errorf("%w: at most %d lines; remove one first", ErrInvalid, MaxMemoryLines)
	}
	copied, err := a.fromDocument(ctx, s, line)
	if err != nil {
		return MemoryLine{}, err
	}
	if copied {
		return MemoryLine{}, ErrFromDocument
	}
	now := a.now()
	l := MemoryLine{ID: a.NewID("mem_"), Text: line, At: now}
	state := map[bool]string{false: "remembered", true: "proposed"}[proposed]
	if err := a.memoryAudit(ctx, s, state, hashOf(line)); err != nil {
		return MemoryLine{}, err
	}
	m.Tenant, m.User, m.At = a.Tenant, s.User(), now
	if proposed {
		m.Proposed = append(m.Proposed, l)
	} else {
		m.Lines = append(m.Lines, l)
	}
	if err := a.Memory.PutMemory(ctx, m); err != nil {
		return MemoryLine{}, fmt.Errorf("%w: memory: %v", ErrUnavailable, err)
	}
	return l, nil
}

// DecideProposal confirms or dismisses one of the caller's proposed lines, in the web UI.
func (a Agent) DecideProposal(ctx context.Context, s tenancy.Scope, id string, confirm bool) (MemoryView, error) {
	if err := a.memoryScope(s); err != nil {
		return MemoryView{}, err
	}
	if !a.memoryOn() {
		return MemoryView{}, ErrMemoryOff
	}
	m, _, err := a.Memory.MemoryOf(ctx, a.Tenant, s.User())
	if err != nil {
		return MemoryView{}, fmt.Errorf("%w: memory: %v", ErrUnavailable, err)
	}
	var keep []MemoryLine
	var got *MemoryLine
	for i, l := range m.Proposed {
		if l.ID == id && got == nil {
			got = &m.Proposed[i]
			continue
		}
		keep = append(keep, l)
	}
	if got == nil {
		return MemoryView{}, fmt.Errorf("%w: no such proposal", ErrInvalid)
	}
	if confirm && len(m.Lines) >= MaxMemoryLines {
		return MemoryView{}, fmt.Errorf("%w: at most %d lines; remove one first", ErrInvalid, MaxMemoryLines)
	}
	state := map[bool]string{true: "remembered", false: "dismissed"}[confirm]
	if err := a.memoryAudit(ctx, s, state, hashOf(got.Text)); err != nil {
		return MemoryView{}, err
	}
	line := *got
	m.Tenant, m.User, m.At, m.Proposed = a.Tenant, s.User(), a.now(), keep
	if confirm {
		line.At = a.now()
		m.Lines = append(m.Lines, line)
	}
	if err := a.Memory.PutMemory(ctx, m); err != nil {
		return MemoryView{}, fmt.Errorf("%w: memory: %v", ErrUnavailable, err)
	}
	return a.MemoryFor(ctx, s)
}

// ownProposals are the caller's proposals as inbox items.
func (a Agent) ownProposals(ctx context.Context, s tenancy.Scope) []InboxItem {
	if !a.memoryOn() {
		return nil
	}
	m, _, err := a.Memory.MemoryOf(ctx, a.Tenant, s.User())
	if err != nil || m.User != s.User() {
		return nil
	}
	var out []InboxItem
	for _, l := range m.Proposed {
		out = append(out, InboxItem{Kind: InboxMemory, ID: l.ID, Title: l.Text, At: l.At, Why: "Proposed by an MCP client; it counts only once you confirm it", Link: "/?tab=data"})
	}
	return out
}

// MemoryOn says the cell keeps people's memory (the manifest's memory isn't off).
func (a Agent) MemoryOn() bool { return a.memoryOn() }
