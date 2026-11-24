package ask

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Caretakers (sb-ask(3), Caretakers; SB-3405): a steward names an active member of the
// space to look after a document whose owner at the source is gone.

// Caretaker is one document's caretaker.
type Caretaker struct {
	Tenant string    `json:"tenant"`
	Doc    string    `json:"doc"`   // KIND:ID
	Space  string    `json:"space"` // spc_NAME or company
	Email  string    `json:"email"`
	By     string    `json:"by"`
	At     time.Time `json:"at"`
}

// CaretakerStore keeps caretakers, one per document.
type CaretakerStore interface {
	PutCaretaker(ctx context.Context, c Caretaker) error
	CaretakerOf(ctx context.Context, tenant, doc string) (Caretaker, bool, error)
	DeleteCaretaker(ctx context.Context, tenant, doc string) error
}

// CaretakerOf is the lookup Agent.Caretaker takes over a store: a failed read is none, so a
// caretaker is never guessed.
func CaretakerOf(st CaretakerStore, tenant string) func(ctx context.Context, doc string) string {
	return func(ctx context.Context, doc string) string {
		c, ok, err := st.CaretakerOf(ctx, tenant, doc)
		if err != nil || !ok {
			return ""
		}
		return c.Email
	}
}

// NameCaretaker names email the caretaker of doc (KIND:ID): doc must be held by the space,
// open to the steward now, and its owner at the source definitely gone; email an active
// member of the space now.
func (x Stewards) NameCaretaker(ctx context.Context, s tenancy.Scope, name, doc, email string) (Caretaker, error) {
	c, err := x.caretakerDoc(ctx, s, name, doc)
	if err != nil {
		return Caretaker{}, err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if x.Members == nil {
		return Caretaker{}, fmt.Errorf("%w: this cell can't list the space's members", ErrUnavailable)
	}
	members, err := x.Members(ctx, SpaceKey(name))
	if err != nil {
		return Caretaker{}, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
	}
	member := false
	for _, m := range members {
		member = member || strings.ToLower(m) == email
	}
	if email == "" || !member {
		return Caretaker{}, fmt.Errorf("%w: a caretaker is an active member of the space", ErrInvalid)
	}
	if err := x.auditCaretaker(ctx, s, name, email, "named"); err != nil {
		return Caretaker{}, err
	}
	ct := Caretaker{Tenant: x.Agent.Tenant, Doc: c.Kind + ":" + c.SourceID, Space: SpaceKey(name), Email: email, By: s.User(), At: x.Now().UTC()}
	if err := x.Caretakers.PutCaretaker(ctx, ct); err != nil {
		return Caretaker{}, fmt.Errorf("%w: caretakers: %v", ErrUnavailable, err)
	}
	return ct, nil
}

// RemoveCaretaker removes doc's caretaker, for a steward of the space.
func (x Stewards) RemoveCaretaker(ctx context.Context, s tenancy.Scope, name, doc string) error {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return err
	}
	if x.Caretakers == nil {
		return fmt.Errorf("%w: this cell keeps no caretakers", ErrUnavailable)
	}
	ct, ok, err := x.Caretakers.CaretakerOf(ctx, x.Agent.Tenant, doc)
	if err != nil {
		return fmt.Errorf("%w: caretakers: %v", ErrUnavailable, err)
	}
	if !ok || ct.Space != SpaceKey(name) {
		return fmt.Errorf("%w: this space names no caretaker for that document", ErrInvalid)
	}
	if err := x.auditCaretaker(ctx, s, name, ct.Email, "removed"); err != nil {
		return err
	}
	if err := x.Caretakers.DeleteCaretaker(ctx, x.Agent.Tenant, doc); err != nil {
		return fmt.Errorf("%w: caretakers: %v", ErrUnavailable, err)
	}
	return nil
}

func (x Stewards) auditCaretaker(ctx context.Context, s tenancy.Scope, name, email, state string) error {
	sum := sha256.Sum256([]byte(email))
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "config", Subject: "caretaker:" + name, State: state,
		PayloadSHA256: hex.EncodeToString(sum[:]), Reason: "document caretaker", Space: auditSpace(SpaceKey(name))}); err != nil {
		return fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	return nil
}

// caretakerDoc is doc's newest first chunk, when the space holds it, the steward can open
// it now and its owner is definitely gone.
func (x Stewards) caretakerDoc(ctx context.Context, s tenancy.Scope, name, doc string) (ports.Chunk, error) {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return ports.Chunk{}, err
	}
	if x.Caretakers == nil || x.Gone == nil {
		return ports.Chunk{}, fmt.Errorf("%w: this cell keeps no caretakers", ErrUnavailable)
	}
	kind, id, ok := strings.Cut(doc, ":")
	if !ok || kind == "" || id == "" {
		return ports.Chunk{}, fmt.Errorf("%w: doc is KIND:ID", ErrInvalid)
	}
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return ports.Chunk{}, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	cs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Refs: []ports.SourceRef{{Kind: kind, ID: id}}})
	if err != nil {
		return ports.Chunk{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	var c ports.Chunk
	found := false
	for _, f := range firsts(latest(cs)) {
		for _, sp := range f.Spaces {
			if sp == SpaceKey(name) || SpaceKey(name) == "company" {
				c, found = f, true
			}
		}
	}
	if !found {
		return ports.Chunk{}, fmt.Errorf("%w: the space holds no such document", ErrInvalid)
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return ports.Chunk{}, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, []ports.Chunk{c}, 1)
	if err != nil {
		return ports.Chunk{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if len(res.Context) == 0 {
		return ports.Chunk{}, fmt.Errorf("%w: the space holds no such document", ErrInvalid) // as if it didn't exist
	}
	if c.Owner == "" {
		return ports.Chunk{}, fmt.Errorf("%w: the document has no owner at the source to stand in for", ErrInvalid)
	}
	gone, err := x.Gone(ctx, c.Owner)
	if err != nil {
		return ports.Chunk{}, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
	}
	if !gone {
		return ports.Chunk{}, fmt.Errorf("%w: the document's owner is still an active user", ErrInvalid)
	}
	return c, nil
}
