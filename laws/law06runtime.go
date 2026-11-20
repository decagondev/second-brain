package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/placement/gcp"
	"example.com/second-brain/internal/adapters/placement/gcp/gcptest"
	"example.com/second-brain/internal/core/runtime"
)

// LAW-06 in the runtime (SB-901): a cell's services only ever run images from a release the
// cell admitted (signed, gated, digests matching), deployed by its own updater; an image
// nobody attested is refused by Binary Authorization even with the deployer's rights.
// LAW-02 in the runtime: one cell's updater can't deploy into another cell's project, and
// every service runs as an account of its own project.

// RuntimeWorld is one cell project with its updater, a release store and an outsider.
type RuntimeWorld interface {
	Release(signed, gated bool) (id string, err error) // build a fresh release
	Update(target string) error                        // the cell's updater runs for target
	Running() map[string]string                        // service -> image now
	Admitted(image string) bool                        // the image is in a release that was signed and gated
	Sideload() error                                   // deploy an unattested image with the updater's rights
	CrossDeploy() error                                // this cell's updater deploys into another project
	Accounts() []string                                // the service accounts services run as
}

// NewRuntimeWorld builds the system under test: the runtime updater over the GCP fake.
var NewRuntimeWorld = func() (RuntimeWorld, func()) {
	f := gcptest.New()
	f.Grant("tok-updater-a", rtProjectA, gcp.UpdaterPermissions...)
	f.Grant("tok-updater-b", rtProjectB, gcp.UpdaterPermissions...)
	rw := newReleaseWorld(false)
	w := &runtimeWorld{f: f, rw: rw, admitted: map[string]bool{}}
	w.cr = w.cloudRun(rtProjectA, "tok-updater-a")
	return w, f.Server.Close
}

const (
	rtProjectA = "gamma-prod-7731"
	rtProjectB = "delta-prod-1204"
	rtRegistry = "europe-west1-docker.pkg.dev/sb-factory/releases"
)

type runtimeWorld struct {
	f        *gcptest.Fake
	rw       *releaseWorld
	cr       gcp.CloudRun
	admitted map[string]bool
}

func (w *runtimeWorld) cloudRun(project, token string) gcp.CloudRun {
	p := gcp.Project{ID: project, HTTP: w.f.Server.Client(), Poll: time.Millisecond,
		Token: func(context.Context) (string, error) { return token, nil }}
	return gcp.CloudRun{Project: p, Location: "europe-west1", Run: w.f.URL(), Analysis: w.f.URL()}
}

func (w *runtimeWorld) updater(target string) runtime.Updater {
	return runtime.Updater{Registry: rtRegistry, Store: w.rw.store, Verifier: w.rw.trusted, Deploy: w.cr,
		Target: func(context.Context) (string, error) { return target, nil }}
}

func (w *runtimeWorld) Release(signed, gated bool) (string, error) {
	id, err := w.rw.Build("fresh")
	if err != nil {
		return "", err
	}
	if signed {
		if err := w.rw.Sign(id); err != nil {
			return "", err
		}
	}
	if err := w.rw.Gate(id, gated); err != nil {
		return "", err
	}
	if signed && gated {
		m, err := w.rw.manifest(id)
		if err != nil {
			return "", err
		}
		u := w.updater(id)
		for _, s := range runtime.Services {
			w.admitted[u.Image(s, m.Code[s.Image])] = true
		}
	}
	return id, nil
}

func (w *runtimeWorld) Update(target string) error {
	_, err := w.updater(target).Run(context.Background())
	return err
}

func (w *runtimeWorld) Running() map[string]string {
	out := map[string]string{}
	for _, s := range runtime.Services {
		kind := "services"
		if s.Job {
			kind = "jobs"
		}
		if img := w.f.Running(rtProjectA, "europe-west1", kind, s.Name); img != "" {
			out[s.Name] = img
		}
	}
	return out
}

func (w *runtimeWorld) Admitted(image string) bool { return w.admitted[image] }

func (w *runtimeWorld) Sideload() error {
	img := rtRegistry + "/api@sha256:" + strings.Repeat("e", 64)
	return w.cr.Deploy(context.Background(), runtime.Services[0], img, "rel_sideload")
}

func (w *runtimeWorld) CrossDeploy() error {
	other := w.cloudRun(rtProjectB, "tok-updater-a")
	img := rtRegistry + "/api@sha256:" + strings.Repeat("a", 64)
	if err := other.Attest(context.Background(), img); err != nil {
		return err
	}
	return other.Deploy(context.Background(), runtime.Services[0], img, "rel_cross")
}

func (w *runtimeWorld) Accounts() []string {
	var out []string
	for k, v := range w.f.Run {
		if !strings.HasPrefix(k, "projects/"+rtProjectA+"/") {
			continue
		}
		tmpl, _ := v["template"].(map[string]any)
		sa, _ := tmpl["serviceAccount"].(string)
		out = append(out, sa)
	}
	return out
}

func law06Runtime(r *rand.Rand, w RuntimeWorld) error {
	deployedAny, admittedAny := false, false
	for i := 0; i < 5; i++ {
		signed, gated := r.Intn(3) != 0, r.Intn(3) != 0
		id, err := w.Release(signed, gated)
		if err != nil {
			return err
		}
		admittedAny = admittedAny || (signed && gated)
		err = w.Update(id)
		if signed && gated && err != nil {
			return fmt.Errorf("an admitted release %s didn't deploy: %v", id, err)
		}
		if (!signed || !gated) && err == nil {
			return fmt.Errorf("release %s (signed %v, gated %v) deployed", id, signed, gated)
		}
		for svc, img := range w.Running() {
			deployedAny = true
			if !w.Admitted(img) {
				return fmt.Errorf("%s runs %s, which no admitted release contains", svc, img)
			}
		}
	}
	if err := w.Sideload(); err == nil {
		return errors.New("an unattested image was deployed: Binary Authorization let it through")
	}
	for svc, img := range w.Running() {
		if !w.Admitted(img) {
			return fmt.Errorf("after a sideload attempt %s runs %s", svc, img)
		}
	}
	if admittedAny && !deployedAny {
		return errors.New("an admitted release was never running: the check proved nothing")
	}
	return nil
}

func law02Runtime(r *rand.Rand, w RuntimeWorld) error {
	id, err := w.Release(true, true)
	if err != nil {
		return err
	}
	if err := w.Update(id); err != nil {
		return err
	}
	for _, sa := range w.Accounts() {
		if !strings.HasSuffix(sa, "@"+rtProjectA+".iam.gserviceaccount.com") {
			return fmt.Errorf("a service runs as %s, outside its cell's project", sa)
		}
	}
	if r.Intn(2) == 0 {
		if err := w.CrossDeploy(); err == nil {
			return errors.New("one cell's updater deployed into another cell's project")
		}
	}
	return nil
}
