---
title: "SB-106: OIDC sign-in per tenant"
summary: "Users sign in with the IdP they already have."
---

# SB-106: OIDC sign-in per tenant

| Field | Value |
|---|---|
| Epic | [E01 Tenancy and Identity](README.md) |
| Wave / slice | W1 / S1.3 Identity |
| Stories | US-01.2 |
| Manual | [`sb-identity(7)`](../../manual/sb-identity.7.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02) |
| Depends on | [SB-105](../E01-tenancy-identity/SB-105-control-plane-tenant-registry-and-cell.md) |
| Status | Done |

## Why

Users sign in with the IdP they already have.

## Acceptance criteria

- [x] Login works with a test OIDC IdP
- [x] Tokens carry tenant id; a token from tenant A is rejected by cell B
- [x] Session lifetime and refresh follow `sb-identity(7)`

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] OIDC adapter behind `IdentityProvider` port
- [x] Session store per cell
- [x] Two-IdP test fixture
- [x] Write or extend the law tests: LAW-02
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: built, standard library only.
- `internal/adapters/oidc` does discovery, then verifies RS256 against the key set (refreshed at most once a minute), and checks issuer, audience, time with 60 seconds of leeway, nonce, tenant and subject. It runs the code flow with PKCE S256. `oidctest` is a TLS test IdP that also mints deliberately wrong tokens.
- `internal/core/session` holds the session rules: 12 hours from sign-in with no extension, re-validated against `ports.Directory` every 60 seconds, and unknown or no answer ends the session.
- `internal/auth` is the cell's `/login` and `/callback`, the session cookie, and `Authenticate`/`TenantOf`, which plugs into the SB-105 edge router.
- `internal/adapters/sessions/memory` is the per-cell store.
- Tests: a full browser sign-in against the test IdP, re-validation and 12-hour expiry, an unknown user refused, a two-IdP fixture (acme's session and acme's token both refused at beta), and one failing token per rule, including `alg: none` and a forged key.
- LAW-02 gains a sign-in law test, red first, then green.

2026-11-19: reviewed by the user; CI green. **Done.**

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
