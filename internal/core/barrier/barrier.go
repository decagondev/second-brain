// Package barrier turns a manifest's information barriers into the facts the policy
// kernel's barrier forbids read (sb-permissions(7), Barriers beyond reads, exactly;
// SB-1203). It never decides anything: the forbids in policy/platform/spaces.cedar do.
package barrier

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/route"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// OwnerGroups resolves a source owner (an email) to their directory groups, as NAME.
type OwnerGroups func(ctx context.Context, owner string) ([]string, error)

// EmailDirectory is the directory as owner lookups need it.
type EmailDirectory interface {
	UsersByEmail(ctx context.Context, tenant, email string) ([]ports.User, error)
	GroupsOf(ctx context.Context, tenant, user string) ([]string, error)
}

// Directory resolves owners through the cell's directory, by any of a user's emails and
// ignoring case (SB-1401: a userName that isn't the email never hides an owner). An owner
// the directory doesn't know (someone outside the company) has no groups; an owner several
// users share has all their groups, so the barrier errs on the side of refusing; a
// directory error is an error.
func Directory(dir EmailDirectory, tenant string) OwnerGroups {
	return func(ctx context.Context, owner string) ([]string, error) {
		users, err := dir.UsersByEmail(ctx, tenant, owner)
		if err != nil {
			return nil, err
		}
		var all []string
		for _, u := range users {
			gs, err := dir.GroupsOf(ctx, tenant, u.ID)
			if err != nil {
				return nil, err
			}
			all = append(all, gs...)
		}
		if all == nil {
			all = []string{}
		}
		return all, nil
	}
}

// Owners is the owners attribute of a chunk: its owner's groups as group:NAME, or an empty
// set for no owner. ok is false when the owner's groups can't be read: the attribute is
// then left out, and the read refuses (LAW-03).
func Owners(ctx context.Context, og OwnerGroups, owner string) (owners []string, ok bool) {
	if owner == "" {
		return []string{}, true
	}
	if og == nil {
		return nil, false
	}
	gs, err := og(ctx, owner)
	if err != nil {
		return nil, false
	}
	return spaces.GroupSet(gs), true
}

// Cell is what a cell with barriers knows to place actions.
type Cell struct {
	// Spaces returns the cell's spaces and barriers now; nil is a cell without spaces.
	Spaces func(ctx context.Context) ([]ports.Space, [][2]string, error)
	// GroupsOf returns a user's directory group names now.
	GroupsOf func(ctx context.Context, user string) ([]string, error)
	// Owners resolves a document owner's groups.
	Owners OwnerGroups
	// Locate fetches a target item ("kind:id") to place it: its location, label and owner.
	Locate func(ctx context.Context, ref ports.SourceRef) (ports.Document, error)
}

// NoteTarget is a steward's note before it is written (sb-connector(3), Notes): the path
// the folder connector writes, notes/SPACE/qs_ID.md, SPACE a space's name or company.
var NoteTarget = regexp.MustCompile(`^notes/([a-z][a-z0-9-]{1,30}|company)/(?:updates/)?qs_[A-Za-z0-9_]{1,40}\.md$`)

// NoteKind is the connector kind notes are written to and indexed from.
const NoteKind = "docs"

// ErrUnplaced is a target or person the cell can't place now: it refuses.
var ErrUnplaced = errors.New("barrier: can't be placed now")

func (c Cell) load(ctx context.Context) ([]ports.Space, [][2]string, bool, error) {
	if c.Spaces == nil {
		return nil, nil, false, nil
	}
	all, bs, err := c.Spaces(ctx)
	if err != nil {
		return nil, nil, false, fmt.Errorf("%w: spaces: %v", ErrUnplaced, err)
	}
	return all, bs, len(bs) > 0, nil
}

// Target is the policy context of a tool call or proposal by s on target: nil for a cell
// without barriers; an error (refuse) if the target can't be placed.
func (c Cell) Target(ctx context.Context, s tenancy.Scope, target string) (map[string]any, error) {
	all, bs, on, err := c.load(ctx)
	if err != nil || !on {
		return nil, err
	}
	var holders []string
	var owners []string
	if m := NoteTarget.FindStringSubmatch(target); m != nil {
		// A note doesn't exist until it is written, but its place does (SB-2001): the docs
		// share's notes/SPACE/ folder, held by the spaces whose scope holds it, and owned
		// by the steward publishing it.
		holders, _ = route.Route(all, NoteKind, "/"+strings.TrimSuffix(target, ".md"), "")
		owners = spaces.GroupSet(s.Groups())
	} else {
		kind, id, ok := strings.Cut(target, ":")
		if !ok || c.Locate == nil {
			return nil, fmt.Errorf("%w: target %q", ErrUnplaced, target)
		}
		doc, err := c.Locate(ctx, ports.SourceRef{Kind: kind, ID: id})
		if err != nil {
			return nil, fmt.Errorf("%w: target %q: %v", ErrUnplaced, target, err)
		}
		holders, _ = route.Route(all, kind, doc.Location, doc.Label)
		if owners, ok = Owners(ctx, c.Owners, strings.ToLower(strings.TrimSpace(doc.Owner))); !ok {
			return nil, fmt.Errorf("%w: the owner of %q", ErrUnplaced, target)
		}
	}
	rc := spaces.ReadContext(all, bs, s.Groups(), spaces.Company)
	return map[string]any{"barred_spaces": rc["barred_spaces"], "barred_groups": spaces.BarredGroups(bs, s.Groups()),
		"target_spaces": holders, "target_owners": owners}, nil
}

// Approval is the policy context of an approval by s of requester's action: nil for a
// cell without barriers; an error (refuse) if the requester's groups can't be read.
func (c Cell) Approval(ctx context.Context, s tenancy.Scope, requester string) (map[string]any, error) {
	_, bs, on, err := c.load(ctx)
	if err != nil || !on {
		return nil, err
	}
	if c.GroupsOf == nil {
		return nil, fmt.Errorf("%w: requester %s", ErrUnplaced, requester)
	}
	gs, err := c.GroupsOf(ctx, requester)
	if err != nil {
		return nil, fmt.Errorf("%w: requester %s: %v", ErrUnplaced, requester, err)
	}
	return map[string]any{"barred_groups": spaces.BarredGroups(bs, s.Groups()), "requester_groups": spaces.GroupSet(gs)}, nil
}
