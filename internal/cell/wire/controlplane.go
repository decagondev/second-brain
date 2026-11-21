package wire

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	relfs "example.com/second-brain/internal/adapters/release/localfs"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/release"
	"example.com/second-brain/internal/core/runtime"
	"example.com/second-brain/internal/errs"
)

// AnchorSigner is the cell's audit anchor key (sb-audit(5), Anchors), read at each use.
// Locally its seed is vault://tenants/TENANT/audit/anchor-seed (32 bytes, base64); on GCP
// it is a data key wrapped by the cell's own KMS key.
func (c *Cell) AnchorSigner(ctx context.Context) (audit.Signer, error) {
	raw, err := c.Secret("audit/anchor-seed")(ctx)
	if err != nil {
		return audit.Signer{}, fmt.Errorf("wire: the anchor key can't be read: %w", err)
	}
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return audit.Signer{}, fmt.Errorf("wire: the anchor seed isn't base64")
	}
	return audit.NewSigner(c.Config.Tenant, seed)
}

// clipped is an audit store seen only up to seq last, for an export with an end.
type clipped struct {
	audit.Store
	last int64
}

func (s clipped) Range(ctx context.Context, from int64, limit int) ([]audit.Row, error) {
	rows, err := s.Store.Range(ctx, from, limit)
	var out []audit.Row
	for _, r := range rows {
		if r.Seq <= s.last {
			out = append(out, r)
		}
	}
	return out, err
}

// seqAt is the first row at or after t (from) or the last row before t (until).
func seqAt(ctx context.Context, st audit.Store, t time.Time, until bool) (int64, error) {
	var found int64
	for from := int64(1); ; {
		rows, err := st.Range(ctx, from, 1000)
		if err != nil {
			return 0, err
		}
		if len(rows) == 0 {
			if until {
				return found, nil
			}
			return from, nil
		}
		for _, r := range rows {
			at, err := time.Parse(time.RFC3339Nano, r.At)
			if err != nil {
				return 0, fmt.Errorf("audit: row %d has a bad time", r.Seq)
			}
			if !until && !at.Before(t) {
				return r.Seq, nil
			}
			if until {
				if at.After(t) {
					return found, nil
				}
				found = r.Seq
			}
		}
		from = rows[len(rows)-1].Seq + 1
	}
}

// auditRows is GET /v1/cell/audit?from=SEQ&limit=N: rows as the store holds them, oldest
// first, for sb audit tail (sb-audit(1)).
func (c *Cell) auditRows(w http.ResponseWriter, r *http.Request) {
	if !c.guardControlPlane(w, r) {
		return
	}
	from, _ := strconv.ParseInt(r.URL.Query().Get("from"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if from < 1 {
		from = 1
	}
	if limit < 1 || limit > 1000 {
		limit = 1000
	}
	rows, err := c.Audit.Store.Range(r.Context(), from, limit)
	if err != nil {
		refuseJSON(w, errs.EUNAVAIL, "The audit log can't be read right now.")
		return
	}
	next := from
	if len(rows) > 0 {
		next = rows[len(rows)-1].Seq + 1
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"rows": rows, "next": next})
}

// auditExport is GET /v1/cell/audit/export?since=TIME&until=TIME: the signed NDJSON export
// of sb-audit(5) for the rows in that window (all of them without since), anchored at the
// row before and at the last row, so it verifies on its own.
func (c *Cell) auditExport(w http.ResponseWriter, r *http.Request) {
	if !c.guardControlPlane(w, r) {
		return
	}
	ctx := r.Context()
	sg, err := c.AnchorSigner(ctx)
	if err != nil {
		refuseJSON(w, errs.EUNAVAIL, "The cell's anchor key can't be read, so nothing can be exported.")
		return
	}
	st := c.Audit.Store
	from := int64(1)
	q := r.URL.Query()
	if v := q.Get("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			refuseJSON(w, errs.EINVAL, "since must be an RFC 3339 time.")
			return
		}
		if from, err = seqAt(ctx, st, t, false); err != nil {
			refuseJSON(w, errs.EUNAVAIL, "The audit log can't be read right now.")
			return
		}
	}
	if v := q.Get("until"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			refuseJSON(w, errs.EINVAL, "until must be an RFC 3339 time.")
			return
		}
		last, err := seqAt(ctx, st, t, true)
		if err != nil {
			refuseJSON(w, errs.EUNAVAIL, "The audit log can't be read right now.")
			return
		}
		st = clipped{Store: st, last: last}
	}
	var buf strings.Builder
	if _, err := audit.Export(ctx, st, sg, from, time.Now(), &buf); err != nil {
		refuseJSON(w, errs.EUNAVAIL, "The audit log can't be exported right now.")
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	_, _ = w.Write([]byte(buf.String()))
}

// Package is the detach export of sb-factory(7), Detach, as files by path: the manifest
// the cell runs, its policies, the whole audit export and the anchor key, and the release
// it runs with its signatures, gate record, SBOM and the release key. The control plane
// adds the runbooks. Anything it can't read stops it: a package is never partial.
func (c *Cell) Package(ctx context.Context) (map[string][]byte, error) {
	files := map[string][]byte{}
	src, err := os.ReadFile(c.Config.Manifest)
	if err != nil {
		return nil, err
	}
	files["manifest.yaml"] = src
	pol, _ := filepath.Glob(filepath.Join(c.Config.Policies, "*.cedar"))
	plat, _ := filepath.Glob(filepath.Join(c.Config.Policies, "platform", "*.cedar"))
	for _, p := range append(plat, pol...) {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		rel, _ := filepath.Rel(c.Config.Policies, p)
		files["policies/"+filepath.ToSlash(rel)] = b
	}
	sg, err := c.AnchorSigner(ctx)
	if err != nil {
		return nil, err
	}
	var buf strings.Builder
	if _, err := audit.Export(ctx, c.Audit.Store, sg, 1, time.Now(), &buf); err != nil {
		return nil, err
	}
	files["audit/audit.ndjson"] = []byte(buf.String())
	files["audit/anchor-key.txt"] = []byte(sg.Key() + "\n")
	if c.Config.Release == (ReleaseConfig{}) {
		return nil, fmt.Errorf("wire: no release block, so the release the cell runs is unknown")
	}
	dep, err := deployer(c.Config, &http.Client{Timeout: 30 * time.Second})
	if err != nil || dep == nil {
		return nil, fmt.Errorf("wire: no deployer to say which release runs")
	}
	_, rel, err := dep.Running(ctx, runtime.Services[0])
	if err != nil || rel == "" {
		return nil, fmt.Errorf("wire: the api's release can't be read")
	}
	u, _ := url.Parse(c.Config.Release.Store)
	if u.Scheme != "file" {
		return nil, fmt.Errorf("wire: the package reads a file release store only, until the control-plane client (SB-105)")
	}
	var store release.Store = relfs.Dir{Root: u.Path}
	for _, f := range []string{"bundle.json", "bundle.sig", "gate.json", "gate.sig"} {
		b, err := store.Get(ctx, rel, f)
		if err != nil {
			return nil, err
		}
		files["release/"+f] = b
	}
	sbom, err := store.Get(ctx, rel, "artefacts/sbom")
	if err != nil {
		return nil, fmt.Errorf("wire: release %s has no SBOM: %w", rel, err)
	}
	files["release/sbom.spdx.json"] = sbom
	key := c.Config.Release.Trusted
	if key == "" {
		key = "cosign:" + c.Config.Release.CosignIdentity
	}
	files["release/release-key.txt"] = []byte(key + "\n")
	files["evals/gate.json"] = files["release/gate.json"] // the release's eval gate record
	return files, nil
}

// packageHandler is GET /v1/cell/package: the detach package's files, base64 in JSON.
func (c *Cell) packageHandler(w http.ResponseWriter, r *http.Request) {
	if !c.guardControlPlane(w, r) {
		return
	}
	files, err := c.Package(r.Context())
	if err != nil {
		c.Log.Error("package failed", "error", err.Error())
		refuseJSON(w, errs.EUNAVAIL, "The export package can't be built: "+err.Error())
		return
	}
	out := map[string]string{}
	for p, b := range files {
		out[p] = base64.StdEncoding.EncodeToString(b)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"tenant": c.Config.Tenant, "files": out})
}
