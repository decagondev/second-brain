// Package hubspot is the read-only HubSpot CRM connector, `crm: hubspot` (sb-connector(3),
// SB-610). It indexes companies and deals, never contacts, checks record-level access live
// from the user's role, teams and the record's owner, and polls the CRM search API.
package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/ports"
)

const (
	// DefaultBase is HubSpot's API.
	DefaultBase    = "https://api.hubapi.com"
	ACLDeadline    = 800 * time.Millisecond
	ACLConcurrency = 8
	kind           = "crm"
	// PublicToken is the broad prefilter entry: any HubSpot user; the live check decides.
	PublicToken = "hubspot:users"
)

// Types are the indexed object types. Contacts are personal data and never indexed.
var Types = []string{"companies", "deals"}

// Access levels in the role map.
const (
	LevelAll   = "all"
	LevelTeam  = "team"
	LevelOwned = "owned"
)

// Creds are the private app token and the role map, read from the vault at call time.
type Creds struct {
	Token string            `json:"token"`
	Roles map[string]string `json:"roles"` // HubSpot role id -> all, team or owned
}

// Registrar learns secret values so logs can redact them.
type Registrar interface{ Register(secret string) }

// Connector talks to one HubSpot portal.
type Connector struct {
	Base   string
	HTTP   *http.Client
	Creds  func(ctx context.Context) (string, error) // vault://tenants/T/connectors/crm (JSON)
	Redact Registrar
}

var _ ports.Connector = (*Connector)(nil)

// ErrNotSupported is Act on this read-only connector.
var ErrNotSupported = errors.New("ENOTSUP: hubspot connector is read-only")

func (c *Connector) APIVersion() int { return ports.ConnectorAPIVersion }

func (c *Connector) creds(ctx context.Context) (Creds, error) {
	raw, err := c.Creds(ctx)
	if err != nil {
		return Creds{}, err
	}
	var k Creds
	if err := json.Unmarshal([]byte(raw), &k); err != nil || k.Token == "" {
		return Creds{}, errors.New("hubspot: credentials are not {token, roles}")
	}
	if c.Redact != nil {
		c.Redact.Register(k.Token)
	}
	for role, lvl := range k.Roles {
		if lvl != LevelAll && lvl != LevelTeam && lvl != LevelOwned {
			return Creds{}, fmt.Errorf("hubspot: role %s has level %q; use all, team or owned", role, lvl)
		}
	}
	return k, nil
}

// apiError is an HTTP failure.
type apiError struct{ status int }

func (e *apiError) Error() string { return fmt.Sprintf("hubspot: HTTP %d", e.status) }

func notFound(err error) bool {
	var e *apiError
	return errors.As(err, &e) && e.status == http.StatusNotFound
}

func (c *Connector) do(ctx context.Context, k Creds, method, path string, body, v any) error {
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	wait := 20 * time.Millisecond
	for {
		req, err := http.NewRequestWithContext(ctx, method, c.Base+path, bytes.NewReader(raw))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+k.Token)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		out, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		_ = resp.Body.Close()
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			select {
			case <-ctx.Done():
				return &apiError{429}
			case <-time.After(wait):
			}
			wait *= 2
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return &apiError{resp.StatusCode}
		}
		return json.Unmarshal(out, v)
	}
}

// fields are the properties read for each type; they become the document's text, in order.
var fields = map[string][]string{
	"companies": {"name", "domain", "industry", "description"},
	"deals":     {"dealname", "dealstage", "amount", "closedate", "description"},
}

var labels = map[string]string{"name": "Name", "domain": "Domain", "industry": "Industry", "description": "Description",
	"dealname": "Deal", "dealstage": "Stage", "amount": "Amount", "closedate": "Close date"}

type record struct {
	ID         string            `json:"id"`
	Properties map[string]string `json:"properties"`
}

func split(ref ports.SourceRef) (string, string, error) {
	typ, id, ok := strings.Cut(ref.ID, "/")
	if !ok || fields[typ] == nil || id == "" || strings.ContainsAny(id, "/?#") {
		return "", "", fmt.Errorf("hubspot: bad record id %q; use companies/ID or deals/ID", ref.ID)
	}
	return typ, id, nil
}

func (c *Connector) record(ctx context.Context, k Creds, ref ports.SourceRef) (string, record, error) {
	typ, id, err := split(ref)
	if err != nil {
		return "", record{}, err
	}
	props := append([]string{"hs_lastmodifieddate", "hubspot_owner_id", "hubspot_team_id"}, fields[typ]...)
	var r record
	err = c.do(ctx, k, http.MethodGet, "/crm/v3/objects/"+typ+"/"+url.PathEscape(id)+"?archived=false&properties="+url.QueryEscape(strings.Join(props, ",")), nil, &r)
	return typ, r, err
}

func micros(ts string) (int, error) {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return 0, err
	}
	return int(t.UnixMicro()), nil
}

// Fetch reads one company or deal.
func (c *Connector) Fetch(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
	k, err := c.creds(ctx)
	if err != nil {
		return ports.Document{}, err
	}
	typ, r, err := c.record(ctx, k, ref)
	if err != nil {
		return ports.Document{}, err
	}
	v, err := micros(r.Properties["hs_lastmodifieddate"])
	if err != nil {
		return ports.Document{}, fmt.Errorf("hubspot: bad hs_lastmodifieddate on %s", ref.ID)
	}
	var lines []string
	for _, f := range fields[typ] {
		if x := strings.TrimSpace(r.Properties[f]); x != "" {
			lines = append(lines, labels[f]+": "+x)
		}
	}
	title := r.Properties["name"]
	if typ == "deals" {
		title = r.Properties["dealname"]
	}
	pre := []string{PublicToken}
	if o := r.Properties["hubspot_owner_id"]; o != "" {
		if email, err := c.ownerEmail(ctx, k, o); err == nil && email != "" {
			pre = append(pre, strings.ToLower(email))
		}
	}
	return ports.Document{Ref: ref, Version: v, Title: title, Text: strings.Join(lines, "\n"), ACLPrefilter: pre}, nil
}

func (c *Connector) ownerEmail(ctx context.Context, k Creds, owner string) (string, error) {
	var o struct {
		Email string `json:"email"`
	}
	err := c.do(ctx, k, http.MethodGet, "/crm/v3/owners/"+url.PathEscape(owner), nil, &o)
	return o.Email, err
}

// who is the asker as HubSpot sees them.
type who struct {
	owner string
	level string
	teams map[string]bool
}

// whoIs finds the user. found is false when HubSpot has no active user by that email.
func (c *Connector) whoIs(ctx context.Context, k Creds, email string) (w who, found bool, err error) {
	var u struct {
		ID         string `json:"id"`
		RoleID     string `json:"roleId"`
		SuperAdmin bool   `json:"superAdmin"`
	}
	if err := c.do(ctx, k, http.MethodGet, "/settings/v3/users/"+url.PathEscape(email)+"?idProperty=EMAIL", nil, &u); err != nil {
		if notFound(err) {
			return who{}, false, nil
		}
		return who{}, false, err
	}
	var os struct {
		Results []struct {
			ID       string `json:"id"`
			UserID   string `json:"userId"`
			Archived bool   `json:"archived"`
			Teams    []struct {
				ID string `json:"id"`
			} `json:"teams"`
		} `json:"results"`
	}
	if err := c.do(ctx, k, http.MethodGet, "/crm/v3/owners?archived=false&email="+url.QueryEscape(email), nil, &os); err != nil {
		return who{}, false, err
	}
	if len(os.Results) == 0 || os.Results[0].Archived || os.Results[0].UserID != u.ID {
		return who{}, false, nil // no active owner: deactivated
	}
	w = who{owner: os.Results[0].ID, teams: map[string]bool{}}
	for _, t := range os.Results[0].Teams {
		w.teams[t.ID] = true
	}
	switch {
	case u.SuperAdmin:
		w.level = LevelAll
	default:
		w.level = k.Roles[u.RoleID] // "" when the role isn't mapped: Unknown
	}
	return w, true, nil
}

// ACL answers per record from the user's level, teams and the record's owner, live.
func (c *Connector) ACL(ctx context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs)) // zero value is Unknown (LAW-03)
	if u.Email == "" {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, ACLDeadline)
	defer cancel()
	k, err := c.creds(ctx)
	if err != nil {
		return out, nil
	}
	me, found, err := c.whoIs(ctx, k, u.Email)
	if err != nil {
		return out, nil
	}
	if !found {
		for i := range out {
			out[i] = ports.VerdictDeny // no HubSpot user, or deactivated
		}
		return out, nil
	}
	if me.level == "" {
		return out, nil // role not in the role map: Unknown
	}
	sem := make(chan struct{}, ACLConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i, ref := range refs {
		wg.Add(1)
		go func(i int, ref ports.SourceRef) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			_, r, err := c.record(ctx, k, ref)
			if err != nil {
				return // archived, missing or failed: Unknown
			}
			owner, team := r.Properties["hubspot_owner_id"], r.Properties["hubspot_team_id"]
			mine := owner != "" && owner == me.owner
			v := ports.VerdictDeny
			switch me.level {
			case LevelAll:
				v = ports.VerdictAllow
			case LevelTeam:
				if mine || (team != "" && me.teams[team]) {
					v = ports.VerdictAllow
				}
			case LevelOwned:
				if mine {
					v = ports.VerdictAllow
				}
			}
			mu.Lock()
			out[i] = v
			mu.Unlock()
		}(i, ref)
	}
	wg.Wait()
	if ctx.Err() != nil {
		return make([]ports.Verdict, len(refs)), nil // late is Unknown
	}
	return out, nil
}

// pos is one type's poll position.
type pos struct {
	Modified string   `json:"modified"` // the last hs_lastmodifieddate seen
	AtIDs    []string `json:"at_ids,omitempty"`
	Archived string   `json:"archived"` // the last archivedAt seen
	ArchIDs  []string `json:"arch_ids,omitempty"`
}

const epoch = "1970-01-01T00:00:00Z"

// Changes polls the CRM search API and the archived lists after the cursor.
func (c *Connector) Changes(ctx context.Context, cur ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	cursor := map[string]*pos{}
	if cur != "" {
		if err := json.Unmarshal([]byte(cur), &cursor); err != nil {
			return nil, cur, errors.New("hubspot: bad cursor")
		}
		for _, t := range Types {
			if p := cursor[t]; p == nil || p.Modified == "" || p.Archived == "" {
				return nil, cur, errors.New("hubspot: bad cursor")
			}
		}
	} else {
		for _, t := range Types {
			cursor[t] = &pos{Modified: epoch, Archived: epoch}
		}
	}
	k, err := c.creds(ctx)
	if err != nil {
		return nil, cur, err
	}
	var set ports.ChangeSet
	for _, t := range Types {
		p := cursor[t]
		mod, err := c.modified(ctx, k, t, p, cur == "")
		if err != nil {
			return nil, cur, err
		}
		arch, err := c.archived(ctx, k, t, p, cur == "")
		if err != nil {
			return nil, cur, err
		}
		set = append(set, mod...)
		set = append(set, arch...)
	}
	b, _ := json.Marshal(cursor)
	return set, ports.Cursor(b), nil
}

func later(a, b string) bool {
	ta, e1 := time.Parse(time.RFC3339Nano, a)
	tb, e2 := time.Parse(time.RFC3339Nano, b)
	return e1 == nil && e2 == nil && ta.After(tb)
}

func near(a, b string) bool {
	ta, e1 := time.Parse(time.RFC3339Nano, a)
	tb, e2 := time.Parse(time.RFC3339Nano, b)
	d := ta.Sub(tb)
	return e1 == nil && e2 == nil && d < time.Second && d > -time.Second
}

// modified lists records changed at or after the cursor, skipping those already reported at
// exactly the cursor time, and advances it.
func (c *Connector) modified(ctx context.Context, k Creds, typ string, p *pos, first bool) (ports.ChangeSet, error) {
	since, err := time.Parse(time.RFC3339Nano, p.Modified)
	if err != nil {
		return nil, errors.New("hubspot: bad cursor")
	}
	seen := map[string]bool{}
	for _, id := range p.AtIDs {
		seen[id] = true
	}
	var set ports.ChangeSet
	sinceStr, after := p.Modified, ""
	for {
		body := map[string]any{
			"filterGroups": []any{map[string]any{"filters": []any{map[string]any{
				"propertyName": "hs_lastmodifieddate", "operator": "GTE", "value": strconv.FormatInt(since.UnixMilli(), 10)}}}},
			"sorts":      []any{map[string]any{"propertyName": "hs_lastmodifieddate", "direction": "ASCENDING"}},
			"properties": []string{"createdate", "hs_lastmodifieddate", "hubspot_owner_assigneddate"},
			"limit":      100,
		}
		if after != "" {
			body["after"] = after
		}
		var r struct {
			Results []record `json:"results"`
			Paging  *struct {
				Next struct {
					After string `json:"after"`
				} `json:"next"`
			} `json:"paging"`
		}
		if err := c.do(ctx, k, http.MethodPost, "/crm/v3/objects/"+typ+"/search", body, &r); err != nil {
			return nil, err
		}
		for _, rec := range r.Results {
			m := rec.Properties["hs_lastmodifieddate"]
			if m == sinceStr && seen[rec.ID] {
				continue
			}
			// Just reassigned, and not just created: the owner or team changed (permission-only).
			aclOnly := !first && near(rec.Properties["hubspot_owner_assigneddate"], m) && !near(rec.Properties["createdate"], m)
			set = append(set, ports.Change{Ref: ports.SourceRef{Kind: kind, ID: typ + "/" + rec.ID}, ACLOnly: aclOnly})
			switch {
			case later(m, p.Modified):
				p.Modified, p.AtIDs = m, []string{rec.ID}
			case m == p.Modified:
				p.AtIDs = append(p.AtIDs, rec.ID)
			}
		}
		if r.Paging == nil || r.Paging.Next.After == "" {
			return set, nil
		}
		after = r.Paging.Next.After
	}
}

// archived lists records archived after the cursor as deletes. A first sync only moves the
// cursor: it doesn't list what's already gone.
func (c *Connector) archived(ctx context.Context, k Creds, typ string, p *pos, first bool) (ports.ChangeSet, error) {
	seen := map[string]bool{}
	for _, id := range p.ArchIDs {
		seen[id] = true
	}
	var set ports.ChangeSet
	start, after := p.Archived, ""
	for {
		var r struct {
			Results []struct {
				ID         string `json:"id"`
				ArchivedAt string `json:"archivedAt"`
			} `json:"results"`
			Paging *struct {
				Next struct {
					After string `json:"after"`
				} `json:"next"`
			} `json:"paging"`
		}
		path := "/crm/v3/objects/" + typ + "?archived=true&limit=100"
		if after != "" {
			path += "&after=" + url.QueryEscape(after)
		}
		if err := c.do(ctx, k, http.MethodGet, path, nil, &r); err != nil {
			return nil, err
		}
		for _, rec := range r.Results {
			a := rec.ArchivedAt
			if later(start, a) || (a == start && seen[rec.ID]) {
				continue
			}
			if !first {
				set = append(set, ports.Change{Ref: ports.SourceRef{Kind: kind, ID: typ + "/" + rec.ID}, Deleted: true})
			}
			switch {
			case later(a, p.Archived):
				p.Archived, p.ArchIDs = a, []string{rec.ID}
			case a == p.Archived:
				p.ArchIDs = append(p.ArchIDs, rec.ID)
			}
		}
		if r.Paging == nil || r.Paging.Next.After == "" {
			return set, nil
		}
		after = r.Paging.Next.After
	}
}

// Act is not supported: the connector is read-only.
func (c *Connector) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, ErrNotSupported
}
