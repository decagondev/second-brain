package wire

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/alert/webhook"
	deployfs "example.com/second-brain/internal/adapters/deploy/localfs"
	"example.com/second-brain/internal/adapters/placement/gcp"
	"example.com/second-brain/internal/adapters/release/cosign"
	"example.com/second-brain/internal/adapters/release/httpstore"
	relfs "example.com/second-brain/internal/adapters/release/localfs"
	"example.com/second-brain/internal/adapters/secrets/localvault"
	"example.com/second-brain/internal/core/attest"
	"example.com/second-brain/internal/core/canary"
	"example.com/second-brain/internal/core/release"
	"example.com/second-brain/internal/core/runtime"
	"example.com/second-brain/internal/obs"
)

// ErrNoRelease is a cell whose configuration has no release block.
var ErrNoRelease = errors.New("wire: the configuration has no release block (sb-cell(5), release)")

// UpdaterJob is the cell's updater (sb-runtime(7), Updates are pulled; SB-1505).
type UpdaterJob struct {
	Updater runtime.Updater
	Alerter canary.Alerter
	Tenant  string
	Metrics *obs.Metrics
}

type channelReader interface {
	Channel(ctx context.Context, channel string) (string, error)
}

// deployer is the configured deployer, or nil without a release block.
func deployer(cfg Config, hc *http.Client) (runtime.Deployer, error) {
	switch cfg.Release.Deploy.Kind {
	case "":
		return nil, nil
	case "local":
		return deployfs.Dir{Root: filepath.Join(cfg.State, "deploy")}, nil
	case "gcp":
		tok := &gcp.MetadataToken{}
		return gcp.CloudRun{Project: gcp.Project{ID: cfg.Release.Deploy.Project, HTTP: hc, Token: tok.Token}, Location: cfg.Release.Deploy.Location}, nil
	}
	return nil, fmt.Errorf("%w: release.deploy.kind %q", ErrConfig, cfg.Release.Deploy.Kind)
}

// NewUpdater builds the updater from the configuration alone: it needs the manifest's
// channel, the vault, the release store and the deployer, and nothing in memory, so it
// runs on its own.
func NewUpdater(cfg Config, alerter canary.Alerter, metrics *obs.Metrics) (*UpdaterJob, error) {
	if cfg.Release == (ReleaseConfig{}) {
		return nil, ErrNoRelease
	}
	src, err := os.ReadFile(cfg.Manifest)
	if err != nil {
		return nil, fmt.Errorf("%w: manifest: %v", ErrConfig, err)
	}
	m, err := checkManifest(cfg, src, cfg.Manifest)
	if err != nil {
		return nil, err
	}
	vault := localvault.Dir{Root: cfg.Vault}
	hc := &http.Client{Timeout: 60 * time.Second}
	var store release.Store
	var ch channelReader
	u, _ := url.Parse(cfg.Release.Store)
	if u.Scheme == "file" {
		d := relfs.Dir{Root: u.Path}
		store, ch = d, d
	} else {
		s := httpstore.Store{Base: strings.TrimRight(cfg.Release.Store, "/"), HTTP: hc,
			Token: func(ctx context.Context) (string, error) {
				return vault.Get(ctx, "vault://tenants/"+cfg.Tenant+"/release-pull")
			}}
		store, ch = s, s
	}
	var v release.Verifier = release.Ed25519Verifier{Trusted: cfg.Release.Trusted}
	if cfg.Release.CosignIdentity != "" {
		v = cosign.Verifier{Identity: cfg.Release.CosignIdentity}
	}
	dep, err := deployer(cfg, hc)
	if err != nil {
		return nil, err
	}
	channel := m.Channel
	var services []runtime.Service
	if cfg.Release.Deploy.Kind == "local" {
		services = runtime.SelfHosted
	}
	return &UpdaterJob{Tenant: cfg.Tenant, Alerter: alerter, Metrics: metrics,
		Updater: runtime.Updater{Registry: cfg.Release.Registry, Store: store, Verifier: v, Deploy: dep, Services: services,
			Target: func(ctx context.Context) (string, error) { return ch.Channel(ctx, channel) }}}, nil
}

// Run pulls, admits and deploys. A release that isn't admitted changes nothing and raises
// a Sev-1 release_unsigned alert (sb-release(7), What a cell admits).
func (j *UpdaterJob) Run(ctx context.Context) (runtime.Result, error) {
	res, err := j.Updater.Run(ctx)
	if errors.Is(err, runtime.ErrUnsigned) || errors.Is(err, runtime.ErrImage) {
		_ = j.Alerter.Alert(ctx, canary.Alert{Severity: "sev1", Kind: "release_unsigned", Tenant: j.Tenant, Where: "updater", Ref: res.Release})
	}
	if j.Metrics != nil {
		result := "ok"
		if err != nil {
			result = "failed"
		}
		j.Metrics.Add("sb_updater_runs_total", "Updater runs by result (SB-1505).", map[string]string{"result": result}, float64(1))
	}
	return res, err
}

// attestor is the verifier's attestation: what the cell's services run, signed with the
// cell's attestation key (vault://tenants/TENANT/attest-seed locally). Without a deployer,
// or without the key, the attestation is unavailable, never made up.
func (c *Cell) attestor() func(ctx context.Context) (attest.Signed, error) {
	dep, err := deployer(c.Config, &http.Client{Timeout: 30 * time.Second})
	return func(ctx context.Context) (attest.Signed, error) {
		if err != nil || dep == nil {
			return attest.Signed{}, fmt.Errorf("wire: no deployer is configured, so nothing can be attested")
		}
		raw, kerr := c.Secret("attest-seed")(ctx)
		if kerr != nil {
			return attest.Signed{}, kerr
		}
		seed, derr := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
		if derr != nil || len(seed) != ed25519.SeedSize {
			return attest.Signed{}, fmt.Errorf("wire: the attestation seed isn't 32 bytes of base64")
		}
		var services []runtime.Service
		if c.Config.Release.Deploy.Kind == "local" {
			services = runtime.SelfHosted
		}
		a := &runtime.Attestor{Tenant: c.Config.Tenant, Deploy: dep, Key: ed25519.NewKeyFromSeed(seed), Now: time.Now, Services: services}
		return a.Attest(ctx)
	}
}

// NewAlerter is the cell's alerter for a service that doesn't build the whole cell: it
// logs every alert to logw and posts it to alerts.webhook when one is configured.
func NewAlerter(cfg Config, logw io.Writer, m *obs.Metrics) canary.Alerter {
	al := alerter{log: obs.NewLogger(logw, obs.NewRedactor()), metrics: m}
	if cfg.Alerts.Webhook != "" {
		al.webhook = webhook.Sender{URL: cfg.Alerts.Webhook, HTTP: &http.Client{Timeout: 10 * time.Second}}
	}
	return al
}
