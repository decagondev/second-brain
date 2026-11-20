// Package drivetest is a recorded fake of the parts of the Google Drive v3 API the
// connector uses, plus Google's OAuth token endpoint for service accounts. It verifies
// the JWT assertion's signature, honours domain-wide delegation (sub), and enforces
// per-user visibility, so the contract kit exercises the real connector code.
package drivetest

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Fake is one Workspace's Drive.
type Fake struct {
	Server *httptest.Server
	Key    *rsa.PrivateKey
	Email  string // the service account

	mu      sync.Mutex
	files   map[string]*File
	log     []change
	clock   time.Time
	tokens  map[string]string // access token -> subject ("" = service account)
	faults  []string          // next user (impersonated) file calls: "500", "timeout", "404"
	saMiss  int               // next service-account file calls that answer 404 (follows a "404" fault)
	limited int               // next N calls of any kind answer 429
	folders map[string]*File  // folders, served by files.get only (SB-1005)
}

// File is a fake Drive file.
type File struct {
	ID, Name, MimeType, Text string
	Version                  int
	Modified                 time.Time
	Trashed                  bool
	Viewers                  []string // user emails who may open it
	Parents                  []string // parent folder ids (SB-1005)
	// Labels are Drive labels: label id -> field id -> selected choice id (SB-1005).
	Labels map[string]map[string]string
	// Owner is the file's owner email; "" for a shared drive's file, which has none (SB-1203).
	Owner string
}

// Folder creates a folder, under parent ("" for the top of the drive).
func (f *Fake) Folder(id, name, parent string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.folders == nil {
		f.folders = map[string]*File{}
	}
	x := &File{ID: id, Name: name, MimeType: "application/vnd.google-apps.folder", Version: 1}
	if parent != "" {
		x.Parents = []string{parent}
	}
	f.folders[id] = x
}

// Move puts a file in a folder: a change, as in Drive.
func (f *Fake) Move(id, parent string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if x, ok := f.files[id]; ok {
		x.Parents, x.Modified = []string{parent}, f.tick()
		x.Version++
		f.log = append(f.log, change{id: id, at: x.Modified})
	}
}

// Classify sets a Drive label field's choice on a file ("" removes the label): a change.
func (f *Fake) Classify(id, label, field, choice string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	x, ok := f.files[id]
	if !ok {
		return
	}
	if x.Labels == nil {
		x.Labels = map[string]map[string]string{}
	}
	if choice == "" {
		delete(x.Labels, label)
	} else {
		x.Labels[label] = map[string]string{field: choice}
	}
	x.Modified = f.tick()
	x.Version++
	f.log = append(f.log, change{id: id, at: x.Modified})
}

type change struct {
	id   string
	at   time.Time
	gone bool
}

// New starts a fake over TLS.
func New() *Fake {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	f := &Fake{Key: k, Email: "sb-docs@acme-sb.iam.gserviceaccount.com", files: map[string]*File{}, tokens: map[string]string{},
		clock: time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC)}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", f.token)
	mux.HandleFunc("GET /drive/v3/changes/startPageToken", f.auth(f.startToken))
	mux.HandleFunc("GET /drive/v3/changes", f.auth(f.changes))
	mux.HandleFunc("GET /drive/v3/files", f.auth(f.list))
	mux.HandleFunc("GET /drive/v3/files/{id}", f.auth(f.get))
	mux.HandleFunc("GET /drive/v3/files/{id}/export", f.auth(f.export))
	f.Server = httptest.NewTLSServer(mux)
	return f
}

// Base is the API base URL for the connector.
func (f *Fake) Base() string { return f.Server.URL + "/drive/v3" }

// KeyJSON is a service account key file pointing at this fake's token endpoint.
func (f *Fake) KeyJSON() string {
	der, _ := x509.MarshalPKCS8PrivateKey(f.Key)
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	b, _ := json.Marshal(map[string]string{"type": "service_account", "client_email": f.Email,
		"private_key": string(pemKey), "token_uri": f.Server.URL + "/token"})
	return string(b)
}

func (f *Fake) tick() time.Time { f.clock = f.clock.Add(10 * time.Second); return f.clock }

// Put creates or edits a file (a content change).
func (f *Fake) Put(id, name, mime, text string, viewers ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.tick()
	x, ok := f.files[id]
	if !ok {
		x = &File{ID: id}
		f.files[id] = x
	}
	x.Name, x.MimeType, x.Text, x.Trashed, x.Modified = name, mime, text, false, now
	x.Version++
	if viewers != nil {
		x.Viewers = viewers
	}
	f.log = append(f.log, change{id: id, at: now})
}

// Own sets a file's owner at the source (SB-1203); "" makes it a shared drive's file.
func (f *Fake) Own(id, owner string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if x, ok := f.files[id]; ok {
		x.Owner = owner
	}
}

// Share changes who may open a file (a permission-only change).
func (f *Fake) Share(id string, viewers ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.tick()
	if x, ok := f.files[id]; ok {
		x.Viewers = viewers
		x.Version++ // Drive bumps version on any change
		f.log = append(f.log, change{id: id, at: now})
	}
}

// Trash moves a file to the trash.
func (f *Fake) Trash(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.tick()
	if x, ok := f.files[id]; ok {
		x.Trashed = true
		f.log = append(f.log, change{id: id, at: now, gone: true})
	}
}

// FailNext makes the next impersonated file calls fail: "500", "timeout" or "404".
func (f *Fake) FailNext(kinds ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.faults = append(f.faults, kinds...)
}

// RateLimitNext makes the next n calls of any kind answer 429.
func (f *Fake) RateLimitNext(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.limited += n
}

func fail(w http.ResponseWriter, status int, reason string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": status, "errors": []map[string]string{{"reason": reason}}}})
}

// token exchanges a signed JWT assertion for an access token bound to its sub.
func (f *Fake) token(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.PostFormValue("assertion"), ".")
	if r.PostFormValue("grant_type") != "urn:ietf:params:oauth:grant-type:jwt-bearer" || len(parts) != 3 {
		fail(w, 400, "invalid_grant")
		return
	}
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(&f.Key.PublicKey, crypto.SHA256, sum[:], sig) != nil {
		fail(w, 400, "invalid_grant")
		return
	}
	var c struct{ Iss, Sub, Scope, Aud string }
	p, _ := base64.RawURLEncoding.DecodeString(parts[1])
	_ = json.Unmarshal(p, &c)
	if c.Iss != f.Email || c.Scope != "https://www.googleapis.com/auth/drive.readonly" || c.Aud != f.Server.URL+"/token" {
		fail(w, 400, "invalid_scope")
		return
	}
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	tok := "ya29." + base64.RawURLEncoding.EncodeToString(b)
	f.mu.Lock()
	f.tokens[tok] = c.Sub
	f.mu.Unlock()
	_ = json.NewEncoder(w).Encode(map[string]any{"access_token": tok, "expires_in": 3600, "token_type": "Bearer"})
}

func (f *Fake) auth(next func(w http.ResponseWriter, r *http.Request, subject string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		f.mu.Lock()
		subject, ok := f.tokens[tok]
		limited := f.limited > 0
		if limited {
			f.limited--
		}
		f.mu.Unlock()
		switch {
		case !ok:
			fail(w, 401, "authError")
		case limited:
			fail(w, 429, "rateLimitExceeded")
		default:
			next(w, r, subject)
		}
	}
}

func (f *Fake) fileJSON(x *File) map[string]any {
	var perms []map[string]string
	for _, v := range x.Viewers {
		perms = append(perms, map[string]string{"type": "user", "emailAddress": v})
	}
	out := map[string]any{"id": x.ID, "name": x.Name, "mimeType": x.MimeType, "version": strconv.Itoa(x.Version),
		"modifiedTime": x.Modified.Format(time.RFC3339Nano), "trashed": x.Trashed, "permissions": perms}
	if len(x.Parents) > 0 {
		out["parents"] = x.Parents
	}
	if x.Owner != "" {
		out["owners"] = []map[string]string{{"emailAddress": x.Owner}}
	}
	return out
}

// withLabels adds labelInfo for the labels a files.get asked for with includeLabels.
func withLabels(m map[string]any, x *File, include string) map[string]any {
	var ls []map[string]any
	for _, id := range strings.Split(include, ",") {
		if fs, ok := x.Labels[id]; ok && id != "" {
			fields := map[string]any{}
			for fid, choice := range fs {
				fields[fid] = map[string]any{"id": fid, "selection": []string{choice}}
			}
			ls = append(ls, map[string]any{"id": id, "fields": fields})
		}
	}
	if ls != nil {
		m["labelInfo"] = map[string]any{"labels": ls}
	}
	return m
}

func (f *Fake) visible(x *File, subject string) bool {
	if subject == "" {
		return true // the service account sees the whole domain
	}
	for _, v := range x.Viewers {
		if v == subject {
			return true
		}
	}
	return false
}

func (f *Fake) get(w http.ResponseWriter, r *http.Request, subject string) {
	f.mu.Lock()
	var fault string
	if subject != "" && len(f.faults) > 0 {
		fault, f.faults = f.faults[0], f.faults[1:]
	}
	x, ok := f.files[r.PathValue("id")]
	if !ok {
		x, ok = f.folders[r.PathValue("id")]
	}
	f.mu.Unlock()
	switch fault {
	case "500":
		fail(w, 500, "backendError")
		return
	case "timeout":
		time.Sleep(time.Second)
		fail(w, 503, "backendError")
		return
	case "404":
		f.mu.Lock()
		f.saMiss++ // the service account's follow-up check fails too: the source can't answer
		f.mu.Unlock()
		fail(w, 404, "notFound")
		return
	}
	if subject == "" {
		f.mu.Lock()
		miss := f.saMiss > 0
		if miss {
			f.saMiss--
		}
		f.mu.Unlock()
		if miss {
			fail(w, 404, "notFound")
			return
		}
	}
	if !ok || !f.visible(x, subject) {
		fail(w, 404, "notFound")
		return
	}
	if r.URL.Query().Get("alt") == "media" {
		_, _ = w.Write([]byte(x.Text))
		return
	}
	_ = json.NewEncoder(w).Encode(withLabels(f.fileJSON(x), x, r.URL.Query().Get("includeLabels")))
}

func (f *Fake) export(w http.ResponseWriter, r *http.Request, subject string) {
	f.mu.Lock()
	x, ok := f.files[r.PathValue("id")]
	f.mu.Unlock()
	if !ok || !f.visible(x, subject) || r.URL.Query().Get("mimeType") != "text/plain" {
		fail(w, 404, "notFound")
		return
	}
	_, _ = w.Write([]byte(x.Text))
}

func (f *Fake) startToken(w http.ResponseWriter, _ *http.Request, _ string) {
	f.mu.Lock()
	n := len(f.log)
	f.mu.Unlock()
	_ = json.NewEncoder(w).Encode(map[string]string{"startPageToken": strconv.Itoa(n)})
}

func (f *Fake) list(w http.ResponseWriter, _ *http.Request, _ string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var ids []string
	for id, x := range f.files {
		if !x.Trashed {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	var out []map[string]string
	for _, id := range ids {
		out = append(out, map[string]string{"id": id})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"files": out})
}

// changes pages two changes at a time, so the connector's paging is exercised.
func (f *Fake) changes(w http.ResponseWriter, r *http.Request, _ string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	from, err := strconv.Atoi(r.URL.Query().Get("pageToken"))
	if err != nil || from < 0 || from > len(f.log) {
		fail(w, 400, "invalidPageToken")
		return
	}
	to := from + 2
	if to > len(f.log) {
		to = len(f.log)
	}
	var out []map[string]any
	for _, c := range f.log[from:to] {
		x := f.files[c.id]
		ch := map[string]any{"fileId": c.id, "removed": false, "time": c.at.Format(time.RFC3339Nano)}
		if c.gone || x.Trashed {
			ch["removed"] = true
		} else {
			ch["file"] = f.fileJSON(x)
		}
		out = append(out, ch)
	}
	resp := map[string]any{"changes": out}
	if to == len(f.log) {
		resp["newStartPageToken"] = strconv.Itoa(to)
	} else {
		resp["nextPageToken"] = strconv.Itoa(to)
	}
	_ = json.NewEncoder(w).Encode(resp)
}
