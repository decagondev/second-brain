---
title: "SB-1502: The api service: every endpoint, mounted"
summary: "The handlers exist one by one; the cell needs them served together behind one sign-in."
---

# SB-1502: The `api` service: every endpoint, mounted

| Field | Value |
|---|---|
| Epic | [E15 Cell Runtime](README.md) |
| Wave / slice | W14 / S14.1 |
| Stories | US-15.1 |
| Manual | [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-runtime(7)`](../../manual/sb-runtime.7.md) |
| Laws | LAW-01, LAW-02, LAW-04, LAW-12 |
| Depends on | [SB-1501](SB-1501-cell-configuration-and-wiring.md) |
| Status | Open |

## Why

The handlers exist one by one; the cell needs them served together behind one sign-in.

## Acceptance criteria

- [ ] `cmd/sb-api` serves ask, drafts, actions, admin, stewards, access reviews, legal holds, spaces, SCIM, the chat app, the web UI, `/metrics` and the verifier, behind the auth middleware
- [ ] Health and readiness endpoints; graceful shutdown that finishes in-flight requests and never half-writes an audit row
- [ ] `tools/check/manual_code.py` sees every route the binary mounts, and a smoke test drives one request of each kind against a local cell

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Binary and routes
- [ ] Health and shutdown
- [ ] Smoke test
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
