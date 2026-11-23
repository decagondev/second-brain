package api

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// NotifyTest sends the fixed test notification (sb-ask(3), Notifications; SB-2602) to one
// person through the cell's real channel: sent false is a channel that doesn't know them,
// and via names the channel (mattermost, slack or email).
type NotifyTest func(ctx context.Context, who string) (sent bool, via string, err error)

// ErrNoChannel is a cell with no chat bot and no email to send from.
var ErrNoChannel = errors.New("api: the cell has no channel to notify through")

// ErrNoPerson is a test to someone who isn't one active directory user with an email.
var ErrNoPerson = errors.New("api: no one active by that name")

// TestEvery is how often one person can be sent a test.
const TestEvery = time.Minute

var tested = struct {
	sync.Mutex
	at map[string]time.Time
}{at: map[string]time.Time{}}

// testDue reports whether key may be sent a test now, and if so counts it.
func testDue(key string, now time.Time) bool {
	tested.Lock()
	defer tested.Unlock()
	if last, ok := tested.at[key]; ok && now.Sub(last) < TestEvery {
		return false
	}
	for k, t := range tested.at {
		if now.Sub(t) >= TestEvery {
			delete(tested.at, k)
		}
	}
	tested.at[key] = now
	return true
}

// runTest runs a test for key and writes its answer through ok and fail.
func runTest(w http.ResponseWriter, r *http.Request, test NotifyTest, tenant, key, who string, ok func(http.ResponseWriter, any), fail func(http.ResponseWriter, *errs.Error)) {
	if test == nil {
		fail(w, errs.New(errs.EUNAVAIL, "This cell can't send a test notification."))
		return
	}
	if !testDue(tenant+"|"+key, time.Now()) {
		fail(w, errs.New(errs.ERATE, "A test was sent less than a minute ago. Wait a minute and try again."))
		return
	}
	sent, via, err := test(r.Context(), who)
	switch {
	case errors.Is(err, ErrNoChannel):
		fail(w, errs.New(errs.EUNAVAIL, "This cell has no chat bot and no email to send notifications through."))
	case errors.Is(err, ErrNoPerson):
		fail(w, errs.New(errs.EINVAL, "No one active in the directory has that email."))
	case err != nil:
		fail(w, errs.New(errs.EUNAVAIL, "The channel didn't take the test. Check its settings and try again."))
	default:
		ok(w, map[string]any{"sent": sent, "via": via})
	}
}

func (h Notifications) test(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	out := h.k().out()
	runTest(w, r, h.Test, s.Tenant(), s.User(), s.User(), func(w http.ResponseWriter, v any) { out.ok(w, "", v) },
		func(w http.ResponseWriter, e *errs.Error) { out.fail(w, "", e) })
}

func (h Admin) notifyTest(w http.ResponseWriter, r *http.Request, s tenancy.Scope) {
	var body struct {
		To string `json:"to"`
	}
	if !readJSON(r, &body) || body.To == "" {
		h.fail(w, errs.New(errs.EINVAL, `The body must be JSON such as {"to": "ann@acme-logistics.example"}.`))
		return
	}
	runTest(w, r, h.NotifyTest, s.Tenant(), "email:"+body.To, body.To, h.ok, h.fail)
}
