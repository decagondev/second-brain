// Package groupsync applies a manifest's groups: list to the cell's directory, for a
// company that signs in through Identity Platform and runs no SCIM (sb-identity(7),
// Identity Platform, exactly; SB-907). The manifest is the source of the directory:
// someone listed in a group is an active user, someone no longer listed is deactivated
// and their sessions revoked, and session re-validation keeps the 5-minute bound (LAW-12)
// even if that revocation is lost.
package groupsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/ports"
)

// UserID is the directory id of the person with this email. The Identity Platform
// verifier finds the user by the token's verified email through the same function.
func UserID(email string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return "usr_ip_" + hex.EncodeToString(h[:10])
}

// GroupID is the directory id of a manifest group.
func GroupID(name string) string { return "grp_ip_" + strings.TrimPrefix(name, "group:") }

// Result counts what Apply changed.
type Result struct{ Active, Deactivated, Revoked int }

// Apply makes the directory match groups. prev is the groups of the manifest applied
// before (nil for the first apply), so people and groups it listed and groups no longer
// lists are removed. Any error stops the apply; the caller retries the whole of it.
func Apply(ctx context.Context, dir ports.DirectoryStore, sess ports.Sessions, tenant string, prev, groups map[string][]string) (Result, error) {
	var res Result
	members := map[string]string{} // user id -> email
	for _, ms := range groups {
		for _, e := range ms {
			members[UserID(e)] = strings.ToLower(e)
		}
	}
	for _, id := range sortedKeys(members) {
		e := members[id]
		if err := dir.PutUser(ctx, ports.User{ID: id, Tenant: tenant, UserName: e, Emails: []string{e}, Active: true}); err != nil {
			return res, fmt.Errorf("groupsync: user %s: %w", id, err)
		}
		res.Active++
	}
	for _, name := range sortedKeys(groups) {
		ids := make([]string, 0, len(groups[name]))
		for _, e := range groups[name] {
			ids = append(ids, UserID(e))
		}
		sort.Strings(ids)
		if err := dir.PutGroup(ctx, ports.Group{ID: GroupID(name), Tenant: tenant, Name: strings.TrimPrefix(name, "group:"), Members: ids}); err != nil {
			return res, fmt.Errorf("groupsync: group %s: %w", name, err)
		}
	}
	for _, name := range sortedKeys(prev) {
		if _, still := groups[name]; !still {
			if err := dir.DeleteGroup(ctx, tenant, GroupID(name)); err != nil {
				return res, fmt.Errorf("groupsync: group %s: %w", name, err)
			}
		}
	}
	gone := map[string]string{}
	for _, ms := range prev {
		for _, e := range ms {
			if id := UserID(e); members[id] == "" {
				gone[id] = strings.ToLower(e)
			}
		}
	}
	for _, id := range sortedKeys(gone) {
		e := gone[id]
		if err := dir.PutUser(ctx, ports.User{ID: id, Tenant: tenant, UserName: e, Emails: []string{e}, Active: false}); err != nil {
			return res, fmt.Errorf("groupsync: user %s: %w", id, err)
		}
		res.Deactivated++
		n, err := sess.RevokeUser(ctx, tenant, id)
		if err != nil {
			return res, fmt.Errorf("groupsync: revoke %s: %w", id, err)
		}
		res.Revoked += n
	}
	return res, nil
}

func sortedKeys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
