---
title: "SB-4203: Action items from meetings"
summary: "What people agreed to do in a meeting is lost once the transcript is filed."
---

# SB-4203: Action items from meetings

| Field | Value |
|---|---|
| Epic | [E42 Agents That Do the Legwork](README.md) |
| Wave / slice | W41 / S41.1 |
| Stories | US-42.1 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-16 |
| Depends on | SB-4006 |
| Status | Open |

## Why

What people agreed to do in a meeting is lost once the transcript is filed.

## Acceptance criteria

- [ ] A transcript's decisions draft gains action items, each with an owner matched to a tenant user and a due date if one was said
- [ ] Nothing reaches an owner before the steward confirms; each owner gets one `action` message and can mark items done
- [ ] An item past its date shows in the owner's inbox; an unmatched owner stays as written and gets no message

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Extraction
- [ ] Records
- [ ] Notify
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
