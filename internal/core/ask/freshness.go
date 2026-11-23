package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ports"
)

// Freshness on an answer (sb-ask(3); SB-2701, SB-2702, SB-2704): a warning when most of an
// answer's documents are stale, the review mark on a cite, and connectors behind on syncing.

// StaleNotice says most of an answer's cited documents are stale.
type StaleNotice struct {
	Stale      int `json:"stale"`
	Of         int `json:"of"`
	OldestDays int `json:"oldest_days"`
}

// SyncNotice is a connector kind behind its freshness target; LastSuccess nil is unknown.
type SyncNotice struct {
	Kind        string     `json:"kind"`
	LastSuccess *time.Time `json:"last_success"`
}

// SyncLag returns, of kinds, those whose last successful sync is past the freshness target,
// or that the cell can't tell about (LastSuccess nil: unknown is never fresh).
type SyncLag func(ctx context.Context, kinds []string) ([]SyncNotice, error)

// staleNotice counts the answer's cited documents, each once; more than half stale warns.
func staleNotice(claims []answer.Claim) *StaleNotice {
	stale := map[string]int{}
	all := map[string]bool{}
	for _, c := range claims {
		for _, ci := range c.Cites {
			k := docKey(ci.Cite)
			all[k] = true
			if ci.Stale && ci.AgeDays != nil && *ci.AgeDays >= stale[k] {
				stale[k] = *ci.AgeDays
			}
		}
	}
	if len(stale)*2 <= len(all) {
		return nil
	}
	n := &StaleNotice{Stale: len(stale), Of: len(all)}
	for _, d := range stale {
		if d > n.OldestDays {
			n.OldestDays = d
		}
	}
	return n
}

// syncNotices are the answer's connector kinds that are behind; an error is unknown for each.
func (a Agent) syncNotices(ctx context.Context, claims []answer.Claim) []SyncNotice {
	if a.SyncLag == nil {
		return nil
	}
	seen := map[string]bool{}
	var kinds []string
	for _, c := range claims {
		for _, ci := range c.Cites {
			if k, _, ok := strings.Cut(ci.Cite, ":"); ok && !seen[k] {
				seen[k] = true
				kinds = append(kinds, k)
			}
		}
	}
	if len(kinds) == 0 {
		return nil
	}
	sort.Strings(kinds)
	out, err := a.SyncLag(ctx, kinds)
	if err != nil {
		out = nil
		for _, k := range kinds {
			out = append(out, SyncNotice{Kind: k})
		}
	}
	return out
}

// marksCache reads each space's review marks once per answer.
type marksCache struct {
	store MarkStore
	by    map[string]map[string]ReviewMark
}

// reviewed is the newest mark at chunk c's version in a space that holds it; a store it
// can't read gives none, so a cite is judged by its age alone.
func (m *marksCache) reviewed(ctx context.Context, tenant string, c ports.Chunk) (ReviewMark, bool) {
	var best ReviewMark
	found := false
	if m == nil || m.store == nil {
		return best, false
	}
	doc := c.Kind + ":" + c.SourceID
	for _, sp := range c.Spaces {
		ms, ok := m.by[sp]
		if !ok {
			list, err := m.store.ReviewMarks(ctx, tenant, sp)
			if err != nil {
				list = nil
			}
			ms = map[string]ReviewMark{}
			for _, x := range list {
				ms[x.Doc] = x
			}
			m.by[sp] = ms
		}
		if x, ok := ms[doc]; ok && x.Version == c.Version && (!found || x.At.After(best.At)) {
			best, found = x, true
		}
	}
	return best, found
}

// StaleLine is the stale warning in the web UI's words, for the chat apps.
func StaleLine(n StaleNotice) string {
	return fmt.Sprintf("Most of the sources for this answer haven't been reviewed in a while (the oldest is %d days old). Check with their owner before relying on it.", n.OldestDays)
}

// SyncLine is a sync warning in the web UI's words, for the chat apps, in UTC.
func SyncLine(n SyncNotice) string {
	if n.LastSuccess == nil {
		return n.Kind + ": when it last synced is unknown; newer changes may be missing."
	}
	return n.Kind + " last synced at " + n.LastSuccess.UTC().Format("15:04 UTC") + "; newer changes may be missing."
}

// Warnings are the answer's stale and sync lines, in order, for the chat apps.
func Warnings(r Response) []string {
	var out []string
	if r.NoticeStale != nil {
		out = append(out, StaleLine(*r.NoticeStale))
	}
	for _, n := range r.NoticeSync {
		out = append(out, SyncLine(n))
	}
	return out
}
