package laws

import (
	"math/rand"
	"testing"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
)

// SB-5405: a gate that takes the first cite's label, or lets everything into a channel, is
// caught.
func TestSelfLabelGateIsCaught(t *testing.T) {
	failsSomewhere(t, "the first cite's label", func(r *rand.Rand) error {
		return law32(r, LabelGate{Channel: ask.MayPostInChannel, Label: func(cl []answer.Claim, cs []ports.Chunk) string {
			for _, c := range cl {
				for _, ci := range c.Cites {
					for _, ch := range cs {
						if ch.Cite == ci.Cite {
							return ch.Label
						}
					}
				}
			}
			return ""
		}})
	})
	failsSomewhere(t, "every label to a channel", func(r *rand.Rand) error {
		return law32(r, LabelGate{Label: ask.CitedLabel, Channel: func(string) bool { return true }})
	})
}
