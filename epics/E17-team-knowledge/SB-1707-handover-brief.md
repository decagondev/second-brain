---
title: "SB-1707: Handover brief for a successor"
summary: "When someone leaves or changes role, the documents they own and the context around them are the knowledge most often lost."
---

# SB-1707: Handover brief for a successor

| Field | Value |
|---|---|
| Epic | [E17 Team Knowledge](README.md) |
| Wave / slice | W16 / S16.3 |
| Stories | US-17.3 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-policy(5)`](../../manual/sb-policy.5.md) |
| Laws | LAW-01, LAW-09, LAW-12, LAW-14 |
| Depends on | [SB-1702](SB-1702-who-to-ask.md) |
| Status | Done |

## Why

When someone leaves or changes role, the documents they own and the context around them are the knowledge most often lost.

## Acceptance criteria

- [x] `handover`, a read-only agent in the release bundle, asked with a person's directory id, briefs the documents that person owns that the asker can open, grouped by location, every line cited, uncited lines dropped
- [x] It adds the count of that person's documents the asker can't open, never their names
- [x] Policy decides who may ask: the person's manager, a steward of one of their spaces, or the person; anyone else is `EPERM`
- [x] An eval suite, `evals/suites/agent-handover.yaml`, in the gate

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Prompt and agent definition
- [x] Policy rule
- [x] Eval suite and tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-22 (slice S16.3): `handover`, a read-only agent with the bundled prompt `handover.md`, served at `POST /v1/handover {person}`. It finds the person's documents by owner (their directory emails), checks each as the asker with `retrieve.Allowed`, briefs the newest twelve grouped by location through the model gateway, drops uncited lines and counts them, and counts the person's documents the asker can't open in `hidden`. The Cedar action `brief_handover` on the person permits only the relations the cell works out from the directory: `self`, `manager` (SCIM manager) or `steward` (a steward of a space whose members include one of the person's groups). It writes a trace and is killed with `sb kill agent handover`. The eval suite `evals/suites/agent-handover.yaml` (manager, self, a refused colleague) is in the gate and in `make evals`. Tests: `TestLAW12_HandoverBriefsOnlyWhatTheAskerCanOpen` (its self-test catches Unknown as Allow) and a running cell (`TestHandoverThroughARunningCell`). `make proofs` green.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
