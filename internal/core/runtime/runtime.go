// Package runtime is a cell's services and its in-cell updater (sb-runtime(7); SB-901). The
// updater pulls the release its channel should run, admits it with the four checks a cell
// always runs (LAW-06), attests each image with the cell's own attestor, and deploys each
// service by digest. The factory never deploys into a cell (ADR-0015).
package runtime

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/attest"
	"example.com/second-brain/internal/core/release"
)

// Service is one thing a cell runs.
type Service struct {
	Name  string // api, ingest, web, jobs, updater
	Job   bool   // a Cloud Run job rather than a service
	Image string // the bundle's Code key for its image, e.g. image/api
}

// SelfHosted are a self-hosted cell's services (sb-selfhost(7)): the api serves the web UI
// itself, so there is no web service; the updater deploys itself last.
var SelfHosted = []Service{
	{Name: "api", Image: "image/api"},
	{Name: "ingest", Image: "image/ingest"},
	{Name: "jobs", Job: true, Image: "image/jobs"},
	{Name: "updater", Job: true, Image: "image/updater"},
}

// Services are every cell's on GCP, in deploy order; the updater deploys itself last.
var Services = []Service{
	{Name: "api", Image: "image/api"},
	{Name: "ingest", Image: "image/ingest"},
	{Name: "web", Image: "image/web"},
	{Name: "jobs", Job: true, Image: "image/jobs"},
	{Name: "updater", Job: true, Image: "image/updater"},
}

// Deployer runs services in one cell's project.
type Deployer interface {
	// Attest records that this cell admitted image (Binary Authorization).
	Attest(ctx context.Context, image string) error
	// Deploy runs svc at image (by digest), labelled with the release id.
	Deploy(ctx context.Context, svc Service, image, releaseID string) error
	// Running returns what svc runs now, or "" if it doesn't exist.
	Running(ctx context.Context, svc Service) (image, releaseID string, err error)
}

// Errors.
var (
	ErrUnsigned = errors.New("runtime: EUNSIGNED the release wasn't admitted; nothing changed")
	ErrImage    = errors.New("runtime: EUNSIGNED the bundle has no image for a service")
)

var digestRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Updater is one cell's in-cell updater job.
type Updater struct {
	Registry string // e.g. europe-west1-docker.pkg.dev/sb-factory/releases
	Store    release.Store
	Verifier release.Verifier
	// Target is the release the cell's channel should run (from the release controller).
	Target func(ctx context.Context) (string, error)
	Deploy Deployer
	// Services are the services to deploy; nil is Services (GCP).
	Services []Service
}

func (u Updater) services() []Service {
	if u.Services != nil {
		return u.Services
	}
	return Services
}

// Image is a service's image reference by digest; tags are never used.
func (u Updater) Image(svc Service, digest string) string {
	return u.Registry + "/" + svc.Name + "@sha256:" + digest
}

// Result is what one run did.
type Result struct {
	Release  string   `json:"release"`
	Deployed []string `json:"deployed"` // services moved to a new image
}

// Run admits the target release and deploys every service at its digest. It changes
// nothing unless the whole release is admitted and every service has an image.
func (u Updater) Run(ctx context.Context) (Result, error) {
	id, err := u.Target(ctx)
	if err != nil {
		return Result{}, err
	}
	m, _, err := release.Admit(ctx, u.Store, id, u.Verifier, nil)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrUnsigned, err)
	}
	images := map[string]string{}
	for _, s := range u.services() {
		d, ok := m.Code[s.Image]
		if !ok || !digestRe.MatchString(d) {
			return Result{}, fmt.Errorf("%w: %s", ErrImage, s.Name)
		}
		images[s.Name] = u.Image(s, d)
	}
	res := Result{Release: id}
	for _, s := range u.services() {
		cur, rel, err := u.Deploy.Running(ctx, s)
		if err != nil {
			return res, err
		}
		if cur == images[s.Name] && rel == id {
			continue
		}
		if err := u.Deploy.Attest(ctx, images[s.Name]); err != nil {
			return res, err
		}
		if err := u.Deploy.Deploy(ctx, s, images[s.Name], id); err != nil {
			return res, err
		}
		res.Deployed = append(res.Deployed, s.Name)
	}
	return res, nil
}

// Attestor reports what the cell's services actually run, read from the runtime at the moment
// of the call, signed with the cell's attestation key (sb-runtime(7), Endpoints; SB-905).
type Attestor struct {
	// Services are the services to report; nil is Services (GCP).
	Services []Service
	Tenant   string
	Deploy   Deployer
	Key      ed25519.PrivateKey
	Now      func() time.Time

	mu  sync.Mutex
	seq int64
}

// Attest builds and signs the attestation. The release is the one api runs; a service
// running another release's image shows up as a digest the bundle doesn't list.
func (a *Attestor) Attest(ctx context.Context) (attest.Signed, error) {
	r := attest.Report{Tenant: a.Tenant, Digests: map[string]string{}, At: a.Now().UTC()}
	services := a.Services
	if services == nil {
		services = Services
	}
	for _, s := range services {
		img, rel, err := a.Deploy.Running(ctx, s)
		if err != nil {
			return attest.Signed{}, err
		}
		if img == "" {
			return attest.Signed{}, fmt.Errorf("runtime: %s isn't running", s.Name)
		}
		at := strings.LastIndex(img, "@sha256:")
		if at < 0 {
			return attest.Signed{}, fmt.Errorf("runtime: %s runs an image not pinned by digest", s.Name)
		}
		r.Digests[s.Image] = img[at+len("@sha256:"):]
		if s.Name == "api" {
			r.Release = rel
		}
	}
	a.mu.Lock()
	// Milliseconds since the epoch, kept increasing: a restarted updater never replays a seq.
	a.seq = max(a.seq+1, r.At.UnixMilli())
	r.Seq = a.seq
	a.mu.Unlock()
	return attest.Sign(a.Key, r)
}
