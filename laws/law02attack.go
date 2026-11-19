package laws

import (
	"fmt"
	"math/rand"
	"strings"

	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/ports"
)

// LAW-02 against hostile Host headers (SB-612): a host that only differs from a cell's by
// case, a trailing dot or a port reaches that cell (with that tenant's token) and no other;
// every other shape a tester tries (suffixes, prefixes, userinfo, look-alike letters,
// percent-encoding, whitespace, a second host) is refused with EPERM, whatever the token.

func law02Attacks(r *rand.Rand, route RouterSUT) error {
	names := []string{"acme-logistics", "beta-freight", "gamma-parts"}
	var cells []ports.CellRecord
	for _, t := range names {
		cells = append(cells, ports.CellRecord{Tenant: t, Region: "eu", Endpoint: "local://" + t, Hosts: []string{t + ".sb.example"},
			Custody: cell.CustodyFactory, Placement: cell.PlacementFactoryAccount, KeyCustodian: cell.KeyCustodianFactory})
	}
	same := []func(string) string{
		strings.ToUpper,
		func(h string) string { return h + "." },
		func(h string) string { return h + ":443" },
		func(h string) string { return strings.ToUpper(h[:1]) + h[1:] + ".:8443" },
	}
	other := []func(h, o string) string{
		func(h, o string) string { return h + ".evil.example" },
		func(h, o string) string { return "evil." + h },
		func(h, o string) string { return h + "@" + o },
		func(h, o string) string { return o + "@" + h },
		func(h, o string) string { return strings.Replace(h, "a", "а", 1) }, // Cyrillic a
		func(h, o string) string { return strings.Replace(h, ".", "%2e", 1) },
		func(h, o string) string { return " " + h },
		func(h, o string) string { return h + "," + o },
		func(h, o string) string { return h + "\x00" + o },
		func(h, o string) string { return "xn--" + h },
		func(h, o string) string { return h + "sb.example" },
	}
	for i := 0; i < 20; i++ {
		owner, rival := names[r.Intn(3)], names[r.Intn(3)]
		token := names[r.Intn(3)]
		h, o := owner+".sb.example", rival+".sb.example"
		var host string
		var want Routed
		if r.Intn(2) == 0 {
			host = same[r.Intn(len(same))](h)
			want = Routed{Code: "ETENANT"}
			if token == owner {
				want = Routed{Cell: owner}
			}
		} else {
			host = other[r.Intn(len(other))](h, o)
			want = Routed{Code: "EPERM"}
		}
		got, err := route(cells, Request{Host: host, TokenTenant: token, TokenValid: true})
		if err != nil {
			return err
		}
		if got.Cell != "" && got.Cell != owner {
			return fmt.Errorf("host %q with a %s token reached %s's cell", host, token, got.Cell)
		}
		if got != want {
			return fmt.Errorf("host %q with a %s token: got %+v, want %+v", host, token, got, want)
		}
	}
	return nil
}
