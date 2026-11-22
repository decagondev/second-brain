---
title: "SB-1703: Save and share a question"
summary: "People re-find the same answer every week and forward answers to colleagues, and the web UI can neither save nor share."
---

# SB-1703: Save and share a question

| Field | Value |
|---|---|
| Epic | [E17 Team Knowledge](README.md) |
| Wave / slice | W16 / S16.1 |
| Stories | US-17.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-01, LAW-02, LAW-07 |
| Depends on | — |
| Status | Done |

## Why

People re-find the same answer every week and forward answers to colleagues, and the web UI can neither save nor share.

## Acceptance criteria

- [x] A person saves a question (their own text only) and lists their saved questions; opening one asks it again, live
- [x] `POST /v1/shares` makes a link `/q/SHARE` that shows the recipient the question and asks it under the recipient's access, never the sharer's answer
- [x] Saved questions and shares are cell records under the tenant's retention, erased with the person; creating a share writes an audit row with the question's hash
- [x] A share from another tenant, or an expired one, is `EPERM`

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Records and endpoints
- [x] Saved tab and share link in the UI
- [x] Tests: a recipient with less access sees less
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-22 (slice S16.1): `ask.Questions` with `POST/GET /v1/saved`, `DELETE /v1/saved/{id}`, `POST /v1/shares` and `GET /v1/shares/{id}`. Only the question is kept; a share id is `sh_` and 128 random bits, valid 30 days, and opening one gives the question only, which the recipient asks under their own access. Creating a share writes an `access` audit row first with the question's SHA-256, never its text. Both trace stores keep saved questions and shares as records under retention (with holds by space) and erase them with the person. The UI's Saved tab asks a saved question again, each answer has Save and Share, and `/q/sh_…` opens on the shared question. Tests: `TestLAW02_SharesStayInTheirTenant` (another tenant's or an expired share is EPERM; its self-test catches a store not keyed by tenant), core, adapter (retention and erasure), API, a running cell (bob opens ann's share and gets no answer from her access) and UI tests.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
