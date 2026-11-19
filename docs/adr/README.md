---
title: Decision Records
summary: Architecture decision records, ADR-0001 onward.
---

# Architecture decision records

An ADR records one decision, the alternatives we rejected and the consequences we accept. ADRs are never edited after acceptance; a later ADR supersedes an earlier one.

| ADR | Decision | Status |
|---|---|---|
| [ADR-0001](ADR-0001-cell-per-tenant.md) | One cell per tenant, stamped from one template | Accepted |
| [ADR-0002](ADR-0002-source-acl-is-authority.md) | The source's live ACL is the authority; the index ACL is only a prefilter | Accepted |
| [ADR-0003](ADR-0003-receipts-define-ran.md) | An action ran only if the target returned a receipt | Accepted |
| [ADR-0004](ADR-0004-channels-not-versions.md) | Tenants choose a channel, never a version; nobody is more than one behind | Accepted |
| [ADR-0005](ADR-0005-supply-chain.md) | Reproducible, signed builds with SBOM; lockfiles regenerated every release | Accepted |
| [ADR-0006](ADR-0006-build-vs-buy.md) | Buy commodity infrastructure; build the trust layer | Accepted |
| [ADR-0007](ADR-0007-language-and-stack.md) | Go for services and CLI; Python for evals; TypeScript for UI | Accepted |
| [ADR-0008](ADR-0008-policy-language-cedar.md) | Write policy in Cedar | Accepted |
| [ADR-0009](ADR-0009-manual-first.md) | The manual is the spec, and CI enforces it | Accepted |
| [ADR-0010](ADR-0010-formal-methods-from-wave-0.md) | TLA+ and Alloy from Wave 0; laws as the main gate from Wave 3 | Accepted |
| [ADR-0011](ADR-0011-no-kubernetes-in-v1.md) | No Kubernetes in v1; use a managed container runtime | Accepted |
| [ADR-0012](ADR-0012-audit-hash-chain.md) | Audit is an append-only hash chain in the cell database, anchored hourly to WORM storage | Accepted |
| [ADR-0013](ADR-0013-release-bundle-pins-behaviour.md) | Prompts, tools, policies and model version are part of the release | Accepted |
| [ADR-0014](ADR-0014-parent-org-is-a-tenant.md) | If the parent organisation reads company data, it does so as a separate tenant, never as a superuser | Proposed |
| [ADR-0015](ADR-0015-factory-and-handoff.md) | The control plane is a factory; custody of a cell can be handed off | Accepted (W7) |
| [ADR-0016](ADR-0016-stdlib-only-go-in-w0.md) | Go standard library only in W0; property tests use an in-repo helper | Accepted |
| [ADR-0017](ADR-0017-first-third-party-go-modules.md) | First third-party Go modules: cedar-go and pgx, pinned, vendored and allow-listed | Accepted |
| [ADR-0018](ADR-0018-cloud-neutral-cells-behind-ports.md) | Cells are cloud-neutral; each cloud is an adapter behind the Placement and Keys ports | Accepted |
| [ADR-0019](ADR-0019-provider-neutral-model-gateway.md) | A provider-neutral model gateway with adapters for OpenAI-compatible APIs, Anthropic, Groq and OpenRouter | Accepted |

## Sign-off (SB-014)

Every ADR needs two reviewers before W0 can exit. Fill in names and dates; an empty cell means not yet reviewed.

| ADR | Reviewer 1 | Reviewer 2 |
|---|---|---|
| ADR-0001 | | |
| ADR-0002 | | |
| ADR-0003 | | |
| ADR-0004 | | |
| ADR-0005 | | |
| ADR-0006 | | |
| ADR-0007 | | |
| ADR-0008 | | |
| ADR-0009 | | |
| ADR-0010 | | |
| ADR-0011 | | |
| ADR-0012 | | |
| ADR-0013 | | |
| ADR-0014 | | |
| ADR-0015 | | |
| ADR-0016 | | |
| ADR-0017 | | |
| ADR-0018 | | |
| ADR-0019 | | |

## Template

Copy [ADR-0000-template.md](ADR-0000-template.md). Number sequentially. Status is one of Proposed, Accepted, Superseded by ADR-nnnn.
