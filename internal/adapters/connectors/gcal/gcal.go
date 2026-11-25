// Package gcal reads a person's own Google Calendar, read-only, as them (sb-connector(3),
// Calendars; SB-4202). It is not a source: nothing it reads is indexed or kept.
package gcal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// Scope is the only OAuth scope the calendar asks for.
const Scope = "https://www.googleapis.com/auth/calendar.events.readonly"

// MaxEvents is the most events one call returns.
const MaxEvents = 50

// Tokens mints an access token for a subject (gdrive.ServiceAccount with Scope).
type Tokens interface {
	Token(ctx context.Context, subject string) (string, error)
}

// Counter counts a failed read (sb_calendar_failed_total).
type Counter interface {
	Add(name, help string, labels map[string]string, v float64)
}

// Calendar is the calendar for one tenant.
type Calendar struct {
	Base    string // https://www.googleapis.com
	Tokens  Tokens
	HTTP    *http.Client
	Metrics Counter
	Sleep   func(context.Context, time.Duration) error
}

var _ ports.Calendar = (*Calendar)(nil)

type event struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Start       struct {
		DateTime string `json:"dateTime"`
		Date     string `json:"date"`
	} `json:"start"`
	Attendees []struct {
		Email          string `json:"email"`
		Self           bool   `json:"self"`
		ResponseStatus string `json:"responseStatus"`
	} `json:"attendees"`
}

var errDenied = errors.New("gcal: the delegation doesn't cover this person")

// Events lists email's own events starting in [from, to), as them. A person the delegation
// doesn't cover reads no events, never someone else's.
func (c *Calendar) Events(ctx context.Context, _ string, email string, from, to time.Time) ([]ports.CalendarEvent, error) {
	if email == "" {
		return nil, nil
	}
	evs, err := c.list(ctx, email, from, to)
	if errors.Is(err, errDenied) {
		c.count("denied")
		return nil, nil
	}
	if err != nil {
		c.count("error")
		return nil, err
	}
	var out []ports.CalendarEvent
	for _, e := range evs {
		if e.Status == "cancelled" {
			continue
		}
		declined := false
		var who []string
		for _, a := range e.Attendees {
			if a.Self && a.ResponseStatus == "declined" {
				declined = true
			}
			if a.Email != "" {
				who = append(who, a.Email)
			}
		}
		if declined {
			continue
		}
		start, err := time.Parse(time.RFC3339, e.Start.DateTime)
		if err != nil {
			if start, err = time.Parse("2006-01-02", e.Start.Date); err != nil {
				continue
			}
		}
		out = append(out, ports.CalendarEvent{ID: e.ID, Title: e.Summary, Agenda: e.Description, Start: start.UTC(), Attendees: who})
	}
	return out, nil
}

func (c *Calendar) list(ctx context.Context, email string, from, to time.Time) ([]event, error) {
	tok, err := c.Tokens.Token(ctx, email)
	if err != nil {
		return nil, errDenied
	}
	q := url.Values{"singleEvents": {"true"}, "orderBy": {"startTime"}, "maxResults": {fmt.Sprint(MaxEvents)},
		"timeMin": {from.UTC().Format(time.RFC3339)}, "timeMax": {to.UTC().Format(time.RFC3339)}}
	u := strings.TrimRight(c.Base, "/") + "/calendar/v3/calendars/primary/events?" + q.Encode()
	backoff := 200 * time.Millisecond
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, fmt.Errorf("gcal: %w", err)
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		_ = resp.Body.Close()
		switch {
		case resp.StatusCode == http.StatusOK:
			var v struct {
				Items []event `json:"items"`
			}
			if err := json.Unmarshal(body, &v); err != nil {
				return nil, errors.New("gcal: an events page that isn't JSON")
			}
			if len(v.Items) > MaxEvents {
				v.Items = v.Items[:MaxEvents]
			}
			return v.Items, nil
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return nil, errDenied
		case resp.StatusCode == http.StatusTooManyRequests && attempt < 3:
			wait := backoff << attempt
			if dl, ok := ctx.Deadline(); ok && time.Until(dl) < wait {
				return nil, errors.New("gcal: rate limited past the deadline")
			}
			if err := c.sleep(ctx, wait); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("gcal: HTTP %d", resp.StatusCode)
		}
	}
}

func (c *Calendar) sleep(ctx context.Context, d time.Duration) error {
	if c.Sleep != nil {
		return c.Sleep(ctx, d)
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *Calendar) count(why string) {
	if c.Metrics != nil {
		c.Metrics.Add("sb_calendar_failed_total", "Calendar reads that failed or weren't covered.", map[string]string{"why": why}, 1)
	}
}
