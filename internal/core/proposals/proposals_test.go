package proposals

import (
	"context"
	"errors"
	"testing"

	"example.com/second-brain/internal/core/manifest"
)

func TestHashIsTheFormattedManifest(t *testing.T) {
	a := manifest.Tenant{Tenant: "acme-logistics", Channel: "stable", Agents: []string{"ask"}}
	b, same := a, a
	b.Agents = []string{"ask", "draft-reply"}
	same.Agents = []string{"ask"}
	if Hash(a) == Hash(b) || Hash(a) != Hash(same) || len(Hash(a)) != 64 {
		t.Fatal("the hash must identify the manifest")
	}
}

func TestMemoryStore(t *testing.T) {
	ctx := context.Background()
	var s MemoryStore
	if _, found, err := s.Get(ctx, "prop_1"); found || err != nil {
		t.Fatal("an empty store found something")
	}
	if _, err := s.Update(ctx, "prop_1", func(*Proposal) error { return nil }); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update of a missing proposal: %v", err)
	}
	p := Proposal{ID: "prop_2", Space: "finance", State: Open, Diff: []string{"- a"}}
	_ = s.Put(ctx, Proposal{ID: "prop_1", State: Open})
	_ = s.Put(ctx, p)
	p.Diff[0] = "changed" // the store keeps its own copy
	got, _, _ := s.Get(ctx, "prop_2")
	if got.Diff[0] != "- a" {
		t.Fatal("the store shares its caller's slice")
	}
	all, _ := s.List(ctx)
	if len(all) != 2 || all[0].ID != "prop_1" {
		t.Fatalf("list %+v", all)
	}
	boom := errors.New("no")
	if _, err := s.Update(ctx, "prop_2", func(x *Proposal) error { x.State = Approved; return boom }); !errors.Is(err, boom) {
		t.Fatal(err)
	}
	if got, _, _ := s.Get(ctx, "prop_2"); got.State != Open {
		t.Fatal("a failed update was stored")
	}
	if got, err := s.Update(ctx, "prop_2", func(x *Proposal) error { x.State = Approved; return nil }); err != nil || got.State != Approved {
		t.Fatalf("update: %+v %v", got, err)
	}
}
