---
title: "sb-permissions(7)"
man: "7"
summary: Allow, deny, unknown; the prefilter versus the live check.
---

# sb-permissions(7)

## NAME

sb-permissions — how the platform decides who may see and do what

## SYNOPSIS

```text
may (principal, action, resource, context)?  →  Allow | Deny | Unknown
Unknown  ≡  Deny
read(source content)  ⇒  Cedar permits  ∧  source.ACL(user, doc) = Allow, asked now
```

## DESCRIPTION

### Three answers

Every authorisation returns `Allow`, `Deny` or `Unknown`. `Unknown` covers every way of not knowing: timeouts, errors, missing documents, permission models the connector can't map. **Callers treat `Unknown` exactly as `Deny`** (LAW-03). There is no flag to change this.

### The policy kernel

Code asks one question: `policy.Authorise(scope, action, resource)`. It returns `Allow`, `Deny` or `Unknown`, always with a reason for the audit trail.

- The principal is built from the request's `tenancy.Scope` (user, tenant, directory groups), never from request input.
- A resource that belongs to another tenant is `Deny` before any policy runs.
- `Unknown` is returned when the scope is missing, the engine returns an error, a policy fails to evaluate, the engine takes longer than 50 ms, the engine panics, or the engine returns anything that isn't exactly `Allow` or `Deny`.
- `policy.Require` returns no error only for `Allow`. Callers use it or switch on all three values. CI refuses a `switch` on a `Decision` without an explicit `Unknown` case (the `exhaustive` linter), a `default:` case in such a switch, and any `!= policy.Deny` test, which would let `Unknown` through.

### Two layers

1. **Policy** ([sb-policy(5)](sb-policy.5.md)) decides agent, tool, workflow and approval rights, and always scopes to the principal's tenant.
2. **The source** decides who can read its documents, live, when the answer is built.

For source content, both must allow.

### Prefilter versus live check

When documents are ingested, their ACL is copied into the index as a **prefilter**. Retrieval uses it to skip documents the user obviously can't see, which keeps search fast. The prefilter can be stale, so it never grants access.

Before any chunk enters the model's context, its document is checked **live** with the connector's `ACL` call, batched, with an 800 ms deadline. Only `Allow` gets through.

```text
candidates ── prefilter (fast, maybe stale) ──▶ live ACL (authoritative) ──▶ context
```

Allows are never reused across requests. Denies may be cached for 60 seconds.

### What users see

- Refused documents are never named. The answer says how many were refused and, if all were, "I found sources but you don't have access to them."
- `EPERM` is used for both deny and unknown, so users can't probe for a document's existence.

### What operators see

The console's refusals view breaks refusals down by connector and reason. A connector whose unknown rate exceeds 1% is a defect to fix in the connector.

## LAWS

[LAW-01](sb-laws.7.md#law-01), [LAW-02](sb-laws.7.md#law-02), [LAW-03](sb-laws.7.md#law-03).

## EXAMPLES

Carol was removed from the finance group at 09:00. At 09:04 the prefilter still lists her. She asks about bonuses:

```text
retrieve:   docs:fin-bonus@3 (prefilter match, stale)
live ACL:   docs → Deny for usr_carol
context:    fin-bonus excluded
answer:     "I found sources but you don't have access to them." refused: 1
```

## SEE ALSO

[sb-policy(5)](sb-policy.5.md), [sb-connector(3)](sb-connector.3.md), [06 Retrieval and authorisation](../docs/architecture/06-retrieval-and-authorisation.md), [ADR-0002](../docs/adr/ADR-0002-source-acl-is-authority.md)
