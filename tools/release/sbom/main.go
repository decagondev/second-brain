// Command sbom writes an SPDX 2.3 JSON SBOM for a release (sb-release(7), Builds; SB-504):
// one package per Go module in the binary's build info, and one per npm package in the
// web lockfile, each with its hash.
//
//	go run ./tools/release/sbom --binary bin/sb --npm web/package-lock.json --name sb --out sbom.spdx.json
package main

import (
	"debug/buildinfo"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

type checksum struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"checksumValue"`
}

type pkg struct {
	SPDXID           string     `json:"SPDXID"`
	Name             string     `json:"name"`
	Version          string     `json:"versionInfo"`
	Download         string     `json:"downloadLocation"`
	FilesAnalyzed    bool       `json:"filesAnalyzed"`
	Checksums        []checksum `json:"checksums,omitempty"`
	ExternalRefs     []ref      `json:"externalRefs,omitempty"`
	LicenseConcluded string     `json:"licenseConcluded"`
	CopyrightText    string     `json:"copyrightText"`
}

type ref struct {
	Category string `json:"referenceCategory"`
	Type     string `json:"referenceType"`
	Locator  string `json:"referenceLocator"`
}

type rel struct {
	From string `json:"spdxElementId"`
	Type string `json:"relationshipType"`
	To   string `json:"relatedSpdxElement"`
}

type doc struct {
	SPDXVersion  string `json:"spdxVersion"`
	DataLicense  string `json:"dataLicense"`
	SPDXID       string `json:"SPDXID"`
	Name         string `json:"name"`
	Namespace    string `json:"documentNamespace"`
	CreationInfo struct {
		Created  string   `json:"created"`
		Creators []string `json:"creators"`
	} `json:"creationInfo"`
	Packages      []pkg `json:"packages"`
	Relationships []rel `json:"relationships"`
}

var idSafe = regexp.MustCompile(`[^A-Za-z0-9.-]+`)

func spdxID(prefix, name, version string) string {
	return "SPDXRef-" + prefix + "-" + strings.Trim(idSafe.ReplaceAllString(name+"-"+version, "-"), "-")
}

// h1 turns a Go module hash ("h1:<base64 sha256>") into a SHA256 checksum.
func h1(sum string) []checksum {
	b64, ok := strings.CutPrefix(sum, "h1:")
	if !ok {
		return nil
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(raw) != 32 {
		return nil
	}
	return []checksum{{Algorithm: "SHA256", Value: hex.EncodeToString(raw)}}
}

// integrity turns an npm integrity ("sha512-<base64>") into a SHA512 checksum.
func integrity(s string) []checksum {
	b64, ok := strings.CutPrefix(s, "sha512-")
	if !ok {
		return nil
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil
	}
	return []checksum{{Algorithm: "SHA512", Value: hex.EncodeToString(raw)}}
}

// goSum maps "path version" to its h1 hash, for vendored builds whose build info has none.
func goSum(path string) map[string]string {
	out := map[string]string{}
	b, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for _, l := range strings.Split(string(b), "\n") {
		f := strings.Fields(l)
		if len(f) == 3 && !strings.HasSuffix(f[1], "/go.mod") {
			out[f[0]+" "+f[1]] = f[2]
		}
	}
	return out
}

func build(binary, npm, name, created string) (doc, error) {
	return buildWith(binary, npm, name, created, nil)
}

func buildWith(binary, npm, name, created string, sums map[string]string) (doc, error) {
	var d doc
	d.SPDXVersion, d.DataLicense, d.SPDXID, d.Name = "SPDX-2.3", "CC0-1.0", "SPDXRef-DOCUMENT", name
	d.CreationInfo.Created = created
	d.CreationInfo.Creators = []string{"Tool: second-brain/tools/release/sbom"}
	root := pkg{SPDXID: "SPDXRef-" + idSafe.ReplaceAllString(name, "-"), Name: name, Version: "release", Download: "NOASSERTION", LicenseConcluded: "NOASSERTION", CopyrightText: "NOASSERTION"}
	d.Packages = append(d.Packages, root)
	d.Relationships = append(d.Relationships, rel{From: "SPDXRef-DOCUMENT", Type: "DESCRIBES", To: root.SPDXID})
	if binary != "" {
		bi, err := buildinfo.ReadFile(binary)
		if err != nil {
			return d, fmt.Errorf("build info: %w", err)
		}
		d.Namespace = "https://second-brain.invalid/spdx/" + name + "-" + bi.GoVersion
		var mods []*struct{ Path, Version, Sum string }
		for _, m := range bi.Deps {
			mm := m
			if m.Replace != nil {
				mm = m.Replace
			}
			sum := mm.Sum
			if sum == "" {
				sum = sums[mm.Path+" "+mm.Version] // vendored: the build info has no hash
			}
			mods = append(mods, &struct{ Path, Version, Sum string }{mm.Path, mm.Version, sum})
		}
		sort.Slice(mods, func(i, j int) bool { return mods[i].Path < mods[j].Path })
		for _, m := range mods {
			p := pkg{SPDXID: spdxID("go", m.Path, m.Version), Name: m.Path, Version: m.Version, Download: "https://proxy.golang.org/" + m.Path + "/@v/" + m.Version + ".zip",
				Checksums: h1(m.Sum), LicenseConcluded: "NOASSERTION", CopyrightText: "NOASSERTION",
				ExternalRefs: []ref{{Category: "PACKAGE-MANAGER", Type: "purl", Locator: "pkg:golang/" + m.Path + "@" + m.Version}}}
			d.Packages = append(d.Packages, p)
			d.Relationships = append(d.Relationships, rel{From: root.SPDXID, Type: "DEPENDS_ON", To: p.SPDXID})
		}
	}
	if npm != "" {
		raw, err := os.ReadFile(npm)
		if err != nil {
			return d, err
		}
		var lock struct {
			Packages map[string]struct {
				Version   string `json:"version"`
				Integrity string `json:"integrity"`
				Resolved  string `json:"resolved"`
			} `json:"packages"`
		}
		if err := json.Unmarshal(raw, &lock); err != nil {
			return d, fmt.Errorf("%s: %w", npm, err)
		}
		var names []string
		for k := range lock.Packages {
			if k != "" {
				names = append(names, k)
			}
		}
		sort.Strings(names)
		for _, k := range names {
			e := lock.Packages[k]
			n := k[strings.LastIndex(k, "node_modules/")+len("node_modules/"):]
			dl := e.Resolved
			if dl == "" {
				dl = "NOASSERTION"
			}
			p := pkg{SPDXID: spdxID("npm", n, e.Version), Name: n, Version: e.Version, Download: dl, Checksums: integrity(e.Integrity),
				LicenseConcluded: "NOASSERTION", CopyrightText: "NOASSERTION",
				ExternalRefs: []ref{{Category: "PACKAGE-MANAGER", Type: "purl", Locator: "pkg:npm/" + strings.ReplaceAll(n, "@", "%40") + "@" + e.Version}}}
			d.Packages = append(d.Packages, p)
			d.Relationships = append(d.Relationships, rel{From: root.SPDXID, Type: "DEPENDS_ON", To: p.SPDXID})
		}
	}
	if d.Namespace == "" {
		d.Namespace = "https://second-brain.invalid/spdx/" + name
	}
	// Duplicate ids (the same npm package at two paths) are merged: SPDX ids must be unique.
	seen := map[string]bool{}
	var ps []pkg
	for _, p := range d.Packages {
		if !seen[p.SPDXID] {
			seen[p.SPDXID] = true
			ps = append(ps, p)
		}
	}
	d.Packages = ps
	rs := map[rel]bool{}
	var out []rel
	for _, r := range d.Relationships {
		if !rs[r] {
			rs[r] = true
			out = append(out, r)
		}
	}
	d.Relationships = out
	return d, nil
}

func main() {
	binary := flag.String("binary", "", "Go binary to read build info from")
	npm := flag.String("npm", "", "npm package-lock.json")
	gosum := flag.String("gosum", "go.sum", "go.sum, for vendored modules' hashes")
	name := flag.String("name", "second-brain", "document name")
	created := flag.String("created", "", "creation time, RFC 3339 (fixed for reproducible SBOMs)")
	out := flag.String("out", "sbom.spdx.json", "where to write it")
	flag.Parse()
	if *created == "" {
		fmt.Fprintln(os.Stderr, "sbom: --created is required, so the SBOM is reproducible")
		os.Exit(2)
	}
	d, err := buildWith(*binary, *npm, *name, *created, goSum(*gosum))
	if err != nil {
		fmt.Fprintln(os.Stderr, "sbom:", err)
		os.Exit(1)
	}
	b, _ := json.MarshalIndent(d, "", "  ")
	if err := os.WriteFile(*out, append(b, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "sbom:", err)
		os.Exit(1)
	}
}
