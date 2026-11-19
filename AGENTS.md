# AGENTS.md: instructions for AI agents working on this repository

- Work ticket by ticket from `epics/`. Respect wave and slice order and dependencies.
- **Manual first.** Update the `manual/` pages listed in the ticket before writing code. If the manual and the code disagree, the manual wins; fix the code or propose a manual change in the same PR.
- Write law tests and regression tests before implementation. Never delete a regression or smoke test.
- `Unknown` is always refusal. Never add a code path, flag or default that treats `Unknown`, a timeout or a missing source as allow.
- `internal/core/**` must not import `internal/adapters/**` or any vendor SDK.
- Never put secrets, document content, prompts or answers in logs, tests' golden files or audit rows. Use ids and hashes.
- Use the error codes in `manual/sb-errors.7.md`. Don't invent new ones without adding them to the registry.
- Never hard-code a cloud account id, a factory-wide signing key, or a push-to-cell deploy. Read the custody record (ADR-0015).
- Keep diffs small. Three lines beat ten.
- After each ticket: add a devlog entry (`living/DEVLOG.md`) with What, Why this way, Learned, Laws; update the assurance ledger and changelog.
- Run `make check` before proposing a change. If you touched approvals, rollout or permissions, run `make proofs`.
- No real customer or company names anywhere. Use the placeholder tenants `acme-logistics`, `beta-freight`, `gamma-parts`.
