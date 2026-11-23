package laws

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retention"
	"example.com/second-brain/internal/core/session"
)

// mutated wraps every real store with one broken behaviour.
type mutated struct {
	retention.Store
	plan   func(s retention.Store, ctx context.Context, tenant string, before time.Time, held retention.Held) (int, int, error)
	export func(s retention.Store, ctx context.Context, tenant, user string) ([]json.RawMessage, error)
}

func (m mutated) Plan(ctx context.Context, tenant string, before time.Time, held retention.Held) (int, int, error) {
	if m.plan != nil {
		return m.plan(m.Store, ctx, tenant, before, held)
	}
	return m.Store.Plan(ctx, tenant, before, held)
}

func (m mutated) Export(ctx context.Context, tenant, user string) ([]json.RawMessage, error) {
	if m.export != nil {
		return m.export(m.Store, ctx, tenant, user)
	}
	return m.Store.Export(ctx, tenant, user)
}

func wrapAll(m mutated) RetentionStores {
	return func(ss []retention.Store) []retention.Store {
		out := make([]retention.Store, len(ss))
		for i, s := range ss {
			w := m
			w.Store = s
			out[i] = w
		}
		return out
	}
}

func TestSelf_LAW16RetentionPlanMutationsAreCaught(t *testing.T) {
	ignoresHolds := wrapAll(mutated{plan: func(s retention.Store, ctx context.Context, tenant string, before time.Time, _ retention.Held) (int, int, error) {
		return s.Plan(ctx, tenant, before, nil)
	}})
	failsSomewhere(t, "a plan that ignores legal holds", func(r *rand.Rand) error { return law16RetentionPlan(r, ignoresHolds) })
	deletes := wrapAll(mutated{plan: func(s retention.Store, ctx context.Context, tenant string, before time.Time, held retention.Held) (int, int, error) {
		return s.Expire(ctx, tenant, before, held)
	}})
	failsSomewhere(t, "a plan that deletes", func(r *rand.Rand) error { return law16RetentionPlan(r, deletes) })
	dayLate := wrapAll(mutated{plan: func(s retention.Store, ctx context.Context, tenant string, before time.Time, held retention.Held) (int, int, error) {
		return s.Plan(ctx, tenant, before.Add(-24*time.Hour), held)
	}})
	failsSomewhere(t, "a plan with its own cutoff", func(r *rand.Rand) error { return law16RetentionPlan(r, dayLate) })
}

func TestSelf_LAW07SubjectExportMutationsAreCaught(t *testing.T) {
	skipsShares := wrapAll(mutated{export: func(s retention.Store, ctx context.Context, tenant, user string) ([]json.RawMessage, error) {
		if s.Name() == "shares" {
			return nil, nil
		}
		return s.Export(ctx, tenant, user)
	}})
	failsSomewhere(t, "an export that leaves out shares", func(r *rand.Rand) error { return law07SubjectExport(r, skipsShares) })
	everyone := wrapAll(mutated{export: func(s retention.Store, ctx context.Context, tenant, _ string) ([]json.RawMessage, error) {
		var out []json.RawMessage
		for _, u := range []string{"usr_ann", "usr_bob", "usr_cy"} {
			rs, err := s.Export(ctx, tenant, u)
			if err != nil {
				return nil, err
			}
			out = append(out, rs...)
		}
		return out, nil
	}})
	failsSomewhere(t, "an export of everyone's records", func(r *rand.Rand) error { return law07SubjectExport(r, everyone) })
}

// onlyOne ends one of the person's sessions, the way signing out of one browser does.
type onlyOne struct {
	store ports.Sessions
	a     session.Auditor
}

func (o onlyOne) Revoke(ctx context.Context, actor, user, reason string) (int, []string, error) {
	if _, err := o.a.Write(ctx, audit.Event{Actor: actor, Kind: "access", Subject: "sessions:" + user, State: "revoke_requested"}); err != nil {
		return 0, nil, err
	}
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("s%d", i)
		if s, found, _ := o.store.Get(ctx, id); found && s.User == user {
			return 1, nil, o.store.Delete(ctx, id)
		}
	}
	return 0, nil, nil
}

// revokeFirst revokes, then audits.
type revokeFirst struct {
	store ports.Sessions
	a     session.Auditor
}

func (o revokeFirst) Revoke(ctx context.Context, actor, user, reason string) (int, []string, error) {
	n, err := o.store.RevokeUser(ctx, "acme-logistics", user)
	if err != nil {
		return 0, nil, err
	}
	_, err = o.a.Write(ctx, audit.Event{Actor: actor, Kind: "access", Subject: "sessions:" + user, State: "revoke_requested"})
	return n, nil, err
}

func TestSelf_LAW12RevokeMutationsAreCaught(t *testing.T) {
	failsSomewhere(t, "revoking one session", func(r *rand.Rand) error {
		return law12Revoke(r, func(s ports.Sessions, a session.Auditor) SessionRevoker { return onlyOne{s, a} })
	})
	failsSomewhere(t, "revoking before the audit row", func(r *rand.Rand) error {
		return law12Revoke(r, func(s ports.Sessions, a session.Auditor) SessionRevoker { return revokeFirst{s, a} })
	})
}

func TestSelf_LAW07PIIMutationsAreCaught(t *testing.T) {
	plain := RealPIIMasker
	plain.Shown = func(text string, _ []string) string { return text }
	failsSomewhere(t, "snippets that show the whole number", func(r *rand.Rand) error { return law07PII(r, plain) })
	echo := RealPIIMasker
	echo.Answer = func(raw string, _ answer.Context) string { return raw }
	failsSomewhere(t, "an answer that echoes the number", func(r *rand.Rand) error { return law07PII(r, echo) })
	firstFour := RealPIIMasker
	firstFour.Shown = func(text string, flags []string) string {
		return strings.ReplaceAll(ingest.Shown(text, flags), "••••", "4111")
	}
	failsSomewhere(t, "a mask that keeps more than the last four", func(r *rand.Rand) error { return law07PII(r, firstFour) })
}
