// Package reviews is the quarterly access review of restricted spaces (sb-spaces(7),
// Access reviews, exactly; SB-1202): stewards certify or remove each person who can use
// the space, removals become change requests an operator approves, and a review open 14
// days is overdue.
package reviews

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/groupsync"
	"example.com/second-brain/internal/core/manifest"
)

// States and decisions.
const (
	Open     = "open"
	Complete = "complete"
	Certify  = "certify"
	Remove   = "remove"
	Never    = "never"
)

// Window is how long a review may stay open before it is overdue.
const Window = 14 * 24 * time.Hour

// Item is one person who can use the space, and the steward's decision on them.
type Item struct {
	User     string   `json:"user"`
	Via      []string `json:"via"`      // the member groups that give the access, group:NAME
	LastUse  string   `json:"last_use"` // RFC 3339, or "never"
	Decision string   `json:"decision,omitempty"`
	By       string   `json:"by,omitempty"`
	At       string   `json:"at,omitempty"`
	Reason   string   `json:"reason,omitempty"`
	AuditSeq int64    `json:"audit_seq,omitempty"`
}

// Request is one change a removal asks for: a manifest change (its proposal) or an IdP one.
type Request struct {
	Kind     string `json:"kind"` // manifest or idp
	User     string `json:"user"`
	Group    string `json:"group"`
	Proposal string `json:"proposal,omitempty"`
}

// Review is one space's review for one quarter.
type Review struct {
	ID        string    `json:"id"`
	Space     string    `json:"space"`
	Quarter   string    `json:"quarter"`
	Stewards  []string  `json:"stewards"`
	Opened    string    `json:"opened"`
	Due       string    `json:"due"`
	Completed string    `json:"completed,omitempty"`
	State     string    `json:"state"`
	Items     []Item    `json:"items"`
	Requests  []Request `json:"requests"`
}

// Quarter names the quarter t is in, such as 2027-Q1.
func Quarter(t time.Time) string {
	t = t.UTC()
	return fmt.Sprintf("%d-Q%d", t.Year(), (int(t.Month())-1)/3+1)
}

// Roster lists the user ids in a directory group, named group:NAME. An error is never empty.
type Roster interface {
	Members(ctx context.Context, group string) ([]string, error)
}

// LastUse is when user last used space, from the audit trail: the latest row written in
// the space with them as actor.
type LastUse func(space, user string) (time.Time, bool)

// FromAudit builds a LastUse from audit rows.
func FromAudit(rows []audit.Row) LastUse {
	last := map[string]time.Time{}
	for _, r := range rows {
		if !strings.HasPrefix(r.Space, "spc_") {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, r.At)
		if err != nil {
			continue
		}
		k := strings.TrimPrefix(r.Space, "spc_") + "\x00" + r.Actor
		if at.After(last[k]) {
			last[k] = at
		}
	}
	return func(space, user string) (time.Time, bool) {
		t, ok := last[space+"\x00"+user]
		return t, ok
	}
}

// OpenAll opens a review for every restricted space in t that has none for this quarter
// (have reports whether one exists). It opens none if any group can't be read.
func OpenAll(ctx context.Context, t manifest.Tenant, now time.Time, roster Roster, last LastUse,
	have func(space, quarter string) bool, newID func(prefix string) string) ([]Review, error) {
	names := make([]string, 0, len(t.Spaces))
	for n, sp := range t.Spaces {
		if sp.Label == "restricted" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	q := Quarter(now)
	var out []Review
	for _, n := range names {
		if have != nil && have(n, q) {
			continue
		}
		sp := t.Spaces[n]
		via := map[string][]string{}
		for _, g := range sp.Members {
			ids, err := roster.Members(ctx, g)
			if err != nil {
				return nil, fmt.Errorf("reviews: space %s: group %s can't be read: %w", n, g, err)
			}
			for _, id := range ids {
				if !contains(via[id], g) {
					via[id] = append(via[id], g)
				}
			}
		}
		users := make([]string, 0, len(via))
		for u := range via {
			users = append(users, u)
		}
		sort.Strings(users)
		items := make([]Item, 0, len(users))
		for _, u := range users {
			lu := Never
			if at, ok := last(n, u); ok {
				lu = at.UTC().Format(time.RFC3339)
			}
			items = append(items, Item{User: u, Via: via[u], LastUse: lu})
		}
		out = append(out, Review{ID: newID("rev_"), Space: n, Quarter: q, Stewards: append([]string{}, sp.Stewards...),
			Opened: now.UTC().Format(time.RFC3339), Due: now.Add(Window).UTC().Format(time.RFC3339), State: Open,
			Items: items, Requests: []Request{}})
	}
	return out, nil
}

// Errors from Decide.
var (
	ErrClosed  = errors.New("reviews: the review is complete")
	ErrNoUser  = errors.New("reviews: that person isn't in the review")
	ErrDecided = errors.New("reviews: that person is already decided")
	ErrInvalid = errors.New("reviews: the decision must be certify, or remove with a reason")
	ErrSelf    = errors.New("reviews: nobody decides on their own access")
)

// Decide returns r with user decided by by. r itself is unchanged. When it decides the last
// person the review is complete; its requests come from Requests.
func Decide(r Review, user, decision, reason, by string, now time.Time) (Review, error) {
	switch {
	case r.State != Open:
		return r, ErrClosed
	case decision != Certify && decision != Remove, decision == Remove && strings.TrimSpace(reason) == "":
		return r, ErrInvalid
	case user == by:
		return r, ErrSelf
	}
	next := Clone(r)
	i := -1
	for j, it := range next.Items {
		if it.User == user {
			i = j
		}
	}
	switch {
	case i < 0:
		return r, ErrNoUser
	case next.Items[i].Decision != "":
		return r, ErrDecided
	}
	next.Items[i].Decision, next.Items[i].By, next.Items[i].Reason = decision, by, reason
	next.Items[i].At = now.UTC().Format(time.RFC3339)
	done := true
	for _, it := range next.Items {
		done = done && it.Decision != ""
	}
	if done {
		next.State, next.Completed = Complete, now.UTC().Format(time.RFC3339)
	}
	return next, nil
}

// Requests turns a complete review's removals into change requests against t. For a group
// in t's groups: the removals take the person out of it in the returned manifest (changed
// is true), which goes to an operator as a proposal; for any other group, an IdP request.
// t is never modified.
func Requests(t manifest.Tenant, r Review) (next manifest.Tenant, changed bool, reqs []Request) {
	next = t
	if t.Groups != nil {
		next.Groups = make(map[string][]string, len(t.Groups))
		for k, v := range t.Groups {
			next.Groups[k] = append([]string(nil), v...)
		}
	}
	reqs = []Request{}
	for _, it := range r.Items {
		if it.Decision != Remove {
			continue
		}
		for _, g := range it.Via {
			emails, listed := next.Groups[g]
			if !listed {
				reqs = append(reqs, Request{Kind: "idp", User: it.User, Group: g})
				continue
			}
			kept := emails[:0:0]
			for _, e := range emails {
				if groupsync.UserID(e) != it.User {
					kept = append(kept, e)
				}
			}
			if len(kept) == len(emails) {
				// Listed in the manifest, but not by this person's email: the IdP holds the
				// membership, so it is an IdP request, never dropped (SB-1401).
				reqs = append(reqs, Request{Kind: "idp", User: it.User, Group: g})
				continue
			}
			next.Groups[g], changed = kept, true
			reqs = append(reqs, Request{Kind: "manifest", User: it.User, Group: g})
		}
	}
	return next, changed, reqs
}

// Overdue reports whether r is still open Window after it opened.
func Overdue(r Review, now time.Time) bool {
	opened, err := time.Parse(time.RFC3339, r.Opened)
	if err != nil {
		return r.State == Open // an unreadable time is never "not yet due"
	}
	return r.State == Open && !now.Before(opened.Add(Window))
}

// Alert is one overdue review, for the tenant operator.
type Alert struct {
	Kind     string `json:"kind"` // review_overdue
	Space    string `json:"space"`
	Review   string `json:"review"`
	Severity string `json:"severity"`
	Since    string `json:"since"`
}

// Gauge sets a metric (obs.Metrics).
type Gauge interface {
	Set(name, help string, labels map[string]string, v float64)
}

// Watch sets sb_access_review_overdue for each space with a review and returns an alert for
// each overdue review.
func Watch(rs []Review, now time.Time, g Gauge) []Alert {
	var out []Alert
	spaces := map[string]bool{}
	for _, r := range rs {
		o := Overdue(r, now)
		spaces[r.Space] = spaces[r.Space] || o
		if o {
			out = append(out, Alert{Kind: "review_overdue", Space: r.Space, Review: r.ID, Severity: "sev2", Since: r.Opened})
		}
	}
	if g != nil {
		for s, o := range spaces {
			v := 0.0
			if o {
				v = 1
			}
			g.Set("sb_access_review_overdue", "1 when a space's access review has been open 14 days (sb-spaces(7)).", map[string]string{"space": s}, v)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Review < out[b].Review })
	return out
}

// Evidence is the audit-ready export of a review: the review and the SHA-256 of its JSON.
func Evidence(r Review) ([]byte, error) {
	body, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(body)
	var b bytes.Buffer
	b.WriteString(`{"review":`)
	b.Write(body)
	b.WriteString(`,"sha256":"` + hex.EncodeToString(h[:]) + `"}`)
	return b.Bytes(), nil
}

// VerifyEvidence checks an evidence document's hash and returns its review.
func VerifyEvidence(doc []byte) (Review, error) {
	var e struct {
		Review json.RawMessage `json:"review"`
		SHA256 string          `json:"sha256"`
	}
	if err := json.Unmarshal(doc, &e); err != nil {
		return Review{}, fmt.Errorf("reviews: evidence isn't JSON: %w", err)
	}
	h := sha256.Sum256(e.Review)
	if hex.EncodeToString(h[:]) != e.SHA256 {
		return Review{}, errors.New("reviews: the evidence hash doesn't match")
	}
	var r Review
	if err := json.Unmarshal(e.Review, &r); err != nil {
		return Review{}, err
	}
	return r, nil
}

// Clone is a deep copy of r.
func Clone(r Review) Review {
	c := r
	c.Stewards = append([]string{}, r.Stewards...)
	c.Items = make([]Item, len(r.Items))
	for i, it := range r.Items {
		it.Via = append([]string{}, it.Via...)
		c.Items[i] = it
	}
	c.Requests = append([]Request{}, r.Requests...)
	return c
}

// Store keeps one cell's reviews.
type Store interface {
	Put(ctx context.Context, r Review) error
	Get(ctx context.Context, id string) (r Review, found bool, err error)
	List(ctx context.Context) ([]Review, error)
	// Update applies f to the stored review atomically, and stores the result only if f
	// returns nil: two decisions at once can't overwrite each other (SB-1401).
	Update(ctx context.Context, id string, f func(r *Review) error) (Review, error)
}

// MemoryStore is an in-process Store (development and tests).
type MemoryStore struct {
	mu sync.Mutex
	m  map[string]Review
}

// Put implements Store.
func (s *MemoryStore) Put(_ context.Context, r Review) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[string]Review{}
	}
	s.m[r.ID] = Clone(r)
	return nil
}

// Get implements Store.
func (s *MemoryStore) Get(_ context.Context, id string) (Review, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.m[id]
	if !ok {
		return Review{}, false, nil
	}
	return Clone(r), true, nil
}

// List implements Store, by id.
func (s *MemoryStore) List(_ context.Context) ([]Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Review, 0, len(s.m))
	for _, r := range s.m {
		out = append(out, Clone(r))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// Run is the quarterly review job: it opens the quarter's reviews that don't exist yet and
// stores them. Any error stops it, and the job's missed run alerts (sb-runtime(7)).
func Run(ctx context.Context, t manifest.Tenant, now time.Time, roster Roster, last LastUse, st Store, newID func(string) string) ([]Review, error) {
	all, err := st.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("reviews: the store can't be read: %w", err)
	}
	have := func(space, quarter string) bool {
		for _, r := range all {
			if r.Space == space && r.Quarter == quarter {
				return true
			}
		}
		return false
	}
	opened, err := OpenAll(ctx, t, now, roster, last, have, newID)
	if err != nil {
		return nil, err
	}
	for _, r := range opened {
		if err := st.Put(ctx, r); err != nil {
			return nil, fmt.Errorf("reviews: review %s can't be stored: %w", r.ID, err)
		}
	}
	return opened, nil
}

// ErrNotFound is a review id the store doesn't hold.
var ErrNotFound = errors.New("reviews: no such review")

// Update implements Store.
func (s *MemoryStore) Update(_ context.Context, id string, f func(r *Review) error) (Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.m[id]
	if !ok {
		return Review{}, ErrNotFound
	}
	c := Clone(r)
	if err := f(&c); err != nil {
		return Review{}, err
	}
	s.m[id] = Clone(c)
	return c, nil
}
