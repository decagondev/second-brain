package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Memory I control (sb-ask(3), Memory I control; SB-4303): a few lines a person writes
// about themselves, sent with their own asks to the ask agent as their own words, never as
// a source. LAW-25: a memory holds only what its person wrote, never text from a document,
// and it is sent with no one else's ask.

// The memory's limits.
const (
	MaxMemoryLines = 20
	MaxMemoryLine  = 200
	// memoryRun is how many words in a row of a passage make a line a copy of it.
	memoryRun = 8
	// memoryWhole is the shortest line that is refused for being inside a passage whole.
	memoryWhole = 40
)

// MemoryLine is one line a person wrote about themselves.
type MemoryLine struct {
	ID   string    `json:"id"`
	Text string    `json:"text"`
	At   time.Time `json:"at"`
}

// Memory is one person's lines, as the cell keeps them.
type Memory struct {
	Tenant string       `json:"tenant"`
	User   string       `json:"user"`
	Lines  []MemoryLine `json:"lines"`
	At     time.Time    `json:"at"`
	// Excluded are the documents the person keeps out of everything made as them (SB-5503),
	// kept here so they are exported, erased and forgotten with the person's memory.
	Excluded []Exclusion `json:"excluded,omitempty"`
}

// MemoryStore keeps each person's memory, one record per person, erased with them and by
// their own Forget, never expired.
type MemoryStore interface {
	MemoryOf(ctx context.Context, tenant, user string) (Memory, bool, error)
	PutMemory(ctx context.Context, m Memory) error
}

// MemoryView is a person's memory as they see it.
type MemoryView struct {
	On    bool         `json:"on"`
	Lines []MemoryLine `json:"lines"`
}

// ErrMemoryOff is a cell whose manifest has memory: off.
var ErrMemoryOff = fmt.Errorf("%w: memory is off for your organisation", ErrUnavailable)

// ErrFromDocument is a line that repeats a document the person can open.
var ErrFromDocument = fmt.Errorf("%w: that reads like a document, not something about you", ErrInvalid)

func (a Agent) memoryOn() bool { return a.Memory != nil }

func (a Agent) memoryScope(s tenancy.Scope) error {
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant {
		return fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	return nil
}

// MemoryFor is the caller's own memory.
func (a Agent) MemoryFor(ctx context.Context, s tenancy.Scope) (MemoryView, error) {
	if err := a.memoryScope(s); err != nil {
		return MemoryView{}, err
	}
	out := MemoryView{On: a.memoryOn(), Lines: []MemoryLine{}}
	if !out.On {
		return out, nil
	}
	m, _, err := a.Memory.MemoryOf(ctx, a.Tenant, s.User())
	if err != nil {
		return MemoryView{}, fmt.Errorf("%w: memory: %v", ErrUnavailable, err)
	}
	if len(m.Lines) > 0 {
		out.Lines = m.Lines
	}
	return out, nil
}

// Remember adds one line the caller wrote. A line that repeats a passage they can open is
// refused, and so is any line when that can't be checked (LAW-03, LAW-25).
func (a Agent) Remember(ctx context.Context, s tenancy.Scope, text string) (MemoryLine, error) {
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
	if len(m.Lines) >= MaxMemoryLines {
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
	if err := a.memoryAudit(ctx, s, "remembered", hashOf(line)); err != nil {
		return MemoryLine{}, err
	}
	m.Tenant, m.User, m.At = a.Tenant, s.User(), now
	m.Lines = append(m.Lines, l)
	if err := a.Memory.PutMemory(ctx, m); err != nil {
		return MemoryLine{}, fmt.Errorf("%w: memory: %v", ErrUnavailable, err)
	}
	return l, nil
}

// ForgetLine removes one of the caller's lines, or all of them when id is "".
func (a Agent) ForgetLine(ctx context.Context, s tenancy.Scope, id string) (int, error) {
	if err := a.memoryScope(s); err != nil {
		return 0, err
	}
	if !a.memoryOn() {
		return 0, ErrMemoryOff
	}
	m, _, err := a.Memory.MemoryOf(ctx, a.Tenant, s.User())
	if err != nil {
		return 0, fmt.Errorf("%w: memory: %v", ErrUnavailable, err)
	}
	var keep []MemoryLine
	gone := 0
	for _, l := range m.Lines {
		if id == "" || l.ID == id {
			gone++
			continue
		}
		keep = append(keep, l)
	}
	if id != "" && gone == 0 {
		return 0, fmt.Errorf("%w: no such line", ErrInvalid)
	}
	if gone == 0 {
		return 0, nil
	}
	if err := a.memoryAudit(ctx, s, "forgotten", fmt.Sprintf("%d", gone)); err != nil {
		return 0, err
	}
	m.Tenant, m.User, m.At, m.Lines = a.Tenant, s.User(), a.now(), keep
	if err := a.Memory.PutMemory(ctx, m); err != nil {
		return 0, fmt.Errorf("%w: memory: %v", ErrUnavailable, err)
	}
	return gone, nil
}

func (a Agent) now() time.Time {
	if a.Now == nil {
		return time.Now().UTC()
	}
	return a.Now().UTC()
}

func (a Agent) memoryAudit(ctx context.Context, s tenancy.Scope, state, sum string) error {
	if a.Audit == nil {
		return fmt.Errorf("%w: the audit log is unavailable", ErrUnavailable)
	}
	if _, err := a.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: "memory:" + s.User(), State: state,
		PayloadSHA256: sum, Reason: "memory"}); err != nil {
		return fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	return nil
}

// fromDocument says whether line repeats a passage s can open: the whole line inside one,
// or memoryRun words of it in a row, ignoring case, spaces and punctuation. It searches as
// s, through every check a search makes, so a refusal names nothing they can't open.
func (a Agent) fromDocument(ctx context.Context, s tenancy.Scope, line string) (bool, error) {
	res, _, err := a.retrieve(ctx, s, line, nil, retrieve.Budget)
	if err != nil {
		return false, err
	}
	words := normWords(line)
	whole := strings.Join(words, " ")
	runs := map[string]bool{}
	for i := 0; i+memoryRun <= len(words); i++ {
		runs[strings.Join(words[i:i+memoryRun], " ")] = true
	}
	for _, c := range res.Context {
		pw := normWords(c.Text)
		text := " " + strings.Join(pw, " ") + " "
		if len(whole) >= memoryWhole && strings.Contains(text, " "+whole+" ") {
			return true, nil
		}
		for i := 0; i+memoryRun <= len(pw) && len(runs) > 0; i++ {
			if runs[strings.Join(pw[i:i+memoryRun], " ")] {
				return true, nil
			}
		}
	}
	return false, nil
}

func normWords(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
}

// memoryLines are the asker's own lines for their ask, or none: a cell without memory, a
// probe, or any agent but ask carries none (LAW-25). A store that can't be read sends none.
// memoryUsed are the lines an ask sends, with their ids, oldest first (SB-5504).
func (a Agent) memoryUsed(ctx context.Context, s tenancy.Scope, agent string, probe bool) []MemoryLine {
	if !a.memoryOn() || probe || (agent != "" && agent != "ask") {
		return nil
	}
	m, found, err := a.Memory.MemoryOf(ctx, a.Tenant, s.User())
	if err != nil || !found || m.User != s.User() || m.Tenant != a.Tenant {
		return nil
	}
	sort.SliceStable(m.Lines, func(i, j int) bool { return m.Lines[i].At.Before(m.Lines[j].At) })
	return m.Lines
}

type memoryKey struct{}

func withMemory(ctx context.Context, lines []string) context.Context {
	if len(lines) == 0 {
		return ctx
	}
	return context.WithValue(ctx, memoryKey{}, lines)
}

func memoryOf(ctx context.Context) []string {
	l, _ := ctx.Value(memoryKey{}).([]string)
	return l
}

// MemoryLead heads the memory block: the asker's own words, never a source.
const MemoryLead = "About the person asking, in their own words (not a source; never cite it):"

func memoryData(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return ingest.Wrap(MemoryLead+"\n- "+strings.Join(lines, "\n- "), "about-the-asker", nil) + "\n"
}
