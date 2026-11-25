// Package caltest is a recorded fake of the Google Calendar events endpoint the calendar
// reads, and Google's token endpoint for service accounts. It verifies the JWT's signature
// and scope, and answers each person only with their own events (sub).
package caltest

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

// Scope is the scope the fake accepts.
const Scope = "https://www.googleapis.com/auth/calendar.events.readonly"

// Event is one fake event in one person's calendar.
type Event struct {
	ID, Title, Agenda, Status string
	Start                     time.Time
	Attendees                 []string
	Declined                  bool
}

// Fake is one Workspace's calendars.
type Fake struct {
	Server *httptest.Server
	Key    *rsa.PrivateKey
	Email  string

	mu      sync.Mutex
	events  map[string][]Event // by owner email
	tokens  map[string]string
	covered map[string]bool // subjects the delegation covers; nil covers everyone
	limited int
	// Calls counts event reads, by subject.
	Calls map[string]int
}

// New starts a fake over TLS.
func New() *Fake {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	f := &Fake{Key: k, Email: "sb-calendar@acme-sb.iam.gserviceaccount.com", events: map[string][]Event{}, tokens: map[string]string{}, Calls: map[string]int{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", f.token)
	mux.HandleFunc("GET /calendar/v3/calendars/primary/events", f.list)
	f.Server = httptest.NewTLSServer(mux)
	return f
}

// Base is the API base for the calendar.
func (f *Fake) Base() string { return f.Server.URL }

// KeyJSON is a service account key file pointing at this fake's token endpoint.
func (f *Fake) KeyJSON() string {
	der, _ := x509.MarshalPKCS8PrivateKey(f.Key)
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	b, _ := json.Marshal(map[string]string{"type": "service_account", "client_email": f.Email,
		"private_key": string(pemKey), "token_uri": f.Server.URL + "/token"})
	return string(b)
}

// Put adds an event to owner's calendar.
func (f *Fake) Put(owner string, e Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events[strings.ToLower(owner)] = append(f.events[strings.ToLower(owner)], e)
}

// Cover limits the delegation to these subjects.
func (f *Fake) Cover(emails ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.covered = map[string]bool{}
	for _, e := range emails {
		f.covered[strings.ToLower(e)] = true
	}
}

// RateLimitNext makes the next n reads answer 429.
func (f *Fake) RateLimitNext(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.limited += n
}

func fail(w http.ResponseWriter, status int, reason string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": status, "message": reason}})
}

func (f *Fake) token(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.PostFormValue("assertion"), ".")
	if len(parts) != 3 {
		fail(w, 400, "invalid_grant")
		return
	}
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(&f.Key.PublicKey, crypto.SHA256, sum[:], sig) != nil {
		fail(w, 400, "invalid_grant")
		return
	}
	var c struct{ Iss, Sub, Scope, Aud string }
	p, _ := base64.RawURLEncoding.DecodeString(parts[1])
	_ = json.Unmarshal(p, &c)
	f.mu.Lock()
	covered := f.covered == nil || f.covered[strings.ToLower(c.Sub)]
	f.mu.Unlock()
	if c.Iss != f.Email || c.Scope != Scope || c.Aud != f.Server.URL+"/token" || c.Sub == "" {
		fail(w, 400, "invalid_scope")
		return
	}
	if !covered {
		fail(w, 401, "unauthorized_client")
		return
	}
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	tok := "ya29." + base64.RawURLEncoding.EncodeToString(b)
	f.mu.Lock()
	f.tokens[tok] = strings.ToLower(c.Sub)
	f.mu.Unlock()
	_ = json.NewEncoder(w).Encode(map[string]any{"access_token": tok, "expires_in": 3600, "token_type": "Bearer"})
}

func (f *Fake) list(w http.ResponseWriter, r *http.Request) {
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	f.mu.Lock()
	sub, ok := f.tokens[tok]
	limited := f.limited > 0
	if limited {
		f.limited--
	}
	if ok {
		f.Calls[sub]++
	}
	evs := append([]Event{}, f.events[sub]...)
	f.mu.Unlock()
	switch {
	case !ok:
		fail(w, 401, "authError")
		return
	case limited:
		fail(w, 429, "rateLimitExceeded")
		return
	}
	q := r.URL.Query()
	from, _ := time.Parse(time.RFC3339, q.Get("timeMin"))
	to, _ := time.Parse(time.RFC3339, q.Get("timeMax"))
	var items []map[string]any
	for _, e := range evs {
		if e.Start.Before(from) || !e.Start.Before(to) {
			continue
		}
		var att []map[string]any
		for _, a := range e.Attendees {
			st := "accepted"
			if strings.EqualFold(a, sub) && e.Declined {
				st = "declined"
			}
			att = append(att, map[string]any{"email": a, "self": strings.EqualFold(a, sub), "responseStatus": st})
		}
		status := e.Status
		if status == "" {
			status = "confirmed"
		}
		items = append(items, map[string]any{"id": e.ID, "status": status, "summary": e.Title, "description": e.Agenda,
			"start": map[string]string{"dateTime": e.Start.UTC().Format(time.RFC3339)}, "attendees": att})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
}
