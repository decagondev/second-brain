package gcp

import (
	"context"
	"net/http"

	"example.com/second-brain/internal/core/hosting"
)

// Hosting publishes a tenant's web UI configuration to its Firebase Hosting site, in the
// cell's own project (sb-runtime(7), Web UI hosting; SB-906). It checks the configuration
// before it calls the API, and never publishes one hosting.Check refuses.
type Hosting struct {
	Project Project
	Base    string // Firebase Hosting API; "" is Google's
}

// HostingPermissions are what Hosting uses.
var HostingPermissions = []string{"firebasehosting.sites.update"}

func (h Hosting) base() string {
	if h.Base == "" {
		return "https://firebasehosting.googleapis.com"
	}
	return h.Base
}

// Publish creates a version with the config, finalizes it, and releases it.
func (h Hosting) Publish(ctx context.Context, tenant, region string, c hosting.Config) (string, error) {
	if err := hosting.Check(c, tenant, h.Project.ID, region); err != nil {
		return "", err
	}
	site := h.base() + "/v1beta1/sites/" + c.Site
	var v struct {
		Name string `json:"name"`
	}
	cfg := map[string]any{"rewrites": c.Rewrites, "headers": c.Headers}
	if err := h.Project.do(ctx, http.MethodPost, site+"/versions?project="+h.Project.ID, map[string]any{"config": cfg}, &v); err != nil {
		return "", err
	}
	if err := h.Project.do(ctx, http.MethodPatch, h.base()+"/v1beta1/"+v.Name+"?updateMask=status", map[string]any{"status": "FINALIZED"}, nil); err != nil {
		return "", err
	}
	var rel struct {
		Name string `json:"name"`
	}
	if err := h.Project.do(ctx, http.MethodPost, site+"/releases?versionName="+v.Name, map[string]any{}, &rel); err != nil {
		return "", err
	}
	return rel.Name, nil
}
