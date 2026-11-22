// Package release is the release bundle (sb-release(7), Bundles; SB-503, ADR-0013): what
// a release pins, its canonical form, and its id. Anything that changes behaviour is in
// the bundle, so any change is a new id.
package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"

	"example.com/second-brain/internal/core/canonjson"
)

// Tool is one registered tool as the bundle pins it.
type Tool struct {
	Name          string `json:"name"`
	Connector     string `json:"connector"`
	Consequential bool   `json:"consequential"`
	Workflow      string `json:"workflow,omitempty"`
	SchemaSHA256  string `json:"schema_sha256"`
}

// Model is the pinned model.
type Model struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	// For a model on the cell's own machine (SB-1710): the weights file's SHA-256, and the
	// embedding model as MODEL@sha256:HEX.
	WeightsSHA256 string `json:"weights_sha256,omitempty"`
	Embedder      string `json:"embedder,omitempty"`
}

var embedderRe = regexp.MustCompile(`^[A-Za-z0-9._:/-]+@sha256:[0-9a-f]{64}$`)

// Bundle is bundle.json without its id.
type Bundle struct {
	Code     map[string]string `json:"code"`     // artefact -> sha256
	Prompts  map[string]string `json:"prompts"`  // name -> sha256
	Tools    []Tool            `json:"tools"`    // sorted by name
	Policies map[string]string `json:"policies"` // file -> sha256
	Model    Model             `json:"model"`
	Gate     string            `json:"gate"` // sha256 of evals/gate.yaml
	// Catalogue is the sha256 of the department template catalogue (SB-1101).
	Catalogue string `json:"catalogue,omitempty"`
	Created   string `json:"created"` // YYYY-MM-DD
}

// Manifest is bundle.json: the bundle and its id.
type Manifest struct {
	ID string `json:"id"`
	Bundle
}

var (
	hexRe  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	idRe   = regexp.MustCompile(`^rel_\d{4}-\d{2}-\d{2}_[0-9a-f]{6}$`)
)

// Validate checks every field is present and well formed.
func (b Bundle) Validate() error {
	if len(b.Code) == 0 || len(b.Prompts) == 0 || len(b.Policies) == 0 {
		return fmt.Errorf("release: bundle needs code, prompts and policies")
	}
	for _, m := range []map[string]string{b.Code, b.Prompts, b.Policies} {
		for k, v := range m {
			if k == "" || !hexRe.MatchString(v) {
				return fmt.Errorf("release: %q has a bad sha256", k)
			}
		}
	}
	for _, t := range b.Tools {
		if t.Name == "" || !hexRe.MatchString(t.SchemaSHA256) {
			return fmt.Errorf("release: tool %q needs a schema hash", t.Name)
		}
	}
	if (b.Model.WeightsSHA256 != "" && !hexRe.MatchString(b.Model.WeightsSHA256)) || (b.Model.Embedder != "" && !embedderRe.MatchString(b.Model.Embedder)) {
		return fmt.Errorf("release: the model's weights or embedder pin is malformed")
	}
	if b.Catalogue != "" && !hexRe.MatchString(b.Catalogue) {
		return fmt.Errorf("release: the catalogue has a bad sha256")
	}
	if b.Model.Provider == "" || b.Model.Model == "" || !hexRe.MatchString(b.Gate) || !dateRe.MatchString(b.Created) {
		return fmt.Errorf("release: bundle needs model, gate hash and created date")
	}
	return nil
}

// Canonical is the RFC 8785 JSON of the bundle (tools sorted by name).
func (b Bundle) Canonical() ([]byte, error) {
	sort.Slice(b.Tools, func(i, j int) bool { return b.Tools[i].Name < b.Tools[j].Name })
	raw, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	return canonjson.Encode(raw)
}

// ID is rel_<created>_<first 6 hex of sha256(canonical)>.
func (b Bundle) ID() (string, error) {
	if err := b.Validate(); err != nil {
		return "", err
	}
	c, err := b.Canonical()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(c)
	return "rel_" + b.Created + "_" + hex.EncodeToString(sum[:])[:6], nil
}

// Digest is the full sha256 of the canonical bundle; signatures cover it.
func (b Bundle) Digest() (string, error) {
	c, err := b.Canonical()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(c)
	return hex.EncodeToString(sum[:]), nil
}

// Seal returns the manifest with its id.
func (b Bundle) Seal() (Manifest, error) {
	id, err := b.ID()
	if err != nil {
		return Manifest{}, err
	}
	return Manifest{ID: id, Bundle: b}, nil
}

// Check reports whether a manifest's id matches its contents.
func (m Manifest) Check() error {
	if !idRe.MatchString(m.ID) {
		return fmt.Errorf("release: bad id %q", m.ID)
	}
	want, err := m.Bundle.ID()
	if err != nil {
		return err
	}
	if want != m.ID {
		return fmt.Errorf("release: id %s doesn't match the bundle's contents (%s)", m.ID, want)
	}
	return nil
}

// SHA256 is a helper for bundlers.
func SHA256(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
