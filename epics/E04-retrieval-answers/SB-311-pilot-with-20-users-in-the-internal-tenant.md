---
title: "SB-311: Pilot with 20 users in the internal tenant"
summary: "Real use before wide use."
---

# SB-311: Pilot with 20 users in the internal tenant

| Field | Value |
|---|---|
| Epic | [E04 Retrieval and Cited Answers](README.md) |
| Wave / slice | W3 / S3.3 Pilot |
| Stories | US-04.5 |
| Manual | [`sb-onboarding(7)`](../../manual/sb-onboarding.7.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-03](../../manual/sb-laws.7.md#law-03) |
| Depends on | [SB-307](../E04-retrieval-answers/SB-307-web-chat-ui.md), [SB-308](../E04-retrieval-answers/SB-308-chat-app-entry-point.md), [SB-310](../E04-retrieval-answers/SB-310-feedback-capture.md) |
| Status | In review |

## Why

Real use before wide use.

## Acceptance criteria

- [ ] 20 users active for two weeks
- [ ] Findings logged in devlog and risks

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [ ] Recruit pilot group
- [ ] Run two-week pilot
- [ ] Retro and backlog updates
- [ ] Write or extend the law tests: LAW-01, LAW-02, LAW-03 (on the pilot's real data this is `sb tenant verify`, SB-601)
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S3.8): this ticket is human work (recruit 20 people, run two weeks, hold the retro), so it gets tooling and a runbook and stays In review. sb-onboarding(7) gained "Running the pilot review": who to pick, the weekly report, triage of wrong-citation reports into tenant eval cases, and the retro. `python3 -m evals.pilot report EXPORT.jsonl` turns the weekly feedback export (SB-310) into the Gate 1 numbers and exits 1 under a 70% thumbs-up rate or with no feedback; it has unit tests in `make test`. [docs/pilot-retro-template.md](../../docs/pilot-retro-template.md) is the retro format: ids, counts and cites only.

**Left (people):** recruit the pilot group, run the two weeks, hold the retro, log findings and risks. **Left (code):** `sb tenant verify` (SB-601) for the pilot's law checks.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
