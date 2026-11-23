package laws

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

// LAW-07 and LAW-13 for personal data (SB-1904): ingest flags every chunk that holds a
// valid IBAN, card number or national identity number (SB-2207), the same flags again after a rebuild, and nothing any
// reader is shown, the model's context, a snippet, or an answer that echoes the number,
// holds the whole number: at most its last four characters.

// PIIMasker is what the law checks: how a chunk's text is shown, and how an answer's
// text is cleaned (ingest.Shown, ingest.MaskPII).
type PIIMasker struct {
	Shown  func(text string, flags []string) string
	Answer func(raw string, ctx answer.Context) string
}

// RealPIIMasker is the real one: the model's context and the composed answer.
var RealPIIMasker = PIIMasker{
	Shown: ingest.Shown,
	Answer: func(raw string, ctx answer.Context) string {
		var b strings.Builder
		for _, c := range answer.Compose(raw, ctx, nil).Claims {
			b.WriteString(c.Text + "\n")
		}
		return b.String()
	},
}

// Numbers that are valid: IBANs (mod-97), cards (Luhn, an issuer prefix) and national
// identity numbers (their format rules and check digits).
var piiNumbers = []string{
	"DE89 3704 0044 0532 0130 00", "GB82WEST12345698765432", "FR14 2004 1010 0505 0001 3M02 606", "NL91ABNA0417164300",
	"4111 1111 1111 1111", "5555-5555-5555-4444", "378282246310005", "6011 1111 1111 1117", "4012888888881881",
	// National identity numbers (SB-2207): UK NINO, US SSN, Dutch BSN, German Steuer-IdNr.
	"AB123456C", "JG 10 37 59 A", "123-45-6789", "078 05 1121", "123456782", "1234.56.782", "86095742719", "86 095 742 719",
}

func compact(s string) string { return strings.NewReplacer(" ", "", "-", "", ".", "").Replace(s) }

func law07PII(r *rand.Rand, m PIIMasker) error {
	w := NewIngestWorld().(*ingestWorld)
	var docs []SourceDoc
	holds := map[string]string{} // doc id -> the number in it
	for i := 0; i < 2+r.Intn(5); i++ {
		id := fmt.Sprintf("d%d", i)
		words := strings.Repeat("supplier payment terms ", 1+r.Intn(120)) // long enough, sometimes, to be cut
		text := words
		if r.Intn(2) == 0 {
			n := piiNumbers[r.Intn(len(piiNumbers))]
			at := r.Intn(len(words)/23+1) * 23
			text = words[:at] + "Pay to " + n + " now. " + words[at:]
			holds[id] = n
		}
		docs = append(docs, SourceDoc{ID: id, Text: text, Viewers: []string{"ann@acme.example"}})
	}
	if err := w.Load(docs); err != nil {
		return err
	}
	if err := w.IngestAll(); err != nil {
		return err
	}
	ctx := context.Background()
	chunks, err := w.idx.All(ctx, "acme-logistics")
	if err != nil {
		return err
	}
	flags := func(cs []ports.Chunk) string {
		var out []string
		for _, c := range cs {
			out = append(out, c.ID+"="+strings.Join(c.Flags, ","))
		}
		return strings.Join(out, " ")
	}
	before := flags(chunks)
	flaggedDocs := map[string]bool{}
	for _, c := range chunks {
		n := holds[c.SourceID]
		hasFlag := contains(c.Flags, ingest.FlagPersonal)
		if hasFlag {
			flaggedDocs[c.SourceID] = true
		}
		if n == "" && hasFlag {
			return fmt.Errorf("chunk %s has no number and is flagged personal_data", c.ID)
		}
		whole := strings.Contains(compact(c.Text), compact(n))
		if n != "" && whole && !hasFlag {
			return fmt.Errorf("chunk %s holds %s and isn't flagged", c.ID, n)
		}
		if n != "" {
			shown := m.Shown(c.Text, c.Flags)
			if leaks(shown, n) {
				return fmt.Errorf("chunk %s is shown with more than the last four of %s: %q", c.ID, n, shown)
			}
		}
	}
	for id := range holds {
		if !flaggedDocs[id] {
			return fmt.Errorf("document %s holds %s and no chunk of it is flagged", id, holds[id])
		}
	}
	actx := answer.NewContext(chunks)
	data := actx.Data()
	for _, n := range holds {
		if leaks(data, n) {
			return fmt.Errorf("the model's context holds %s", n)
		}
		raw, _ := json.Marshal(map[string]any{"claims": []map[string]any{{"text": "The account is " + n + ".", "cites": []string{"C1"}}}})
		if out := m.Answer(string(raw), actx); leaks(out, n) {
			return fmt.Errorf("an answer echoing %s keeps it: %q", n, out)
		}
	}
	// LAW-13: the flags are derived, so a rebuild gives the same ones.
	if err := w.Drop(); err != nil {
		return err
	}
	if err := w.Rebuild(); err != nil {
		return err
	}
	again, err := w.idx.All(ctx, "acme-logistics")
	if err != nil {
		return err
	}
	if after := flags(again); after != before {
		return fmt.Errorf("a rebuild changed the flags:\n%s\n%s", before, after)
	}
	return nil
}

// leaks reports whether s holds more of the number than its last four characters: any
// run of five of its characters in order, separators aside.
func leaks(s, number string) bool {
	c, have := compact(number), compact(s)
	for i := 0; i+5 <= len(c); i++ {
		if strings.Contains(have, c[i:i+5]) && i+5 <= len(c) {
			return true
		}
	}
	return false
}
