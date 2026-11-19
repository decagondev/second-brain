package laws

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/url"
	"sort"
	"strings"
)

// LAW-07 against encodings (SB-612, the pen test's first finding): a resolved secret
// can't reach a log line URL-escaped, base64-encoded (either alphabet, padded or not) or
// hex-encoded, and an attribute named like a credential (authorization, cookie,
// credential) never carries its value.

// Encodings are how a careless caller might transform a secret before logging it.
var Encodings = map[string]func(string) string{
	"raw":       func(s string) string { return s },
	"query":     url.QueryEscape,
	"path":      url.PathEscape,
	"b64":       func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) },
	"b64url":    func(s string) string { return base64.URLEncoding.EncodeToString([]byte(s)) },
	"b64raw":    func(s string) string { return base64.RawStdEncoding.EncodeToString([]byte(s)) },
	"b64rawurl": func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) },
	"hex":       func(s string) string { return hex.EncodeToString([]byte(s)) },
}

func law07Encoded(r *rand.Rand, w SecretWorld) error {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=:-_."
	var secrets []string
	for i := 0; i < 1+r.Intn(3); i++ {
		b := make([]byte, 20+r.Intn(20))
		for j := range b {
			b[j] = alphabet[r.Intn(len(alphabet))]
		}
		v := "sbcanary-" + string(b)
		got, err := w.Resolve("acme-logistics", []string{"docs", "chat", "crm"}[i%3], v)
		if err != nil {
			return err
		}
		if got != v {
			return fmt.Errorf("resolved %q, stored %q", got, v)
		}
		secrets = append(secrets, v)
	}
	names := make([]string, 0, len(Encodings))
	for n := range Encodings {
		names = append(names, n)
	}
	sort.Strings(names)
	var evs []LogEvent
	for i := 0; i < 10; i++ {
		s := secrets[r.Intn(len(secrets))]
		enc := Encodings[names[r.Intn(len(names))]](s)
		val := []string{enc, "Basic " + enc, "url?sig=" + enc + "&x=1", "prefix " + enc + " suffix"}[r.Intn(4)]
		evs = append(evs, LogEvent{Path: []string{"msg", "attr", "err", "group", "named", "authorization"}[r.Intn(6)], Value: val})
	}
	out, err := w.Log(evs)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) == "" {
		return fmt.Errorf("the logger wrote nothing; the check would prove nothing")
	}
	for _, s := range secrets {
		for _, n := range names {
			if e := Encodings[n](s); strings.Contains(out, e) {
				return fmt.Errorf("secret %s reached the log as %s (%s):\n%s", s, n, e, out)
			}
		}
	}
	return nil
}
