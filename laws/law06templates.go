package laws

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"example.com/second-brain/internal/core/manifest"
)

// LAW-06 for the template catalogue (SB-1107): a release passes the gate only if every
// template's suite passed, so the gate must name each template's suite with zero failures
// allowed, and the suite must exist. A gate that leaves one out would let a template change
// reach tenants untested.

func repoRoot() string {
	_, here, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(here), "..")
}

// GateFile is the release gate's text (evals/gate.yaml in the real system).
type GateFile func() (string, error)

func realGate() (string, error) {
	b, err := os.ReadFile(filepath.Join(repoRoot(), "evals", "gate.yaml"))
	return string(b), err
}

var gateLine = regexp.MustCompile(`(?m)^([a-z0-9-]+):\s*\{\s*failures_max:\s*0\s*\}`)

func law06Templates(r *rand.Rand, gate GateFile) error {
	text, err := gate()
	if err != nil {
		return err
	}
	gated := map[string]bool{}
	for _, m := range gateLine.FindAllStringSubmatch(text, -1) {
		gated[m[1]] = true
	}
	names := make([]string, 0, len(manifest.Templates))
	for n := range manifest.Templates {
		names = append(names, n)
	}
	// Check the templates in a random order, so no single one is the only one looked at.
	r.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	for _, n := range names {
		t := manifest.Templates[n]
		if n == "custom" {
			continue // custom has no department of its own; its spaces bring their own suites
		}
		if t.Suite == "" {
			return fmt.Errorf("template %s ships no suite", n)
		}
		if !gated[t.Suite] {
			return fmt.Errorf("the gate doesn't hold template %s's suite %s to zero failures", n, t.Suite)
		}
		if _, err := os.Stat(filepath.Join(repoRoot(), "evals", "suites", t.Suite+".yaml")); err != nil {
			return fmt.Errorf("template %s's suite %s doesn't exist", n, t.Suite)
		}
	}
	return nil
}

// dropOne is a gate with one template suite's line removed.
func dropOne(r *rand.Rand) GateFile {
	return func() (string, error) {
		text, err := realGate()
		if err != nil {
			return "", err
		}
		var lines []string
		for _, l := range strings.Split(text, "\n") {
			if strings.HasPrefix(l, "template-") {
				lines = append(lines, l)
			}
		}
		if len(lines) == 0 {
			return text, nil
		}
		return strings.Replace(text, lines[r.Intn(len(lines))]+"\n", "", 1), nil
	}
}
