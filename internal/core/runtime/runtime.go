// Package runtime is a cell's services and its in-cell updater (sb-runtime(7); SB-901). The
// updater pulls the release its channel should run, admits it with the four checks a cell
// always runs (LAW-06), attests each image with the cell's own attestor, and deploys each
// service by digest. The factory never deploys into a cell (ADR-0015).
package runtime

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"example.com/second-brain/internal/core/release"
)

// Service is one thing a cell runs.
type Service struct {
	Name  string // api, ingest, web, jobs, updater
	Job   bool   // a Cloud Run job rather than a service
	Image string // the bundle's Code key for its image, e.g. image/api
}

// Services are every cell's, in deploy order; the updater deploys itself last.
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
	for _, s := range Services {
		d, ok := m.Code[s.Image]
		if !ok || !digestRe.MatchString(d) {
			return Result{}, fmt.Errorf("%w: %s", ErrImage, s.Name)
		}
		images[s.Name] = u.Image(s, d)
	}
	res := Result{Release: id}
	for _, s := range Services {
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
