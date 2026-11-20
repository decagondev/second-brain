// Package gcp is the Placement and Keys adapter for Google Cloud (sb-tenancy(7), Placement
// on GCP, exactly; SB-703). It stamps a cell into one project with the access token of a
// least-privilege deploy service account, over Google's REST APIs and the standard library.
package gcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"example.com/second-brain/internal/core/access"
	"example.com/second-brain/internal/core/ports"
)

// Endpoints are the API base URLs; the zero value means Google's.
type Endpoints struct {
	SQL, KMS, Secrets, PubSub, CRM string
}

func (e Endpoints) withDefaults() Endpoints {
	def := func(v, d string) string {
		if v == "" {
			return d
		}
		return v
	}
	return Endpoints{SQL: def(e.SQL, "https://sqladmin.googleapis.com"), KMS: def(e.KMS, "https://cloudkms.googleapis.com"),
		Secrets: def(e.Secrets, "https://secretmanager.googleapis.com"), PubSub: def(e.PubSub, "https://pubsub.googleapis.com"),
		CRM: def(e.CRM, "https://cloudresourcemanager.googleapis.com")}
}

// Locations maps the manifest's regions to GCP locations.
var Locations = map[string]string{"eu": "europe-west1", "us": "us-central1"}

// DeployPermissions are exactly the IAM permissions the adapter uses. The bootstrap script
// grants these and no others (a test compares them).
var DeployPermissions = []string{
	"cloudkms.cryptoKeyVersions.useToDecrypt",
	"cloudkms.cryptoKeyVersions.useToEncrypt",
	"cloudkms.cryptoKeys.create",
	"cloudkms.cryptoKeys.get",
	"cloudkms.cryptoKeys.setIamPolicy",
	"cloudkms.keyRings.create",
	"cloudkms.keyRings.get",
	"cloudsql.databases.create",
	"cloudsql.databases.get",
	"cloudsql.instances.create",
	"cloudsql.instances.get",
	"pubsub.topics.create",
	"pubsub.topics.get",
	"resourcemanager.projects.get",
	"secretmanager.secrets.create",
	"secretmanager.secrets.get",
}

// Project is one GCP project as a cell account.
type Project struct {
	ID        string
	HTTP      *http.Client
	Token     func(ctx context.Context) (string, error) // the deploy service account's access token
	Endpoints Endpoints
	// Poll is the wait between checks of a long-running operation; zero means one second.
	Poll time.Duration
}

var (
	_ ports.Placement = Project{}
	_ ports.Keys      = Project{}

	safe      = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
	projectRx = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)
)

// Account is the placement name.
func (p Project) Account() string { return "gcp://" + p.ID }

// apiError is an HTTP failure; only 404 means absent.
type apiError struct {
	status int
	path   string
}

func (e *apiError) Error() string { return fmt.Sprintf("gcp: HTTP %d on %s", e.status, e.path) }

func absent(err error) bool {
	var e *apiError
	return errors.As(err, &e) && e.status == http.StatusNotFound
}

func (p Project) do(ctx context.Context, method, rawURL string, body, v any) error {
	tok, err := p.Token(ctx)
	if err != nil {
		return err
	}
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		u, _ := url.Parse(rawURL)
		return &apiError{resp.StatusCode, u.Path}
	}
	if v == nil {
		return nil
	}
	return json.Unmarshal(raw, v)
}

type labelled struct {
	Name     string            `json:"name"`
	Labels   map[string]string `json:"labels"`
	Region   string            `json:"region"` // Cloud SQL instances; the others carry it in labels
	Settings struct {
		UserLabels      map[string]string `json:"userLabels"` // Cloud SQL keeps its labels here
		IPConfiguration struct {
			IPv4Enabled    bool   `json:"ipv4Enabled"`
			PrivateNetwork string `json:"privateNetwork"`
		} `json:"ipConfiguration"`
	} `json:"settings"`
	DiskEncryption struct {
		KMSKeyName string `json:"kmsKeyName"`
	} `json:"diskEncryptionConfiguration"`
}

// check refuses an existing resource that belongs to another cell, kind or location.
func check(what string, got labelled, r ports.CellResource, loc string) error {
	if r.Kind == "key-ring" {
		// A Cloud KMS key ring has no labels: it is the cell's by its name, sb-TENANT, in
		// the cell's own project and location (SB-1401).
		if !strings.HasSuffix(got.Name, "/locations/"+loc+"/keyRings/sb-"+r.Tenant) {
			return fmt.Errorf("gcp: %s %q isn't sb-%s in %s", what, got.Name, r.Tenant, loc)
		}
		return nil
	}
	l := got.Labels
	if len(l) == 0 {
		l = got.Settings.UserLabels
	}
	if l["sb-tenant"] != r.Tenant || l["sb-kind"] != r.Kind || l["sb-location"] != loc {
		return fmt.Errorf("gcp: %s exists for %s/%s in %s, not %s/%s in %s", what, l["sb-tenant"], l["sb-kind"], l["sb-location"], r.Tenant, r.Kind, loc)
	}
	return nil
}

func labels(r ports.CellResource, loc string) map[string]string {
	return map[string]string{"sb-tenant": r.Tenant, "sb-kind": r.Kind, "sb-location": loc}
}

func (p Project) prepare(r ports.CellResource) (string, error) {
	if !projectRx.MatchString(p.ID) {
		return "", fmt.Errorf("gcp: bad project id %q", p.ID)
	}
	if !safe.MatchString(r.Tenant) || !safe.MatchString(r.Kind) {
		return "", fmt.Errorf("gcp: unsafe name %q/%q", r.Tenant, r.Kind)
	}
	loc, ok := Locations[r.Region]
	if !ok {
		return "", fmt.Errorf("gcp: no GCP location for region %q", r.Region)
	}
	return loc, nil
}

// ensure GETs a resource and, only on 404, creates it with create and GETs it again.
func (p Project) ensure(ctx context.Context, what, getURL string, r ports.CellResource, loc string, create func() error) (labelled, error) {
	var got labelled
	err := p.do(ctx, http.MethodGet, getURL, nil, &got)
	if err == nil {
		return got, check(what, got, r, loc)
	}
	if !absent(err) {
		return labelled{}, err // unknown is not absent
	}
	if err := create(); err != nil {
		return labelled{}, err
	}
	if err := p.do(ctx, http.MethodGet, getURL, nil, &got); err != nil {
		return labelled{}, err
	}
	return got, check(what, got, r, loc)
}

// wait polls a long-running operation until it is done.
func (p Project) wait(ctx context.Context, opURL string) error {
	poll := p.Poll
	if poll == 0 {
		poll = time.Second
	}
	for {
		var op struct {
			Status string `json:"status"`
			Done   bool   `json:"done"`
			Error  any    `json:"error"`
		}
		if err := p.do(ctx, http.MethodGet, opURL, nil, &op); err != nil {
			return err
		}
		if op.Error != nil {
			return fmt.Errorf("gcp: operation %s failed", opURL)
		}
		if op.Status == "DONE" || op.Done {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(poll):
		}
	}
}

// Ensure finds or creates one resource and returns its id.
func (p Project) Ensure(ctx context.Context, r ports.CellResource) (string, error) {
	loc, err := p.prepare(r)
	if err != nil {
		return "", err
	}
	e := p.Endpoints.withDefaults()
	switch r.Kind {
	case "database", "index-schema":
		db, err := p.database(ctx, e, r, loc)
		if err != nil {
			return "", err
		}
		if r.Kind == "index-schema" {
			return "gcp:sql:" + db + "/schemas/" + r.Name, nil // made by the cell's migrations
		}
		return "gcp:sql:" + db, nil
	case "vault-namespace":
		name := "sb-" + r.Tenant + "-namespace"
		base := e.Secrets + "/v1/projects/" + p.ID + "/secrets"
		got, err := p.ensure(ctx, "secret "+name, base+"/"+name, r, loc, func() error {
			body := map[string]any{"labels": labels(r, loc), "replication": map[string]any{"userManaged": map[string]any{"replicas": []any{map[string]any{"location": loc}}}}}
			return p.do(ctx, http.MethodPost, base+"?secretId="+url.QueryEscape(name), body, nil)
		})
		if err != nil {
			return "", err
		}
		return "gcp:secrets:" + got.Name, nil
	case "queue":
		if !safe.MatchString(r.Name) {
			return "", fmt.Errorf("gcp: unsafe topic name %q", r.Name)
		}
		u := e.PubSub + "/v1/projects/" + p.ID + "/topics/" + r.Name
		got, err := p.ensure(ctx, "topic "+r.Name, u, r, loc, func() error {
			body := map[string]any{"labels": labels(r, loc), "messageStoragePolicy": map[string]any{"allowedPersistenceRegions": []string{loc}}}
			return p.do(ctx, http.MethodPut, u, body, nil)
		})
		if err != nil {
			return "", err
		}
		return "gcp:pubsub:" + got.Name, nil
	case "root-key":
		return p.EnsureRootKey(ctx, r)
	}
	return "", fmt.Errorf("gcp: unknown resource kind %q", r.Kind)
}

// database ensures the cell's Cloud SQL instance and its database; it returns their path.
func (p Project) database(ctx context.Context, e Endpoints, r ports.CellResource, loc string) (string, error) {
	inst := "sb-" + r.Tenant
	dbName := "sb_" + strings.ReplaceAll(r.Tenant, "-", "_")
	base := e.SQL + "/v1/projects/" + p.ID + "/instances"
	ir := ports.CellResource{Kind: "database", Name: dbName, Tenant: r.Tenant, Region: r.Region} // the instance is labelled as the database
	key, err := p.sqlKey(ctx, r)
	if err != nil {
		return "", err
	}
	network := "projects/" + p.ID + "/global/networks/sb-cell"
	got, err := p.ensure(ctx, "instance "+inst, base+"/"+inst, ir, loc, func() error {
		body := map[string]any{"name": inst, "region": loc, "databaseVersion": "POSTGRES_17",
			"diskEncryptionConfiguration": map[string]any{"kmsKeyName": key},
			"settings": map[string]any{"tier": "db-custom-2-7680", "userLabels": labels(ir, loc),
				"ipConfiguration": map[string]any{"ipv4Enabled": false, "privateNetwork": network},
				"databaseFlags":   []any{map[string]any{"name": "cloudsql.enable_pgvector", "value": "on"}}}}
		var op struct {
			Name string `json:"name"`
		}
		if err := p.do(ctx, http.MethodPost, base, body, &op); err != nil {
			return err
		}
		return p.wait(ctx, e.SQL+"/v1/projects/"+p.ID+"/operations/"+op.Name)
	})
	if err != nil {
		return "", err
	}
	if got.Region != loc {
		return "", fmt.Errorf("gcp: instance %s is in %s, not %s", inst, got.Region, loc)
	}
	if got.Settings.IPConfiguration.IPv4Enabled || got.Settings.IPConfiguration.PrivateNetwork != network || got.DiskEncryption.KMSKeyName != key {
		return "", fmt.Errorf("gcp: instance %s isn't private-IP on %s with the cell's CMEK key (SB-902)", inst, network)
	}
	dbURL := base + "/" + inst + "/databases/" + dbName
	var db struct {
		Name string `json:"name"`
	}
	err = p.do(ctx, http.MethodGet, dbURL, nil, &db)
	if absent(err) {
		var op struct {
			Name string `json:"name"`
		}
		if err := p.do(ctx, http.MethodPost, base+"/"+inst+"/databases", map[string]any{"name": dbName, "instance": inst}, &op); err != nil {
			return "", err
		}
		if err := p.wait(ctx, e.SQL+"/v1/projects/"+p.ID+"/operations/"+op.Name); err != nil {
			return "", err
		}
		err = p.do(ctx, http.MethodGet, dbURL, nil, &db)
	}
	if err != nil {
		return "", err
	}
	return "projects/" + p.ID + "/instances/" + inst + "/databases/" + dbName, nil
}

// EnsureRootKey finds or creates the cell's KMS key and returns its resource name.
func (p Project) EnsureRootKey(ctx context.Context, r ports.CellResource) (string, error) {
	if r.Kind != "root-key" {
		return "", fmt.Errorf("gcp: %s is not a root key", r.Kind)
	}
	return p.ensureKey(ctx, r)
}

// ensureKey finds or creates a key in the cell's key ring, labelled with r's kind.
func (p Project) ensureKey(ctx context.Context, r ports.CellResource) (string, error) {
	loc, err := p.prepare(r)
	if err != nil {
		return "", err
	}
	if !safe.MatchString(r.Name) {
		return "", fmt.Errorf("gcp: unsafe key name %q", r.Name)
	}
	e := p.Endpoints.withDefaults()
	ringRes := ports.CellResource{Kind: "key-ring", Name: "sb-" + r.Tenant, Tenant: r.Tenant, Region: r.Region}
	ring := "projects/" + p.ID + "/locations/" + loc + "/keyRings/sb-" + r.Tenant
	if _, err := p.ensure(ctx, "key ring", e.KMS+"/v1/"+ring, ringRes, loc, func() error {
		return p.do(ctx, http.MethodPost, e.KMS+"/v1/projects/"+p.ID+"/locations/"+loc+"/keyRings?keyRingId=sb-"+r.Tenant,
			map[string]any{}, nil)
	}); err != nil {
		return "", err
	}
	key := ring + "/cryptoKeys/" + r.Name
	got, err := p.ensure(ctx, "key "+r.Name, e.KMS+"/v1/"+key, r, loc, func() error {
		return p.do(ctx, http.MethodPost, e.KMS+"/v1/"+ring+"/cryptoKeys?cryptoKeyId="+url.QueryEscape(r.Name),
			map[string]any{"purpose": "ENCRYPT_DECRYPT", "labels": labels(r, loc)}, nil)
	})
	if err != nil {
		return "", err
	}
	return got.Name, nil
}

// number is the project's number, which names its Google-managed service agents.
func (p Project) number(ctx context.Context) (string, error) {
	var pr struct {
		ProjectNumber string `json:"projectNumber"`
	}
	if err := p.do(ctx, http.MethodGet, p.Endpoints.withDefaults().CRM+"/v1/projects/"+p.ID, nil, &pr); err != nil {
		return "", err
	}
	if pr.ProjectNumber == "" {
		return "", fmt.Errorf("gcp: project %s has no number", p.ID)
	}
	return pr.ProjectNumber, nil
}

// sqlKey makes the database's CMEK key and lets only this project's Cloud SQL agent use it.
func (p Project) sqlKey(ctx context.Context, r ports.CellResource) (string, error) {
	key, err := p.ensureKey(ctx, ports.CellResource{Kind: "database-key", Name: "cell-" + r.Tenant + "-sql", Tenant: r.Tenant, Region: r.Region})
	if err != nil {
		return "", err
	}
	num, err := p.number(ctx)
	if err != nil {
		return "", err
	}
	agent := "serviceAccount:service-" + num + "@gcp-sa-cloud-sql.iam.gserviceaccount.com"
	body := map[string]any{"policy": map[string]any{"bindings": []any{map[string]any{"role": "roles/cloudkms.cryptoKeyEncrypterDecrypter", "members": []string{agent}}}}}
	if err := p.do(ctx, http.MethodPost, p.Endpoints.withDefaults().KMS+"/v1/"+key+":setIamPolicy", body, nil); err != nil {
		return "", err
	}
	return key, nil
}

func (p Project) owns(keyID string) error {
	if !strings.HasPrefix(keyID, "projects/"+p.ID+"/locations/") || !strings.Contains(keyID, "/cryptoKeys/") {
		return fmt.Errorf("gcp: key %s is not in project %s", keyID, p.ID)
	}
	return nil
}

// Wrap encrypts a data key with the cell's KMS key; the key's name is the AAD.
func (p Project) Wrap(ctx context.Context, keyID string, dataKey []byte) ([]byte, error) {
	return p.kms(ctx, keyID, "encrypt", "plaintext", "ciphertext", dataKey)
}

// Unwrap decrypts a wrapped data key; a key wrapped under another cell's key fails.
func (p Project) Unwrap(ctx context.Context, keyID string, wrapped []byte) ([]byte, error) {
	return p.kms(ctx, keyID, "decrypt", "ciphertext", "plaintext", wrapped)
}

// kms runs one KMS encrypt or decrypt on the cell's own key, with the key's name as AAD.
func (p Project) kms(ctx context.Context, keyID, verb, in, out string, data []byte) ([]byte, error) {
	if err := p.owns(keyID); err != nil {
		return nil, err
	}
	var res map[string]string
	body := map[string]any{in: base64.StdEncoding.EncodeToString(data), "additionalAuthenticatedData": base64.StdEncoding.EncodeToString([]byte(keyID))}
	if err := p.do(ctx, http.MethodPost, p.Endpoints.withDefaults().KMS+"/v1/"+keyID+":"+verb, body, &res); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(res[out])
}

// VerifierPermissions are what the read-only verifier identity needs to enumerate access
// paths (sb-factory(7), Access paths, exactly). The tenant grants them; the deploy role
// never has them.
var VerifierPermissions = []string{
	"cloudkms.cryptoKeys.getIamPolicy",
	"iam.serviceAccounts.getIamPolicy",
	"resourcemanager.projects.getIamPolicy",
}

// GrantRoles maps the roles that serve a retained grant (sb-factory(7)); the custom role is
// relative to the project.
var GrantRoles = map[string]string{"roles/sbPlatformKill": "platform-kill", "roles/monitoring.viewer": "support-metrics"}

// IAM is the access-path source for one tenant project: every binding of a factory
// principal on the project, the deploy service account and the cell's KMS key.
type IAM struct {
	Project   Project // its Token is the verifier identity's
	Factory   []string
	Location  string // the cell's GCP location, for its key
	Endpoints struct{ CRM, IAM string }
}

var _ access.Source = IAM{}

// Paths implements access.Source.
func (s IAM) Paths(ctx context.Context, tenant string) ([]access.Path, error) {
	bs, err := s.FactoryBindings(ctx, tenant)
	if err != nil {
		return nil, err
	}
	out := make([]access.Path, len(bs))
	for i, b := range bs {
		kind := "iam-binding"
		if b.Resource == "deploy-account" {
			kind = "impersonation"
		}
		out[i] = access.Path{Source: "gcp-iam", Kind: kind, ID: b.Resource + ":" + b.Role + ":" + b.Member, Grant: b.Grant}
	}
	return out, nil
}

// Name implements access.Source.
func (IAM) Name() string { return "gcp-iam" }

type policy struct {
	Bindings []struct {
		Role    string   `json:"role"`
		Members []string `json:"members"`
	} `json:"bindings"`
}

// Policies reads the three IAM policies of a cell's tenant project.
func (s IAM) Policies(ctx context.Context, tenant string) (map[string]policy, error) {
	p := s.Project
	if !projectRx.MatchString(p.ID) || !safe.MatchString(tenant) {
		return nil, fmt.Errorf("gcp: bad project %q or tenant %q", p.ID, tenant)
	}
	crm, iam := s.Endpoints.CRM, s.Endpoints.IAM
	if crm == "" {
		crm = "https://cloudresourcemanager.googleapis.com"
	}
	if iam == "" {
		iam = "https://iam.googleapis.com"
	}
	e := p.Endpoints.withDefaults()
	sa := "sb-cell-deployer@" + p.ID + ".iam.gserviceaccount.com"
	key := "projects/" + p.ID + "/locations/" + s.Location + "/keyRings/sb-" + tenant + "/cryptoKeys/cell-" + tenant + "-root"
	urls := map[string][2]string{
		"project":        {http.MethodPost, crm + "/v1/projects/" + p.ID + ":getIamPolicy"},
		"deploy-account": {http.MethodPost, iam + "/v1/projects/" + p.ID + "/serviceAccounts/" + sa + ":getIamPolicy"},
		"cell-key":       {http.MethodGet, e.KMS + "/v1/" + key + ":getIamPolicy"},
	}
	out := map[string]policy{}
	for name, u := range urls {
		var pol policy
		var body any
		if u[0] == http.MethodPost {
			body = map[string]any{}
		}
		if err := p.do(ctx, u[0], u[1], body, &pol); err != nil {
			return nil, fmt.Errorf("gcp: %s policy: %w", name, err) // unknown is a failure, never "no paths"
		}
		out[name] = pol
	}
	return out, nil
}

// Binding is one factory binding found in a policy.
type Binding struct {
	Resource, Role, Member, Grant string
}

// FactoryBindings lists every binding of a factory principal in the tenant project's policies.
func (s IAM) FactoryBindings(ctx context.Context, tenant string) ([]Binding, error) {
	pols, err := s.Policies(ctx, tenant)
	if err != nil {
		return nil, err
	}
	factory := map[string]bool{}
	for _, f := range s.Factory {
		factory[f] = true
	}
	var out []Binding
	for _, res := range []string{"project", "deploy-account", "cell-key"} {
		for _, b := range pols[res].Bindings {
			for _, m := range b.Members {
				if !factory[m] {
					continue
				}
				role := strings.TrimPrefix(b.Role, "projects/"+s.Project.ID+"/")
				grant := ""
				if res == "project" {
					grant = GrantRoles[role]
				}
				out = append(out, Binding{Resource: res, Role: b.Role, Member: m, Grant: grant})
			}
		}
	}
	return out, nil
}
