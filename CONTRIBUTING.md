---
title: Contributing
summary: How to pick up a ticket and get it merged.
---

# Contributing

## The workflow

1. **Pick a ticket.** Take the lowest open wave and slice in [epics/](epics/README.md). Check its dependencies are done.
2. **Write the manual first.** Update the man pages listed on the ticket and get them reviewed (ADR-0009). CI rejects behaviour changes without a `manual/` diff.
3. **Write the laws.** Add or extend the ticket's law tests in `laws/`, plus regression tests. They should fail first, for the right reason.
4. **Implement** behind the ports in `internal/core`. Follow the [engineering standards](docs/06-engineering-standards.md). No vendor import in the core.
5. **Make it green:** `make check` (it must pass on a clean clone). If you touched approvals, rollout or permissions, run `make proofs` and `make crosscheck` too. If a law your ticket implements turns green, take it off `laws/expected_red.txt`; the ratchet will tell you.
6. **Update the living docs:** a [devlog](living/DEVLOG.md) entry (always), the [assurance ledger](living/ASSURANCE_LEDGER.md), the [changelog](living/CHANGELOG.md), and [bugs](living/BUGS_AND_MITIGATIONS.md) if you fixed one.
7. **Open a PR** titled `SB-123: summary` and tick the [Definition of Done](docs/05-living-documents.md#definition-of-done).

## Ground rules

- **Unknown is refusal.** No flag, option or environment variable changes that.
- **Three lines beat ten** (P3). If a reviewer can delete half your diff and keep the behaviour, they will ask you to.
- Two reviewers, one from security, for `policy`, `audit`, `toolbus`, `approvals`, `actions`, `release`, `proofs` and `evals/gate.yaml`.
- Every bug fix comes with a regression test and an answer to "which law should have caught this?".
- Never delete a regression or smoke test.
- Never hard-code a cloud account id or assume the factory holds a cell's keys. Read the custody record (ADR-0015, SB-114).
- No customer names in code, docs, fixtures or commit messages. Use `acme-logistics`, `beta-freight`, `gamma-parts`.

## Docs

```sh
pip install markdown
make docs          # python3 tools/site/build.py && python3 tools/check/links.py site
```
