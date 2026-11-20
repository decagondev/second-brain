---
title: "Space red-team game day runbook"
summary: "Attack a staging cell's department spaces with a friendly tenant, and record what broke (SB-1208)."
---

# Space red-team game day runbook

Once before the first company relies on HR or Finance spaces, then yearly, in staging, on a cell holding a friendly tenant's real-shaped test data (`acme-logistics` or `beta-freight`, never a real customer). The plan and the attacks are in [docs/red-team-spaces.md](../red-team-spaces.md). Rehearse first: `make evals` must pass `space-redteam`, and `go test ./laws/...` must have LAW-19 to LAW-22 green.

## Roles

| Role | Does |
|---|---|
| Attacker | tries every attack in the plan, then anything else they can think of |
| Scribe | writes each attempt, its time (UTC) and its result as it happens |
| Tenant steward | a steward of the friendly tenant's People and Finance spaces, who confirms what should and shouldn't be visible |

## Plan (a week before)

1. Pick the tenant and get its stewards' agreement in writing. Expect: the tenant, the spaces in scope and the window in the retro.
2. Seed the staging sources with the plan's mistakes: one mislabelled case file, one misfiled restricted file, one planted instruction, one deal file shared by its owner, one moved person. Expect: a list of the seeded items in the retro, with their source ids.
3. Run `sb space diff` on the planned move (SB-1206). Expect: the moved person loses the space, and nothing else changes.

## Run

1. For each attack in the plan, ask as the attacker in the web UI and in Slack. Expect: every row's expected result, with the decision list of the trace (`sb ask --trace`) showing the refusing check (`space_denied`, `label_denied`, `barrier_denied`, `owner_unknown`).
2. Apply the move in the IdP. Expect: the moved person's next ask in the space is `ESPACE`, within 5 minutes of the SCIM change.
3. Try attack 9 on purpose, to confirm the accepted risk behaves as written. Expect: answered, and the steward agrees the label at the source was the mistake.
4. Free-form: anything else. Expect: every attempt in the scribe's log.
5. Check the audit trail: `sb audit tail --space people` and `--space finance`. Expect: every attempt that reached a decision is there, with ids and no content.

## After

1. Every attack that worked, or behaved unlike the plan, is a finding in `security/findings.yaml` (source `internal`), with its severity. Expect: `make check` passes, so no critical or high finding is open.
2. Every new attack becomes a case in `evals/suites/space-redteam.yaml`. Expect: the suite still passes, or the finding is open.
3. Write the retro: attempts, findings, time spent. Expect: the retro linked from SB-1208's Progress, and SB-1208 moved to Done.
