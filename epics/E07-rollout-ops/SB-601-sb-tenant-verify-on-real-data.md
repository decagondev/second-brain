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
| Status | Open |

## Why

Operators prove their own tenant is sound before go-live.

## Acceptance criteria

- [ ] Prints pass or fail per law; unknown counts as fail
- [ ] Samples real documents and users; never prints content

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Verify runner
- [ ] Sampling strategy
- [ ] Report format
- [ ] Exit codes
- [ ] Write or extend the law tests: LAW-01, LAW-02, LAW-03, LAW-14
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
