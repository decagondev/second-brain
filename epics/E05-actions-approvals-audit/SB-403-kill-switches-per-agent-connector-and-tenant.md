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
| Status | Open |

## Why

Stop first, investigate second.

## Acceptance criteria

- [ ] `sb kill agent|connector|tenant NAME` takes effect in 10 seconds or less
- [ ] Killed scope refuses new tool calls with `EKILLED`; in-flight calls are not started after the switch is seen
- [ ] Every kill and unkill is audited with a reason

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Flag store in control plane
- [ ] Push plus poll propagation
- [ ] Tool bus check
- [ ] Timed test
- [ ] Write or extend the law tests: LAW-10
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
