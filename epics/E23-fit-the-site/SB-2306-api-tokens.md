---
title: "SB-2306: Read-only API tokens"
summary: "Scripts can't ask without a browser session."
---

# SB-2306: Read-only API tokens

| Field | Value |
|---|---|
| Epic | [E23 Fit the Site](README.md) |
| Wave / slice | W22 / S22.3 |
| Stories | US-23.4 |
| Manual | [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-identity(7)`](../../manual/sb-identity.7.md) |
| Laws | LAW-12, LAW-01, LAW-02 |
| Depends on | — |
| Status | Open |

## Why

Scripts can't ask without a browser session.

## Acceptance criteria

- [ ] A person makes a token on the web UI that can `POST /v1/ask` and `GET /v1/search` as them, nothing else, for up to 90 days; the cell keeps only its hash
- [ ] Every use is audited with the token's id; revoking it, signing out everywhere or deprovisioning ends it at once; groups are read at each use
- [ ] A token can't approve, act, administer or reach another tenant

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Token store
- [ ] Auth path
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
