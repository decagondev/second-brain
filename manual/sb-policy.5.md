---
title: "sb-policy(5)"
man: "5"
summary: Cedar policy files, the schema, and the rules CI enforces on them.
---

# sb-policy(5)

## NAME

sb-policy — platform and tenant authorisation policies, written in Cedar

## SYNOPSIS

```text
policy/schema.cedarschema          platform schema (in the release bundle)
policy/platform/*.cedar            platform policies (in the release bundle)
tenants/<tenant>/policy/*.cedar    tenant policies (next to the manifest)
```

## DESCRIPTION

Policies decide what principals may do with agents, tools, workflows and approvals. They don't decide who may read a source document: that's the source's live ACL ([sb-permissions(7)](sb-permissions.7.md)). Both must say yes.

Cedar is default-deny, and `forbid` always beats `permit`.

### Entities

| Entity | Attributes |
|---|---|
| `User` | `tenant`; member of its `Group`s |
| `Group` | `tenant` |
| `Document` | `tenant`, `source` (connector kind) |
| `Chunk` | `tenant`, `cite`; in its `Document` |
| `Agent` | `tenant`, `name`, `consequential` |
| `Tool` | `tenant`, `name`, `connector`, `consequential` |
| `Workflow` | `tenant`, `name`, `approvers` (group) |
| `ProposedAction` | `tenant`, `workflow`, `requester` |
| `Cell` | `tenant` |

Every entity carries `tenant`, so every rule can be scoped. The kernel sets the resource's `tenant` from the request scope when the caller leaves it empty.

### Actions

Cedar's own `Action` type names the verbs: `Action::"read"` (on `Chunk` or `Document`), `"use_agent"`, `"call_tool"`, `"propose"`, `"approve"`, `"admin"`.

### Rules CI enforces

1. Every file validates against the schema.
2. Every `permit` has a `principal.tenant == resource.tenant` condition, or is in `policy/platform/` and carries one in its `when` (SB-112).
3. Tenant policies can only **narrow** platform policy: they may add `forbid`s, and `permit`s only for `use_agent`, `propose` and `approve` within the tenant.
4. A PR that changes policy shows the permission diff: which principals gained or lost which actions.

`make policy` (part of `make check`) runs rules 1 to 3 with `tools/check/policy.py`. On a pull request, CI also runs `tools/check/policy.py --diff BASE`, which evaluates a fixed matrix of members, groups, actions and resources against the old and new policies and prints each permission gained (`+`) or lost (`-`).

## EXAMPLES

Platform base (shipped):

```cedar
permit (principal, action == Action::"read", resource is Chunk)
when { principal.tenant == resource.tenant };

forbid (principal, action == Action::"approve", resource is ProposedAction)
when { resource.requester == principal };
```

Tenant policy: only the finance group may use `draft-reply` on finance tickets.

```cedar
forbid (principal, action == Action::"use_agent", resource == Agent::"draft-reply")
when { context has target_project && context.target_project == "FIN" && !(principal in Group::"grp_finance") };
```

Approvers for the tenant's workflow:

```cedar
permit (principal in Group::"grp_it_leads", action == Action::"approve", resource is ProposedAction)
when { resource.workflow == "ticket-update" && principal.tenant == resource.tenant };
```

## LAWS

[LAW-02](sb-laws.7.md#law-02), [LAW-03](sb-laws.7.md#law-03), [LAW-15](sb-laws.7.md#law-15).

## SEE ALSO

[sb-permissions(7)](sb-permissions.7.md), [02 Policy engine](../docs/architecture/02-policy-engine.md), [ADR-0008](../docs/adr/ADR-0008-policy-language-cedar.md)
