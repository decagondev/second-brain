// Package linear is the read-only Linear connector, `tickets: linear` (sb-connector(3),
// SB-211). It speaks Linear's GraphQL API with fixed, named operations and polls for
// changes by updatedAt, so it needs no inbound endpoint.
package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/ports"
)

const (
	// DefaultEndpoint is Linear's GraphQL API.
	DefaultEndpoint = "https://api.linear.app/graphql"
	ACLDeadline     = 800 * time.Millisecond
	ACLConcurrency  = 8
	kind            = "tickets"
	// PublicToken is the prefilter entry for a public team: any full workspace member.
	PublicToken = "linear:members"
)

// Registrar learns secret values so logs can redact them.
type Registrar interface{ Register(secret string) }

// Connector talks to one Linear workspace.
type Connector struct {
	Endpoint string
	HTTP     *http.Client
	Key      func(ctx context.Context) (string, error) // vault://tenants/T/connectors/tickets
	Redact   Registrar
}

var _ ports.Connector = (*Connector)(nil)

// ErrNotSupported is Act before SB-410.
var ErrNotSupported = errors.New("ENOTSUP: linear connector is read-only until SB-410")

func (c *Connector) APIVersion() int { return ports.ConnectorAPIVersion }

// gqlError is an HTTP failure or a GraphQL error.
type gqlError struct {
	status int
	code   string
}

func (e *gqlError) Error() string { return fmt.Sprintf("linear: %d %s", e.status, e.code) }

// Operations, sent by name; the fake and the real API both accept them.
const (
	opIssue = `query IssueByID($id: String!) { issue(id: $id) { id identifier title description updatedAt archivedAt trashed
  team { id key private members(first: 250) { nodes { id email } } } comments(first: 100) { nodes { body } } } }`
	opUser   = `query UserByEmail($email: String!) { users(filter: { email: { eq: $email } }) { nodes { id active guest } } }`
	opTeam   = `query TeamAccess($id: String!) { team(id: $id) { id private members(first: 250) { nodes { id } } } }`
	opIssues = `query IssuesSince($after: String, $since: DateTimeOrDuration) { issues(first: 100, after: $after, includeArchived: true,
  orderBy: updatedAt, filter: { updatedAt: { gte: $since } }) { nodes { id updatedAt archivedAt trashed } pageInfo { hasNextPage endCursor } } }`
	opTeams = `query TeamsChangedSince($since: DateTimeOrDuration) { teams(filter: { updatedAt: { gt: $since } }) { nodes { id updatedAt } }
  teamMemberships(filter: { updatedAt: { gt: $since } }) { nodes { updatedAt team { id } } } }`
	opTeamIssues = `query TeamIssues($id: String!, $after: String) { team(id: $id) { issues(first: 100, after: $after, includeArchived: true) {
  nodes { id } pageInfo { hasNextPage endCursor } } } }`
)

func (c *Connector) do(ctx context.Context, op, name string, vars map[string]any, v any) error {
	key, err := c.Key(ctx)
	if err != nil {
		return err
	}
	if c.Redact != nil {
		c.Redact.Register(key)
	}
	body, _ := json.Marshal(map[string]any{"query": op, "operationName": name, "variables": vars})
	wait := 20 * time.Millisecond
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", key)
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		_ = resp.Body.Close()
		if err != nil {
			return err
		}
		var env struct {
			Data   json.RawMessage `json:"data"`
			Errors []struct {
				Message    string `json:"message"`
				Extensions struct {
					Code string `json:"code"`
				} `json:"extensions"`
			} `json:"errors"`
		}
		_ = json.Unmarshal(raw, &env)
		limited := resp.StatusCode == 429 || (len(env.Errors) > 0 && env.Errors[0].Extensions.Code == "RATELIMITED")
		if limited {
			select {
			case <-ctx.Done():
				return &gqlError{429, "RATELIMITED"}
			case <-time.After(wait):
			}
			wait *= 2
			continue
		}
		if resp.StatusCode != 200 {
			return &gqlError{resp.StatusCode, "http"}
		}
		if len(env.Errors) > 0 {
			return &gqlError{200, env.Errors[0].Extensions.Code}
		}
		return json.Unmarshal(env.Data, v)
	}
}

type member struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type issue struct {
	ID          string  `json:"id"`
	Identifier  string  `json:"identifier"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	UpdatedAt   string  `json:"updatedAt"`
	ArchivedAt  *string `json:"archivedAt"`
	Trashed     bool    `json:"trashed"`
	Team        struct {
		ID      string `json:"id"`
		Key     string `json:"key"`
		Private bool   `json:"private"`
		Members struct {
			Nodes []member `json:"nodes"`
		} `json:"members"`
	} `json:"team"`
	Comments struct {
		Nodes []struct {
			Body string `json:"body"`
		} `json:"nodes"`
	} `json:"comments"`
}

func (i issue) gone() bool { return i.ArchivedAt != nil || i.Trashed }

func (c *Connector) issue(ctx context.Context, id string) (issue, error) {
	var r struct {
		Issue *issue `json:"issue"`
	}
	if err := c.do(ctx, opIssue, "IssueByID", map[string]any{"id": id}, &r); err != nil {
		return issue{}, err
	}
	if r.Issue == nil {
		return issue{}, &gqlError{200, "NOT_FOUND"}
	}
	return *r.Issue, nil
}

func micros(ts string) (int, error) {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return 0, err
	}
	return int(t.UnixMicro()), nil
}

// Fetch reads one issue with its comments.
func (c *Connector) Fetch(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
	is, err := c.issue(ctx, ref.ID)
	if err != nil {
		return ports.Document{}, err
	}
	if is.gone() {
		return ports.Document{}, fmt.Errorf("linear: %s is archived or trashed", ref.ID)
	}
	v, err := micros(is.UpdatedAt)
	if err != nil {
		return ports.Document{}, fmt.Errorf("linear: bad updatedAt on %s", ref.ID)
	}
	var b strings.Builder
	b.WriteString(is.Description)
	for _, cm := range is.Comments.Nodes {
		b.WriteString("\n\n" + cm.Body)
	}
	pre := []string{PublicToken}
	if is.Team.Private {
		pre = nil
		for _, m := range is.Team.Members.Nodes {
			pre = append(pre, strings.ToLower(m.Email))
		}
	}
	return ports.Document{Ref: ref, Version: v, Title: is.Identifier + " " + is.Title, Text: b.String(), ACLPrefilter: pre}, nil
}

// ACL answers per issue from team privacy and membership, live.
func (c *Connector) ACL(ctx context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs)) // zero value is Unknown (LAW-03)
	if u.Email == "" {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, ACLDeadline)
	defer cancel()
	var who struct {
		Users struct {
			Nodes []struct {
				ID     string `json:"id"`
				Active bool   `json:"active"`
				Guest  bool   `json:"guest"`
			} `json:"nodes"`
		} `json:"users"`
	}
	if err := c.do(ctx, opUser, "UserByEmail", map[string]any{"email": u.Email}, &who); err != nil {
		return out, nil
	}
	if len(who.Users.Nodes) == 0 || !who.Users.Nodes[0].Active {
		for i := range out {
			out[i] = ports.VerdictDeny // no Linear account, or deactivated
		}
		return out, nil
	}
	me := who.Users.Nodes[0]
	sem := make(chan struct{}, ACLConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i, r := range refs {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			is, err := c.issue(ctx, id)
			if err != nil || is.gone() {
				return // Unknown
			}
			member := false
			for _, m := range is.Team.Members.Nodes {
				member = member || m.ID == me.ID
			}
			v := ports.VerdictDeny
			if member || (!is.Team.Private && !me.Guest) {
				v = ports.VerdictAllow
			}
			mu.Lock()
			out[i] = v
			mu.Unlock()
		}(i, r.ID)
	}
	wg.Wait()
	if ctx.Err() != nil {
		return make([]ports.Verdict, len(refs)), nil // late is Unknown
	}
	return out, nil
}

// cursor is the poll position: the last updatedAt seen for issues and for teams.
type cursor struct {
	Issues string   `json:"issues"`
	AtIDs  []string `json:"at_ids,omitempty"` // issues already reported at exactly Issues
	Teams  string   `json:"teams"`
}

// Changes polls issues and team changes after the cursor.
func (c *Connector) Changes(ctx context.Context, cur ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	var pos cursor
	if cur != "" {
		if err := json.Unmarshal([]byte(cur), &pos); err != nil || pos.Issues == "" || pos.Teams == "" {
			return nil, cur, errors.New("linear: bad cursor")
		}
	} else {
		pos = cursor{Issues: "1970-01-01T00:00:00Z", Teams: time.Now().UTC().Format(time.RFC3339Nano)}
	}
	var set ports.ChangeSet
	after := ""
	// The poll asks for updatedAt >= the cursor time (fixed for the whole pagination), so an
	// issue updated in the same instant as the cursor isn't missed, and skips the ids the
	// cursor says were already seen at exactly that time.
	since, seenAt := pos.Issues, map[string]bool{}
	for _, id := range pos.AtIDs {
		seenAt[id] = true
	}
	for {
		var r struct {
			Issues struct {
				Nodes    []issue `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"issues"`
		}
		vars := map[string]any{"since": since}
		if after != "" {
			vars["after"] = after
		}
		if err := c.do(ctx, opIssues, "IssuesSince", vars, &r); err != nil {
			return nil, cur, err
		}
		for _, is := range r.Issues.Nodes {
			if is.UpdatedAt == since && seenAt[is.ID] {
				continue // already reported at exactly the cursor time
			}
			if !is.gone() || cur != "" { // a first sync doesn't list what's already gone
				set = append(set, ports.Change{Ref: ports.SourceRef{Kind: kind, ID: is.ID}, Deleted: is.gone()})
			}
			switch {
			case later(is.UpdatedAt, pos.Issues):
				pos.Issues, pos.AtIDs = is.UpdatedAt, []string{is.ID}
			case is.UpdatedAt == pos.Issues:
				pos.AtIDs = append(pos.AtIDs, is.ID)
			}
		}
		if !r.Issues.PageInfo.HasNextPage {
			break
		}
		after = r.Issues.PageInfo.EndCursor
	}
	if cur != "" {
		teams, latest, err := c.teamsChanged(ctx, pos.Teams)
		if err != nil {
			return nil, cur, err
		}
		pos.Teams = latest
		for _, t := range teams {
			ids, err := c.teamIssues(ctx, t)
			if err != nil {
				return nil, cur, err
			}
			for _, id := range ids {
				set = append(set, ports.Change{Ref: ports.SourceRef{Kind: kind, ID: id}, ACLOnly: true})
			}
		}
	}
	b, _ := json.Marshal(pos)
	return set, ports.Cursor(b), nil
}

func later(a, b string) bool {
	ta, e1 := time.Parse(time.RFC3339Nano, a)
	tb, e2 := time.Parse(time.RFC3339Nano, b)
	return e1 == nil && e2 == nil && ta.After(tb)
}

func (c *Connector) teamsChanged(ctx context.Context, since string) ([]string, string, error) {
	var r struct {
		Teams struct {
			Nodes []struct {
				ID        string `json:"id"`
				UpdatedAt string `json:"updatedAt"`
			} `json:"nodes"`
		} `json:"teams"`
		Memberships struct {
			Nodes []struct {
				UpdatedAt string `json:"updatedAt"`
				Team      struct {
					ID string `json:"id"`
				} `json:"team"`
			} `json:"nodes"`
		} `json:"teamMemberships"`
	}
	if err := c.do(ctx, opTeams, "TeamsChangedSince", map[string]any{"since": since}, &r); err != nil {
		return nil, since, err
	}
	seen := map[string]bool{}
	var ids []string
	latest := since
	add := func(id, at string) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
		if later(at, latest) {
			latest = at
		}
	}
	for _, t := range r.Teams.Nodes {
		add(t.ID, t.UpdatedAt)
	}
	for _, m := range r.Memberships.Nodes {
		add(m.Team.ID, m.UpdatedAt)
	}
	return ids, latest, nil
}

func (c *Connector) teamIssues(ctx context.Context, team string) ([]string, error) {
	var ids []string
	after := ""
	for {
		var r struct {
			Team struct {
				Issues struct {
					Nodes []struct {
						ID string `json:"id"`
					} `json:"nodes"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"issues"`
			} `json:"team"`
		}
		vars := map[string]any{"id": team}
		if after != "" {
			vars["after"] = after
		}
		if err := c.do(ctx, opTeamIssues, "TeamIssues", vars, &r); err != nil {
			return nil, err
		}
		for _, n := range r.Team.Issues.Nodes {
			ids = append(ids, n.ID)
		}
		if !r.Team.Issues.PageInfo.HasNextPage {
			return ids, nil
		}
		after = r.Team.Issues.PageInfo.EndCursor
	}
}

// Act is not supported until SB-410.
func (c *Connector) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, ErrNotSupported
}
