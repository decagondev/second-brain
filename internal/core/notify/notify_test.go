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
	t     time.Time
	found bool
}

func (c *clock) Last(context.Context) (time.Time, bool, error) { return c.t, c.found, nil }
func (c *clock) Mark(_ context.Context, t time.Time) error     { c.t, c.found = t, true; return nil }

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
	c := &clock{t: now.Add(-5 * time.Minute), found: true}
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
