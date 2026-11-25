package wire

import (
	"context"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/connectors/gcal"
	"example.com/second-brain/internal/adapters/connectors/gcal/caltest"
	"example.com/second-brain/internal/adapters/connectors/gdrive"
)

// SB-4202: the calendar as the cell wires it, through the service account with the
// calendar scope, against the recorded fake.
type calCount map[string]float64

func (c calCount) Add(_, _ string, l map[string]string, v float64) { c[l["why"]] += v }

func TestEventsReadsOnlyThePersonsOwnCalendar(t *testing.T) {
	f := caltest.New()
	defer f.Server.Close()
	at := time.Date(2026, 11, 25, 10, 30, 0, 0, time.UTC)
	f.Put("ann@acme-logistics.example", caltest.Event{ID: "e1", Title: "Carrier review", Agenda: "- Rates\n- Contracts", Start: at, Attendees: []string{"ann@acme-logistics.example", "bob@acme-logistics.example"}})
	f.Put("ann@acme-logistics.example", caltest.Event{ID: "e2", Title: "Cancelled", Start: at, Status: "cancelled"})
	f.Put("ann@acme-logistics.example", caltest.Event{ID: "e3", Title: "Declined", Start: at, Attendees: []string{"ann@acme-logistics.example"}, Declined: true})
	f.Put("bob@acme-logistics.example", caltest.Event{ID: "b1", Title: "Bob's", Start: at})
	sa := &gdrive.ServiceAccount{Key: func(context.Context) (string, error) { return f.KeyJSON(), nil }, HTTP: f.Server.Client(), Now: time.Now, Scope: gcal.Scope}
	m := calCount{}
	c := &gcal.Calendar{Base: f.Base(), Tokens: sa, HTTP: f.Server.Client(), Metrics: m, Sleep: func(context.Context, time.Duration) error { return nil }}
	from, to := at.Add(-time.Hour), at.Add(time.Hour)
	evs, err := c.Events(context.Background(), "acme-logistics", "ann@acme-logistics.example", from, to)
	if err != nil || len(evs) != 1 || evs[0].ID != "e1" || evs[0].Agenda == "" || len(evs[0].Attendees) != 2 || !evs[0].Start.Equal(at) {
		t.Fatalf("ann's events: %+v %v", evs, err)
	}
	f.RateLimitNext(1)
	if evs, err := c.Events(context.Background(), "acme-logistics", "bob@acme-logistics.example", from, to); err != nil || len(evs) != 1 || evs[0].ID != "b1" {
		t.Fatalf("bob's events after a 429: %+v %v", evs, err)
	}
	f.Cover("ann@acme-logistics.example")
	sa2 := &gdrive.ServiceAccount{Key: func(context.Context) (string, error) { return f.KeyJSON(), nil }, HTTP: f.Server.Client(), Now: time.Now, Scope: gcal.Scope}
	c.Tokens = sa2
	if evs, err := c.Events(context.Background(), "acme-logistics", "eve@acme-logistics.example", from, to); err != nil || len(evs) != 0 || m["denied"] != 1 {
		t.Fatalf("someone the delegation doesn't cover: %+v %v %v", evs, err, m)
	}
}
