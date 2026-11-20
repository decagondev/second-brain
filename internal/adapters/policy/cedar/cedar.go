// Package cedar evaluates policies with cedar-go (ADR-0008, ADR-0017), behind the
// policy.Evaluator interface. A policy that fails to evaluate makes the answer Unknown,
// never a silent deny or allow.
package cedar

import (
	"context"
	"fmt"
	"sort"
	"strings"

	cedargo "github.com/cedar-policy/cedar-go"

	"example.com/second-brain/internal/core/policy"
)

// Engine is a loaded policy set: platform policies plus the tenant's own.
type Engine struct{ set *cedargo.PolicySet }

var _ policy.Evaluator = Engine{}

// Load parses policy files (name -> Cedar text) into one set. Any parse error fails the load.
func Load(files map[string][]byte) (Engine, error) {
	set := cedargo.NewPolicySet()
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		list, err := cedargo.NewPolicyListFromBytes(n, files[n])
		if err != nil {
			return Engine{}, fmt.Errorf("cedar: %s: %w", n, err)
		}
		for i, p := range list {
			set.Add(cedargo.PolicyID(fmt.Sprintf("%s#%d", n, i)), p)
		}
	}
	return Engine{set: set}, nil
}

func uid(typ, id string) cedargo.EntityUID {
	return cedargo.NewEntityUID(cedargo.EntityType(typ), cedargo.String(id))
}

func value(v any) (cedargo.Value, error) {
	switch x := v.(type) {
	case string:
		return cedargo.String(x), nil
	case bool:
		return cedargo.Boolean(x), nil
	case int64:
		return cedargo.Long(x), nil
	case int:
		return cedargo.Long(int64(x)), nil
	case policy.Ref:
		return uid(x.Type, x.ID), nil
	case []string:
		vs := make([]cedargo.Value, len(x))
		for i, s := range x {
			vs[i] = cedargo.String(s)
		}
		return cedargo.NewSet(vs...), nil
	}
	return nil, fmt.Errorf("cedar: unsupported attribute type %T", v)
}

func record(m map[string]any) (cedargo.Record, error) {
	rm := cedargo.RecordMap{}
	for k, v := range m {
		cv, err := value(v)
		if err != nil {
			return cedargo.Record{}, err
		}
		rm[cedargo.String(k)] = cv
	}
	return cedargo.NewRecord(rm), nil
}

// Evaluate answers one request. Entities are built only from the request: the principal
// (with its groups and tenant), the resource, and the tenant itself.
func (e Engine) Evaluate(_ context.Context, r policy.Request) (policy.Result, error) {
	if e.set == nil {
		return policy.Result{}, fmt.Errorf("cedar: no policies loaded")
	}
	tenant := uid("Tenant", r.Principal.Tenant)
	ents := cedargo.EntityMap{tenant: {UID: tenant}}
	var parents []cedargo.EntityUID
	for _, g := range r.Principal.Groups {
		gu := uid("Group", g)
		parents = append(parents, gu)
		ents[gu] = cedargo.Entity{UID: gu, Attributes: cedargo.NewRecord(cedargo.RecordMap{"tenant": tenant})}
	}
	principal := uid("User", r.Principal.User)
	ents[principal] = cedargo.Entity{UID: principal, Parents: cedargo.NewEntityUIDSet(parents...),
		Attributes: cedargo.NewRecord(cedargo.RecordMap{"tenant": tenant})}

	attrs := map[string]any{}
	for k, v := range r.Resource.Attrs {
		attrs[k] = v
	}
	attrs["tenant"] = policy.Ref{Type: "Tenant", ID: r.Resource.Tenant}
	rec, err := record(attrs)
	if err != nil {
		return policy.Result{}, err
	}
	resource := uid(r.Resource.Type, r.Resource.ID)
	ents[resource] = cedargo.Entity{UID: resource, Attributes: rec}
	ctxRec, err := record(r.Context)
	if err != nil {
		return policy.Result{}, err
	}

	ok, diag := cedargo.Authorize(e.set, ents, cedargo.Request{Principal: principal, Action: uid("Action", r.Action), Resource: resource, Context: ctxRec})
	if len(diag.Errors) > 0 {
		msgs := make([]string, len(diag.Errors))
		for i, de := range diag.Errors {
			msgs[i] = de.String()
		}
		return policy.Result{Decision: policy.Unknown, Reason: strings.Join(msgs, "; ")}, nil
	}
	var ids []string
	for _, rs := range diag.Reasons {
		ids = append(ids, string(rs.PolicyID))
	}
	if ok == cedargo.Allow {
		return policy.Result{Decision: policy.Allow, Reason: "permitted by " + strings.Join(ids, ", ")}, nil
	}
	if len(ids) > 0 {
		return policy.Result{Decision: policy.Deny, Reason: "forbidden by " + strings.Join(ids, ", ")}, nil
	}
	return policy.Result{Decision: policy.Deny, Reason: "no policy permits this (default deny)"}, nil
}
