package ask

import (
	"context"
	"errors"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/gateway"
	"example.com/second-brain/internal/core/ports"
)

// Passages when the model is down (sb-ask(3), When the model is down; SB-5602): with no
// gated model to write an answer, an ask returns the best passages the asker can open,
// cited, and says no answer was written. Nothing is labelled as an answer (LAW-14).

// NoticeModelDown is the notice of an ask answered with passages only.
const NoticeModelDown = "model_unavailable"

// ModelDownLine is what the asker reads above the passages.
const ModelDownLine = "No answer written: the model is unavailable."

// MaxDownPassages is the most passages an ask without a model shows.
const MaxDownPassages = 5

// modelDown says err is the model failing, not the budget or a refusal.
func modelDown(err error) bool {
	return errors.Is(err, ErrUnavailable) && !errors.Is(err, ErrBudget) && !errors.Is(err, gateway.ErrSecretInPrompt)
}

// passagesOnly is the response's passages when the model can't write: the first
// MaxDownPassages allowed chunks, one per document, as search results with masked snippets.
func (a Agent) passagesOnly(ctx context.Context, chunks []ports.Chunk) []Result {
	own := a.owners()
	seen := map[string]bool{}
	out := []Result{}
	for _, c := range chunks {
		k := c.Kind + ":" + c.SourceID
		if seen[k] || len(out) == MaxDownPassages || c.ID == "table-tool" {
			continue
		}
		seen[k] = true
		out = append(out, a.result(ctx, own, c))
	}
	return out
}

// composeOrPassages composes the answer, or, when no model can write it, returns no claims
// and the passages instead.
func (a Agent) composeOrPassages(ctx context.Context, compose func() ([]answer.Claim, error), chunks []ports.Chunk, resp *Response) ([]answer.Claim, error) {
	claims, err := compose()
	if err == nil || !modelDown(err) {
		return claims, err
	}
	n := NoticeModelDown
	resp.Notice = &n
	resp.Passages = a.passagesOnly(ctx, chunks)
	return []answer.Claim{}, nil
}
