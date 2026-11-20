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
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/adapters/policy/cedar"
	spacemem "example.com/second-brain/internal/adapters/spaces/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/spaces"
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
		// Path and Label are the document's location and source label (SB-1005).
		Path  string `json:"path"`
		Label string `json:"label"`
	} `json:"documents"`
	// Spaces and Barriers make the local cell a cell with spaces (sb-eval(5); SB-1006).
	Spaces map[string]struct {
		Members  []string            `json:"members"`
		Label    string              `json:"label"`
		Scope    map[string][]string `json:"scope"`
		Template string              `json:"template"` // SB-1102
		Agents   []string            `json:"agents"`
	} `json:"spaces"`
	// Agents are the agents the world's manifest enables (SB-1102); ask is always on.
	Agents   []string    `json:"agents"`
	Barriers [][2]string `json:"barriers"`
}

// spaceList is the world's spaces as the registry holds them.
func (w world) spaceList() []ports.Space {
	var out []ports.Space
	for name, sp := range w.Spaces {
		out = append(out, ports.Space{ID: spaces.ID(name), Tenant: w.Tenant, Name: name, Members: sp.Members, Label: sp.Label, Scope: sp.Scope,
			Template: sp.Template, Agents: sp.Agents})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

type input struct {
	World    world  `json:"world"`
	Model    string `json:"model"`
	Policies string `json:"policies"`
	Cases    []struct {
		Key    string `json:"key"`
		As     string `json:"as"`
		Ask    string `json:"ask"`
		Space  string `json:"space"` // "" or "company" is company-wide
		Agent  string `json:"agent"` // "" is ask
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
	if len(os.Args) > 1 && os.Args[1] == "--verify" {
		os.Exit(verifyMain())
	}
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

// cell is one local cell built from a world.
type cell struct {
	w     world
	src   *mock.Source
	conn  *mock.Connector
	agent ask.Agent
	users map[string]int
	texts map[string]string
}

func build(ctx context.Context, w world, modelName, policies string) (*cell, error) {
	if w.Tenant == "" {
		return nil, fmt.Errorf("world has no tenant")
	}
	var model ports.Model
	switch modelName {
	case "extractive":
		model = extractive{shift: false}
	case "misattributing":
		model = extractive{shift: true}
	default:
		return nil, fmt.Errorf("unknown model %q (extractive, misattributing)", modelName)
	}

	src := mock.NewSource("docs")
	texts := map[string]string{} // doc id -> the normalised text that spans index into
	for _, d := range w.Documents {
		src.PutAt(d.ID, d.Title, d.Text, d.Path, d.Label, d.Viewers...)
		if len(d.Viewers) == 0 {
			src.SetViewers(d.ID)
		}
		clean, _ := ingest.Sanitiser{}.Sanitise(d.Title + "\n\n" + d.Text) // as the pipeline does
		texts[d.ID] = ingest.Normalise(clean)
	}
	conn := mock.New(src, false)
	idx := memindex.New()
	p := ingest.Pipeline{Tenant: w.Tenant, Release: "rel_eval", Connector: conn, Embedder: hashing.Embedder{}, Index: idx}
	all := w.spaceList()
	if len(all) > 0 {
		p.Spaces = func(context.Context) ([]ports.Space, error) { return all, nil }
	}
	if _, err := p.Sync(ctx, ""); err != nil {
		return nil, fmt.Errorf("ingest: %w", err)
	}
	for _, d := range w.Documents {
		if d.ViewersNow != nil {
			src.SetViewers(d.ID, (*d.ViewersNow)...) // not synced: the index keeps the old prefilter
		}
	}
	engine, err := loadPolicies(policies)
	if err != nil {
		return nil, err
	}
	users := map[string]int{}
	for i, u := range w.Users {
		users[u.ID] = i
	}
	auth := retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": conn}, Policy: policy.Kernel{Engine: engine, Recorder: nopRecorder{}}}
	if len(all) > 0 {
		auth.Spaces = func(context.Context) ([]ports.Space, [][2]string, error) { return all, w.Barriers, nil }
	}
	n := 0
	agent := ask.Agent{Tenant: w.Tenant, Release: "rel_eval", Index: idx.Bind(w.Tenant), Embedder: hashing.Embedder{},
		Auth:   auth,
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
		Now:   time.Now,
		Enabled: func(name string) bool {
			for _, x := range w.Agents {
				if x == name {
					return true
				}
			}
			return false
		},
		SpaceOf: func(ctx context.Context, id string) (ports.Space, bool, error) {
			st := registry(w)
			if st == nil {
				return ports.Space{}, false, nil
			}
			return st.Space(ctx, w.Tenant, id)
		},
		AgentKilled: func(string) bool { return false }}

	return &cell{w: w, src: src, conn: conn, agent: agent, users: users, texts: texts}, nil
}

func run(ctx context.Context, in input) (map[string]result, error) {
	c, err := build(ctx, in.World, in.Model, in.Policies)
	if err != nil {
		return nil, err
	}
	w, src, agent, users, texts := in.World, c.src, c.agent, c.users, c.texts
	issuer := tenancy.NewIssuer(w.Tenant)
	out := map[string]result{}
	for _, c := range in.Cases {
		i, ok := users[c.As]
		if !ok {
			out[c.Key] = result{Error: "unknown user " + c.As}
			continue
		}
		// The space, resolved as the auth middleware does (sb-spaces(7)): a space the asker
		// isn't a member of is refused, never asked company-wide.
		space, err := spaces.Resolve(ctx, registry(w), w.Tenant, w.Users[i].Groups, c.Space)
		if err != nil {
			out[c.Key] = result{Error: "ESPACE: " + err.Error()}
			continue
		}
		s, err := issuer.IssueIn(w.Tenant, c.As, w.Users[i].Groups, space)
		if err != nil {
			return nil, err
		}
		if c.Inject.ACL != "" {
			src.QueueFaults(mock.Fault(c.Inject.ACL))
		}
		resp, err := agent.Ask(ctx, s, ask.Request{Question: c.Ask, Agent: c.Agent})
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

// registry is the world's space registry, or nil for a world without spaces.
func registry(w world) ports.SpaceStore {
	all := w.spaceList()
	if len(all) == 0 {
		return nil
	}
	st := spacemem.New()
	for _, sp := range all {
		_ = st.PutSpace(context.Background(), sp)
	}
	return st
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
