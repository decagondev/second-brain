package laws

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"example.com/second-brain/internal/adapters/placement/gcp"
	"example.com/second-brain/internal/adapters/placement/gcp/gcptest"
	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
)

// LAW-02 across two perimeters (SB-902): each cell's project sits in its own VPC Service
// Controls perimeter that lets only its own service accounts through. Even when IAM is
// wrongly granted across cells, one cell's identity can't read or decrypt the other's data.

// PerimeterWorld is two cell projects, each stamped and fenced.
type PerimeterWorld interface {
	// Reach has cell a's identity try one restricted call on cell b's project.
	Reach(a, b string, call string) error
}

// NewPerimeterWorld builds the system under test over the GCP fake.
var NewPerimeterWorld = func(mistake bool) (PerimeterWorld, func(), error) {
	f := gcptest.New()
	w := &perimeterWorld{f: f, projects: map[string]gcp.Project{}}
	for _, c := range []struct{ name, project string }{{"gamma-parts", rtProjectA}, {"delta-tools", rtProjectB}} {
		tok := "tok-" + c.name
		f.Grant(tok, c.project, gcp.DeployPermissions...)
		f.Principal(tok, "serviceAccount:"+gcp.ServiceAccount(c.project, "api"))
		p := gcp.Project{ID: c.project, HTTP: f.Server.Client(), Poll: time.Millisecond,
			Token:     func(context.Context) (string, error) { return tok, nil },
			Endpoints: gcp.Endpoints{SQL: f.URL(), KMS: f.URL(), Secrets: f.URL(), PubSub: f.URL(), CRM: f.URL()}}
		if _, err := cell.Stamp(context.Background(), p, p, manifest.Tenant{Tenant: c.name, Region: "eu"}); err != nil {
			f.Server.Close()
			return nil, nil, err
		}
		f.Perimeter(c.project, "serviceAccount:"+gcp.ServiceAccount(c.project, "api"))
		w.projects[c.name] = p
		w.tenants = append(w.tenants, c.name)
	}
	if mistake { // someone grants cell a's identity full rights on cell b's project
		f.Grant("tok-gamma-parts", rtProjectB, gcp.DeployPermissions...)
	}
	return w, f.Server.Close, nil
}

type perimeterWorld struct {
	f        *gcptest.Fake
	projects map[string]gcp.Project
	tenants  []string
}

func (w *perimeterWorld) Reach(a, b, call string) error {
	p := w.projects[b]
	p.Token = w.projects[a].Token
	ctx := context.Background()
	switch call {
	case "database":
		_, err := p.Ensure(ctx, cellResource("database", "sb_"+tenantUnderscore(b), b))
		return err
	case "decrypt":
		_, err := p.Unwrap(ctx, "projects/"+p.ID+"/locations/europe-west1/keyRings/sb-"+b+"/cryptoKeys/cell-"+b+"-root", []byte("x"))
		return err
	default:
		_, err := p.Ensure(ctx, cellResource("vault-namespace", "tenants/"+b, b))
		return err
	}
}

func law02Perimeter(r *rand.Rand, mistake bool) error {
	w, done, err := NewPerimeterWorld(mistake)
	if err != nil {
		return err
	}
	defer done()
	pw := w.(*perimeterWorld)
	a, b := pw.tenants[0], pw.tenants[1]
	if r.Intn(2) == 0 {
		a, b = b, a
	}
	if mistake && a != "gamma-parts" {
		a, b = b, a // the wrong grant goes from gamma to delta
	}
	call := []string{"database", "decrypt", "secrets"}[r.Intn(3)]
	if err := w.Reach(a, b, call); err == nil {
		return fmt.Errorf("%s's identity made a %s call in %s's project (IAM mistake %v)", a, call, b, mistake)
	}
	if err := w.Reach(b, b, "database"); err != nil {
		return fmt.Errorf("%s couldn't reach its own database inside its perimeter: %v", b, err)
	}
	return nil
}

func tenantUnderscore(t string) string {
	out := []byte(t)
	for i, c := range out {
		if c == '-' {
			out[i] = '_'
		}
	}
	return string(out)
}

func cellResource(kind, name, tenant string) ports.CellResource {
	return ports.CellResource{Kind: kind, Name: name, Tenant: tenant, Region: "eu"}
}
