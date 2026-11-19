package laws

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	dirmem "example.com/second-brain/internal/adapters/directory/memory"
	sessmem "example.com/second-brain/internal/adapters/sessions/memory"
	"example.com/second-brain/internal/auth"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/scim"
)

// LAW-12 (SB-107): within 5 minutes of a SCIM deactivate, no request from that user
// succeeds. Checked on a fake clock, with and without a lost revocation fan-out, and
// without collateral damage: other users keep working.

// DeprovSystem is one cell with a directory, sessions and a SCIM endpoint, on a fake clock.
type DeprovSystem interface {
	Provision(user string) error          // SCIM create, active
	SignIn(user string) (string, error)   // returns a session id
	Deactivate(user string) error         // SCIM PATCH active=false
	Request(session string) (bool, error) // true if the request is served
	Advance(d time.Duration)
}

// NewDeprovSystem builds the system under test: the real SCIM endpoint and session
// checks (SB-107, SB-106) on a fake clock, with in-memory directory and sessions.
// lostFanout drops the immediate session revocation, so only re-validation can
// enforce the law.
var NewDeprovSystem = func(lostFanout bool) DeprovSystem {
	d := &deprov{now: time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC), ids: map[string]string{}}
	dir := dirmem.New()
	store := sessmem.New()
	var sessions ports.Sessions = store
	if lostFanout {
		sessions = lostRevocations{store}
	}
	d.scim = (&scim.Endpoint{Tenant: "acme-logistics", Directory: dir, Sessions: sessions,
		Token: func(context.Context) (string, error) { return "scim-token", nil }}).Handler()
	d.cell = &auth.Cell{Tenant: "acme-logistics", Sessions: store, Directory: dir, Now: func() time.Time { return d.now }}
	d.store = store
	return d
}

type deprov struct {
	now   time.Time
	scim  http.Handler
	cell  *auth.Cell
	store ports.Sessions
	ids   map[string]string // user name -> SCIM id
	n     int
}

type lostRevocations struct{ ports.Sessions }

func (lostRevocations) RevokeUser(context.Context, string, string) (int, error) { return 0, nil }

func (d *deprov) call(method, path, body string) (int, []byte) {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer scim-token")
	w := httptest.NewRecorder()
	d.scim.ServeHTTP(w, r)
	return w.Code, w.Body.Bytes()
}

func (d *deprov) Provision(user string) error {
	code, body := d.call("POST", "/scim/v2/Users", `{"userName":"`+user+`","active":true}`)
	var u struct{ ID string }
	if code != http.StatusCreated || json.Unmarshal(body, &u) != nil {
		return fmt.Errorf("SCIM create %s: %d %s", user, code, body)
	}
	d.ids[user] = u.ID
	return nil
}

func (d *deprov) SignIn(user string) (string, error) {
	d.n++
	s := ports.Session{ID: fmt.Sprintf("s%d", d.n), Tenant: "acme-logistics", User: d.ids[user], SignedIn: d.now, Validated: d.now}
	return s.ID, d.store.Put(context.Background(), s)
}

func (d *deprov) Deactivate(user string) error {
	body := `{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],"Operations":[{"op":"Replace","path":"active","value":false}]}`
	if code, b := d.call("PATCH", "/scim/v2/Users/"+d.ids[user], body); code != http.StatusOK {
		return fmt.Errorf("SCIM deactivate %s: %d %s", user, code, b)
	}
	return nil
}

func (d *deprov) Request(session string) (bool, error) {
	r := httptest.NewRequest("GET", "/v1/ask", nil)
	r.AddCookie(&http.Cookie{Name: auth.Cookie, Value: session})
	_, err := d.cell.Authenticate(r)
	return err == nil, nil
}

func (d *deprov) Advance(t time.Duration) { d.now = d.now.Add(t) }

const deprovBound = 5 * time.Minute

func law12(r *rand.Rand, mk func(bool) DeprovSystem) error {
	lost := r.Intn(2) == 0
	sys := mk(lost)
	users := []string{"usr_1", "usr_2", "usr_3", "usr_4"}[:2+r.Intn(3)]
	sessions := map[string][]string{}
	for _, u := range users {
		if err := sys.Provision(u); err != nil {
			return err
		}
		for k := 0; k < 1+r.Intn(2); k++ {
			s, err := sys.SignIn(u)
			if err != nil {
				return err
			}
			sessions[u] = append(sessions[u], s)
		}
	}
	sys.Advance(time.Duration(r.Intn(600)) * time.Second)
	leaver := users[r.Intn(len(users))]
	if err := sys.Deactivate(leaver); err != nil {
		return err
	}
	var since time.Duration
	for i := 0; i < 15; i++ {
		step := time.Duration(r.Intn(90)) * time.Second
		sys.Advance(step)
		since += step
		u := users[r.Intn(len(users))]
		s := sessions[u][r.Intn(len(sessions[u]))]
		ok, err := sys.Request(s)
		if err != nil {
			return err
		}
		switch {
		case u == leaver && ok && since >= deprovBound:
			return fmt.Errorf("deactivated %s was served %v after the deactivate (lost fan-out: %v)", u, since, lost)
		case u != leaver && !ok:
			return fmt.Errorf("%s, who is still active, was refused %v after another user's deactivate", u, since)
		}
	}
	return nil
}
