package ask

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/tenancy"
)

// Save to the brain (sb-ask(3), The chat app; SB-3703): a person sends a chat thread to the
// stewards of the channel's space as a saved thread; only a steward's reply and note
// publish it.

// MaxThreadMessages is the most messages of a thread a save keeps.
const MaxThreadMessages = 50

// ThreadMessage is one message of a saved thread: its author's email and its text.
type ThreadMessage struct {
	Email, Text string
}

// SavedThread marks a steward question as a saved chat thread.
type SavedThread struct {
	Link      string `json:"link"`
	Text      string `json:"text"`
	Discarded bool   `json:"discarded,omitempty"`
	// Agent and Kind mark a draft an agent wrote on a schedule or a trigger (SB-4005,
	// SB-4006): the agent, and brief or decisions. It ran as Asker, so only they see Text.
	Agent string `json:"agent,omitempty"`
	Kind  string `json:"kind,omitempty"`
	// Doc is a decisions draft's transcript (KIND:ID), and Actions its action items, seen
	// only by Asker until they confirm them (SB-4203).
	Doc     string        `json:"doc,omitempty"`
	Actions []DraftAction `json:"actions,omitempty"`
	// Goal is a check-in's goal (KIND:ID; SB-4207).
	Goal string `json:"goal,omitempty"`
	// Supersede are older decisions a decisions draft suggests it replaces, for Asker to
	// confirm one when publishing (SB-4505).
	Supersede []Result `json:"supersede,omitempty"`
	// Asked, AskDocs and Sources are what an agent draft asked, the documents it was
	// narrowed to and those it cited or offered (KIND:ID): a redraft asks again from them
	// alone. Versions are its earlier texts, oldest first, and Rounds its redrafts (SB-4602).
	Asked    string   `json:"asked,omitempty"`
	AskDocs  []string `json:"ask_docs,omitempty"`
	Sources  []string `json:"sources,omitempty"`
	Versions []string `json:"versions,omitempty"`
	Rounds   int      `json:"rounds,omitempty"`
	// Release is the release the agent ran under, its version (SB-4706).
	Release string `json:"release,omitempty"`
}

// ThreadView is a saved thread on the steward screen.
type ThreadView struct {
	By   string `json:"by"`
	Link string `json:"link"`
	Text string `json:"text"`
	// Agent and Kind: an agent's draft (SB-4005, SB-4006). Hidden is a draft that ran as
	// another steward: its text is theirs to see alone, checked as them (LAW-12).
	Agent  string `json:"agent,omitempty"`
	Kind   string `json:"kind,omitempty"`
	Hidden bool   `json:"hidden,omitempty"`
	// Actions are a decisions draft's action items (SB-4203), for the steward it ran as.
	Actions []DraftAction `json:"actions,omitempty"`
	// Supersede are the decisions it may replace (SB-4505), for the steward it ran as.
	Supersede []Result `json:"supersede,omitempty"`
	// Versions, Rounds and Diff are an agent draft's earlier texts, its rounds of changes and
	// the line diff from the last to now (SB-4602), for the steward it ran as.
	Versions []string `json:"versions,omitempty"`
	Rounds   int      `json:"rounds,omitempty"`
	Diff     []string `json:"diff,omitempty"`
	// Trace is an agent draft's trace, for How this was made (SB-4606), for the steward it
	// ran as.
	Trace string `json:"trace,omitempty"`
	// Mark is which agent and version wrote a draft (SB-4706, LAW-28), shown to everyone.
	Mark *Mark `json:"mark,omitempty"`
}

// ThreadText is a thread's messages as lines EMAIL: text, at most MaxThreadMessages of them
// and MaxReply characters in all.
func ThreadText(ms []ThreadMessage) string {
	var b strings.Builder
	for i, m := range ms {
		if i == MaxThreadMessages {
			break
		}
		who := strings.TrimSpace(m.Email)
		if who == "" {
			who = "someone"
		}
		line := who + ": " + strings.Join(strings.Fields(m.Text), " ") + "\n"
		if utf8.RuneCountInString(b.String()+line) > MaxReply {
			break
		}
		b.WriteString(line)
	}
	return strings.TrimRight(b.String(), "\n")
}

// SaveThread sends a chat thread to the people who answer s's space, as s.
func (x Stewards) SaveThread(ctx context.Context, s tenancy.Scope, link string, ms []ThreadMessage) (StewardQuestion, error) {
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant {
		return StewardQuestion{}, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	text := ThreadText(ms)
	if text == "" {
		return StewardQuestion{}, fmt.Errorf("%w: the thread has no messages", ErrInvalid)
	}
	space := s.Space()
	if space == "" {
		space = "company"
	}
	if _, _, ok := x.Facts(space); !ok {
		return StewardQuestion{}, fmt.Errorf("%w: nobody answers questions sent from here", ErrInvalid)
	}
	first := strings.SplitN(text, "\n", 2)[0]
	if rs := []rune(first); len(rs) > MaxQuestion {
		first = string(rs[:MaxQuestion-1]) + "…"
	}
	sq := StewardQuestion{ID: x.NewID("qs_"), Tenant: x.Agent.Tenant, Asker: s.User(), Question: first, Space: space, At: x.Now().UTC(),
		Thread: &SavedThread{Link: link, Text: text}}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: sq.ID, State: "saved",
		PayloadSHA256: hashOf(text), Reason: "chat thread saved to stewards", Space: auditSpace(space)}); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Store.PutStewardQuestion(ctx, sq); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	return sq, nil
}

// Discard drops an open saved thread of the space: its text goes, and it leaves the queue.
func (x Stewards) Discard(ctx context.Context, s tenancy.Scope, name, id string) error {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return err
	}
	q, err := x.question(ctx, name, id)
	if err != nil {
		return fmt.Errorf("%w: no such saved thread in this space", ErrInvalid)
	}
	if q.Thread == nil || q.Thread.Discarded || q.Reply != nil {
		return fmt.Errorf("%w: only an open saved thread can be discarded", ErrInvalid)
	}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: id, State: "discarded",
		Reason: "saved thread discarded", Space: auditSpace(SpaceKey(name))}); err != nil {
		return fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	_, err = x.Store.UpdateStewardQuestion(ctx, x.Agent.Tenant, id, func(q *StewardQuestion) error {
		if q.Thread == nil || q.Reply != nil {
			return fmt.Errorf("%w: only an open saved thread can be discarded", ErrInvalid)
		}
		// The text goes; which agent drafted it, its kind and its rounds stay, counts for its
		// value (SB-4701).
		q.Thread = &SavedThread{Discarded: true, Agent: q.Thread.Agent, Kind: q.Thread.Kind, Rounds: q.Thread.Rounds, Release: q.Thread.Release}
		q.Question = "(discarded)"
		return nil
	})
	return err
}
