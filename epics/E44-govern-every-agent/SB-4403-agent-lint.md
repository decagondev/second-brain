---
title: "SB-4403: Agent lint in the gate"
summary: "A bad agent definition is caught only if a reviewer happens to notice."
---

# SB-4403: Agent lint in the gate

| Field | Value |
|---|---|
| Epic | [E44 Govern Every Agent](README.md) |
| Wave / slice | W43 / S43.1 |
| Stories | US-44.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | LAW-04, LAW-06, LAW-09 |
| Depends on | SB-4402 |
| Status | Open |

## Why

A bad agent definition is caught only if a reviewer happens to notice.

## Acceptance criteria

- [ ] `sb agents lint` runs in `make check` and the release gate
- [ ] It fails an undeclared tool, a write outside approvals, no department or template, a restricted space without `restricted: true`, no eval suite, or a prompt that acts on document text, naming the line
- [ ] The selftest breaks each rule once and goes red; every shipped agent passes

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Lint
- [ ] CI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
