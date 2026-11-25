---
title: "SB-5107: Pause from the notice"
summary: "To pause an agent from its own notification, people have to find the agent first."
---

# SB-5107: Pause from the notice

| Field | Value |
|---|---|
| Epic | [E51 Guard What Goes In](README.md) |
| Wave / slice | W50 / S50.2 |
| Stories | US-51.2 |
| Manual | [`sb-kill-switches(7)`](../../manual/sb-kill-switches.7.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-10 |
| Depends on | SB-4702 |
| Status | Done |

## Why

To pause an agent from its own notification, people have to find the agent first.

## Acceptance criteria

- [x] Every notification about an agent run carries **Pause this agent here** for people allowed to pause it
- [x] The link asks for a reason and pauses only that agent in that space, the same audited pause as SB-4702
- [x] People not allowed to pause never see the link

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Link on agent notices
- [x] Pause with reason
- [x] Tests
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `notify.Message` carries the `Agent` and `Space` it is about; the notify job adds `Pause` (`/?pause=AGENT&space=NAME`) for each recipient `Job.MayPause` allows: the `pause-link` feature on for them and `Stewards.PauseVia` passing (with branch moderation only when that feature is on too). Updates and leader's weeks that couldn't be drafted and checks on an agent's change carry it; alerts about no one space don't. Slack, Mattermost and email render it as **Pause this agent here**; the web app opens on it with the reason prompt (`PauseHere`), which makes the same audited pause. Tested in TestPauseLinkOnlyForThoseWhoMay and web `pause.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
