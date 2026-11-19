---
title: "SB-409: `draft-reply` agent where a human always sends"
summary: "The first agent saves time and changes nothing on its own."
---

# SB-409: `draft-reply` agent where a human always sends

| Field | Value |
|---|---|
| Epic | [E05 Actions, Approvals, Audit and Kill Switches](README.md) |
| Wave / slice | W4 / S4.3 Execution |
| Stories | US-05.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | [LAW-04](../../manual/sb-laws.7.md#law-04), [LAW-14](../../manual/sb-laws.7.md#law-14) |
| Depends on | [SB-306](../E04-retrieval-answers/SB-306-ask-agent-and-post-v1-ask.md), [SB-405](../E05-actions-approvals-audit/SB-405-tool-bus-policy-audit-then-execute.md) |
| Status | Done |

## Why

The first agent saves time and changes nothing on its own.

## Acceptance criteria

- [x] Drafts cite sources like answers do
- [x] Send is a human click in the target app or our UI
- [x] Approve and reject rates recorded

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Agent prompt and tools in release bundle
- [x] Draft UI
- [x] Metrics
- [x] Write or extend the law tests: LAW-04, LAW-14
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S4.3c): `ask.Drafts` is the draft-reply agent. It validates the context (`chat:<channel>/<ts>` or `tickets:<key>`) and the instruction, checks the kill switch, and asks policy for `use_agent` on `Agent::"draft-reply"` with `target_project`, so a tenant policy can forbid drafting on `FIN` tickets. It live-checks the context item itself (anything but Allow is `EPERM`, with nothing returned about the item) and makes it data block `C1`. Then it retrieves with the instruction, composes with the bundled `draft` prompt, and stores a trace. The agent has no tools: a test counts `Act` calls at zero. `Outcome` records `used` or `discarded` from the person who drafted, in `sb_drafts_total{outcome}`. `POST /v1/drafts` and `POST /v1/drafts/{trace}/outcome` are in `internal/api`. Base policies gained a tenant-scoped `draft-reply` permit that needs the agent's `enabled` attribute (an optional schema attribute, so older entities still evaluate to Deny). The web UI's draft panel shows the draft with citations and the Uncited label, and offers Copy (it records `used`; the person sends the reply in the target app) and Discard. A test checks it never calls `/v1/actions`, and the panel is axe-clean. LAW-14 holds through the same composer, and LAW-04 trivially (no tool calls).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
