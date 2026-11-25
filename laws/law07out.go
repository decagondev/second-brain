package laws

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/notify"
)

// LAW-07 on the way out (SB-5102): whatever text an API response or a notice carries, a
// credential or a whole card number never leaves the cell, and a JSON body is still the
// same JSON with only its strings changed.

// OutFilter is what the law checks: how a JSON body and a notice leave the cell.
type OutFilter struct {
	JSON   func(b []byte) []byte
	Notice func(text string) string
}

// RealOutFilter is the real one: the API's JSON filter and the notify job's sender.
var RealOutFilter = OutFilter{
	JSON: func(b []byte) []byte { out, _ := ingest.MaskJSON(b); return out },
	Notice: func(text string) string {
		var got string
		_ = notify.Masking{Next: senderFunc(func(m notify.Message) { got = m.Text })}.Send(context.Background(), notify.Person{ID: "usr_ann"}, notify.Message{Kind: notify.KindReplies, Text: text})
		return got
	},
}

type senderFunc func(m notify.Message)

func (f senderFunc) Send(_ context.Context, _ notify.Person, m notify.Message) error {
	f(m)
	return nil
}

// outSecrets are credentials of the shapes the secret scanner knows, built from parts.
func outSecrets(r *rand.Rand) []string {
	hex := func(n int) string {
		var b strings.Builder
		for i := 0; i < n; i++ {
			b.WriteByte("ABCDEFGHJKLMNPQRSTUVWXYZ234567"[r.Intn(30)])
		}
		return b.String()
	}
	return []string{"AKIA" + hex(16), "gh" + "p_" + hex(36), "xox" + "b-" + hex(24), "sk-" + "ant-" + hex(32), "sbt_" + hex(43)}
}

func law07Out(r *rand.Rand, f OutFilter) error {
	secrets := outSecrets(r)
	words := []string{"quokka", "refund", "limit", "500", "Tuesday", `"quoted"`, "a\\b", "line\nbreak", "ünïcode"}
	var planted []string
	text := func() string {
		var parts []string
		for i := 0; i < 1+r.Intn(6); i++ {
			parts = append(parts, words[r.Intn(len(words))])
		}
		switch r.Intn(3) {
		case 0:
			s := secrets[r.Intn(len(secrets))]
			planted = append(planted, s)
			parts = append(parts, s)
		case 1:
			n := piiNumbers[r.Intn(len(piiNumbers))]
			planted = append(planted, n)
			parts = append(parts, n)
		}
		r.Shuffle(len(parts), func(i, j int) { parts[i], parts[j] = parts[j], parts[i] })
		return strings.Join(parts, " ")
	}
	body := map[string]any{"count": r.Intn(1000)}
	var list []any
	for i := 0; i < 1+r.Intn(5); i++ {
		list = append(list, map[string]any{"text": text(), "n": i})
	}
	body["claims"], body["note"] = list, text()
	in, err := json.Marshal(body)
	if err != nil {
		return err
	}
	out := f.JSON(in)
	var back map[string]any
	if err := json.Unmarshal(out, &back); err != nil {
		return fmt.Errorf("the filtered body isn't JSON: %v", err)
	}
	if fmt.Sprint(back["count"]) != fmt.Sprint(body["count"]) || len(back["claims"].([]any)) != len(list) {
		return fmt.Errorf("the filter changed more than strings: %s", out)
	}
	notice := text()
	sent := f.Notice(notice)
	for _, p := range planted {
		for what, s := range map[string]string{"an API body": string(out), "a notice": sent} {
			if strings.Contains(s, p) || strings.Contains(compact(s), compact(p)) && len(compact(p)) > 8 {
				if what == "a notice" && !strings.Contains(notice, p) {
					continue
				}
				return fmt.Errorf("%s still holds %q", what, p)
			}
		}
	}
	return nil
}
