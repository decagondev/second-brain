---
title: "SB-601: `sb tenant verify` on real data"
summary: "Operators prove their own tenant is sound before go-live."
---

# SB-601: `sb tenant verify` on real data

| Field | Value |
|---|---|
| Epic | [E07 Roll-out, Admin and Operations](README.md) |
| Wave / slice | W6 / S6.1 Verify and console |
| Stories | US-07.1 |
| Manual | [`sb-tenant(1)`](../../manual/sb-tenant.1.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-03](../../manual/sb-laws.7.md#law-03), [LAW-14](../../manual/sb-laws.7.md#law-14) |
| Depends on | [SB-302](../E04-retrieval-answers/SB-302-live-authorisation-of-every-chunk-before.md), [SB-309](../E06-evals-release/SB-309-citation-faithfulness-eval-set-per-tenant.md) |
| Status | In review |

## Why

Operators prove their own tenant is sound before go-live.

## Acceptance criteria

- [x] Prints pass or fail per law; unknown counts as fail
- [x] Samples real documents and users; never prints content

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Verify runner
- [x] Sampling strategy
- [x] Report format
- [x] Exit codes
- [x] Write or extend the law tests: LAW-01, LAW-02, LAW-03, LAW-14
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S6.1a): `internal/core/verify` is the runner over a `verify.Target`, the same interface for a local or a deployed cell. It samples users and document titles with a printed seed. LAW-01 re-asks the source, live, about every cited document and checks the answer's decision list shows a live check. LAW-02 asks for other tenants' document ids by id and by text. LAW-03 times one connector out and asks again. LAW-14 counts cited, uncited and unlabelled claims. Every line is counts only, `unknown` is a failure, and the exit status is 0 only if every law passes. `tools/evalcell --verify` runs it on a local cell over a fixture world. Its test passes on the permission world (with a stale prefilter), prints no content, fails a cell that cites what the asker can't open, and fails as `unknown` with no users. Unit tests fail each law on its own fault. `sb tenant verify` names what it needs (a deployed cell's verifier endpoint) and exits `ENOTIMPL` until then.

**Left:** the verifier endpoint in a deployed cell (W7), and the onboarding gates (SB-604) that call it.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
