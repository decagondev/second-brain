---
title: Examples
summary: Example manifests, policies, eval suites and a connector.
---

# Examples

Every example here is referenced from the manual, and CI checks them (SB-009): valid manifests must pass `sb tenant check`, the invalid one must produce exactly its `.expected` output, and the connector must compile.

| Example | Shows | Manual |
|---|---|---|
| [tenants/pilot.yaml](tenants/pilot.yaml) | Day 0–14 pilot manifest | [sb-tenant(5)](../manual/sb-tenant.5.md) |
| [tenants/full.yaml](tenants/full.yaml) | Day 46+ manifest with one gated workflow | [sb-tenant(5)](../manual/sb-tenant.5.md) |
| [tenants/tenant.schema.json](tenants/tenant.schema.json) | JSON Schema for editors, generated from the checker's types | [sb-tenant(5)](../manual/sb-tenant.5.md) |
| [tenants/invalid.yaml](tenants/invalid.yaml) | Every common mistake, and [the errors `check` prints](tenants/invalid.expected) | [sb-tenant(1)](../manual/sb-tenant.1.md) |
| [policies/acme-logistics.cedar](policies/acme-logistics.cedar) | Tenant policies that narrow platform policy | [sb-policy(5)](../manual/sb-policy.5.md) |
| [evals/permission.yaml](evals/permission.yaml) | Permission suite, including a stale prefilter and an injected timeout | [sb-eval(5)](../manual/sb-eval.5.md) |
| [evals/injection.yaml](evals/injection.yaml) | Injection suite with canaries | [sb-evals(7)](../manual/sb-evals.7.md) |
| [evals/gate.yaml](evals/gate.yaml) | Release gate thresholds | [sb-evals(7)](../manual/sb-evals.7.md) |
| [connector/](connector/README.md) | A minimal read-only connector in Go | [sb-connector(3)](../manual/sb-connector.3.md) |
