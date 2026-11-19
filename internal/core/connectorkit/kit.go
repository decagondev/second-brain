// Package connectorkit is the connector contract kit (sb-connector(3), SB-201). Every
// connector's tests call Run with a Sandbox; all cases must pass before a manifest may
// name the connector.
package connectorkit

import (
	"context"
	"crypto/sha256"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// ACLDeadline is the caller's deadline for one ACL batch (sb-connector(3)).
const ACLDeadline = 800 * time.Millisecond

// Sandbox is a recorded or sandbox tenant the kit drives.
type Sandbox interface {
	Connector() ports.Connector
	Doc() ports.SourceRef                  // an existing document
	Allowed() ports.UserRef                // can open Doc
	Denied() ports.UserRef                 // can't open Doc
	Refs(n int) []ports.SourceRef          // n existing documents
	InjectACLFault(kind string)            // "500", "timeout" or "404" on the next ACL call
	Edit(ref ports.SourceRef)              // make a content change
	ChangePermissions(ref ports.SourceRef) // make a permission-only change
	// Action returns a valid write for a writable connector; ok is false if read-only.
	Action() (a ports.Action, ok bool)
	// Effects counts changes applied by Act (writable connectors only).
	Effects() int
}

// Run runs every kit case (docs/architecture/03-connectors.md, The contract kit).
func Run(t *testing.T, sb Sandbox) {
	ctx := context.Background()
	c := sb.Connector()

	t.Run("version/negotiated", func(t *testing.T) {
		if err := ports.Negotiate(c); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("fetch/version-stable", func(t *testing.T) {
		a, err := c.Fetch(ctx, sb.Doc())
		if err != nil {
			t.Fatal(err)
		}
		b, err := c.Fetch(ctx, sb.Doc())
		if err != nil {
			t.Fatal(err)
		}
		if a.Version != b.Version || hash(a) != hash(b) {
			t.Fatalf("same version %d returned different content", a.Version)
		}
	})

	t.Run("acl/deny-is-deny", func(t *testing.T) {
		v := acl(t, c, sb.Denied(), sb.Doc())
		if v != ports.VerdictDeny {
			t.Fatalf("user without access got %v, want deny", v)
		}
		if v := acl(t, c, sb.Allowed(), sb.Doc()); v != ports.VerdictAllow {
			t.Fatalf("user with access got %v, want allow", v)
		}
	})

	t.Run("acl/error-is-unknown", func(t *testing.T) {
		for _, f := range []string{"500", "timeout", "404"} {
			sb.InjectACLFault(f)
			if v := acl(t, c, sb.Allowed(), sb.Doc()); v != ports.VerdictUnknown {
				t.Errorf("fault %s: got %v, want unknown", f, v)
			}
		}
	})

	t.Run("acl/batch-200", func(t *testing.T) {
		refs := sb.Refs(200)
		start := time.Now()
		vs, err := c.ACL(ctx, sb.Allowed(), refs)
		if err != nil || len(vs) != len(refs) {
			t.Fatalf("%d verdicts for %d refs, err %v", len(vs), len(refs), err)
		}
		if d := time.Since(start); d > ACLDeadline {
			t.Fatalf("200 refs took %v, deadline %v", d, ACLDeadline)
		}
	})

	t.Run("changes/resume", func(t *testing.T) {
		_, cur, err := c.Changes(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		sb.Edit(sb.Doc())
		first, next, err := c.Changes(ctx, cur)
		if err != nil || !contains(first, sb.Doc(), false) {
			t.Fatalf("edit not reported after cursor %q: %v %v", cur, first, err)
		}
		again, next2, err := c.Changes(ctx, cur) // restart from the old cursor
		if err != nil || len(again) != len(first) || next2 != next {
			t.Fatalf("restarting from a cursor lost or added changes: %v vs %v", again, first)
		}
		if rest, _, err := c.Changes(ctx, next); err != nil || len(rest) != 0 {
			t.Fatalf("after the latest cursor: %v %v", rest, err)
		}
	})

	t.Run("changes/acl-delta", func(t *testing.T) {
		_, cur, err := c.Changes(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		sb.ChangePermissions(sb.Doc())
		set, _, err := c.Changes(ctx, cur)
		if err != nil || !contains(set, sb.Doc(), true) {
			t.Fatalf("permission change not reported as a change: %v %v", set, err)
		}
	})

	a, writable := sb.Action()
	if !writable {
		t.Run("act/not-supported", func(t *testing.T) {
			r, err := c.Act(ctx, ports.Action{Target: "x"}, "apr_kit")
			if err == nil || r.TargetID != "" || !strings.Contains(err.Error(), "ENOTSUP") {
				t.Fatalf("read-only Act: receipt %q err %v; want ENOTSUP and no receipt", r.TargetID, err)
			}
		})
		return
	}

	t.Run("act/receipt-or-error", func(t *testing.T) {
		r, err := c.Act(ctx, a, "apr_kit_1")
		if err == nil && r.TargetID == "" {
			t.Fatal("Act succeeded without a receipt id")
		}
	})

	t.Run("act/idempotent", func(t *testing.T) {
		before := sb.Effects()
		r1, err1 := c.Act(ctx, a, "apr_kit_2")
		r2, err2 := c.Act(ctx, a, "apr_kit_2")
		if err1 != nil || err2 != nil || r1.TargetID == "" || r1 != r2 || sb.Effects() != before+1 {
			t.Fatalf("same key twice: %v %v, %v %v, effects %d -> %d", r1, err1, r2, err2, before, sb.Effects())
		}
	})
}

func acl(t *testing.T, c ports.Connector, u ports.UserRef, ref ports.SourceRef) ports.Verdict {
	t.Helper()
	vs, err := c.ACL(context.Background(), u, []ports.SourceRef{ref})
	if err != nil || len(vs) != 1 {
		return ports.VerdictUnknown // an error is Unknown, never Allow
	}
	return vs[0]
}

func hash(d ports.Document) [32]byte {
	return sha256.Sum256([]byte(d.Title + "\x00" + d.Text))
}

func contains(set ports.ChangeSet, ref ports.SourceRef, aclOnly bool) bool {
	for _, c := range set {
		if c.Ref.ID == ref.ID && c.ACLOnly == aclOnly {
			return true
		}
	}
	return false
}
