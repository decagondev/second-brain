package laws

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	spacemem "example.com/second-brain/internal/adapters/spaces/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-20 at the chat app (SB-1007): /ask in a space's channel is asked in that space, for
// its members only. Someone outside the space gets no answer, never a company-wide one
// that could carry the space's restricted content to them by another route; an unmapped
// channel is company-wide.

// ChatSpaces is the registry the chat app reads (the memory registry in the real system).
type ChatSpaces func(st *spacemem.Store) ports.SpaceStore

type scopeRecorder struct{ spaces []string }

func (s *scopeRecorder) Ask(_ context.Context, sc tenancy.Scope, _ ask.Request) (ask.Response, error) {
	s.spaces = append(s.spaces, sc.Space())
	return ask.Response{Trace: "tr_1"}, nil
}
func (s *scopeRecorder) Trace(context.Context, tenancy.Scope, string) (ask.Trace, error) {
	return ask.Trace{}, ask.ErrNotFound
}

type chatUsers struct{}

func (chatUsers) UsersByEmail(_ context.Context, tenant, email string) ([]ports.User, error) {
	return []ports.User{{ID: strings.Split(email, "@")[0], Tenant: tenant, Emails: []string{email}, Active: true}}, nil
}

func law20Chat(r *rand.Rand, wrap ChatSpaces) error {
	const tenant, team, secret = "acme-logistics", "T1", "s3cret"
	st := spacemem.New()
	groups := []string{"finance", "people", "eng"}
	owner := map[string]string{} // channel -> space
	member := map[string]string{}
	for i, n := range []string{"finance", "people", "eng"}[:1+r.Intn(3)] {
		ch := "#" + n + "-ask"
		g := groups[(i+r.Intn(2))%3]
		member[n] = g
		owner[ch] = n
		if err := st.PutSpace(context.Background(), ports.Space{ID: spaces.ID(n), Tenant: tenant, Name: n,
			Members: []string{"group:" + g}, Channels: []string{ch}}); err != nil {
			return err
		}
	}
	rec := &scopeRecorder{}
	now := time.Unix(1_800_000_000, 0)
	var asker []string
	app := &slackapp.App{Tenant: tenant, TeamID: team,
		Secret: func(context.Context) (string, error) { return secret, nil },
		Email:  func(_ context.Context, u string) (string, error) { return strings.ToLower(u) + "@acme.example", nil },
		Users:  chatUsers{},
		Groups: func(context.Context, string, string) ([]string, error) { return asker, nil },
		Issuer: tenancy.NewIssuer(tenant), Agent: rec, Spaces: wrap(st),
		Post: func(context.Context, string, slackapp.Message) error { return nil },
		Go:   func(f func()) { f() }, Now: func() time.Time { return now }}
	mux := http.NewServeMux()
	app.Register(mux)
	for i := 0; i < 10; i++ {
		asker = []string{groups[r.Intn(3)]}
		channel := []string{"finance-ask", "people-ask", "eng-ask", "general", "directmessage"}[r.Intn(5)]
		form := url.Values{"team_id": {team}, "user_id": {"UANN"}, "command": {"/ask"}, "text": {"what now?"},
			"response_url": {"https://hooks.slack.com/commands/T1/1/x"}, "channel_name": {channel}}
		body := form.Encode()
		ts := strconv.FormatInt(now.Unix(), 10)
		req := httptest.NewRequest("POST", "/chat/slack/commands", strings.NewReader(body))
		req.Header.Set("X-Slack-Request-Timestamp", ts)
		req.Header.Set("X-Slack-Signature", slackapp.Sign(secret, ts, []byte(body)))
		before := len(rec.spaces)
		mux.ServeHTTP(httptest.NewRecorder(), req)
		got := ""
		if len(rec.spaces) > before {
			got = rec.spaces[len(rec.spaces)-1]
		}
		want := "company"
		if sp, mapped := owner["#"+channel]; mapped {
			want = ""
			if member[sp] == asker[0] {
				want = spaces.ID(sp)
			}
		}
		if got != want {
			return fmt.Errorf("/ask in #%s as group %s was asked in %q, want %q (channels %v, members %v)", channel, asker[0], got, want, owner, member)
		}
	}
	return nil
}

// forgetful is a registry that loses its answer (an empty read): the mistake of treating
// "nothing came back" as "this channel isn't mapped".
type forgetful struct{ ports.SpaceStore }

func (forgetful) Spaces(context.Context, string) ([]ports.Space, error) { return nil, nil }
