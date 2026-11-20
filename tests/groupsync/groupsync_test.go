package groupsync_test

import (
	"context"
	"reflect"
	"testing"

	dirmem "example.com/second-brain/internal/adapters/directory/memory"
	sessmem "example.com/second-brain/internal/adapters/sessions/memory"
	"example.com/second-brain/internal/core/groupsync"
	"example.com/second-brain/internal/core/ports"
)

func TestApply(t *testing.T) {
	ctx := context.Background()
	dir, sess := dirmem.New(), sessmem.New()
	ana, bo := groupsync.UserID("ana@delta-tools.example"), groupsync.UserID("Bo@delta-tools.example")
	first := map[string][]string{"group:finance": {"ana@delta-tools.example", "bo@delta-tools.example"}, "group:support": {"bo@delta-tools.example"}}
	if res, err := groupsync.Apply(ctx, dir, sess, "delta-tools", nil, first); err != nil || res.Active != 2 {
		t.Fatalf("first apply: %+v %v", res, err)
	}
	if g, _ := dir.GroupsOf(ctx, "delta-tools", bo); !reflect.DeepEqual(g, []string{"finance", "support"}) {
		t.Fatalf("bo's groups = %v", g)
	}
	if err := sess.Put(ctx, ports.Session{ID: "s1", Tenant: "delta-tools", User: bo}); err != nil {
		t.Fatal(err)
	}
	second := map[string][]string{"group:finance": {"ana@delta-tools.example"}}
	res, err := groupsync.Apply(ctx, dir, sess, "delta-tools", first, second)
	if err != nil || res.Deactivated != 1 || res.Revoked != 1 {
		t.Fatalf("second apply: %+v %v", res, err)
	}
	if active, known, _ := dir.Active(ctx, "delta-tools", bo); !known || active {
		t.Fatal("bo is still active")
	}
	if active, _, _ := dir.Active(ctx, "delta-tools", ana); !active {
		t.Fatal("ana was deactivated")
	}
	if _, found, _ := dir.Group(ctx, "delta-tools", groupsync.GroupID("group:support")); found {
		t.Fatal("a group the manifest dropped is still there")
	}
	if _, found, _ := dir.User(ctx, "beta-freight", ana); found {
		t.Fatal("a user appeared in another tenant")
	}
}
