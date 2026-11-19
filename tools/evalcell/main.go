// Command evalcell is the local eval cell (sb-eval(1), --model extractive): it builds one
// tenant's cell from a world (sb-eval(5), Worlds for the local cell) with the real
// pipeline, index, live check, policies, ask agent and composer, and answers eval cases.
//
// Input (stdin, JSON): {"world": {...}, "model": "extractive"|"misattributing",
// "policies": "DIR", "cases": [{"key", "as", "ask"}]}. Output (stdout, JSON):
// {"results": {"key": result}}, where a result has the shape evals/checks.py reads.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/adapters/policy/cedar"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

type world struct {
	Tenant string `json:"tenant"`
	Users  []struct {
		ID     string   `json:"id"`
		Email  string   `json:"email"`
		Groups []string `json:"groups"`
	} `json:"users"`
	Documents []struct {
		ID      string   `json:"id"`
		Title   string   `json:"title"`
		Text    string   `json:"text"`
		Viewers []string `json:"viewers"`
		// ViewersNow changes who may open the document after ingest, without a sync: the
		// prefilter is stale and only the live check knows (permission suite).
		ViewersNow *[]string `json:"viewers_now"`
	} `json:"documents"`
}

type input struct {
	World    world  `json:"world"`
	Model    string `json:"model"`
	Policies string `json:"policies"`
	Cases    []struct {
		Key    string `json:"key"`
		As     string `json:"as"`
		Ask    string `json:"ask"`
		Inject struct {
			ACL string `json:"acl"` // "timeout", "500" or "404" for the docs connector, this case only
		} `json:"inject"`
	} `json:"cases"`
}

type claim struct {
	Text      string   `json:"text"`
	Cites     []string `json:"cites"`
	CitedText string   `json:"cited_text"`
}

type result struct {
	Cites     []string `json:"cites"`
	Uncited   int      `json:"uncited"`
	Notice    *string  `json:"notice"`
	Text      string   `json:"text"`
	Claims    []claim  `json:"claims"`
	ToolCalls int      `json:"tool_calls"`
	Error     string   `json:"error,omitempty"`
}

func main() {
	var in input
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
		fail("evalcell: bad input: %v", err)
	}
	out, err := run(context.Background(), in)
	if err != nil {
		fail("evalcell: %v", err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"results": out}); err != nil {
		fail("evalcell: %v", err)
	}
}

func fail(f string, a ...any) {
	fmt.Fprintf(os.Stderr, f+"\n", a...)
	os.Exit(7)
}

func run(ctx context.Context, in input) (map[string]result, error) {
	w := in.World
	if w.Tenant == "" {
		return nil, fmt.Errorf("world has no tenant")
	}
	var model ports.Model
	switch in.Model {
	case "extractive":
		model = extractive{shift: false}
	case "misattributing":
		model = extractive{shift: true}
	default:
		return nil, fmt.Errorf("unknown model %q (extractive, misattributing)", in.Model)
	}

	src := mock.NewSource("docs")
	texts := map[string]string{} // doc id -> the normalised text that spans index into
	for _, d := range w.Documents {
		src.Put(d.ID, d.Title, d.Text, d.Viewers...)
		if len(d.Viewers) == 0 {
			src.SetViewers(d.ID)
		}
		clean, _ := ingest.Sanitiser{}.Sanitise(d.Title + "\n\n" + d.Text) // as the pipeline does
		texts[d.ID] = ingest.Normalise(clean)
	}
	conn := mock.New(src, false)
	idx := memindex.New()
	p := ingest.Pipeline{Tenant: w.Tenant, Release: "rel_eval", Connector: conn, Embedder: hashing.Embedder{}, Index: idx}
	if _, err := p.Sync(ctx, ""); err != nil {
		return nil, fmt.Errorf("ingest: %w", err)
	}
	for _, d := range w.Documents {
		if d.ViewersNow != nil {
			src.SetViewers(d.ID, (*d.ViewersNow)...) // not synced: the index keeps the old prefilter
		}
	}
	engine, err := loadPolicies(in.Policies)
	if err != nil {
		return nil, err
	}
	users := map[string]int{}
	for i, u := range w.Users {
		users[u.ID] = i
	}
	n := 0
	agent := ask.Agent{Tenant: w.Tenant, Release: "rel_eval", Index: idx.Bind(w.Tenant), Embedder: hashing.Embedder{},
		Auth:   retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": conn}, Policy: policy.Kernel{Engine: engine, Recorder: nopRecorder{}}},
		Model:  model,
		Store:  tracemem.New(),
		Killed: func(context.Context) bool { return false },
		Emails: func(_ context.Context, user string) ([]string, error) {
			if i, ok := users[user]; ok && w.Users[i].Email != "" {
				return []string{w.Users[i].Email}, nil
			}
			return nil, nil
		},
		NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) },
		Now:   time.Now}

	issuer := tenancy.NewIssuer(w.Tenant)
	out := map[string]result{}
	for _, c := range in.Cases {
		i, ok := users[c.As]
		if !ok {
			out[c.Key] = result{Error: "unknown user " + c.As}
			continue
		}
		s, err := issuer.Issue(w.Tenant, c.As, w.Users[i].Groups)
		if err != nil {
			return nil, err
		}
		if c.Inject.ACL != "" {
			src.QueueFaults(mock.Fault(c.Inject.ACL))
		}
		resp, err := agent.Ask(ctx, s, ask.Request{Question: c.Ask})
		src.InjectFault(mock.FaultNone)
		if err != nil {
			out[c.Key] = result{Error: err.Error()}
			continue
		}
		r := result{Notice: resp.Notice, Cites: []string{}, Claims: []claim{}}
		var all []string
		seen := map[string]bool{}
		for _, cl := range resp.Answer.Claims {
			all = append(all, cl.Text)
			if cl.Uncited {
				r.Uncited++
				continue
			}
			cc := claim{Text: cl.Text}
			var cited []string
			for _, ci := range cl.Cites {
				cc.Cites = append(cc.Cites, ci.Cite)
				if !seen[ci.Cite] {
					seen[ci.Cite] = true
					r.Cites = append(r.Cites, ci.Cite)
				}
				cited = append(cited, span(texts[docID(ci.Cite)], ci.Span))
			}
			cc.CitedText = strings.Join(cited, "\n")
			r.Claims = append(r.Claims, cc)
		}
		r.Text = strings.Join(all, " ")
		out[c.Key] = r
	}
	return out, nil
}

func docID(cite string) string {
	s := strings.TrimPrefix(cite, "docs:")
	if i := strings.LastIndex(s, "@"); i >= 0 {
		return s[:i]
	}
	return s
}

func span(text string, sp [2]int) string {
	r := []rune(text)
	if sp[0] < 0 || sp[1] > len(r) || sp[0] > sp[1] {
		return ""
	}
	return string(r[sp[0]:sp[1]])
}

func loadPolicies(dir string) (policy.Evaluator, error) {
	files := map[string][]byte{}
	ms, _ := filepath.Glob(filepath.Join(dir, "*.cedar"))
	for _, m := range ms {
		b, err := os.ReadFile(m)
		if err != nil {
			return nil, err
		}
		files[filepath.Base(m)] = b
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no policies in %s", dir)
	}
	return cedar.Load(files)
}

type nopRecorder struct{}

func (nopRecorder) RecordDecision(context.Context, policy.Record) error { return nil }

// extractive answers each data block's first sentence, citing it; with shift, it cites
// the next block instead (misattribution, for the self-test).
type extractive struct{ shift bool }

var block = regexp.MustCompile(`(?s)<<data cite="(C[0-9]+) [^"]*" flags="[^"]*">>\n(.*?)\n<</data>>`)

func (m extractive) Complete(_ context.Context, req ports.ModelRequest) (ports.ModelResponse, error) {
	last := req.Messages[len(req.Messages)-1].Content
	bs := block.FindAllStringSubmatch(last, -1)
	type cl struct {
		Text  string   `json:"text"`
		Cites []string `json:"cites"`
	}
	var cs []cl
	for i, b := range bs {
		label := b[1]
		if m.shift {
			label = bs[(i+1)%len(bs)][1]
			if len(bs) == 1 {
				label = "C99"
			}
		}
		cs = append(cs, cl{Text: firstSentence(b[2]), Cites: []string{label}})
	}
	if len(cs) == 0 {
		cs = append(cs, cl{Text: "I don't know.", Cites: nil})
	}
	raw, err := json.Marshal(map[string]any{"claims": cs})
	return ports.ModelResponse{Text: string(raw)}, err
}

func firstSentence(t string) string {
	t = strings.TrimSpace(t)
	for _, sep := range []string{". ", ".\n", "? ", "! "} {
		if i := strings.Index(t, sep); i >= 0 {
			return t[:i+1]
		}
	}
	return t
}
