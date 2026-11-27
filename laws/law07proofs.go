package laws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/msgraph"
	"example.com/second-brain/internal/adapters/msgraph/graphtest"
	"example.com/second-brain/internal/cell/prove"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/proof"
)

// LAW-07 for proofs (SB-6001; sb-laws(7)): whatever a real system holds, a signed proof of
// a ticket carries none of it: no document text, no name, no address and no unhashed id.
// Random drives, items, people and mailboxes on the recorded Graph, through the real runner
// and sealer.

// ProofMaker is what the law checks: a ticket's proof of env, as the file sb prove writes.
type ProofMaker func(ctx context.Context, ticket string, env prove.Env) ([]byte, error)

// RealProofMaker is the real one.
var RealProofMaker ProofMaker = func(ctx context.Context, ticket string, env prove.Env) ([]byte, error) {
	p, err := prove.Run(ctx, ticket, env)
	if err != nil {
		return nil, err
	}
	sg, err := audit.NewSigner(env.Tenant, make([]byte, 32))
	if err != nil {
		return nil, err
	}
	s, err := proof.Seal(p, sg, time.Now())
	if err != nil {
		return nil, err
	}
	return json.Marshal(s)
}

func law07Proofs(r *rand.Rand, mk ProofMaker) error {
	f := graphtest.New()
	defer f.Server.Close()
	word := func(p string) string { return fmt.Sprintf("%s%04d", p, r.Intn(10000)) }
	var secrets []string
	drive := word("drv")
	secrets = append(secrets, drive)
	person := word("pers") + "@acme.example"
	secrets = append(secrets, person)
	for i := 0; i < 1+r.Intn(6); i++ {
		it := graphtest.Item{ID: word("item"), Name: word("name") + ".md", Path: "/" + word("folder"), Owner: word("own") + "@acme.example",
			Content: []byte("text " + word("body")), Label: word("lbl")}
		switch r.Intn(3) {
		case 0:
			it.Users = []string{person}
		case 1:
			g := word("grp")
			f.Groups[g] = []string{person}
			f.Mails[g] = word("gmail") + "@acme.example"
			it.Groups = []string{g}
			secrets = append(secrets, g, f.Mails[g])
		default:
			it.Links = []string{"anonymous"}
		}
		f.Put(drive, it)
		secrets = append(secrets, it.ID, it.Name, it.Path[1:], it.Owner, string(it.Content[5:]), it.Label)
	}
	box := word("box") + "@acme.example"
	f.Boxes[box] = []graphtest.Message{{ID: word("msg"), From: word("from") + "@client.example", Subject: word("subj"), Body: word("mbody")}}
	secrets = append(secrets, box, f.Boxes[box][0].ID, f.Boxes[box][0].From, f.Boxes[box][0].Subject, f.Boxes[box][0].Body)
	g := &msgraph.Client{Base: f.Base(), Login: f.Login(), HTTP: f.Server.Client(), Creds: func(context.Context) (string, error) { return graphtest.Creds, nil }}
	sp := &msgraph.SharePoint{Graph: g, Drives: []string{drive}, Domain: "acme.example", Dir: &msgraph.Directory{Graph: g},
		Readers: msgraph.Readers{Text: func(_ string, b []byte) (string, bool) { return string(b), true }}}
	env := prove.Env{Tenant: "acme-logistics", Graph: g, SharePoint: sp, Mailboxes: []string{box}, Probe: word("probe") + "@acme.example", Person: person}
	for _, t := range []string{"SB-5901", "SB-5902", "SB-5903", "SB-5907"} {
		b, err := mk(context.Background(), t, env)
		if err != nil {
			return fmt.Errorf("%s: %v", t, err)
		}
		raw := string(b)
		for _, s := range secrets {
			if s != "" && strings.Contains(raw, s) {
				return fmt.Errorf("%s: the proof holds %q", t, s)
			}
		}
		if enc := base64.StdEncoding.EncodeToString([]byte(person)); strings.Contains(raw, enc) {
			return fmt.Errorf("%s: the proof holds a name, encoded", t)
		}
	}
	return nil
}
