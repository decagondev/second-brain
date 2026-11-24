---
title: "SB-4103: Policies to read"
summary: "A policy everyone must read has no way to ask people to read it or see who has."
---

# SB-4103: Policies to read

| Field | Value |
|---|---|
| Epic | [E41 Run the Company](README.md) |
| Wave / slice | W40 / S40.1 |
| Stories | US-41.1 |
| Manual | [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-01, LAW-07, LAW-11 |
| Depends on | SB-3901 |
| Status | Done |

## Why

A policy everyone must read has no way to ask people to read it or see who has.

## Acceptance criteria

- [x] `POST /v1/spaces/NAME/policies` marks a document as a policy for the branch; members who can open it are told once (`policy`)
- [x] I've read it is recorded per person and version and audited; a new version asks again
- [x] Stewards see who hasn't read it by name; members who can't open it are counted

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API and records
- [x] Notify
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. ask.PolicyStore (memory and Postgres kinds policy, policyread); POST/GET /v1/spaces/NAME/policies, …/policies/read, …/policies/status (not_read by email, read, cant_open); the policy notify kind once per version to members who can open it; audit policy_marked and policy_read with no text; web PoliciesToRead and StewardPolicies.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
