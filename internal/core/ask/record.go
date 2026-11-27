package ask

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/canonjson"
	"example.com/second-brain/internal/core/reviews"
)

// An agent's record (sb-agents(7), An agent's record; SB-4707): one signed file an auditor
// can be handed: the definition and owner, its versions, eval, value by quarter, pauses,
// spend against its cap, access reviews and the reports on its answers. Counts, dates and
// ids only: no document text, question or asker.

// RecordKind is what the record's signature says it signs.
const RecordKind = "agent-record"

// RecordVersion is one release the agent's kept traces answered under.
type RecordVersion struct {
	Release string `json:"release"`
	First   string `json:"first"`
	Last    string `json:"last"`
	Runs    int    `json:"runs"`
}

// RecordQuarter is Value for one quarter, every space together.
type RecordQuarter struct {
	Quarter         string  `json:"quarter"`
	Drafts          int     `json:"drafts"`
	Published       int     `json:"published"`
	PublishedEdited int     `json:"published_edited"`
	Discarded       int     `json:"discarded"`
	Rounds          int     `json:"rounds"`
	Rated           int     `json:"rated"`
	Helpful         int     `json:"helpful"`
	CostUSD         float64 `json:"cost_usd"`
}

// RecordPause is one kill switch row on the agent, tenant-wide or in a space.
type RecordPause struct {
	Seq    int64  `json:"seq"`
	At     string `json:"at"`
	Scope  string `json:"scope"`
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

// RecordReview is the agent's item in one agent review.
type RecordReview struct {
	ID       string `json:"id"`
	Space    string `json:"space"`
	Quarter  string `json:"quarter"`
	Item     string `json:"item"`
	Unused   bool   `json:"unused"`
	Decision string `json:"decision,omitempty"`
	By       string `json:"by,omitempty"`
	AuditSeq int64  `json:"audit_seq,omitempty"`
}

// RecordReport is one report of an answer it wrote: ids and dates only.
type RecordReport struct {
	ID    string `json:"id"`
	At    string `json:"at"`
	Space string `json:"space"`
}

// RecordEval is its suite and whether the release's eval run holds it now.
type RecordEval struct {
	Suite string `json:"suite"`
	Held  bool   `json:"held"`
}

// AgentRecord is the record.
type AgentRecord struct {
	Agent         string          `json:"agent"`
	Tenant        string          `json:"tenant"`
	Made          string          `json:"made"`
	Release       string          `json:"release"`
	Definition    AgentDef        `json:"definition"`
	Owner         string          `json:"owner"`
	OwnerInactive bool            `json:"owner_inactive,omitempty"`
	PromptSHA256  string          `json:"prompt_sha256"`
	Eval          RecordEval      `json:"eval"`
	Versions      []RecordVersion `json:"versions"`
	Value         []RecordQuarter `json:"value"`
	Pauses        []RecordPause   `json:"pauses"`
	Spend         []AgentSpend    `json:"spend"`
	Reviews       []RecordReview  `json:"reviews"`
	Reports       []RecordReport  `json:"reports"`
	// Job and Targets are the agent's job and every target with its last ValueWeeks weeks
	// (SB-5004); none without a goal.
	Job     string       `json:"job,omitempty"`
	Targets []TargetView `json:"targets"`
	// Reach is what the agent can read now and last week's count (SB-5406).
	Reach *AgentReach `json:"reach,omitempty"`
}

// SignedRecord is the record with its SHA-256 and the cell's signature.
type SignedRecord struct {
	Record   AgentRecord `json:"record"`
	SHA256   string      `json:"sha256"`
	SignedAt string      `json:"signed_at"`
	Sig      string      `json:"sig"`
	Key      string      `json:"key"`
}

// RecordInputs are what a record is counted from: everything the cell keeps of the agent.
type RecordInputs struct {
	Tenant, Release string
	Now             time.Time
	Def             AgentDef
	OwnerInactive   bool
	PromptSHA256    string
	EvalHeld        bool
	Traces          []Trace
	Questions       []StewardQuestion
	Feedback        []Feedback
	Audit           []audit.Row
	Reviews         []reviews.Review
	Caps            map[string]float64
	// Goal is the agent's job and targets from the manifest, nil for none (SB-5004).
	Goal *GoalDef
	// Reach is the agent's reach now (SB-5406); nil leaves it out.
	Reach *AgentReach
}

func quarterStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), time.Month((int(t.Month())-1)/3*3+1), 1, 0, 0, 0, 0, time.UTC)
}

// BuildRecord counts the record of in.Def from in.
func BuildRecord(in RecordInputs) AgentRecord {
	name := in.Def.Name
	r := AgentRecord{Agent: name, Tenant: in.Tenant, Made: in.Now.UTC().Format(time.RFC3339), Release: in.Release, Definition: in.Def,
		Owner: in.Def.Owner, OwnerInactive: in.OwnerInactive, PromptSHA256: in.PromptSHA256, Eval: RecordEval{Suite: in.Def.Suite, Held: in.EvalHeld},
		Versions: []RecordVersion{}, Value: []RecordQuarter{}, Pauses: []RecordPause{}, Spend: []AgentSpend{}, Reviews: []RecordReview{}, Reports: []RecordReport{},
		Targets: []TargetView{}, Reach: in.Reach}
	if in.Goal != nil { // SB-5004: every target, with its history
		vs := GoalViews(map[string]GoalDef{name: *in.Goal}, in.Questions, in.Traces, in.Feedback, in.Now, ValueWeeks, nil)
		r.Job, r.Targets = in.Goal.Job, vs[0].Targets
	}
	r.Versions = recordVersions(in.Traces, name)
	r.Value = recordValue(in, name)
	r.Spend = recordSpend(in, name)
	for _, row := range in.Audit {
		if row.Kind == "kill" && (row.Subject == "agent:"+name || strings.HasPrefix(row.Subject, "agent:"+name+"@")) {
			r.Pauses = append(r.Pauses, RecordPause{Seq: row.Seq, At: row.At, Scope: row.Subject, State: row.State, Reason: row.Reason})
		}
	}
	r.Reviews = recordReviews(in.Reviews, name)
	agentOf := map[string]string{}
	for _, t := range in.Traces {
		agentOf[t.ID] = traceAgent(t)
	}
	for _, q := range in.Questions {
		if q.Report != nil && agentOf[q.Trace] == name {
			r.Reports = append(r.Reports, RecordReport{ID: q.ID, At: q.At.UTC().Format(time.RFC3339), Space: strings.TrimPrefix(q.Space, "spc_")})
		}
	}
	sort.Slice(r.Reports, func(i, j int) bool { return r.Reports[i].At+r.Reports[i].ID < r.Reports[j].At+r.Reports[j].ID })
	return r
}

func recordVersions(traces []Trace, name string) []RecordVersion {
	by := map[string]*RecordVersion{}
	for _, t := range traces {
		if traceAgent(t) != name {
			continue
		}
		rel := t.Response.Release
		at := t.At.UTC().Format(time.RFC3339)
		v, ok := by[rel]
		if !ok {
			v = &RecordVersion{Release: rel, First: at, Last: at}
			by[rel] = v
		}
		v.Runs++
		if at < v.First {
			v.First = at
		}
		if at > v.Last {
			v.Last = at
		}
	}
	out := []RecordVersion{}
	for _, v := range by {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].First+out[i].Release < out[j].First+out[j].Release })
	return out
}

func recordValue(in RecordInputs, name string) []RecordQuarter {
	starts := map[time.Time]bool{}
	for _, q := range in.Questions {
		if draftAgent(q) == name {
			starts[quarterStart(q.At)] = true
		}
	}
	for _, t := range in.Traces {
		if traceAgent(t) == name {
			starts[quarterStart(t.At)] = true
		}
	}
	var qs []time.Time
	for s := range starts {
		qs = append(qs, s)
	}
	sort.Slice(qs, func(i, j int) bool { return qs[i].Before(qs[j]) })
	out := []RecordQuarter{}
	for _, from := range qs {
		to := from.AddDate(0, 3, 0)
		var sq []StewardQuestion
		for _, q := range in.Questions {
			if !q.At.Before(from) && q.At.Before(to) {
				sq = append(sq, q)
			}
		}
		var st []Trace
		for _, t := range in.Traces {
			if !t.At.Before(from) && t.At.Before(to) {
				st = append(st, t)
			}
		}
		var sf []Feedback
		for _, f := range in.Feedback {
			if !f.At.Before(from) && f.At.Before(to) {
				sf = append(sf, f)
			}
		}
		rq := RecordQuarter{Quarter: reviews.Quarter(from)}
		for _, row := range AgentValue(sq, st, sf, from, []string{name}) {
			rq.Drafts += row.Drafts
			rq.Published += row.Published
			rq.PublishedEdited += row.PublishedEdited
			rq.Discarded += row.Discarded
			rq.Rounds += row.Rounds
			rq.Rated += row.Rated
			if row.HelpfulRate != nil {
				rq.Helpful += int(*row.HelpfulRate*float64(row.Rated) + 0.5)
			}
			rq.CostUSD += row.CostUSD
		}
		out = append(out, rq)
	}
	return out
}

func recordSpend(in RecordInputs, name string) []AgentSpend {
	months := map[string][]Trace{}
	for _, t := range in.Traces {
		if traceAgent(t) == name {
			m := t.At.UTC().Format("2006-01")
			months[m] = append(months[m], t)
		}
	}
	var keys []string
	for m := range months {
		keys = append(keys, m)
	}
	sort.Strings(keys)
	out := []AgentSpend{}
	for _, m := range keys {
		out = append(out, MonthSpend(months[m], m, in.Caps, []string{name})...)
	}
	return out
}

// reviewItems are the items of an agent review that stand for agent name.
func reviewItems(name string) []string {
	items := []string{name, "brief:" + name}
	switch name {
	case "update":
		items = append(items, reviews.ItemUpdates)
	case "leader-week":
		items = append(items, reviews.ItemLeaderWeek)
	}
	return items
}

func recordReviews(rs []reviews.Review, name string) []RecordReview {
	out := []RecordReview{}
	items := reviewItems(name)
	for _, rv := range rs {
		if rv.Kind != reviews.KindAgents {
			continue
		}
		for _, it := range rv.Items {
			if contains(items, it.User) {
				out = append(out, RecordReview{ID: rv.ID, Space: rv.Space, Quarter: rv.Quarter, Item: it.User, Unused: it.Unused,
					Decision: it.Decision, By: it.By, AuditSeq: it.AuditSeq})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Quarter+out[i].Space+out[i].Item < out[j].Quarter+out[j].Space+out[j].Item
	})
	return out
}

// recordSHA is the SHA-256 of the record's canonical JSON.
func recordSHA(raw []byte) (string, error) {
	c, err := canonjson.EncodeDocument(raw)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(c)
	return hex.EncodeToString(h[:]), nil
}

// SignRecord signs r with the cell's anchor key.
func SignRecord(r AgentRecord, sg audit.Signer, now time.Time) (SignedRecord, error) {
	raw, err := json.Marshal(r)
	if err != nil {
		return SignedRecord{}, err
	}
	sha, err := recordSHA(raw)
	if err != nil {
		return SignedRecord{}, err
	}
	at, sig, key := sg.SignDocument(RecordKind, sha, now)
	return SignedRecord{Record: r, SHA256: sha, SignedAt: at, Sig: sig, Key: key}, nil
}

// ErrRecord is a record that doesn't verify.
var ErrRecord = errors.New("ask: the record doesn't verify")

// VerifyRecord checks a signed record's hash and signature, and its key against pinned when
// pinned is set.
func VerifyRecord(doc []byte, pinned string) (SignedRecord, error) {
	var raw struct {
		Record   json.RawMessage `json:"record"`
		SHA256   string          `json:"sha256"`
		SignedAt string          `json:"signed_at"`
		Sig      string          `json:"sig"`
		Key      string          `json:"key"`
	}
	if err := json.Unmarshal(doc, &raw); err != nil {
		return SignedRecord{}, fmt.Errorf("%w: it isn't JSON: %v", ErrRecord, err)
	}
	sha, err := recordSHA(raw.Record)
	if err != nil || sha != raw.SHA256 {
		return SignedRecord{}, fmt.Errorf("%w: the SHA-256 doesn't match the record", ErrRecord)
	}
	var sr SignedRecord
	if err := json.Unmarshal(doc, &sr); err != nil {
		return SignedRecord{}, fmt.Errorf("%w: %v", ErrRecord, err)
	}
	if pinned != "" && pinned != raw.Key {
		return SignedRecord{}, fmt.Errorf("%w: it was signed by %s, not the pinned key", ErrRecord, raw.Key)
	}
	if !audit.VerifyDocument(RecordKind, sr.Record.Tenant, raw.Key, raw.SHA256, raw.SignedAt, raw.Sig) {
		return SignedRecord{}, fmt.Errorf("%w: the signature doesn't verify", ErrRecord)
	}
	return sr, nil
}

// RecordMarkdown is the record as Markdown, for a reader.
func RecordMarkdown(sr SignedRecord) string {
	r := sr.Record
	var b strings.Builder
	fmt.Fprintf(&b, "# Agent record: %s\n\n", r.Agent)
	fmt.Fprintf(&b, "Tenant %s, made %s under release %s.\n\n", r.Tenant, r.Made, r.Release)
	owner := r.Owner
	if r.OwnerInactive {
		owner += " (no active person answers for it now)"
	}
	fmt.Fprintf(&b, "## Definition\n\n- Owner: %s\n- Department: %s\n- Prompt: %s (SHA-256 %s)\n- Eval suite: %s, %s\n- Tools: %s\n- Writes: %v, approvals: %v\n- Labels: %s\n",
		owner, r.Definition.Department, r.Definition.Prompt, r.PromptSHA256, r.Eval.Suite, map[bool]string{true: "held now", false: "passed"}[r.Eval.Held],
		listOr(r.Definition.Tools, "none"), r.Definition.Writes, r.Definition.Approvals, listOr(r.Definition.Labels, "none"))
	if r.Definition.Shadow != nil {
		fmt.Fprintf(&b, "- Shadow: %s for %d days\n", r.Definition.Shadow.Prompt, r.Definition.Shadow.Days)
	}
	b.WriteString("\n## Versions\n\n")
	if len(r.Versions) == 0 {
		b.WriteString("No kept traces.\n")
	}
	for _, v := range r.Versions {
		fmt.Fprintf(&b, "- %s: %d runs, %s to %s\n", v.Release, v.Runs, v.First, v.Last)
	}
	b.WriteString("\n## Value by quarter\n\n| Quarter | Drafts | Published | Edited | Discarded | Rounds | Rated | Helpful | Cost |\n|---|---|---|---|---|---|---|---|---|\n")
	for _, q := range r.Value {
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %d | %d | %d | $%.2f |\n", q.Quarter, q.Drafts, q.Published, q.PublishedEdited, q.Discarded, q.Rounds, q.Rated, q.Helpful, q.CostUSD)
	}
	b.WriteString(targetsMarkdown(r))
	b.WriteString("\n## Pauses and kill switches\n\n")
	if len(r.Pauses) == 0 {
		b.WriteString("None.\n")
	}
	for _, p := range r.Pauses {
		fmt.Fprintf(&b, "- %s %s %s (audit seq %d): %s\n", p.At, p.Scope, p.State, p.Seq, p.Reason)
	}
	b.WriteString("\n## Spend\n\n")
	if len(r.Spend) == 0 {
		b.WriteString("None.\n")
	}
	for _, s := range r.Spend {
		cp := "no cap"
		if s.MonthlyUSD != nil {
			cp = fmt.Sprintf("cap $%.2f", *s.MonthlyUSD)
		}
		fmt.Fprintf(&b, "- %s: $%.2f, %s\n", s.Month, s.SpentUSD, cp)
	}
	b.WriteString("\n## Access reviews\n\n")
	if len(r.Reviews) == 0 {
		b.WriteString("None.\n")
	}
	for _, v := range r.Reviews {
		d := "undecided"
		if v.Decision != "" {
			d = v.Decision + " by " + v.By
		}
		fmt.Fprintf(&b, "- %s %s, %s (%s): unused %v, %s\n", v.Quarter, v.Space, v.Item, v.ID, v.Unused, d)
	}
	fmt.Fprintf(&b, "\n## Reports on its answers\n\n%d reported.\n", len(r.Reports))
	for _, p := range r.Reports {
		fmt.Fprintf(&b, "- %s %s (%s)\n", p.At, p.Space, p.ID)
	}
	fmt.Fprintf(&b, "\n## Signature\n\nSHA-256 %s, signed %s with %s.\n", sr.SHA256, sr.SignedAt, sr.Key)
	return b.String()
}

func listOr(xs []string, none string) string {
	if len(xs) == 0 {
		return none
	}
	return strings.Join(xs, ", ")
}

// targetsMarkdown is the record's job and targets section, "" without a goal (SB-5004).
func targetsMarkdown(r AgentRecord) string {
	if r.Job == "" && len(r.Targets) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n## Job and targets\n\n")
	if r.Job != "" {
		fmt.Fprintf(&b, "Job: %s\n\n", r.Job)
	}
	for _, t := range r.Targets {
		aim := ""
		if t.AtLeast != nil {
			aim = fmt.Sprintf("at least %g", *t.AtLeast)
		} else if t.AtMost != nil {
			aim = fmt.Sprintf("at most %g", *t.AtMost)
		}
		fmt.Fprintf(&b, "- %s %s:", t.Metric, aim)
		for _, w := range t.Weeks {
			switch {
			case w.Value == nil:
				fmt.Fprintf(&b, " %s none;", w.Week)
			case w.Met != nil && *w.Met:
				fmt.Fprintf(&b, " %s %.2f met;", w.Week, *w.Value)
			default:
				fmt.Fprintf(&b, " %s %.2f missed;", w.Week, *w.Value)
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}
