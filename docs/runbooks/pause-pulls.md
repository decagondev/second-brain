---
title: "Pausing release pulls"
summary: "What a tenant does to stop taking releases for a while, and when a pause becomes a detach (SB-707)."
---

# Pausing release pulls

For the tenant owner of a cell in `tenant` custody. The cell pulls signed releases from the factory's store with a read-only token from your own vault ([sb-release(7), Cells in a tenant's account](../../manual/sb-release.7.md#cells-in-a-tenants-account-sb-707)). You can stop that at any time, because the token is yours.

## Pause

1. Tell the factory operators the dates and the reason, in the hand-off ticket. Expect: an acknowledgement. Your cell will hold up promotions for every tenant once it is more than one release behind, so they need to know.
2. Disable the secret version at `vault://tenants/TENANT/release-pull` in your secret manager. Expect: the cell's next pull fails with `EUNSIGNED` and the cell keeps running the release it has.
3. Check the cell still serves. Expect: its smoke test passes on the same release as before, and `sb release status` shows your cell's attested release unchanged.

## Resume

1. Enable the secret version again. Expect: the cell pulls and admits the current stable within one pull interval, and its next attestation shows it.
2. Tell the factory operators. Expect: the promote guard no longer counts your cell as behind.

## When a pause becomes a detach

A pause that lasts longer than one release cycle blocks everyone's promotions. The factory operators will ask you to resume or to detach. To leave the train for good, follow `sb factory detach` ([sb-factory(1)](../../manual/sb-factory.1.md)). Expect: the export package in the directory you name, your cell removed from the promote guard, and the `publish-releases` grant revoked. After detach the factory guarantees nothing about the cell, and LAW-06 and LAW-08 stop applying to it.

Laws: [LAW-06](../../manual/sb-laws.7.md#law-06), [LAW-08](../../manual/sb-laws.7.md#law-08).
