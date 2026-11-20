package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	spacemem "example.com/second-brain/internal/adapters/spaces/memory"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

func spacesCall(t *testing.T, h Spaces, groups []string) (int, []SpaceItem) {
	t.Helper()
	s, err := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", "usr_ann", groups)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.Register(mux)
	r := httptest.NewRequest("GET", "/v1/spaces", nil).WithContext(tenancy.WithScope(context.Background(), s))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	var body struct{ Spaces []SpaceItem }
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body.Spaces
}

// SB-1007: only the asker's spaces are listed; another tenant's never are.
func TestSpacesListsOnlyMine(t *testing.T) {
	st := spacemem.New()
	ctx := context.Background()
	for _, sp := range []ports.Space{
		{ID: "spc_finance", Tenant: "acme-logistics", Name: "finance", Label: "restricted", Members: []string{"group:finance"}},
		{ID: "spc_eng", Tenant: "acme-logistics", Name: "eng", Label: "internal", Members: []string{"group:eng"}},
		{ID: "spc_people", Tenant: "beta-freight", Name: "people", Label: "restricted", Members: []string{"group:finance"}},
	} {
		if err := st.PutSpace(ctx, sp); err != nil {
			t.Fatal(err)
		}
	}
	code, got := spacesCall(t, Spaces{Store: st}, []string{"finance"})
	if code != 200 || len(got) != 1 || got[0] != (SpaceItem{Name: "finance", Label: "restricted"}) {
		t.Fatalf("%d %+v", code, got)
	}
	if code, got := spacesCall(t, Spaces{}, []string{"finance"}); code != 200 || len(got) != 0 {
		t.Fatalf("no registry: %d %+v", code, got)
	}
	st.Fail = errors.New("down")
	if code, _ := spacesCall(t, Spaces{Store: st}, []string{"finance"}); code != 503 {
		t.Fatalf("a failing registry: %d", code)
	}
}
