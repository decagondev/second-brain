---
title: "SB-2302: Audit stream to the SIEM"
summary: "Audit rows can be exported but not streamed to the company's SIEM."
---

# SB-2302: Audit stream to the SIEM

| Field | Value |
|---|---|
| Epic | [E23 Fit the Site](README.md) |
| Wave / slice | W22 / S22.1 |
| Stories | US-23.1 |
| Manual | [`sb-audit(7)`](../../manual/sb-audit.7.md), [`sb-cell(5)`](../../manual/sb-cell.5.md) |
| Laws | LAW-11, LAW-07 |
| Depends on | — |
| Status | Open |

## Why

Audit rows can be exported but not streamed to the company's SIEM.

## Acceptance criteria

- [ ] `audit.stream: syslog+tls://HOST:PORT` forwards each audit row as written, RFC 5424, as the export has it (no content)
- [ ] A SIEM that's down never blocks a write: rows queue on disk up to a day, a Sev-2 alert fires after an hour, and the stream resumes from the last row taken
- [ ] A fake receiver gets every row in order across a restart

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Syslog adapter
- [ ] Queue and cursor
- [ ] Alert
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
