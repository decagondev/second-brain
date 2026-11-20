// Package mock is an in-memory source system and its connector (SB-201, SB-202). The
// source is fully controllable: documents, viewers, edits, permission changes, an Act
// target that does or doesn't issue ids, and faults injected into the next ACL call.
// It passes the contract kit and is the system laws and evals run against.
package mock

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// Fault is injected into the next ACL call.
type Fault string

const (
	FaultNone     Fault = ""
	Fault500      Fault = "500"
	FaultTimeout  Fault = "timeout"
	FaultNotFound Fault = "404"
)

// ACLDeadline is the caller's deadline for one ACL batch (sb-connector(3)).
const ACLDeadline = 800 * time.Millisecond

type doc struct {
	title, text     string
	location, label string
	version         int
	viewers         map[string]bool // user emails
	deleted         bool
}

type change struct {
	id               string
	deleted, aclOnly bool
}

// Source is the fake company system.
type Source struct {
	mu        sync.Mutex
	kind      string
	docs      map[string]*doc
	log       []change
	faults    []Fault // one per coming ACL call
	scripted  map[string]ports.Verdict
	issuesIDs bool
	receipts  map[ports.IdempotencyKey]string
	effects   int
}

// NewSource makes an empty source of one connector kind (docs, chat, tickets, crm).
func NewSource(kind string) *Source {
	return &Source{kind: kind, docs: map[string]*doc{}, receipts: map[ports.IdempotencyKey]string{}, issuesIDs: true,
		scripted: map[string]ports.Verdict{}}
}

// Put creates or edits a document and records the change.
func (s *Source) Put(id, title, text string, viewers ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.docs[id]
	if !ok {
		d = &doc{viewers: map[string]bool{}}
		s.docs[id] = d
	}
	d.title, d.text, d.deleted = title, text, false
	d.version++
	if viewers != nil {
		d.viewers = map[string]bool{}
		for _, v := range viewers {
			d.viewers[v] = true
		}
	}
	s.log = append(s.log, change{id: id})
}

// PutAt is Put with the document's location and source label, in one version (SB-1005).
func (s *Source) PutAt(id, title, text, location, label string, viewers ...string) {
	s.Put(id, title, text, viewers...)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[id].location, s.docs[id].label = location, label
}

// Place sets where a document lives and its source label (SB-1005): a content change, as
// moving a file or changing its classification is at a real source.
func (s *Source) Place(id, location, label string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.docs[id]; ok {
		d.location, d.label = location, label
		d.version++
		s.log = append(s.log, change{id: id})
	}
}

// SetViewers changes who may open a document: a permission-only change.
func (s *Source) SetViewers(id string, viewers ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.docs[id]; ok {
		d.viewers = map[string]bool{}
		for _, v := range viewers {
			d.viewers[v] = true
		}
		s.log = append(s.log, change{id: id, aclOnly: true})
	}
}

// Delete removes a document at the source.
func (s *Source) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.docs[id]; ok {
		d.deleted = true
		s.log = append(s.log, change{id: id, deleted: true})
	}
}

// InjectFault makes the next ACL call fail as f (FaultNone clears the queue).
func (s *Source) InjectFault(f Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.faults = nil
	if f != FaultNone {
		s.faults = []Fault{f}
	}
}

// QueueFaults scripts the coming ACL calls: one fault each, in order.
func (s *Source) QueueFaults(fs ...Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.faults = append(s.faults, fs...)
}

// ScriptVerdict forces the verdict for one user and document, e.g. VerdictUnknown for an
// unmapped permission model. It overrides the viewers until cleared with ClearScript.
func (s *Source) ScriptVerdict(email, id string, v ports.Verdict) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scripted[email+"\x00"+id] = v
}

// ClearScript removes every scripted verdict.
func (s *Source) ClearScript() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scripted = map[string]ports.Verdict{}
}

// IssueIDs sets whether the Act target returns an id for a change.
func (s *Source) IssueIDs(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.issuesIDs = v
}

// Effects is how many changes the Act target has applied.
func (s *Source) Effects() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.effects
}

// Connector is the mock's connector. writable controls whether Act is supported.
type Connector struct {
	src      *Source
	writable bool
}

var _ ports.Connector = (*Connector)(nil)

// New returns a connector over src.
func New(src *Source, writable bool) *Connector { return &Connector{src: src, writable: writable} }

func (c *Connector) APIVersion() int { return ports.ConnectorAPIVersion }

// ErrNotFound is a fetch of a document the source doesn't have.
var ErrNotFound = errors.New("EUNAVAIL: no such document at the source")

func (c *Connector) Fetch(_ context.Context, ref ports.SourceRef) (ports.Document, error) {
	c.src.mu.Lock()
	defer c.src.mu.Unlock()
	d, ok := c.src.docs[ref.ID]
	if !ok || d.deleted {
		return ports.Document{}, ErrNotFound
	}
	var pre []string
	for v := range d.viewers {
		pre = append(pre, v)
	}
	return ports.Document{Ref: ref, Version: d.version, Title: d.title, Text: d.text, ACLPrefilter: pre, Location: d.location, Label: d.label}, nil
}

// ACL answers live. Every fault makes every verdict Unknown; so does a late answer.
func (c *Connector) ACL(ctx context.Context, user ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs)) // zero value is VerdictUnknown (LAW-03)
	ctx, cancel := context.WithTimeout(ctx, ACLDeadline)
	defer cancel()
	c.src.mu.Lock()
	fault := FaultNone
	if len(c.src.faults) > 0 {
		fault, c.src.faults = c.src.faults[0], c.src.faults[1:]
	}
	c.src.mu.Unlock()
	switch fault {
	case Fault500:
		return out, nil
	case FaultTimeout:
		<-ctx.Done()
		return out, nil
	case FaultNotFound, FaultNone:
	}
	c.src.mu.Lock()
	defer c.src.mu.Unlock()
	for i, r := range refs {
		d, ok := c.src.docs[r.ID]
		if fault == FaultNotFound || !ok || d.deleted {
			continue // Unknown
		}
		if v, ok := c.src.scripted[user.Email+"\x00"+r.ID]; ok {
			out[i] = v
			continue
		}
		out[i] = ports.VerdictDeny
		if d.viewers[user.Email] {
			out[i] = ports.VerdictAllow
		}
	}
	if ctx.Err() != nil {
		return make([]ports.Verdict, len(refs)), nil
	}
	return out, nil
}

// Changes returns everything after cursor; the cursor is the log position.
func (c *Connector) Changes(_ context.Context, cur ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	c.src.mu.Lock()
	defer c.src.mu.Unlock()
	from := 0
	if cur != "" {
		n, err := strconv.Atoi(string(cur))
		if err != nil || n < 0 || n > len(c.src.log) {
			return nil, cur, fmt.Errorf("mock: bad cursor %q", cur)
		}
		from = n
	}
	var set ports.ChangeSet
	for _, ch := range c.src.log[from:] {
		set = append(set, ports.Change{Ref: ports.SourceRef{Kind: c.src.kind, ID: ch.id}, Deleted: ch.deleted, ACLOnly: ch.aclOnly})
	}
	return set, ports.Cursor(strconv.Itoa(len(c.src.log))), nil
}

// ErrNotSupported is Act on a read-only connector.
var ErrNotSupported = errors.New("ENOTSUP: this connector is read-only")

// ErrNoReceipt is a change the target applied without issuing an id.
var ErrNoReceipt = errors.New("ENORECEIPT: the target issued no id for the change")

// Act applies one change, once per key, and returns the target's id for it.
func (c *Connector) Act(_ context.Context, a ports.Action, key ports.IdempotencyKey) (ports.Receipt, error) {
	if !c.writable {
		return ports.Receipt{}, ErrNotSupported
	}
	c.src.mu.Lock()
	defer c.src.mu.Unlock()
	if id, done := c.src.receipts[key]; done {
		return ports.Receipt{TargetID: id}, nil
	}
	if !c.src.issuesIDs {
		return ports.Receipt{}, ErrNoReceipt
	}
	c.src.effects++
	id := fmt.Sprintf("%s-%d", a.Target, c.src.effects)
	c.src.receipts[key] = id
	return ports.Receipt{TargetID: id}, nil
}
