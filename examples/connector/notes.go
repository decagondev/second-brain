// Package notes is a minimal example connector for a fictional "Notes" system.
// It shows the shape every connector takes: four calls, no state but a cursor,
// credentials resolved at call time, and Unknown for anything uncertain.
//
// Real connectors live in internal/adapters/connectors/<kind>-<vendor>/ and must
// pass the contract kit (sb-connector(3)) before any manifest can name them.
package notes

import (
	"context"
	"errors"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// Client is the vendor API surface we use. Keeping it an interface lets the
// contract kit run against a recorded sandbox.
type Client interface {
	GetNote(ctx context.Context, token, id string) (Note, error)
	CanView(ctx context.Context, token, userEmail string, ids []string) (map[string]bool, error)
	ListChanges(ctx context.Context, token, since string) ([]Change, string, error)
}

type Note struct {
	ID, Title, Body string
	Version         int
	Viewers         []string // emails and group names; used only as a prefilter
}

type Change struct {
	ID      string
	Deleted bool
	ACLOnly bool
}

type Connector struct {
	client  Client
	secrets ports.Secrets
	tenant  string
}

func New(c Client, s ports.Secrets, tenant string) *Connector {
	return &Connector{client: c, secrets: s, tenant: tenant}
}

// token is fetched per call and never stored on the struct (LAW-07).
func (c *Connector) token(ctx context.Context) (string, error) {
	return c.secrets.Get(ctx, "vault://tenants/"+c.tenant+"/connectors/docs")
}

func (c *Connector) Fetch(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
	tok, err := c.token(ctx)
	if err != nil {
		return ports.Document{}, err
	}
	n, err := c.client.GetNote(ctx, tok, ref.ID)
	if err != nil {
		return ports.Document{}, err
	}
	return ports.Document{
		Ref: ref, Version: n.Version, Title: n.Title, Text: n.Body,
		ACLPrefilter: n.Viewers,
	}, nil
}

// ACL is the authority. Anything we can't answer cleanly is Unknown (LAW-03).
func (c *Connector) ACL(ctx context.Context, user ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs)) // zero value is VerdictUnknown
	tok, err := c.token(ctx)
	if err != nil {
		return out, nil
	}
	ids := make([]string, len(refs))
	for i, r := range refs {
		ids[i] = r.ID
	}
	ctx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()
	can, err := c.client.CanView(ctx, tok, user.Email, ids)
	if err != nil {
		return out, nil
	}
	for i, id := range ids {
		if v, ok := can[id]; ok {
			out[i] = ports.VerdictDeny
			if v {
				out[i] = ports.VerdictAllow
			}
		}
	}
	return out, nil
}

func (c *Connector) Changes(ctx context.Context, cur ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	tok, err := c.token(ctx)
	if err != nil {
		return ports.ChangeSet{}, cur, err
	}
	chs, next, err := c.client.ListChanges(ctx, tok, string(cur))
	if err != nil {
		return ports.ChangeSet{}, cur, err
	}
	var set ports.ChangeSet
	for _, ch := range chs {
		set = append(set, ports.Change{Ref: ports.SourceRef{Kind: "docs", ID: ch.ID}, Deleted: ch.Deleted, ACLOnly: ch.ACLOnly})
	}
	return set, ports.Cursor(next), nil
}

// Act is not supported: this connector is read-only.
func (c *Connector) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, ErrNotSupported
}

var ErrNotSupported = errors.New("ENOTSUP: notes connector is read-only")

var _ ports.Connector = (*Connector)(nil)
