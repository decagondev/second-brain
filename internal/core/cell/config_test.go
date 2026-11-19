package cell

import "testing"

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestConfig(t *testing.T) {
	a, err := LoadConfig(env(map[string]string{"SB_CONTROL_PLANE": "https://factory.sb.example", "SB_IDENTITY_CERT": "/etc/sb/cell.pem"}))
	if err != nil || a.ControlPlane != "https://factory.sb.example" {
		t.Fatalf("%+v %v", a, err)
	}
	// Re-pointing is a config change only.
	b, err := LoadConfig(env(map[string]string{"SB_CONTROL_PLANE": "https://cp.tenant.example", "SB_IDENTITY_CERT": "/etc/sb/cell.pem"}))
	if err != nil || b.ControlPlane == a.ControlPlane {
		t.Fatalf("%+v %v", b, err)
	}
	for _, m := range []map[string]string{
		{},
		{"SB_CONTROL_PLANE": "https://factory.sb.example"},
		{"SB_CONTROL_PLANE": "http://factory.sb.example", "SB_IDENTITY_CERT": "/c.pem"},
		{"SB_CONTROL_PLANE": "factory", "SB_IDENTITY_CERT": "/c.pem"},
	} {
		if _, err := LoadConfig(env(m)); err == nil {
			t.Errorf("%v: want an error", m)
		}
	}
}
