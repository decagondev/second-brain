package laws

import (
	"example.com/second-brain/internal/core/drift"
	"example.com/second-brain/internal/core/ports"
)

// The real drift detector (SB-510).
func init() {
	NewDriftDetector = func() DriftDetector { return detectDrift(false) }
}

// detectDrift wraps drift.Detect; attrsOnly is a mutation that ignores resources present
// on one side only.
func detectDrift(attrsOnly bool) DriftDetector {
	return func(applied, live []ports.Resource) []string {
		var out []string
		for _, f := range drift.Detect(applied, live) {
			if attrsOnly && f.Present != "" {
				continue
			}
			out = append(out, f.Key())
		}
		return out
	}
}
