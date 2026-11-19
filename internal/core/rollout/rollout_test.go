package rollout

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2027, 3, 2, 10, 0, 0, 0, time.UTC)

func world() *State {
	return New("rel_a", []Tenant{
		{Name: "internal", Channel: Testing, Ring: 0},
		{Name: "beta-freight", Channel: Testing, Ring: 1},
		{Name: "acme-logistics", Channel: Stable, Ring: 2},
		{Name: "gamma-parts", Channel: Stable, Ring: 2},
	})
}

func TestPromotePath(t *testing.T) {
	s := world()
	if err := s.Register("rel_b", true); err != nil {
		t.Fatal(err)
	}
	if err := s.Promote("rel_b", t0); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "soaked") {
		t.Fatalf("unsoaked: %v", err)
	}
	_ = s.DeployTesting("internal", "rel_b", t0)
	_ = s.DeployTesting("beta-freight", "rel_b", t0.Add(time.Hour))
	if err := s.Promote("rel_b", t0.Add(48*time.Hour)); err == nil {
		t.Fatal("ring 1 hadn't soaked 48h")
	}
	if err := s.Promote("rel_b", t0.Add(50*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if s.Stable != "rel_b" || s.PrevStable != "rel_a" {
		t.Fatalf("%s %s", s.Stable, s.PrevStable)
	}
	_ = s.Register("rel_c", true)
	_ = s.DeployTesting("internal", "rel_c", t0)
	_ = s.DeployTesting("beta-freight", "rel_c", t0)
	err := s.Promote("rel_c", t0.Add(100*time.Hour))
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "promote guard") {
		t.Fatalf("guard: %v", err)
	}
	_ = s.DeployStable("acme-logistics")
	_ = s.DeployStable("gamma-parts")
	if err := s.Promote("rel_c", t0.Add(100*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if v := s.Violations(); len(v) != 0 {
		t.Fatal(v)
	}
}

func TestUngatedNeverRuns(t *testing.T) {
	s := world()
	_ = s.Register("rel_x", false)
	if err := s.DeployTesting("internal", "rel_x", t0); !errors.Is(err, ErrRefused) {
		t.Fatalf("%v", err)
	}
}

func TestRollbackDrill(t *testing.T) {
	s := world()
	_ = s.Register("rel_b", true)
	_ = s.DeployTesting("internal", "rel_b", t0)
	_ = s.DeployTesting("beta-freight", "rel_b", t0)
	_ = s.Promote("rel_b", t0.Add(49*time.Hour))
	_ = s.DeployStable("acme-logistics")
	_ = s.DeployStable("gamma-parts")
	clock := t0
	now := func() time.Time { return clock }
	var deployed []string
	rep, err := s.Drill(func(tenant, rel string) error {
		clock = clock.Add(time.Minute)
		if tenant == "gamma-parts" {
			return errors.New("cell unreachable")
		}
		deployed = append(deployed, tenant+"="+rel)
		return nil
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Revoked != "rel_b" || rep.Restored != "rel_a" || rep.Cells != 4 || rep.Passed || len(rep.Failed) != 1 {
		t.Fatalf("%+v", rep)
	}
	if !s.Tenants["gamma-parts"].Draining || !s.Releases["rel_b"].Revoked {
		t.Fatal("a cell whose redeploy failed must stay draining")
	}
	if v := s.Violations(); len(v) != 0 {
		t.Fatal(v)
	}
	if err := s.Promote("rel_b", t0.Add(200*time.Hour)); err == nil {
		t.Fatal("a revoked release was promoted")
	}
	if _, err := s.Rollback(); err == nil {
		t.Fatal("rolled back past the previous stable")
	}
}

// SB-707: a tenant-custody cell counts in the promote guard; once detached it doesn't.
func TestTenantCustodyCellsCount(t *testing.T) {
	s := world()
	if err := s.Detach("gamma-parts"); !errors.Is(err, ErrRefused) {
		t.Fatalf("a factory cell detached: %v", err)
	}
	if err := s.SetCustody("gamma-parts", "tenant"); err != nil {
		t.Fatal(err)
	}
	_ = s.Register("rel_b", true)
	_ = s.DeployTesting("internal", "rel_b", t0)
	_ = s.DeployTesting("beta-freight", "rel_b", t0)
	_ = s.Promote("rel_b", t0.Add(50*time.Hour))
	_ = s.Register("rel_c", true)
	_ = s.DeployTesting("internal", "rel_c", t0)
	_ = s.DeployTesting("beta-freight", "rel_c", t0)
	_ = s.DeployStable("acme-logistics") // gamma-parts, in its own account, hasn't pulled
	err := s.Promote("rel_c", t0.Add(100*time.Hour))
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "gamma-parts") {
		t.Fatalf("the guard ignored a tenant-custody cell: %v", err)
	}
	if err := s.Detach("gamma-parts"); err != nil {
		t.Fatal(err)
	}
	if err := s.Promote("rel_c", t0.Add(100*time.Hour)); err != nil {
		t.Fatalf("a detached cell still blocks: %v", err)
	}
	if err := s.DeployStable("gamma-parts"); !errors.Is(err, ErrRefused) {
		t.Fatal("deployed to a detached cell")
	}
	if len(s.Detached) != 1 || len(s.Violations()) != 0 {
		t.Fatalf("%v %v", s.Detached, s.Violations())
	}
}
