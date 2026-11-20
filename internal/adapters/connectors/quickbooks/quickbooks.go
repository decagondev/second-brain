// Package quickbooks is the read-only accounting connector (sb-connector(3), accounting:
// quickbooks-online; SB-1105): the chart of accounts as declared fields, budgets, and the
// profit and loss and balance sheet for the last 12 months. Account and routing numbers,
// card details, tax ids and payroll are never read.
package quickbooks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// Declared are an account's fields; none is sensitive, and nothing sensitive is read.
var Declared = []ports.Field{{Name: "Name"}, {Name: "AccountType"}, {Name: "AccountSubType"}, {Name: "Classification"},
	{Name: "CurrentBalance"}, {Name: "Description"}}

// Reports are the reports indexed, for each of the last Months calendar months.
var Reports = []string{"ProfitAndLoss", "BalanceSheet"}

// Months is how many calendar months of reports are indexed.
const Months = 12

// Connector reads one QuickBooks Online company.
type Connector struct {
	Base   string // "" is https://quickbooks.api.intuit.com
	HTTP   *http.Client
	Redact Registrar
	// Creds returns vault://tenants/TENANT/connectors/accounting:
	// {"access_token": "…", "realm_id": "…", "readers": ["…"]}.
	Creds func(ctx context.Context) (string, error)
	Now   func() time.Time
}

var (
	_ ports.Connector     = (*Connector)(nil)
	_ ports.FieldDeclarer = (*Connector)(nil)
)

// ErrNotSupported is Act on this read-only connector.
var ErrNotSupported = errors.New("ENOTSUP: quickbooks-online connector is read-only")

// APIVersion is the connector contract version.
func (c *Connector) APIVersion() int { return ports.ConnectorAPIVersion }

// Fields declares an account's fields.
func (c *Connector) Fields() ([]ports.Field, bool) {
	return append([]ports.Field(nil), Declared...), true
}

// Act is not supported.
func (c *Connector) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, ErrNotSupported
}

type creds struct {
	AccessToken string    `json:"access_token"`
	RealmID     string    `json:"realm_id"`
	Readers     *[]string `json:"readers"`
}

func (c *Connector) creds(ctx context.Context) (creds, error) {
	raw, err := c.Creds(ctx)
	if err != nil {
		return creds{}, fmt.Errorf("quickbooks: credentials: %w", err)
	}
	var k creds
	if err := json.Unmarshal([]byte(raw), &k); err != nil || k.AccessToken == "" || k.RealmID == "" {
		return creds{}, errors.New("quickbooks: credentials must be {\"access_token\", \"realm_id\", \"readers\"}")
	}
	if c.Redact != nil {
		c.Redact.Register(k.AccessToken)
	}
	return k, nil
}

func (c *Connector) get(ctx context.Context, k creds, p string, q url.Values, v any) error {
	base := c.Base
	if base == "" {
		base = "https://quickbooks.api.intuit.com"
	}
	u := base + "/v3/company/" + url.PathEscape(k.RealmID) + p
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+k.AccessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("quickbooks: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("quickbooks: HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(v)
}

// account holds only what may be read; AcctNum, BankNum and the rest are never decoded.
type account struct {
	ID             string      `json:"Id"`
	Name           string      `json:"Name"`
	AccountType    string      `json:"AccountType"`
	AccountSubType string      `json:"AccountSubType"`
	Classification string      `json:"Classification"`
	Description    string      `json:"Description"`
	CurrentBalance json.Number `json:"CurrentBalance"`
	Status         string      `json:"status"`
	MetaData       struct{ LastUpdatedTime string }
}

type budget struct {
	ID           string `json:"Id"`
	Name         string `json:"Name"`
	StartDate    string `json:"StartDate"`
	EndDate      string `json:"EndDate"`
	BudgetDetail []struct {
		AccountRef struct {
			Name string `json:"name"`
		} `json:"AccountRef"`
		Amount json.Number `json:"Amount"`
	} `json:"BudgetDetail"`
	Status   string `json:"status"`
	MetaData struct{ LastUpdatedTime string }
}

func micros(ts string) (int, error) {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return 0, err
	}
	return int(t.UnixMicro()), nil
}

func (c *Connector) now() time.Time {
	if c.Now == nil {
		return time.Now().UTC()
	}
	return c.Now().UTC()
}

// months are the last Months calendar months, newest first, as YYYY-MM.
func (c *Connector) months() []string {
	n := c.now()
	first := time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, time.UTC)
	out := make([]string, Months)
	for i := range out {
		out[i] = first.AddDate(0, -i, 0).Format("2006-01")
	}
	return out
}

// queryPage is the page size of a QuickBooks query; the API's default of 100 rows would
// silently drop the rest (SB-1401).
const queryPage = 1000

func (c *Connector) accounts(ctx context.Context, k creds) ([]account, error) {
	var all []account
	for start := 1; ; start += queryPage {
		var v struct {
			QueryResponse struct {
				Account []account `json:"Account"`
			} `json:"QueryResponse"`
		}
		q := fmt.Sprintf("select * from Account startposition %d maxresults %d", start, queryPage)
		if err := c.get(ctx, k, "/query", url.Values{"query": {q}}, &v); err != nil {
			return nil, err
		}
		all = append(all, v.QueryResponse.Account...)
		if len(v.QueryResponse.Account) < queryPage {
			return all, nil
		}
	}
}

func (c *Connector) budgets(ctx context.Context, k creds) ([]budget, error) {
	var all []budget
	for start := 1; ; start += queryPage {
		var v struct {
			QueryResponse struct {
				Budget []budget `json:"Budget"`
			} `json:"QueryResponse"`
		}
		q := fmt.Sprintf("select * from Budget startposition %d maxresults %d", start, queryPage)
		if err := c.get(ctx, k, "/query", url.Values{"query": {q}}, &v); err != nil {
			return nil, err
		}
		all = append(all, v.QueryResponse.Budget...)
		if len(v.QueryResponse.Budget) < queryPage {
			return all, nil
		}
	}
}

// Fetch reads one account, budget or monthly report.
func (c *Connector) Fetch(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
	k, err := c.creds(ctx)
	if err != nil {
		return ports.Document{}, err
	}
	parts := strings.Split(ref.ID, "/")
	switch {
	case len(parts) == 2 && parts[0] == "accounts":
		var v struct {
			Account account `json:"Account"`
		}
		if err := c.get(ctx, k, "/account/"+url.PathEscape(parts[1]), nil, &v); err != nil {
			return ports.Document{}, err
		}
		a := v.Account
		ver, err := micros(a.MetaData.LastUpdatedTime)
		if err != nil {
			return ports.Document{}, fmt.Errorf("quickbooks: bad LastUpdatedTime on account %s", a.ID)
		}
		fields := map[string]string{"Name": a.Name, "AccountType": a.AccountType, "AccountSubType": a.AccountSubType,
			"Classification": a.Classification, "CurrentBalance": a.CurrentBalance.String(), "Description": a.Description}
		for f, val := range fields {
			if val == "" {
				delete(fields, f)
			}
		}
		return ports.Document{Ref: ref, Version: ver, Title: a.Name, Fields: fields, ACLPrefilter: []string{Readers},
			Location: "accounts/" + a.Classification + "/" + a.Name}, nil
	case len(parts) == 2 && parts[0] == "budgets":
		var v struct {
			Budget budget `json:"Budget"`
		}
		if err := c.get(ctx, k, "/budget/"+url.PathEscape(parts[1]), nil, &v); err != nil {
			return ports.Document{}, err
		}
		b := v.Budget
		ver, err := micros(b.MetaData.LastUpdatedTime)
		if err != nil {
			return ports.Document{}, fmt.Errorf("quickbooks: bad LastUpdatedTime on budget %s", b.ID)
		}
		lines := []string{"Period: " + b.StartDate + " to " + b.EndDate}
		for _, l := range b.BudgetDetail {
			lines = append(lines, l.AccountRef.Name+": "+l.Amount.String())
		}
		return ports.Document{Ref: ref, Version: ver, Title: b.Name, Text: strings.Join(lines, "\n"), ACLPrefilter: []string{Readers},
			Location: "budgets/" + b.Name}, nil
	case len(parts) == 3 && parts[0] == "reports" && contains(Reports, parts[1]) && contains(c.months(), parts[2]):
		return c.report(ctx, k, ref, parts[1], parts[2])
	}
	return ports.Document{}, fmt.Errorf("quickbooks: bad document id %q", ref.ID)
}

func (c *Connector) report(ctx context.Context, k creds, ref ports.SourceRef, name, month string) (ports.Document, error) {
	start, _ := time.Parse("2006-01", month)
	end := start.AddDate(0, 1, -1)
	var v struct {
		Rows struct {
			Row []struct {
				ColData []struct {
					Value string `json:"value"`
				} `json:"ColData"`
			} `json:"Row"`
		} `json:"Rows"`
	}
	if err := c.get(ctx, k, "/reports/"+name, url.Values{"start_date": {start.Format("2006-01-02")}, "end_date": {end.Format("2006-01-02")}}, &v); err != nil {
		return ports.Document{}, err
	}
	var lines []string
	for _, r := range v.Rows.Row {
		if len(r.ColData) >= 2 {
			lines = append(lines, r.ColData[0].Value+": "+r.ColData[1].Value)
		}
	}
	accts, err := c.accounts(ctx, k)
	if err != nil {
		return ports.Document{}, err
	}
	ver := 0
	for _, a := range accts {
		if m, err := micros(a.MetaData.LastUpdatedTime); err == nil && m > ver {
			ver = m
		}
	}
	title := map[string]string{"ProfitAndLoss": "Profit and loss", "BalanceSheet": "Balance sheet"}[name] + ", " + month
	return ports.Document{Ref: ref, Version: ver, Title: title, Text: strings.Join(lines, "\n"), ACLPrefilter: []string{Readers},
		Location: "reports/" + name + "/" + month}, nil
}

// Readers is the prefilter entry for the company's listed readers.
const Readers = "quickbooks:readers"

// ACL answers from the tenant's list of QuickBooks readers, for documents QuickBooks still
// holds. Anything QuickBooks can't answer is Unknown (LAW-03).
func (c *Connector) ACL(ctx context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs))
	k, err := c.creds(ctx)
	if err != nil || k.Readers == nil || u.Email == "" {
		return out, nil
	}
	reader := false
	for _, r := range *k.Readers {
		reader = reader || strings.EqualFold(r, u.Email)
	}
	if !reader {
		for i := range out {
			out[i] = ports.VerdictDeny
		}
		return out, nil
	}
	exists := map[string]bool{}
	var loadedA, loadedB bool
	for i, ref := range refs {
		parts := strings.Split(ref.ID, "/")
		switch {
		case len(parts) == 2 && parts[0] == "accounts":
			if !loadedA {
				as, err := c.accounts(ctx, k)
				if err != nil {
					return out, nil // the rest stay Unknown
				}
				for _, a := range as {
					exists["accounts/"+a.ID] = true
				}
				loadedA = true
			}
		case len(parts) == 2 && parts[0] == "budgets":
			if !loadedB {
				bs, err := c.budgets(ctx, k)
				if err != nil {
					return out, nil
				}
				for _, b := range bs {
					exists["budgets/"+b.ID] = true
				}
				loadedB = true
			}
		case len(parts) == 3 && parts[0] == "reports" && contains(Reports, parts[1]) && contains(c.months(), parts[2]):
			exists[ref.ID] = true
		default:
			continue // an id QuickBooks can't place stays Unknown
		}
		out[i] = map[bool]ports.Verdict{true: ports.VerdictAllow, false: ports.VerdictDeny}[exists[ref.ID]]
	}
	return out, nil
}

type cursor struct {
	T       string `json:"t"`
	Readers string `json:"readers"`
}

func readersHash(k creds) string {
	if k.Readers == nil {
		return ""
	}
	rs := append([]string(nil), *k.Readers...)
	for i := range rs {
		rs[i] = strings.ToLower(rs[i])
	}
	sort.Strings(rs)
	h := sha256.Sum256([]byte(strings.Join(rs, "\n")))
	return hex.EncodeToString(h[:8])
}

// Changes reports changed accounts, budgets and reports after the cursor; "" lists all.
func (c *Connector) Changes(ctx context.Context, cur ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	k, err := c.creds(ctx)
	if err != nil {
		return nil, cur, err
	}
	var prev cursor
	if cur != "" {
		if err := json.Unmarshal([]byte(cur), &prev); err != nil {
			return nil, cur, errors.New("quickbooks: bad cursor")
		}
	}
	since := time.Unix(0, 0).UTC()
	if prev.T != "" {
		if since, err = time.Parse(time.RFC3339, prev.T); err != nil {
			return nil, cur, errors.New("quickbooks: bad cursor")
		}
	}
	var v struct {
		CDCResponse []struct {
			QueryResponse []struct {
				Account []account `json:"Account"`
				Budget  []budget  `json:"Budget"`
			} `json:"QueryResponse"`
		} `json:"CDCResponse"`
	}
	if err := c.get(ctx, k, "/cdc", url.Values{"entities": {"Account,Budget"}, "changedSince": {since.Format(time.RFC3339)}}, &v); err != nil {
		return nil, cur, err
	}
	latest := since
	var set ports.ChangeSet
	accountChanged := false
	after := func(ts string) bool {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return false
		}
		if t.After(latest) {
			latest = t
		}
		return cur == "" || t.After(since)
	}
	for _, r := range v.CDCResponse {
		for _, q := range r.QueryResponse {
			for _, a := range q.Account {
				if !after(a.MetaData.LastUpdatedTime) || (cur == "" && a.Status == "Deleted") {
					continue
				}
				accountChanged = true
				set = append(set, ports.Change{Ref: ports.SourceRef{Kind: "accounting", ID: "accounts/" + a.ID}, Deleted: a.Status == "Deleted"})
			}
			for _, b := range q.Budget {
				if !after(b.MetaData.LastUpdatedTime) || (cur == "" && b.Status == "Deleted") {
					continue
				}
				set = append(set, ports.Change{Ref: ports.SourceRef{Kind: "accounting", ID: "budgets/" + b.ID}, Deleted: b.Status == "Deleted"})
			}
		}
	}
	if accountChanged || cur == "" {
		for _, name := range Reports {
			for _, m := range c.months() {
				set = append(set, ports.Change{Ref: ports.SourceRef{Kind: "accounting", ID: "reports/" + name + "/" + m}})
			}
		}
	}
	h := readersHash(k)
	if cur != "" && h != prev.Readers {
		changed := map[string]bool{}
		for _, ch := range set {
			changed[ch.Ref.ID] = true
		}
		ids, err := c.allIDs(ctx, k)
		if err != nil {
			return nil, cur, err
		}
		for _, id := range ids {
			if !changed[id] {
				set = append(set, ports.Change{Ref: ports.SourceRef{Kind: "accounting", ID: id}, ACLOnly: true})
			}
		}
	}
	b, _ := json.Marshal(cursor{T: latest.UTC().Format(time.RFC3339), Readers: h})
	return set, ports.Cursor(b), nil
}

func (c *Connector) allIDs(ctx context.Context, k creds) ([]string, error) {
	as, err := c.accounts(ctx, k)
	if err != nil {
		return nil, err
	}
	bs, err := c.budgets(ctx, k)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, a := range as {
		out = append(out, "accounts/"+a.ID)
	}
	for _, b := range bs {
		out = append(out, "budgets/"+b.ID)
	}
	for _, name := range Reports {
		for _, m := range c.months() {
			out = append(out, "reports/"+name+"/"+m)
		}
	}
	return out, nil
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}
