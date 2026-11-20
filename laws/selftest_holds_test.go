package laws

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/api"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/tenancy"

	"example.com/second-brain/internal/core/holds"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retention"
	"example.com/second-brain/internal/core/spaces"
)

func TestLAW13_LegalHoldsStopDeletion(t *testing.T) {
	holdCounts.kept, holdCounts.aged = 0, 0
	Check(t, "LAW-13", runs, func(r *rand.Rand) error { return law13Holds(r, RealHoldCell) })
	if holdCounts.kept == 0 || holdCounts.aged == 0 {
		t.Fatalf("held old records %d, aged-out records %d: the law test checked nothing for one of them", holdCounts.kept, holdCounts.aged)
	}
}

// The hold law test catches a sweep that ignores holds, and a space sync that removes a
// held space.
func TestSelf_HoldMutationsAreCaught(t *testing.T) {
	noHolds := RealHoldCell
	noHolds.Held = func(holds.Set) retention.Held { return nil }
	noSyncCheck := RealHoldCell
	noSyncCheck.Sync = func(ctx context.Context, st ports.SpaceStore, tn manifest.Tenant, _ func(string) bool) error {
		return spaces.SyncHeld(ctx, st, nil, tn, nil)
	}
	for name, c := range map[string]HoldCell{"a sweep that ignores holds": noHolds, "a sync that removes held spaces": noSyncCheck} {
		if !violates(runs, func(r *rand.Rand) error { return law13Holds(r, c) }) {
			t.Errorf("LAW-13 (holds): %s was not caught", name)
		}
	}
}

// Only a Legal steward places or releases a hold, through the real handler and policies.
func TestHoldsNeedALegalSteward(t *testing.T) {
	ev, err := LoadPolicies()
	if err != nil {
		t.Fatal(err)
	}
	m, ds := manifest.Check([]byte(`tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint }
agents: [ask]
spaces:
  legal:
    template: legal
    members: [group:legal]
    stewards: [group:legal-leads]
    scope: { docs: ["/Legal/**"] }
  finance:
    template: finance
    members: [group:finance]
    stewards: [group:finance-leads]
    scope: { docs: ["/Finance/**"] }
`))
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	n := 0
	h := api.Holds{Tenant: "acme-logistics", Policy: policy.Kernel{Engine: ev, Recorder: okRecorder{}}, Manifests: &appliedManifest{t: m},
		Service: holds.Service{Store: &holds.MemoryStore{}, Audit: &flakyAudit{}, NewID: func(p string) string { n++; return fmt.Sprint(p, n) }, Now: time.Now}}
	mux := http.NewServeMux()
	h.Register(mux)
	call := func(user string, groups []string, path, body string) int {
		s, _ := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", user, groups)
		req := httptest.NewRequest("POST", path, strings.NewReader(body)).WithContext(tenancy.WithScope(context.Background(), s))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w.Code
	}
	body := `{"matter": "orion", "space": "finance", "reason": "litigation"}`
	if code := call("usr_fin", []string{"finance-leads", "sb-operators"}, "/v1/holds", body); code != 403 {
		t.Fatalf("a finance steward and operator placed a hold: %d", code)
	}
	if code := call("usr_counsel", []string{"legal-leads"}, "/v1/holds", body); code != 200 {
		t.Fatalf("a Legal steward couldn't place a hold: %d", code)
	}
	if code := call("usr_fin", []string{"finance-leads"}, "/v1/holds/hold_1/release", `{"reason": "done"}`); code != 403 {
		t.Fatalf("a non-Legal steward released a hold: %d", code)
	}
	if code := call("usr_counsel", []string{"legal-leads"}, "/v1/holds/hold_1/release", `{"reason": "done"}`); code != 200 {
		t.Fatalf("a Legal steward couldn't release: %d", code)
	}
}
