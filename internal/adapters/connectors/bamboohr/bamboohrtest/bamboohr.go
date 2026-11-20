// Package bamboohrtest is a recorded fake of the parts of the BambooHR API the HRIS
// connector uses (SB-1104): the employee directory, employee fields, changed employees,
// and company files. It records which fields were asked for, so a test can check that
// sensitive values are never even read unless allow-listed.
package bamboohrtest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"time"
)

// Employee is a fake employee record.
type Employee struct {
	ID      string
	Fields  map[string]string // every field, sensitive ones included
	Status  string            // "Active" or "Inactive"
	Changed time.Time
	Deleted bool
}

// File is a fake company file.
type File struct {
	ID, Name, Category, Text string
	Shared                   bool // shared with all employees
	Created                  time.Time
	Gone                     bool
}

// Fake is one company.
type Fake struct {
	Server    *httptest.Server
	Subdomain string
	Key       string

	mu     sync.Mutex
	clock  time.Time
	emps   map[string]*Employee
	files  map[string]*File
	faults []string
	// Asked records every field name asked for on employee reads.
	Asked map[string]int
}

// New starts a fake over TLS.
func New() *Fake {
	f := &Fake{Subdomain: "acme", Key: "bhr_fake_key_0123", clock: time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC),
		emps: map[string]*Employee{}, files: map[string]*File{}, Asked: map[string]int{}}
	mux := http.NewServeMux()
	p := "/api/gateway.php/{sub}/v1"
	mux.HandleFunc("GET "+p+"/employees/directory", f.auth(f.directory))
	mux.HandleFunc("GET "+p+"/employees/changed", f.auth(f.changed))
	mux.HandleFunc("GET "+p+"/employees/{id}", f.auth(f.employee))
	mux.HandleFunc("GET "+p+"/files/view", f.auth(f.listFiles))
	mux.HandleFunc("GET "+p+"/files/{id}", f.auth(f.getFile))
	f.Server = httptest.NewTLSServer(mux)
	return f
}

// Base is the API's base URL.
func (f *Fake) Base() string { return f.Server.URL }

// CredsJSON is what the vault holds for the connector.
func (f *Fake) CredsJSON() string { return `{"key":"` + f.Key + `","subdomain":"` + f.Subdomain + `"}` }

func (f *Fake) tick() time.Time { f.clock = f.clock.Add(time.Second); return f.clock }

// PutEmployee creates or edits an employee.
func (f *Fake) PutEmployee(id string, fields map[string]string, active bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st := "Inactive"
	if active {
		st = "Active"
	}
	cp := map[string]string{}
	for k, v := range fields {
		cp[k] = v
	}
	f.emps[id] = &Employee{ID: id, Fields: cp, Status: st, Changed: f.tick()}
}

// DeleteEmployee removes an employee.
func (f *Fake) DeleteEmployee(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e, ok := f.emps[id]; ok {
		e.Deleted, e.Changed = true, f.tick()
	}
}

// PutFile adds a company file.
func (f *Fake) PutFile(id, name, category, text string, shared bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.files[id] = &File{ID: id, Name: name, Category: category, Text: text, Shared: shared, Created: f.tick()}
}

// Unshare stops sharing a file with all employees.
func (f *Fake) Unshare(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.files[id].Shared = false
}

// FailNext makes the next directory read fail: "500", "timeout" or "404".
func (f *Fake) FailNext(kind string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.faults = append(f.faults, kind)
}

func (f *Fake) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _, ok := r.BasicAuth()
		if !ok || user != f.Key {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.PathValue("sub") != f.Subdomain {
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

func (f *Fake) directory(w http.ResponseWriter, _ *http.Request) {
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
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := make([]string, 0, len(f.emps))
	for id := range f.emps {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var out []map[string]string
	for _, id := range ids {
		e := f.emps[id]
		if e.Deleted {
			continue
		}
		out = append(out, map[string]string{"id": id, "displayName": e.Fields["displayName"], "workEmail": e.Fields["workEmail"],
			"department": e.Fields["department"], "status": e.Status})
	}
	write(w, map[string]any{"employees": out})
}

func (f *Fake) employee(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.emps[r.PathValue("id")]
	if !ok || e.Deleted {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	out := map[string]string{"id": e.ID, "lastChanged": e.Changed.Format(time.RFC3339), "status": e.Status}
	for _, name := range strings.Split(r.URL.Query().Get("fields"), ",") {
		if name == "" {
			continue
		}
		f.Asked[name]++
		if v, ok := e.Fields[name]; ok {
			out[name] = v
		}
	}
	write(w, out)
}

func (f *Fake) changed(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	since, _ := time.Parse(time.RFC3339, r.URL.Query().Get("since"))
	out := map[string]any{}
	for id, e := range f.emps {
		if !e.Changed.Before(since) {
			action := "Updated"
			if e.Deleted {
				action = "Deleted"
			}
			out[id] = map[string]string{"id": id, "action": action, "lastChanged": e.Changed.Format(time.RFC3339)}
		}
	}
	write(w, map[string]any{"employees": out})
}

func (f *Fake) listFiles(w http.ResponseWriter, _ *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	byCat := map[string][]map[string]any{}
	for _, x := range f.files {
		if x.Gone {
			continue
		}
		byCat[x.Category] = append(byCat[x.Category], map[string]any{"id": x.ID, "name": x.Name,
			"shareWithEmployee": map[bool]string{true: "yes", false: "no"}[x.Shared], "dateCreated": x.Created.Format(time.RFC3339)})
	}
	cats := make([]string, 0, len(byCat))
	for c := range byCat {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	var out []map[string]any
	for _, c := range cats {
		fs := byCat[c]
		sort.Slice(fs, func(i, j int) bool { return fs[i]["id"].(string) < fs[j]["id"].(string) })
		out = append(out, map[string]any{"name": c, "files": fs})
	}
	write(w, map[string]any{"categories": out})
}

func (f *Fake) getFile(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	x, ok := f.files[r.PathValue("id")]
	if !ok || x.Gone {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	_, _ = w.Write([]byte(x.Text))
}
