package laws

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"

	relfs "example.com/second-brain/internal/adapters/release/localfs"
	"example.com/second-brain/internal/core/release"
)

// The real release path (SB-503, SB-504, SB-506): bundles sealed with their ids, signed
// with the release key, gated with a signed record, stored, and admitted by a cell that
// trusts only the release key.
func init() {
	NewReleaseWorld = func() ReleaseWorld { return newReleaseWorld(false) }
}

type releaseWorld struct {
	store   relfs.Dir
	signer  release.Ed25519
	forger  release.Ed25519
	n       int
	art     map[string][]byte // release -> its "sb" artefact
	trusted release.Verifier
	// lax is a mutation: the cell checks the signature but not the gate record.
	lax bool
}

func newReleaseWorld(lax bool) *releaseWorld {
	dir, err := os.MkdirTemp("", "sb-law06-")
	if err != nil {
		panic(err)
	}
	w := &releaseWorld{store: relfs.Dir{Root: dir}, art: map[string][]byte{}, lax: lax,
		signer: release.Ed25519{Priv: ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, 32))},
		forger: release.Ed25519{Priv: ed25519.NewKeyFromSeed(bytes.Repeat([]byte{2}, 32))}}
	w.trusted = release.Ed25519Verifier{Trusted: w.signer.Public()}
	return w
}

func (w *releaseWorld) bundle(change string) (release.Bundle, []byte) {
	w.n++
	art := []byte(fmt.Sprintf("sb binary build %d", w.n))
	b := release.Bundle{
		Code:     map[string]string{"sb": release.SHA256(art)},
		Prompts:  map[string]string{"answer": release.SHA256([]byte("answer"))},
		Tools:    []release.Tool{{Name: "ticket-update.comment", Connector: "tickets", Consequential: true, Workflow: "ticket-update", SchemaSHA256: release.SHA256([]byte("schema"))}},
		Policies: map[string]string{"policy/platform/base.cedar": release.SHA256([]byte("base"))},
		Model:    release.Model{Provider: "openaicompat", Model: "local-dev", MaxTokens: 1024},
		Gate:     release.SHA256([]byte("gate")),
		Created:  "2027-03-02",
	}
	b.Code["sb"] = release.SHA256([]byte("sb binary build 0")) // the same artefact unless code changes
	art = []byte("sb binary build 0")
	switch change {
	case "prompt":
		b.Prompts["answer"] = release.SHA256([]byte("answer, edited"))
	case "policy":
		b.Policies["policy/platform/base.cedar"] = release.SHA256([]byte("base, edited"))
	case "tool":
		b.Tools[0].SchemaSHA256 = release.SHA256([]byte("schema, edited"))
	case "model":
		b.Model.Model = "local-dev-2"
	case "code":
		art = []byte("sb binary build 1")
		b.Code["sb"] = release.SHA256(art)
	case "gate":
		b.Gate = release.SHA256([]byte("gate, edited"))
	case "fresh":
		b.Model.MaxTokens = 1024 + w.n // never built before
	}
	return b, art
}

func (w *releaseWorld) Build(change string) (string, error) {
	b, art := w.bundle(change)
	m, err := b.Seal()
	if err != nil {
		return "", err
	}
	raw, _ := json.Marshal(m)
	if err := w.store.Put(m.ID, "bundle.json", raw); err != nil {
		return "", err
	}
	if err := w.store.Put(m.ID, "artefacts/sb", art); err != nil {
		return "", err
	}
	w.art[m.ID] = art
	return m.ID, nil
}

func (w *releaseWorld) manifest(id string) (release.Manifest, error) {
	raw, err := w.store.Get(context.Background(), id, "bundle.json")
	if err != nil {
		return release.Manifest{}, err
	}
	var m release.Manifest
	return m, json.Unmarshal(raw, &m)
}

func (w *releaseWorld) Sign(id string) error {
	m, err := w.manifest(id)
	if err != nil {
		return err
	}
	d, err := m.Digest()
	if err != nil {
		return err
	}
	s, _ := w.signer.Sign(context.Background(), d)
	raw, _ := json.Marshal(s)
	return w.store.Put(id, "bundle.sig", raw)
}

func (w *releaseWorld) Gate(id string, pass bool) error {
	g := release.GateRecord{Release: id, Passed: pass, ResultsSHA256: release.SHA256([]byte("results " + id))}
	d, err := g.Digest()
	if err != nil {
		return err
	}
	s, _ := w.signer.Sign(context.Background(), d)
	graw, _ := json.Marshal(g)
	sraw, _ := json.Marshal(s)
	if err := w.store.Put(id, "gate.json", graw); err != nil {
		return err
	}
	return w.store.Put(id, "gate.sig", sraw)
}

// tamper changes something after signing, as an attacker with write access to the
// store would.
func (w *releaseWorld) tamper(id, what string) error {
	m, err := w.manifest(id)
	if err != nil {
		return err
	}
	switch what {
	case "code":
		return w.store.Put(id, "artefacts/sb", []byte("a different binary"))
	case "gate":
		g := release.GateRecord{Release: id, Passed: true, ResultsSHA256: "forged"}
		raw, _ := json.Marshal(g)
		return w.store.Put(id, "gate.json", raw)
	case "model":
		d, _ := m.Digest()
		s, _ := w.forger.Sign(context.Background(), d) // re-signed with someone else's key
		raw, _ := json.Marshal(s)
		return w.store.Put(id, "bundle.sig", raw)
	default: // prompt, policy, tool: edit the bundle in place
		m.Prompts = map[string]string{"answer": release.SHA256([]byte("swapped prompt"))}
		raw, _ := json.Marshal(m)
		return w.store.Put(id, "bundle.json", raw)
	}
}

func (w *releaseWorld) Admit(id, tamper string) (bool, error) {
	if tamper != "" {
		if err := w.tamper(id, tamper); err != nil {
			return false, err
		}
	}
	if w.lax {
		m, err := w.manifest(id)
		if err != nil {
			return false, nil
		}
		d, _ := m.Digest()
		raw, err := w.store.Get(context.Background(), id, "bundle.sig")
		if err != nil {
			return false, nil
		}
		var s release.Signature
		_ = json.Unmarshal(raw, &s)
		return m.Check() == nil && w.trusted.Verify(context.Background(), d, s) == nil, nil
	}
	_, _, err := release.Admit(context.Background(), w.store, id, w.trusted, []string{"sb"})
	return err == nil, nil
}
