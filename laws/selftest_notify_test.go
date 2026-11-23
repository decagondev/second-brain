package laws

import (
	"context"
	"math/rand"
	"testing"
	"time"

	dirmem "example.com/second-brain/internal/adapters/directory/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/ports"
)

// staleNotify reads the stewards group once, when the job is built.
type staleNotify struct{}

type snapshotDir struct {
	notify.Directory
	members map[string][]string
}

func (d snapshotDir) Members(_ context.Context, _, g string) ([]string, error) {
	return d.members[g], nil
}

func (staleNotify) Job(store *tracemem.Store, dir *dirmem.Dir, facts ask.SpaceFacts, send notify.Sender, now func() time.Time) notify.Job {
	j := RealNotify{}.Job(store, dir, facts, send, now)
	ms, _ := dir.Members(context.Background(), "acme-logistics", "support-leads")
	j.Dir = snapshotDir{dir, map[string][]string{"support-leads": ms}}
	return j
}

// activeNotify tells deprovisioned people too.
type activeNotify struct{}

type activeDir struct{ notify.Directory }

func (d activeDir) User(ctx context.Context, tenant, id string) (ports.User, bool, error) {
	u, ok, err := d.Directory.User(ctx, tenant, id)
	u.Active = true
	return u, ok, err
}

func (activeNotify) Job(store *tracemem.Store, dir *dirmem.Dir, facts ask.SpaceFacts, send notify.Sender, now func() time.Time) notify.Job {
	j := RealNotify{}.Job(store, dir, facts, send, now)
	j.Dir = activeDir{dir}
	return j
}

// leakyNotify adds the question to each message, as a "helpful" preview would.
type leakyNotify struct{}

type previewSender struct {
	notify.Sender
	store *tracemem.Store
	now   func() time.Time
}

func (p previewSender) Send(ctx context.Context, to notify.Person, m notify.Message) error {
	qs, _ := p.store.StewardQuestionsBetween(ctx, "acme-logistics", time.Time{}, p.now())
	if len(qs) > 0 {
		m.Text += ": " + qs[0].Question
	}
	return p.Sender.Send(ctx, to, m)
}

func (leakyNotify) Job(store *tracemem.Store, dir *dirmem.Dir, facts ask.SpaceFacts, send notify.Sender, now func() time.Time) notify.Job {
	return RealNotify{}.Job(store, dir, facts, previewSender{send, store, now}, now)
}

// The notifications law passes on the real job, and catches stewards read once, the
// deprovisioned told, and a message that previews the question.
func TestSelf_NotifyLaw(t *testing.T) {
	if violates(60, func(r *rand.Rand) error { return law21Notify(r, RealNotify{}) }) {
		t.Fatal("the real job breaks the notifications law")
	}
	for name, sys := range map[string]NotifySystem{"stewards read once": staleNotify{}, "the deprovisioned told": activeNotify{},
		"a question previewed": leakyNotify{}} {
		if !violates(60, func(r *rand.Rand) error { return law21Notify(r, sys) }) {
			t.Errorf("%s was not caught", name)
		}
	}
}
