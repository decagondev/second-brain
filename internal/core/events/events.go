// Package events posts signed events out (sb-api(3), Events out; SB-4404): the kind, the
// space, an id and the time, never a title, text or name (LAW-07).
package events

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"example.com/second-brain/internal/core/audit"
)

// Event is one thing that happened, as ids.
type Event struct {
	Kind  string    `json:"kind"`
	Space string    `json:"space"`
	ID    string    `json:"id"`
	At    time.Time `json:"at"`
	// Version makes an event once per version of its item (a goal's), never sent.
	Version int `json:"-"`
}

// Key is what makes an event the same event on a later run.
func (e Event) Key() string {
	k := e.Kind + "|" + e.Space + "|" + e.ID
	if e.Version > 0 {
		k += "@" + strconv.Itoa(e.Version)
	}
	return k
}

// Source finds the events of one kind now.
type Source func(ctx context.Context, now time.Time) ([]Event, error)

// Sent remembers the events delivered, so each goes once.
type Sent interface {
	Delivered(key string) bool
	MarkDelivered(key string, at time.Time) error
}

// Backoff is the wait before each retry within a run.
var Backoff = []time.Duration{2 * time.Second, 8 * time.Second}

// MinKey is the shortest signing key.
const MinKey = 32

// Job posts each new event of the chosen kinds once.
type Job struct {
	URL     string
	Kinds   []string
	Sources map[string]Source
	Key     func(ctx context.Context) ([]byte, error)
	HTTP    *http.Client
	Audit   *audit.Log
	Sent    Sent
	Now     func() time.Time
	Sleep   func(time.Duration)
}

// Result counts one run.
type Result struct{ Sent, Failed int }

// Sign is the signature header's value for a body at a time: sha256= and the HMAC-SHA256
// of the Unix seconds, a dot and the body.
func Sign(key, body []byte, at time.Time) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(strconv.FormatInt(at.Unix(), 10) + "."))
	m.Write(body)
	return "sha256=" + hex.EncodeToString(m.Sum(nil))
}

// Verify checks a signature as a receiver does: the right key and a time within 5 minutes.
func Verify(key, body []byte, ts, sig string, now time.Time) bool {
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}
	at := time.Unix(sec, 0)
	if d := now.Sub(at); d > 5*time.Minute || d < -5*time.Minute {
		return false
	}
	return hmac.Equal([]byte(Sign(key, body, at)), []byte(sig))
}

// Run finds and posts the new events. A key that can't be read sends nothing.
func (j Job) Run(ctx context.Context) (Result, error) {
	var res Result
	if j.URL == "" || len(j.Kinds) == 0 {
		return res, nil
	}
	key, err := j.Key(ctx)
	if err != nil {
		return res, fmt.Errorf("events: the signing key can't be read: %w", err)
	}
	if len(key) < MinKey {
		return res, fmt.Errorf("events: the signing key is under %d bytes", MinKey)
	}
	now := j.Now().UTC()
	var all []Event
	for _, k := range j.Kinds {
		src := j.Sources[k]
		if src == nil {
			continue
		}
		evs, err := src(ctx, now)
		if err != nil {
			return res, fmt.Errorf("events: %s: %w", k, err)
		}
		for _, e := range evs {
			e.Kind = k
			if !j.Sent.Delivered(e.Key()) {
				all = append(all, e)
			}
		}
	}
	sort.SliceStable(all, func(a, b int) bool { return all[a].At.Before(all[b].At) })
	var failed error
	for _, e := range all {
		ok := j.deliver(ctx, key, e)
		state := "event_sent"
		if !ok {
			state, res.Failed = "event_failed", res.Failed+1
		} else {
			res.Sent++
		}
		if j.Audit != nil {
			if _, err := j.Audit.Write(ctx, audit.Event{Actor: "sys_events", Kind: "access", Subject: e.ID, State: state, Reason: e.Kind,
				Space: auditSpace(e.Space)}); err != nil {
				return res, fmt.Errorf("events: audit: %w", err)
			}
		}
		if ok {
			if err := j.Sent.MarkDelivered(e.Key(), now); err != nil {
				return res, fmt.Errorf("events: %w", err)
			}
		}
	}
	if res.Failed > 0 {
		failed = fmt.Errorf("events: %d of %d deliveries failed", res.Failed, len(all))
	}
	return res, failed
}

// deliver posts one event, retrying with backoff; it says whether a 2xx came back.
func (j Job) deliver(ctx context.Context, key []byte, e Event) bool {
	e.At = e.At.UTC().Truncate(time.Second)
	body, _ := json.Marshal(e)
	for try := 0; ; try++ {
		if j.post(ctx, key, body) == nil {
			return true
		}
		if try >= len(Backoff) {
			return false
		}
		if j.Sleep != nil {
			j.Sleep(Backoff[try])
		}
	}
}

var errStatus = errors.New("events: the receiver didn't answer 2xx")

func (j Job) post(ctx context.Context, key, body []byte) error {
	at := j.Now()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, j.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-SB-Timestamp", strconv.FormatInt(at.Unix(), 10))
	req.Header.Set("X-SB-Signature", Sign(key, body, at))
	hc := j.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errStatus
	}
	return nil
}

func auditSpace(name string) string {
	if name == "" || name == "company" {
		return ""
	}
	return "spc_" + name
}

// Memory is a Sent kept in memory, for tests and one-off runs.
type Memory map[string]time.Time

// Delivered implements Sent.
func (m Memory) Delivered(key string) bool { _, ok := m[key]; return ok }

// MarkDelivered implements Sent.
func (m Memory) MarkDelivered(key string, at time.Time) error { m[key] = at; return nil }
