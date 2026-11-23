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
| Status | Done |

## Why

Audit rows can be exported but not streamed to the company's SIEM.

## Acceptance criteria

- [x] `audit.stream: syslog+tls://HOST:PORT` forwards each audit row as written, RFC 5424, as the export has it (no content)
- [x] A SIEM that's down never blocks a write: rows queue on disk up to a day, a Sev-2 alert fires after an hour, and the stream resumes from the last row taken
- [x] A fake receiver gets every row in order across a restart

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Syslog adapter
- [x] Queue and cursor
- [x] Alert
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23: done in S22.1. `internal/core/auditstream` sends each row after the cursor as RFC 5424 over TLS (`internal/adapters/audit/syslogtls`, octet-counted framing), the text the row as the export writes it; the audit log is the queue and `state/audit/stream.json` the cursor. Rows older than a day when the SIEM comes back go in one gap message; `audit_stream_behind` (Sev-2) fires after an hour. A fake receiver gets every row in order across a restart and an outage (`TestAuditStreamToASIEM`, `TestLAW11_TheSIEMGetsEveryRowInOrder` with a self-test). Manuals: sb-audit(7), sb-cell(5).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
