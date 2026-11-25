package ask

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/tenancy"
)

// Saved and shared questions (sb-ask(3), Saved and shared questions; SB-1703). Only the
// question is kept; opening one asks it again, live, as whoever opens it.
const (
	ShareTTL = 30 * 24 * time.Hour
	MaxSaved = 100
)

// SavedQuestion is one question a person kept.
type SavedQuestion struct {
	ID       string    `json:"id"`
	Tenant   string    `json:"-"`
	User     string    `json:"-"`
	Question string    `json:"question"`
	Space    string    `json:"space"`
	At       time.Time `json:"at"`
	// Watch (SB-2104): the owner is told weekly when Sources, the SHA-256 of the
	// question's search results as them, changes. No result is kept.
	Watch   bool   `json:"watch"`
	Sources string `json:"-"`
	// Routine asks it for its owner on a schedule (SB-4302).
	Routine *Routine `json:"routine,omitempty"`
}

// WatchStore changes and lists watched saved questions (SB-2104).
type WatchStore interface {
	// UpdateSaved changes one of user's saved questions; found is false for anyone else's.
	UpdateSaved(ctx context.Context, tenant, user, id string, f func(*SavedQuestion)) (found bool, err error)
	Watched(ctx context.Context, tenant string) ([]SavedQuestion, error)
}

// SourcesHash is the SHA-256 of a search's cites (each carries its version), sorted.
func SourcesHash(cites []string) string {
	c := append([]string(nil), cites...)
	sort.Strings(c)
	h := sha256.Sum256([]byte(strings.Join(c, "\n")))
	return hex.EncodeToString(h[:])
}

// Share is a question passed on by link. It holds no answer.
type Share struct {
	ID       string    `json:"id"`
	Tenant   string    `json:"-"`
	User     string    `json:"-"` // who shared it, for erasure; never shown to the recipient
	Question string    `json:"question"`
	Space    string    `json:"space"`
	At       time.Time `json:"-"`
	Expires  time.Time `json:"expires"`
}

// QuestionStore keeps saved questions and shares, under the tenant's retention.
type QuestionStore interface {
	PutSaved(ctx context.Context, q SavedQuestion) error
	SavedOf(ctx context.Context, tenant, user string) ([]SavedQuestion, error) // newest first
	DeleteSaved(ctx context.Context, tenant, user, id string) (found bool, err error)
	PutShare(ctx context.Context, sh Share) error
	GetShare(ctx context.Context, tenant, id string) (Share, bool, error)
}

// Auditor writes one audit row; the audit log is one.
type Auditor interface {
	Write(ctx context.Context, e audit.Event) (audit.Row, error)
}

// Questions is the saved and shared questions service for one cell.
type Questions struct {
	Tenant string
	Store  QuestionStore
	Audit  Auditor
	NewID  func(prefix string) string
	Now    func() time.Time
	// Token makes a share id's random part; nil is 128 bits from crypto/rand.
	Token func() (string, error)
	// Watches and Sources watch saved questions (SB-2104): Sources is the hash of a
	// question's search results as user, in space, now. Nil Watches can't watch.
	Watches WatchStore
	Sources func(ctx context.Context, user, space, question string) (string, error)
}

// ErrNoShare is an unknown, expired or other tenant's share, or someone else's saved question (EPERM).
var ErrNoShare = errors.New("ask: no such question for this user")

var shareID = regexp.MustCompile(`^sh_[0-9a-f]{32}$`)

func question(s tenancy.Scope, tenant, text string) (string, error) {
	if err := s.Check(); err != nil || s.Tenant() != tenant {
		return "", fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	q := strings.TrimSpace(text)
	if q == "" || utf8.RuneCountInString(q) > MaxQuestion {
		return "", fmt.Errorf("%w: question must be 1 to %d characters", ErrInvalid, MaxQuestion)
	}
	return q, nil
}

// Save keeps a question for the scope's user, in the space of the request.
func (x Questions) Save(ctx context.Context, s tenancy.Scope, text string) (SavedQuestion, error) {
	q, err := question(s, x.Tenant, text)
	if err != nil {
		return SavedQuestion{}, err
	}
	have, err := x.Store.SavedOf(ctx, x.Tenant, s.User())
	if err != nil {
		return SavedQuestion{}, fmt.Errorf("%w: saved: %v", ErrUnavailable, err)
	}
	if len(have) >= MaxSaved {
		return SavedQuestion{}, fmt.Errorf("%w: at most %d saved questions; remove one first", ErrInvalid, MaxSaved)
	}
	sq := SavedQuestion{ID: x.NewID("sq_"), Tenant: x.Tenant, User: s.User(), Question: q, Space: s.Space(), At: x.Now().UTC()}
	if err := x.Store.PutSaved(ctx, sq); err != nil {
		return SavedQuestion{}, fmt.Errorf("%w: saved: %v", ErrUnavailable, err)
	}
	return sq, nil
}

// Saved lists the scope's user's own saved questions.
func (x Questions) Saved(ctx context.Context, s tenancy.Scope) ([]SavedQuestion, error) {
	if err := s.Check(); err != nil || s.Tenant() != x.Tenant {
		return nil, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	out, err := x.Store.SavedOf(ctx, x.Tenant, s.User())
	if err != nil {
		return nil, fmt.Errorf("%w: saved: %v", ErrUnavailable, err)
	}
	if out == nil {
		out = []SavedQuestion{}
	}
	return out, nil
}

// Unsave removes one of the scope's user's saved questions.
func (x Questions) Unsave(ctx context.Context, s tenancy.Scope, id string) error {
	if err := s.Check(); err != nil || s.Tenant() != x.Tenant {
		return ErrNoShare
	}
	found, err := x.Store.DeleteSaved(ctx, x.Tenant, s.User(), id)
	if err != nil {
		return fmt.Errorf("%w: saved: %v", ErrUnavailable, err)
	}
	if !found {
		return ErrNoShare
	}
	return nil
}

// Watch starts or stops watching one of the scope's user's saved questions. Starting
// takes the sources' hash now, so the first weekly check compares with today.
func (x Questions) Watch(ctx context.Context, s tenancy.Scope, id string, on bool) (SavedQuestion, error) {
	if err := s.Check(); err != nil || s.Tenant() != x.Tenant {
		return SavedQuestion{}, ErrNoShare
	}
	if x.Watches == nil || x.Sources == nil {
		return SavedQuestion{}, fmt.Errorf("%w: this cell can't watch questions", ErrUnavailable)
	}
	have, err := x.Store.SavedOf(ctx, x.Tenant, s.User())
	if err != nil {
		return SavedQuestion{}, fmt.Errorf("%w: saved: %v", ErrUnavailable, err)
	}
	var q *SavedQuestion
	for i := range have {
		if have[i].ID == id {
			q = &have[i]
		}
	}
	if q == nil {
		return SavedQuestion{}, ErrNoShare
	}
	hash := ""
	if on {
		if hash, err = x.Sources(ctx, s.User(), q.Space, q.Question); err != nil {
			return SavedQuestion{}, fmt.Errorf("%w: search: %v", ErrUnavailable, err)
		}
	}
	found, err := x.Watches.UpdateSaved(ctx, x.Tenant, s.User(), id, func(sq *SavedQuestion) {
		sq.Watch, sq.Sources = on, hash
	})
	if err != nil {
		return SavedQuestion{}, fmt.Errorf("%w: saved: %v", ErrUnavailable, err)
	}
	if !found {
		return SavedQuestion{}, ErrNoShare
	}
	q.Watch, q.Sources = on, hash
	return *q, nil
}

// Share makes a link for a question. The audit row carries the question's hash only.
func (x Questions) Share(ctx context.Context, s tenancy.Scope, text string) (Share, error) {
	q, err := question(s, x.Tenant, text)
	if err != nil {
		return Share{}, err
	}
	tok := x.Token
	if tok == nil {
		tok = randomToken
	}
	t, err := tok()
	if err != nil {
		return Share{}, fmt.Errorf("%w: share id: %v", ErrUnavailable, err)
	}
	now := x.Now().UTC()
	sh := Share{ID: "sh_" + t, Tenant: x.Tenant, User: s.User(), Question: q, Space: s.Space(), At: now, Expires: now.Add(ShareTTL)}
	if !shareID.MatchString(sh.ID) {
		return Share{}, fmt.Errorf("%w: share id", ErrUnavailable)
	}
	h := sha256.Sum256([]byte(q))
	space := ""
	if s.Space() != "company" {
		space = s.Space()
	}
	// The audit row first: a share nobody can trace never exists (LAW-04's order).
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: sh.ID, State: "shared",
		PayloadSHA256: hex.EncodeToString(h[:]), Reason: "question shared", Space: space}); err != nil {
		return Share{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Store.PutShare(ctx, sh); err != nil {
		return Share{}, fmt.Errorf("%w: shares: %v", ErrUnavailable, err)
	}
	return sh, nil
}

// Open reads a share for any signed-in person of the same tenant; the recipient then asks
// it under their own access.
func (x Questions) Open(ctx context.Context, s tenancy.Scope, id string) (Share, error) {
	if err := s.Check(); err != nil || s.Tenant() != x.Tenant || !shareID.MatchString(id) {
		return Share{}, ErrNoShare
	}
	sh, found, err := x.Store.GetShare(ctx, x.Tenant, id)
	if err != nil {
		return Share{}, fmt.Errorf("%w: shares: %v", ErrUnavailable, err)
	}
	if !found || sh.Tenant != s.Tenant() || !x.Now().Before(sh.Expires) {
		return Share{}, ErrNoShare
	}
	return sh, nil
}

func randomToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
