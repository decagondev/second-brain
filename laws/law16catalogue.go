package laws

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/release"
)

// LAW-16 for the template catalogue (SB-1101): templates are data a release pins, so a
// change to any template, a label, a source, an agent or a retention, is a different
// release, and a catalogue the checker refuses never ships. Two cells on one release
// therefore always read the same templates.

// BundleWith builds a release bundle for a catalogue (the bundle tool in the real system).
type BundleWith func(catalogue []byte) (release.Manifest, error)

func bundleWith(catalogue []byte) (release.Manifest, error) {
	b := release.Bundle{
		Code:      map[string]string{"sb": release.SHA256([]byte("sb"))},
		Prompts:   map[string]string{"answer": release.SHA256([]byte("answer"))},
		Policies:  map[string]string{"policy/platform/base.cedar": release.SHA256([]byte("base"))},
		Model:     release.Model{Provider: "openaicompat", Model: "local-dev", MaxTokens: 1024},
		Gate:      release.SHA256([]byte("gate")),
		Created:   "2027-03-02",
		Catalogue: release.SHA256(catalogue),
	}
	return b.Seal()
}

func law16Catalogue(r *rand.Rand, build BundleWith) error {
	var c struct {
		Version   int                 `json:"version"`
		About     string              `json:"about"`
		Templates []manifest.Template `json:"templates"`
	}
	if err := json.Unmarshal(manifest.Catalogue, &c); err != nil {
		return err
	}
	base, err := build(manifest.Catalogue)
	if err != nil {
		return err
	}
	k := r.Intn(len(c.Templates))
	t := &c.Templates[k]
	what := ""
	switch r.Intn(4) {
	case 0:
		t.Label = map[string]string{"internal": "confidential", "confidential": "restricted", "restricted": "internal"}[t.Label]
		what = "label"
	case 1:
		t.Sources = append(t.Sources, "docs-extra")
		what = "sources"
	case 2:
		t.Agents = append(t.Agents, "draft-reply")
		what = "agents"
	default:
		t.RetentionDays += 30 + r.Intn(300)
		what = "retention"
	}
	edited, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if _, err := manifest.ParseCatalogue(edited); err != nil {
		return nil // a refused catalogue never ships; nothing to compare
	}
	next, err := build(edited)
	if err != nil {
		return err
	}
	if next.ID == base.ID {
		return fmt.Errorf("changing %s's %s left the release id %s unchanged", t.Name, what, base.ID)
	}
	again, err := build(edited)
	if err != nil || again.ID != next.ID {
		return fmt.Errorf("the same catalogue gave two release ids: %s and %s", next.ID, again.ID)
	}
	return nil
}
