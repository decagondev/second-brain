// Package bamboohr is the read-only HRIS connector (sb-connector(3), hris: bamboohr;
// SB-1104): employee records as declared fields, and company files shared with all
// employees as text. Sensitive fields are asked for only when a space allow-lists them.
package bamboohr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// Registrar learns secret values so logs can redact them.
type Registrar interface{ Register(secret string) }

// Declared are the fields the connector can return (sb-connector(3), hris: bamboohr).
var Declared = []ports.Field{
	{Name: "displayName"}, {Name: "firstName"}, {Name: "lastName"}, {Name: "jobTitle"}, {Name: "department"},
	{Name: "division"}, {Name: "location"}, {Name: "workEmail"}, {Name: "supervisor"},
	{Name: "payRate", Sensitive: true}, {Name: "payType", Sensitive: true}, {Name: "ssn", Sensitive: true},
	{Name: "dateOfBirth", Sensitive: true}, {Name: "address1", Sensitive: true}, {Name: "homePhone", Sensitive: true},
	{Name: "mobilePhone", Sensitive: true}, {Name: "maritalStatus", Sensitive: true}, {Name: "gender", Sensitive: true},
	{Name: "ethnicity", Sensitive: true}, {Name: "terminationReason", Sensitive: true},
}

// Connector reads one BambooHR company.
type Connector struct {
	Base   string // "" is https://api.bamboohr.com
	HTTP   *http.Client
	Redact Registrar
	// Creds returns vault://tenants/TENANT/connectors/hris: {"key": "…", "subdomain": "…"}.
	Creds func(ctx context.Context) (string, error)
	// Allowed returns the sensitive fields some space's allow-list names; nil is none.
	Allowed func() []string
}

var (
	_ ports.Connector     = (*Connector)(nil)
	_ ports.FieldDeclarer = (*Connector)(nil)
)

// ErrNotSupported is Act on this read-only connector.
var ErrNotSupported = errors.New("ENOTSUP: bamboohr connector is read-only")

// APIVersion is the connector contract version.
func (c *Connector) APIVersion() int { return ports.ConnectorAPIVersion }

// Fields declares what the connector can return.
func (c *Connector) Fields() ([]ports.Field, bool) {
	return append([]ports.Field(nil), Declared...), true
}

// Act is not supported.
func (c *Connector) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, ErrNotSupported
}

type apiError struct{ status int }

func (e *apiError) Error() string { return fmt.Sprintf("bamboohr: HTTP %d", e.status) }

type creds struct{ Key, Subdomain string }

func (c *Connector) creds(ctx context.Context) (creds, error) {
	raw, err := c.Creds(ctx)
	if err != nil {
		return creds{}, fmt.Errorf("bamboohr: credentials: %w", err)
	}
	var k creds
	if err := json.Unmarshal([]byte(raw), &k); err != nil || k.Key == "" || k.Subdomain == "" {
		return creds{}, errors.New("bamboohr: credentials must be {\"key\", \"subdomain\"}")
	}
	if c.Redact != nil {
		c.Redact.Register(k.Key)
	}
	return k, nil
}

func (c *Connector) get(ctx context.Context, k creds, p string, q url.Values, v any) ([]byte, error) {
	base := c.Base
	if base == "" {
		base = "https://api.bamboohr.com"
	}
	u := base + "/api/gateway.php/" + url.PathEscape(k.Subdomain) + "/v1" + p
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(k.Key, "x")
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bamboohr: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return nil, &apiError{resp.StatusCode}
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil || v == nil {
		return b, err
	}
	return b, json.Unmarshal(b, v)
}

func micros(ts string) (int, error) {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return 0, err
	}
	return int(t.UnixMicro()), nil
}

type dirEntry struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	WorkEmail   string `json:"workEmail"`
	Department  string `json:"department"`
	Status      string `json:"status"`
}

func (c *Connector) directory(ctx context.Context, k creds) ([]dirEntry, error) {
	var d struct {
		Employees []dirEntry `json:"employees"`
	}
	_, err := c.get(ctx, k, "/employees/directory", nil, &d)
	return d.Employees, err
}

type file struct {
	ID, Name, Category string
	Shared             bool
	Created            string
}

func (c *Connector) files(ctx context.Context, k creds) (map[string]file, error) {
	var v struct {
		Categories []struct {
			Name  string `json:"name"`
			Files []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Share       string `json:"shareWithEmployee"`
				DateCreated string `json:"dateCreated"`
			} `json:"files"`
		} `json:"categories"`
	}
	if _, err := c.get(ctx, k, "/files/view", nil, &v); err != nil {
		return nil, err
	}
	out := map[string]file{}
	for _, cat := range v.Categories {
		for _, f := range cat.Files {
			out[f.ID] = file{ID: f.ID, Name: f.Name, Category: cat.Name, Shared: f.Share == "yes", Created: f.DateCreated}
		}
	}
	return out, nil
}

// split parses employees/ID or files/ID.
func split(id string) (kind, key string, err error) {
	kind, key, ok := strings.Cut(id, "/")
	if !ok || key == "" || (kind != "employees" && kind != "files") {
		return "", "", fmt.Errorf("bamboohr: bad document id %q", id)
	}
	return kind, key, nil
}

// wanted are the fields asked for: every non-sensitive declared field and the sensitive
// ones a space allow-lists. Other sensitive values are never read.
func (c *Connector) wanted() []string {
	allow := map[string]bool{}
	if c.Allowed != nil {
		for _, f := range c.Allowed() {
			allow[f] = true
		}
	}
	var out []string
	for _, f := range Declared {
		if !f.Sensitive || allow[f.Name] {
			out = append(out, f.Name)
		}
	}
	return out
}

// Fetch reads one employee record or shared company file.
func (c *Connector) Fetch(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
	kind, key, err := split(ref.ID)
	if err != nil {
		return ports.Document{}, err
	}
	k, err := c.creds(ctx)
	if err != nil {
		return ports.Document{}, err
	}
	if kind == "files" {
		fs, err := c.files(ctx, k)
		if err != nil {
			return ports.Document{}, err
		}
		f, ok := fs[key]
		if !ok {
			return ports.Document{}, fmt.Errorf("bamboohr: no file %s", key)
		}
		text, err := c.get(ctx, k, "/files/"+url.PathEscape(key), nil, nil)
		if err != nil {
			return ports.Document{}, err
		}
		v, err := micros(f.Created)
		if err != nil {
			return ports.Document{}, fmt.Errorf("bamboohr: bad dateCreated on file %s", key)
		}
		var pre []string
		if f.Shared {
			pre = []string{AllEmployees}
		}
		return ports.Document{Ref: ref, Version: v, Title: f.Name, Text: string(text), ACLPrefilter: pre,
			Location: "files/" + f.Category + "/" + f.Name}, nil
	}
	var e map[string]string
	if _, err := c.get(ctx, k, "/employees/"+url.PathEscape(key), url.Values{"fields": {strings.Join(c.wanted(), ",")}}, &e); err != nil {
		return ports.Document{}, err
	}
	v, err := micros(e["lastChanged"])
	if err != nil {
		return ports.Document{}, fmt.Errorf("bamboohr: bad lastChanged on employee %s", key)
	}
	fields := map[string]string{}
	for name, val := range e {
		if name != "id" && name != "lastChanged" && name != "status" && val != "" {
			fields[name] = val
		}
	}
	return ports.Document{Ref: ref, Version: v, Title: e["displayName"], Fields: fields, ACLPrefilter: []string{AllEmployees},
		Location: "employees/" + e["department"] + "/" + key}, nil
}

// AllEmployees is the prefilter entry for something every active employee may read.
const AllEmployees = "bamboohr:employees"

// ACL answers live: an active employee may read employee records and shared files.
func (c *Connector) ACL(ctx context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs)) // Unknown until answered (LAW-03)
	if u.Email == "" {
		return out, nil
	}
	k, err := c.creds(ctx)
	if err != nil {
		return out, nil
	}
	dir, err := c.directory(ctx, k)
	if err != nil {
		return out, nil
	}
	active, present := false, map[string]bool{}
	for _, e := range dir {
		present[e.ID] = true
		if strings.EqualFold(e.WorkEmail, u.Email) && e.Status == "Active" {
			active = true
		}
	}
	if !active {
		for i := range out {
			out[i] = ports.VerdictDeny
		}
		return out, nil
	}
	var fs map[string]file
	for i, ref := range refs {
		kind, key, err := split(ref.ID)
		if err != nil {
			continue
		}
		if kind == "employees" {
			out[i] = map[bool]ports.Verdict{true: ports.VerdictAllow, false: ports.VerdictDeny}[present[key]]
			continue
		}
		if fs == nil {
			if fs, err = c.files(ctx, k); err != nil {
				return out, nil // the rest stay Unknown
			}
		}
		out[i] = map[bool]ports.Verdict{true: ports.VerdictAllow, false: ports.VerdictDeny}[fs[key].Shared]
	}
	return out, nil
}

type cursor struct {
	T      string   `json:"t"`
	Shared []string `json:"shared"`
}

// Changes reports changed employees and files after the cursor; "" lists everything.
func (c *Connector) Changes(ctx context.Context, cur ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	k, err := c.creds(ctx)
	if err != nil {
		return nil, cur, err
	}
	var prev cursor
	if cur != "" {
		if err := json.Unmarshal([]byte(cur), &prev); err != nil {
			return nil, cur, errors.New("bamboohr: bad cursor")
		}
	}
	since := time.Unix(0, 0).UTC()
	if prev.T != "" {
		if since, err = time.Parse(time.RFC3339, prev.T); err != nil {
			return nil, cur, errors.New("bamboohr: bad cursor")
		}
	}
	var ch struct {
		Employees map[string]struct {
			ID          string `json:"id"`
			Action      string `json:"action"`
			LastChanged string `json:"lastChanged"`
		} `json:"employees"`
	}
	if _, err := c.get(ctx, k, "/employees/changed", url.Values{"since": {since.Format(time.RFC3339)}}, &ch); err != nil {
		return nil, cur, err
	}
	latest := since
	var set ports.ChangeSet
	ids := make([]string, 0, len(ch.Employees))
	for id := range ch.Employees {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		e := ch.Employees[id]
		t, err := time.Parse(time.RFC3339, e.LastChanged)
		if err != nil || (cur != "" && !t.After(since)) {
			continue
		}
		if t.After(latest) {
			latest = t
		}
		if cur == "" && e.Action == "Deleted" {
			continue
		}
		set = append(set, ports.Change{Ref: ports.SourceRef{Kind: "hris", ID: "employees/" + id}, Deleted: e.Action == "Deleted"})
	}
	fs, err := c.files(ctx, k)
	if err != nil {
		return nil, cur, err
	}
	was := map[string]bool{}
	for _, id := range prev.Shared {
		was[id] = true
	}
	var shared []string
	fids := make([]string, 0, len(fs))
	for id := range fs {
		fids = append(fids, id)
	}
	sort.Strings(fids)
	for _, id := range fids {
		f := fs[id]
		ref := ports.SourceRef{Kind: "hris", ID: "files/" + id}
		t, _ := time.Parse(time.RFC3339, f.Created)
		if t.After(latest) {
			latest = t
		}
		switch {
		case f.Shared && (cur == "" || t.After(since) || !was[id]):
			set = append(set, ports.Change{Ref: ref})
		case !f.Shared && was[id]:
			set = append(set, ports.Change{Ref: ref, ACLOnly: true}) // no longer shared: the live check denies
		}
		if f.Shared {
			shared = append(shared, id)
		}
	}
	for _, id := range prev.Shared {
		if _, ok := fs[id]; !ok {
			set = append(set, ports.Change{Ref: ports.SourceRef{Kind: "hris", ID: "files/" + id}, Deleted: true})
		}
	}
	b, _ := json.Marshal(cursor{T: latest.UTC().Format(time.RFC3339), Shared: shared})
	return set, ports.Cursor(b), nil
}
