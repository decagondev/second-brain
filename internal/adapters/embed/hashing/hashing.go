// Package hashing is a deterministic local embedder (sb-ingest(7)): feature hashing of
// lower-cased words into 64 dimensions, L2-normalised. No network, no model; it stands in
// until the embedding provider is chosen (Q-04).
package hashing

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
	"unicode"

	"example.com/second-brain/internal/core/ports"
)

// Dim is the vector size.
const Dim = 64

// Embedder is stateless.
type Embedder struct{}

var _ ports.Embedder = Embedder{}

func (Embedder) Model() string { return "local-hashing-64" }

func (Embedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		v := make([]float32, Dim)
		for _, w := range strings.FieldsFunc(strings.ToLower(t), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
			h := fnv.New32a()
			_, _ = h.Write([]byte(w))
			v[h.Sum32()%Dim]++
		}
		var n float64
		for _, x := range v {
			n += float64(x) * float64(x)
		}
		if n > 0 {
			n = math.Sqrt(n)
			for j := range v {
				v[j] = float32(float64(v[j]) / n)
			}
		}
		out[i] = v
	}
	return out, nil
}
