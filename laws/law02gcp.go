package laws

import (
	"context"
	"time"

	"example.com/second-brain/internal/adapters/placement/gcp"
	"example.com/second-brain/internal/adapters/placement/gcp/gcptest"
	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/manifest"
)

// LAW-02 at the cell level on GCP (SB-703): the same isolation law as law02Cells, with
// cells stamped into GCP projects through the gcp adapter and its recorded fake. Each
// project has its own deploy account, granted only the adapter's permissions there.

// NewGCPCellStamper builds the system under test over one fake; close it when done.
var NewGCPCellStamper = func() (CellStamper, func()) {
	f := gcptest.New()
	return func(account string, t manifest.Tenant) ([]Stamped, error) {
		f.Grant("tok-"+account, account, gcp.DeployPermissions...)
		p := gcp.Project{ID: account, HTTP: f.Server.Client(), Poll: time.Millisecond,
			Token:     func(context.Context) (string, error) { return "tok-" + account, nil },
			Endpoints: gcp.Endpoints{SQL: f.URL(), KMS: f.URL(), Secrets: f.URL(), PubSub: f.URL()}}
		rs, err := cell.Stamp(context.Background(), p, p, t)
		if err != nil {
			return nil, err
		}
		out := make([]Stamped, len(rs))
		for i, r := range rs {
			out[i] = Stamped{Kind: r.Kind, Name: r.Name, ID: r.ID, Tenant: r.Tenant, Region: r.Region}
		}
		return out, nil
	}, f.Server.Close
}
