package barrier

import (
	"context"
	"sort"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ports"
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
