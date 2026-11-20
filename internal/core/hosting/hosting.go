// Package hosting generates and checks a tenant's web UI hosting configuration (sb-runtime(7),
// Web UI hosting, exactly; SB-906). The site lives in the cell's own project and rewrites
// /v1/** to that cell's api only, so no site can reach another tenant's cell (LAW-02).
package hosting

import (
	"fmt"
	"strings"
)

// Run is a rewrite to a Cloud Run service.
type Run struct {
	ServiceID string `json:"serviceId"`
	Region    string `json:"region"`
}

// Rewrite is one rewrite rule.
type Rewrite struct {
	Source      string `json:"source"`
	Run         *Run   `json:"run,omitempty"`
	Destination string `json:"destination,omitempty"`
	Function    string `json:"function,omitempty"`
}

// Header is one header rule.
type Header struct {
	Source  string            `json:"source"`
	Headers map[string]string `json:"headers"`
}

// Config is the Firebase Hosting configuration of one site.
type Config struct {
	Site     string    `json:"site"`
	Project  string    `json:"project"`
	Public   string    `json:"public"`
	Rewrites []Rewrite `json:"rewrites"`
	Headers  []Header  `json:"headers"`
}

// Generate makes the one allowed configuration for a tenant's cell.
func Generate(tenant, project, region string) Config {
	return Config{
		Site: "sb-" + tenant, Project: project, Public: "web/dist",
		Rewrites: []Rewrite{
			{Source: "/v1/**", Run: &Run{ServiceID: "api", Region: region}},
			{Source: "**", Destination: "/index.html"},
		},
		Headers: []Header{{Source: "**", Headers: map[string]string{
			"Content-Security-Policy": "default-src 'self'; frame-ancestors 'none'", // as sb-ask(3), The web UI
			"X-Content-Type-Options":  "nosniff",
			"Referrer-Policy":         "no-referrer",
		}}},
	}
}

// Check refuses any configuration that could send a request anywhere but this cell's api.
func Check(c Config, tenant, cellProject, region string) error {
	if c.Site != "sb-"+tenant {
		return fmt.Errorf("hosting: site %q isn't sb-%s", c.Site, tenant)
	}
	if c.Project != cellProject {
		return fmt.Errorf("hosting: site %s is in %s, not the cell's project %s", c.Site, c.Project, cellProject)
	}
	api := false
	for _, r := range c.Rewrites {
		switch {
		case r.Function != "":
			return fmt.Errorf("hosting: %s rewrites to a function; only the cell's api is allowed", r.Source)
		case r.Run != nil:
			if r.Run.ServiceID != "api" || r.Run.Region != region {
				return fmt.Errorf("hosting: %s rewrites to %s in %s; only api in %s", r.Source, r.Run.ServiceID, r.Run.Region, region)
			}
			if r.Source != "/v1/**" {
				return fmt.Errorf("hosting: only /v1/** may reach the api, not %s", r.Source)
			}
			api = true
		case strings.Contains(r.Destination, "://") || strings.HasPrefix(r.Destination, "//"):
			return fmt.Errorf("hosting: %s rewrites to another host", r.Source)
		case !strings.HasPrefix(r.Destination, "/"):
			return fmt.Errorf("hosting: %s has a bad destination", r.Source)
		}
	}
	if !api {
		return fmt.Errorf("hosting: no rewrite sends /v1/** to the cell's api")
	}
	return nil
}
