package cell

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
	"example.com/second-brain/internal/core/holds"
)

const cpToken = "control-plane-token"

const spacesManifest = docsManifest + `spaces:
  finance:
    template: finance
    members: [group:finance]
    stewards: [group:finance-leads]
    scope:
      docs: ["/Finance/**"]
`

func applyCell(t *testing.T) running {
	t.Helper()
	return startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		return celltest.FixtureFor(t, idp, base, docsManifest, map[string]string{"control-plane": cpToken,
			"connectors/docs": `{"type":"service_account"}`})
	})
}

func (r running) apply(t *testing.T, manifest, token string) (int, map[string]any) {
	t.Helper()
	hdr := []string{"Content-Type", "application/yaml"}
	if token != "" {
		hdr = append(hdr, "Authorization", "Bearer "+token)
	}
	code, body := r.do(t, "POST", "/v1/cell/manifest", manifest, hdr...)
	var v map[string]any
	_ = json.Unmarshal([]byte(body), &v)
	return code, v
}

// TestApplyToARunningCell (SB-1506, LAW-16): the control plane sends a checked manifest;
// the cell runs it at once, syncs its spaces, rewrites its manifest file and audits it;
// the same manifest again changes nothing.
func TestApplyToARunningCell(t *testing.T) {
	r := applyCell(t)
	ctx := context.Background()
	if code, _ := r.apply(t, spacesManifest, ""); code != 401 {
		t.Fatalf("no identity: %d", code)
	}
	if code, _ := r.apply(t, spacesManifest, "wrong"); code != 403 {
		t.Fatalf("wrong identity: %d", code)
	}
	if code, v := r.apply(t, "tenant: beta-freight\n", cpToken); code != 422 {
		t.Fatalf("another tenant's manifest: %d %v", code, v)
	}
	code, v := r.apply(t, spacesManifest, cpToken)
	if code != 200 || v["changed"] != true {
		t.Fatalf("apply: %d %v", code, v)
	}
	if _, found, err := r.cell.Spaces.Space(ctx, tenant, "spc_finance"); err != nil || !found {
		t.Fatalf("the new space isn't in the registry: %v %v", found, err)
	}
	if _, ok := r.cell.Manifest().Spaces["finance"]; !ok {
		t.Fatal("the cell doesn't run the applied manifest")
	}
	if b, _ := os.ReadFile(r.cell.Config.Manifest); string(b) != spacesManifest {
		t.Fatal("the manifest file wasn't rewritten, so a restart would run the old one")
	}
	rows, _ := r.cell.Audit.Store.Range(ctx, 1, 1000)
	audited := false
	for _, row := range rows {
		audited = audited || (row.Kind == "config" && row.Subject == "manifest" && row.State == "applied")
	}
	if !audited {
		t.Fatal("the apply wasn't audited")
	}
	if code, v := r.apply(t, spacesManifest, cpToken); code != 200 || v["changed"] != false {
		t.Fatalf("the same manifest twice: %d %v", code, v)
	}
}

// TestApplyRefusesToRemoveAHeldSpace: a manifest that drops a space a legal hold covers
// changes nothing at all (EHOLD), and goes through once the hold is released.
func TestApplyRefusesToRemoveAHeldSpace(t *testing.T) {
	r := applyCell(t)
	ctx := context.Background()
	if code, v := r.apply(t, spacesManifest, cpToken); code != 200 {
		t.Fatalf("apply: %d %v", code, v)
	}
	hs := holds.Service{Store: r.cell.Holds, Audit: r.cell.Audit, NewID: r.cell.NewID, Now: r.cell.Audit.Now}
	h, err := hs.Place(ctx, "usr_counsel", "matter-1", "finance", nil, "litigation")
	if err != nil {
		t.Fatal(err)
	}
	before := r.cell.Parts().SHA256
	if code, v := r.apply(t, docsManifest, cpToken); code != 409 {
		t.Fatalf("removing a held space: %d %v, want 409 EHOLD", code, v)
	}
	if r.cell.Parts().SHA256 != before {
		t.Fatal("a refused apply changed what the cell runs")
	}
	if _, found, _ := r.cell.Spaces.Space(ctx, tenant, "spc_finance"); !found {
		t.Fatal("LAW-13: a held space was removed")
	}
	if b, _ := os.ReadFile(r.cell.Config.Manifest); string(b) != spacesManifest {
		t.Fatal("a refused apply rewrote the manifest file")
	}
	if _, err := hs.Release(ctx, "usr_counsel", h.ID, "settled"); err != nil {
		t.Fatal(err)
	}
	if code, v := r.apply(t, docsManifest, cpToken); code != 200 {
		t.Fatalf("after release: %d %v", code, v)
	}
}

// TestDriftFromTheRunningCell: the hourly drift check finds a manifest file changed by
// hand, and the live configuration answers the control plane only.
func TestDriftFromTheRunningCell(t *testing.T) {
	r := applyCell(t)
	ctx := context.Background()
	if fs, err := r.cell.Drift(ctx); err != nil || len(fs) != 0 {
		t.Fatalf("a fresh cell drifts: %v %v", fs, err)
	}
	if err := os.WriteFile(r.cell.Config.Manifest, []byte(strings.Replace(docsManifest, "retention_days: 365", "retention_days: 400", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	fs, err := r.cell.Drift(ctx)
	if err != nil || len(fs) != 1 || !strings.Contains(fs[0].Message(), "retention_days") {
		t.Fatalf("drift %v %v, want the retention_days setting", fs, err)
	}
	if code, _ := r.do(t, "GET", "/v1/cell/live", ""); code != 401 {
		t.Fatalf("live without identity: %d", code)
	}
	code, body := r.do(t, "GET", "/v1/cell/live", "", "Authorization", "Bearer "+cpToken)
	if code != 200 || !strings.Contains(body, `"resources"`) {
		t.Fatalf("live: %d %s", code, body)
	}
	if code, _ := r.do(t, "GET", "/v1/verifier/attestation", "", "Authorization", "Bearer wrong"); code != 403 {
		t.Fatalf("the verifier took a wrong identity: %d", code)
	}
}
