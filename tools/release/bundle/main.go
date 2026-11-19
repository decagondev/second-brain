// Command bundle builds a release's bundle.json (sb-release(7), Bundles; SB-503): the
// artefacts' digests, the bundled prompts, the tool registry's schemas, the platform
// policies, the pinned model and the eval gate, sealed with its release id.
//
//	go run ./tools/release/bundle --artefact sb=bin/sb [--artefact web=web/dist.tar] \
//	    --created 2027-03-02 --out bundle.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/prompts"
	"example.com/second-brain/internal/core/release"
	"example.com/second-brain/internal/core/workflow"
)

type list []string

func (l *list) String() string     { return strings.Join(*l, ",") }
func (l *list) Set(v string) error { *l = append(*l, v); return nil }

func main() {
	var arts list
	flag.Var(&arts, "artefact", "name=path of a built artefact (repeatable)")
	root := flag.String("root", ".", "repository root")
	created := flag.String("created", "", "build date, YYYY-MM-DD")
	out := flag.String("out", "bundle.json", "where to write the manifest")
	flag.Parse()
	m, err := build(*root, arts, *created)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bundle:", err)
		os.Exit(1)
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(*out, append(b, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "bundle:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, m.ID)
}

func build(root string, arts []string, created string) (release.Manifest, error) {
	b := release.Bundle{Code: map[string]string{}, Prompts: map[string]string{}, Policies: map[string]string{}, Created: created}
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
	gate, err := os.ReadFile(filepath.Join(root, "evals", "gate.yaml"))
	if err != nil {
		return release.Manifest{}, err
	}
	b.Gate = release.SHA256(gate)
	return b.Seal()
}
