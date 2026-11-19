package release

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type mem map[string][]byte

func (m mem) Get(_ context.Context, rel, name string) ([]byte, error) {
	b, ok := m[rel+"/"+name]
	if !ok {
		return nil, errors.New("not found")
	}
	return b, nil
}

func signed(t *testing.T, k Ed25519, d string) []byte {
	s, _ := k.Sign(context.Background(), d)
	b, _ := json.Marshal(s)
	return b
}

func setupStore(t *testing.T) (mem, Manifest, Ed25519) {
	k := Ed25519{Priv: ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, 32))}
	b := base()
	art := []byte("sb-binary")
	b.Code["sb"] = SHA256(art)
	m, err := b.Seal()
	if err != nil {
		t.Fatal(err)
	}
	st := mem{}
	raw, _ := json.Marshal(m)
	st[m.ID+"/bundle.json"] = raw
	st[m.ID+"/artefacts/sb"] = art
	d, _ := m.Digest()
	st[m.ID+"/bundle.sig"] = signed(t, k, d)
	g := GateRecord{Release: m.ID, Passed: true, ResultsSHA256: SHA256([]byte("r"))}
	gd, _ := g.Digest()
	graw, _ := json.Marshal(g)
	st[m.ID+"/gate.json"] = graw
	st[m.ID+"/gate.sig"] = signed(t, k, gd)
	return st, m, k
}

func TestAdmitOrderAndRefusals(t *testing.T) {
	ctx := context.Background()
	st, m, k := setupStore(t)
	v := Ed25519Verifier{Trusted: k.Public()}
	if _, arts, err := Admit(ctx, st, m.ID, v, []string{"sb"}); err != nil || string(arts["sb"]) != "sb-binary" {
		t.Fatalf("good release: %v", err)
	}
	other := Ed25519{Priv: ed25519.NewKeyFromSeed(bytes.Repeat([]byte{8}, 32))}
	cases := map[string]func(mem){
		"unsigned":     func(s mem) { delete(s, m.ID+"/bundle.sig") },
		"someone else": func(s mem) { d, _ := m.Digest(); s[m.ID+"/bundle.sig"] = signed(t, other, d) },
		"edited bundle": func(s mem) {
			s[m.ID+"/bundle.json"] = bytes.Replace(s[m.ID+"/bundle.json"], []byte(`"local-dev"`), []byte(`"other"`), 1)
		},
		"swapped artefact": func(s mem) { s[m.ID+"/artefacts/sb"] = []byte("evil") },
		"no gate":          func(s mem) { delete(s, m.ID+"/gate.json") },
		"failed gate": func(s mem) {
			g := GateRecord{Release: m.ID, Passed: false}
			b, _ := json.Marshal(g)
			s[m.ID+"/gate.json"] = b
		},
		"gate for another release": func(s mem) {
			g := GateRecord{Release: "rel_2027-01-01_aaaaaa", Passed: true}
			b, _ := json.Marshal(g)
			s[m.ID+"/gate.json"] = b
		},
		"unsigned gate edit": func(s mem) {
			g := GateRecord{Release: m.ID, Passed: true, ResultsSHA256: "forged"}
			b, _ := json.Marshal(g)
			s[m.ID+"/gate.json"] = b
		},
	}
	for name, f := range cases {
		s := mem{}
		for k2, v2 := range st {
			s[k2] = append([]byte(nil), v2...)
		}
		f(s)
		if _, _, err := Admit(ctx, s, m.ID, v, []string{"sb"}); !errors.Is(err, ErrUnsigned) {
			t.Errorf("%s: admitted (%v)", name, err)
		}
	}
	if _, _, err := Admit(ctx, st, m.ID, v, []string{"web"}); err == nil || !strings.Contains(err.Error(), "isn't in the bundle") {
		t.Errorf("unlisted artefact: %v", err)
	}
}
