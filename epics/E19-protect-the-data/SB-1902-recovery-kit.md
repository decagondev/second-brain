---
title: "SB-1902: Recovery kit in shares"
summary: "The keys that make a restore possible live only on the box; the runbook says to back them up 'with the machine's own encrypted backup', which many sites don't have."
---

# SB-1902: Recovery kit in shares

| Field | Value |
|---|---|
| Epic | [E19 Protect the Data](README.md) |
| Wave / slice | W18 / S18.1 |
| Stories | US-19.1 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md), [`sb-factory(7)`](../../manual/sb-factory.7.md) |
| Laws | LAW-18 |
| Depends on | SB-1901 |
| Status | Open |

## Why

The keys that make a restore possible live only on the box; the runbook says to back them up "with the machine's own encrypted backup", which many sites don't have.

## Acceptance criteria

- [ ] `install.sh --recovery-kit` splits the recovery private key and an archive of the box's keys into 5 shares, any 3 of which rebuild it (Shamir over GF(256))
- [ ] Each share is a text page with its number and a checksum; a mistyped share fails its checksum, and any 2 shares reveal nothing
- [ ] `restore.sh --from-kit` takes 3 shares; who holds a share is recorded, never the share

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Shamir in Go with tests
- [ ] `sb recovery split` / `join`
- [ ] Scripts and runbook
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
