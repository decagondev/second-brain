package toolbus

import (
	"context"
	"time"

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
)

// Reconcile timings (sb-approvals(7), Timeouts with unknown effect).
const (
	ReconcileEvery  = 5 * time.Minute
	ReconcileWindow = 24 * time.Hour
)

// Finder is implemented by connectors that can look a change up by idempotency key.
type Finder interface {
	Find(ctx context.Context, key ports.IdempotencyKey) (ports.Receipt, bool, error)
}

// ReconcileResult counts one pass.
type ReconcileResult struct {
	Checked, Found, GaveUp, NoFinder int
}

// Reconcile asks the target about every unconfirmed action, once. It never calls Act.
func (b *Bus) Reconcile(ctx context.Context) (ReconcileResult, error) {
	var res ReconcileResult
	list, err := b.Actions.Unconfirmed(ctx, b.Tenant)
	if err != nil {
		return res, err
	}
	for _, a := range list {
		res.Checked++
		key := a.Approval
		if key == "" {
			key = a.ID // calls that needed no approval use their own id as the key
		}
		f, ok := b.Connectors[b.Tools[a.Tool].Connector].(Finder)
		if !ok {
			res.NoFinder++
			continue
		}
		rc, found, err := f.Find(ctx, ports.IdempotencyKey(key))
		switch {
		case err != nil:
			continue // try again next pass
		case found && rc.TargetID != "":
			if _, err := b.Audit.Write(ctx, audit.Event{Actor: "sys_reconciler", Kind: "tool_call", Subject: a.ID, State: string(approvals.Ran),
				Approval: a.Approval, PayloadSHA256: a.PayloadSHA256, Receipt: rc.TargetID, Reason: "reconciled"}); err != nil {
				continue // no audit row, no change
			}
			if b.Receipts != nil {
				_ = b.Receipts.Put(ctx, b.Tenant, key, rc.TargetID)
			}
			done, err := b.Actions.Move(ctx, b.Tenant, a.ID, approvals.DidNotRun, func(x *approvals.Action) error {
				x.Receipt, x.Reason, x.Unconfirmed = rc.TargetID, "reconciled", false
				return nil
			})
			if err == nil {
				res.Found++
				b.notify(ctx, done, "reconciled")
			}
		case b.Now().Sub(a.FinishedAt) >= ReconcileWindow:
			if _, err := b.Actions.Move(ctx, b.Tenant, a.ID, approvals.DidNotRun, func(x *approvals.Action) error {
				x.Reason, x.Unconfirmed = "not found after 24h", false
				return nil
			}); err == nil {
				res.GaveUp++
			}
		}
	}
	return res, nil
}

// RunReconciler reconciles every ReconcileEvery until ctx ends.
func (b *Bus) RunReconciler(ctx context.Context) {
	t := time.NewTicker(ReconcileEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_, _ = b.Reconcile(ctx)
		}
	}
}
