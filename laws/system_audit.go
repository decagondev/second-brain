package laws

import (
	"bytes"
	"context"
	"time"

	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	"example.com/second-brain/internal/core/audit"
)

// The real audit chain (SB-401, SB-402): the log over the memory store, the export with
// signed anchors, and the offline verifier with the cell's key pinned.
func init() {
	NewAuditChain = func() AuditChain { return newChainWorld(false) }
}

type chainWorld struct {
	log    *audit.Log
	store  *auditmem.Store
	signer audit.Signer
	clock  time.Time
	// trusting is a mutation: a verifier that doesn't recompute row hashes.
	trusting bool
}

func newChainWorld(trusting bool) *chainWorld {
	s, _ := audit.NewSigner("acme-logistics", bytes.Repeat([]byte{7}, 32))
	w := &chainWorld{store: auditmem.New(), signer: s, clock: time.Unix(1_800_000_000, 0), trusting: trusting}
	w.log = &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_law" }, Store: w.store,
		Now: func() time.Time { w.clock = w.clock.Add(time.Second); return w.clock }}
	return w
}

func (w *chainWorld) Append(kind, subject string) error {
	_, err := w.log.Write(context.Background(), audit.Event{Actor: "usr_law", Kind: kind, Subject: subject, Reason: "law-11"})
	return err
}

func (w *chainWorld) Export() ([]byte, error) {
	var b bytes.Buffer
	_, err := audit.Export(context.Background(), w.store, w.signer, 1, w.clock, &b)
	return b.Bytes(), err
}

func (w *chainWorld) Verify(exp []byte) (bool, error) {
	if w.trusting {
		return bytes.Count(exp, []byte(`"anchor":true`)) >= 2, nil // checks anchors exist, nothing else
	}
	_, err := audit.Verify(bytes.NewReader(exp), "acme-logistics", w.signer.Key())
	return err == nil, nil
}
