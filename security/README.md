---
title: Security review
summary: The external pen test's scope and rules of engagement, the findings register and the attack corpus (SB-612).
---

# Security review

The independent pen test before scale ([sb-laws(7), External review](../manual/sb-laws.7.md#external-review-sb-612)). The testers are people outside the platform team. This folder is what they work from and what they leave behind.

## Scope

| In scope | Notes |
|---|---|
| The edge router and a staging cell's API (`/v1/ask`, `/v1/feedback`, `/v1/drafts`, `/v1/actions`, `/v1/admin/*`) | Two test tenants, `acme-logistics` and `beta-freight`, each with an operator, an approver and two users |
| The web UI and the Slack `/ask` app | A sandbox Slack workspace |
| Connector ingestion | Testers may plant documents in the test tenants' sandbox sources (Drive, Slack, Linear, HubSpot) |
| The release path | Bundle signatures, admission and attestation on staging |
| Cell isolation | Any attempt to read or act across the two tenants |

**Out of scope:** production, real customer data, the identity provider itself, the model provider, volumetric denial of service, and social engineering of staff.

## Rules of engagement

1. Staging only, in the agreed window. Stop and call the security lead at once on finding a **critical** issue, or on seeing anything that looks like real data.
2. Use only the issued test accounts. Never use or keep a credential found during the test beyond showing that it works.
3. Report each finding with the request or document that shows it, the expected and actual result, and a severity (critical, high, medium, low, info).
4. Don't change kill switches, manifests or releases except through the endpoints in scope, and say when you do.

## What the platform team does with a finding

1. Add it to `findings.yaml` (`PT-nnnn`), with the laws it touches. Its status is `open`.
2. If it breaks a law, write the law test that would have caught it and watch it go red.
3. Fix it, then add a `BUG-nnnn` entry to `living/BUGS_AND_MITIGATIONS.md`, name the finding in `docs/07-threat-model.md`, and set the status to `fixed` with the bug id.
4. `make check` (`tools/check/findings.py`) fails while a critical or high finding isn't fixed.

## The attack corpus

`attacks/injection.txt` holds the indirect prompt-injection payloads the testers and we have tried, one per line. `TestLAW09_PenTestCorpusIsFlagged` plants each one in a benign document and requires it to be flagged or stripped. Add every new form a tester finds.
