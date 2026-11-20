package laws

import (
	"math/rand"
	"testing"

	"example.com/second-brain/internal/adapters/connectors/github"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

func TestLAW03_GitHubUnknownIsNeverAllow(t *testing.T) {
	w := NewGitHubWorld()
	Check(t, "LAW-03", 30, func(r *rand.Rand) error { return law03Connector(r, w) })
}

func TestLAW01_GitHubAccessIsLive(t *testing.T) {
	Check(t, "LAW-01", 20, func(r *rand.Rand) error { return law01GitHub(r, liveACL) })
}

func TestLAW07_IngestNeverStoresASecret(t *testing.T) {
	Check(t, "LAW-07", runs, func(r *rand.Rand) error { return law07Ingest(r, ingest.SecretIn) })
}

// The GitHub law tests catch a check that keeps its first answer instead of asking
// GitHub each time, and an ingest without its secret scanner.
func TestSelf_GitHubMutationsAreCaught(t *testing.T) {
	stale := func() LiveCheck {
		cached := map[string]ports.Verdict{}
		return func(c *github.Connector, ref ports.SourceRef) ports.Verdict {
			if v, ok := cached[ref.ID]; ok {
				return v
			}
			v := liveACL(c, ref)
			cached[ref.ID] = v
			return v
		}
	}
	if !violates(20, func(r *rand.Rand) error { return law01GitHub(r, stale()) }) {
		t.Error("LAW-01 (github): a check that keeps its first answer was not caught")
	}
	if !violates(runs, func(r *rand.Rand) error { return law07Ingest(r, func(string) string { return "" }) }) {
		t.Error("LAW-07 (ingest): an ingest without the secret scanner was not caught")
	}
}
