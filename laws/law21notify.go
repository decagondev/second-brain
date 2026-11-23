package laws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	dirmem "example.com/second-brain/internal/adapters/directory/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/ports"
)

// Notifications (SB-2101, SB-2102) under LAW-21 and LAW-07. A question to a space's
// stewards is told to its stewards as the directory has them when the job runs: someone
// who left the stewards group, or was deprovisioned, after the question was asked hears
// nothing, and someone who joined hears it. An asker hears that their question has a
// reply. No message says what was asked, what was replied, what was cited or who asked.
// The real notify job, traces store, directory and Slack direct-message adapter, against
// a fake Slack that records what each person was sent.

// NotifySystem builds the notify job under test. It is built before the directory changes.
type NotifySystem interface {
	Job(store *tracemem.Store, dir *dirmem.Dir, facts ask.SpaceFacts, send notify.Sender, now func() time.Time) notify.Job
}

// RealNotify is the real one.
type RealNotify struct{}

// Job implements NotifySystem.
func (RealNotify) Job(store *tracemem.Store, dir *dirmem.Dir, facts ask.SpaceFacts, send notify.Sender, now func() time.Time) notify.Job {
	return notify.Job{Tenant: "acme-logistics", Base: "https://brain.acme-logistics.example", Questions: store, Facts: facts,
		Dir: dir, Prefs: store, Send: send, Clock: memClock{}, Now: now}
}

type memClock map[string]time.Time

func (c memClock) Last(_ context.Context, part string) (time.Time, bool, error) {
	t, ok := c[part]
	return t, ok, nil
}
func (c memClock) Mark(_ context.Context, part string, t time.Time) error { c[part] = t; return nil }

// fakeSlack is Slack's Web API for direct messages: each email is a user, each post kept.
type fakeSlack struct {
	mu    sync.Mutex
	posts map[string][]string // email -> texts
}

func (f *fakeSlack) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch r.URL.Path {
	case "/users.lookupByEmail":
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "user": map[string]string{"id": "U" + r.URL.Query().Get("email")}})
	case "/conversations.open":
		var b map[string]string
		_ = json.NewDecoder(r.Body).Decode(&b)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "channel": map[string]string{"id": "D" + strings.TrimPrefix(b["users"], "U")}})
	case "/chat.postMessage":
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		ch, _ := b["channel"].(string)
		text, _ := b["text"].(string)
		f.posts[strings.TrimPrefix(ch, "D")] = append(f.posts[strings.TrimPrefix(ch, "D")], text)
		_, _ = io.WriteString(w, `{"ok":true}`)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func law21Notify(r *rand.Rand, sys NotifySystem) error {
	ctx := context.Background()
	const tenant = "acme-logistics"
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	store, dir := tracemem.New(), dirmem.New()
	names := []string{"ann", "bob", "cat", "dan", "eve", "fay", "gus", "hal", "ivy", "jon"}
	r.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	for _, n := range names {
		if err := dir.PutUser(ctx, ports.User{ID: "usr_" + n, Tenant: tenant, UserName: n, Emails: []string{n + "@acme.example"}, Active: true}); err != nil {
			return err
		}
	}
	stewards := names[:2+r.Intn(3)]
	askers := names[5:]
	group := func(ns []string) ports.Group {
		g := ports.Group{ID: "grp_leads", Tenant: tenant, Name: "support-leads"}
		for _, n := range ns {
			g.Members = append(g.Members, "usr_"+n)
		}
		return g
	}
	if err := dir.PutGroup(ctx, group(stewards)); err != nil {
		return err
	}
	facts := func(space string) ([]string, string, bool) {
		return []string{"group:support-leads"}, "internal", space == "spc_support" || space == "company"
	}
	slack := &fakeSlack{posts: map[string][]string{}}
	srv := httptest.NewServer(slack)
	defer srv.Close()
	send := slackapp.DM{Base: srv.URL, Token: func(context.Context) (string, error) { return "xoxb-law", nil }, Client: srv.Client()}
	clock := now
	job := sys.Job(store, dir, facts, send, func() time.Time { return clock })
	// The first run reads nothing yet; questions and replies come after it.
	if _, err := job.Run(ctx); err != nil {
		return err
	}
	canary := fmt.Sprintf("CANARY%x", r.Int63())
	replied := map[string]bool{}
	asked := 1 + r.Intn(4)
	for i := 0; i < asked; i++ {
		a := askers[r.Intn(len(askers))]
		q := ask.StewardQuestion{ID: fmt.Sprintf("qs_%d", i), Tenant: tenant, Asker: "usr_" + a, Question: canary + " question",
			Space: "spc_support", Trace: "tr_" + canary, At: now.Add(time.Duration(i+1) * time.Second)}
		if r.Intn(2) == 0 {
			q.Reply = &ask.StewardReply{By: "usr_" + stewards[0], Text: canary + " reply", Cites: []string{"docs:" + canary + "@1"}, At: q.At.Add(time.Second)}
			replied[a] = true
		}
		if err := store.PutStewardQuestion(ctx, q); err != nil {
			return err
		}
	}
	// The directory changes between the questions and the run.
	now2 := stewards[r.Intn(len(stewards)):]
	if r.Intn(2) == 0 {
		now2 = append(append([]string(nil), now2...), names[4])
	}
	if err := dir.PutGroup(ctx, group(now2)); err != nil {
		return err
	}
	gone := ""
	if r.Intn(2) == 0 {
		gone = names[r.Intn(len(names))]
		if err := dir.PutUser(ctx, ports.User{ID: "usr_" + gone, Tenant: tenant, UserName: gone, Emails: []string{gone + "@acme.example"}}); err != nil {
			return err
		}
	}
	clock = now.Add(5 * time.Minute)
	if _, err := job.Run(ctx); err != nil {
		return err
	}
	var want, got []string
	for _, n := range now2 {
		if n != gone {
			want = append(want, n)
		}
	}
	for email, texts := range slack.posts {
		for _, t := range texts {
			for _, bad := range []string{canary, "usr_", "@acme.example"} {
				if strings.Contains(t, bad) {
					return fmt.Errorf("LAW-07: a notification to %s holds %q: %q", email, bad, t)
				}
			}
			n := strings.TrimSuffix(email, "@acme.example")
			switch {
			case strings.Contains(t, "new question"):
				got = append(got, n)
			case strings.Contains(t, "has a reply") || strings.Contains(t, "have replies"):
				if !replied[n] || n == gone {
					return fmt.Errorf("%s was told of a reply they don't have (deprovisioned: %v)", n, n == gone)
				}
				delete(replied, n)
			default:
				return fmt.Errorf("an unexpected notification to %s: %q", email, t)
			}
		}
	}
	sort.Strings(want)
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		return fmt.Errorf("LAW-21: new questions were told to %v; the stewards now are %v (were %v, deprovisioned %q)", got, want, stewards, gone)
	}
	delete(replied, gone)
	if len(replied) > 0 {
		return fmt.Errorf("askers with replies weren't told: %v", replied)
	}
	return nil
}
