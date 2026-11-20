package laws

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"example.com/second-brain/internal/adapters/connectors/github"
	"example.com/second-brain/internal/adapters/connectors/github/githubtest"
	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/obs"
)

// The GitHub code connector under the connector laws (SB-1103): LAW-03 (anything GitHub
// can't answer is Unknown), LAW-01 (the live permission decides, never the copy taken at
// fetch), and LAW-07 at ingest (no chunk holding a secret is ever stored).

func newGitHub() (*githubtest.Fake, *github.Connector) {
	f := githubtest.New()
	f.AddMember("ann", "ann@acme.example")
	f.AddMember("bob", "bob@acme.example")
	c := &github.Connector{Base: f.Base(), HTTP: f.Server.Client(), Redact: obs.NewRedactor(),
		Creds: func(context.Context) (string, error) { return f.CredsJSON(), nil }}
	return f, c
}

// NewGitHubWorld is the connector-level world: access means ann can read the issue's
// private repository; "script" removes the first ref's repository.
var NewGitHubWorld = func() ConnectorWorld {
	f, c := newGitHub()
	f.AddRepo("in", "private")
	f.AddRepo("out", "private")
	f.Grant("in", "ann", "read")
	f.Grant("out", "ann", "none")
	return &githubWorld{f: f, c: c}
}

type githubWorld struct {
	f *githubtest.Fake
	c *github.Connector
	n int
}

func (w *githubWorld) ACL(access []bool, fault string) ([]string, error) {
	w.n++
	refs := make([]ports.SourceRef, len(access))
	repos := map[string]bool{}
	for i, ok := range access {
		repo := "out"
		if ok {
			repo = "in"
		}
		if fault == "script" && i == 0 {
			repo = fmt.Sprintf("g%d", w.n)
			w.f.AddRepo(repo, "private")
			w.f.Grant(repo, "ann", "read")
		}
		repos[repo] = true
		w.f.PutIssue(repo, 1000*w.n+i, "t", "x", false)
		refs[i] = ports.SourceRef{Kind: "code", ID: fmt.Sprintf("%s/issues/%d", repo, 1000*w.n+i)}
	}
	switch fault {
	case "script":
		w.f.Remove(refs[0].ID[:strings.Index(refs[0].ID, "/")])
	case "":
	default:
		for range repos { // one permission call per repository
			w.f.FailNext(fault)
		}
	}
	vs, err := w.c.ACL(context.Background(), ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}, refs)
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.String()
	}
	return out, err
}

func (w *githubWorld) Act(key string, _ bool) (string, error) {
	r, err := w.c.Act(context.Background(), ports.Action{Target: "code:x"}, ports.IdempotencyKey(key))
	return r.TargetID, err
}

func (w *githubWorld) Effects() int { return 0 }

// LiveCheck asks whether ann may read a document now (the connector's ACL in the real system).
type LiveCheck func(c *github.Connector, ref ports.SourceRef) ports.Verdict

func liveACL(c *github.Connector, ref ports.SourceRef) ports.Verdict {
	vs, err := c.ACL(context.Background(), ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}, []ports.SourceRef{ref})
	if err != nil || len(vs) != 1 {
		return ports.VerdictUnknown
	}
	return vs[0]
}

// law01GitHub: grants and revocations after a fetch; the verdict always follows GitHub now.
func law01GitHub(r *rand.Rand, check LiveCheck) error {
	f, c := newGitHub()
	f.AddRepo("payments", "private")
	f.PutIssue("payments", 1, "Stalls", "The queue stalls.", false)
	ref := ports.SourceRef{Kind: "code", ID: "payments/issues/1"}
	can := r.Intn(2) == 0
	f.Grant("payments", "ann", map[bool]string{true: "read", false: "none"}[can])
	if _, err := c.Fetch(context.Background(), ref); err != nil {
		return err
	}
	for i := 0; i < 6; i++ {
		can = r.Intn(2) == 0
		f.Grant("payments", "ann", map[bool]string{true: "write", false: "none"}[can])
		want := map[bool]ports.Verdict{true: ports.VerdictAllow, false: ports.VerdictDeny}[can]
		if got := check(c, ref); got != want {
			return fmt.Errorf("after GitHub set ann's access to %v, the check said %v", can, got)
		}
	}
	return nil
}

// Scanner is ingest's secret scanner (ingest.SecretIn in the real system).
type Scanner func(text string) string

// canaries are fake credentials, assembled from parts so the repository never holds a
// string a secret scanner would take for a real key.
var canaries = []string{
	"-----BEGIN RSA " + "PRIVATE KEY-----",
	"AKIA" + "QWERTYUIOPASDFGH",
	"gh" + "p_0123456789abcdefghijklmnopqrstuvwxyzAB",
	"xo" + "xb-998877665544-qwertyuiopas",
	"client_" + `secret = "9f8e7d6c5b4a39281706"`,
}

// law07Ingest: documents with secrets planted in random paragraphs; nothing stored holds
// one, and documents without secrets are indexed in full.
func law07Ingest(r *rand.Rand, scan Scanner) error {
	ctx := context.Background()
	src := mock.NewSource("code")
	idx := memindex.New()
	dropped := 0
	p := ingest.Pipeline{Tenant: "acme-logistics", Release: "rel_test", Connector: mock.New(src, false),
		Embedder: hashing.Embedder{}, Index: idx, Secrets: scan, SecretDropped: func(string, string) { dropped++ }}
	planted, clean := 0, map[string]bool{}
	for i := 0; i < 3+r.Intn(5); i++ {
		var paras []string
		for j := 0; j < 1+r.Intn(4); j++ {
			paras = append(paras, fmt.Sprintf("Paragraph %d of file %d explains how the payments queue drains.", j, i))
		}
		id := fmt.Sprintf("payments/blob/docs/f%d.md", i)
		if r.Intn(2) == 0 {
			paras[r.Intn(len(paras))] += " " + canaries[r.Intn(len(canaries))]
			planted++
		} else {
			clean[id] = true
		}
		src.Put(id, "f.md", strings.Join(paras, "\n\n"), "ann@acme.example")
	}
	if _, err := p.Sync(ctx, ""); err != nil {
		return err
	}
	all, err := idx.All(ctx, "acme-logistics")
	if err != nil {
		return err
	}
	indexed := map[string]bool{}
	for _, ch := range all {
		indexed[ch.SourceID] = true
		for _, cn := range canaries {
			if strings.Contains(ch.Text, cn) {
				return fmt.Errorf("chunk %s of %s holds a secret", ch.ID, ch.SourceID)
			}
		}
	}
	for id := range clean {
		if !indexed[id] {
			return fmt.Errorf("%s, which holds no secret, wasn't indexed", id)
		}
	}
	if dropped < planted {
		return fmt.Errorf("%d secrets planted, %d chunks counted as dropped", planted, dropped)
	}
	return nil
}
