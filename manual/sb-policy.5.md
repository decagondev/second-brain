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
| `User` | `tenant`; member of its `Group`s; and, when the directory has them, `department`, `division`, `cost_center`, `title` and `manager` (a user id), read at each request (SB-1205). Each is optional: a rule that uses one tests `has` first, and a missing one never matches |
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

### Spaces, labels and barriers (SB-802, built in SB-1004)

Schema v2 (`policy/schema.cedarschema`) carries spaces as facts the kernel sets, not as new entities, so a rule can never look a space up for itself:

| Where | Attribute | What it is |
|---|---|---|
| `Chunk` | `label` (String, optional) | the chunk's label: `public`, `internal`, `confidential` or `restricted` (sb-spaces(7)) |
| `Chunk` | `spaces` (Set of String, optional) | the ids (`spc_NAME`) of the spaces that hold its document |
| `read` context | `space` (String, optional) | the space asked in, or `company` |
| `read` context | `member_spaces` (Set of String, optional) | the spaces the asker is a member of, from the directory now |
| `read` context | `barred_spaces` (Set of String, optional) | the spaces held across a barrier from the asker, from the manifest's barriers |
| `Chunk` | `owners` (Set of String, optional) | the groups (`group:NAME`) of the person who owns its document at the source, from the directory now; empty for a document with no owner (SB-1203) |
| `read`, `call_tool`, `propose`, `approve` context | `barred_groups` (Set of String, optional) | the groups across a barrier from the asker (`group:NAME`) |
| `call_tool`, `propose` context | `barred_spaces`, `target_spaces`, `target_owners` (Sets of String, optional) | the spaces across a barrier from the requester, and the spaces holding and groups owning the action's target item |
| `approve` context | `requester_groups` (Set of String, optional) | the requester's groups (`group:NAME`) |

In a cell without spaces the kernel sets none of them, and the space rules don't apply. In a cell with spaces it sets all of them; `spaces.ReadContext` and `spaces.ChunkAttrs` build them. The rules are `policy/platform/spaces.cedar`:

```cedar
// A cell with spaces gives every fact, or the read is refused.
forbid (principal, action == Action::"read", resource is Chunk)
when {
  context has space &&
  !(context has member_spaces && context has barred_spaces && context has barred_groups &&
    resource has label && resource has spaces && resource has owners)
};

// An ask in a space reads only what that space holds, and only for its members.
forbid (principal, action == Action::"read", resource is Chunk)
when {
  context has space && context has member_spaces && resource has spaces &&
  context.space != "company" &&
  !(resource.spaces.contains(context.space) && context.member_spaces.contains(context.space))
};

// Only public and internal reach people outside the spaces that hold a chunk.
forbid (principal, action == Action::"read", resource is Chunk)
when {
  context has space && context has member_spaces && resource has label && resource has spaces &&
  !(resource.label == "public" || resource.label == "internal") &&
  !resource.spaces.containsAny(context.member_spaces)
};

// Nothing held by a space across a barrier from the asker.
forbid (principal, action == Action::"read", resource is Chunk)
when {
  context has barred_spaces && resource has spaces &&
  resource.spaces.containsAny(context.barred_spaces)
};

// Nothing owned at the source by someone across a barrier (SB-1203).
forbid (principal, action == Action::"read", resource is Chunk)
when { context has barred_groups && resource has owners && resource.owners.containsAny(context.barred_groups) };

// No tool call or proposal on an item across a barrier; a target that can't be placed refuses.
forbid (principal, action in [Action::"call_tool", Action::"propose"], resource)
when {
  context has barred_groups &&
  !(context has barred_spaces && context has target_spaces && context has target_owners)
};

forbid (principal, action in [Action::"call_tool", Action::"propose"], resource)
when {
  context has barred_spaces && context has target_spaces &&
  context.target_spaces.containsAny(context.barred_spaces)
};

forbid (principal, action in [Action::"call_tool", Action::"propose"], resource)
when {
  context has barred_groups && context has target_owners &&
  context.target_owners.containsAny(context.barred_groups)
};

// Nobody approves for a requester across a barrier.
forbid (principal, action == Action::"approve", resource is ProposedAction)
when { context has barred_groups && !(context has requester_groups) };

forbid (principal, action == Action::"approve", resource is ProposedAction)
when {
  context has barred_groups && context has requester_groups &&
  context.requester_groups.containsAny(context.barred_groups)
};
```

The rules every space policy is held to:

1. **A space policy can only forbid.** No file but `policy/platform/base.cedar` may `permit` `read`, and a `permit` with an unconstrained action counts as one. Space and label rules are `forbid`s, so a space can never make a chunk usable that the base policy and the source don't allow (LAW-19). `tools/check/policy.py` enforces this for every policy, base and tenant.
2. **Barriers come from the manifest**: the kernel turns the manifest's barriers into `barred_spaces` and `barred_groups`, never a hand-written rule (`internal/core/barrier`).
3. **A missing fact refuses.** In a cell with spaces, a chunk with no `label`, `spaces` or `owners`, or a context with no `member_spaces`, `barred_spaces` or `barred_groups`, is refused by the first rule (LAW-03). A label that isn't one of the four is treated as `restricted`.

### Rules CI enforces

1. Every file validates against the schema.
2. Every `permit` has a `principal.tenant == resource.tenant` condition, or is in `policy/platform/` and carries one in its `when` (SB-112).
3. Tenant policies can only **narrow** platform policy: they may add `forbid`s, and `permit`s only for `use_agent`, `propose` and `approve` within the tenant.
4. A PR that changes policy shows the permission diff: which principals gained or lost which actions.
5. **Narrowing (SB-1004).** Only `base.cedar` may permit `read`, and over a matrix of space contexts, memberships, barriers, labels (including an unknown one and none) and holders, the whole policy set allows no read that `base.cedar`'s read permits alone don't (LAW-19).

`make policy` (part of `make check`) runs rules 1 to 3 and 5 with `tools/check/policy.py`. On a pull request, CI also runs `tools/check/policy.py --diff BASE`, which evaluates a fixed matrix of members, groups, actions and resources against the old and new policies and prints each permission gained (`+`) or lost (`-`).

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
