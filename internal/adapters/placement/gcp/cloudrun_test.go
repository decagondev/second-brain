package gcp

import (
	"context"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/placement/gcp/gcptest"
	"example.com/second-brain/internal/core/runtime"
)

func cloudRun(f *gcptest.Fake, project, token string) CloudRun {
	p := Project{ID: project, HTTP: f.Server.Client(), Poll: time.Millisecond, Token: func(context.Context) (string, error) { return token, nil }}
	return CloudRun{Project: p, Location: "europe-west1", Run: f.URL(), Analysis: f.URL()}
}

// SB-901: digests only, attested images only, and each service as its own project's account.
func TestCloudRunAdmission(t *testing.T) {
	ctx := context.Background()
	f := gcptest.New()
	defer f.Server.Close()
	f.Grant("tok", "gamma-prod-7731", UpdaterPermissions...)
	c := cloudRun(f, "gamma-prod-7731", "tok")
	api := runtime.Services[0]
	img := "europe-west1-docker.pkg.dev/sb-factory/releases/api@sha256:" + strings.Repeat("a", 64)
	if err := c.Deploy(ctx, api, "europe-west1-docker.pkg.dev/sb-factory/releases/api:latest", "rel_x"); err == nil {
		t.Fatal("a tag was deployed")
	}
	if err := c.Deploy(ctx, api, img, "rel_x"); err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("an unattested image: %v", err)
	}
	if err := c.Attest(ctx, img); err != nil {
		t.Fatal(err)
	}
	if err := c.Deploy(ctx, api, img, "rel_x"); err != nil {
		t.Fatal(err)
	}
	got, rel, err := c.Running(ctx, api)
	if err != nil || got != img || rel != "rel_x" {
		t.Fatalf("running: %q %q %v", got, rel, err)
	}
	img2 := strings.Replace(img, "aaaa", "bbbb", 1)
	_ = c.Attest(ctx, img2)
	if err := c.Deploy(ctx, api, img2, "rel_y"); err != nil { // an update, by PATCH
		t.Fatal(err)
	}
	if got, rel, _ := c.Running(ctx, api); got != img2 || rel != "rel_y" {
		t.Fatalf("after update: %q %q", got, rel)
	}
	other := cloudRun(f, "delta-prod-1204", "tok")
	if err := other.Attest(ctx, img); err == nil {
		t.Fatal("the updater acted in another project")
	}
}

func TestServiceRolesStayInTheProject(t *testing.T) {
	for _, s := range runtime.Services {
		roles, ok := ServiceRoles[s.Name]
		if !ok {
			t.Errorf("%s has no roles entry", s.Name)
		}
		for _, r := range roles {
			if strings.Contains(r, "projects/") || strings.Contains(r, "organizations/") {
				t.Errorf("%s: role %s names a resource outside the cell's project", s.Name, r)
			}
		}
	}
	if len(ServiceRoles["web"]) != 0 {
		t.Error("web must have no data access")
	}
}
