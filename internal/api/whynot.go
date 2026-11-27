package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Ask why (sb-api(3); SB-5505 to SB-5507): Why not this document, the stewards' retrieval
// view, and confirming a memory line an MCP client proposed.

// WhyNoter explains why an answer didn't use a document (SB-5506).
type WhyNoter interface {
	WhyNot(ctx context.Context, s tenancy.Scope, id, name string) (ask.WhyNot, error)
}

func (h Ask) whyNot(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		h.fail(w, "", errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	var body struct {
		Document string `json:"document"`
	}
	if !readJSON(r, &body) || strings.TrimSpace(body.Document) == "" {
		h.fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with the document's title, link or KIND:ID."))
		return
	}
	v, err := h.Agent.(WhyNoter).WhyNot(r.Context(), s, r.PathValue("id"), body.Document)
	if err != nil {
		h.fail(w, "", mapErr(err))
		return
	}
	h.ok(w, r.PathValue("id"), v)
}

// retrieval is the replay of a reported or unhelpful answer, for the space's stewards.
func (h Questions) retrieval(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	v, err := h.Service.Retrieval(r.Context(), s, r.PathValue("name"), r.URL.Query().Get("trace"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", v)
}

// decideProposal confirms or dismisses a proposed memory line (SB-5505), in the web UI only:
// an API token can't (the MCP client that proposed it uses one).
func (h Me) decideProposal(confirm bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := tenancy.FromContext(r.Context())
		if err != nil {
			fail(w, errs.New(errs.EAUTH, "Sign in again."))
			return
		}
		if tenancy.ClientOf(r.Context()) == tenancy.ClientToken || tenancy.ClientOf(r.Context()) == tenancy.ClientMCP {
			fail(w, errs.New(errs.EPERM, "Confirm a proposed line in the web app."))
			return
		}
		v, err := h.Agent.DecideProposal(r.Context(), s, r.PathValue("id"), confirm)
		if err != nil {
			h.memoryErr(w, err)
			return
		}
		writeJSON(w, v)
	}
}

// MemoryURI is the one resource MCP offers: the caller's own memory lines (SB-5505).
const MemoryURI = "memory://me"

// resources answers resources/list and resources/read with the caller's own memory, read
// only, when memory is on and their groups have the memory family.
func (h MCP) resources(w http.ResponseWriter, r *http.Request, s tenancy.Scope, req rpcReq) {
	on := h.memoryOn() && h.granted(s, FamilyMemory)
	if req.Method == "resources/list" {
		out := []map[string]string{}
		if on {
			out = append(out, map[string]string{"uri": MemoryURI, "name": "Your memory", "mimeType": "application/json",
				"description": "The lines you keep about yourself, sent with your own asks. Read-only; propose a new one with memory_propose."})
		}
		rpc(w, req.ID, map[string]any{"resources": out}, nil)
		return
	}
	var p struct {
		URI string `json:"uri"`
	}
	if !on || json.Unmarshal(req.Params, &p) != nil || p.URI != MemoryURI {
		rpc(w, req.ID, nil, &rpcErr{Code: -32602, Message: "no such resource"})
		return
	}
	v, err := h.Memory.MemoryFor(r.Context(), s)
	if err != nil {
		rpc(w, req.ID, nil, &rpcErr{Code: -32603, Message: "your memory can't be read now; try again"})
		return
	}
	b, _ := json.Marshal(map[string]any{"lines": v.Lines})
	rpc(w, req.ID, map[string]any{"contents": []map[string]string{{"uri": MemoryURI, "mimeType": "application/json", "text": string(b)}}}, nil)
}

// MCPMemory is the person's memory as MCP reads it and proposes to it (SB-5505).
type MCPMemory interface {
	MemoryOn() bool
	MemoryFor(ctx context.Context, s tenancy.Scope) (ask.MemoryView, error)
	ProposeMemory(ctx context.Context, s tenancy.Scope, text string) (ask.MemoryLine, error)
}

func (h MCP) memoryOn() bool { return h.Memory != nil && h.Memory.MemoryOn() }
