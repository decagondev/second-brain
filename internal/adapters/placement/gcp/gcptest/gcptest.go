// Package gcptest is a recorded fake of the GCP REST calls the gcp placement adapter makes
// (Cloud SQL Admin, Cloud KMS, Secret Manager, Pub/Sub). Every call needs a bearer token
// granted on the project, and each route needs one IAM permission, so a test can show a
// deploy account reaching only its own project with only the permissions it was given.
package gcptest

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// Fake is any number of GCP projects behind one server.
type Fake struct {
	Server *httptest.Server

	mu     sync.Mutex
	grants map[string]map[string]map[string]bool // token -> project -> permission
	res    map[string]map[string]any             // resource path -> JSON
	keys   map[string][]byte                     // KMS key path -> key material (never returned)
	faults []int
	// policies: resource name -> role -> members (getIamPolicy answers; empty if unset)
	policies map[string]map[string][]string
	n        int
	// Calls counts requests by project, so a test can show no call reached a project.
	Calls map[string]int
}

// New starts the fake.
func New() *Fake {
	f := &Fake{policies: map[string]map[string][]string{}, grants: map[string]map[string]map[string]bool{}, res: map[string]map[string]any{}, keys: map[string][]byte{}, Calls: map[string]int{}}
	f.Server = httptest.NewTLSServer(http.HandlerFunc(f.serve))
	return f
}

// URL is the base for every API.
func (f *Fake) URL() string { return f.Server.URL }

// Grant gives token the permissions on project.
func (f *Fake) Grant(token, project string, perms ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.grants[token] == nil {
		f.grants[token] = map[string]map[string]bool{}
	}
	if f.grants[token][project] == nil {
		f.grants[token][project] = map[string]bool{}
	}
	for _, p := range perms {
		f.grants[token][project][p] = true
	}
}

// FailNext makes the next n requests answer status.
func (f *Fake) FailNext(status, n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := 0; i < n; i++ {
		f.faults = append(f.faults, status)
	}
}

// Put stores a resource as if someone else made it (to test identity checks).
func (f *Fake) Put(path string, v map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.res[path] = v
}

// Has reports whether a resource exists.
func (f *Fake) Has(path string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.res[path]
	return ok
}

func reply(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func errBody(code int, status string) map[string]any {
	return map[string]any{"error": map[string]any{"code": code, "status": status}}
}

// route names the permission a request needs and the resource path it acts on.
func route(method, path string) (perm, project, res string, create bool) {
	p := strings.TrimPrefix(path, "/v1/")
	if strings.HasSuffix(p, ":getIamPolicy") {
		res := strings.TrimSuffix(p, ":getIamPolicy")
		proj := strings.Split(res, "/")
		if len(proj) < 2 {
			return "", "", "", false
		}
		switch {
		case strings.Contains(res, "/cryptoKeys/"):
			return "cloudkms.cryptoKeys.getIamPolicy", proj[1], "iam:" + res, false
		case strings.Contains(res, "/serviceAccounts/"):
			return "iam.serviceAccounts.getIamPolicy", proj[1], "iam:" + res, false
		case len(proj) == 2:
			return "resourcemanager.projects.getIamPolicy", proj[1], "iam:" + res, false
		}
	}
	parts := strings.Split(p, "/")
	if len(parts) < 2 || parts[0] != "projects" {
		return "", "", "", false
	}
	project = parts[1]
	switch {
	case len(parts) >= 3 && parts[2] == "operations":
		return "cloudsql.instances.get", project, p, false
	case len(parts) == 3 && parts[2] == "instances":
		return "cloudsql.instances.create", project, p, true
	case len(parts) == 4 && parts[2] == "instances":
		return "cloudsql.instances.get", project, p, false
	case len(parts) == 5 && parts[4] == "databases":
		return "cloudsql.databases.create", project, p, true
	case len(parts) == 6 && parts[4] == "databases":
		return "cloudsql.databases.get", project, p, false
	case len(parts) == 3 && parts[2] == "secrets":
		return "secretmanager.secrets.create", project, p, true
	case len(parts) == 4 && parts[2] == "secrets":
		return "secretmanager.secrets.get", project, p, false
	case len(parts) == 4 && parts[2] == "topics" && method == http.MethodPut:
		return "pubsub.topics.create", project, p, true
	case len(parts) == 4 && parts[2] == "topics":
		return "pubsub.topics.get", project, p, false
	case len(parts) == 5 && parts[4] == "keyRings":
		return "cloudkms.keyRings.create", project, p, true
	case len(parts) == 6 && parts[4] == "keyRings":
		return "cloudkms.keyRings.get", project, p, false
	case len(parts) == 7 && parts[6] == "cryptoKeys":
		return "cloudkms.cryptoKeys.create", project, p, true
	case len(parts) == 8 && strings.HasSuffix(parts[7], ":encrypt"):
		return "cloudkms.cryptoKeyVersions.useToEncrypt", project, strings.TrimSuffix(p, ":encrypt"), false
	case len(parts) == 8 && strings.HasSuffix(parts[7], ":decrypt"):
		return "cloudkms.cryptoKeyVersions.useToDecrypt", project, strings.TrimSuffix(p, ":decrypt"), false
	case len(parts) == 8 && parts[6] == "cryptoKeys":
		return "cloudkms.cryptoKeys.get", project, p, false
	}
	return "", project, "", false
}

func (f *Fake) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.faults) > 0 {
		code := f.faults[0]
		f.faults = f.faults[1:]
		reply(w, code, errBody(code, "UNAVAILABLE"))
		return
	}
	perm, project, path, create := route(r.Method, r.URL.Path)
	f.Calls[project]++
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if perm == "" {
		reply(w, 404, errBody(404, "NOT_FOUND"))
		return
	}
	if !f.grants[tok][project][perm] {
		reply(w, 403, errBody(403, "PERMISSION_DENIED"))
		return
	}
	var body map[string]any
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	switch {
	case strings.HasSuffix(perm, "useToEncrypt"), strings.HasSuffix(perm, "useToDecrypt"):
		f.crypt(w, path, perm, body)
	case strings.Contains(path, "/operations/"):
		reply(w, 200, map[string]any{"name": path, "status": "DONE"})
	case create:
		f.create(w, r, path, body)
	case strings.HasPrefix(path, "iam:"):
		bs := f.policies[strings.TrimPrefix(path, "iam:")]
		var out []any
		for role, ms := range bs {
			out = append(out, map[string]any{"role": role, "members": ms})
		}
		reply(w, 200, map[string]any{"bindings": out})
	default:
		v, ok := f.res[path]
		if !ok {
			reply(w, 404, errBody(404, "NOT_FOUND"))
			return
		}
		reply(w, 200, v)
	}
}

func (f *Fake) create(w http.ResponseWriter, r *http.Request, path string, body map[string]any) {
	labels, _ := body["labels"].(map[string]any)
	q := r.URL.Query()
	var name string
	out := map[string]any{"labels": labels}
	switch {
	case strings.HasSuffix(path, "/instances"):
		n, _ := body["name"].(string)
		name = path + "/" + n
		out = map[string]any{"settings": body["settings"], "region": body["region"], "databaseVersion": body["databaseVersion"]}
	case strings.HasSuffix(path, "/databases"):
		n, _ := body["name"].(string)
		name = path + "/" + n
	case strings.HasSuffix(path, "/secrets"):
		name = path + "/" + q.Get("secretId")
	case strings.HasSuffix(path, "/keyRings"):
		name = path + "/" + q.Get("keyRingId")
	case strings.HasSuffix(path, "/cryptoKeys"):
		name = path + "/" + q.Get("cryptoKeyId")
		k := make([]byte, 32)
		_, _ = rand.Read(k)
		f.keys[name] = k
	default:
		name = path // a topic is PUT at its own path
	}
	if _, dup := f.res[name]; dup {
		reply(w, 409, errBody(409, "ALREADY_EXISTS"))
		return
	}
	out["name"] = name
	f.res[name] = out
	f.n++
	if strings.Contains(path, "/instances") {
		reply(w, 200, map[string]any{"name": "op-" + strings.Repeat("0", 3) + string(rune('a'+f.n%26)), "status": "PENDING"})
		return
	}
	reply(w, 200, out)
}

func (f *Fake) crypt(w http.ResponseWriter, key, perm string, body map[string]any) {
	k, ok := f.keys[key]
	if !ok {
		reply(w, 404, errBody(404, "NOT_FOUND"))
		return
	}
	block, _ := aes.NewCipher(k)
	g, _ := cipher.NewGCM(block)
	aad, _ := base64.StdEncoding.DecodeString(str(body["additionalAuthenticatedData"]))
	if strings.HasSuffix(perm, "useToEncrypt") {
		pt, err := base64.StdEncoding.DecodeString(str(body["plaintext"]))
		if err != nil {
			reply(w, 400, errBody(400, "INVALID_ARGUMENT"))
			return
		}
		nonce := make([]byte, g.NonceSize())
		_, _ = rand.Read(nonce)
		reply(w, 200, map[string]any{"name": key, "ciphertext": base64.StdEncoding.EncodeToString(g.Seal(nonce, nonce, pt, aad))})
		return
	}
	ct, err := base64.StdEncoding.DecodeString(str(body["ciphertext"]))
	if err != nil || len(ct) < g.NonceSize() {
		reply(w, 400, errBody(400, "INVALID_ARGUMENT"))
		return
	}
	pt, err := g.Open(nil, ct[:g.NonceSize()], ct[g.NonceSize():], aad)
	if err != nil {
		reply(w, 400, errBody(400, "INVALID_ARGUMENT")) // decryption failed
		return
	}
	reply(w, 200, map[string]any{"plaintext": base64.StdEncoding.EncodeToString(pt)})
}

func str(v any) string { s, _ := v.(string); return s }

// Bind adds member to role in a resource's IAM policy (the project "projects/P", a service
// account or a KMS key, by resource name).
func (f *Fake) Bind(resource, role, member string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.policies[resource] == nil {
		f.policies[resource] = map[string][]string{}
	}
	f.policies[resource][role] = append(f.policies[resource][role], member)
}

// Unbind removes member from role on a resource.
func (f *Fake) Unbind(resource, role, member string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var keep []string
	for _, m := range f.policies[resource][role] {
		if m != member {
			keep = append(keep, m)
		}
	}
	f.policies[resource][role] = keep
}
