package notify

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
)

type questions struct{ qs []ask.StewardQuestion }

func (q questions) PutStewardQuestion(context.Context, ask.StewardQuestion) error { return nil }
func (q questions) GetStewardQuestion(context.Context, string, string) (ask.StewardQuestion, bool, error) {
	return ask.StewardQuestion{}, false, nil
}
func (q questions) StewardQuestionsOf(context.Context, string, string) ([]ask.StewardQuestion, error) {
	return nil, nil
}
func (q questions) StewardQueue(context.Context, string, string) ([]ask.StewardQuestion, error) {
	return nil, nil
}
func (q questions) UpdateStewardQuestion(context.Context, string, string, func(*ask.StewardQuestion) error) (ask.StewardQuestion, error) {
	return ask.StewardQuestion{}, nil
}
func (q questions) StewardQuestionsBetween(_ context.Context, _ string, from, to time.Time) ([]ask.StewardQuestion, error) {
	var out []ask.StewardQuestion
	for _, x := range q.qs {
		if !x.At.Before(from) && x.At.Before(to) {
			out = append(out, x)
		}
	}
	return out, nil
}

type dir struct {
	groups map[string][]string
	users  map[string]ports.User
}

func (d dir) Members(_ context.Context, _, g string) ([]string, error) {
	m, ok := d.groups[g]
	if !ok {
		return nil, errors.New("no group")
	}
	return m, nil
}
func (d dir) User(_ context.Context, _, id string) (ports.User, bool, error) {
	u, ok := d.users[id]
	return u, ok, nil
}

type clock struct {
	t      time.Time
	found  bool
	weekly map[string]time.Time
}

func (c *clock) Last(_ context.Context, part string) (time.Time, bool, error) {
	if part == "weekly" {
		t, ok := c.weekly[part]
		return t, ok, nil
	}
	return c.t, c.found, nil
}
func (c *clock) Mark(_ context.Context, part string, t time.Time) error {
	if part == "weekly" {
		c.weekly[part] = t
		return nil
	}
	c.t, c.found = t, true
	return nil
}

type sent struct {
	to  Person
	msg Message
}
type sender struct {
	got  []sent
	fail string
}

func (s *sender) Send(_ context.Context, to Person, m Message) error {
	if to.Email == s.fail {
		return errors.New("down")
	}
	if strings.HasPrefix(to.Email, "unknown") {
		return ErrNoSuchPerson
	}
	s.got = append(s.got, sent{to, m})
	return nil
}

type prefs map[string]Prefs

func (p prefs) NotifyPrefs(_ context.Context, _, u string) (Prefs, error) { return p[u], nil }
func (p prefs) PutNotifyPrefs(_ context.Context, _, u string, x Prefs) error {
	p[u] = x
	return nil
}

func user(id string, active bool) ports.User {
	return ports.User{ID: id, Active: active, Emails: []string{id + "@acme.example"}}
}

func world(now time.Time) (Job, *sender, *clock) {
	qs := questions{[]ask.StewardQuestion{
		{ID: "qs_1", Asker: "usr_ann", Question: "SECRETQUESTION one", Space: "spc_finance", At: now.Add(-2 * time.Minute)},
		{ID: "qs_2", Asker: "usr_bob", Question: "SECRETQUESTION two", Space: "spc_finance", At: now.Add(-3 * time.Minute)},
		{ID: "qs_3", Asker: "usr_ann", Question: "SECRETQUESTION three", Space: "company", At: now.Add(-1 * time.Minute)},
		{ID: "qs_0", Asker: "usr_bob", Question: "SECRETQUESTION old", Space: "spc_finance", At: now.Add(-48 * time.Hour),
			Reply: &ask.StewardReply{Text: "SECRETREPLY", Cites: []string{"docs:SECRETCITE@1"}, At: now.Add(-time.Minute), NoteAt: now.Add(-30 * time.Second)}},
		{ID: "qs_9", Asker: "usr_cat", Question: "SECRETQUESTION nine", Space: "spc_finance", At: now.Add(-48 * time.Hour),
			Reply: &ask.StewardReply{Text: "SECRETREPLY", At: now.Add(-time.Hour)}}, // told last run
	}}
	d := dir{groups: map[string][]string{"finance-leads": {"usr_sam", "usr_gone", "usr_sam"}, "helpdesk": {"usr_hal"}},
		users: map[string]ports.User{"usr_ann": user("usr_ann", true), "usr_bob": user("usr_bob", true), "usr_cat": user("usr_cat", true),
			"usr_sam": user("usr_sam", true), "usr_gone": user("usr_gone", false), "usr_hal": user("usr_hal", true)}}
	s := &sender{}
	c := &clock{t: now.Add(-5 * time.Minute), found: true, weekly: map[string]time.Time{}}
	return Job{Tenant: "acme-logistics", Base: "https://brain.acme.example/", Questions: qs,
		Facts: func(sp string) ([]string, string, bool) {
			switch sp {
			case "spc_finance":
				return []string{"group:finance-leads", "group:missing"}, "restricted", true
			case "company":
				return []string{"group:helpdesk"}, "", true
			}
			return nil, "", false
		},
		Dir: d, Prefs: prefs{}, Send: s, Clock: c, Now: func() time.Time { return now }}, s, c
}

func TestRunTellsStewardsAndAskers(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	j, s, c := world(now)
	res, err := j.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range s.got {
		got = append(got, fmt.Sprintf("%s %s: %s <%s>", m.msg.Kind, m.to.ID, m.msg.Text, m.msg.Link))
		for _, bad := range []string{"SECRET", "usr_"} {
			if strings.Contains(m.msg.Text+m.msg.Link, bad) {
				t.Errorf("a message holds %q: %+v", bad, m.msg)
			}
		}
	}
	want := []string{
		"questions usr_hal: 1 new question sent company-wide <https://brain.acme.example/?steward=company>",
		"questions usr_sam: 2 new questions for the finance space <https://brain.acme.example/?steward=finance>",
		"replies usr_bob: 1 of your questions has a reply <https://brain.acme.example/?tab=questions>",
		"replies usr_bob: 1 of your questions was published as a note <https://brain.acme.example/?tab=questions>",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("sent:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if res.Sent != 4 || !c.t.Equal(now) {
		t.Errorf("result %+v, clock %v", res, c.t)
	}
	// A second run in the same minute tells nothing again.
	s.got = nil
	if _, err := j.Run(context.Background()); err != nil || len(s.got) != 0 {
		t.Errorf("second run sent %v, %v", s.got, err)
	}
}

func TestRunHonoursSettingsAndCountsFailures(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	j, s, _ := world(now)
	j.Prefs = prefs{"usr_sam": {Off: []string{KindQuestions}}}
	s.fail = "usr_bob@acme.example"
	failed := 0
	j.Failed = func(string) { failed++ }
	res, err := j.Run(context.Background())
	if !errors.Is(err, ErrSend) || res.Failed != 2 || failed != 2 || res.Skipped != 1 {
		t.Fatalf("result %+v, %v, %d", res, err, failed)
	}
	for _, m := range s.got {
		if m.to.ID == "usr_sam" {
			t.Error("a steward who turned questions off was told")
		}
	}
}

func TestRunWithoutSenderOrHistory(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	j, _, c := world(now)
	j.Send = nil
	if res, err := j.Run(context.Background()); err != nil || res != (Result{}) || !c.t.Equal(now) {
		t.Fatalf("no sender: %+v %v", res, err)
	}
	j, s, c := world(now)
	c.found = false
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(s.got) != 4 { // the first run reads FirstLookback
		t.Errorf("first run sent %d", len(s.got))
	}
}

func TestUnknownChatPersonIsSkipped(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	j, _, _ := world(now)
	d := j.Dir.(dir)
	d.users["usr_sam"] = ports.User{ID: "usr_sam", Active: true, Emails: []string{"unknown@acme.example"}}
	res, err := j.Run(context.Background())
	if err != nil || res.Skipped != 1 || res.Sent != 3 {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestPrefsCheck(t *testing.T) {
	p, err := Prefs{Off: []string{"replies", "digest", "replies"}}.Check()
	if err != nil || strings.Join(p.Off, ",") != "digest,replies" || p.Wants("replies") || !p.Wants("questions") {
		t.Fatalf("%+v %v", p, err)
	}
	if _, err := (Prefs{Off: []string{"everything"}}).Check(); !errors.Is(err, ErrKind) {
		t.Fatal(err)
	}
	if p, _ := (Prefs{}).Check(); p.Off == nil {
		t.Fatal("nil off")
	}
}

func TestMonday(t *testing.T) {
	berlin, _ := time.LoadLocation("Europe/Berlin")
	for _, c := range []struct {
		now  string
		zone *time.Location
		want string
	}{
		{"2026-11-23T08:00:00Z", nil, "2026-11-23T08:00:00Z"},    // a Monday, 08:00 UTC
		{"2026-11-23T07:59:00Z", nil, "2026-11-16T08:00:00Z"},    // not yet this week's
		{"2026-11-25T12:00:00Z", nil, "2026-11-23T08:00:00Z"},    // Wednesday
		{"2026-11-23T07:00:00Z", berlin, "2026-11-23T07:00:00Z"}, // 08:00 in Berlin is 07:00 UTC
		{"2026-11-23T06:59:00Z", berlin, "2026-11-16T07:00:00Z"},
	} {
		now, _ := time.Parse(time.RFC3339, c.now)
		if got := Monday(now, c.zone).Format(time.RFC3339); got != c.want {
			t.Errorf("Monday(%s) = %s; want %s", c.now, got, c.want)
		}
	}
}

type watches struct{ qs []ask.SavedQuestion }

func (w *watches) UpdateSaved(_ context.Context, _, user, id string, f func(*ask.SavedQuestion)) (bool, error) {
	for i := range w.qs {
		if w.qs[i].ID == id && w.qs[i].User == user {
			f(&w.qs[i])
			return true, nil
		}
	}
	return false, nil
}
func (w *watches) Watched(context.Context, string) ([]ask.SavedQuestion, error) {
	var out []ask.SavedQuestion
	for _, q := range w.qs {
		if q.Watch {
			out = append(out, q)
		}
	}
	return out, nil
}

func TestWeeklyDigestAndWatches(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC) // a Monday
	j, s, c := world(now)
	j.Questions = questions{}
	c.weekly["weekly"] = Monday(now, nil).AddDate(0, 0, -7)
	j.Digest, j.Zone = true, time.UTC
	j.Askers = func(context.Context, time.Time, time.Time) ([]string, error) {
		return []string{"usr_ann", "usr_bob", "usr_ann", "usr_gone", "usr_cat"}, nil
	}
	j.Changes = func(_ context.Context, u string) ([]string, int, error) {
		switch u {
		case "usr_ann":
			return []string{"Refund policy", "Carrier contracts"}, 3, nil
		case "usr_bob":
			return nil, 0, nil // nothing changed for bob
		}
		return []string{"Should not be sent"}, 0, nil
	}
	w := &watches{[]ask.SavedQuestion{
		{ID: "sq_1", User: "usr_cat", Question: "SECRETQUESTION a", Space: "spc_finance", Watch: true, Sources: "h1"},
		{ID: "sq_2", User: "usr_cat", Question: "SECRETQUESTION b", Space: "company", Watch: true, Sources: "h2"},
		{ID: "sq_3", User: "usr_bob", Question: "SECRETQUESTION c", Watch: true, Sources: "same"},
		{ID: "sq_4", User: "usr_gone", Question: "SECRETQUESTION d", Watch: true, Sources: "h4"},
		{ID: "sq_5", User: "usr_ann", Question: "SECRETQUESTION e", Watch: false, Sources: "h5"},
	}}
	j.Watches = w
	j.Sources = func(_ context.Context, u, _, q string) (string, error) {
		if u == "usr_bob" {
			return "same", nil
		}
		return "new", nil
	}
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range s.got {
		got = append(got, m.msg.Kind+" "+m.to.ID+": "+m.msg.Text+" <"+m.msg.Link+">")
	}
	want := []string{
		"digest usr_ann: 5 documents changed in your spaces this week: Refund policy; Carrier contracts and 3 more <https://brain.acme.example/?tab=documents>",
		"digest usr_cat: 1 document changed in your spaces this week: Should not be sent <https://brain.acme.example/?tab=documents>",
		"watch usr_cat: The sources of 2 of your saved questions changed <https://brain.acme.example/?tab=saved>",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("sent:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if w.qs[0].Sources != "new" || w.qs[1].Sources != "new" || w.qs[3].Sources != "h4" || w.qs[4].Sources != "h5" {
		t.Errorf("hashes %+v", w.qs)
	}
	// The same week again sends nothing weekly.
	s.got = nil
	j.Now = func() time.Time { return now.Add(time.Hour) }
	if _, err := j.Run(context.Background()); err != nil || len(s.got) != 0 {
		t.Fatalf("again: %v %v", s.got, err)
	}
	// Off: no digest, but watches are still checked.
	j, s, c = world(now.AddDate(0, 0, 7))
	j.Questions = questions{}
	c.weekly["weekly"] = now
	j.Digest, j.Askers, j.Changes = false, func(context.Context, time.Time, time.Time) ([]string, error) { return []string{"usr_ann"}, nil }, func(context.Context, string) ([]string, int, error) { return []string{"x"}, 0, nil }
	w.qs[0].Sources = "old"
	j.Watches, j.Sources = w, func(context.Context, string, string, string) (string, error) { return "new", nil }
	if _, err := j.Run(context.Background()); err != nil || len(s.got) != 2 || s.got[0].msg.Kind != KindWatch || s.got[1].msg.Kind != KindWatch {
		t.Fatalf("digest off: %+v %v", s.got, err)
	}
}

func TestFirstWeekSendsNothing(t *testing.T) {
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	j, s, c := world(now)
	j.Questions = questions{}
	j.Digest = true
	j.Askers = func(context.Context, time.Time, time.Time) ([]string, error) { return []string{"usr_ann"}, nil }
	j.Changes = func(context.Context, string) ([]string, int, error) { return []string{"x"}, 0, nil }
	if _, err := j.Run(context.Background()); err != nil || len(s.got) != 0 || !c.weekly["weekly"].Equal(Monday(now, nil)) {
		t.Fatalf("%v %v %v", s.got, err, c.weekly)
	}
}

// SB-2506: a paused steward gets nothing, the other stewards are still told, and the pause
// ends on its day.
func TestPauseSkipsOnlyThePausedPerson(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	p, err := Prefs{}.Pause(3, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (Prefs{}).Pause(31, now); !errors.Is(err, ErrPause) {
		t.Fatalf("a 31-day pause: %v", err)
	}
	j, s, _ := world(now)
	j.Facts = func(sp string) ([]string, string, bool) {
		return []string{"group:finance-leads", "group:helpdesk"}, "", sp == "spc_finance" || sp == "company"
	}
	j.Prefs = prefs{"usr_sam": p}
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	hal := false
	for _, m := range s.got {
		if m.to.ID == "usr_sam" {
			t.Fatal("a paused steward was told")
		}
		hal = hal || (m.to.ID == "usr_hal" && m.msg.Kind == KindQuestions)
	}
	if !hal {
		t.Fatal("the other steward wasn't told")
	}
	if !p.Paused(now.Add(71*time.Hour)) || p.Paused(now.Add(72*time.Hour)) {
		t.Fatal("the pause doesn't end on its day")
	}
	if q, _ := p.Pause(0, now); q.Paused(now) {
		t.Fatal("a pause of 0 days doesn't end it")
	}
}

// SB-2507: a steward's digest has a line per space they steward now, counts only.
func TestDigestStewardLines(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC) // a Monday
	j, s, c := world(now)
	c.weekly["weekly"] = Monday(now, nil).AddDate(0, 0, -7)
	j.Questions = questions{[]ask.StewardQuestion{
		{ID: "qs_1", Asker: "usr_ann", Question: "SECRETQUESTION", Space: "spc_finance", At: now.Add(-4 * 24 * time.Hour)},
		{ID: "qs_2", Asker: "usr_ann", Question: "SECRETQUESTION", Space: "spc_finance", At: now.Add(-time.Hour)},
		{ID: "qs_3", Asker: "usr_ann", Question: "SECRETQUESTION", Space: "spc_finance", At: now.Add(-9 * 24 * time.Hour),
			Reply: &ask.StewardReply{Text: "SECRETREPLY", At: now.Add(-2 * 24 * time.Hour)}},
		{ID: "qs_4", Asker: "usr_ann", Question: "SECRETQUESTION", Space: "company", At: now.Add(-20 * 24 * time.Hour),
			Reply: &ask.StewardReply{Text: "SECRETREPLY", At: now.Add(-19 * 24 * time.Hour)}},
		{ID: "qs_5", Asker: "usr_ann", Question: "SECRETQUESTION", Space: "spc_gone", At: now.Add(-time.Hour)},
	}}
	j.Clock.(*clock).t = now.Add(-time.Minute) // nothing new to tell in this run's window
	j.Digest, j.Zone = true, time.UTC
	j.Askers = func(context.Context, time.Time, time.Time) ([]string, error) {
		return []string{"usr_sam", "usr_ann"}, nil
	}
	j.Changes = func(_ context.Context, u string) ([]string, int, error) {
		if u == "usr_sam" {
			return []string{"Refund policy"}, 0, nil
		}
		return nil, 0, nil
	}
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range s.got {
		if m.msg.Kind == KindDigest {
			got = append(got, m.to.ID+": "+m.msg.Text)
		}
		if strings.Contains(m.msg.Text, "SECRET") {
			t.Fatal("a question or reply in the digest")
		}
	}
	want := []string{
		"usr_sam: 1 document changed in your spaces this week: Refund policy\nfinance: 1 answered this week, 2 still open, the oldest 4 days old",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("digests %q, want %q", got, want)
	}
	// hal stewards company-wide questions, but none waits or was answered this week.
	// Remove sam from the stewards: no line for them.
	j, s, c = world(now)
	c.weekly["weekly"] = Monday(now, nil).AddDate(0, 0, -7)
	j.Questions = questions{[]ask.StewardQuestion{{ID: "qs_1", Space: "spc_finance", At: now.Add(-2 * 24 * time.Hour)}}}
	j.Clock.(*clock).t = now.Add(-time.Minute)
	j.Digest, j.Askers, j.Changes = true, func(context.Context, time.Time, time.Time) ([]string, error) { return nil, nil }, func(context.Context, string) ([]string, int, error) { return nil, 0, nil }
	j.Dir.(dir).groups["finance-leads"] = []string{"usr_hal"}
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(s.got) != 1 || s.got[0].to.ID != "usr_hal" || s.got[0].msg.Text != "finance: 0 answered this week, 1 still open, the oldest 2 days old" {
		t.Fatalf("after sam left: %+v", s.got)
	}
}

// SB-2606: a token's owner is told once, seven days before it expires, by its name only;
// never a deprovisioned owner.
func TestTokenExpiryToldOnce(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	toks := []Expiring{
		{Owner: "usr_ann", Name: "nightly report", Expires: now.Add(TokenWarn - time.Minute)},
		{Owner: "usr_gone", Name: "old script", Expires: now.Add(TokenWarn - 2*time.Minute)},
		{Owner: "usr_bob", Name: "later", Expires: now.Add(TokenWarn + 3*time.Minute)},
	}
	j, s, c := world(now)
	j.Questions = questions{}
	j.Tokens = func(_ context.Context, from, to time.Time) ([]Expiring, error) {
		var out []Expiring
		for _, x := range toks {
			if !x.Expires.Before(from) && x.Expires.Before(to) {
				out = append(out, x)
			}
		}
		return out, nil
	}
	tell := func(at time.Time) []string {
		t.Helper()
		j.Now = func() time.Time { return at }
		s.got = nil
		if _, err := j.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, m := range s.got {
			got = append(got, m.to.ID+": "+m.msg.Kind+" "+m.msg.Text)
		}
		return got
	}
	if got := tell(now); strings.Join(got, "|") != "usr_ann: tokens Your API token nightly report expires in 7 days" {
		t.Fatalf("first run: %q", got)
	}
	if got := tell(now.Add(time.Minute)); len(got) != 0 {
		t.Fatalf("told twice: %q", got)
	}
	if got := tell(now.Add(5 * time.Minute)); strings.Join(got, "|") != "usr_bob: tokens Your API token later expires in 7 days" {
		t.Fatalf("bob's: %q", got)
	}
	_ = c
	j.Prefs = prefs{"usr_bob": {Off: []string{KindTokens}}}
	c.t = now.Add(4 * time.Minute)
	if got := tell(now.Add(5 * time.Minute)); len(got) != 0 {
		t.Fatalf("a person who turned tokens off was told: %q", got)
	}
}
