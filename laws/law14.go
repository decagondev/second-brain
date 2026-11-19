package laws

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ports"
)

// LAW-14 (SB-304): every claim is cited or labelled uncited, by code. Whatever the model
// returns (valid labels, made-up labels, labels of refused chunks, garbage), each claim
// that comes out either cites only chunks from this answer's allowed context or is
// marked uncited with no citations. LAW-01 follows: no citation outside the context.

// ComposedClaim is one claim as the composer emits it.
type ComposedClaim struct {
	Text    string
	Cites   []string // cites (kind:id@version)
	Uncited bool
}

// Composer turns raw model output into claims, given the allowed context: label -> cite.
type Composer func(raw string, context map[string]string) ([]ComposedClaim, error)

// NewComposer builds the system under test: answer.Compose (SB-304).
var NewComposer = func() Composer {
	return func(raw string, labels map[string]string) ([]ComposedClaim, error) {
		var order []string
		for l := range labels {
			order = append(order, l)
		}
		sort.Strings(order)
		ctx := answer.Context{Labels: map[string]ports.Chunk{}}
		for _, l := range order {
			ch := ports.Chunk{ID: "chk_" + l, Cite: labels[l]}
			ctx.Labels[l] = ch
			ctx.Order = append(ctx.Order, l)
		}
		res := answer.Compose(raw, ctx, nil)
		out := make([]ComposedClaim, len(res.Claims))
		for i, c := range res.Claims {
			out[i] = ComposedClaim{Text: c.Text, Uncited: c.Uncited}
			for _, ci := range c.Cites {
				out[i].Cites = append(out[i].Cites, ci.Cite)
			}
		}
		return out, nil
	}
}

func law14(r *rand.Rand, compose Composer) error {
	ctx := map[string]string{}
	for i := 1; i <= 1+r.Intn(5); i++ {
		ctx[fmt.Sprintf("C%d", i)] = fmt.Sprintf("docs:d%d@%d", i, 1+r.Intn(9))
	}
	allowed := map[string]bool{}
	for _, c := range ctx {
		allowed[c] = true
	}
	var raw string
	if r.Intn(6) == 0 {
		raw = []string{"not json at all", `{"claims": "nope"}`, `[1,2,3]`, ``}[r.Intn(4)]
	} else {
		type claim struct {
			Text  string   `json:"text"`
			Cites []string `json:"cites"`
		}
		var cs []claim
		for i := 0; i < 1+r.Intn(5); i++ {
			var labels []string
			for j := 0; j < r.Intn(4); j++ {
				switch r.Intn(4) {
				case 0:
					labels = append(labels, fmt.Sprintf("C%d", 50+r.Intn(5))) // not in context
				case 1:
					labels = append(labels, "docs:secret@1") // a raw cite the model made up
				default:
					labels = append(labels, fmt.Sprintf("C%d", 1+r.Intn(len(ctx))))
				}
			}
			cs = append(cs, claim{Text: fmt.Sprintf("claim %d", i), Cites: labels})
		}
		b, _ := json.Marshal(map[string]any{"claims": cs})
		raw = string(b)
	}
	out, err := compose(raw, ctx)
	if err != nil {
		return err
	}
	if len(out) == 0 {
		return fmt.Errorf("no claims from %q", raw)
	}
	for _, c := range out {
		switch {
		case c.Uncited && len(c.Cites) > 0:
			return fmt.Errorf("claim %q is uncited but cites %v", c.Text, c.Cites)
		case !c.Uncited && len(c.Cites) == 0:
			return fmt.Errorf("claim %q has no citation and isn't labelled uncited (raw %q)", c.Text, raw)
		}
		for _, cite := range c.Cites {
			if !allowed[cite] {
				return fmt.Errorf("claim %q cites %s, which wasn't in the allowed context (raw %q)", c.Text, cite, raw)
			}
		}
	}
	return nil
}
