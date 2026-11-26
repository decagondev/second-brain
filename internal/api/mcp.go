package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
)

// MCP serves POST /v1/mcp (sb-api(3), MCP; SB-3801): the Model Context Protocol over
// streamable HTTP, one JSON response a request, with three read-only tools that run the same
// code as their paths, as the caller. Mounted only with sb-cell(5) mcp: true.
type MCP struct {
	Agent interface {
		Asker
		Passager
		Searcher
	}
	Release string
	// Agents are the department agents the person may use now, each with its spaces by
	// name, and ScopeIn the person as a scope in one of them (SB-4306); nil lists none.
	Agents  func(ctx context.Context, s tenancy.Scope) (map[string][]string, error)
	ScopeIn func(ctx context.Context, user, space string) (tenancy.Scope, error)
	// Legwork plans and opens research reports for research_plan and research_report.
	Legwork interface {
		Plan(ctx context.Context, s tenancy.Scope, q string, docs []string) (ask.ResearchView, error)
		Open(ctx context.Context, s tenancy.Scope, id string) (ask.ResearchView, error)
	}
	// Skills are the skills of the person's spaces, listed as prompts (SB-4601); nil lists none.
	Skills func(ctx context.Context, s tenancy.Scope) ([]ask.SpaceSkill, error)
	// Starters are the starter questions of the person's spaces, listed as prompts with no
	// arguments (SB-4906); nil lists none.
	Starters func(ctx context.Context, s tenancy.Scope) ([]ask.SpaceStarter, error)
	// Families are the tool families the person's groups grant and whether they may use MCP
	// at all (sb-cell(5), mcp_groups; SB-5305); nil grants every family to everyone.
	Families func(s tenancy.Scope) (map[string]bool, bool)
}

// MCP tool families (SB-5305).
const (
	FamilySearch   = "search"
	FamilyAsk      = "ask"
	FamilyAgents   = "agents"
	FamilyResearch = "research"
	FamilySkills   = "skills"
)

// MCPFamilies are the families mcp_groups may grant.
var MCPFamilies = []string{FamilySearch, FamilyAsk, FamilyAgents, FamilyResearch, FamilySkills}

// familyOf is a tool's family.
func familyOf(tool string) string {
	switch {
	case tool == "search":
		return FamilySearch
	case tool == "ask" || tool == "passage":
		return FamilyAsk
	case strings.HasPrefix(tool, "research_"):
		return FamilyResearch
	case strings.HasPrefix(tool, "agent_"):
		return FamilyAgents
	}
	return ""
}

// granted says the person may use a family; with no Families, every one.
func (h MCP) granted(s tenancy.Scope, family string) bool {
	if h.Families == nil {
		return true
	}
	fs, ok := h.Families(s)
	return ok && fs[family]
}

// MCPVersion is the protocol revision the endpoint speaks.
const MCPVersion = "2025-06-18"

// Register mounts the endpoint.
func (h MCP) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/mcp", h.serve)
}

type rpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations map[string]any `json:"annotations"`
}

func strSchema(req []string, props map[string]string) map[string]any {
	p := map[string]any{}
	for k, d := range props {
		p[k] = map[string]any{"type": "string", "description": d}
	}
	return map[string]any{"type": "object", "properties": p, "required": req, "additionalProperties": false}
}

var readOnly = map[string]any{"readOnlyHint": true, "destructiveHint": false, "openWorldHint": false}

// MCPTools are the endpoint's tools.
var MCPTools = []mcpTool{
	{Name: "search", Description: "Search the company's documents you can open, without the model. Returns titles, cites and short quotes.",
		InputSchema: strSchema([]string{"query"}, map[string]string{"query": "What to look for"}), Annotations: readOnly},
	{Name: "ask", Description: "Ask the company's second brain. Every claim cites a document you can open now; check the cites before relying on it.",
		InputSchema: strSchema([]string{"question"}, map[string]string{"question": "The question, up to 4,000 characters"}), Annotations: readOnly},
	{Name: "passage", Description: "The cited text of one cite of an answer, checked again as you.",
		InputSchema: strSchema([]string{"trace", "cite"}, map[string]string{"trace": "The answer's trace id (tr_…)", "cite": "One of its cites"}), Annotations: readOnly},
}

func (h MCP) serve(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		http.Error(w, "sign in again", http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	var req rpcReq
	if err != nil || len(body) > MaxBody || json.Unmarshal(body, &req) != nil || req.JSONRPC != "2.0" || req.Method == "" {
		rpc(w, nil, nil, &rpcErr{Code: -32700, Message: "send one JSON-RPC 2.0 request"})
		return
	}
	if len(req.ID) == 0 || string(req.ID) == "null" { // a notification: nothing to answer
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if h.Families != nil { // SB-5305: only the groups mcp_groups names
		if _, ok := h.Families(s); !ok {
			rpc(w, req.ID, nil, &rpcErr{Code: -32001, Message: "EPERM: your groups aren't allowed to use MCP in this cell"})
			return
		}
	}
	r = r.WithContext(tenancy.WithClient(r.Context(), tenancy.ClientMCP)) // SB-5305: counted as MCP
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		v := MCPVersion
		if p.ProtocolVersion != "" && p.ProtocolVersion < MCPVersion {
			v = p.ProtocolVersion // an older client gets its own revision; the tools are the same
		}
		rpc(w, req.ID, map[string]any{"protocolVersion": v, "capabilities": map[string]any{"tools": map[string]any{"listChanged": false}, "prompts": map[string]any{"listChanged": false}},
			"serverInfo":   map[string]any{"name": "second-brain", "version": h.Release},
			"instructions": "Read-only. Every result is checked live as you; cite the sources you use."}, nil)
	case "ping":
		rpc(w, req.ID, map[string]any{}, nil)
	case "tools/list":
		tools, err := h.tools(r.Context(), s)
		if err != nil {
			rpc(w, req.ID, nil, &rpcErr{Code: -32603, Message: "the tools can't be listed now; try again"})
			return
		}
		kept := []mcpTool{}
		for _, t := range tools {
			if h.granted(s, familyOf(t.Name)) {
				kept = append(kept, t)
			}
		}
		rpc(w, req.ID, map[string]any{"tools": kept}, nil)
	case "tools/call":
		var p struct {
			Name      string            `json:"name"`
			Arguments map[string]string `json:"arguments"`
		}
		if json.Unmarshal(req.Params, &p) != nil {
			rpc(w, req.ID, nil, &rpcErr{Code: -32602, Message: "tools/call takes a name and string arguments"})
			return
		}
		if !h.granted(s, familyOf(p.Name)) {
			rpc(w, req.ID, nil, &rpcErr{Code: -32602, Message: "no such tool: see tools/list"})
			return
		}
		out, err := h.call(r.Context(), s, p.Name, p.Arguments)
		if errors.Is(err, errNoTool) {
			rpc(w, req.ID, nil, &rpcErr{Code: -32602, Message: "no such tool: see tools/list"})
			return
		}
		if err != nil {
			rpc(w, req.ID, map[string]any{"content": []map[string]string{{"type": "text", "text": err.Error()}}, "isError": true}, nil)
			return
		}
		b, _ := json.Marshal(out)
		rpc(w, req.ID, map[string]any{"content": []map[string]string{{"type": "text", "text": string(b)}}, "structuredContent": out}, nil)
	case "prompts/list", "prompts/get":
		if !h.granted(s, FamilySkills) {
			if req.Method == "prompts/list" {
				rpc(w, req.ID, map[string]any{"prompts": []any{}}, nil)
			} else {
				rpc(w, req.ID, nil, &rpcErr{Code: -32602, Message: "no such prompt"})
			}
			return
		}
		h.prompts(w, r, s, req)
	default:
		rpc(w, req.ID, nil, &rpcErr{Code: -32601, Message: "no such method"})
	}
}

// prompts answers prompts/list and prompts/get with the skills of the person's spaces
// (SB-4601) and their starter questions (SB-4906): a prompt runs nothing; the client calls
// a tool, as the person, so a starter's answer is composed fresh for whoever asks it.
func (h MCP) prompts(w http.ResponseWriter, r *http.Request, s tenancy.Scope, req rpcReq) {
	skills, starters, err := h.promptSources(r.Context(), s)
	if err != nil {
		rpc(w, req.ID, nil, &rpcErr{Code: -32603, Message: "the prompts can't be listed now; try again"})
		return
	}
	if req.Method == "prompts/list" {
		out := []map[string]any{}
		for _, sk := range skills {
			out = append(out, map[string]any{"name": sk.Space + "/" + sk.Name, "description": "The " + sk.Name + " skill of " + sk.Space + ": " + sk.Instruction,
				"arguments": []map[string]any{{"name": "question", "description": "The question to ask", "required": true}}})
		}
		for _, st := range starters {
			out = append(out, map[string]any{"name": starterPrompt(st), "description": "A starter question of " + st.Space + ": " + st.Question,
				"arguments": []map[string]any{}})
		}
		rpc(w, req.ID, map[string]any{"prompts": out}, nil)
		return
	}
	var p struct {
		Name      string            `json:"name"`
		Arguments map[string]string `json:"arguments"`
	}
	if json.Unmarshal(req.Params, &p) != nil {
		rpc(w, req.ID, nil, &rpcErr{Code: -32602, Message: "prompts/get takes a name and string arguments"})
		return
	}
	for _, sk := range skills {
		if sk.Space+"/"+sk.Name != p.Name {
			continue
		}
		q := strings.TrimSpace(p.Arguments["question"])
		text := "Use the ask tool, with ?space=" + sk.Space + ", to ask: /" + sk.Name + " " + q +
			"\n\nThen give the answer the way the skill says, keeping every cite: " + sk.Instruction
		rpc(w, req.ID, map[string]any{"description": "The " + sk.Name + " skill of " + sk.Space,
			"messages": []map[string]any{{"role": "user", "content": map[string]string{"type": "text", "text": text}}}}, nil)
		return
	}
	for _, st := range starters {
		if starterPrompt(st) != p.Name {
			continue
		}
		text := "Use the ask tool, with ?space=" + st.Space + ", to ask: " + st.Question + "\n\nThen give the answer, keeping every cite."
		rpc(w, req.ID, map[string]any{"description": "A starter question of " + st.Space,
			"messages": []map[string]any{{"role": "user", "content": map[string]string{"type": "text", "text": text}}}}, nil)
		return
	}
	rpc(w, req.ID, nil, &rpcErr{Code: -32602, Message: "no such prompt: see prompts/list"})
}

// starterPrompt is a starter question's prompt name: SPACE/start-N.
func starterPrompt(st ask.SpaceStarter) string { return fmt.Sprintf("%s/start-%d", st.Space, st.N) }

// promptSources are the skills and starter questions of the person's spaces.
func (h MCP) promptSources(ctx context.Context, s tenancy.Scope) ([]ask.SpaceSkill, []ask.SpaceStarter, error) {
	skills, starters := []ask.SpaceSkill{}, []ask.SpaceStarter{}
	var err error
	if h.Skills != nil {
		if skills, err = h.Skills(ctx, s); err != nil {
			return nil, nil, err
		}
	}
	if h.Starters != nil {
		if starters, err = h.Starters(ctx, s); err != nil {
			return nil, nil, err
		}
	}
	return skills, starters, nil
}

var errNoTool = errors.New("api: no such MCP tool")

// call runs one tool as s: the same calls its path makes.
func (h MCP) call(ctx context.Context, s tenancy.Scope, name string, args map[string]string) (any, error) {
	var out any
	var err error
	switch name {
	case "search":
		out, err = h.Agent.Search(ctx, s, ask.SearchRequest{Query: args["query"]})
	case "ask":
		out, err = h.Agent.Ask(ctx, s, ask.Request{Question: args["question"]})
	case "passage":
		out, err = h.Agent.Passage(ctx, s, strings.TrimSpace(args["trace"]), strings.TrimSpace(args["cite"]), -1)
	case "research_plan":
		if h.Legwork == nil {
			return nil, errNoTool
		}
		out, err = h.Legwork.Plan(ctx, s, args["question"], nil)
	case "research_report":
		if h.Legwork == nil {
			return nil, errNoTool
		}
		out, err = h.Legwork.Open(ctx, s, strings.TrimSpace(args["id"]))
	default:
		if out, err = h.callAgent(ctx, s, name, args); errors.Is(err, errNoTool) {
			return nil, err
		}
	}
	if err != nil {
		var over *ask.RateError
		if errors.As(err, &over) {
			return nil, errors.New("you've asked a lot in the last minute: wait a little and ask again [ERATE]")
		}
		return nil, mapErr(err)
	}
	return out, nil
}

func rpc(w http.ResponseWriter, id json.RawMessage, result any, e *rpcErr) {
	w.Header().Set("Content-Type", "application/json")
	msg := map[string]any{"jsonrpc": "2.0", "id": id}
	if id == nil {
		msg["id"] = nil
	}
	if e != nil {
		msg["error"] = e
	} else {
		msg["result"] = result
	}
	_ = json.NewEncoder(w).Encode(msg)
}

// AgentTool is a department agent's tool name (SB-4306): agent_ and its name, - as _.
func AgentTool(agent string) string { return "agent_" + strings.ReplaceAll(agent, "-", "_") }

// tools are the fixed tools, the research tools and the department agents the person may
// use now, made at each call so a killed agent is gone at once.
func (h MCP) tools(ctx context.Context, s tenancy.Scope) ([]mcpTool, error) {
	out := append([]mcpTool(nil), MCPTools...)
	if h.Legwork != nil {
		out = append(out,
			mcpTool{Name: "research_plan", Description: "Plan a research report on a question, as you, in this space. It is kept for you to read and Run in the web UI; it never runs from here.",
				InputSchema: strSchema([]string{"question"}, map[string]string{"question": "The question, up to 4,000 characters"}), Annotations: readOnly},
			mcpTool{Name: "research_report", Description: "Read one of your research reports, checked again as you.",
				InputSchema: strSchema([]string{"id"}, map[string]string{"id": "The report's id (rs_…)"}), Annotations: readOnly})
	}
	if h.Agents == nil || h.ScopeIn == nil {
		return out, nil
	}
	agents, err := h.Agents(ctx, s)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(agents))
	for n := range agents {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		sp := agents[n]
		out = append(out, mcpTool{Name: AgentTool(n), Description: "Ask the " + n + " agent in one of your spaces, as you. Every claim cites a document you can open now.",
			InputSchema: map[string]any{"type": "object", "required": []string{"space", "question"}, "additionalProperties": false,
				"properties": map[string]any{"space": map[string]any{"type": "string", "enum": sp, "description": "The space to ask in"},
					"question": map[string]any{"type": "string", "description": "The question, up to 4,000 characters"}}},
			Annotations: readOnly})
	}
	return out, nil
}

// callAgent runs a department agent's tool exactly as an ask with that agent, as the
// person in the space; an agent or space they can't use now is the tool's error.
func (h MCP) callAgent(ctx context.Context, s tenancy.Scope, tool string, args map[string]string) (any, error) {
	if h.Agents == nil || h.ScopeIn == nil || !strings.HasPrefix(tool, "agent_") {
		return nil, errNoTool
	}
	agents, err := h.Agents(ctx, s)
	if err != nil {
		return nil, ask.ErrUnavailable
	}
	for n, sp := range agents {
		if AgentTool(n) != tool {
			continue
		}
		space := strings.TrimSpace(args["space"])
		for _, x := range sp {
			if x != space {
				continue
			}
			in, err := h.ScopeIn(ctx, s.User(), ask.SpaceKey(space))
			if err != nil {
				return nil, ask.ErrInvalid
			}
			return h.Agent.Ask(ctx, in, ask.Request{Question: args["question"], Agent: n})
		}
		return nil, fmt.Errorf("%w: %s doesn't run in a space of yours called %q", ask.ErrInvalid, n, space)
	}
	return nil, fmt.Errorf("%w: no agent %s you can use now", ask.ErrInvalid, tool)
}
