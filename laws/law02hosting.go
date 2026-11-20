package laws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/placement/gcp"
	"example.com/second-brain/internal/adapters/placement/gcp/gcptest"
	"example.com/second-brain/internal/core/hosting"
	"example.com/second-brain/internal/core/runtime"
)

// LAW-02 at the front door (SB-906): each tenant's web UI site lives in its cell's project
// and can only send /v1/** to that cell's api. Whatever configuration someone tries to
// publish, no released site rewrites to another project, another service or another host.

// Publisher publishes a hosting config for tenant through its own cell's credentials.
type Publisher func(ctx context.Context, h gcp.Hosting, tenant, region string, c hosting.Config) error

// NewPublisher builds the system under test: gcp.Hosting.Publish, which checks first.
var NewPublisher = func() Publisher {
	return func(ctx context.Context, h gcp.Hosting, tenant, region string, c hosting.Config) error {
		_, err := h.Publish(ctx, tenant, region, c)
		return err
	}
}

func law02Hosting(r *rand.Rand, publish Publisher) error {
	f := gcptest.New()
	defer f.Server.Close()
	ctx := context.Background()
	cells := map[string]string{"gamma-parts": rtProjectA, "delta-tools": rtProjectB}
	hosts := map[string]gcp.Hosting{}
	for t, p := range cells {
		tok := "tok-" + t
		f.Grant(tok, p, append(append([]string{}, gcp.UpdaterPermissions...), gcp.HostingPermissions...)...)
		proj := gcp.Project{ID: p, HTTP: f.Server.Client(), Poll: time.Millisecond, Token: func(context.Context) (string, error) { return tok, nil }}
		cr := gcp.CloudRun{Project: proj, Location: "europe-west1", Run: f.URL(), Analysis: f.URL()}
		img := rtRegistry + "/api@sha256:" + strings.Repeat("c", 64)
		if err := cr.Attest(ctx, img); err != nil {
			return err
		}
		if err := cr.Deploy(ctx, runtime.Services[0], img, "rel_x"); err != nil {
			return err
		}
		f.Site(p, "sb-"+t)
		hosts[t] = gcp.Hosting{Project: proj, Base: f.URL()}
	}
	for t := range cells {
		if err := publish(ctx, hosts[t], t, "europe-west1", hosting.Generate(t, cells[t], "europe-west1")); err != nil {
			return fmt.Errorf("the generated config for %s was refused: %v", t, err)
		}
	}
	// An attempt: tenant a publishes something else.
	a, b := "gamma-parts", "delta-tools"
	if r.Intn(2) == 0 {
		a, b = b, a
	}
	c := hosting.Generate(a, cells[a], "europe-west1")
	switch r.Intn(4) {
	case 0:
		c.Project = cells[b]
	case 1:
		c.Rewrites[1].Destination = "https://sb-" + b + ".web.app/index.html"
	case 2:
		c.Rewrites = append(c.Rewrites, hosting.Rewrite{Source: "/other/**", Destination: "//sb-" + b + ".web.app"})
	default:
		c.Site = "sb-" + b
	}
	_ = publish(ctx, hosts[a], a, "europe-west1", c) // refused or harmless; the law judges what's released
	for t, p := range cells {
		cfg := f.Released["sb-"+t]
		if cfg == nil {
			return fmt.Errorf("%s's site has no release", t)
		}
		rws, _ := cfg["rewrites"].([]any)
		for _, x := range rws {
			rw, _ := x.(map[string]any)
			if dest, _ := rw["destination"].(string); strings.Contains(dest, "//") {
				return fmt.Errorf("%s's site (project %s) rewrites to another host: %s", t, p, dest)
			}
		}
	}
	return nil
}

// publishRaw is the mutation's publisher: the same three Hosting API calls, with no check.
func publishRaw(ctx context.Context, h gcp.Hosting, c hosting.Config) error {
	tok, _ := h.Project.Token(ctx)
	call := func(method, u string, body any, out any) error {
		b, _ := json.Marshal(body)
		req, _ := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := h.Project.HTTP.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		if out != nil {
			return json.NewDecoder(resp.Body).Decode(out)
		}
		return nil
	}
	site := h.Base + "/v1beta1/sites/" + c.Site
	var v struct{ Name string }
	if err := call(http.MethodPost, site+"/versions", map[string]any{"config": map[string]any{"rewrites": c.Rewrites, "headers": c.Headers}}, &v); err != nil {
		return err
	}
	if err := call(http.MethodPatch, h.Base+"/v1beta1/"+v.Name+"?updateMask=status", map[string]any{"status": "FINALIZED"}, nil); err != nil {
		return err
	}
	return call(http.MethodPost, site+"/releases?versionName="+v.Name, map[string]any{}, nil)
}
