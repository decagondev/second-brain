---
title: "Hand-off runbook"
summary: "Move a cell from factory custody into the company's own cloud account under its own key (SB-701)."
---

# Hand-off runbook

For the factory operators and the tenant owner moving one cell to `tenant` custody. The concepts and phases are in [sb-factory(7)](../../manual/sb-factory.7.md#the-hand-off-exactly-sb-701), and the commands in [sb-factory(1)](../../manual/sb-factory.1.md). Every step names its expected result. If a result differs, stop: before cutover, `sb factory handoff abort` is always safe.

## Roles

| Role | Who | Does |
|---|---|---|
| Planner | a factory operator | makes the plan and runs `execute` |
| Factory approver | a **second** factory operator | approves the plan hash for the factory |
| Tenant owner | a member of the tenant's IdP group `sb-owners` | prepares the account and KMS, approves the plan hash for the tenant |
| Security reviewer | the tenant's security lead | checks the access-path list before approving, and `sb factory access` after |

## Before (the tenant, a week ahead)

1. Create the target cloud account (GCP project) in the cell's pinned region. Expect: the account id, written in the hand-off ticket.
2. Run the deploy-role bootstrap script from the release (SB-703) in that account. Expect: a scoped deploy role that can create only the cell's resources, and its id in the ticket.
3. Create a KMS key ring and key for the cell, held by the tenant. Expect: the key's resource name in the ticket, with the factory's deploy role allowed to wrap and unwrap only.
4. Make sure the people who will approve are in `sb-owners` in the tenant's IdP, with step-up enabled. Expect: `sb factory status TENANT`, run by them, works and shows `custody factory`.

## Plan and approve

1. Planner: `sb factory handoff plan TENANT --to tenant-account://gcp/ID --region REGION [--retain GRANT]`. Expect: a plan hash, the resources, the key steps, every factory access path to revoke, and the retained grants.
2. Security reviewer: compare the revoke list with what you know of the factory's access (deploy role, break-glass roles, IAM bindings, routes, the key wrap). Expect: nothing you know of is missing. If something is, don't approve; the planner fixes the enumeration and makes a new plan.
3. Factory approver: `sb factory handoff approve PLAN --as factory`. Expect: the approval recorded for that hash; the planner's own approval is refused with `EPERM`.
4. Tenant owner: `sb factory handoff approve PLAN --as tenant`, signed in through the tenant's IdP with step-up. Expect: the approval recorded for that hash; `handoff show PLAN` lists both approvals.

## Execute

1. Planner: `sb factory handoff execute PLAN`. Expect: the phases `Provisioned`, `KeyDual`, `Moved` and `Verified` in turn, each with a custody audit row. The factory cell keeps serving throughout.
2. Watch the `Verified` step. Expect: `sb tenant verify` passes, the audit chain verifies with the cell's pinned key, and the attestation matches. If any check fails, `execute` stops before cutover. Fix the problem and run `execute` again, or abort.
3. Cutover runs. Expect: traffic on the cell's hosts reaches the tenant-account cell, the smoke test passes, and the phase is `CutOver`.
4. The factory's wrap of the root key is destroyed. Expect: the phase is `TenantOwned` and `sb factory status TENANT` shows `custody tenant`, `placement tenant-account://gcp/ID` and `key_custodian tenant`.

## After

1. Security reviewer: `sb factory access TENANT`. Expect: exit 0, with only the retained grants listed. Any unlisted path is a Sev-1 (LAW-17).
2. Tenant owner: try to decrypt a backup with a factory credential (the factory operator provides one for the test). Expect: refused. Only the tenant's KMS unwraps the root key (LAW-18).
3. Record the hand-off in the devlog: plan hash, both approvers, times of each phase, and the `access` result. Expect: the hand-off record in the registry's custody history matches.
4. Put the monthly `sb factory access TENANT` check on the tenant's calendar. Expect: the first run is booked a month out.

## Abort

Before cutover, either side may stop: `sb factory handoff abort PLAN --reason TEXT`. Expect: the phase is `Managed`, the tenant's key wrap and any resources provisioned in the tenant account are removed, and the factory cell never stopped serving. After cutover, going back is a hand-back with a new plan and new approvals.

Laws: [LAW-17](../../manual/sb-laws.7.md#law-17), [LAW-18](../../manual/sb-laws.7.md#law-18).
