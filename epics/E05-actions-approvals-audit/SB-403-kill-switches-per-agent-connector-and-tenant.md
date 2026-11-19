---
title: "SB-403: Kill switches per agent, connector and tenant"
summary: "Stop first, investigate second."
---

# SB-403: Kill switches per agent, connector and tenant

| Field | Value |
|---|---|
| Epic | [E05 Actions, Approvals, Audit and Kill Switches](README.md) |
| Wave / slice | W4 / S4.1 Audit and kill |
| Stories | US-05.3 |
| Manual | [`sb-kill(1)`](../../manual/sb-kill.1.md), [`sb-kill-switches(7)`](../../manual/sb-kill-switches.7.md) |
| Laws | [LAW-10](../../manual/sb-laws.7.md#law-10) |
| Depends on | [SB-401](../E05-actions-approvals-audit/SB-401-append-only-hash-chained-audit-log-written.md) |
| Status | In review |

## Why

Stop first, investigate second.

## Acceptance criteria

- [x] `sb kill agent|connector|tenant NAME` takes effect in 10 seconds or less (2-second poll; LAW-10 timed test)
- [ ] Killed scope refuses new tool calls with `EKILLED`; in-flight calls are not started after the switch is seen (checker built; the tool bus's two check points come with SB-405 and SB-408)
- [x] Every kill and unkill is audited with a reason

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Flag store in control plane
- [x] Push plus poll propagation (`Checker.Push` and `Run`; the control plane's push channel comes with the cell runtime)
- [ ] Tool bus check (SB-405)
- [x] Timed test
- [x] Write or extend the law tests: LAW-10
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S4.1b, with SB-404): `internal/core/kill`. `Control.Kill` and `Unkill` write the `kind: kill` audit row first, then change the store, so an audit failure means no change. The `Checker` reloads every 2 seconds, accepts pushes, keeps its old list when a reload fails, and fails safe: after 30 seconds without a load, consequential calls are `EKILLED` (`kill_store_unreachable`), and before the first load only admin goes ahead. It is wired as the ask agent's `Killed` hook (`AgentKilled`) and the live check's (`retrieve.Authoriser.Killed`: a killed connector's documents stay Unknown). There are memory and local-control-plane stores. `sb kill|unkill agent|connector|tenant … --tenant T --reason R` and `sb kill list` work against `file:///DIR`, with smoke tests. `TestLAW10_KillWins` (fake clock, random kills, unkills and outages) went red first, then green, and a checker that never reloads is caught.

**Left:** the tool bus's check before authorise and before execute (SB-405, SB-408), the control plane's push channel, and a Postgres kill store (cell runtime).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
