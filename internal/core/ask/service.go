package ask

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"example.com/second-brain/internal/core/tenancy"
)

// Department service levels (sb-ask(3), Department service levels; SB-4507): per category
// and ISO week, how fast a space's requests and questions to its stewards get a first reply
// and a reply, and how many were replied to late. Counts and medians only, read from the
// queue's records, each row under 5 masked (LAW-07), and a department sees its teams' as
// counts only (LAW-24).

// The service levels' window and the category of a question that is no request.
const (
	ServiceWeeks    = 12
	ServiceQuestion = "question"
	ServiceMin      = 5
)

// ServiceRow is one category's week.
type ServiceRow struct {
	Category   string   `json:"category"`
	Week       string   `json:"week"`
	Count      int      `json:"count"`
	FirstReply *float64 `json:"first_reply_hours,omitempty"`
	Close      *float64 `json:"close_hours,omitempty"`
	LateShare  *float64 `json:"late_share,omitempty"`
	FewerThan5 bool     `json:"fewer_than_5,omitempty"`
}

// ServiceTeam is one team's counts over the window.
type ServiceTeam struct {
	Space      string `json:"space"`
	Count      int    `json:"count"`
	Closed     int    `json:"closed"`
	Late       int    `json:"late"`
	FewerThan5 bool   `json:"fewer_than_5,omitempty"`
}

// ServiceResponse is GET /v1/spaces/NAME/service.
type ServiceResponse struct {
	Space string        `json:"space"`
	Weeks []string      `json:"weeks"`
	Rows  []ServiceRow  `json:"rows"`
	Teams []ServiceTeam `json:"teams"`
}

// isoWeek is t's ISO week in zone, as 2026-W48.
func isoWeek(t time.Time, zone *time.Location) string {
	y, w := t.In(zoneOr(zone)).ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

// serviceWeeks are the last ServiceWeeks ISO weeks up to now's, oldest first, and when the
// oldest began.
func serviceWeeks(now time.Time, zone *time.Location) ([]string, time.Time) {
	n := now.In(zoneOr(zone))
	day := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, n.Location())
	monday := day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
	start := monday.AddDate(0, 0, -7*(ServiceWeeks-1))
	var out []string
	for i := 0; i < ServiceWeeks; i++ {
		out = append(out, isoWeek(start.AddDate(0, 0, 7*i).Add(12*time.Hour), zone))
	}
	return out, start.UTC()
}

// serviceCategory is a queue item's category: a request's, question for a question to the
// stewards, "" for anything else (a saved thread, a draft, a report).
func serviceCategory(q StewardQuestion) string {
	switch {
	case q.Request != nil:
		return q.Request.Category
	case q.Thread != nil || q.Draft != nil || q.Report != nil:
		return ""
	default:
		return ServiceQuestion
	}
}

// late reports whether a replied item was replied to after its due date (a request) or
// after OverdueAfter working days (a question).
func late(q StewardQuestion, zone *time.Location) bool {
	if q.Reply == nil {
		return false
	}
	if q.Request != nil {
		return q.Reply.At.After(q.Request.Due)
	}
	return WorkingDays(q.At, q.Reply.At, zone) >= OverdueAfter
}

// firstReply is when an item first had a claim or a reply, zero for neither.
func firstReply(q StewardQuestion) time.Time {
	t := q.FirstClaim
	if q.Reply != nil && (t.IsZero() || q.Reply.At.Before(t)) {
		t = q.Reply.At
	}
	return t
}

func median(xs []float64) *float64 {
	if len(xs) == 0 {
		return nil
	}
	sort.Float64s(xs)
	m := xs[len(xs)/2]
	if len(xs)%2 == 0 {
		m = (xs[len(xs)/2-1] + xs[len(xs)/2]) / 2
	}
	m = math.Round(m*10) / 10
	return &m
}

// ownItems are the items of a queue asked in it (not those that reached it from below),
// sent at or after from.
func ownItems(qs []StewardQuestion, key string, from time.Time) []StewardQuestion {
	var out []StewardQuestion
	for _, q := range qs {
		if q.Space == key && !q.At.Before(from) && serviceCategory(q) != "" {
			out = append(out, q)
		}
	}
	return out
}

// Service is space name's service levels, for a steward of it.
func (x Stewards) Service(ctx context.Context, s tenancy.Scope, name string) (ServiceResponse, error) {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return ServiceResponse{}, err
	}
	now := x.Now().UTC()
	weeks, from := serviceWeeks(now, x.Zone)
	out := ServiceResponse{Space: name, Weeks: weeks, Rows: []ServiceRow{}, Teams: []ServiceTeam{}}
	qs, err := x.Store.StewardQueue(ctx, x.Agent.Tenant, SpaceKey(name))
	if err != nil {
		return out, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	type key struct{ cat, week string }
	rows := map[key][]StewardQuestion{}
	for _, q := range ownItems(qs, SpaceKey(name), from) {
		k := key{serviceCategory(q), isoWeek(q.At, x.Zone)}
		rows[k] = append(rows[k], q)
	}
	keys := make([]key, 0, len(rows))
	for k := range rows {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].week < keys[j].week || keys[i].week == keys[j].week && keys[i].cat < keys[j].cat
	})
	for _, k := range keys {
		out.Rows = append(out.Rows, serviceRow(k.cat, k.week, rows[k], x.Zone))
	}
	if x.Children != nil {
		for _, team := range x.Children(name) {
			t, err := x.serviceTeam(ctx, team, from)
			if err != nil {
				return out, err
			}
			out.Teams = append(out.Teams, t)
		}
	}
	return out, nil
}

// serviceRow is one category's week: masked under ServiceMin.
func serviceRow(cat, week string, qs []StewardQuestion, zone *time.Location) ServiceRow {
	r := ServiceRow{Category: cat, Week: week, Count: len(qs)}
	if len(qs) < ServiceMin {
		return ServiceRow{Category: cat, Week: week, FewerThan5: true}
	}
	var first, closeH []float64
	closed, lateN := 0, 0
	for _, q := range qs {
		if t := firstReply(q); !t.IsZero() {
			first = append(first, t.Sub(q.At).Hours())
		}
		if q.Reply != nil {
			closed++
			closeH = append(closeH, q.Reply.At.Sub(q.At).Hours())
			if late(q, zone) {
				lateN++
			}
		}
	}
	r.FirstReply, r.Close = median(first), median(closeH)
	if closed > 0 {
		v := math.Round(float64(lateN)/float64(closed)*1000) / 1000
		r.LateShare = &v
	}
	return r
}

// serviceTeam is a team's counts over the window, masked under ServiceMin.
func (x Stewards) serviceTeam(ctx context.Context, team string, from time.Time) (ServiceTeam, error) {
	qs, err := x.Store.StewardQueue(ctx, x.Agent.Tenant, SpaceKey(team))
	if err != nil {
		return ServiceTeam{}, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	t := ServiceTeam{Space: team}
	for _, q := range ownItems(qs, SpaceKey(team), from) {
		t.Count++
		if q.Reply != nil {
			t.Closed++
			if late(q, x.Zone) {
				t.Late++
			}
		}
	}
	if t.Count < ServiceMin {
		return ServiceTeam{Space: team, FewerThan5: true}, nil
	}
	return t, nil
}

// lateCount is how many of a team's items over the window were replied to late, for the
// leader's week; ok is false under ServiceMin (SB-4503).
func (x Stewards) lateCount(ctx context.Context, team string) (int, bool, error) {
	_, from := serviceWeeks(x.Now().UTC(), x.Zone)
	t, err := x.serviceTeam(ctx, team, from)
	return t.Late, !t.FewerThan5, err
}
