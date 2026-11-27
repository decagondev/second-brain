// Package inventory is the AI inventory export (sb-admin(3), Inventory; SB-5007): every
// agent, skill, rhythm, shared routine, trigger and starter question the cell runs, with its
// owner, spaces, cap, targets, last run and last change, as CSV and as JSON signed with the
// cell's anchor key. A personal routine is never named: it is only counted per space.
package inventory

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/canonjson"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/proposals"
)

// Kind is the signed document's kind.
const Kind = "ai-inventory"

// The kinds of item.
const (
	ItemAgent   = "agent"
	ItemSkill   = "skill"
	ItemRhythm  = "rhythm"
	ItemRoutine = "shared-routine"
	ItemTrigger = "trigger"
	ItemStarter = "starter"
)

// Item is one thing that runs.
type Item struct {
	Kind       string          `json:"kind"`
	Name       string          `json:"name"`
	Owner      string          `json:"owner,omitempty"`
	Spaces     []string        `json:"spaces"`
	MonthlyUSD *float64        `json:"monthly_usd,omitempty"`
	Job        string          `json:"job,omitempty"`
	Targets    []ask.TargetDef `json:"targets,omitempty"`
	LastRun    string          `json:"last_run,omitempty"`
	LastChange string          `json:"last_change,omitempty"`
	Members    *int            `json:"members,omitempty"` // a shared routine's opted-in members, never who
}

// PersonalRoutines counts one space's personal routines.
type PersonalRoutines struct {
	Space string `json:"space"`
	Count int    `json:"count"`
}

// Inventory is the export.
type Inventory struct {
	Tenant           string             `json:"tenant"`
	Release          string             `json:"release"`
	Made             string             `json:"made"`
	Items            []Item             `json:"items"`
	PersonalRoutines []PersonalRoutines `json:"personal_routines"`
	// Models is every model provider the cell may call, with its region (SB-5705).
	Models []Model `json:"models,omitempty"`
}

// Model is one provider by tier, where it runs, and whether calls may go to it.
type Model struct {
	Tier     string `json:"tier"`
	Provider string `json:"provider"`
	Region   string `json:"region,omitempty"`
	InRegion bool   `json:"in_region"`
}

// Signed is the inventory with its SHA-256 and the anchor key's signature.
type Signed struct {
	Inventory Inventory `json:"inventory"`
	SHA256    string    `json:"sha256"`
	SignedAt  string    `json:"signed_at"`
	Sig       string    `json:"sig"`
	Key       string    `json:"key"`
}

// Agent is one registry row: an agent and the spaces it runs in.
type Agent struct {
	Name   string
	Spaces []string
}

// Inputs are what the export reads.
type Inputs struct {
	Tenant, Release string
	Now             time.Time
	Manifest        manifest.Tenant
	Agents          []Agent // the registry's rows (GET /v1/admin/agents)
	Traces          []ask.Trace
	Questions       []ask.StewardQuestion
	Routines        []ask.SavedQuestion
	Starters        []ask.Suggested
	Proposals       []proposals.Proposal
	RhythmRuns      []ask.RhythmRun
	Models          []Model // SB-5705
}

func stamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// Build makes the inventory, items by kind then name.
func Build(in Inputs) Inventory {
	m := in.Manifest
	inv := Inventory{Tenant: in.Tenant, Release: in.Release, Made: stamp(in.Now), Items: []Item{}, PersonalRoutines: []PersonalRoutines{}, Models: in.Models}
	last := ask.LastUses(in.Questions, in.Traces)
	approved := []proposals.Proposal{}
	for _, p := range in.Proposals {
		if p.State == proposals.Approved {
			approved = append(approved, p)
		}
	}
	// changed is when an approved proposal of one of spaces last touched a line naming all of words.
	changed := func(spaces []string, words ...string) string {
		var at time.Time
		for _, p := range approved {
			if !contains(spaces, p.Space) || !p.At.After(at) {
				continue
			}
			for _, l := range p.Diff {
				all := true
				for _, w := range words {
					all = all && strings.Contains(l, w)
				}
				if all {
					at = p.At
					break
				}
			}
		}
		return stamp(at)
	}
	ran := func(spaces []string, item string) string {
		var at time.Time
		for _, s := range spaces {
			if t := last[s+"\x00"+item]; t.After(at) {
				at = t
			}
		}
		return stamp(at)
	}
	stewards := func(space string) string { return strings.Join(m.Spaces[space].Stewards, " ") }
	goals := m.AgentGoals
	for _, a := range in.Agents {
		def, _ := ask.AgentDefOf(a.Name)
		it := Item{Kind: ItemAgent, Name: a.Name, Owner: def.Owner, Spaces: append([]string{}, a.Spaces...),
			LastRun: ran(a.Spaces, a.Name), LastChange: changed(a.Spaces, "agents", a.Name)}
		if usd, ok := m.AgentCaps[a.Name]; ok {
			it.MonthlyUSD = &usd
		}
		if g, ok := goals[a.Name]; ok {
			it.Job = g.Job
			for _, t := range g.Targets {
				it.Targets = append(it.Targets, ask.TargetDef{Metric: t.Metric, AtLeast: t.AtLeast, AtMost: t.AtMost})
			}
		}
		inv.Items = append(inv.Items, it)
	}
	names := make([]string, 0, len(m.Spaces))
	for n := range m.Spaces {
		names = append(names, n)
	}
	sort.Strings(names)
	skillRun := map[string]time.Time{}
	for _, t := range in.Traces {
		if t.Response.Skill != "" {
			k := strings.TrimPrefix(t.Space, "spc_") + "\x00" + t.Response.Skill
			if t.At.After(skillRun[k]) {
				skillRun[k] = t.At
			}
		}
	}
	rhythmRun := map[string]time.Time{}
	for _, r := range in.RhythmRuns {
		k := r.Space + "\x00" + r.Rhythm
		at := r.Started
		if at.IsZero() {
			at = r.At
		}
		if at.After(rhythmRun[k]) {
			rhythmRun[k] = at
		}
	}
	for _, n := range names {
		sp := m.Spaces[n]
		one := []string{n}
		for _, k := range sortedKeys(sp.Skills) {
			inv.Items = append(inv.Items, Item{Kind: ItemSkill, Name: k, Owner: stewards(n), Spaces: one,
				LastRun: stamp(skillRun[n+"\x00"+k]), LastChange: changed(one, "skills", k)})
		}
		for _, k := range sortedKeys(sp.Rhythms) {
			inv.Items = append(inv.Items, Item{Kind: ItemRhythm, Name: k, Owner: stewards(n), Spaces: one,
				LastRun: stamp(rhythmRun[n+"\x00"+k]), LastChange: changed(one, "rhythms", k)})
		}
		if sp.Brief != nil {
			item := "brief:" + sp.Brief.Agent
			inv.Items = append(inv.Items, Item{Kind: ItemTrigger, Name: item + " weekly " + sp.Brief.Weekly, Owner: stewards(n), Spaces: one,
				LastRun: ran(one, item), LastChange: changed(one, "brief")})
		}
		if sp.Updates {
			inv.Items = append(inv.Items, Item{Kind: ItemTrigger, Name: "updates", Owner: stewards(n), Spaces: one,
				LastRun: ran(one, "updates"), LastChange: changed(one, "updates")})
		}
		for _, on := range sp.On {
			inv.Items = append(inv.Items, Item{Kind: ItemTrigger, Name: on, Owner: stewards(n), Spaces: one, LastChange: changed(one, "on")})
		}
		for _, sg := range in.Starters {
			if sg.Space != ask.SpaceKey(n) {
				continue
			}
			for _, q := range sg.Questions {
				inv.Items = append(inv.Items, Item{Kind: ItemStarter, Name: q, Owner: stewards(n), Spaces: one, LastChange: stamp(sg.At)})
			}
		}
	}
	inv.Items, inv.PersonalRoutines = routinesOf(in.Routines, inv.Items, stewards)
	order := map[string]int{ItemAgent: 0, ItemSkill: 1, ItemRhythm: 2, ItemRoutine: 3, ItemTrigger: 4, ItemStarter: 5}
	sort.SliceStable(inv.Items, func(i, j int) bool {
		a, b := inv.Items[i], inv.Items[j]
		if a.Kind != b.Kind {
			return order[a.Kind] < order[b.Kind]
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return strings.Join(a.Spaces, ",") < strings.Join(b.Spaces, ",")
	})
	return inv
}

// routinesOf adds the shared routines to items, with their members counted, and counts the
// personal routines per space, never naming one.
func routinesOf(rs []ask.SavedQuestion, items []Item, stewards func(string) string) ([]Item, []PersonalRoutines) {
	pr := []PersonalRoutines{}
	members := map[string]int{}
	personal := map[string]int{}
	for _, q := range rs {
		switch {
		case q.Routine == nil:
		case q.From != "":
			members[q.From]++
		case q.Shared == "":
			space := strings.TrimPrefix(q.Space, "spc_")
			if space == "" {
				space = "company"
			}
			personal[space]++
		}
	}
	for _, q := range rs {
		if q.Routine == nil || q.Shared == "" || q.From != "" {
			continue
		}
		n := members[q.ID]
		name := q.ID + " " + q.Routine.Every
		if q.Routine.Day != "" {
			name += " " + q.Routine.Day
		}
		items = append(items, Item{Kind: ItemRoutine, Name: name, Owner: stewards(q.Shared), Spaces: []string{q.Shared},
			LastRun: stamp(q.Routine.Last), Members: &n})
	}
	for _, s := range sortedKeys(personal) {
		pr = append(pr, PersonalRoutines{Space: s, Count: personal[s]})
	}
	return items, pr
}

// CSV is the items, then a row per space of personal routines counted.
func (inv Inventory) CSV() []byte {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"kind", "name", "owner", "spaces", "monthly_usd", "job", "targets", "last_run", "last_change", "members"})
	for _, it := range inv.Items {
		usd, members := "", ""
		if it.MonthlyUSD != nil {
			usd = strconv.FormatFloat(*it.MonthlyUSD, 'f', -1, 64)
		}
		if it.Members != nil {
			members = strconv.Itoa(*it.Members)
		}
		var ts []string
		for _, t := range it.Targets {
			switch {
			case t.AtLeast != nil:
				ts = append(ts, fmt.Sprintf("%s>=%g", t.Metric, *t.AtLeast))
			case t.AtMost != nil:
				ts = append(ts, fmt.Sprintf("%s<=%g", t.Metric, *t.AtMost))
			}
		}
		_ = w.Write([]string{it.Kind, it.Name, it.Owner, strings.Join(it.Spaces, " "), usd, it.Job, strings.Join(ts, " "), it.LastRun, it.LastChange, members})
	}
	for _, p := range inv.PersonalRoutines {
		_ = w.Write([]string{"personal-routines", "", "", p.Space, "", "", "", "", "", strconv.Itoa(p.Count)})
	}
	w.Flush()
	return b.Bytes()
}

func digest(raw []byte) (string, error) {
	c, err := canonjson.EncodeDocument(raw)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(c)
	return hex.EncodeToString(h[:]), nil
}

// Sign signs inv with the cell's anchor key.
func Sign(inv Inventory, sg audit.Signer, now time.Time) (Signed, error) {
	raw, err := json.Marshal(inv)
	if err != nil {
		return Signed{}, err
	}
	sha, err := digest(raw)
	if err != nil {
		return Signed{}, err
	}
	at, sig, key := sg.SignDocument(Kind, sha, now)
	return Signed{Inventory: inv, SHA256: sha, SignedAt: at, Sig: sig, Key: key}, nil
}

// ErrSignature is an export that doesn't verify.
var ErrSignature = errors.New("inventory: the export doesn't verify")

// Verify checks a signed export's hash and signature, and its key against pinned when set.
func Verify(doc []byte, pinned string) (Signed, error) {
	var raw struct {
		Inventory json.RawMessage `json:"inventory"`
		SHA256    string          `json:"sha256"`
		SignedAt  string          `json:"signed_at"`
		Sig       string          `json:"sig"`
		Key       string          `json:"key"`
	}
	if err := json.Unmarshal(doc, &raw); err != nil {
		return Signed{}, fmt.Errorf("%w: it isn't JSON: %v", ErrSignature, err)
	}
	sha, err := digest(raw.Inventory)
	if err != nil || sha != raw.SHA256 {
		return Signed{}, fmt.Errorf("%w: the SHA-256 doesn't match the inventory", ErrSignature)
	}
	var s Signed
	if err := json.Unmarshal(doc, &s); err != nil {
		return Signed{}, fmt.Errorf("%w: %v", ErrSignature, err)
	}
	if pinned != "" && pinned != raw.Key {
		return Signed{}, fmt.Errorf("%w: it was signed by %s, not the pinned key", ErrSignature, raw.Key)
	}
	if !audit.VerifyDocument(Kind, s.Inventory.Tenant, raw.Key, raw.SHA256, raw.SignedAt, raw.Sig) {
		return Signed{}, fmt.Errorf("%w: the signature doesn't verify", ErrSignature)
	}
	return s, nil
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}
