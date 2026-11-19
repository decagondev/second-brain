package access

import (
	"context"
	"errors"
	"testing"
)

type src struct {
	ps  []Path
	err error
}

func (s src) Name() string                                  { return "test" }
func (s src) Paths(context.Context, string) ([]Path, error) { return s.ps, s.err }

type gauge struct{ v map[string]float64 }

func (g *gauge) Set(name, _ string, _ map[string]string, v float64) { g.v[name] = v }

func TestCheckAndWatch(t *testing.T) {
	ctx := context.Background()
	ok := src{ps: []Path{{ID: "a", Grant: "platform-kill"}, {ID: "b", Grant: ""}, {ID: "c", Grant: "support-metrics"}}}
	res, err := Check(ctx, "gamma-parts", []Source{ok}, []string{"publish-releases", "platform-kill"})
	if err != nil || len(res.Paths) != 3 || len(res.Unlisted) != 2 || res.Unlisted[0].ID != "b" || res.Unlisted[1].ID != "c" {
		t.Fatalf("%+v %v", res, err)
	}
	g := &gauge{v: map[string]float64{}}
	if _, err := Watch(ctx, "gamma-parts", []Source{ok}, []string{"platform-kill"}, g); err != nil || g.v["sb_unlisted_access_paths"] != 2 {
		t.Fatalf("watch: %v %v", g.v, err)
	}
	bad := src{err: errors.New("403")}
	if _, err := Watch(ctx, "gamma-parts", []Source{ok, bad}, nil, g); !errors.Is(err, ErrUnavailable) || g.v["sb_unlisted_access_paths"] != 1 {
		t.Fatalf("an unreadable source must fail and alert: %v %v", g.v, err)
	}
	if _, err := Check(ctx, "gamma-parts", nil, nil); !errors.Is(err, ErrUnavailable) {
		t.Fatal("no sources passed")
	}
}
