package api

import (
	"net/http"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/errs"
)

// Cases from wrong citations (sb-evals(7); SB-2204): a space's stewards add, list and
// delete the tenant's eval cases.

func (h Questions) addEval(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var req struct {
		Report string `json:"report"`
		Expect string `json:"expect"`
	}
	if !readJSON(r, &req) || req.Report == "" || req.Expect == "" {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with report, a reported answer's trace, and expect, KIND:ID."))
		return
	}
	c, err := h.Service.AddEvalCase(r.Context(), s, r.PathValue("name"), req.Report, req.Expect)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", c)
}

func (h Questions) evals(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	cs, err := h.Service.EvalCasesOf(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string][]ask.EvalCase{"items": cs})
}

func (h Questions) deleteEval(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	if err := h.Service.DeleteEvalCase(r.Context(), s, r.PathValue("name"), r.PathValue("id")); err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]bool{"deleted": true})
}
