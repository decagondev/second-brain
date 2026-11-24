package ask

import (
	"context"
	"fmt"
	"sort"
	"time"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Status updates (sb-spaces(7), Status updates; SB-4102): a space with updates: true gets a
// weekly draft from the update agent; its steward publishes it as a note under
// notes/SPACE/updates/. A department's Start tab lists its teams' latest published updates,
// each checked as the reader.

// TeamUpdate is one team's latest published update, as the reader can open it.
type TeamUpdate struct {
	Space string    `json:"space"`
	Title string    `json:"title"`
	Cite  string    `json:"cite"`
	URL   string    `json:"url,omitempty"`
	At    time.Time `json:"at"`
}

// UpdatesResponse is GET /v1/spaces/NAME/updates.
type UpdatesResponse struct {
	Space   string       `json:"space"`
	Updates []TeamUpdate `json:"updates"`
	// Hidden counts teams whose latest update the reader can't open: never which.
	Hidden int `json:"hidden"`
}

// TeamUpdates are the latest published updates of space name's teams, for a member of it.
func (x Stewards) TeamUpdates(ctx context.Context, s tenancy.Scope, name string) (UpdatesResponse, error) {
	if err := x.member(ctx, s, name); err != nil {
		return UpdatesResponse{}, err
	}
	out := UpdatesResponse{Space: name, Updates: []TeamUpdate{}}
	if x.Children == nil {
		return out, nil
	}
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return out, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return out, err
	}
	for _, team := range x.Children(name) {
		cs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Under: []ports.SourceRef{{Kind: noteKind, ID: "/notes/" + team + "/updates/"}}})
		if err != nil {
			return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
		}
		var cand []ports.Chunk
		for _, c := range firsts(latest(cs)) {
			cand = append(cand, c)
		}
		if len(cand) == 0 {
			continue
		}
		sort.Slice(cand, func(i, j int) bool {
			return cand[i].Seen.After(cand[j].Seen) || cand[i].Seen.Equal(cand[j].Seen) && cand[i].ID > cand[j].ID
		})
		res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, cand[:1], 1)
		if err != nil {
			return out, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		if len(res.Context) == 0 {
			out.Hidden++
			continue
		}
		c := res.Context[0]
		r := x.Agent.result(ctx, nil, c)
		out.Updates = append(out.Updates, TeamUpdate{Space: team, Title: r.Title, Cite: c.Cite, URL: r.URL, At: c.Seen.UTC()})
	}
	return out, nil
}
