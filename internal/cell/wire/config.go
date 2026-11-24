// Package wire is a cell's one composition root (sb-cell(5), sb-runtime(7); SB-1501): it
// reads the cell's configuration and builds every part of the running cell from it, once.
package wire

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"example.com/second-brain/internal/adapters/audit/syslogtls"
	"example.com/second-brain/internal/adapters/backup/s3"
	"example.com/second-brain/internal/adapters/smtpmail"
	"example.com/second-brain/internal/core/offsite"
)

// Config is a cell's configuration file (sb-cell(5)).
type Config struct {
	Tenant   string `json:"tenant"`
	Manifest string `json:"manifest"`
	State    string `json:"state"`
	Vault    string `json:"vault"`
	Policies string `json:"policies"`
	Listen   struct {
		API string `json:"api"`
	} `json:"listen"`
	BaseURL string `json:"base_url"`
	OIDC    struct {
		ClientID string `json:"client_id"`
		Issuer   string `json:"issuer"`
	} `json:"oidc"`
	Web string `json:"web"`
	// MCP serves POST /v1/mcp, the read-only MCP endpoint (sb-api(3), MCP; SB-3801).
	MCP   bool `json:"mcp"`
	Model struct {
		Provider string `json:"provider"`
		Base     string `json:"base"`
		ID       string `json:"id"`
		Price    struct {
			InPerM  float64 `json:"in_per_m"`
			OutPerM float64 `json:"out_per_m"`
		} `json:"price"`
		// Local is a model on the cell's own machine (SB-1710): no key, and a price of 0.
		Local bool `json:"local"`
	} `json:"model"`
	// Embed is the embedder (sb-ingest(7); SB-1710): "" or hashing, or openaicompat.
	Embed struct {
		Provider string `json:"provider"`
		Base     string `json:"base"`
		Model    string `json:"model"`
		Dim      int    `json:"dim"`
	} `json:"embed"`
	Connectors map[string]ConnectorConfig `json:"connectors"`
	Chat       struct {
		TeamID string `json:"team_id"`
	} `json:"chat"`
	// Mattermost is the Mattermost slash command (SB-2005): the server and the team whose
	// command reaches POST /chat/mattermost. Empty is none.
	Mattermost struct {
		Base   string `json:"base"`
		TeamID string `json:"team_id"`
	} `json:"mattermost"`
	// Email sends notifications by email on a cell with no chat bot (SB-2307). Empty is none.
	Email struct {
		Server string `json:"server"`
		From   string `json:"from"`
	} `json:"email"`
	Alerts struct {
		Webhook string `json:"webhook"`
		// ReportWebhook is where the weekly report is posted; "" is Webhook (SB-1608).
		ReportWebhook string `json:"report_webhook"`
	} `json:"alerts"`
	Backups []Backup `json:"backups"`
	// Audit.Stream forwards every audit row to the company's SIEM, syslog+tls://HOST:PORT
	// (sb-audit(7), Streaming to a SIEM; SB-2302).
	Audit struct {
		Stream string `json:"stream"`
	} `json:"audit"`
	Custody struct {
		Tenant bool     `json:"tenant"`
		Retain []string `json:"retain"`
	} `json:"custody"`
	Release ReleaseConfig `json:"release"`
	// Database is the cell's Postgres (a DSN, e.g. postgres:///sb_acme?host=/var/run/postgresql):
	// every record store and the audit log live there (SB-1508). Without it they are in memory.
	Database string     `json:"database"`
	Worm     WormConfig `json:"worm"`
	Durable  bool       `json:"durable"`
}

// ReleaseConfig is where the cell's updater pulls releases from and how it deploys them
// (sb-runtime(7), Updates are pulled; SB-1505).
type ReleaseConfig struct {
	Store    string `json:"store"`    // file:///DIR or an https release store
	Registry string `json:"registry"` // where the bundle's images live, e.g. europe-west1-docker.pkg.dev/sb-factory/releases
	// Exactly one of Trusted (the release key, ed25519:BASE64) and CosignIdentity (the
	// release workflow's identity, verified with cosign).
	Trusted        string `json:"trusted"`
	CosignIdentity string `json:"cosign_identity"`
	Deploy         struct {
		Kind     string `json:"kind"` // local or gcp
		Project  string `json:"project"`
		Location string `json:"location"`
	} `json:"deploy"`
}

// ConnectorConfig overrides one connector (sb-cell(5), Fields).
type ConnectorConfig struct {
	Base string `json:"base"` // the vendor API's address, for a sandbox or a recorded fake
}

// Backup is one copy the nightly backup writes (sb-audit(7), Backups): three copies on
// two kinds of media, one of them off-site.
type Backup struct {
	Dir     string `json:"dir"`
	Media   string `json:"media"`
	Offsite bool   `json:"offsite"`
	// Worm, instead of dir and media, is the copy in the write-once store (worm): an
	// object store, off the box when its endpoint is (SB-1603).
	Worm bool `json:"worm"`
	// Seal, on a dir or s3 copy, is the recovery key's public half (sb-recovery-pub:…):
	// every object is sealed to it before it is written there, as NAME.sealed (SB-1901).
	Seal string `json:"seal"`
	// S3, instead of dir and media, is a company bucket with no object lock,
	// s3://BUCKET/PREFIX, its key pair at vault://tenants/TENANT/offsite (SB-2303). It
	// must be sealed: nothing unsealed leaves the box.
	S3 string `json:"s3"`
}

// WormConfig is the write-once store for anchors and a backup copy (sb-audit(7), WORM
// storage; SB-1603): an S3-compatible bucket with compliance-mode object lock of at
// least seven years, its credentials at vault://tenants/TENANT/worm. Without it the
// anchors go to files under state, which is for development only.
type WormConfig struct {
	Endpoint string `json:"endpoint"`
	Bucket   string `json:"bucket"`
	Region   string `json:"region"`
	Offsite  bool   `json:"offsite"`
}

// ErrConfig is a configuration a cell can't start with.
var ErrConfig = errors.New("wire: bad cell configuration")

// Providers are the model providers a cell can run (ADR-0019), and the local one.
var Providers = map[string]bool{"extractive": true, "openaicompat": true, "anthropic": true, "groq": true, "openrouter": true}

// Load reads and checks a configuration file strictly: unknown fields, missing required
// fields and bad values are errors.
func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("%w: %v", ErrConfig, err)
	}
	return Parse(b)
}

// Parse checks a configuration.
func Parse(b []byte) (Config, error) {
	var c Config
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("%w: %w", ErrConfig, err)
	}
	if d.More() {
		return Config{}, fmt.Errorf("%w: more than one JSON value", ErrConfig)
	}
	var missing []string
	for name, v := range map[string]string{"tenant": c.Tenant, "manifest": c.Manifest, "state": c.State, "vault": c.Vault,
		"policies": c.Policies, "listen.api": c.Listen.API, "base_url": c.BaseURL, "oidc.client_id": c.OIDC.ClientID, "model.provider": c.Model.Provider} {
		if strings.TrimSpace(v) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("%w: missing %s", ErrConfig, strings.Join(sorted(missing), ", "))
	}
	if _, _, err := net.SplitHostPort(c.Listen.API); err != nil {
		return Config{}, fmt.Errorf("%w: listen.api %q must be host:port", ErrConfig, c.Listen.API)
	}
	for name, v := range map[string]string{"base_url": c.BaseURL, "oidc.issuer": c.OIDC.Issuer, "model.base": c.Model.Base} {
		if v == "" {
			continue
		}
		if u, err := url.Parse(v); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return Config{}, fmt.Errorf("%w: %s %q must be an http(s) URL", ErrConfig, name, v)
		}
	}
	if err := c.checkModel(); err != nil {
		return Config{}, err
	}
	for field, v := range map[string]string{"alerts.webhook": c.Alerts.Webhook, "alerts.report_webhook": c.Alerts.ReportWebhook} {
		if u, err := url.Parse(v); v != "" && (err != nil || u.Scheme != "https" || u.Host == "") {
			return Config{}, fmt.Errorf("%w: %s must be an https URL", ErrConfig, field)
		}
	}
	if c.Audit.Stream != "" {
		if _, err := syslogtls.Parse(c.Audit.Stream); err != nil {
			return Config{}, fmt.Errorf("%w: %v", ErrConfig, err)
		}
	}
	if err := c.checkMattermost(); err != nil {
		return Config{}, err
	}
	if err := c.Release.check(); err != nil {
		return Config{}, err
	}
	if c.Worm != (WormConfig{}) {
		if u, err := url.Parse(c.Worm.Endpoint); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || c.Worm.Bucket == "" {
			return Config{}, fmt.Errorf("%w: worm needs an http(s) endpoint and a bucket", ErrConfig)
		}
	}
	if err := c.checkBackups(); err != nil {
		return Config{}, err
	}
	return c, nil
}

// checkBackups refuses a backups[] copy that is none of dir, s3 or worm, or is two of them,
// or an s3 copy that isn't sealed (SB-2303).
func (c Config) checkBackups() error {
	for i, b := range c.Backups {
		switch {
		case b.Worm && c.Worm == (WormConfig{}):
			return fmt.Errorf("%w: backups[%d] is the worm copy, but there is no worm store", ErrConfig, i)
		case b.S3 != "" && (b.Worm || b.Dir != "" || b.Media != ""):
			return fmt.Errorf("%w: backups[%d]: s3 is a copy of its own, without dir, media or worm", ErrConfig, i)
		case b.S3 != "" && b.Seal == "":
			return fmt.Errorf("%w: backups[%d]: an s3 copy leaves the box, so it needs seal", ErrConfig, i)
		case b.S3 == "" && !b.Worm && (b.Dir == "" || b.Media == ""):
			return fmt.Errorf("%w: backups[%d] needs dir and media, s3, or worm", ErrConfig, i)
		case b.Seal != "" && b.Worm:
			return fmt.Errorf("%w: backups[%d]: seal is for a dir or s3 copy", ErrConfig, i)
		}
		if b.S3 != "" {
			if _, _, err := s3.URL(b.S3); err != nil {
				return fmt.Errorf("%w: backups[%d].s3: %v", ErrConfig, i, err)
			}
		}
		if b.Seal != "" {
			if _, err := offsite.ParsePublic(b.Seal); err != nil {
				return fmt.Errorf("%w: backups[%d].seal: %v", ErrConfig, i, err)
			}
		}
	}
	return nil
}

func sorted(xs []string) []string {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j] < xs[j-1]; j-- {
			xs[j], xs[j-1] = xs[j-1], xs[j]
		}
	}
	return xs
}

// ErrInProcess is a service started on its own while the cell's stores are in memory: it
// would share nothing with the api, so it runs inside sb-api instead (sb-cell(5)).
var ErrInProcess = errors.New("wire: this cell's stores are in memory, so ingest and jobs run inside the api: sb-api --config FILE --with ingest,jobs")

// Separate reports whether a service other than the api may run on its own with c: only
// when the stores are shared, which needs durable ones (SB-1508).
func Separate(c Config) error {
	if !c.Durable {
		return ErrInProcess
	}
	return nil
}

// check: a release block is all or nothing.
func (r ReleaseConfig) check() error {
	if r == (ReleaseConfig{}) {
		return nil
	}
	if u, err := url.Parse(r.Store); err != nil || (u.Scheme != "file" && u.Scheme != "https") {
		return fmt.Errorf("%w: release.store must be file:///DIR or https://HOST", ErrConfig)
	}
	if r.Registry == "" {
		return fmt.Errorf("%w: release.registry is required", ErrConfig)
	}
	if (r.Trusted == "") == (r.CosignIdentity == "") {
		return fmt.Errorf("%w: release needs exactly one of trusted and cosign_identity", ErrConfig)
	}
	if r.Trusted != "" && !strings.HasPrefix(r.Trusted, "ed25519:") {
		return fmt.Errorf("%w: release.trusted must be ed25519:BASE64", ErrConfig)
	}
	switch r.Deploy.Kind {
	case "local":
	case "gcp":
		if r.Deploy.Project == "" || r.Deploy.Location == "" {
			return fmt.Errorf("%w: release.deploy gcp needs project and location", ErrConfig)
		}
	default:
		return fmt.Errorf("%w: release.deploy.kind must be local or gcp", ErrConfig)
	}
	return nil
}

// checkModel checks the model and the embedder (sb-cell(5); SB-1710).
// checkMattermost: none, or an https base (http on 127.0.0.1 only) and a team (SB-2005);
// and email: none, or a server and one sender address (SB-2307).
func (c Config) checkMattermost() error {
	if e := c.Email; e.Server != "" || e.From != "" {
		if smtpmail.Parse(e.Server) != nil {
			return fmt.Errorf("%w: email.server must be HOST:PORT", ErrConfig)
		}
		if _, err := smtpmail.Address(e.From); err != nil {
			return fmt.Errorf("%w: email.from must be one email address", ErrConfig)
		}
	}
	mm := c.Mattermost
	if mm.Base == "" && mm.TeamID == "" {
		return nil
	}
	u, err := url.Parse(mm.Base)
	local := err == nil && u.Scheme == "http" && strings.HasPrefix(u.Host, "127.0.0.1:")
	if err != nil || (u.Scheme != "https" && !local) || u.Host == "" || u.User != nil || mm.TeamID == "" {
		return fmt.Errorf("%w: mattermost needs an https base and a team_id", ErrConfig)
	}
	return nil
}

func (c Config) checkModel() error {
	if !Providers[c.Model.Provider] {
		return fmt.Errorf("%w: model.provider %q is not one of extractive, openaicompat, anthropic, groq, openrouter", ErrConfig, c.Model.Provider)
	}
	if c.Model.Provider != "extractive" && (c.Model.Base == "" || c.Model.ID == "") {
		return fmt.Errorf("%w: model.provider %s needs model.base and model.id", ErrConfig, c.Model.Provider)
	}
	switch c.Embed.Provider {
	case "", "hashing":
	case "openaicompat":
		if c.Embed.Base == "" || c.Embed.Model == "" || c.Embed.Dim < 1 || c.Embed.Dim > 4096 {
			return fmt.Errorf("%w: embed.provider openaicompat needs embed.base, embed.model and embed.dim (1 to 4096)", ErrConfig)
		}
	default:
		return fmt.Errorf("%w: embed.provider %q is not hashing or openaicompat", ErrConfig, c.Embed.Provider)
	}
	if c.Model.Local && c.Model.Provider != "openaicompat" {
		return fmt.Errorf("%w: model.local is for an openaicompat model on the cell's own machine", ErrConfig)
	}
	if c.Model.Provider != "extractive" && !c.Model.Local && (c.Model.Price.InPerM <= 0 || c.Model.Price.OutPerM <= 0) {
		return fmt.Errorf("%w: model.provider %s needs model.price, or the budget can't be kept", ErrConfig, c.Model.Provider)
	}
	return nil
}

// Secrets are the vault paths (under vault://tenants/TENANT/) the api service reads with
// this configuration (SB-2604): required ones must be there; optional ones are checked
// only when present. Connectors' own secrets depend on the manifest and are left out.
func (c Config) Secrets() (required, optional []string) {
	required = []string{"oidc", "scim", "control-plane", "audit/anchor-seed", "attest-seed"}
	if p, ok := strings.CutPrefix(c.Database, "vault://tenants/"+c.Tenant+"/"); ok {
		required = append(required, p)
	}
	if c.Mattermost.Base != "" {
		required = append(required, "mattermost")
	}
	if c.Worm.Endpoint != "" {
		required = append(required, "worm")
	}
	optional = []string{"email"}
	return required, optional
}
