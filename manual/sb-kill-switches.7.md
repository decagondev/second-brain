---
title: "sb-kill-switches(7)"
man: "7"
summary: Stopping agents, connectors and tenants.
---

# sb-kill-switches(7)

## NAME

sb-kill-switches — stop first, investigate second

## SYNOPSIS

```text
sb kill agent|connector|tenant [NAME] --tenant T --reason TEXT     effective ≤ 10 s
```

## DESCRIPTION

### Why kill switches

When an agent misbehaves or a source is compromised, an operator must be able to stop it **now**, without a deploy, a manifest change or a call to anyone.

### Scopes

| Scope | Stops |
|---|---|
| agent | New runs and tool calls of that agent |
| connector | All four connector calls; its chunks become `Unknown`, so they're dropped from answers |
| space | Asks, drafts and tool calls in that space, and its content everywhere: a company-wide ask drops every chunk the space holds (SB-1008) |
| tenant | Everything but admin, audit read and unkill |

### Guarantees

- Takes effect within 10 seconds in every affected cell (LAW-10).
- Checked before authorising **and** again right before executing an approved action.
- Every kill and unkill is audited with its reason.
- **Fail-safe:** a cell that can't reach the kill-switch store for 30 seconds stops consequential tool calls and keeps serving reads.

### How it works, exactly

1. **Set.** `sb kill` writes a `kind: kill` audit row (`subject` = `agent:NAME`, `connector:KIND`, `space:NAME` or `tenant`, `state` = `killed`, `reason` = the operator's text) **first**, then records the switch in the control plane's kill store. If the audit write fails, the switch isn't recorded and `sb kill` fails, so nothing changes without a trace. `sb unkill` does the same with `state` = `cleared`.
2. **Propagate.** Each cell keeps a kill checker. It reloads the whole switch list from the store every **2 seconds**, and the control plane also pushes each change as it's made. The 10-second bound therefore has 8 seconds of margin over the poll alone.
3. **Check.** The checker answers from the last list it loaded, at three points: before an agent run starts, before a tool call is authorised, and again right before an approved call executes. A killed connector also counts at the live check: its chunks are treated as `Unknown`, so they drop out of answers. A killed space refuses an ask, a draft or a tool call in it with `EKILLED`, and at retrieval every chunk it holds is refused (`space_killed` in the trace) before the live ACL batch, in any ask, so its content leaves company-wide answers too. A document another space also holds is dropped as well: a kill never leaves part of a space readable. A killed tenant refuses everything except admin, audit read and `sb unkill`, all with `EKILLED`.
4. **Fail safe.** If the checker hasn't loaded the list successfully for **30 seconds**, it treats every consequential tool call as killed (`EKILLED`, reason `kill_store_unreachable`), and reads go on as before. A checker that has never loaded the list at all, for example at cell start, treats every agent run, connector call and tool call as killed until its first load. Only admin and audit reads go ahead.

### Making it permanent

A kill is runtime state. To keep something off, remove it from the manifest. The console offers this as a one-click manifest diff.

### Drills

Every release fires every scope in staging and checks the 10-second bound (SB-404). The scopes are the tenant, every agent and connector kind in its manifest, and every space (`kill.DrillScopes`, SB-1008). For each scope, the drill sets the switch through the normal audited path (actor `sys_kill_drill`), then tries a new op in that scope every 250 ms until the cell refuses it. A refusal later than 10 seconds fails the drill. The drill then clears the switch. The release notes get a table with each scope, how long it took to take effect, and pass or fail. A failed drill blocks the release like a failed eval.

## LAWS

[LAW-10](sb-laws.7.md#law-10), [LAW-04](sb-laws.7.md#law-04).

## EXAMPLES

```sh
sb kill connector chat --tenant acme-logistics --reason "token possibly leaked, rotating"
sb kill list --tenant acme-logistics
```

## SEE ALSO

[sb-kill(1)](sb-kill.1.md), [11 Kill switches](../docs/architecture/11-kill-switches.md)
