package gcp

import (
	"context"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/placement/gcp/gcptest"
	"example.com/second-brain/internal/core/schedule"
)

func TestSchedulerEnsure(t *testing.T) {
	ctx := context.Background()
	f := gcptest.New()
	defer f.Server.Close()
	f.Grant("tok", "gamma-prod-7731", SchedulerPermissions...)
	s := Scheduler{Project: project(f, "gamma-prod-7731", "tok"), Location: "europe-west1", Topic: "projects/gamma-prod-7731/topics/gamma-parts-jobs", Base: f.URL()}
	if err := s.Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	if len(f.Schedules) != len(schedule.Jobs) {
		t.Fatalf("%d jobs, want %d", len(f.Schedules), len(schedule.Jobs))
	}
	name := "projects/gamma-prod-7731/locations/europe-west1/jobs/sb-anchor"
	f.Schedules[name]["schedule"] = "0 0 1 1 *" // someone slowed the anchors to once a year
	if err := s.Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	if f.Schedules[name]["schedule"] != "5 * * * *" {
		t.Fatalf("drift not corrected: %v", f.Schedules[name]["schedule"])
	}
}

func TestBucketIsWORM(t *testing.T) {
	ctx := context.Background()
	f := gcptest.New()
	defer f.Server.Close()
	f.Grant("tok", "gamma-prod-7731", BucketPermissions...)
	p := project(f, "gamma-prod-7731", "tok")
	f.Bucket("gamma-prod-7731", "sb-gamma-parts-worm", MinRetention, false)
	b := Bucket{Project: p, Bucket: "sb-gamma-parts-worm", Base: f.URL()}
	if err := b.Put(ctx, "a", []byte("x"), time.Time{}); err == nil || !strings.Contains(err.Error(), "locked") {
		t.Fatalf("an unlocked bucket was used: %v", err)
	}
	f.Bucket("gamma-prod-7731", "sb-gamma-parts-short", 30*24*time.Hour, true)
	if err := (Bucket{Project: p, Bucket: "sb-gamma-parts-short", Base: f.URL()}).Check(ctx); err == nil {
		t.Fatal("a 30-day bucket was accepted")
	}
	f.Bucket("gamma-prod-7731", "sb-gamma-parts-anchors", MinRetention, true)
	b.Bucket = "sb-gamma-parts-anchors"
	if err := b.Put(ctx, "gamma-parts/2027-03-02/anchor-0001.json", []byte("one"), time.Time{}); err != nil {
		t.Fatal(err)
	}
	if err := b.Put(ctx, "gamma-parts/2027-03-02/anchor-0001.json", []byte("two"), time.Time{}); err == nil {
		t.Fatal("an anchor was overwritten")
	}
	if err := f.DeleteObject("sb-gamma-parts-anchors", "gamma-parts/2027-03-02/anchor-0001.json", time.Now()); err == nil {
		t.Fatal("an anchor was deleted inside its retention")
	}
	got, err := b.Get(ctx, "gamma-parts/2027-03-02/anchor-0001.json")
	if err != nil || string(got) != "one" {
		t.Fatalf("get: %q %v", got, err)
	}
	names, err := b.List(ctx, "gamma-parts/")
	if err != nil || len(names) != 1 {
		t.Fatalf("list: %v %v", names, err)
	}
}
