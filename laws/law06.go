package laws

import (
	"fmt"
	"math/rand"
)

// LAW-06 (SB-503, SB-506): only signed, gated builds run. A cell admits a release only if
// its bundle is signed by the release key, its id matches its contents, and it passed the
// eval gate; any change to a prompt, policy, tool, model or artefact is a different id
// and needs its own signature and gate.

// ReleaseWorld builds bundles, signs and gates them, and asks a cell to admit one.
type ReleaseWorld interface {
	// Build makes a bundle; change names one field to alter ("" for none).
	Build(change string) (id string, err error)
	Sign(id string) error
	Gate(id string, pass bool) error
	// Admit asks a cell to run id; tamper names a field changed after signing ("" for none).
	Admit(id, tamper string) (admitted bool, err error)
}

// NewReleaseWorld builds the system under test.
var NewReleaseWorld = func() ReleaseWorld { return pendingRelease{} }

type pendingRelease struct{}

func (pendingRelease) Build(string) (string, error) {
	return "", fmt.Errorf("%w: release bundles arrive with SB-503 and SB-506", ErrPending)
}
func (pendingRelease) Sign(string) error                  { return nil }
func (pendingRelease) Gate(string, bool) error            { return nil }
func (pendingRelease) Admit(string, string) (bool, error) { return false, nil }

var bundleFields = []string{"prompt", "policy", "tool", "model", "code", "gate"}

func law06(r *rand.Rand, w ReleaseWorld) error {
	base, err := w.Build("")
	if err != nil {
		return err
	}
	seen := map[string]string{base: ""}
	for _, f := range bundleFields {
		id, err := w.Build(f)
		if err != nil {
			return err
		}
		if prev, dup := seen[id]; dup {
			return fmt.Errorf("changing %q kept release id %s (same as change %q)", f, id, prev)
		}
		seen[id] = f
	}
	for i := 0; i < 6; i++ {
		id, err := w.Build(bundleFields[r.Intn(len(bundleFields))])
		if err != nil {
			return err
		}
		signed, gated := r.Intn(3) != 0, r.Intn(3) != 0
		if signed {
			if err := w.Sign(id); err != nil {
				return err
			}
		}
		if err := w.Gate(id, gated); err != nil {
			return err
		}
		tamper := ""
		if r.Intn(4) == 0 {
			tamper = bundleFields[r.Intn(len(bundleFields))]
		}
		ok, err := w.Admit(id, tamper)
		if err != nil {
			return err
		}
		good := signed && gated && tamper == ""
		if ok && !good {
			return fmt.Errorf("a cell admitted %s (signed=%v gated=%v tampered=%q)", id, signed, gated, tamper)
		}
		if !ok && good {
			return fmt.Errorf("a cell refused %s, which is signed, gated and untouched", id)
		}
	}
	return nil
}
