// Package qbotest is a recorded fake of the parts of the QuickBooks Online accounting API
// the accounting connector uses (SB-1105): accounts and budgets by query and by id, change
// data capture, and the profit and loss and balance sheet reports. Accounts carry account
// numbers, so a test can check they are never stored.
package qbotest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"time"
)

// Account is a fake account.
type Account struct {
	ID, Name, Type, SubType, Classification, Description string
	Balance                                              float64
	AcctNum, BankNum                                     string // never to be stored
	Updated                                              time.Time
	Deleted                                              bool
}

// Budget is a fake budget: lines of account name and amount for one period.
type Budget struct {
	ID, Name, Start, End string
	Lines                map[string]float64
	Updated              time.Time
	Deleted              bool
}

// Fake is one company.
type Fake struct {
	Server *httptest.Server
	Realm  string
	Token  string

	mu       sync.Mutex
	clock    time.Time
	accounts map[string]*Account
	budgets  map[string]*Budget
	faults   []string
}

// New starts a fake over TLS.
func New() *Fake {
	f := &Fake{Realm: "9130", Token: "qbo_fake_access_0123", clock: time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC),
		accounts: map[string]*Account{}, budgets: map[string]*Budget{}}
	mux := http.NewServeMux()
	p := "/v3/company/{realm}"
	mux.HandleFunc("GET "+p+"/query", f.auth(f.query))
	mux.HandleFunc("GET "+p+"/account/{id}", f.auth(f.account))
	mux.HandleFunc("GET "+p+"/budget/{id}", f.auth(f.budget))
	mux.HandleFunc("GET "+p+"/cdc", f.auth(f.cdc))
	mux.HandleFunc("GET "+p+"/reports/{name}", f.auth(f.report))
	f.Server = httptest.NewTLSServer(mux)
	return f
}

// Base is the API's base URL.
func (f *Fake) Base() string { return f.Server.URL }

// CredsJSON is what the vault holds for the connector, with these readers.
func (f *Fake) CredsJSON(readers ...string) string {
	b, _ := json.Marshal(map[string]any{"access_token": f.Token, "realm_id": f.Realm, "readers": readers})
	return string(b)
}

func (f *Fake) tick() time.Time { f.clock = f.clock.Add(time.Second); return f.clock }

// PutAccount creates or edits an account.
func (f *Fake) PutAccount(a Account) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a.Updated = f.tick()
	f.accounts[a.ID] = &a
}

// DeleteAccount makes an account inactive (QuickBooks's delete).
func (f *Fake) DeleteAccount(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if a, ok := f.accounts[id]; ok {
		a.Deleted, a.Updated = true, f.tick()
	}
}

// PutBudget creates or edits a budget.
func (f *Fake) PutBudget(b Budget) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b.Updated = f.tick()
	f.budgets[b.ID] = &b
}

// FailNext makes the next call of any kind fail: "500", "timeout" or "404".
func (f *Fake) FailNext(kind string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.faults = append(f.faults, kind)
}

func (f *Fake) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+f.Token || r.PathValue("realm") != f.Realm {
			http.Error(w, `{"fault":"AuthenticationFailed"}`, http.StatusUnauthorized)
			return
		}
		f.mu.Lock()
		var fault string
		if len(f.faults) > 0 {
			fault, f.faults = f.faults[0], f.faults[1:]
		}
		f.mu.Unlock()
		switch fault {
		case "500":
			http.Error(w, "error", http.StatusInternalServerError)
			return
		case "timeout":
			time.Sleep(time.Second)
			http.Error(w, "error", http.StatusBadGateway)
			return
		case "404":
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		next(w, r)
	}
}

func write(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func meta(t time.Time) map[string]string {
	return map[string]string{"LastUpdatedTime": t.Format(time.RFC3339)}
}

func accountJSON(a *Account) map[string]any {
	m := map[string]any{"Id": a.ID, "Name": a.Name, "AccountType": a.Type, "AccountSubType": a.SubType,
		"Classification": a.Classification, "Description": a.Description, "CurrentBalance": a.Balance,
		"AcctNum": a.AcctNum, "BankNum": a.BankNum, "Active": !a.Deleted, "MetaData": meta(a.Updated)}
	if a.Deleted {
		m["status"] = "Deleted"
	}
	return m
}

func budgetJSON(b *Budget) map[string]any {
	var lines []map[string]any
	names := make([]string, 0, len(b.Lines))
	for n := range b.Lines {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		lines = append(lines, map[string]any{"AccountRef": map[string]string{"name": n}, "Amount": b.Lines[n]})
	}
	m := map[string]any{"Id": b.ID, "Name": b.Name, "StartDate": b.Start, "EndDate": b.End, "BudgetDetail": lines, "MetaData": meta(b.Updated)}
	if b.Deleted {
		m["status"] = "Deleted"
	}
	return m
}

func (f *Fake) query(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	q := strings.ToLower(r.URL.Query().Get("query"))
	switch {
	case strings.Contains(q, "from account"):
		var out []map[string]any
		for _, id := range sortedKeys(f.accounts) {
			if !f.accounts[id].Deleted {
				out = append(out, accountJSON(f.accounts[id]))
			}
		}
		write(w, map[string]any{"QueryResponse": map[string]any{"Account": out}})
	case strings.Contains(q, "from budget"):
		var out []map[string]any
		for _, id := range sortedKeys(f.budgets) {
			if !f.budgets[id].Deleted {
				out = append(out, budgetJSON(f.budgets[id]))
			}
		}
		write(w, map[string]any{"QueryResponse": map[string]any{"Budget": out}})
	default:
		http.Error(w, "unsupported query", http.StatusBadRequest)
	}
}

func (f *Fake) account(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.accounts[r.PathValue("id")]
	if !ok || a.Deleted {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	write(w, map[string]any{"Account": accountJSON(a)})
}

func (f *Fake) budget(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.budgets[r.PathValue("id")]
	if !ok || b.Deleted {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	write(w, map[string]any{"Budget": budgetJSON(b)})
}

func (f *Fake) cdc(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	since, _ := time.Parse(time.RFC3339, r.URL.Query().Get("changedSince"))
	var accts, buds []map[string]any
	for _, id := range sortedKeys(f.accounts) {
		if a := f.accounts[id]; !a.Updated.Before(since) {
			accts = append(accts, accountJSON(a))
		}
	}
	for _, id := range sortedKeys(f.budgets) {
		if b := f.budgets[id]; !b.Updated.Before(since) {
			buds = append(buds, budgetJSON(b))
		}
	}
	write(w, map[string]any{"CDCResponse": []any{map[string]any{"QueryResponse": []any{
		map[string]any{"Account": accts}, map[string]any{"Budget": buds}}}}})
}

// report answers ProfitAndLoss and BalanceSheet with one row per active account of the
// classifications each report shows; the amounts are the balances (the fake has no ledger).
func (f *Fake) report(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	name := r.PathValue("name")
	show := map[string]bool{"Revenue": name == "ProfitAndLoss", "Expense": name == "ProfitAndLoss",
		"Asset": name == "BalanceSheet", "Liability": name == "BalanceSheet", "Equity": name == "BalanceSheet"}
	if name != "ProfitAndLoss" && name != "BalanceSheet" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var rows []map[string]any
	for _, id := range sortedKeys(f.accounts) {
		a := f.accounts[id]
		if a.Deleted || !show[a.Classification] {
			continue
		}
		rows = append(rows, map[string]any{"ColData": []map[string]string{{"value": a.Name}, {"value": formatAmount(a.Balance)}}})
	}
	write(w, map[string]any{"Header": map[string]string{"ReportName": name, "StartPeriod": r.URL.Query().Get("start_date"),
		"EndPeriod": r.URL.Query().Get("end_date"), "Time": f.clock.Format(time.RFC3339)}, "Rows": map[string]any{"Row": rows}})
}

func formatAmount(x float64) string {
	b, _ := json.Marshal(x)
	return string(b)
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
