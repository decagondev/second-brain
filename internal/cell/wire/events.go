package wire

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/events"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
)

// Events out (sb-api(3), Events out; SB-4404).

// EventsWindow is how far back a decision or a policy is new.
const EventsWindow = 7 * 24 * time.Hour

// sentFile keeps the events delivered, by key, for 90 days: state/events/sent.json.
type sentFile struct {
	path string
	mu   sync.Mutex
	m    map[string]time.Time
}

func openSent(state string, now time.Time) (*sentFile, error) {
	f := &sentFile{path: filepath.Join(state, "events", "sent.json"), m: map[string]time.Time{}}
	b, err := os.ReadFile(f.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &f.m); err != nil {
			return nil, err
		}
	}
	for k, at := range f.m {
		if now.Sub(at) > 90*24*time.Hour {
			delete(f.m, k)
		}
	}
	return f, nil
}

func (f *sentFile) Delivered(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.m[key]
	return ok
}

func (f *sentFile) MarkDelivered(key string, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.m[key] = at
	if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
		return err
	}
	b, _ := json.Marshal(f.m)
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}

// RunEvents posts the new events of the kinds the manifest chose, each once; nothing when
// it chose none.
func (c *Cell) RunEvents(ctx context.Context) (events.Result, error) {
	m := c.Manifest()
	if m.Events == nil {
		return events.Result{}, nil
	}
	now := time.Now()
	sent, err := openSent(c.Config.State, now)
	if err != nil {
		return events.Result{}, err
	}
	hc := c.opt.EventsHTTP
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	j := events.Job{URL: m.Events.URL, Kinds: m.Events.Kinds, Sources: c.eventSources(m), HTTP: hc, Audit: c.Audit, Sent: sent,
		Now: time.Now, Sleep: c.opt.Sleep,
		Key: func(ctx context.Context) ([]byte, error) {
			k, err := c.eventKey(ctx)
			return []byte(k), err
		}}
	if j.Sleep == nil {
		j.Sleep = time.Sleep
	}
	return j.Run(ctx)
}

// spaceChunks are the latest chunks of each space, by name; a cell without spaces is one
// brain, "company".
func (c *Cell) spaceChunks(ctx context.Context, m manifest.Tenant) (map[string][]ports.Chunk, error) {
	lk, ok := c.Index.(ports.DocLookup)
	if !ok {
		return nil, nil
	}
	out := map[string][]ports.Chunk{}
	names := sortedSpaces(m)
	if len(names) == 0 {
		cs, err := lk.Docs(ctx, c.Config.Tenant, ports.DocQuery{})
		out["company"] = latest(cs)
		return out, err
	}
	for _, name := range names {
		cs, err := lk.Docs(ctx, c.Config.Tenant, ports.DocQuery{Space: ask.SpaceKey(name)})
		if err != nil {
			return nil, err
		}
		out[name] = latest(cs)
	}
	return out, nil
}

func docID(ch ports.Chunk) string { return ch.Kind + ":" + ch.SourceID }

// eventSources finds each kind's events from what the cell already keeps.
func (c *Cell) eventSources(m manifest.Tenant) map[string]events.Source {
	tenant := c.Config.Tenant
	chunked := func(pick func(name string, ch ports.Chunk, now time.Time) (events.Event, bool)) events.Source {
		return func(ctx context.Context, now time.Time) ([]events.Event, error) {
			sp, err := c.spaceChunks(ctx, m)
			if err != nil {
				return nil, err
			}
			var out []events.Event
			seen := map[string]bool{}
			for _, name := range append([]string{"company"}, sortedSpaces(m)...) {
				for _, ch := range sp[name] {
					if e, ok := pick(name, ch, now); ok && !seen[name+e.ID] {
						seen[name+e.ID] = true
						out = append(out, e)
					}
				}
			}
			return out, nil
		}
	}
	return map[string]events.Source{
		"decision": chunked(func(name string, ch ports.Chunk, now time.Time) (events.Event, bool) {
			return events.Event{Space: name, ID: docID(ch), At: ch.Seen}, ask.IsDecision(ch) && !ch.Seen.IsZero() && now.Sub(ch.Seen) <= EventsWindow
		}),
		"goal_at_risk": chunked(func(name string, ch ports.Chunk, now time.Time) (events.Event, bool) {
			g, ok := ask.ParseGoal(ch.Text)
			return events.Event{Space: name, ID: docID(ch), At: now, Version: ch.Version},
				ok && ch.Start == 0 && (g.Status == "at risk" || g.Status == "off track")
		}),
		"request_overdue": func(ctx context.Context, now time.Time) ([]events.Event, error) {
			var out []events.Event
			for _, name := range append([]string{"company"}, sortedSpaces(m)...) {
				q, err := c.Traces.StewardQueue(ctx, tenant, ask.SpaceKey(name))
				if err != nil {
					return nil, err
				}
				for _, sq := range q {
					if sq.Request != nil && sq.Reply == nil && now.After(sq.Request.Due) && sq.Space == ask.SpaceKey(name) {
						out = append(out, events.Event{Space: name, ID: sq.ID, At: sq.Request.Due})
					}
				}
			}
			return out, nil
		},
		"policy_published": func(ctx context.Context, now time.Time) ([]events.Event, error) {
			var out []events.Event
			for _, name := range sortedSpaces(m) {
				ps, err := c.Traces.Policies(ctx, tenant, name)
				if err != nil {
					return nil, err
				}
				for _, p := range ps {
					if now.Sub(p.At) <= EventsWindow {
						out = append(out, events.Event{Space: name, ID: p.Doc, At: p.At})
					}
				}
			}
			return out, nil
		},
		"action_due": func(ctx context.Context, now time.Time) ([]events.Event, error) {
			as, err := c.Traces.ActionsSince(ctx, tenant, now.AddDate(-1, 0, 0))
			if err != nil {
				return nil, err
			}
			var out []events.Event
			for _, a := range as {
				d, err := time.Parse("2006-01-02", a.Due)
				if a.DoneAt == nil && err == nil && now.After(d.Add(24*time.Hour)) {
					out = append(out, events.Event{Space: a.Space, ID: a.ID, At: d})
				}
			}
			return out, nil
		},
	}
}
