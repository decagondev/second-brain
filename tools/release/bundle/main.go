// Command bundle builds a release's bundle.json (sb-release(7), Bundles; SB-503): the
// artefacts' digests, the bundled prompts, the tool registry's schemas, the platform
// policies, the pinned model and the eval gate, sealed with its release id.
//
//	go run ./tools/release/bundle --artefact sb=bin/sb [--artefact web=web/dist.tar] \
//	    [--image api=sha256:…]... --created 2027-03-02 --out bundle.json
//
// --image records a service's image by digest as image/NAME (sb-release(7)), what the
// updater deploys by; make images prints them (sb-selfhost(7)).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/prompts"
	"example.com/second-brain/internal/core/release"
	"example.com/second-brain/internal/core/workflow"
)

type list []string

func (l *list) String() string     { return strings.Join(*l, ",") }
func (l *list) Set(v string) error { *l = append(*l, v); return nil }

func main() {
	var arts, images list
	flag.Var(&images, "image", "name=sha256:DIGEST of a service image (repeatable)")
	flag.Var(&arts, "artefact", "name=path of a built artefact (repeatable)")
	root := flag.String("root", ".", "repository root")
	created := flag.String("created", "", "build date, YYYY-MM-DD")
	out := flag.String("out", "bundle.json", "where to write the manifest")
	digestOut := flag.String("digest-out", "", "also write the message to sign: sb-release:v1:<bundle digest>")
	weights := flag.String("model-weights", "", "a model on the box (SB-1710): JSON {model, weights_sha256, embedder} over release/model.json")
	flag.Parse()
	m, err := buildWith(*root, arts, *created, *weights, images...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bundle:", err)
		os.Exit(1)
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(*out, append(b, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "bundle:", err)
		os.Exit(1)
	}
	if *digestOut != "" {
		d, err := m.Digest()
		if err != nil {
			fmt.Fprintln(os.Stderr, "bundle:", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*digestOut, []byte("sb-release:v1:"+d), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "bundle:", err)
			os.Exit(1)
		}
	}
	fmt.Fprintln(os.Stdout, m.ID)
}

var imageDigest = regexp.MustCompile(`^sha256:([0-9a-f]{64})$`)

func buildWith(root string, arts []string, created, weights string, images ...string) (release.Manifest, error) {
	b := release.Bundle{Code: map[string]string{}, Prompts: map[string]string{}, Policies: map[string]string{}, Created: created}
	for _, im := range images {
		name, d, ok := strings.Cut(im, "=")
		m := imageDigest.FindStringSubmatch(d)
		if !ok || name == "" || m == nil {
			return release.Manifest{}, fmt.Errorf("--image wants name=sha256:<64 hex>, got %q", im)
		}
		b.Code["image/"+name] = m[1]
	}
	for _, a := range arts {
		name, path, ok := strings.Cut(a, "=")
		if !ok || name == "" {
			return release.Manifest{}, fmt.Errorf("--artefact wants name=path, got %q", a)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return release.Manifest{}, err
		}
		b.Code[name] = release.SHA256(data)
	}
	ps, err := prompts.All()
	if err != nil {
		return release.Manifest{}, err
	}
	for _, p := range ps {
		b.Prompts[p.Name] = p.SHA256
	}
	names := make([]string, 0, len(workflow.Templates))
	for n := range workflow.Templates {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		t := workflow.Templates[n]
		b.Tools = append(b.Tools, release.Tool{Name: t.Tool, Connector: t.Connector, Consequential: true, Workflow: n, SchemaSHA256: release.SHA256([]byte(t.Schema))})
	}
	// A steward's note (SB-1705): the one tool a person calls, on the folder connector.
	b.Tools = append(b.Tools, release.Tool{Name: ask.NoteTool, Connector: "docs", SchemaSHA256: release.SHA256([]byte(ask.NoteSchema))})
	files, _ := filepath.Glob(filepath.Join(root, "policy", "platform", "*.cedar"))
	files = append(files, filepath.Join(root, "policy", "schema.cedarschema"))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return release.Manifest{}, err
		}
		rel, _ := filepath.Rel(root, f)
		b.Policies[filepath.ToSlash(rel)] = release.SHA256(data)
	}
	mb, err := os.ReadFile(filepath.Join(root, "release", "model.json"))
	if err != nil {
		return release.Manifest{}, err
	}
	if err := json.Unmarshal(mb, &b.Model); err != nil {
		return release.Manifest{}, fmt.Errorf("release/model.json: %w", err)
	}
	if weights != "" {
		if err := overlayWeights(&b.Model, weights); err != nil {
			return release.Manifest{}, err
		}
	}
	gate, err := os.ReadFile(filepath.Join(root, "evals", "gate.yaml"))
	if err != nil {
		return release.Manifest{}, err
	}
	b.Gate = release.SHA256(gate)
	b.Catalogue = release.SHA256(manifest.Catalogue)
	return b.Seal()
}

// overlayWeights pins a model on the box over release/model.json (SB-1710).
func overlayWeights(m *release.Model, path string) error {
	wb, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var w struct {
		Model         string `json:"model"`
		WeightsSHA256 string `json:"weights_sha256"`
		Embedder      string `json:"embedder"`
	}
	if err := json.Unmarshal(wb, &w); err != nil || w.Model == "" || w.WeightsSHA256 == "" {
		return fmt.Errorf("--model-weights wants {model, weights_sha256, embedder}")
	}
	m.Provider, m.Model, m.WeightsSHA256, m.Embedder = "openaicompat", w.Model, w.WeightsSHA256, w.Embedder
	return nil
}
