package ask

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Did it help? (sb-agents(7), Did it help?; SB-5005): a change that shipped (a steward
// proposal approved, a policy marked, a fix applied, an agent's new release) is checked
// CheckAfter later, comparing the CheckAfter before it with the CheckAfter after it in its
// space, or for its agent. The result is a short draft for the space's stewards or the
// agent's owner: better, worse, no clear change, or not enough data. It never undoes
// anything.
const (
	CheckAfter   = 28 * 24 * time.Hour
	CheckMinN    = 20   // answers each side, or "not enough data"
	CheckRateBy  = 0.05 // a rate moves clearly by 5 points
	CheckHoursBy = 0.1  // a median moves clearly by a tenth
)

// The kinds of change a check follows.
const (
	ChangeSkill   = "skill"
	ChangeAgents  = "agents"
	ChangeRouting = "routing"
	ChangeSpace   = "space"
	ChangePolicy  = "policy"
	ChangeFix     = "fix"
	ChangeRelease = "release"
)

// The verdicts.
const (
	VerdictBetter  = "better"
	VerdictWorse   = "worse"
	VerdictNoClear = "no clear change"
	VerdictTooFew  = "not enough data"
)

// Change is one shipped change: its source record (an audit row id, a proposal or a
// release), the space it changed (a name), or the agent, and when.
type Change struct {
	Kind   string    `json:"kind"`
	Ref    string    `json:"ref"`
	Space  string    `json:"space,omitempty"`
	Agent  string    `json:"agent,omitempty"`
	At     time.Time `json:"at"`
	By     string    `json:"by,omitempty"`
	Source string    `json:"source"` // the record to open: an audit row's link or a route
}

// Key is the change's one check, whatever runs it.
func (c Change) Key() string {
	sum := sha256.Sum256([]byte(c.Kind + "\x00" + c.Ref + "\x00" + c.Space + "\x00" + c.Agent + "\x00" + c.At.UTC().Format(time.RFC3339)))
	return "chk_" + hex.EncodeToString(sum[:8])
}

// CheckNumbers are one side's counts: answers, those that found nothing, thumbs, drafts
// and discards, and the median hours to a steward's first reply.
type CheckNumbers struct {
	Answers     int      `json:"answers"`
	Gaps        int      `json:"gaps"`
	GapRate     *float64 `json:"gap_rate,omitempty"`
	Rated       int      `json:"rated"`
	HelpfulRate *float64 `json:"helpful_rate,omitempty"`
	Drafts      int      `json:"drafts"`
	DiscardRate *float64 `json:"discard_rate,omitempty"`
	Replies     int      `json:"replies"`
	ReplyHours  *float64 `json:"median_reply_hours,omitempty"`
}

// Check is a check's result.
type Check struct {
	ID      string       `json:"id"`
	Change  Change       `json:"change"`
	Before  CheckNumbers `json:"before"`
	After   CheckNumbers `json:"after"`
	Verdict string       `json:"verdict"`
	Text    string       `json:"text"`
	Made    time.Time    `json:"made"`
}

// CheckDue says c's check is due at now.
func CheckDue(c Change, now time.Time) bool { return !now.Before(c.At.Add(CheckAfter)) }

// RunCheck counts c's before and after from the traces, steward questions and feedback,
// which must cover [c.At-CheckAfter, c.At+CheckAfter).
func RunCheck(c Change, qs []StewardQuestion, traces []Trace, fb []Feedback, now time.Time) Check {
	before := checkNumbers(c, qs, traces, fb, c.At.Add(-CheckAfter), c.At)
	after := checkNumbers(c, qs, traces, fb, c.At, c.At.Add(CheckAfter))
	out := Check{ID: c.Key(), Change: c, Before: before, After: after, Made: now.UTC()}
	out.Verdict = verdict(before, after)
	out.Text = checkText(out)
	return out
}

func (c Change) counts(space, agent string) bool {
	if c.Agent != "" && agent != c.Agent {
		return false
	}
	return c.Space == "" || spaceName(space) == c.Space
}

func checkNumbers(c Change, qs []StewardQuestion, traces []Trace, fb []Feedback, from, to time.Time) CheckNumbers {
	in := func(t time.Time) bool { return !t.Before(from) && t.Before(to) }
	var n CheckNumbers
	ids := map[string]bool{}
	for _, t := range traces {
		a := traceAgent(t)
		if a == "" || t.Feature == FeatureEval || !in(t.At) || !c.counts(t.Space, a) {
			continue
		}
		n.Answers++
		ids[t.ID] = true
		if nothing(t) {
			n.Gaps++
		}
	}
	up := 0
	for _, f := range fb {
		if ids[f.Trace] {
			n.Rated++
			if f.Rating == "up" {
				up++
			}
		}
	}
	discarded := 0
	var hours []float64
	for _, q := range qs {
		if !in(q.At) {
			continue
		}
		if a := draftAgent(q); a != "" {
			if c.counts(q.Space, a) {
				n.Drafts++
				if q.Thread.Discarded {
					discarded++
				}
			}
			continue
		}
		if c.Agent == "" && c.counts(q.Space, "") && q.Reply != nil {
			n.Replies++
			hours = append(hours, q.Reply.At.Sub(q.At).Hours())
		}
	}
	n.GapRate = rate(n.Gaps, n.Answers)
	n.HelpfulRate = rate(up, n.Rated)
	n.DiscardRate = rate(discarded, n.Drafts)
	n.ReplyHours = median(hours)
	return n
}

func rate(a, b int) *float64 {
	if b == 0 {
		return nil
	}
	x := float64(a) / float64(b)
	return &x
}

// verdict is better when something moved clearly the right way and nothing the wrong way,
// worse the other way round, and no clear change otherwise.
func verdict(b, a CheckNumbers) string {
	if b.Answers < CheckMinN || a.Answers < CheckMinN {
		return VerdictTooFew
	}
	good, bad := 0, 0
	move := func(x, y *float64, by float64, higherBetter, relative bool) {
		if x == nil || y == nil {
			return
		}
		d := *y - *x
		if relative {
			if *x == 0 {
				return
			}
			d /= *x
		}
		if !higherBetter {
			d = -d
		}
		switch {
		case d >= by:
			good++
		case d <= -by:
			bad++
		}
	}
	move(b.HelpfulRate, a.HelpfulRate, CheckRateBy, true, false)
	move(b.GapRate, a.GapRate, CheckRateBy, false, false)
	move(b.DiscardRate, a.DiscardRate, CheckRateBy, false, false)
	move(b.ReplyHours, a.ReplyHours, CheckHoursBy, false, true)
	switch {
	case good > 0 && bad == 0:
		return VerdictBetter
	case bad > 0 && good == 0:
		return VerdictWorse
	}
	return VerdictNoClear
}

func pct(x *float64) string {
	if x == nil {
		return "none"
	}
	return fmt.Sprintf("%.0f%%", 100**x)
}

// checkText is the draft: what changed, the verdict and the counts. Never a question,
// an answer or who asked.
func checkText(c Check) string {
	what := c.Change.Kind + " " + c.Change.Ref
	where := ""
	switch {
	case c.Change.Agent != "":
		where = " for " + c.Change.Agent
	case c.Change.Space != "":
		where = " in " + c.Change.Space
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Did it help? %s%s, shipped %s: %s.", what, where, c.Change.At.UTC().Format(time.DateOnly), c.Verdict)
	fmt.Fprintf(&b, " 4 weeks before and after: %d and %d answers, helpful %s and %s, found nothing %s and %s, drafts discarded %s and %s.",
		c.Before.Answers, c.After.Answers, pct(c.Before.HelpfulRate), pct(c.After.HelpfulRate), pct(c.Before.GapRate), pct(c.After.GapRate),
		pct(c.Before.DiscardRate), pct(c.After.DiscardRate))
	if c.Verdict == VerdictTooFew {
		fmt.Fprintf(&b, " Fewer than %d answers on a side, so nothing can be said yet.", CheckMinN)
	}
	b.WriteString(" Nothing was changed or undone.")
	return b.String()
}

// ProposalChange is the kind of change a steward proposal's diff lines make: skills,
// agents, routing, or anything else of the space.
func ProposalChange(diff []string) string {
	kind := ChangeSpace
	for _, l := range diff {
		switch {
		case strings.Contains(l, "skills"):
			return ChangeSkill
		case strings.Contains(l, "routing"):
			kind = ChangeRouting
		case strings.Contains(l, "agents") && kind == ChangeSpace:
			kind = ChangeAgents
		}
	}
	return kind
}

// ReleaseChanges are the agents' new releases: the first trace of each agent under a
// release after one under another, in [from, to).
func ReleaseChanges(traces []Trace, from, to time.Time) []Change {
	ts := append([]Trace{}, traces...)
	sort.SliceStable(ts, func(i, j int) bool { return ts[i].At.Before(ts[j].At) })
	last := map[string]string{}
	var out []Change
	for _, t := range ts {
		a := traceAgent(t)
		if a == "" || t.Response.Release == "" || t.Feature == FeatureEval {
			continue
		}
		prev, seen := last[a]
		last[a] = t.Response.Release
		if seen && prev != t.Response.Release && !t.At.Before(from) && t.At.Before(to) {
			out = append(out, Change{Kind: ChangeRelease, Ref: t.Response.Release, Agent: a, At: t.At, Source: "/v1/admin/agents/" + a + "/record"})
		}
	}
	return out
}

// AdminLine is one line of the admin's week (SB-5006): counts and ids, and the link to its
// source record.
type AdminLine struct {
	Section string `json:"section"`
	Text    string `json:"text"`
	Link    string `json:"link"`
}

// AdminWeekText is the digest's text, section by section, each line with its link.
func AdminWeekText(week string, lines []AdminLine) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Your admin's week from %s", week)
	if len(lines) == 0 {
		b.WriteString(": nothing changed, and nothing needs you.")
		return b.String()
	}
	b.WriteString(":")
	section := ""
	for _, l := range lines {
		if l.Section != section {
			section = l.Section
			fmt.Fprintf(&b, "\n%s:", section)
		}
		fmt.Fprintf(&b, "\n- %s (%s)", l.Text, l.Link)
	}
	return b.String()
}
