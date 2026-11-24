package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
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
		rpc(w, req.ID, map[string]any{"protocolVersion": v, "capabilities": map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":   map[string]any{"name": "second-brain", "version": h.Release},
			"instructions": "Read-only. Every result is checked live as you; cite the sources you use."}, nil)
	case "ping":
		rpc(w, req.ID, map[string]any{}, nil)
	case "tools/list":
		rpc(w, req.ID, map[string]any{"tools": MCPTools}, nil)
	case "tools/call":
		var p struct {
			Name      string            `json:"name"`
			Arguments map[string]string `json:"arguments"`
		}
		if json.Unmarshal(req.Params, &p) != nil {
			rpc(w, req.ID, nil, &rpcErr{Code: -32602, Message: "tools/call takes a name and string arguments"})
			return
		}
		out, err := h.call(r.Context(), s, p.Name, p.Arguments)
		if errors.Is(err, errNoTool) {
			rpc(w, req.ID, nil, &rpcErr{Code: -32602, Message: "no such tool: search, ask or passage"})
			return
		}
		if err != nil {
			rpc(w, req.ID, map[string]any{"content": []map[string]string{{"type": "text", "text": err.Error()}}, "isError": true}, nil)
			return
		}
		b, _ := json.Marshal(out)
		rpc(w, req.ID, map[string]any{"content": []map[string]string{{"type": "text", "text": string(b)}}, "structuredContent": out}, nil)
	default:
		rpc(w, req.ID, nil, &rpcErr{Code: -32601, Message: "no such method"})
	}
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
	default:
		return nil, errNoTool
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
