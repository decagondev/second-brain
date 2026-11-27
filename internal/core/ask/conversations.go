package ask

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/tenancy"
)

// Carry the conversation (sb-ask(3), Your conversations; SB-5802 to SB-5804). A person's
// conversations are built from what the cell already keeps: their conversation records (the
// questions, theirs alone) and their traces (30 days). A conversation started in a chat app
// is listed for them on the web with Continue here (SB-5802). With Keep my conversations on
// every device on (off by default), the conversations they start on the web are listed on
// every device too (SB-5803): each entry (its questions, kept and dropped documents and
// trace ids, never an answer) lives in the person's own memory record, so it is exported
// and forgotten with it, goes when its traces do, and turning it off deletes every entry.
// Opening one shows each answer checked again as the person now (LAW-26): an answer they
// can't open every cite of now shows as no longer available to them. Share conversation
// (SB-5804) sends the whole conversation to a colleague: each answer is checked as the
// reader, one they can't open every cite of is left out and counted, and the share carries
// the highest label of what they see (LAW-32).

// The limits.
const (
	MaxSyncedConversations = 50
	MaxListedConversations = 50
)

// NoLongerAvailable is how an answer the person can't open every cite of now reads.
const NoLongerAvailable = "No longer available to you"

// SyncedConversation is one web conversation kept for every device (SB-5803).
type SyncedConversation struct {
	ID        string    `json:"id"`
	Questions []string  `json:"questions"`
	Kept      []string  `json:"kept,omitempty"`
	Dropped   []string  `json:"dropped,omitempty"`
	Traces    []string  `json:"traces"`
	At        time.Time `json:"at"`
}

// ConversationItem is one conversation in a person's list.
type ConversationItem struct {
	ID     string    `json:"id"`
	Client string    `json:"client"` // where it started: web or chat
	First  string    `json:"first"`  // its first question
	Turns  int       `json:"turns"`
	At     time.Time `json:"at"` // its last answer
	// Synced says it is listed because Keep my conversations on every device is on.
	Synced bool `json:"synced,omitempty"`
}

// Conversations is GET /v1/me/conversations.
type Conversations struct {
	Sync          bool               `json:"sync"`
	Conversations []ConversationItem `json:"conversations"`
}

// Turn is one answer of a conversation, checked again as the person reading it now.
type Turn struct {
	Question    string         `json:"question,omitempty"`
	Trace       string         `json:"trace"`
	At          time.Time      `json:"at"`
	Claims      []answer.Claim `json:"claims,omitempty"`
	Label       string         `json:"label,omitempty"`
	Unavailable bool           `json:"unavailable,omitempty"`
}

// ConversationView is one conversation to continue, or a shared one to read.
type ConversationView struct {
	ID    string `json:"id,omitempty"`
	Turns []Turn `json:"turns"`
	// Left is how many answers of a shared conversation are left out because the reader
	// can't open every cite (SB-5804); Label the highest label of what they see.
	Left  int    `json:"left,omitempty"`
	Label string `json:"label,omitempty"`
}

// ErrNoConversation is a conversation that isn't the person's, or is gone (EPERM).
var ErrNoConversation = errors.New("ask: no such conversation for this user")

func (a Agent) ownScope(s tenancy.Scope) error {
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant {
		return fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	return nil
}

// tracesOf are the person's traces still kept, oldest first.
func (a Agent) tracesOf(ctx context.Context, user string) ([]Trace, error) {
	if a.TracesOf == nil {
		return nil, nil
	}
	ts, err := a.TracesOf(ctx, user, a.Now().Add(-TraceTTL))
	if err != nil {
		return nil, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	out := ts[:0:0]
	for _, t := range ts {
		if t.User == user && t.Tenant == a.Tenant && a.Now().Sub(t.At) <= TraceTTL && t.Response.Conversation != "" {
			out = append(out, t)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out, nil
}

// syncRecord is the person's memory record, for their synced conversations.
func (a Agent) syncRecord(ctx context.Context, s tenancy.Scope) (Memory, error) {
	if a.Exclusions == nil {
		return Memory{}, fmt.Errorf("%w: conversations can't be kept here", ErrUnavailable)
	}
	m, found, err := a.Exclusions.MemoryOf(ctx, a.Tenant, s.User())
	if err != nil {
		return Memory{}, fmt.Errorf("%w: memory: %v", ErrUnavailable, err)
	}
	if !found {
		m = Memory{Tenant: a.Tenant, User: s.User(), Lines: []MemoryLine{}}
	}
	m.Conversations = a.liveSynced(m.Conversations)
	return m, nil
}

// liveSynced drops the entries whose last answer is past the traces' 30 days.
func (a Agent) liveSynced(cs []SyncedConversation) []SyncedConversation {
	var out []SyncedConversation
	for _, c := range cs {
		if a.Now().Sub(c.At) <= TraceTTL {
			out = append(out, c)
		}
	}
	return out
}

// SetSync turns Keep my conversations on every device on or off; off deletes every entry.
func (a Agent) SetSync(ctx context.Context, s tenancy.Scope, on bool) (Conversations, error) {
	if err := a.ownScope(s); err != nil {
		return Conversations{}, err
	}
	m, err := a.syncRecord(ctx, s)
	if err != nil {
		return Conversations{}, err
	}
	m.Sync, m.At = on, a.Now().UTC()
	if !on {
		m.Conversations = nil
	}
	if err := a.Exclusions.PutMemory(ctx, m); err != nil {
		return Conversations{}, fmt.Errorf("%w: memory: %v", ErrUnavailable, err)
	}
	return a.ListConversations(ctx, s)
}

// noteSynced keeps one web ask in the person's synced list when they turned it on. A record
// that can't be read keeps nothing; the answer stands either way.
func (a Agent) noteSynced(ctx context.Context, s tenancy.Scope, conv, q, trace string, keep, drop []string) {
	if a.Exclusions == nil || tenancy.ClientOf(ctx) != tenancy.ClientWeb {
		return
	}
	m, found, err := a.Exclusions.MemoryOf(ctx, a.Tenant, s.User())
	if err != nil || !found || !m.Sync {
		return
	}
	m.Conversations = a.liveSynced(m.Conversations)
	now := a.Now().UTC()
	i := -1
	for j, c := range m.Conversations {
		if c.ID == conv {
			i = j
		}
	}
	if i < 0 {
		m.Conversations = append(m.Conversations, SyncedConversation{ID: conv})
		i = len(m.Conversations) - 1
	}
	c := &m.Conversations[i]
	c.Questions, c.Traces, c.At = append(c.Questions, q), append(c.Traces, trace), now
	c.Kept, c.Dropped = union(c.Kept, keep), union(c.Dropped, drop)
	sort.SliceStable(m.Conversations, func(x, y int) bool { return m.Conversations[x].At.After(m.Conversations[y].At) })
	if len(m.Conversations) > MaxSyncedConversations {
		m.Conversations = m.Conversations[:MaxSyncedConversations]
	}
	m.At = now
	_ = a.Exclusions.PutMemory(ctx, m)
}

func union(a, b []string) []string {
	for _, x := range b {
		if !contains(a, x) {
			a = append(a, x)
		}
	}
	return a
}

// ListConversations is the person's conversations: those they started in a chat app, and,
// with Keep my conversations on every device on, those they started on the web. Newest first.
func (a Agent) ListConversations(ctx context.Context, s tenancy.Scope) (Conversations, error) {
	if err := a.ownScope(s); err != nil {
		return Conversations{}, err
	}
	out := Conversations{Conversations: []ConversationItem{}}
	ts, err := a.tracesOf(ctx, s.User())
	if err != nil {
		return Conversations{}, err
	}
	byID := map[string]*ConversationItem{}
	var order []string
	for _, t := range ts {
		c := byID[t.Response.Conversation]
		if c == nil {
			c = &ConversationItem{ID: t.Response.Conversation, Client: clientOf(t)}
			byID[c.ID] = c
			order = append(order, c.ID)
		}
		c.Turns++
		c.At = t.At
	}
	if a.Exclusions != nil {
		m, found, err := a.Exclusions.MemoryOf(ctx, a.Tenant, s.User())
		if err != nil {
			return Conversations{}, fmt.Errorf("%w: memory: %v", ErrUnavailable, err)
		}
		if found && m.Sync {
			out.Sync = true
			for _, sc := range a.liveSynced(m.Conversations) {
				if c := byID[sc.ID]; c != nil {
					c.Synced = true
				}
			}
		}
	}
	for _, id := range order {
		c := byID[id]
		if c.Client != tenancy.ClientChat && !c.Synced {
			continue // a web conversation lives in its browser unless the person syncs it
		}
		qs, found, err := a.Store.Questions(ctx, a.Tenant, s.User(), id)
		if err != nil {
			return Conversations{}, fmt.Errorf("%w: conversations: %v", ErrUnavailable, err)
		}
		if !found || len(qs) == 0 {
			continue
		}
		c.First = qs[0]
		out.Conversations = append(out.Conversations, *c)
	}
	sort.SliceStable(out.Conversations, func(i, j int) bool { return out.Conversations[i].At.After(out.Conversations[j].At) })
	if len(out.Conversations) > MaxListedConversations {
		out.Conversations = out.Conversations[:MaxListedConversations]
	}
	return out, nil
}

func clientOf(t Trace) string {
	if t.Client == "" {
		return tenancy.ClientWeb
	}
	return t.Client
}

// OpenConversation is one of the person's own conversations to continue, each answer
// checked again as them now.
func (a Agent) OpenConversation(ctx context.Context, s tenancy.Scope, id string) (ConversationView, error) {
	if err := a.ownScope(s); err != nil {
		return ConversationView{}, err
	}
	qs, found, err := a.Store.Questions(ctx, a.Tenant, s.User(), id)
	if err != nil {
		return ConversationView{}, fmt.Errorf("%w: conversations: %v", ErrUnavailable, err)
	}
	if !found {
		return ConversationView{}, ErrNoConversation
	}
	ts, err := a.tracesOf(ctx, s.User())
	if err != nil {
		return ConversationView{}, err
	}
	v := ConversationView{ID: id, Turns: []Turn{}}
	for _, t := range ts {
		if t.Response.Conversation != id {
			continue
		}
		turn, ok, err := a.turnAs(ctx, s, t)
		if err != nil {
			return ConversationView{}, err
		}
		if t.Turn < len(qs) {
			turn.Question = qs[t.Turn]
		}
		if !ok {
			turn.Unavailable, turn.Claims, turn.Label = true, nil, ""
		}
		v.Turns = append(v.Turns, turn)
	}
	return v, nil
}

// turnAs is a stored answer as s may read it now: ok only when s can open every cite.
func (a Agent) turnAs(ctx context.Context, s tenancy.Scope, t Trace) (Turn, bool, error) {
	var cites []string
	for _, c := range t.Response.Answer.Claims {
		for _, ci := range c.Cites {
			cites = append(cites, ci.Cite)
		}
	}
	open, err := a.opensCites(ctx, s, cites)
	if err != nil {
		return Turn{}, false, err
	}
	for _, c := range cites {
		if !open[c] {
			return Turn{Trace: t.ID, At: t.At}, false, nil
		}
	}
	return Turn{Trace: t.ID, At: t.At, Claims: t.Response.Answer.Claims, Label: t.Response.Label}, true, nil
}

// ConversationShares share whole conversations (SB-5804), as shares of one question do.
type ConversationShares struct {
	Agent     Agent
	Questions Questions
}

// Share makes a link to one of s's own conversations, for colleagues of the same tenant.
// The audit row carries the conversation's hash only.
func (x ConversationShares) Share(ctx context.Context, s tenancy.Scope, id string) (Share, error) {
	a := x.Agent
	v, err := a.OpenConversation(ctx, s, id)
	if err != nil {
		return Share{}, err
	}
	if len(v.Turns) == 0 {
		return Share{}, ErrNoConversation
	}
	tok := x.Questions.Token
	if tok == nil {
		tok = randomToken
	}
	t, err := tok()
	if err != nil {
		return Share{}, fmt.Errorf("%w: share id: %v", ErrUnavailable, err)
	}
	now := a.Now().UTC()
	sh := Share{ID: "sh_" + t, Tenant: a.Tenant, User: s.User(), Space: s.Space(), At: now, Expires: now.Add(ShareTTL)}
	for _, turn := range v.Turns {
		sh.Traces, sh.Questions = append(sh.Traces, turn.Trace), append(sh.Questions, turn.Question)
	}
	sh.Question = sh.Questions[0]
	if !shareID.MatchString(sh.ID) {
		return Share{}, fmt.Errorf("%w: share id", ErrUnavailable)
	}
	h := sha256.Sum256([]byte(id))
	if _, err := x.Questions.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: sh.ID, State: "shared",
		PayloadSHA256: hex.EncodeToString(h[:]), Reason: "conversation shared", Space: auditSpace(s.Space())}); err != nil {
		return Share{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Questions.Store.PutShare(ctx, sh); err != nil {
		return Share{}, fmt.Errorf("%w: shares: %v", ErrUnavailable, err)
	}
	return sh, nil
}

// Open is a shared conversation as s reads it: each answer checked as s now, one they can't
// open every cite of left out and counted, and the highest label of what they see (LAW-26,
// LAW-32).
func (x ConversationShares) Open(ctx context.Context, s tenancy.Scope, id string) (ConversationView, error) {
	a := x.Agent
	sh, err := x.open(ctx, s, id)
	if err != nil {
		return ConversationView{}, err
	}
	v := ConversationView{Turns: []Turn{}}
	for i, tid := range sh.Traces {
		t, found, err := a.Store.GetTrace(ctx, a.Tenant, tid)
		if err != nil {
			return ConversationView{}, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
		}
		if !found || t.User != sh.User || t.Tenant != a.Tenant || a.Now().Sub(t.At) > TraceTTL {
			v.Left++
			continue
		}
		turn, ok, err := a.turnAs(ctx, s, t)
		if err != nil {
			return ConversationView{}, err
		}
		if !ok {
			v.Left++
			continue
		}
		if i < len(sh.Questions) {
			turn.Question = sh.Questions[i]
		}
		if ingest.Rank(turn.Label) > ingest.Rank(v.Label) {
			v.Label = turn.Label
		}
		v.Turns = append(v.Turns, turn)
	}
	return v, nil
}

// Revoke ends a conversation share; only the person who shared it may.
func (x ConversationShares) Revoke(ctx context.Context, s tenancy.Scope, id string) error {
	sh, err := x.open(ctx, s, id)
	if err != nil {
		return err
	}
	if sh.User != s.User() {
		return ErrNoShare
	}
	sh.Expires = x.Agent.Now().UTC()
	if _, err := x.Questions.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: sh.ID, State: "revoked",
		PayloadSHA256: hashOf(sh.ID), Reason: "conversation share revoked", Space: auditSpace(sh.Space)}); err != nil {
		return fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	rv, ok := x.Questions.Store.(ShareRevoker)
	if !ok {
		return fmt.Errorf("%w: shares can't be revoked here", ErrUnavailable)
	}
	if err := rv.RevokeShare(ctx, x.Agent.Tenant, sh.ID, sh.Expires); err != nil {
		return fmt.Errorf("%w: shares: %v", ErrUnavailable, err)
	}
	return nil
}

func (x ConversationShares) open(ctx context.Context, s tenancy.Scope, id string) (Share, error) {
	a := x.Agent
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant || !shareID.MatchString(id) {
		return Share{}, ErrNoShare
	}
	sh, found, err := x.Questions.Store.GetShare(ctx, a.Tenant, id)
	if err != nil {
		return Share{}, fmt.Errorf("%w: shares: %v", ErrUnavailable, err)
	}
	if !found || sh.Tenant != a.Tenant || len(sh.Traces) == 0 || !a.Now().Before(sh.Expires) {
		return Share{}, ErrNoShare
	}
	return sh, nil
}
