---
title: Risks
summary: The risk register.
---

# Risk register

Likelihood and impact are 1 (low) to 3 (high). Review at every wave exit.

| ID | Risk | L | I | Owner | Mitigation | Status |
|---|---|---|---|---|---|---|
| R-01 | Live ACL checks push p95 ask latency over 6 s | 2 | 2 | Platform lead | Batching, parallel fan-out, deny caching; measured from W3 (SB-303) | Open |
| R-02 | A source's permission model can't be mapped faithfully | 2 | 3 | Connector owner | `Unknown` until it can; contract kit; never relax LAW-03 | Open |
| R-03 | Indirect prompt injection via ingested content | 3 | 3 | Security lead | Sanitiser, data wrapping, argument provenance, injection eval gate (LAW-09) | Open |
| R-04 | Companies demand per-company customisation | 3 | 2 | Product owner | Manifest fields for everyone or nothing (P3); escalation to the parent | Open |
| R-05 | Formal models drift from code | 2 | 2 | Platform lead | Shared state names; spec review in the DoD | Open |
| R-06 | Models not yet run may hide a modelling error | 2 | 2 | Platform lead | Run in CI as the first task of SB-011 to SB-013; mutation checks | Open |
| R-07 | Model provider changes behaviour or terms | 2 | 2 | Parent-org procurement | Model pinned per release; eval gate; provider behind a port | Open |
| R-08 | Unknown rate makes the product feel broken | 2 | 2 | Connector owner | Unknown-rate dashboards and alerts; fix in connectors | Open |
| R-09 | Rollback is not atomic across cells (BUG-0001) | 2 | 2 | Platform lead | Revoked builds start no new consequential calls; refine model in SB-012 | Open |
| R-10 | Approvers rubber-stamp | 2 | 2 | Company leadership | Diff view, required click, approve-to-reject tracking, weekly report | Open |
| R-11 | v1 shortcuts erode the hand-off seams (hard-coded account ids, factory-wide keys, push deploys) | 2 | 3 | Platform lead | SB-114 lint; seams in ADR-0015 each with a test; review checklist | Open |
| R-12 | A detached cell runs unpatched and is still associated with the parent | 2 | 2 | Parent-org legal | Detach requires tenant-owner acceptance; export includes SBOM; registry keeps custody history | Open |
| R-13 | A mislabelled or over-scoped source puts restricted content in a wide space | 2 | 3 | Company stewards | Stricter label wins; source labels win; leak eval suites; quarterly access reviews (SB-1005, SB-1107, SB-1202) | Open |
| R-14 | Too many small spaces make answers worse and people go back to asking around | 2 | 2 | Product owner | Company-wide ask by default; templates merge easily; adoption per space in the weekly report | Open |
| R-15 | Cloud Run cold starts push ask p95 over budget | 2 | 2 | Platform lead | Minimum instances per tenant; measured on staging in W9 (SB-908) | Open |
