package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"example.com/second-brain/internal/adapters/placement/gcp"
	"example.com/second-brain/internal/adapters/placement/gcp/gcptest"
	"example.com/second-brain/internal/core/access"
	"example.com/second-brain/internal/core/keywrap"
)

// LAW-17 in the access-path check (SB-706): for random IAM bindings on a tenant project, its
// deploy account and its key, and random key wraps, the check reports exactly the factory's
// paths that no retained grant serves; a tenant's own bindings are never paths; and an
// enumeration that fails never passes.

// AccessWorld is one tenant project with its policies and wrap inventory.
type AccessWorld interface {
	Bind(resource, role, member string) // resource: "project", "deploy-account" or "cell-key"
	FactoryWrap()                       // the factory still wraps a data key
	FailNext()                          // the next policy read fails
	Check(retain []string) (unlisted []string, err error)
}

// NewAccessWorld builds the system under test: access.Check over the gcp IAM source (through
// its fake, read with a verifier token) and the wrap-inventory source.
var NewAccessWorld = func() (AccessWorld, func()) {
	f := gcptest.New()
	f.Grant("tok-verifier", accProject, gcp.VerifierPermissions...)
	p := gcp.Project{ID: accProject, HTTP: f.Server.Client(), Poll: time.Millisecond,
		Token:     func(context.Context) (string, error) { return "tok-verifier", nil },
		Endpoints: gcp.Endpoints{SQL: f.URL(), KMS: f.URL(), Secrets: f.URL(), PubSub: f.URL(), CRM: f.URL()}}
	src := gcp.IAM{Project: p, Factory: accFactory, Location: "europe-west1"}
	src.Endpoints.CRM, src.Endpoints.IAM = f.URL(), f.URL()
	w := &accessWorld{f: f, sources: []access.Source{src, access.Wraps{Store: &keywrap.MemoryStore{}}}}
	w.wraps = w.sources[1].(access.Wraps).Store.(*keywrap.MemoryStore)
	return w, f.Server.Close
}

const accProject = "gamma-prod-7731"

var accFactory = []string{"serviceAccount:sb-factory@factory-prod.iam.gserviceaccount.com", "group:sb-breakglass@factory.example"}

var accResources = map[string]string{
	"project":        "projects/" + accProject,
	"deploy-account": "projects/" + accProject + "/serviceAccounts/sb-cell-deployer@" + accProject + ".iam.gserviceaccount.com",
	"cell-key":       "projects/" + accProject + "/locations/europe-west1/keyRings/sb-gamma-parts/cryptoKeys/cell-gamma-parts-root",
}

type accessWorld struct {
	f       *gcptest.Fake
	sources []access.Source
	wraps   *keywrap.MemoryStore
}

func (w *accessWorld) Bind(res, role, member string) { w.f.Bind(accResources[res], role, member) }
func (w *accessWorld) FactoryWrap() {
	_ = w.wraps.Put(context.Background(), "gamma-parts", keywrap.Wrap{Purpose: keywrap.PurposeAuditSigning, Custodian: "factory", KeyID: "k1", Wrapped: []byte{1}})
}
func (w *accessWorld) FailNext() { w.f.FailNext(500, 1) }
func (w *accessWorld) Check(retain []string) ([]string, error) {
	res, err := access.Check(context.Background(), "gamma-parts", w.sources, retain)
	var out []string
	for _, p := range res.Unlisted {
		out = append(out, p.ID)
	}
	return out, err
}

func law17Access(r *rand.Rand, w AccessWorld) error {
	tenantMember := "group:gamma-it@gamma.example"
	type binding struct{ res, role, member string }
	choices := []binding{
		{"project", "roles/owner", accFactory[1]},
		{"project", "roles/viewer", accFactory[0]},
		{"project", "projects/" + accProject + "/roles/sbPlatformKill", accFactory[0]},
		{"project", "roles/monitoring.viewer", accFactory[0]},
		{"project", "roles/owner", tenantMember},
		{"deploy-account", "roles/iam.serviceAccountTokenCreator", accFactory[0]},
		{"deploy-account", "roles/iam.serviceAccountTokenCreator", tenantMember},
		{"cell-key", "roles/cloudkms.cryptoKeyEncrypterDecrypter", accFactory[0]},
		{"cell-key", "roles/cloudkms.admin", tenantMember},
	}
	retain := []string{"publish-releases"}
	for _, g := range []string{"platform-kill", "support-metrics"} {
		if r.Intn(2) == 0 {
			retain = append(retain, g)
		}
	}
	keep := map[string]bool{}
	for _, g := range retain {
		keep[g] = true
	}
	var want []string
	for _, b := range choices {
		if r.Intn(2) != 0 {
			continue
		}
		w.Bind(b.res, b.role, b.member)
		factory := b.member != tenantMember
		grant := map[string]string{"projects/" + accProject + "/roles/sbPlatformKill": "platform-kill", "roles/monitoring.viewer": "support-metrics"}[b.role]
		if factory && (b.res != "project" || grant == "" || !keep[grant]) {
			want = append(want, b.res+":"+b.role+":"+b.member)
		}
	}
	if r.Intn(2) == 0 {
		w.FactoryWrap()
		want = append(want, "key-wrap:"+keywrap.PurposeAuditSigning+":k1")
	}
	sort.Strings(want)
	got, err := w.Check(retain)
	if err != nil {
		return err
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		return fmt.Errorf("retaining %v, unlisted paths are %v; want %v", retain, got, want)
	}
	w.FailNext()
	if got, err := w.Check(retain); err == nil || !errors.Is(err, access.ErrUnavailable) {
		return fmt.Errorf("a failed policy read passed the check (unlisted %v, err %v)", got, err)
	}
	return nil
}
