---
title: Glossary
summary: One line per term used in any interface.
---

# Glossary

| Term | Meaning |
|---|---|
| **ACL prefilter** | A copy of a document's permissions taken at ingest, used only to narrow retrieval. Never grants access |
| **Act** | The connector call that performs a write and returns a receipt |
| **Action** | A proposed change to a company system, produced by a gated workflow |
| **Agent** | A pinned definition (prompt, tools, limits) in the release bundle: `ask`, `draft-reply`, or a workflow |
| **Anchor** | The signed hourly copy of the latest audit hash in WORM storage |
| **Approval** | A record binding one approver to one payload hash |
| **Barrier** | An information barrier: two groups in the manifest whose members never see each other's content, drafts or actions (LAW-22) |
| **Bundle** | A signed release: images, prompts, tools, policies, model id, SBOM, eval results |
| **Cell** | One tenant's isolated stack: database, index, vault namespace, key, queue, region |
| **Channel** | `stable` or `testing`. Tenants choose a channel, never a version |
| **Citation** | `kind:source-id@version` pointing at an allowed chunk |
| **Claim** | One statement in an answer; cited or `uncited` |
| **Connector** | Adapter for one kind of company system, implementing `fetch`, `acl`, `changes`, `act` |
| **Contract kit** | The test suite every connector must pass |
| **Control plane** | Shared services with no tenant content: registry, releases, kill flags, attestation |
| **Custody** | Who holds a cell: `factory`, `tenant` or `detached`. A factory record, never a manifest field |
| **Detach** | A tenant-custody cell leaves the release train with an export package |
| **Draft** | Text an agent writes for a human to send. Not an action |
| **Eval** | A test of model-dependent behaviour, scored over cases |
| **Factory** | The control plane, cell template and release train that produce and run every tenant's Second Brain |
| **Gate** | The release check: evals, laws, models, signatures |
| **Hand-off** | Dual-approved transfer of a cell's custody from the factory to its company |
| **Key custodian** | Whoever's KMS can unwrap a cell's root key |
| **Kill switch** | Runtime stop for an agent, connector or tenant; effective in 10 seconds or less |
| **Label** | A document's sensitivity: `public`, `internal`, `confidential` or `restricted`. From the source if it has one, else the space default |
| **Law** | An invariant the platform guarantees for every input (`LAW-nn`) |
| **Live ACL** | The connector's `acl` call at answer time; the authority on who may see a document |
| **Manifest** | The one YAML file describing a tenant |
| **Placement** | Which cloud account a cell's resources live in |
| **Promote guard** | Rule that a release becomes stable only when all stable tenants run the current stable |
| **Receipt** | Target-issued id proving an action happened |
| **Ring** | A rollout stage: 0 internal, 1 friendly (testing), 2 all (stable) |
| **Scope** | `tenancy.Scope`: tenant + user + groups, carried by every call |
| **Slice** | A thin, end-to-end, demoable increment inside a wave |
| **Space** | A department brain inside a company's cell: a scope over its sources, with its own members, stewards, agents, label default, retention and data key. It can only narrow access (ADR-0020) |
| **Steward** | A department lead who proposes changes to their space and re-certifies its members; the tenant operator approves |
| **Tenant** | One company using the platform |
| **Tool bus** | The single path for tool calls: kill, authorise, audit, approve, execute |
| **Trace** | One ask or agent run (`tr_…`) |
| **Uncited** | Label set by code on a claim with no valid citation |
| **Unknown** | A permission result that isn't a clear allow or deny. Always refused |
| **Wave** | A release-sized group of slices with an exit gate |
