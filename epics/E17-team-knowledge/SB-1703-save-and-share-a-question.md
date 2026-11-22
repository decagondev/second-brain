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
| Status | Open |

## Why

People re-find the same answer every week and forward answers to colleagues, and the web UI can neither save nor share.

## Acceptance criteria

- [ ] A person saves a question (their own text only) and lists their saved questions; opening one asks it again, live
- [ ] `POST /v1/shares` makes a link `/q/SHARE` that shows the recipient the question and asks it under the recipient's access, never the sharer's answer
- [ ] Saved questions and shares are cell records under the tenant's retention, erased with the person; creating a share writes an audit row with the question's hash
- [ ] A share from another tenant, or an expired one, is `EPERM`

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Records and endpoints
- [ ] Saved tab and share link in the UI
- [ ] Tests: a recipient with less access sees less
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Planned 2026-11-22 with W16, accepted by the user the same day ([09-team-knowledge.md](../../docs/09-team-knowledge.md)). Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
