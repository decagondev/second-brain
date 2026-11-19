package edge

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

type cells struct {
	c   []ports.CellRecord
	err error
}

func (s cells) CellByHost(_ context.Context, host string) (ports.CellRecord, bool, error) {
	if s.err != nil {
		return ports.CellRecord{}, false, s.err
	}
	for _, c := range s.c {
		for _, h := range c.Hosts {
			if h == host {
				return c, true, nil
			}
		}
	}
	return ports.CellRecord{}, false, nil
}
func (s cells) CellByTenant(context.Context, string) (ports.CellRecord, bool, error) {
	return ports.CellRecord{}, false, errors.New("unused")
}

func router(s cells, tenant string) (Router, *string) {
	var reached string
	return Router{
		Cells:  s,
		Verify: func(*http.Request) (string, error) { return tenant, nil },
		Forward: func(c ports.CellRecord) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { reached = c.Tenant })
		},
	}, &reached
}

var acme = ports.CellRecord{Tenant: "acme-logistics", Hosts: []string{"acme-logistics.sb.example"}}

func serve(rt Router, host string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", "/v1/ask", nil)
	r.Host = host
	w := httptest.NewRecorder()
	rt.ServeHTTP(w, r)
	return w
}

func TestHostIsNormalised(t *testing.T) {
	rt, reached := router(cells{c: []ports.CellRecord{acme}}, "acme-logistics")
	serve(rt, "ACME-Logistics.sb.example.:443")
	if *reached != "acme-logistics" {
		t.Fatalf("reached %q", *reached)
	}
}

// Every refusal is a coded JSON error with the registry's HTTP status (sb-api(3)).
func TestRefusals(t *testing.T) {
	cases := []struct {
		name   string
		s      cells
		host   string
		tenant string
		status int
		code   string
	}{
		{"registry down", cells{err: errors.New("timeout")}, "acme-logistics.sb.example", "acme-logistics", 503, "EUNAVAIL"},
		{"unknown host", cells{c: []ports.CellRecord{acme}}, "evil.example", "acme-logistics", 403, "EPERM"},
		{"no token", cells{c: []ports.CellRecord{acme}}, "acme-logistics.sb.example", "", 401, "EAUTH"},
		{"other tenant", cells{c: []ports.CellRecord{acme}}, "acme-logistics.sb.example", "beta-freight", 403, "ETENANT"},
	}
	for _, c := range cases {
		rt, reached := router(c.s, c.tenant)
		w := serve(rt, c.host)
		if *reached != "" || w.Code != c.status || !strings.Contains(w.Body.String(), `"code":"`+c.code+`"`) {
			t.Errorf("%s: reached %q, %d %s", c.name, *reached, w.Code, w.Body.String())
		}
	}
}
