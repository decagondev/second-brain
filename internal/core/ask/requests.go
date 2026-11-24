package ask

import (
	"context"
	"fmt"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/tenancy"
)

// Requests to a department (sb-ask(3), Requests; SB-4104): a member of a space with
// `requests` sends one in a category, and it waits in the space's steward queue as a
// steward question with a due date, so claims, replies, notifications and the climb up the
// tree work as for any question. The brain records and routes a request; it never does it.

// RequestMeta marks a steward question as a request: its category and when it is due,
// DueDays working days after it was sent.
type RequestMeta struct {
	Category string    `json:"category"`
	DueDays  int       `json:"due_days"`
	Due      time.Time `json:"due"`
}

// AddWorkingDays is the day n working days (Monday to Friday, in zone) after from, at the
// end of that day: a request sent on a Friday with 1 day is due at the end of Monday.
func AddWorkingDays(from time.Time, n int, zone *time.Location) time.Time {
	if zone == nil {
		zone = time.UTC
	}
	f := from.In(zone)
	d := time.Date(f.Year(), f.Month(), f.Day(), 12, 0, 0, 0, zone)
	for n > 0 {
		d = d.AddDate(0, 0, 1)
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			n--
		}
	}
	return time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, zone).UTC()
}

// Request is a member of space name sending text in category to its stewards.
func (x Stewards) Request(ctx context.Context, s tenancy.Scope, name, category, text string) (StewardQuestion, error) {
	q, err := question(s, x.Agent.Tenant, text)
	if err != nil {
		return StewardQuestion{}, err
	}
	if err := x.member(ctx, s, name); err != nil {
		return StewardQuestion{}, err
	}
	if x.Requests == nil {
		return StewardQuestion{}, fmt.Errorf("%w: space %q takes no requests", ErrInvalid, name)
	}
	cats, due, ok := x.Requests(name)
	if !ok {
		return StewardQuestion{}, fmt.Errorf("%w: space %q takes no requests", ErrInvalid, name)
	}
	cat := ""
	for _, c := range cats {
		if strings.EqualFold(c, strings.TrimSpace(category)) {
			cat = c
		}
	}
	if cat == "" {
		return StewardQuestion{}, fmt.Errorf("%w: category must be one of %s", ErrInvalid, strings.Join(cats, ", "))
	}
	key := SpaceKey(name)
	if _, _, ok := x.Facts(key); !ok {
		return StewardQuestion{}, fmt.Errorf("%w: nobody answers requests sent here", ErrInvalid)
	}
	now := x.Now().UTC()
	sq := StewardQuestion{ID: x.NewID("qs_"), Tenant: x.Agent.Tenant, Asker: s.User(), Question: q, Space: key, At: now,
		Request: &RequestMeta{Category: cat, DueDays: due, Due: AddWorkingDays(now, due, x.Zone)}}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: sq.ID, State: "asked",
		PayloadSHA256: hashOf(q), Reason: "request to stewards", Space: name}); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Store.PutStewardQuestion(ctx, sq); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	return sq, nil
}

// RequestForm is what a member of a space may send it: its categories and due days.
type RequestForm struct {
	Space      string   `json:"space"`
	Categories []string `json:"categories"`
	DueDays    int      `json:"due_days"`
}

// RequestFormOf is space name's request form for one of its members; a space that takes
// no requests has no categories.
func (x Stewards) RequestFormOf(ctx context.Context, s tenancy.Scope, name string) (RequestForm, error) {
	if err := x.member(ctx, s, name); err != nil {
		return RequestForm{}, err
	}
	f := RequestForm{Space: name, Categories: []string{}}
	if x.Requests != nil {
		if cats, due, ok := x.Requests(name); ok {
			f.Categories, f.DueDays = cats, due
		}
	}
	return f, nil
}
