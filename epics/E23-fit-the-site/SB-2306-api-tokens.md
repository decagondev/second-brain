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
| Status | Done |

## Why

Scripts can't ask without a browser session.

## Acceptance criteria

- [x] A person makes a token on the web UI that can `POST /v1/ask` and `GET /v1/search` as them, nothing else, for up to 90 days; the cell keeps only its hash
- [x] Every use is audited with the token's id; revoking it, signing out everywhere or deprovisioning ends it at once; groups are read at each use
- [x] A token can't approve, act, administer or reach another tenant

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Token store
- [x] Auth path
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-23 (S22.3). A person makes a read-only API token in the web UI (**API tokens**) or with `POST /v1/tokens`: `sbt_` and 43 random characters, shown once, for 1 to 90 days (default 30). The cell keeps only its SHA-256 as a session of the person (`session.NewToken`), so deprovisioning or `POST /v1/sessions/revoke` ends it with their sessions (LAW-12). A token reaches only `POST /v1/ask` and `GET /v1/search`, as its owner; anything else is `EPERM`. Each use writes a `token_used` audit row before the request runs, and a cell that can't write it answers `EUNAVAIL`. `GET /v1/tokens` lists names and dates, `DELETE /v1/tokens/{id}` revokes. The secrets scan knows the `sbt_` shape. Tests: `internal/core/session/token_test.go`, `tests/cell/tokens_test.go`, `web/src/tokens.test.tsx`; LAW-12's generator signs in with tokens every other time.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
