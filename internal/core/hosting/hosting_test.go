package hosting

import "testing"

func TestGenerateIsAccepted(t *testing.T) {
	if err := Check(Generate("gamma-parts", "gamma-prod-7731", "europe-west1"), "gamma-parts", "gamma-prod-7731", "europe-west1"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRefusesEveryOtherRoute(t *testing.T) {
	bad := map[string]func(c *Config){
		"another project":   func(c *Config) { c.Project = "delta-prod-1204" },
		"another site name": func(c *Config) { c.Site = "sb-delta-tools" },
		"another service":   func(c *Config) { c.Rewrites[0].Run.ServiceID = "ingest" },
		"another region":    func(c *Config) { c.Rewrites[0].Run.Region = "us-central1" },
		"a function":        func(c *Config) { c.Rewrites = append(c.Rewrites, Rewrite{Source: "/x", Function: "proxy"}) },
		"an absolute URL":   func(c *Config) { c.Rewrites[1].Destination = "https://evil.example/index.html" },
		"a scheme-less URL": func(c *Config) { c.Rewrites[1].Destination = "//evil.example/" },
		"no api rewrite":    func(c *Config) { c.Rewrites = c.Rewrites[1:] },
		"api on all paths":  func(c *Config) { c.Rewrites[0].Source = "**" },
	}
	for name, mutate := range bad {
		c := Generate("gamma-parts", "gamma-prod-7731", "europe-west1")
		mutate(&c)
		if err := Check(c, "gamma-parts", "gamma-prod-7731", "europe-west1"); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}
