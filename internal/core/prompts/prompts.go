// Package prompts is the bundled prompt set (sb-agents(7), MODEL GATEWAY). Prompts are
// loaded by name with their SHA-256, which the release bundle pins (SB-503); nothing
// outside this set can become a system prompt.
package prompts

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed set/*.md
var set embed.FS

// Prompt is one bundled prompt.
type Prompt struct {
	Name   string
	Text   string
	SHA256 string
}

// Get loads a prompt by name ("answer" is set/answer.md).
func Get(name string) (Prompt, error) {
	if strings.ContainsAny(name, `/\.`) {
		return Prompt{}, fmt.Errorf("prompts: bad name %q", name)
	}
	b, err := set.ReadFile("set/" + name + ".md")
	if err != nil {
		return Prompt{}, fmt.Errorf("prompts: no prompt %q in the bundled set", name)
	}
	sum := sha256.Sum256(b)
	return Prompt{Name: name, Text: string(b), SHA256: hex.EncodeToString(sum[:])}, nil
}

// All lists every bundled prompt, sorted by name, for the release manifest.
func All() ([]Prompt, error) {
	entries, err := fs.ReadDir(set, "set")
	if err != nil {
		return nil, err
	}
	var out []Prompt
	for _, e := range entries {
		p, err := Get(strings.TrimSuffix(e.Name(), ".md"))
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
