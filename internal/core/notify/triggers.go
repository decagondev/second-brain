package notify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Agents when something happens (sb-agents(7), Scheduled and triggered agents; SB-4006):
// only two triggers. A person SCIM adds to a space's group gets one welcome brief, and a
// new transcript in a space drafts its decisions to the space's stewards.

// Welcome is one new member's welcome brief, its text resolved as them, now.
type Welcome struct {
	ID, User, Space, Text string
}

// NewDoc is a transcript first seen in a space, by space key and KIND:ID.
type NewDoc struct {
	Space, Ref, Title string
	// Closed says this version of an incident's ticket is closed (SB-4604).
	Closed bool
}

// triggers plans the welcome briefs and runs the decisions drafts of the transcripts seen
// in [from, now). A welcome is marked done once the run has sent; a transcript is marked
// when its draft ran, so each runs once.
func (j Job) triggers(ctx context.Context, from, now time.Time) ([]outgoing, []func() error, error) {
	var out []outgoing
	var marks []func() error
	if j.Welcomes != nil {
		ws, err := j.Welcomes(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("notify: welcomes: %w", err)
		}
		for _, w := range ws {
			w := w
			p, ok, err := j.person(ctx, w.User)
			if err != nil {
				return nil, nil, err
			}
			if ok && w.Text != "" && j.Send != nil {
				out = append(out, outgoing{p, Message{Kind: KindWelcome, Text: w.Text, Link: j.tab("answer")}})
			}
			marks = append(marks, func() error { return j.Welcomed(ctx, w) })
		}
	}
	pol, polMarks, err := j.policies(ctx, now)
	if err != nil {
		return nil, nil, err
	}
	out, marks = append(out, pol...), append(marks, polMarks...)
	lim, limMarks, err := j.nearLimit(ctx, now)
	if err != nil {
		return nil, nil, err
	}
	out, marks = append(out, lim...), append(marks, limMarks...)
	if j.SourceChanged != nil {
		if err := j.SourceChanged(ctx, from, now); err != nil {
			return nil, nil, fmt.Errorf("notify: sources changed: %w", err)
		}
	}
	if j.Transcripts == nil || j.RunDecisions == nil {
		return out, marks, nil
	}
	docs, err := j.Transcripts(ctx, from, now)
	if err != nil {
		return nil, nil, fmt.Errorf("notify: transcripts: %w", err)
	}
	for _, d := range docs {
		sum := sha256.Sum256([]byte(d.Space + "|" + d.Ref))
		key := "decisions-" + hex.EncodeToString(sum[:8])
		if _, done, err := j.Clock.Last(ctx, key); err != nil || done {
			if err != nil {
				return nil, nil, err
			}
			continue
		}
		if _, err := j.RunDecisions(ctx, d); err != nil {
			return nil, nil, fmt.Errorf("notify: decisions of a transcript: %w", err)
		}
		if err := j.Clock.Mark(ctx, key, now); err != nil {
			return nil, nil, err
		}
	}
	return out, marks, nil
}

// nearLimit tells each person at 80% of their personal limit once a month, marked after the
// run sends (SB-5003). The marker's name carries no id.
func (j Job) nearLimit(ctx context.Context, now time.Time) ([]outgoing, []func() error, error) {
	if j.NearLimit == nil {
		return nil, nil, nil
	}
	users, err := j.NearLimit(ctx, now)
	if err != nil {
		return nil, nil, fmt.Errorf("notify: limits: %w", err)
	}
	var out []outgoing
	var marks []func() error
	month := now.UTC().Format("2006-01")
	for _, u := range users {
		sum := sha256.Sum256([]byte(u))
		key := "limit-" + hex.EncodeToString(sum[:8]) + "-" + month
		if _, done, err := j.Clock.Last(ctx, key); err != nil || done {
			if err != nil {
				return nil, nil, err
			}
			continue
		}
		p, ok, err := j.person(ctx, u)
		if err != nil {
			return nil, nil, err
		}
		if ok && j.Send != nil {
			out = append(out, outgoing{p, Message{Kind: KindLimit, Text: "You have used 80% of your monthly AI limit. At 100% your own asks and research use the economy model until the month ends; anything you review keeps working", Link: j.tab("answer")}})
		}
		marks = append(marks, func() error { return j.Clock.Mark(ctx, key, now) })
	}
	return out, marks, nil
}

// policies tells each policy version's readers once, marked after the run sends.
func (j Job) policies(ctx context.Context, now time.Time) ([]outgoing, []func() error, error) {
	if j.PolicyNotices == nil {
		return nil, nil, nil
	}
	ns, err := j.PolicyNotices(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("notify: policies: %w", err)
	}
	var out []outgoing
	var marks []func() error
	for _, n := range ns {
		key := "policy-" + n.Key
		if _, done, err := j.Clock.Last(ctx, key); err != nil || done {
			if err != nil {
				return nil, nil, err
			}
			continue
		}
		for _, id := range n.To {
			p, ok, err := j.person(ctx, id)
			if err != nil {
				return nil, nil, err
			}
			if ok && j.Send != nil {
				out = append(out, outgoing{p, Message{Kind: KindPolicy, Text: fmt.Sprintf("Please read %s in %s, then press I've read it", n.Title, n.Space), Link: j.tab("answer")}})
			}
		}
		marks = append(marks, func() error { return j.Clock.Mark(ctx, key, now) })
	}
	return out, marks, nil
}
