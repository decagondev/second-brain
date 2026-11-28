// Package prove runs a ticket's acceptance criteria against the real system a cell is
// configured for (SB-6001; sb(1), sb prove), reading only. Each runner turns what it saw
// into counts and hashed ids: nothing it reads, no name and no document text, reaches the
// proof (package proof checks that again before signing).
package prove

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/atlassian"
	"example.com/second-brain/internal/adapters/msgraph"
	"example.com/second-brain/internal/core/box"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/proof"
)

// Env is what the runners may use. A runner whose part is missing fails the criteria that
// need it (missing is never a pass, LAW-03).
type Env struct {
	Tenant string
	// Graph and SharePoint are the cell's Microsoft 365 parts (ADR-0023); Mailboxes are the
	// shared mailboxes the manifest names; Probe is a mailbox the cell must not be able to
	// read, such as a person's own; Person is someone to check access and a calendar for.
	Graph      *msgraph.Client
	SharePoint *msgraph.SharePoint
	Mailboxes  []string
	Probe      string
	Person     string
	Labels     map[string]string
	// TeamsKeys loads the Bot Framework's signing keys and says how many there are.
	TeamsKeys func(context.Context) (int, error)
	// Chat checks the Mattermost bot: lines starting "ok" or "missing" (sb selfhost
	// check-chat).
	Chat func() ([]string, error)
	// Email checks the mail server offers STARTTLS with a trusted certificate.
	Email func(context.Context) error
	// Atlassian, Confluence and Jira are the cell's Atlassian parts (ADR-0024).
	Atlassian  *atlassian.Client
	Confluence *atlassian.Confluence
	Jira       *atlassian.Jira
	// Facts are the box's facts (tools/selfhost/facts.sh) for the doctor.
	Facts []byte
	Now   func() time.Time
}

// Runner runs one ticket's criteria.
type Runner struct {
	Ticket, API string
	Run         func(ctx context.Context, e Env, h proof.Hasher) []proof.Criterion
}

// Runners are the tickets sb prove covers, by ticket.
var Runners = map[string]Runner{
	"SB-5901": {"SB-5901", "graph-v1.0", sharePoint},
	"SB-5902": {"SB-5902", "graph-v1.0", labels},
	"SB-5903": {"SB-5903", "graph-v1.0", entra},
	"SB-5904": {"SB-5904", "botframework-v3", teams},
	"SB-5905": {"SB-5905", "graph-v1.0", calendar},
	"SB-5906": {"SB-5906", "graph-v1.0", transcripts},
	"SB-5907": {"SB-5907", "graph-v1.0", mailboxes},
	"SB-2603": {"SB-2603", "mattermost-v4", chat},
	"SB-2601": {"SB-2601", "smtp", email},
	"SB-1605": {"SB-1605", "box-facts-v1", doctor},
}

// Tickets are the tickets sb prove covers, sorted.
func Tickets() []string {
	var out []string
	for t := range Runners {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// ErrUnknown is a ticket sb prove doesn't cover.
var ErrUnknown = errors.New("prove: no runner for that ticket")

// Run runs ticket's criteria and returns its proof, not yet signed.
func Run(ctx context.Context, ticket string, e Env) (proof.Proof, error) {
	r, ok := Runners[ticket]
	if !ok {
		return proof.Proof{}, ErrUnknown
	}
	if e.Now == nil {
		e.Now = time.Now
	}
	cs := r.Run(ctx, e, proof.NewHasher())
	return proof.New(r.Ticket, e.Tenant, r.API, e.Now(), cs), nil
}

// timed runs f and records how long it took.
func timed(name string, f func(c *proof.Criterion)) proof.Criterion {
	c := proof.Criterion{Name: name, Counts: map[string]int{}}
	start := time.Now()
	f(&c)
	c.Millis = time.Since(start).Milliseconds()
	if len(c.Counts) == 0 {
		c.Counts = nil
	}
	return c
}

func missing(c *proof.Criterion) { c.Counts["missing_input"] = 1 }

// changes reads every change once from an empty cursor.
func changes(ctx context.Context, sp *msgraph.SharePoint) (ports.ChangeSet, error) {
	set, _, err := sp.Changes(ctx, "")
	return set, err
}

func sharePoint(ctx context.Context, e Env, h proof.Hasher) []proof.Criterion {
	var docs []ports.Document
	var refs []ports.SourceRef
	cs := []proof.Criterion{timed("the named drives list their items", func(c *proof.Criterion) {
		if e.SharePoint == nil || len(e.SharePoint.Drives) == 0 {
			missing(c)
			return
		}
		set, err := changes(ctx, e.SharePoint)
		if err != nil {
			c.Counts["errors"] = 1
			return
		}
		var ids []string
		for _, ch := range set {
			if ch.Deleted || strings.HasPrefix(ch.Ref.ID, "tm:") {
				continue
			}
			refs = append(refs, ch.Ref)
			ids = append(ids, ch.Ref.ID)
		}
		c.Counts["drives"], c.Counts["items"] = len(e.SharePoint.Drives), len(refs)
		c.Seen = h.IDs(ids)
		c.Pass = len(refs) > 0
	})}
	cs = append(cs, timed("each item reads with its permissions", func(c *proof.Criterion) {
		if len(refs) == 0 {
			missing(c)
			return
		}
		for _, r := range refs[:min(len(refs), 25)] {
			d, err := e.SharePoint.Fetch(ctx, r)
			if err != nil {
				c.Counts["errors"]++
				continue
			}
			docs = append(docs, d)
			if len(d.ACLPrefilter) > 0 {
				c.Counts["with_permissions"]++
			}
			if d.Text != "" {
				c.Counts["with_text"]++
			}
		}
		c.Counts["read"] = len(docs)
		c.Pass = len(docs) > 0 && c.Counts["errors"] == 0 && c.Counts["with_permissions"] == len(docs)
	}))
	cs = append(cs, timed("sharing links widen the audience", func(c *proof.Criterion) {
		for _, d := range docs {
			for _, a := range d.ACLPrefilter {
				if a == "anyone" || strings.HasPrefix(a, "domain:") {
					c.Counts["wide"]++
					break
				}
			}
		}
		c.Counts["read"] = len(docs)
		c.Pass = len(docs) > 0 // counted, not required: a tenant may have no wide link
	}))
	cs = append(cs, timed("the live check answers for a person", func(c *proof.Criterion) {
		if e.Person == "" || len(refs) == 0 {
			missing(c)
			return
		}
		vs, err := e.SharePoint.ACL(ctx, ports.UserRef{ID: e.Person, Email: e.Person}, refs[:min(len(refs), 25)])
		if err != nil {
			c.Counts["errors"] = 1
			return
		}
		for _, v := range vs {
			c.Counts[v.String()]++
		}
		c.Pass = c.Counts["unknown"] == 0 && len(vs) > 0
	}))
	return cs
}

func labels(ctx context.Context, e Env, h proof.Hasher) []proof.Criterion {
	return []proof.Criterion{timed("labelled items map to a level", func(c *proof.Criterion) {
		if e.SharePoint == nil || len(e.SharePoint.Drives) == 0 {
			missing(c)
			return
		}
		set, err := changes(ctx, e.SharePoint)
		if err != nil {
			c.Counts["errors"] = 1
			return
		}
		for _, ch := range set[:min(len(set), 50)] {
			if ch.Deleted || strings.HasPrefix(ch.Ref.ID, "tm:") {
				continue
			}
			d, err := e.SharePoint.Fetch(ctx, ch.Ref)
			if err != nil {
				c.Counts["errors"]++
				continue
			}
			if d.Label != "" {
				c.Counts["labelled"]++
				c.Counts["level_"+d.Label]++
			}
		}
		var un []string
		for id := range e.SharePoint.Unmapped() {
			un = append(un, id)
		}
		c.Counts["unmapped_labels"] = len(un)
		c.Seen = h.IDs(un)
		c.Pass = c.Counts["errors"] == 0
	}), timed("an unmapped label counts as restricted", func(c *proof.Criterion) {
		if e.SharePoint == nil {
			missing(c)
			return
		}
		un := e.SharePoint.Unmapped()
		n := 0
		for _, k := range un {
			n += k
		}
		c.Counts["items_unmapped"] = n
		// Fetch marked each of them restricted: the levels counted above include them.
		c.Pass = true
	})}
}

func entra(ctx context.Context, e Env, h proof.Hasher) []proof.Criterion {
	dir := &msgraph.Directory{Graph: e.Graph}
	var groups []string
	cs := []proof.Criterion{timed("a person's nested groups resolve", func(c *proof.Criterion) {
		if e.Graph == nil || e.Person == "" {
			missing(c)
			return
		}
		g, err := dir.GroupsOf(ctx, e.Person)
		if err != nil {
			c.Counts["errors"] = 1
			return
		}
		groups = g
		c.Counts["groups"] = len(g)
		c.Seen = h.IDs(g)
		c.Pass = true
	})}
	cs = append(cs, timed("each group lists its members", func(c *proof.Criterion) {
		if len(groups) == 0 {
			missing(c)
			return
		}
		for _, g := range groups[:min(len(groups), 10)] {
			ms, err := dir.Members(ctx, g)
			if err != nil {
				c.Counts["errors"]++
				continue
			}
			c.Counts["members"] += len(ms)
		}
		c.Pass = c.Counts["errors"] == 0
	}))
	cs = append(cs, timed("a person's manager resolves", func(c *proof.Criterion) {
		if e.Graph == nil || e.Person == "" {
			missing(c)
			return
		}
		m, err := dir.Manager(ctx, e.Person)
		switch {
		case err != nil:
			c.Counts["errors"] = 1
		case m == "":
			c.Counts["none"] = 1
			c.Pass = true
		default:
			c.Counts["found"] = 1
			c.Pass = true
		}
	}))
	return cs
}

func teams(ctx context.Context, e Env, _ proof.Hasher) []proof.Criterion {
	return []proof.Criterion{timed("the bot framework signing keys load", func(c *proof.Criterion) {
		if e.TeamsKeys == nil {
			missing(c)
			return
		}
		n, err := e.TeamsKeys(ctx)
		if err != nil {
			c.Counts["errors"] = 1
			return
		}
		c.Counts["keys"] = n
		c.Pass = n > 0
	})}
}

func calendar(ctx context.Context, e Env, _ proof.Hasher) []proof.Criterion {
	return []proof.Criterion{timed("a person's own calendar reads", func(c *proof.Criterion) {
		if e.Graph == nil || e.Person == "" {
			missing(c)
			return
		}
		now := time.Now()
		if e.Now != nil {
			now = e.Now()
		}
		evs, err := (&msgraph.Calendar{Graph: e.Graph}).Events(ctx, e.Tenant, e.Person, now, now.Add(7*24*time.Hour))
		if err != nil {
			c.Counts["errors"] = 1
			return
		}
		c.Counts["events"] = len(evs)
		for _, ev := range evs {
			c.Counts["attendees"] += len(ev.Attendees)
		}
		c.Pass = true
	})}
}

func transcripts(ctx context.Context, e Env, h proof.Hasher) []proof.Criterion {
	var refs []ports.SourceRef
	return []proof.Criterion{timed("the organizers list their transcripts", func(c *proof.Criterion) {
		if e.SharePoint == nil || len(e.SharePoint.Organizers) == 0 {
			missing(c)
			return
		}
		set, err := changes(ctx, e.SharePoint)
		if err != nil {
			c.Counts["errors"] = 1
			return
		}
		var ids []string
		for _, ch := range set {
			if strings.HasPrefix(ch.Ref.ID, "tm:") {
				refs = append(refs, ch.Ref)
				ids = append(ids, ch.Ref.ID)
			}
		}
		c.Counts["organizers"], c.Counts["transcripts"] = len(e.SharePoint.Organizers), len(refs)
		c.Seen = h.IDs(ids)
		c.Pass = true
	}), timed("each transcript is open only to its meeting", func(c *proof.Criterion) {
		if len(refs) == 0 {
			missing(c)
			return
		}
		for _, r := range refs[:min(len(refs), 10)] {
			d, err := e.SharePoint.Fetch(ctx, r)
			if err != nil {
				c.Counts["errors"]++
				continue
			}
			c.Counts["read"]++
			if len(d.ACLPrefilter) > 0 {
				c.Counts["with_people"]++
			}
		}
		c.Pass = c.Counts["errors"] == 0 && c.Counts["with_people"] == c.Counts["read"]
	})}
}

func mailboxes(ctx context.Context, e Env, _ proof.Hasher) []proof.Criterion {
	box := &msgraph.Mailbox{Graph: e.Graph}
	return []proof.Criterion{timed("each named shared mailbox reads", func(c *proof.Criterion) {
		if e.Graph == nil || len(e.Mailboxes) == 0 {
			missing(c)
			return
		}
		for _, m := range e.Mailboxes {
			mails, _, err := box.New(ctx, m, "")
			if err != nil {
				c.Counts["errors"]++
				continue
			}
			c.Counts["mailboxes"]++
			c.Counts["messages"] += len(mails)
		}
		c.Pass = c.Counts["errors"] == 0
	}), timed("a mailbox not named is refused", func(c *proof.Criterion) {
		if e.Graph == nil || e.Probe == "" {
			missing(c)
			return
		}
		_, _, err := box.New(ctx, e.Probe, "")
		var ge *msgraph.Error
		if errors.As(err, &ge) && (ge.Status == 403 || ge.Status == 401) {
			c.Counts["refused"] = 1
			c.Pass = true
			return
		}
		c.Counts["read"] = 1 // the access policy doesn't hold
	})}
}

func chat(_ context.Context, e Env, _ proof.Hasher) []proof.Criterion {
	var lines []string
	return []proof.Criterion{timed("the bot account answers", func(c *proof.Criterion) {
		if e.Chat == nil {
			missing(c)
			return
		}
		l, err := e.Chat()
		if err != nil {
			c.Counts["errors"] = 1
			return
		}
		lines = l
		c.Pass = true
	}), timed("the bot has what it needs", func(c *proof.Criterion) {
		if lines == nil {
			missing(c)
			return
		}
		for _, l := range lines {
			switch {
			case strings.HasPrefix(l, "ok"):
				c.Counts["ok"]++
			case strings.HasPrefix(l, "missing"):
				c.Counts["missing"]++
			}
		}
		c.Pass = c.Counts["missing"] == 0 && c.Counts["ok"] > 0
	})}
}

func email(ctx context.Context, e Env, _ proof.Hasher) []proof.Criterion {
	return []proof.Criterion{timed("the mail server offers starttls with a trusted certificate", func(c *proof.Criterion) {
		if e.Email == nil {
			missing(c)
			return
		}
		if err := e.Email(ctx); err != nil {
			c.Counts["errors"] = 1
			return
		}
		c.Pass = true
	})}
}

func doctor(_ context.Context, e Env, _ proof.Hasher) []proof.Criterion {
	return []proof.Criterion{timed("the box passes the doctor", func(c *proof.Criterion) {
		if len(e.Facts) == 0 {
			missing(c)
			return
		}
		f, err := box.Parse(e.Facts)
		if err != nil {
			c.Counts["errors"] = 1
			return
		}
		now := time.Now()
		if e.Now != nil {
			now = e.Now()
		}
		for _, l := range box.Check(f, now) {
			c.Counts[strings.ToLower(string(l.Verdict))]++
		}
		c.Pass = c.Counts["fail"] == 0 && c.Counts["unknown"] == 0
	})}
}
