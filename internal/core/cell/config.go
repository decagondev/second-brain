package cell

import (
	"errors"
	"net/url"
)

// Config is how a cell finds its control plane (sb-tenancy(7), SB-105). It is plain
// configuration, so a cell can be re-pointed (for example at hand-off) without a rebuild.
// There are no defaults: a missing value is an error, never a guess.
type Config struct {
	ControlPlane string // https URL of the control plane
	IdentityCert string // path to the cell's identity certificate
}

// LoadConfig reads SB_CONTROL_PLANE and SB_IDENTITY_CERT through getenv.
func LoadConfig(getenv func(string) string) (Config, error) {
	c := Config{ControlPlane: getenv("SB_CONTROL_PLANE"), IdentityCert: getenv("SB_IDENTITY_CERT")}
	u, err := url.Parse(c.ControlPlane)
	switch {
	case c.ControlPlane == "" || c.IdentityCert == "":
		return Config{}, errors.New("cell config: SB_CONTROL_PLANE and SB_IDENTITY_CERT are both required")
	case err != nil || u.Scheme != "https" || u.Host == "":
		return Config{}, errors.New("cell config: SB_CONTROL_PLANE must be an https URL")
	}
	return c, nil
}
