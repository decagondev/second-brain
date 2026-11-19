---
title: Living Documents
summary: The documents that change with every step of design and build, and the Definition of Done.
---

# Living documents

A living document changes with the work, in the same PR as the work. If a PR changes behaviour, a decision or our understanding, one of these changes too.

| Document | Updated when | Format rule |
|---|---|---|
| [DEVLOG](../living/DEVLOG.md) | **Every** PR and every significant design step | Newest first; fixed headings; never rewritten |
| [BUGS_AND_MITIGATIONS](../living/BUGS_AND_MITIGATIONS.md) | Every defect found, at any stage | Symptom, root cause, fix, the law or test that now catches it |
| [ASSURANCE_LEDGER](../living/ASSURANCE_LEDGER.md) | A law changes status | One row per law; status only moves up |
| [CHANGELOG](../living/CHANGELOG.md) | A user- or operator-visible change | Keep a Changelog style, grouped by release |
| [RISKS](../living/RISKS.md) | A risk is found, changes or closes | Likelihood, impact, owner, mitigation |
| [OPEN_QUESTIONS](../living/OPEN_QUESTIONS.md) | A question is raised or answered | Who decides, by when, default if unanswered |
| [GLOSSARY](../living/GLOSSARY.md) | A new term enters an interface | One line per term |
| [ADRs](adr/README.md) | A decision is made | Immutable once accepted |
| [Threat model](07-threat-model.md) | Architecture or trust boundaries change | STRIDE + LLM threats, mapped to laws |

## Devlog entry format

```markdown
## 2027-01-12 SB-109 Policy engine v0

**Wave/slice:** W1 / S1.4 · **Author:** name · **PR:** #123

**What:** what changed, in two or three sentences.

**Why this way:** the design choice and the alternatives rejected.

**Learned:** what surprised us. This is the most valuable line.

**Laws:** LAW-03 Tested → Enforced. Mutation check: mapping Unknown to Allow fails 14 cases.

**Next:** the obvious follow-up, if any.
```

## Bug entry format

```markdown
## BUG-0007 Chunks from a revoked group still cited for 9 minutes

| Found | Severity | Ticket | Status |
|---|---|---|---|
| 2027-02-03, W3 pilot | Sev-1 | SB-302 | Fixed |

**Symptom:** …
**Root cause:** …
**Fix:** …
**Caught by now:** LAW-01 property test `stale_prefilter` + regression `BUG-0007_revoked_group`.
**Which law should have caught it, and why didn't it?** …
```

## Ticket status

Each ticket file has a `Status` row with one of four values:

| Status | Meaning |
|---|---|
| Open | Not started |
| In progress | Someone is on it |
| In review | The work is done, but an acceptance item needs a human reviewer or a CI run that hasn't happened yet. The ticket's **Progress** section says exactly what's left |
| Done | Every acceptance item met |

## Definition of done

A ticket is done when **all** of these are true:

- [ ] The man pages listed on the ticket were updated **first** and reviewed.
- [ ] Acceptance criteria on the ticket are met and demonstrated.
- [ ] The ticket's laws are green in CI; the ledger is updated.
- [ ] Any formal model touching this area still checks; state names still match.
- [ ] New guardrail tests were **mutation-checked once** (break the rule, watch it fail).
- [ ] Regression tests cover every new error path and every bug fixed.
- [ ] The dependency rule passes; no vendor import in `internal/core`.
- [ ] No secret, prompt text or document content appears in logs (scanner green).
- [ ] Devlog entry added; changelog, bugs and risks updated if relevant.
- [ ] Two reviewers for changes to `policy`, `audit`, `toolbus`, `approvals`, `release` or `proofs`.
