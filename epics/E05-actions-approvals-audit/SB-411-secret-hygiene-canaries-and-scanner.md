---
title: "SB-411: Secret hygiene: canaries and scanner"
summary: "Assume a leak will be attempted; prove it fails."
---

# SB-411: Secret hygiene: canaries and scanner

| Field | Value |
|---|---|
| Epic | [E05 Actions, Approvals, Audit and Kill Switches](README.md) |
| Wave / slice | W4 / S4.4 Hygiene |
| Stories | US-05.6 |
| Manual | [`sb-audit(5)`](../../manual/sb-audit.5.md) |
| Laws | [LAW-07](../../manual/sb-laws.7.md#law-07) |
| Depends on | [SB-205](../E03-connectors-ingestion/SB-205-connector-credentials-via-vault-references.md) |
| Status | Open |

## Why

Assume a leak will be attempted; prove it fails.

## Acceptance criteria

- [ ] Canary secrets planted in vault and sources never appear in prompts, logs or audit (scanner in CI and prod)
- [ ] Scanner finding is a Sev-1 alert

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Canary generator
- [ ] Log and prompt scanner
- [ ] Alert wiring
- [ ] LAW-07 green
- [ ] Write or extend the law tests: LAW-07
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
