---
title: Runbooks
summary: One runbook per Sev-1 law alert (SB-608).
---

# Runbooks

One runbook for each law with a production alert ([sb-observability(7)](../../manual/sb-observability.7.md#law-alerts-exactly-sb-608)).

- [LAW-01](LAW-01.md): Cite only what the asker can open
- [LAW-02](LAW-02.md): No tenant crossing
- [LAW-04](LAW-04.md): Audit before act
- [LAW-06](LAW-06.md): Only signed, gated builds run
- [LAW-07](LAW-07.md): No secret in prompt, log or audit
- [LAW-08](LAW-08.md): At most one behind stable
- [LAW-11](LAW-11.md): Audit chain unbroken
- [LAW-12](LAW-12.md): Deprovision revokes fast
- [LAW-16](LAW-16.md): The manifest is the truth
- [LAW-17](LAW-17.md): Hand-off leaves no unlisted access
- [LAW-18](LAW-18.md): The root key always has a custodian

Hand-off:

- [Hand-off](handoff.md): move a cell to the company's own cloud account under its own key (SB-701)

Releases:

- [Pausing release pulls](pause-pulls.md): a tenant-custody cell stops taking releases for a while, or detaches (SB-707)

Game days:

- [DR game day](dr-game-day.md): lose a cell's database, restore it inside its region, and score it (SB-613)
