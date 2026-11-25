package ask

import (
	"context"
	"encoding/json"
	"time"

	"example.com/second-brain/internal/core/tenancy"
)

// What the brain keeps about me (sb-ask(3), Your own data; SB-3501).

// OwnStores are the retention stores a person's own export holds.
var OwnStores = map[string]bool{"traces": true, "conversations": true, "feedback": true, "saved questions": true, "shares": true,
	"steward questions": true, "notification settings": true, "memory": true, "collections": true}

// OwnTrace is a trace as its asker's own export holds it: the answer's claims with the
// cites the asker can open now. The questions are the conversations' records.
type OwnTrace struct {
	ID     string     `json:"id"`
	At     time.Time  `json:"at"`
	Space  string     `json:"space"`
	Answer []OwnClaim `json:"answer"`
}

// OwnClaim is one claim of an exported answer.
type OwnClaim struct {
	Text  string   `json:"text"`
	Cites []string `json:"cites"`
}

// OwnExport rewrites the person's records from the retention export for s's own copy: a
// trace keeps only its question and its claims, each cite checked again as them now; a
// claim left with no cite, and every cite they can't open, is left out and counted. A
// steward question keeps only the reply's cites they can open, and never its repeat link.
// Other records pass unchanged: they are the person's own text and settings.
func (a Agent) OwnExport(ctx context.Context, s tenancy.Scope, records map[string][]json.RawMessage) (map[string][]json.RawMessage, int, error) {
	out := map[string][]json.RawMessage{}
	left := 0
	can := map[string]bool{}
	check := func(cites []string) ([]string, error) {
		var ask []string
		for _, c := range cites {
			if _, seen := can[c]; !seen {
				ask = append(ask, c)
			}
		}
		if len(ask) > 0 {
			ok, _, err := a.CheckCites(ctx, s, ask)
			if err != nil {
				return nil, err
			}
			for _, c := range ask {
				can[c] = false
			}
			for _, r := range ok {
				can[r.Cite] = true
			}
		}
		kept := []string{}
		for _, c := range cites {
			if can[c] {
				kept = append(kept, c)
			} else {
				left++
			}
		}
		return kept, nil
	}
	for name, rs := range records {
		if !OwnStores[name] {
			continue // a steward's or operator's record: the operator's export has it
		}
		out[name] = []json.RawMessage{}
		for _, raw := range rs {
			switch name {
			case "traces":
				var t Trace
				if json.Unmarshal(raw, &t) != nil {
					left++
					continue
				}
				ot := OwnTrace{ID: t.ID, At: t.At, Space: t.Space, Answer: []OwnClaim{}}
				for _, cl := range t.Response.Answer.Claims {
					var cites []string
					for _, ci := range cl.Cites {
						cites = append(cites, ci.Cite)
					}
					kept, err := check(cites)
					if err != nil {
						return nil, 0, err
					}
					if len(cites) > 0 && len(kept) == 0 {
						left++ // its text came from documents they can't open now
						continue
					}
					ot.Answer = append(ot.Answer, OwnClaim{Text: cl.Text, Cites: kept})
				}
				b, _ := json.Marshal(ot)
				out[name] = append(out[name], b)
			case "steward questions":
				var q StewardQuestion
				if json.Unmarshal(raw, &q) != nil {
					left++
					continue
				}
				q.Like, q.Also = "", ""
				if q.Reply != nil {
					kept, err := check(q.Reply.Cites)
					if err != nil {
						return nil, 0, err
					}
					q.Reply.Cites = kept
					q.Reply.By = "" // the steward, never named to the asker
				}
				b, _ := json.Marshal(q)
				out[name] = append(out[name], b)
			case "feedback":
				var f Feedback
				if json.Unmarshal(raw, &f) != nil {
					left++
					continue
				}
				for _, c := range []*string{&f.WrongCitation, &f.OutdatedCitation} {
					if *c != "" {
						kept, err := check([]string{*c})
						if err != nil {
							return nil, 0, err
						}
						if len(kept) == 0 {
							*c = ""
						}
					}
				}
				b, _ := json.Marshal(f)
				out[name] = append(out[name], b)
			default:
				out[name] = append(out[name], raw)
			}
		}
	}
	return out, left, nil
}
