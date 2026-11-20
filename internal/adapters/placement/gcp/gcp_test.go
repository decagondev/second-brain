package gcp

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/placement/gcp/gcptest"
	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/cell/celltest"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
)

func project(f *gcptest.Fake, id, token string) Project {
	return Project{ID: id, HTTP: f.Server.Client(), Poll: time.Millisecond,
		Token:     func(context.Context) (string, error) { return token, nil },
		Endpoints: Endpoints{SQL: f.URL(), KMS: f.URL(), Secrets: f.URL(), PubSub: f.URL(), CRM: f.URL()}}
}

func TestContract(t *testing.T) {
	f := gcptest.New()
	t.Cleanup(f.Server.Close)
	celltest.Contract(t, func(name string) celltest.Adapter {
		f.Grant("tok-"+name, name, DeployPermissions...)
		p := project(f, name, "tok-"+name)
		return celltest.Adapter{Placement: p, Keys: p}
	})
}

var gamma = manifest.Tenant{Tenant: "gamma-parts", Region: "eu"}

// Two tenant projects: a deploy account granted on one can't create, read or use anything
// in the other, and a refusal is never taken for "absent".
func TestTwoAccounts(t *testing.T) {
	f := gcptest.New()
	t.Cleanup(f.Server.Close)
	f.Grant("tok-gamma", "gamma-prod-7731", DeployPermissions...)
	f.Grant("tok-delta", "delta-prod-1204", DeployPermissions...)
	ctx := context.Background()
	mine := project(f, "gamma-prod-7731", "tok-gamma")
	rs, err := cell.Stamp(ctx, mine, mine, gamma)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rs {
		if !strings.Contains(r.ID, "gamma-prod-7731") {
			t.Errorf("%s stamped outside the project: %s", r.Kind, r.ID)
		}
	}
	theirs := project(f, "delta-prod-1204", "tok-gamma") // gamma's deployer aimed at delta
	if _, err := cell.Stamp(ctx, theirs, theirs, gamma); err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("stamping into another tenant's project: %v", err)
	}
	if f.Has("projects/delta-prod-1204/topics/gamma-parts-jobs") || f.Has("projects/delta-prod-1204/instances/sb-gamma-parts") {
		t.Fatal("a resource was created in the other project")
	}
	var key string
	for _, r := range rs {
		if r.Kind == cell.KindRootKey {
			key = r.ID
		}
	}
	_, wrapped, err := cell.NewDataKey(ctx, mine, key)
	if err != nil {
		t.Fatal(err)
	}
	delta := project(f, "delta-prod-1204", "tok-delta")
	if _, err := delta.Unwrap(ctx, key, wrapped); err == nil {
		t.Fatal("another project unwrapped this cell's data key")
	}
}

func TestUnknownIsNotAbsent(t *testing.T) {
	f := gcptest.New()
	t.Cleanup(f.Server.Close)
	f.Grant("tok", "gamma-prod-7731", DeployPermissions...)
	p := project(f, "gamma-prod-7731", "tok")
	f.FailNext(500, 1)
	if _, err := p.Ensure(context.Background(), ports.CellResource{Kind: "queue", Name: "gamma-parts-jobs", Tenant: "gamma-parts", Region: "eu"}); err == nil {
		t.Fatal("a 500 on the lookup was taken as absent")
	}
	if f.Has("projects/gamma-prod-7731/topics/gamma-parts-jobs") {
		t.Fatal("a topic was created after a failed lookup")
	}
}

func TestIdentityMismatchRefused(t *testing.T) {
	f := gcptest.New()
	t.Cleanup(f.Server.Close)
	f.Grant("tok", "gamma-prod-7731", DeployPermissions...)
	f.Put("projects/gamma-prod-7731/topics/gamma-parts-jobs", map[string]any{"name": "projects/gamma-prod-7731/topics/gamma-parts-jobs",
		"labels": map[string]any{"sb-tenant": "delta-tools", "sb-kind": "queue", "sb-location": "europe-west1"}})
	p := project(f, "gamma-prod-7731", "tok")
	if _, err := p.Ensure(context.Background(), ports.CellResource{Kind: "queue", Name: "gamma-parts-jobs", Tenant: "gamma-parts", Region: "eu"}); err == nil {
		t.Fatal("another cell's topic was reused")
	}
	if _, err := p.Ensure(context.Background(), ports.CellResource{Kind: "queue", Name: "x-jobs", Tenant: "gamma-parts", Region: "ap"}); err == nil {
		t.Fatal("an unsupported region was accepted")
	}
}

// Least privilege: every permission in DeployPermissions is needed. Without any one of them,
// stamping a cell, wrapping a data key or unwrapping it fails.
func TestEveryPermissionIsNeeded(t *testing.T) {
	ctx := context.Background()
	for i, missing := range DeployPermissions {
		f := gcptest.New()
		var perms []string
		for j, p := range DeployPermissions {
			if j != i {
				perms = append(perms, p)
			}
		}
		f.Grant("tok", "gamma-prod-7731", perms...)
		p := project(f, "gamma-prod-7731", "tok")
		err := func() error {
			rs, err := cell.Stamp(ctx, p, p, gamma)
			if err != nil {
				return err
			}
			for _, r := range rs {
				if r.Kind == cell.KindRootKey {
					_, wrapped, err := cell.NewDataKey(ctx, p, r.ID)
					if err != nil {
						return err
					}
					_, err = p.Unwrap(ctx, r.ID, wrapped)
					return err
				}
			}
			return nil
		}()
		f.Server.Close()
		if err == nil {
			t.Errorf("%s is granted but never needed", missing)
		}
	}
}

// The bootstrap script grants exactly DeployPermissions, and it is valid bash.
func TestBootstrapScriptMatchesTheCode(t *testing.T) {
	const path = "../../../../deploy/gcp/bootstrap-deploy-role.sh"
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	block := regexp.MustCompile(`(?s)permissions=\((.*?)\)`).FindStringSubmatch(string(b))
	if block == nil {
		t.Fatal("no permissions=( ... ) list in the script")
	}
	got := strings.Fields(block[1])
	if strings.Join(got, ",") != strings.Join(DeployPermissions, ",") {
		t.Fatalf("script grants %v\ncode uses %v", got, DeployPermissions)
	}
	if bash, err := exec.LookPath("bash"); err == nil {
		if out, err := exec.Command(bash, "-n", path).CombinedOutput(); err != nil {
			t.Fatalf("bash -n: %v %s", err, out)
		}
	}
}

// SB-902: the database is private-IP only and encrypted with the cell's own key, which only
// this project's Cloud SQL agent may use; destroying the key makes the data unreadable.
func TestDatabaseIsPrivateWithCMEK(t *testing.T) {
	ctx := context.Background()
	f := gcptest.New()
	t.Cleanup(f.Server.Close)
	f.Grant("tok", "gamma-prod-7731", DeployPermissions...)
	p := project(f, "gamma-prod-7731", "tok")
	db := ports.CellResource{Kind: "database", Name: "sb_gamma_parts", Tenant: "gamma-parts", Region: "eu"}
	if _, err := p.Ensure(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Ensure(ctx, db); err != nil {
		t.Fatalf("re-stamping a private CMEK database: %v", err)
	}
	f.DestroyKey("projects/gamma-prod-7731/locations/europe-west1/keyRings/sb-gamma-parts/cryptoKeys/cell-gamma-parts-sql")
	if _, err := p.Ensure(ctx, db); err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("the database was still readable after its key was destroyed: %v", err)
	}
}

// A public instance made by someone else isn't reused.
func TestPublicInstanceRefused(t *testing.T) {
	ctx := context.Background()
	f := gcptest.New()
	t.Cleanup(f.Server.Close)
	f.Grant("tok", "gamma-prod-7731", DeployPermissions...)
	f.Put("projects/gamma-prod-7731/instances/sb-gamma-parts", map[string]any{"region": "europe-west1",
		"settings": map[string]any{"userLabels": map[string]any{"sb-tenant": "gamma-parts", "sb-kind": "database", "sb-location": "europe-west1"},
			"ipConfiguration": map[string]any{"ipv4Enabled": true}}})
	p := project(f, "gamma-prod-7731", "tok")
	_, err := p.Ensure(ctx, ports.CellResource{Kind: "database", Name: "sb_gamma_parts", Tenant: "gamma-parts", Region: "eu"})
	if err == nil || !strings.Contains(err.Error(), "SB-902") {
		t.Fatalf("a public instance was reused: %v", err)
	}
}

func TestPerimeterScriptIsValidBash(t *testing.T) {
	const path = "../../../../deploy/gcp/bootstrap-perimeter.sh"
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if bash, err := exec.LookPath("bash"); err == nil {
		if out, err := exec.Command(bash, "-n", path).CombinedOutput(); err != nil {
			t.Fatalf("bash -n: %v %s", err, out)
		}
	}
}
