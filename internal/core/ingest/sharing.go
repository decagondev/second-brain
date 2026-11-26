package ingest

import (
	"context"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/ports"
)

// FlagSecretRemoved is on every chunk of a document that had a chunk dropped for holding a
// secret (sb-ingest(7), Secrets; SB-5401): the secret is gone, but the document held one.
const FlagSecretRemoved = "secret_removed"

// SharingChange is how a document's ACL copy changed between two indexed versions
// (sb-ingest(7), Sharing changes; SB-5403): the grants added and removed, never content.
type SharingChange struct {
	Ref       ports.SourceRef
	Version   int
	Owner     string
	Spaces    []string
	Added     []string // grants the new copy has that the old one didn't
	Removed   []string
	Sensitive bool // personal data, a secret it held, or a confidential or restricted label
}

// SensitiveLabel is confidential or restricted (sb-spaces(7), Labels).
func SensitiveLabel(l string) bool { return l == "confidential" || l == "restricted" }

// Changed says the ACL copy changed at all.
func (c SharingChange) Changed() bool { return len(c.Added) > 0 || len(c.Removed) > 0 }

// Widened says a group, a domain or anyone with the link was added: a person added by name
// is not a widening anyone is told about.
func (c SharingChange) Widened() bool {
	for _, g := range c.Added {
		if IsBroadGrant(g) {
			return true
		}
	}
	return false
}

// IsBroadGrant is a grant to more than one named person: a group, a domain or anyone.
func IsBroadGrant(g string) bool {
	return g == "anyone" || strings.HasPrefix(g, "domain:") || strings.HasPrefix(g, "group:")
}

// SharingChangeOf compares two ACL copies.
func SharingChangeOf(before, after []string) SharingChange {
	was, is := set(before), set(after)
	var ch SharingChange
	for g := range is {
		if !was[g] {
			ch.Added = append(ch.Added, g)
		}
	}
	for g := range was {
		if !is[g] {
			ch.Removed = append(ch.Removed, g)
		}
	}
	sort.Strings(ch.Added)
	sort.Strings(ch.Removed)
	return ch
}

func set(xs []string) map[string]bool {
	out := map[string]bool{}
	for _, x := range xs {
		out[x] = true
	}
	return out
}

// aclBefore is the ACL copy of the newest indexed version of ref, or nil when the index
// can't say or holds none: a document seen for the first time has no change.
func (p Pipeline) aclBefore(ctx context.Context, ref ports.SourceRef) []string {
	lk, ok := p.Index.(ports.DocLookup)
	if !ok {
		return nil
	}
	cs, err := lk.Docs(ctx, p.Tenant, ports.DocQuery{Refs: []ports.SourceRef{ref}})
	if err != nil || len(cs) == 0 {
		return nil
	}
	top := cs[0]
	for _, c := range cs {
		if c.Version > top.Version {
			top = c
		}
	}
	return append([]string{}, top.ACLPrefilter...)
}

func sensitiveChunks(cs []ports.Chunk) bool {
	for _, c := range cs {
		if SensitiveLabel(c.Label) {
			return true
		}
		for _, f := range c.Flags {
			if f == FlagPersonal || f == FlagSecretRemoved {
				return true
			}
		}
	}
	return false
}
