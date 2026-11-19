---
title: "SB-014: Threat model v0 and ADR sign-off"
summary: "Security decisions made in W0 are cheap; made in W5 they are rewrites."
---

# SB-014: Threat model v0 and ADR sign-off

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W0 / S0.4 Laws red, models green |
| Stories | — |
| Manual | [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | [LAW-07](../../manual/sb-laws.7.md#law-07), [LAW-09](../../manual/sb-laws.7.md#law-09) |
| Depends on | [SB-005](../E00-foundations/SB-005-first-drafts-sb-connector-3-sb-permissions-7.md) |
| Status | In review |

## Why

Security decisions made in W0 are cheap; made in W5 they are rewrites.

## Acceptance criteria

- [ ] docs/07-threat-model.md covers STRIDE and LLM threats with a mitigation and a law for each
- [ ] ADR-0001 to ADR-0014 accepted or explicitly left Proposed with an owner

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [ ] Run a threat-modelling session; update the model
- [ ] Collect two reviewer sign-offs per ADR
- [x] Log new risks in living/RISKS.md
- [ ] Write or extend the law tests: LAW-07, LAW-09
- [x] Devlog entry, ledger and changelog updated

## Progress

The threat model covers STRIDE, LLM threats and factory hand-off. ADR-0001 to ADR-0016 are written, and new risks are in `living/RISKS.md`. Sign-off tables are in `docs/adr/README.md` and `docs/07-threat-model.md`.

**Left:** a threat-modelling session and two reviewer sign-offs per ADR.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
