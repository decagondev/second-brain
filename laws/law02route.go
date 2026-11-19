package laws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"

	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/edge"
)

// LAW-02 at the request level (SB-105): a request reaches only the cell its host names,
// and only if its token belongs to that tenant. Anything unknown is refused.

// Routed is what the router did with one request: the tenant of the cell it forwarded
// to, or the error code it refused with.
type Routed struct{ Cell, Code string }

// Request is one incoming request as the edge router sees it.
type Request struct {
	Host        string
	TokenTenant string
	TokenValid  bool
}

// RouterSUT routes one request with the given cells registered.
type RouterSUT func(cells []ports.CellRecord, req Request) (Routed, error)

// NewRouter builds the system under test: the real edge router (SB-105), driven over
// HTTP, with an in-memory registry and a token check that returns the request's tenant.
var NewRouter = func() RouterSUT {
	return func(cells []ports.CellRecord, req Request) (Routed, error) {
		var reached string
		rt := edge.Router{
			Cells: memCells(cells),
			Verify: func(*http.Request) (string, error) {
				if !req.TokenValid {
					return "", errors.New("bad token")
				}
				return req.TokenTenant, nil
			},
			Forward: func(c ports.CellRecord) http.Handler {
				return http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = c.Tenant })
			},
		}
		hr := httptest.NewRequest("GET", "/v1/ask", nil)
		hr.Host = req.Host
		w := httptest.NewRecorder()
		rt.ServeHTTP(w, hr)
		if reached != "" {
			return Routed{Cell: reached}, nil
		}
		var body struct{ Error struct{ Code string } }
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			return Routed{}, fmt.Errorf("refusal is not a JSON error: %q", w.Body.String())
		}
		return Routed{Code: body.Error.Code}, nil
	}
}

type memCells []ports.CellRecord

func (m memCells) CellByHost(_ context.Context, host string) (ports.CellRecord, bool, error) {
	for _, c := range m {
		for _, h := range c.Hosts {
			if h == host {
				return c, true, nil
			}
		}
	}
	return ports.CellRecord{}, false, nil
}

func (m memCells) CellByTenant(_ context.Context, tenant string) (ports.CellRecord, bool, error) {
	for _, c := range m {
		if c.Tenant == tenant {
			return c, true, nil
		}
	}
	return ports.CellRecord{}, false, nil
}

func law02Requests(r *rand.Rand, route RouterSUT) error {
	names := []string{"acme-logistics", "beta-freight", "gamma-parts"}
	r.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	n := 2 + r.Intn(2)
	var cells []ports.CellRecord
	hostOwner := map[string]string{}
	for _, t := range names[:n] {
		hosts := []string{t + ".sb.example"}
		if r.Intn(2) == 0 {
			hosts = append(hosts, "brain."+t+".example")
		}
		for _, h := range hosts {
			hostOwner[h] = t
		}
		cells = append(cells, ports.CellRecord{Tenant: t, Region: "eu", Endpoint: "local://" + t, Hosts: hosts,
			Custody: cell.CustodyFactory, Placement: cell.PlacementFactoryAccount, KeyCustodian: cell.KeyCustodianFactory})
	}
	hosts := []string{"unknown.sb.example", ""}
	for h := range hostOwner {
		hosts = append(hosts, h)
	}
	for i := 0; i < 20; i++ {
		req := Request{
			Host:        hosts[r.Intn(len(hosts))],
			TokenTenant: append(names, "", "delta-unknown")[r.Intn(len(names)+2)],
			TokenValid:  r.Intn(4) != 0,
		}
		got, err := route(cells, req)
		if err != nil {
			return err
		}
		owner, known := hostOwner[req.Host]
		var want Routed
		switch {
		case !known:
			want = Routed{Code: "EPERM"}
		case !req.TokenValid || req.TokenTenant == "":
			want = Routed{Code: "EAUTH"}
		case req.TokenTenant != owner:
			want = Routed{Code: "ETENANT"}
		default:
			want = Routed{Cell: owner}
		}
		if got.Cell != "" && got.Cell != owner {
			return fmt.Errorf("request for host %q (tenant %q) with a %q token reached cell %q", req.Host, owner, req.TokenTenant, got.Cell)
		}
		if got != want {
			return fmt.Errorf("request %+v: got %+v, want %+v", req, got, want)
		}
	}
	return nil
}
