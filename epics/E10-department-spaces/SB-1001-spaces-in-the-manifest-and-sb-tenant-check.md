---
title: "SB-1001: Spaces in the manifest and sb tenant check"
summary: "Spaces are configuration; the manifest stays the truth."
---

# SB-1001: Spaces in the manifest and `sb tenant check`

| Field | Value |
|---|---|
| Epic | [E10 Department Spaces](README.md) |
| Wave / slice | W10 / S10.1 Space model |
| Stories | US-10.1 |
| Manual | `sb-space(5)` (new), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-801](../E00-foundations/SB-801-sb-spaces-7-and-sb-space-5-first-drafts.md) |
| Status | Done |

## Why

Spaces are configuration; the manifest stays the truth.

## Acceptance criteria

- [x] The parser accepts `spaces:` and `barriers:` exactly as `sb-space(5)` says, with line-precise errors
- [x] Rules: unknown template, member or steward groups not in `group:` form, a scope naming a connector the tenant lacks, and a restricted space with no steward are all errors
- [x] `sb tenant apply --plan` shows space changes

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Schema and parser
- [x] Validation rules
- [x] Plan output
- [x] Examples in `examples/tenants`
- [x] Write or extend the law tests: LAW-16
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S10.1a): sb-space(5) is no longer a draft: it gained rules 7 to 9 (group form and duplicates, a restricted space needs stewards, a space's agents must be enabled), the template defaults, and the apply refusal. sb-tenant(5), sb-tenant(1) (the `space` and `barrier` plan lines), sb-spaces(7) and sb-onboarding(7) changed with it. `manifest.Check` parses both blocks with line-precise errors for every rule, fills each space's label, agents and retention from its template (`manifest.Templates`, the sb-departments(7) catalogue), and `Format` writes them back; the published JSON Schema has both. `plan.Desired` lists each space and barrier, so `apply --plan` shows them. `apply.Run` refuses a manifest with either block, before any write, with `ENOTIMPL` naming SB-1006, so a company never runs believing a space protects what nothing enforces yet. `examples/tenants/spaces.yaml` passes check, round-trips, plans, and is refused by apply in the smoke test. The SB-801 law test that refused spaces became `TestLAW16_SpacesAreExactlyAsDeclared`: every declared space and barrier is in the checked tenant and the plan, exactly as written, and none other; the parser that drops spaces is still caught.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
