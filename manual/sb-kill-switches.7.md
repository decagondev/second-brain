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
| tenant | Everything but admin, audit read and unkill |

### Guarantees

- Takes effect within 10 seconds in every affected cell (LAW-10).
- Checked before authorising **and** again right before executing an approved action.
- Every kill and unkill is audited with its reason.
- **Fail-safe:** a cell that can't reach the kill-switch store for 30 seconds stops consequential tool calls and keeps serving reads.

### Making it permanent

A kill is runtime state. To keep something off, remove it from the manifest. The console offers this as a one-click manifest diff.

### Drills

Every release fires every scope in staging and checks the 10-second bound.

## LAWS

[LAW-10](sb-laws.7.md#law-10), [LAW-04](sb-laws.7.md#law-04).

## EXAMPLES

```sh
sb kill connector chat --tenant acme-logistics --reason "token possibly leaked, rotating"
sb kill list --tenant acme-logistics
```

## SEE ALSO

[sb-kill(1)](sb-kill.1.md), [11 Kill switches](../docs/architecture/11-kill-switches.md)
