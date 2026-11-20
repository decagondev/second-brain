// Package route gives every document version its spaces and its label before any chunk
// is written (sb-ingest(7), Spaces and labels; SB-1005). It is pure: the same location,
// source label and spaces always give the same answer, so a rebuild routes every
// document exactly as before (LAW-13).
package route

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"example.com/second-brain/internal/core/ports"
)

// Ladder is the labels, least to most sensitive (sb-spaces(7), Labels).
var Ladder = []string{"public", "internal", "confidential", "restricted"}

func rank(label string) int {
	for i, l := range Ladder {
		if l == label {
			return i
		}
	}
	return -1
}

// Default is the label of a document no space holds and the source doesn't label.
const Default = "internal"

// Match reports whether a scope pattern matches a location: ** matches anything, *
// matches anything but '/', and every other character matches itself.
func Match(pattern, location string) bool {
	p, s := []rune(pattern), []rune(location)
	var m func(i, j int) bool
	memo := map[[2]int]bool{}
	seen := map[[2]int]bool{}
	m = func(i, j int) bool {
		k := [2]int{i, j}
		if seen[k] {
			return memo[k]
		}
		seen[k] = true
		var ok bool
		switch {
		case i == len(p):
			ok = j == len(s)
		case p[i] == '*' && i+1 < len(p) && p[i+1] == '*':
			ok = m(i+2, j) || (j < len(s) && m(i, j+1))
		case p[i] == '*':
			ok = m(i+1, j) || (j < len(s) && s[j] != '/' && m(i, j+1))
		default:
			ok = j < len(s) && s[j] == p[i] && m(i+1, j+1)
		}
		memo[k] = ok
		return ok
	}
	return m(0, 0)
}

// Route returns the ids of the spaces that hold a document of this connector kind at this
// location, sorted, and its label: a known source label wins, an unknown one is
// restricted, and otherwise the strictest default among the holders, or Default.
func Route(spaces []ports.Space, kind, location, sourceLabel string) (holders []string, label string) {
	holders = []string{}
	strictest := -1
	for _, sp := range spaces {
		for _, pat := range sp.Scope[kind] {
			if Match(pat, location) {
				holders = append(holders, sp.ID)
				if r := rank(sp.Label); r > strictest {
					strictest = r
				} else if r < 0 {
					strictest = len(Ladder) - 1 // a space default the cell doesn't know is restricted
				}
				break
			}
		}
	}
	sort.Strings(holders)
	switch {
	case sourceLabel != "" && rank(sourceLabel) >= 0:
		label = sourceLabel
	case sourceLabel != "":
		label = "restricted"
	case strictest >= 0:
		label = Ladder[strictest]
	default:
		label = Default
	}
	return holders, label
}

// Fingerprint identifies a space configuration for routing: every space's id, label and
// scope. A cell without spaces has the empty fingerprint.
func Fingerprint(spaces []ports.Space) string {
	if len(spaces) == 0 {
		return ""
	}
	type entry struct {
		ID, Label string
		Scope     map[string][]string
	}
	es := make([]entry, len(spaces))
	for i, s := range spaces {
		es[i] = entry{s.ID, s.Label, s.Scope}
	}
	sort.Slice(es, func(i, j int) bool { return es[i].ID < es[j].ID })
	b, _ := json.Marshal(es) // map keys marshal sorted
	h := sha256.Sum256(b)
	return "rt_" + hex.EncodeToString(h[:8])
}
