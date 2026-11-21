package cell

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	relfs "example.com/second-brain/internal/adapters/release/localfs"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
	"example.com/second-brain/internal/core/release"
	"example.com/second-brain/internal/core/runtime"
	"example.com/second-brain/internal/obs"
)

var relKey = release.Ed25519{Priv: ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, 32))}

// publish builds, signs and gates one release in store, as the release pipeline does.
func publish(t *testing.T, store relfs.Dir, n int, sign, pass bool) string {
	t.Helper()
	ctx := context.Background()
	b := release.Bundle{
		Code:     map[string]string{},
		Prompts:  map[string]string{"answer": release.SHA256([]byte("answer"))},
		Policies: map[string]string{"policy/platform/base.cedar": release.SHA256([]byte("base"))},
		Model:    release.Model{Provider: "openaicompat", Model: "local-dev", MaxTokens: 1000 + n},
		Gate:     release.SHA256([]byte("gate")),
		Created:  "2027-03-02",
	}
	for _, s := range runtime.Services {
		b.Code[s.Image] = release.SHA256([]byte(fmt.Sprintf("%s image %d", s.Name, n)))
	}
	sbom := []byte(`{"spdxVersion":"SPDX-2.3"}`)
	b.Code["sbom"] = release.SHA256(sbom)
	m, err := b.Seal()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(m)
	if err := store.Put(m.ID, "bundle.json", raw); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(m.ID, "artefacts/sbom", sbom); err != nil {
		t.Fatal(err)
	}
	if sign {
		d, _ := m.Digest()
		s, _ := relKey.Sign(ctx, d)
		sraw, _ := json.Marshal(s)
		if err := store.Put(m.ID, "bundle.sig", sraw); err != nil {
			t.Fatal(err)
		}
	}
	g := release.GateRecord{Release: m.ID, Passed: pass, ResultsSHA256: release.SHA256([]byte("results"))}
	gd, _ := g.Digest()
	gs, _ := relKey.Sign(ctx, gd)
	graw, _ := json.Marshal(g)
	gsraw, _ := json.Marshal(gs)
	if err := store.Put(m.ID, "gate.json", graw); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(m.ID, "gate.sig", gsraw); err != nil {
		t.Fatal(err)
	}
	return m.ID
}

func updaterFixture(t *testing.T) (wire.Config, relfs.Dir) {
	t.Helper()
	idp := oidctest.New(tenant, "sb-acme-logistics", "oidc-secret")
	t.Cleanup(idp.Server.Close)
	cfg := celltest.FixtureFor(t, idp, "https://cell.example", docsManifest, nil)
	store := relfs.Dir{Root: filepath.Join(t.TempDir(), "releases")}
	cfg.Release = wire.ReleaseConfig{Store: "file://" + filepath.ToSlash(store.Root), Registry: "registry.example/sb", Trusted: relKey.Public()}
	cfg.Release.Deploy.Kind = "local"
	return cfg, store
}

// TestUpdaterPullsAdmitsAndDeploys (SB-1505, LAW-06): the updater deploys every service of
// the channel's release by digest; a second run changes nothing; an unsigned or ungated
// release changes nothing at all and raises a Sev-1.
func TestUpdaterPullsAdmitsAndDeploys(t *testing.T) {
	cfg, store := updaterFixture(t)
	ctx := context.Background()
	got := &alerts{}
	job, err := wire.NewUpdater(cfg, got, obs.NewMetrics())
	if err != nil {
		t.Fatal(err)
	}
	good := publish(t, store, 1, true, true)
	if err := store.SetChannel("stable", good); err != nil {
		t.Fatal(err)
	}
	res, err := job.Run(ctx)
	if err != nil || res.Release != good || len(res.Deployed) != len(runtime.Services) {
		t.Fatalf("first run: %+v %v", res, err)
	}
	if res, err := job.Run(ctx); err != nil || len(res.Deployed) != 0 {
		t.Fatalf("second run deployed again: %+v %v", res, err)
	}
	running := func() map[string]string {
		out := map[string]string{}
		for _, s := range runtime.Services {
			img, rel, err := job.Updater.Deploy.Running(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			out[s.Name] = img + " " + rel
		}
		return out
	}
	before := running()
	for name, rel := range map[string]string{"unsigned": publish(t, store, 2, false, true), "ungated": publish(t, store, 3, true, false)} {
		got.got = nil
		if err := store.SetChannel("stable", rel); err != nil {
			t.Fatal(err)
		}
		if _, err := job.Run(ctx); !errors.Is(err, runtime.ErrUnsigned) {
			t.Fatalf("%s: %v, want EUNSIGNED", name, err)
		}
		if after := running(); fmt.Sprint(after) != fmt.Sprint(before) {
			t.Fatalf("%s: a refused release changed what runs", name)
		}
		if len(got.got) != 1 || got.got[0].Severity != "sev1" || got.got[0].Kind != "release_unsigned" {
			t.Fatalf("%s: alerts %+v, want one Sev-1 release_unsigned", name, got.got)
		}
	}
}

func TestUpdaterNeedsAReleaseBlock(t *testing.T) {
	cfg, _ := updaterFixture(t)
	cfg.Release = wire.ReleaseConfig{}
	if _, err := wire.NewUpdater(cfg, &alerts{}, obs.NewMetrics()); !errors.Is(err, wire.ErrNoRelease) {
		t.Fatalf("%v", err)
	}
}
