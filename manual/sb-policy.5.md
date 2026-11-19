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
| `CellCustody` | `tenant` (always the factory's own tenant), `cell` (the tenant whose cell it is) |

Every entity carries `tenant`, so every rule can be scoped. The kernel sets the resource's `tenant` from the request scope when the caller leaves it empty.

### Actions

Cedar's own `Action` type names the verbs: `Action::"read"` (on `Chunk` or `Document`), `"use_agent"`, `"call_tool"`, `"propose"`, `"approve"`, `"admin"`, and the custody verbs `"view_custody"`, `"plan_handoff"`, `"approve_handoff"`, `"execute_handoff"`, `"abort_handoff"` and `"detach"` (Factory roles, below).

### Factory roles, exactly (SB-702)

The factory and the tenant owner are roles, never superusers. Both are Cedar policies in `policy/platform/base.cedar`, and both carry the tenant condition like every other permit.

| Role | Who | May | On |
|---|---|---|---|
| `factory-operator` | members of the group `sb-factory-operators` in the **factory's own tenant** (ring 0) | `view_custody`, `plan_handoff`, `approve_handoff`, `execute_handoff`, `abort_handoff` | `CellCustody` records, whose `tenant` is the factory's tenant |
| `tenant-owner` | members of the group `sb-owners` in the tenant's own IdP | `view_custody`, `approve_handoff`, `detach` | the tenant's own `Cell` |

- **Nobody approves their own plan.** `approve_handoff` is forbidden when `context.planner` is the principal, so the factory approval needs a second operator.
- **No role reads content.** Neither role adds a `read` permit, and a base `forbid` refuses `read` to `sb-factory-operators` and `sb-owners` on any tenant other than their own. A factory operator reads ring 0's own documents only as an ordinary user of ring 0.
- **Custody records always name their tenant.** The kernel normally fills an empty resource `tenant` from the caller's scope. For `CellCustody` it doesn't: an empty `tenant` is `Unknown`, which refuses. Otherwise a tenant with its own group called `sb-factory-operators` could act on a custody record it built itself.
- **Factory actions are control-plane calls.** They go through the policy kernel, so every decision is recorded before it is returned (LAW-04), and each step of a hand-off is also a `kind: custody` audit row (sb-factory(1)).

### Spaces, labels and barriers (SB-802)

The schema for spaces (schema v2, built in SB-1004) adds three entities. This is the draft that ticket builds:

| Entity | Attributes |
|---|---|
| `Space` | `tenant`, `name`, `label_default`, `members` (set of `Group`), `stewards` (set of `Group`) |
| `Label` | one of `public`, `internal`, `confidential`, `restricted`, as an ordered value |
| `Barrier` | `tenant`, `a` (`Group`), `b` (`Group`) |

`Chunk` gains `label` and `spaces` (the set of `Space` that hold its document), and the `read` request gains `context.space` (the space asked in, or `company`).

```text
// Draft (SB-1004), in Cedar: a space can only forbid. A text block, because the regression
// test validates every cedar block against today's schema, and v2 isn't it yet. These sit beside the base read permit,
// which stays the only permit on read; the source's live ACL is still checked after.
forbid (principal, action == Action::"read", resource is Chunk)
when { context.space != "company" && !(resource.spaces.contains(context.space_entity)) };

forbid (principal, action == Action::"read", resource is Chunk)
when { (resource.label == "confidential" || resource.label == "restricted") && !(principal.member_of_holder) };
```

The rules every space policy is held to:

1. **A space policy can only forbid.** No policy file may `permit` `read` except the base permit. Space and label rules are `forbid`s, so a space can never make a chunk usable that the base policy and the source don't allow (LAW-19). `tools/check/policy.py` will enforce this for every space policy, base and tenant (SB-1004).
2. **Barriers are forbids generated from the manifest**, one per barrier and direction, never written by hand.
3. **A missing attribute is `Unknown`, which refuses.** A chunk with no `label` is evaluated as `restricted`, and a chunk with no `spaces` is held by no space.

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
