package barrier

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

type dir struct{ users []ports.User }

func (d dir) UsersByEmail(_ context.Context, _, email string) ([]ports.User, error) {
	var out []ports.User
	for _, u := range d.users {
		for _, e := range u.Emails {
			if strings.EqualFold(e, email) {
				out = append(out, u)
			}
		}
	}
	return out, nil
}

func (dir) GroupsOf(_ context.Context, _, user string) ([]string, error) {
	return map[string][]string{"usr_js": {"deal-team"}, "usr_js2": {"legal"}}[user], nil
}

// SB-1401: an owner is found by any email, whatever the case, not by userName.
func TestOwnerByEmail(t *testing.T) {
	d := dir{users: []ports.User{{ID: "usr_js", UserName: "JSmith", Emails: []string{"J.Smith@Acme.example"}}}}
	gs, err := Directory(d, "acme-logistics")(context.Background(), "j.smith@acme.example")
	if err != nil || len(gs) != 1 || gs[0] != "deal-team" {
		t.Fatalf("owner groups %v %v", gs, err)
	}
	d.users = append(d.users, ports.User{ID: "usr_js2", Emails: []string{"j.smith@acme.example"}})
	gs, _ = Directory(d, "acme-logistics")(context.Background(), "j.smith@acme.example")
	sort.Strings(gs)
	if len(gs) != 2 {
		t.Fatalf("an email two users share must count both: %v", gs)
	}
	if gs, err := Directory(d, "acme-logistics")(context.Background(), "outsider@other.example"); err != nil || gs == nil || len(gs) != 0 {
		t.Fatalf("an outside owner: %v %v", gs, err)
	}
}

func scope(t *testing.T, user string, groups ...string) tenancy.Scope {
	s, err := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", user, groups)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// The cell's facts for actions and approvals: none without barriers, refusals for what it
// can't place, and the barred groups when it can.
func TestCellFacts(t *testing.T) {
	ctx := context.Background()
	s := scope(t, "usr_eve", "eng")
	none := Cell{}
	if f, err := none.Target(ctx, s, "docs:d1"); f != nil || err != nil {
		t.Fatalf("a cell without spaces: %v %v", f, err)
	}
	all := []ports.Space{{ID: "spc_deals", Name: "deals", Members: []string{"group:deal-team"}, Scope: map[string][]string{"docs": {"/Deals/**"}}}}
	bars := [][2]string{{"group:deal-team", "group:eng"}}
	c := Cell{Spaces: func(context.Context) ([]ports.Space, [][2]string, error) { return all, bars, nil },
		Owners: func(context.Context, string) ([]string, error) { return []string{"deal-team"}, nil },
		Locate: func(_ context.Context, r ports.SourceRef) (ports.Document, error) {
			if r.ID == "gone" {
				return ports.Document{}, errors.New("not found")
			}
			return ports.Document{Location: "/Deals/orion", Owner: "dana@acme.example"}, nil
		},
		GroupsOf: func(_ context.Context, u string) ([]string, error) {
			if u == "usr_broken" {
				return nil, errors.New("down")
			}
			return []string{"deal-team"}, nil
		}}
	f, err := c.Target(ctx, s, "docs:d1")
	if err != nil {
		t.Fatal(err)
	}
	if ts, _ := f["target_spaces"].([]string); len(ts) != 1 || ts[0] != "spc_deals" {
		t.Fatalf("target spaces %v", f["target_spaces"])
	}
	if bg, _ := f["barred_groups"].([]string); len(bg) != 1 || bg[0] != "group:deal-team" {
		t.Fatalf("barred groups %v", f["barred_groups"])
	}
	for _, target := range []string{"docs:gone", "no-colon"} {
		if _, err := c.Target(ctx, s, target); !errors.Is(err, ErrUnplaced) {
			t.Errorf("%s: %v", target, err)
		}
	}
	if a, err := c.Approval(ctx, s, "usr_dana"); err != nil || len(a["requester_groups"].([]string)) != 1 {
		t.Fatalf("approval facts %v %v", a, err)
	}
	if _, err := c.Approval(ctx, s, "usr_broken"); !errors.Is(err, ErrUnplaced) {
		t.Fatalf("an unreadable requester: %v", err)
	}
	broken := c
	broken.Spaces = func(context.Context) ([]ports.Space, [][2]string, error) { return nil, nil, errors.New("down") }
	if _, err := broken.Target(ctx, s, "docs:d1"); !errors.Is(err, ErrUnplaced) {
		t.Fatalf("unreadable spaces: %v", err)
	}
	if o, ok := Owners(ctx, nil, "x@acme.example"); ok || o != nil {
		t.Fatal("an owner with no directory must be unknown")
	}
}
