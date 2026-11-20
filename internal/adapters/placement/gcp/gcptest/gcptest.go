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
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
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
	// attested: project -> image references the project's attestor has signed (Binary Authorization)
	attested map[string]map[string]bool
	// Scheduler jobs by name; buckets by name (with their project and retention); objects by bucket/name
	Schedules map[string]map[string]any
	// sites: Firebase Hosting site -> its project; Released: site -> the released config
	sites    map[string]string
	Released map[string]map[string]any
	versions map[string]map[string]any
	buckets  map[string]*bucket
	objects  map[string][]byte
	// destroyed CMEK keys; perimeters: project -> principals allowed through; principals: token -> member
	destroyed  map[string]bool
	perimeters map[string]map[string]bool
	principals map[string]string
	// Run is Cloud Run's state: "projects/P/locations/L/services/NAME" -> resource.
	Run map[string]map[string]any
	n   int
	// Calls counts requests by project, so a test can show no call reached a project.
	Calls map[string]int
}

// New starts the fake.
func New() *Fake {
	f := &Fake{sites: map[string]string{}, Released: map[string]map[string]any{}, versions: map[string]map[string]any{}, Schedules: map[string]map[string]any{}, buckets: map[string]*bucket{}, objects: map[string][]byte{}, destroyed: map[string]bool{}, perimeters: map[string]map[string]bool{}, principals: map[string]string{}, attested: map[string]map[string]bool{}, Run: map[string]map[string]any{}, policies: map[string]map[string][]string{}, grants: map[string]map[string]map[string]bool{}, res: map[string]map[string]any{}, keys: map[string][]byte{}, Calls: map[string]int{}}
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
	if strings.HasSuffix(p, ":setIamPolicy") && strings.Contains(p, "/cryptoKeys/") {
		res := strings.TrimSuffix(p, ":setIamPolicy")
		return "cloudkms.cryptoKeys.setIamPolicy", strings.Split(res, "/")[1], "setiam:" + res, false
	}
	if method == http.MethodGet && strings.Count(p, "/") == 1 && strings.HasPrefix(p, "projects/") {
		return "resourcemanager.projects.get", strings.Split(p, "/")[1], "project:" + p, false
	}
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
	if strings.HasPrefix(r.URL.Path, "/v1beta1/sites/") {
		f.hosting(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/storage/v1/b/") || strings.HasPrefix(r.URL.Path, "/upload/storage/v1/b/") {
		f.storage(w, r)
		return
	}
	if strings.Contains(r.URL.Path, "/jobs") && strings.HasPrefix(r.URL.Path, "/v1/projects/") && !strings.Contains(r.URL.Path, "/instances") {
		f.scheduler(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/v2/projects/") || strings.HasSuffix(r.URL.Path, "/occurrences") {
		f.runtime(w, r)
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
	if allowed, fenced := f.perimeters[project]; fenced && restricted(perm) && !allowed[f.principals[tok]] {
		reply(w, 403, errBody(403, "PERMISSION_DENIED: Request is prohibited by organization's policy (VPC_SERVICE_CONTROLS)"))
		return
	}
	var body map[string]any
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	switch {
	case strings.HasPrefix(path, "project:"):
		reply(w, 200, map[string]any{"projectId": project, "projectNumber": projectNumber(project)})
		return
	case strings.HasPrefix(path, "setiam:"):
		key := strings.TrimPrefix(path, "setiam:")
		if _, ok := f.keys[key]; !ok {
			reply(w, 404, errBody(404, "NOT_FOUND"))
			return
		}
		pol, _ := body["policy"].(map[string]any)
		bs, _ := pol["bindings"].([]any)
		f.policies[key] = map[string][]string{}
		for _, b := range bs {
			bm, _ := b.(map[string]any)
			role, _ := bm["role"].(string)
			ms, _ := bm["members"].([]any)
			for _, m := range ms {
				ss, _ := m.(string)
				f.policies[key][role] = append(f.policies[key][role], ss)
			}
		}
		reply(w, 200, map[string]any{"bindings": bs})
		return
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
		if i := strings.Index(path, "/databases/"); i >= 0 {
			inst := f.res[path[:i]]
			if dec, _ := inst["diskEncryptionConfiguration"].(map[string]any); dec != nil {
				if key, _ := dec["kmsKeyName"].(string); f.destroyed[key] {
					reply(w, 400, errBody(400, "FAILED_PRECONDITION: the instance's CMEK key is destroyed; its data can't be read"))
					return
				}
			}
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
		out = map[string]any{"settings": body["settings"], "region": body["region"], "databaseVersion": body["databaseVersion"],
			"diskEncryptionConfiguration": body["diskEncryptionConfiguration"]}
		if dec, ok := body["diskEncryptionConfiguration"].(map[string]any); ok {
			key, _ := dec["kmsKeyName"].(string)
			project := strings.Split(path, "/")[1]
			agent := "serviceAccount:service-" + projectNumber(project) + "@gcp-sa-cloud-sql.iam.gserviceaccount.com"
			if _, exists := f.keys[key]; !exists || !strings.HasPrefix(key, "projects/"+project+"/") || !contains(f.policies[key]["roles/cloudkms.cryptoKeyEncrypterDecrypter"], agent) {
				reply(w, 400, errBody(400, "INVALID_ARGUMENT: the Cloud SQL service agent can't use the CMEK key"))
				return
			}
		}
	case strings.HasSuffix(path, "/databases"):
		n, _ := body["name"].(string)
		name = path + "/" + n
	case strings.HasSuffix(path, "/secrets"):
		name = path + "/" + q.Get("secretId")
	case strings.HasSuffix(path, "/keyRings"):
		name = path + "/" + q.Get("keyRingId")
		out = map[string]any{} // Cloud KMS key rings have no labels (SB-1401)
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
	if f.destroyed[key] {
		reply(w, 400, errBody(400, "FAILED_PRECONDITION: the key version is destroyed"))
		return
	}
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

// runtime serves Cloud Run Admin v2 and Container Analysis occurrences, enforcing a Binary
// Authorization policy that admits only images this project attested, and service accounts
// only from this project (SB-901).
func (f *Fake) runtime(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/v2/"), "/v1/")
	parts := strings.Split(p, "/")
	if len(parts) < 3 {
		reply(w, 404, errBody(404, "NOT_FOUND"))
		return
	}
	project := parts[1]
	f.Calls[project]++
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var body map[string]any
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	if parts[2] == "occurrences" {
		if !f.grants[tok][project]["containeranalysis.occurrences.create"] {
			reply(w, 403, errBody(403, "PERMISSION_DENIED"))
			return
		}
		uri, _ := body["resourceUri"].(string)
		if f.attested[project] == nil {
			f.attested[project] = map[string]bool{}
		}
		f.attested[project][uri] = true
		reply(w, 200, map[string]any{"name": p + "/occ-1"})
		return
	}
	// projects/P/locations/L/{services|jobs}[/NAME]
	if len(parts) < 5 || (parts[4] != "services" && parts[4] != "jobs") {
		reply(w, 404, errBody(404, "NOT_FOUND"))
		return
	}
	kind := parts[4]
	verb := map[string]string{http.MethodGet: "get", http.MethodPost: "create", http.MethodPatch: "update"}[r.Method]
	if verb == "" || !f.grants[tok][project]["run."+kind+"."+verb] {
		reply(w, 403, errBody(403, "PERMISSION_DENIED"))
		return
	}
	var name string
	if len(parts) == 6 {
		name = parts[5]
	} else {
		name = r.URL.Query().Get(map[string]string{"services": "serviceId", "jobs": "jobId"}[kind])
	}
	key := strings.Join(parts[:5], "/") + "/" + name
	switch r.Method {
	case http.MethodGet:
		v, ok := f.Run[key]
		if !ok {
			reply(w, 404, errBody(404, "NOT_FOUND"))
			return
		}
		reply(w, 200, v)
		return
	case http.MethodPost:
		if _, dup := f.Run[key]; dup {
			reply(w, 409, errBody(409, "ALREADY_EXISTS"))
			return
		}
	case http.MethodPatch:
		if _, ok := f.Run[key]; !ok {
			reply(w, 404, errBody(404, "NOT_FOUND"))
			return
		}
	}
	tmpl, _ := body["template"].(map[string]any)
	cs, _ := tmpl["containers"].([]any)
	sa, _ := tmpl["serviceAccount"].(string)
	image := ""
	if len(cs) == 1 {
		c, _ := cs[0].(map[string]any)
		image, _ = c["image"].(string)
	}
	if !f.attested[project][image] {
		reply(w, 400, errBody(400, "FAILED_PRECONDITION: denied by Binary Authorization"))
		return
	}
	if !strings.HasSuffix(sa, "@"+project+".iam.gserviceaccount.com") {
		reply(w, 400, errBody(400, "INVALID_ARGUMENT: service account from another project"))
		return
	}
	body["name"] = key
	f.Run[key] = body
	reply(w, 200, body)
}

// Running is what a service or job runs in the fake, or "".
func (f *Fake) Running(project, location, kind, name string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.Run["projects/"+project+"/locations/"+location+"/"+kind+"/"+name]
	if !ok {
		return ""
	}
	tmpl, _ := v["template"].(map[string]any)
	cs, _ := tmpl["containers"].([]any)
	if len(cs) == 0 {
		return ""
	}
	c, _ := cs[0].(map[string]any)
	img, _ := c["image"].(string)
	return img
}

func projectNumber(project string) string {
	n := 100000000000
	for _, c := range project {
		n = (n*31 + int(c)) % 900000000000
	}
	return strconv.Itoa(n + 100000000000)
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// restricted are the services a VPC Service Controls perimeter covers (sb-runtime(7)).
func restricted(perm string) bool {
	for _, p := range []string{"cloudsql.", "cloudkms.", "secretmanager.", "pubsub.", "storage."} {
		if strings.HasPrefix(perm, p) {
			return true
		}
	}
	return false
}

// DestroyKey destroys a KMS key's material (crypto-shredding).
func (f *Fake) DestroyKey(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.destroyed[key] = true // the key resource stays; its material is gone
}

// Principal says which member a token authenticates as.
func (f *Fake) Principal(token, member string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.principals[token] = member
}

// Perimeter puts project inside a VPC Service Controls perimeter that lets only members through.
func (f *Fake) Perimeter(project string, members ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.perimeters[project] = map[string]bool{}
	for _, m := range members {
		f.perimeters[project][m] = true
	}
}

type bucket struct {
	project   string
	retention time.Duration
	locked    bool
	created   map[string]time.Time
}

// Bucket makes a bucket in project with a retention policy, locked or not (bootstrap does this).
func (f *Fake) Bucket(project, name string, retention time.Duration, locked bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.buckets[name] = &bucket{project: project, retention: retention, locked: locked, created: map[string]time.Time{}}
}

// DeleteObject tries to delete an object as an admin would; a locked policy refuses it.
func (f *Fake) DeleteObject(name, object string, now time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	b := f.buckets[name]
	if b == nil {
		return errors.New("no bucket")
	}
	if at, ok := b.created[object]; ok && now.Sub(at) < b.retention {
		return errors.New("403 object is under a retention policy")
	}
	delete(f.objects, name+"/"+object)
	delete(b.created, object)
	return nil
}

func (f *Fake) scheduler(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/v1/")
	parts := strings.Split(p, "/")
	project := parts[1]
	f.Calls[project]++
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	verb := map[string]string{http.MethodGet: "get", http.MethodPost: "create", http.MethodPatch: "update"}[r.Method]
	if verb == "" || !f.grants[tok][project]["cloudscheduler.jobs."+verb] {
		reply(w, 403, errBody(403, "PERMISSION_DENIED"))
		return
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch r.Method {
	case http.MethodGet:
		v, ok := f.Schedules[p]
		if !ok {
			reply(w, 404, errBody(404, "NOT_FOUND"))
			return
		}
		reply(w, 200, v)
	case http.MethodPost:
		name, _ := body["name"].(string)
		if _, dup := f.Schedules[name]; dup {
			reply(w, 409, errBody(409, "ALREADY_EXISTS"))
			return
		}
		f.Schedules[name] = body
		reply(w, 200, body)
	default:
		if _, ok := f.Schedules[p]; !ok {
			reply(w, 404, errBody(404, "NOT_FOUND"))
			return
		}
		f.Schedules[p] = body
		reply(w, 200, body)
	}
}

func (f *Fake) storage(w http.ResponseWriter, r *http.Request) {
	upload := strings.HasPrefix(r.URL.Path, "/upload/")
	rest := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/upload"), "/storage/v1/b/")
	name, obj, _ := strings.Cut(rest, "/o")
	obj = strings.TrimPrefix(obj, "/")
	b := f.buckets[name]
	if b == nil {
		reply(w, 404, errBody(404, "NOT_FOUND"))
		return
	}
	f.Calls[b.project]++
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	need := "storage.buckets.get"
	switch {
	case upload:
		need = "storage.objects.create"
	case obj != "":
		need = "storage.objects.get"
	case strings.HasSuffix(r.URL.Path, "/o"):
		need = "storage.objects.list"
	}
	if !f.grants[tok][b.project][need] {
		reply(w, 403, errBody(403, "PERMISSION_DENIED"))
		return
	}
	if allowed, fenced := f.perimeters[b.project]; fenced && !allowed[f.principals[tok]] {
		reply(w, 403, errBody(403, "PERMISSION_DENIED: VPC_SERVICE_CONTROLS"))
		return
	}
	switch need {
	case "storage.buckets.get":
		reply(w, 200, map[string]any{"name": name, "retentionPolicy": map[string]any{
			"retentionPeriod": strconv.FormatInt(int64(b.retention/time.Second), 10), "isLocked": b.locked}})
	case "storage.objects.create":
		o := r.URL.Query().Get("name")
		if r.URL.Query().Get("ifGenerationMatch") != "0" {
			reply(w, 400, errBody(400, "INVALID_ARGUMENT: writes to this bucket must not overwrite"))
			return
		}
		if _, exists := f.objects[name+"/"+o]; exists {
			reply(w, 412, errBody(412, "FAILED_PRECONDITION"))
			return
		}
		data, _ := io.ReadAll(r.Body)
		f.objects[name+"/"+o] = data
		b.created[o] = time.Now()
		reply(w, 200, map[string]any{"name": o})
	case "storage.objects.get":
		o, _ := url.PathUnescape(obj)
		data, ok := f.objects[name+"/"+o]
		if !ok {
			reply(w, 404, errBody(404, "NOT_FOUND"))
			return
		}
		_, _ = w.Write(data)
	default:
		prefix := r.URL.Query().Get("prefix")
		var items []any
		var keys []string
		for k := range f.objects {
			if o := strings.TrimPrefix(k, name+"/"); o != k && strings.HasPrefix(o, prefix) {
				keys = append(keys, o)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			items = append(items, map[string]any{"name": k})
		}
		reply(w, 200, map[string]any{"items": items})
	}
}

// Site makes a Firebase Hosting site in project.
func (f *Fake) Site(project, site string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sites[site] = project
}

// hosting serves Firebase Hosting's versions and releases. Like the real service, a version
// may rewrite only to a Cloud Run service that exists in the site's own project.
func (f *Fake) hosting(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/v1beta1/sites/")
	site := strings.SplitN(rest, "/", 2)[0]
	project, ok := f.sites[site]
	if !ok {
		reply(w, 404, errBody(404, "NOT_FOUND"))
		return
	}
	f.Calls[project]++
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !f.grants[tok][project]["firebasehosting.sites.update"] {
		reply(w, 403, errBody(403, "PERMISSION_DENIED"))
		return
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch {
	case strings.HasSuffix(r.URL.Path, "/versions") && r.Method == http.MethodPost:
		cfg, _ := body["config"].(map[string]any)
		rws, _ := cfg["rewrites"].([]any)
		for _, x := range rws {
			rw, _ := x.(map[string]any)
			run, _ := rw["run"].(map[string]any)
			if run == nil {
				continue
			}
			svc, _ := run["serviceId"].(string)
			region, _ := run["region"].(string)
			if _, exists := f.Run["projects/"+project+"/locations/"+region+"/services/"+svc]; !exists {
				reply(w, 400, errBody(400, "INVALID_ARGUMENT: rewrite to a Cloud Run service not in this project"))
				return
			}
		}
		name := "sites/" + site + "/versions/v" + strconv.Itoa(len(f.versions)+1)
		f.versions[name] = map[string]any{"name": name, "config": cfg, "status": "CREATED"}
		reply(w, 200, f.versions[name])
	case r.Method == http.MethodPatch:
		name := strings.TrimPrefix(r.URL.Path, "/v1beta1/")
		v, ok := f.versions[name]
		if !ok {
			reply(w, 404, errBody(404, "NOT_FOUND"))
			return
		}
		v["status"] = "FINALIZED"
		reply(w, 200, v)
	case strings.HasSuffix(r.URL.Path, "/releases") && r.Method == http.MethodPost:
		v, ok := f.versions[r.URL.Query().Get("versionName")]
		if !ok || v["status"] != "FINALIZED" || !strings.HasPrefix(v["name"].(string), "sites/"+site+"/") {
			reply(w, 400, errBody(400, "FAILED_PRECONDITION: release a finalized version of this site"))
			return
		}
		cfg, _ := v["config"].(map[string]any)
		f.Released[site] = cfg
		reply(w, 200, map[string]any{"name": "sites/" + site + "/releases/r" + strconv.Itoa(len(f.Released))})
	default:
		reply(w, 404, errBody(404, "NOT_FOUND"))
	}
}
