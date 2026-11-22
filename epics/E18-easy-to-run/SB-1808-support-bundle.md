---
title: "SB-1808: A support bundle with no content"
summary: "When something breaks, IT sends screenshots and log snippets; nobody is sure what's safe to send."
---

# SB-1808: A support bundle with no content

| Field | Value |
|---|---|
| Epic | [E18 Easy to Run](README.md) |
| Wave / slice | W17 / S17.3 |
| Stories | US-18.3 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) |
| Laws | LAW-07 |
| Depends on | SB-1801 |
| Status | Open |

## Why

When something breaks, IT sends screenshots and log snippets; nobody is sure what's safe to send.

## Acceptance criteria

- [ ] `install.sh --support` writes one tar: versions and digests, `--doctor` output, `ps`, and the last 1,000 log lines per service
- [ ] Settings that name files are kept as names only; nothing under `/etc/second-brain` or `build/selfhost/vault` is read
- [ ] Before it's written, the tar's text is scanned with the secret scanner (SB-411); any hit aborts the bundle

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Script
- [ ] Scanner as `sb secrets scan`
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
