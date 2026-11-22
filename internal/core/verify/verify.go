// Package verify is `sb tenant verify` (sb-tenant(1), How verify decides; SB-601): law
// checks against a cell's real data and users that print counts, never content, and treat
// unknown as failure.
package verify

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
)

// Cite is one citation in an answer.
type Cite struct {
	Doc       string // kind:id
	Version   int
	Connector string
}

// Answer is what verify needs from one ask: citations, claim labels, and the documents the
// answer's decision list says were checked live.
type Answer struct {
	Cites       []Cite
	Cited       int // claims with at least one citation
	Uncited     int // claims labelled uncited
	Unlabelled  int // claims with neither (must be 0)
	LiveChecked map[string]bool
}

// Target is a cell under verification.
type Target interface {
	Users(ctx context.Context) ([]string, error)
	Questions(ctx context.Context) ([]string, error)
	Ask(ctx context.Context, user, question string) (Answer, error)
	// Allowed asks the source, live, whether user may open doc now.
	Allowed(ctx context.Context, user, doc string) (bool, error)
	// ForeignIDs are other tenants' document ids, from the registry (ids only).
	ForeignIDs(ctx context.Context) ([]string, error)
	// Timeout makes one connector time out until restore is called.
	Timeout(connector string) (restore func(), err error)
	Connectors() []string
}

// StartTarget is a cell with spaces' start lists (SB-1706): Unreachable names, by space and
// position only, every item no member of its space can open now.
type StartTarget interface {
	Unreachable(ctx context.Context) (items int, unreachable []string, err error)
}

// AccessTarget is a cell that can run the LAW-17 access-path check (sb-factory(7)).
type AccessTarget interface {
	UnlistedAccess(ctx context.Context) (unlisted, paths int, err error)
}

// Result is one law's line.
type Result struct {
	Law    string
	Status string // pass, fail, unknown
	Detail string
}

// Report is the whole run.
type Report struct {
	Seed    int64
	Results []Result
}

// Passed is true only if every law passed.
func (r Report) Passed() bool {
	for _, x := range r.Results {
		if x.Status != "pass" {
			return false
		}
	}
	return len(r.Results) > 0
}

// String is the human report: one line per law, counts only.
func (r Report) String() string {
	var b strings.Builder
	passed, failed := 0, 0
	for _, x := range r.Results {
		fmt.Fprintf(&b, "%-7s %-7s %s\n", x.Law, x.Status, x.Detail)
		if x.Status == "pass" {
			passed++
		} else {
			failed++
		}
	}
	fmt.Fprintf(&b, "verify: %d passed, %d failed (seed %d)\n", passed, failed, r.Seed)
	return b.String()
}

// Options choose laws and sample size.
type Options struct {
	Laws   []string // empty: LAW-01, LAW-02, LAW-03, LAW-14
	Sample int      // default 50
	Seed   int64
}

type sample struct {
	user, question string
	a              Answer
}

// verifier is one run's state: the target, the sampled answers and the seeded source.
type verifier struct {
	ctx     context.Context
	t       Target
	o       Options
	r       *rand.Rand
	users   []string
	samples []sample
	askErr  int
}

func (v *verifier) unknown(law, why string) Result { return Result{law, "unknown", why} }

// Run verifies a target.
func Run(ctx context.Context, t Target, o Options) Report {
	if o.Sample <= 0 {
		o.Sample = 50
	}
	laws := o.Laws
	if len(laws) == 0 {
		laws = []string{"LAW-01", "LAW-02", "LAW-03", "LAW-14"}
		if _, ok := t.(StartTarget); ok {
			laws = append(laws, "START")
		}
	}
	v := &verifier{ctx: ctx, t: t, o: o, r: rand.New(rand.NewSource(o.Seed))}
	users, uerr := t.Users(ctx)
	qs, qerr := t.Questions(ctx)
	v.users = users
	if uerr == nil && qerr == nil && len(users) > 0 && len(qs) > 0 {
		for i := 0; i < o.Sample; i++ {
			u, q := users[v.r.Intn(len(users))], qs[v.r.Intn(len(qs))]
			a, err := t.Ask(ctx, u, q)
			if err != nil {
				v.askErr++
				continue
			}
			v.samples = append(v.samples, sample{u, q, a})
		}
	}
	checks := map[string]func(string) Result{"LAW-01": v.law01, "LAW-02": v.law02, "LAW-03": v.law03, "LAW-14": v.law14, "LAW-17": v.law17, "START": v.start}
	rep := Report{Seed: o.Seed}
	for _, law := range laws {
		if check, ok := checks[law]; ok {
			rep.Results = append(rep.Results, check(law))
		} else {
			rep.Results = append(rep.Results, v.unknown(law, "no such check"))
		}
	}
	return rep
}

func (v *verifier) law01(law string) Result {
	if len(v.samples) == 0 || v.askErr > 0 {
		return v.unknown(law, fmt.Sprintf("%d answers, %d failed to answer", len(v.samples), v.askErr))
	}
	cites, bad := 0, 0
	for _, s := range v.samples {
		for _, c := range s.a.Cites {
			cites++
			ok, err := v.t.Allowed(v.ctx, s.user, c.Doc)
			if err != nil || !ok || !s.a.LiveChecked[c.Doc] {
				bad++
			}
		}
	}
	if bad > 0 {
		return Result{law, "fail", fmt.Sprintf("%d of %d citations not openable now or not checked live", bad, cites)}
	}
	return Result{law, "pass", fmt.Sprintf("%d/%d sampled answers, %d citations, source asked for all", len(v.samples), v.o.Sample, cites)}
}

func (v *verifier) law02(law string) Result {
	ids, err := v.t.ForeignIDs(v.ctx)
	if err != nil || len(v.users) == 0 {
		return v.unknown(law, "no foreign ids or no users")
	}
	queries, hits := 0, 0
	for _, id := range ids {
		for _, q := range []string{id, "show me " + id} {
			queries++
			a, err := v.t.Ask(v.ctx, v.users[v.r.Intn(len(v.users))], q)
			if err != nil {
				continue // a refusal is fine
			}
			for _, c := range a.Cites {
				if c.Doc == id {
					hits++
				}
			}
		}
	}
	if hits > 0 {
		return Result{law, "fail", fmt.Sprintf("%d adversarial queries, %d cross-tenant results", queries, hits)}
	}
	return Result{law, "pass", fmt.Sprintf("%d adversarial queries, 0 cross-tenant results", queries)}
}

func (v *verifier) law03(law string) Result {
	cs := v.t.Connectors()
	if len(cs) == 0 || len(v.samples) == 0 {
		return v.unknown(law, "no connectors or no answers")
	}
	conn := cs[v.r.Intn(len(cs))]
	restore, err := v.t.Timeout(conn)
	if err != nil {
		return v.unknown(law, "can't inject a timeout")
	}
	defer restore()
	used := 0
	for i := 0; i < len(v.samples) && i < 10; i++ {
		a, err := v.t.Ask(v.ctx, v.samples[i].user, v.samples[i].question)
		if err != nil {
			continue
		}
		for _, c := range a.Cites {
			if c.Connector == conn {
				used++
			}
		}
	}
	if used > 0 {
		return Result{law, "fail", fmt.Sprintf("%s timeout injected, %d %s chunks used", conn, used, conn)}
	}
	return Result{law, "pass", fmt.Sprintf("%s timeout injected, 0 %s chunks used", conn, conn)}
}

func (v *verifier) law14(law string) Result {
	if len(v.samples) == 0 {
		return v.unknown(law, "no answers")
	}
	cited, unc, unl := 0, 0, 0
	for _, s := range v.samples {
		cited, unc, unl = cited+s.a.Cited, unc+s.a.Uncited, unl+s.a.Unlabelled
	}
	d := fmt.Sprintf("%d claims: %d cited, %d labelled uncited, %d unlabelled", cited+unc+unl, cited, unc, unl)
	if unl > 0 {
		return Result{law, "fail", d}
	}
	return Result{law, "pass", d}
}

// start checks every space's start list: an item no member can open fails (SB-1706).
func (v *verifier) start(law string) Result {
	st, ok := v.t.(StartTarget)
	if !ok {
		return v.unknown(law, "this cell has no start lists")
	}
	items, bad, err := st.Unreachable(v.ctx)
	if err != nil {
		return v.unknown(law, "start lists couldn't be checked: "+err.Error())
	}
	if len(bad) > 0 {
		return Result{law, "fail", fmt.Sprintf("%d of %d start-here items no member can open: %s", len(bad), items, strings.Join(bad, ", "))}
	}
	return Result{law, "pass", fmt.Sprintf("%d start-here items, each open to a member", items)}
}

func (v *verifier) law17(law string) Result {
	at, ok := v.t.(AccessTarget)
	if !ok {
		return v.unknown(law, "this cell can't enumerate access paths")
	}
	unlisted, paths, err := at.UnlistedAccess(v.ctx)
	if err != nil {
		return v.unknown(law, "access paths couldn't be enumerated: "+err.Error())
	}
	if unlisted > 0 {
		return Result{law, "fail", fmt.Sprintf("%d of %d factory access paths aren't in the hand-off record", unlisted, paths)}
	}
	return Result{law, "pass", fmt.Sprintf("%d factory access paths, all retained grants", paths)}
}
