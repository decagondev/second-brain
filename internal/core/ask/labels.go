package ask

import (
	"context"
	"fmt"
	"strings"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

// Labels travel (sb-citations(7), Labels; SB-5405, LAW-32): anything built from labelled
// documents carries the highest label among what it cites, and content labelled
// confidential or above goes only to a person, never to a chat channel.

// CitedLabel is the highest label among the chunks claims cite; "" when no cite is labelled.
func CitedLabel(claims []answer.Claim, chunks []ports.Chunk) string {
	byCite := map[string]string{}
	for _, c := range chunks {
		byCite[c.Cite] = ingest.Highest(byCite[c.Cite], c.Label)
	}
	best := ""
	for _, cl := range claims {
		for _, ci := range cl.Cites {
			best = ingest.Highest(best, byCite[ci.Cite])
		}
	}
	return best
}

// MayPostInChannel says an answer with this label may be posted to a chat channel (LAW-32).
func MayPostInChannel(label string) bool { return !ingest.PersonOnly(label) }

// LabelLine is how an answer's label reads in a chat reply or a download; "" for none or
// for public and internal, which every member may read anyway.
func LabelLine(label string) string {
	if !ingest.PersonOnly(label) {
		return ""
	}
	return strings.ToUpper(label[:1]) + label[1:] + ": cites " + label + " documents. Don't forward it."
}

// citesLabel is the highest label among cites (KIND:ID@VERSION), read from the index; ""
// when the index can't say, and restricted when a cite is gone: unknown never drops a label.
func (a Agent) citesLabel(ctx context.Context, cites []string) string {
	if len(cites) == 0 {
		return ""
	}
	lk, ok := a.Index.(ports.CiteLookup)
	if !ok {
		return ingest.LabelRestricted
	}
	chunks, err := lk.ChunksOf(ctx, a.Tenant, cites)
	if err != nil {
		return ingest.LabelRestricted
	}
	best := ""
	found := map[string]bool{}
	for _, c := range chunks {
		found[c.Cite] = true
		best = ingest.Highest(best, c.Label)
	}
	for _, c := range cites {
		if !found[c] {
			return ingest.LabelRestricted
		}
	}
	return best
}

// ChannelLabelOK refuses a message to a channel drafted from cites labelled confidential or
// above (LAW-32): it goes to a person instead.
func (x Stewards) ChannelLabelOK(ctx context.Context, cites []string) error {
	if l := x.Agent.citesLabel(ctx, cites); !MayPostInChannel(l) {
		return fmt.Errorf("%w: this draft cites %s documents, so it can't be posted in a channel; send it to a person instead", ErrInvalid, l)
	}
	return nil
}
