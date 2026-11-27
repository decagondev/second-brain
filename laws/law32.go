package laws

import (
	"fmt"
	"math/rand"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

// LAW-32 A label never drops: anything built from a labelled document carries its highest
// label and goes nowhere that label forbids (SB-5405; sb-laws(7)). Random answers over
// random labelled chunks, through the real label and channel gate: the answer's label is
// the highest of what it cites, and it may go to a chat channel only when that is below
// confidential. A front matter sensitivity only ever raises a label.

// LabelGate is what the law checks: an answer's label from its claims and chunks, and
// whether an answer with a label may go to a channel.
type LabelGate struct {
	Label   func(claims []answer.Claim, chunks []ports.Chunk) string
	Channel func(label string) bool
}

// RealLabelGate is the real one.
var RealLabelGate = LabelGate{Label: ask.CitedLabel, Channel: ask.MayPostInChannel}

func law32(r *rand.Rand, g LabelGate) error {
	ladder := append([]string{""}, ingest.Labels...)
	for i := 0; i < 20; i++ {
		var chunks []ports.Chunk
		for j := 0; j < 1+r.Intn(6); j++ {
			chunks = append(chunks, ports.Chunk{Cite: fmt.Sprintf("docs:d%d@1", j), Label: ladder[r.Intn(len(ladder))]})
		}
		var claims []answer.Claim
		want := -1
		for j := 0; j < 1+r.Intn(4); j++ {
			var cs []answer.Cite
			for _, c := range chunks {
				if r.Intn(2) == 0 {
					cs = append(cs, answer.Cite{Cite: c.Cite})
					if k := ingest.Rank(c.Label); k > want {
						want = k
					}
				}
			}
			claims = append(claims, answer.Claim{Text: "x", Cites: cs})
		}
		got := g.Label(claims, chunks)
		if ingest.Rank(got) != want {
			return fmt.Errorf("an answer citing labels up to rank %d is labelled %q", want, got)
		}
		if may := g.Channel(got); may != (want < ingest.Rank("confidential")) {
			return fmt.Errorf("an answer labelled %q may go to a channel: %v", got, may)
		}
	}
	return nil
}
