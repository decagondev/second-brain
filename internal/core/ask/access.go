package ask

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// Asking for access (sb-ask(3); SB-2205).
const (
	MaxAccessDocs = 20
	AccessWindow  = 7 * 24 * time.Hour // a trace this recent; one request per document a week
)

// AccessResponse is what the asker learns: counts, never which documents or who.
type AccessResponse struct {
	Sent    int `json:"sent"`
	Already int `json:"already"`
	NoOwner int `json:"no_owner"`
}

// AccessStore remembers which requests were sent, by a key that names neither the asker nor
// the document in the clear.
type AccessStore interface {
	AccessRequested(ctx context.Context, tenant, key string) (bool, error)
	PutAccessRequest(ctx context.Context, tenant, key, user string, at time.Time) error
}

// AccessNote tells a document's owner that asker asked a question it answers: the owner's
// email, the asker's, and the document as the owner sees it.
type AccessNote func(ctx context.Context, owner, asker string, doc Result) error

// RequestAccess asks the owners of the documents the source refused in the asker's own
// recent answer or search to share them, once per document a week.
func (a Agent) RequestAccess(ctx context.Context, s tenancy.Scope, trace string, st AccessStore, note AccessNote) (AccessResponse, error) {
	var out AccessResponse
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant {
		return out, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	if st == nil || note == nil {
		return out, fmt.Errorf("%w: this cell can't tell owners", ErrUnavailable)
	}
	t, found, err := a.Store.GetTrace(ctx, a.Tenant, trace)
	if err != nil {
		return out, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	now := a.Now().UTC()
	if !found || t.User != s.User() {
		return out, ErrNotFound
	}
	if now.Sub(t.At) > AccessWindow {
		return out, fmt.Errorf("%w: ask for access within 7 days of the answer", ErrInvalid)
	}
	var cites []string
	seen := map[string]bool{}
	for _, d := range t.Decisions {
		if k := docKey(d.Cite); d.Reason == "source_denied" && !seen[k] && len(cites) < MaxAccessDocs {
			seen[k] = true
			cites = append(cites, d.Cite) // the source refused it: sharing there would open it
		}
	}
	if len(cites) == 0 {
		return out, nil
	}
	lk, ok := a.Index.(ports.CiteLookup)
	if !ok {
		return out, fmt.Errorf("%w: the index can't look up citations", ErrUnavailable)
	}
	cs, err := lk.ChunksOf(ctx, a.Tenant, cites)
	if err != nil {
		return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	asker, err := a.who(ctx, s)
	if err != nil {
		return out, err
	}
	if asker.Email == "" {
		return out, fmt.Errorf("%w: you have no email for the owners to share with", ErrInvalid)
	}
	week := strconv.FormatInt(now.Unix()/int64(AccessWindow/time.Second), 10)
	own := a.owners()
	docs := firsts(cs)
	for _, ci := range cites {
		c, ok := docs[docKey(ci)]
		owner := own.of(ctx, c)
		if !ok || owner == nil {
			out.NoOwner++ // not indexed now, or nobody to ask: counted, never named
			continue
		}
		h := sha256.Sum256([]byte(s.User() + "\x00" + docKey(ci) + "\x00" + week))
		key := hex.EncodeToString(h[:])
		done, err := st.AccessRequested(ctx, a.Tenant, key)
		if err != nil {
			return out, fmt.Errorf("%w: access requests: %v", ErrUnavailable, err)
		}
		if done {
			out.Already++
			continue
		}
		r := a.result(ctx, own, c)
		r.Snippet = ""
		if err := note(ctx, owner.Email, asker.Email, r); err != nil {
			return out, fmt.Errorf("%w: notify: %v", ErrUnavailable, err)
		}
		if err := st.PutAccessRequest(ctx, a.Tenant, key, s.User(), now); err != nil {
			return out, fmt.Errorf("%w: access requests: %v", ErrUnavailable, err)
		}
		out.Sent++
	}
	return out, nil
}
