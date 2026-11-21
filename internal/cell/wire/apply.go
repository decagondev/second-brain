package wire

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/drift"
	"example.com/second-brain/internal/core/groupsync"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/plan"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/errs"
)

// MaxManifest is the largest manifest a cell takes.
const MaxManifest = 1 << 20

// Errors of Apply.
var (
	ErrManifest = errors.New("wire: EMANIFEST")
	ErrHeld     = errors.New("wire: EHOLD a legal hold covers a space the manifest removes")
	ErrRestart  = errors.New("wire: EMANIFEST this field changes only with a restart")
)

func sha(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

// checkManifest checks a manifest's text for this cell: valid, this tenant's, and served
// by the cell's connector adapters.
func checkManifest(cfg Config, src []byte, name string) (manifest.Tenant, error) {
	m, ds := manifest.Check(src)
	if len(ds) > 0 {
		var lines []string
		for _, d := range ds {
			lines = append(lines, d.Format(name))
		}
		return manifest.Tenant{}, fmt.Errorf("%w: %w: the manifest doesn't check: %s", ErrConfig, ErrManifest, strings.Join(lines, "; "))
	}
	if m.Tenant != cfg.Tenant {
		return manifest.Tenant{}, fmt.Errorf("%w: %w: the manifest is for %s, not %s", ErrConfig, ErrManifest, m.Tenant, cfg.Tenant)
	}
	for kind, vendor := range m.Connectors {
		if Vendors[vendor] != kind {
			return manifest.Tenant{}, fmt.Errorf("%w: %w: connector %s: no adapter for vendor %q", ErrConfig, ErrManifest, kind, vendor)
		}
	}
	return m, nil
}

// Applied is what one apply did.
type Applied struct {
	SHA256  string   `json:"sha256"`
	Changed bool     `json:"changed"`
	Changes []string `json:"changes,omitempty"` // in the plan's format (sb-tenant(1))
}

// Apply makes the running cell run src (sb-tenant(1), apply against a running cell): it
// checks it, builds every part from it, syncs the spaces registry (refusing, and changing
// nothing, if a held space would go), applies an Identity Platform cell's groups, swaps
// the new parts in, writes the manifest file the configuration names, and audits the
// change. The same manifest twice changes nothing the second time.
func (c *Cell) Apply(ctx context.Context, src []byte, by string) (Applied, error) {
	c.applyMu.Lock()
	defer c.applyMu.Unlock()
	m, err := checkManifest(c.Config, src, "manifest")
	if err != nil {
		return Applied{}, err
	}
	cur := c.Parts()
	digest := sha(src)
	if digest == cur.SHA256 {
		return Applied{SHA256: digest}, nil
	}
	if m.IdP != cur.Manifest.IdP {
		return Applied{}, fmt.Errorf("%w: idp (restart the cell with the new manifest)", ErrRestart)
	}
	next, err := c.assemble(m, digest)
	if err != nil {
		return Applied{}, err
	}
	err = spaces.SyncHeld(ctx, c.Spaces, nil, m, func(id string) bool { return c.HeldNow(ctx).HeldSpace(id) })
	if errors.Is(err, spaces.ErrHeld) {
		return Applied{}, fmt.Errorf("%w: %v", ErrHeld, err)
	}
	if err != nil {
		return Applied{}, err
	}
	if len(m.Groups) > 0 || len(cur.Manifest.Groups) > 0 {
		if _, err := groupsync.Apply(ctx, c.Directory, c.Sessions, c.Config.Tenant, cur.Manifest.Groups, m.Groups); err != nil {
			return Applied{}, err
		}
	}
	p, err := plan.Make(m, plan.Desired(cur.Manifest), true)
	if err != nil {
		return Applied{}, fmt.Errorf("%w: %v", ErrManifest, err)
	}
	c.parts.Store(next)
	if err := writeAtomic(c.Config.Manifest, src); err != nil {
		return Applied{}, fmt.Errorf("wire: applied, but the manifest file can't be written: %w", err)
	}
	if _, err := c.Audit.Write(ctx, audit.Event{Actor: by, Kind: "config", Subject: "manifest", State: "applied", PayloadSHA256: digest,
		Reason: fmt.Sprintf("%d changes", len(p.Changes))}); err != nil {
		return Applied{}, fmt.Errorf("wire: applied, but the audit row can't be written: %w", err)
	}
	out := Applied{SHA256: digest, Changed: true}
	for _, ch := range p.Changes {
		out.Changes = append(out.Changes, ch.Line())
	}
	return out, nil
}

func writeAtomic(path string, b []byte) error {
	tmp := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Live is the cell's running configuration in the plan's terms (sb-tenant(5), Drift): the
// manifest it runs, with the spaces as the registry holds them.
func (c *Cell) Live(ctx context.Context) ([]ports.Resource, error) {
	cur := c.Parts()
	desired := plan.Desired(cur.Manifest)
	reg, err := c.Spaces.Spaces(ctx, c.Config.Tenant)
	if err != nil {
		return nil, err
	}
	have := map[string]bool{}
	for _, sp := range reg {
		have[strings.TrimPrefix(sp.ID, "spc_")] = true
	}
	var out []ports.Resource
	for _, r := range desired {
		if r.Kind == "space" {
			if !have[r.Name] {
				continue
			}
			delete(have, r.Name)
		}
		out = append(out, r)
	}
	var extra []string
	for n := range have {
		extra = append(extra, n)
	}
	sort.Strings(extra)
	for _, n := range extra {
		out = append(out, ports.Resource{Kind: "space", Name: n, Attrs: map[string]string{}})
	}
	return out, nil
}

// Drift compares what the cell runs with the manifest file its configuration names, the
// last one applied: a file changed by hand, or a registry changed outside apply, is drift.
// A file that can't be read or checked is itself a finding, never "no drift".
func (c *Cell) Drift(ctx context.Context) ([]drift.Finding, error) {
	live, err := c.Live(ctx)
	if err != nil {
		return nil, err
	}
	src, err := os.ReadFile(c.Config.Manifest)
	if err != nil {
		return nil, fmt.Errorf("wire: the applied manifest can't be read: %w", err)
	}
	m, err := checkManifest(c.Config, src, c.Config.Manifest)
	if err != nil {
		return nil, err
	}
	return drift.Detect(plan.Desired(m), live), nil
}

// controlPlane says whether r carries the control plane's identity: a bearer token equal
// to vault://tenants/TENANT/control-plane. No secret, no identity.
func (c *Cell) controlPlane(r *http.Request) bool {
	got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || got == "" {
		return false
	}
	want, err := c.Secret("control-plane")(r.Context())
	if err != nil || want == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func refuseJSON(w http.ResponseWriter, code errs.Code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(errs.HTTP(code))
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": string(code), "message": msg, "remedy": errs.Remedy(code)}})
}

func (c *Cell) guardControlPlane(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("Authorization") == "" {
		refuseJSON(w, errs.EAUTH, "The control plane's identity is required.")
		return false
	}
	if !c.controlPlane(r) {
		refuseJSON(w, errs.EPERM, "Only the control plane may do this.")
		return false
	}
	return true
}

// applyHandler is POST /v1/cell/manifest: the body is the manifest's text.
func (c *Cell) applyHandler(w http.ResponseWriter, r *http.Request) {
	if !c.guardControlPlane(w, r) {
		return
	}
	src, err := io.ReadAll(io.LimitReader(r.Body, MaxManifest+1))
	if err != nil || len(src) > MaxManifest {
		refuseJSON(w, errs.EINVAL, "The manifest can't be read, or is larger than 1 MiB.")
		return
	}
	res, err := c.Apply(r.Context(), src, "control-plane")
	switch {
	case errors.Is(err, ErrHeld):
		refuseJSON(w, errs.EHOLD, "A legal hold covers a space this manifest removes; nothing changed.")
		return
	case errors.Is(err, ErrManifest), errors.Is(err, ErrRestart), errors.Is(err, ErrConfig):
		refuseJSON(w, errs.EMANIFEST, err.Error())
		return
	case err != nil:
		c.Log.Error("apply failed", "error", err.Error())
		refuseJSON(w, errs.EUNAVAIL, "The cell couldn't apply the manifest; run apply again.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

// liveHandler is GET /v1/cell/live: the running configuration, for the control plane's
// drift check.
func (c *Cell) liveHandler(w http.ResponseWriter, r *http.Request) {
	if !c.guardControlPlane(w, r) {
		return
	}
	live, err := c.Live(r.Context())
	if err != nil {
		refuseJSON(w, errs.EUNAVAIL, "The cell's configuration can't be read right now.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"tenant": c.Config.Tenant, "sha256": c.Parts().SHA256, "resources": live})
}
