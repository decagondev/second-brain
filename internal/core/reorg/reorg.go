// Package reorg previews a reorg's effect on spaces before the IdP changes (sb-space(1)
// diff; SB-1206). It reads a manifest and a snapshot and changes nothing.
package reorg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/manifest"
)

// Person is one person a reorg moves: their groups before and after.
type Person struct {
	User   string   `json:"user"`
	Before []string `json:"before"`
	After  []string `json:"after"`
}

// Snapshot is the file sb space diff reads.
type Snapshot struct {
	Version int      `json:"version"`
	People  []Person `json:"people"`
}

// Flags.
const (
	LosesAll        = "loses-all"
	GainsRestricted = "gains-restricted"
	Barrier         = "barrier"
)

// Change is one person whose spaces change, or who is flagged.
type Change struct {
	User  string   `json:"user"`
	Gains []string `json:"gains"`
	Loses []string `json:"loses"`
	Flags []string `json:"flags"`
}

// Report is the whole preview.
type Report struct {
	People    []Change `json:"people"`
	Unchanged int      `json:"unchanged"`
}

// ErrSnapshot is a snapshot that breaks its format.
var ErrSnapshot = errors.New("reorg: bad snapshot")

var groupRe = regexp.MustCompile(`^group:[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// Parse reads a snapshot strictly: unknown fields, another version, a person twice, an
// empty user or a group not in group:NAME form are errors.
func Parse(b []byte) (Snapshot, error) {
	var s Snapshot
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&s); err != nil {
		return Snapshot{}, fmt.Errorf("%w: %v", ErrSnapshot, err)
	}
	if d.More() {
		return Snapshot{}, fmt.Errorf("%w: more than one JSON value", ErrSnapshot)
	}
	if s.Version != 1 {
		return Snapshot{}, fmt.Errorf("%w: version must be 1, not %d", ErrSnapshot, s.Version)
	}
	seen := map[string]bool{}
	for i, p := range s.People {
		u := strings.ToLower(strings.TrimSpace(p.User))
		switch {
		case u == "":
			return Snapshot{}, fmt.Errorf("%w: people[%d] has no user", ErrSnapshot, i)
		case seen[u]:
			return Snapshot{}, fmt.Errorf("%w: %s is listed twice", ErrSnapshot, u)
		}
		seen[u] = true
		for _, g := range append(append([]string{}, p.Before...), p.After...) {
			if !groupRe.MatchString(g) {
				return Snapshot{}, fmt.Errorf("%w: %s: %q must be group:NAME", ErrSnapshot, u, g)
			}
		}
	}
	return s, nil
}

// Spaces are the spaces of t that someone with groups can use: those with one of the
// groups among their members, sorted.
func Spaces(t manifest.Tenant, groups []string) []string {
	have := map[string]bool{}
	for _, g := range groups {
		have[g] = true
	}
	var out []string
	for name, sp := range t.Spaces {
		for _, m := range sp.Members {
			if have[m] {
				out = append(out, name)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

// Diff previews snap against t. It reads both and changes neither.
func Diff(t manifest.Tenant, snap Snapshot) Report {
	r := Report{People: []Change{}}
	for _, p := range snap.People {
		before, after := Spaces(t, p.Before), Spaces(t, p.After)
		c := Change{User: strings.ToLower(strings.TrimSpace(p.User)), Gains: minus(after, before), Loses: minus(before, after), Flags: []string{}}
		if len(before) > 0 && len(after) == 0 {
			c.Flags = append(c.Flags, LosesAll)
		}
		for _, s := range c.Gains {
			if t.Spaces[s].Label == "restricted" {
				c.Flags = append(c.Flags, GainsRestricted)
				break
			}
		}
		if straddles(t.Barriers, p.After) {
			c.Flags = append(c.Flags, Barrier)
		}
		if len(c.Gains) == 0 && len(c.Loses) == 0 && len(c.Flags) == 0 {
			r.Unchanged++
			continue
		}
		r.People = append(r.People, c)
	}
	sort.Slice(r.People, func(i, j int) bool { return r.People[i].User < r.People[j].User })
	return r
}

func minus(a, b []string) []string {
	out := []string{}
	for _, x := range a {
		found := false
		for _, y := range b {
			found = found || x == y
		}
		if !found {
			out = append(out, x)
		}
	}
	return out
}

func straddles(barriers [][2]string, groups []string) bool {
	have := map[string]bool{}
	for _, g := range groups {
		have[g] = true
	}
	for _, b := range barriers {
		if have[b[0]] && have[b[1]] {
			return true
		}
	}
	return false
}
