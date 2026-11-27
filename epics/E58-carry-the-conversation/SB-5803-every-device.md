---
title: "SB-5803: Your conversations on every device"
summary: "Recent conversations live only in the browser they were asked in."
---

# SB-5803: Your conversations on every device

| Field | Value |
|---|---|
| Epic | [E58 Carry the Conversation](README.md) |
| Wave / slice | W57 / S57.1 |
| Stories | US-58.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | LAW-25, LAW-26 |
| Depends on | SB-2405, SB-3501, SB-3502 |
| Status | Open |

## Why

Recent conversations live only in the browser they were asked in.

## Acceptance criteria

- [ ] **Keep my conversations on every device** on My data stores the person's conversation list server-side: questions, kept and dropped documents and trace ids, never answers
- [ ] Entries go with the traces (30 days), are in the export and go with forget; turning it off deletes the copy; off by default
- [ ] Nobody else can list it, admins included

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Conversation list in the existing store
- [ ] My data switch
- [ ] Export and forget
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
