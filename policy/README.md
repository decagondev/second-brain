---
title: Policy
summary: The platform's Cedar schema and base policies, shipped in every release bundle.
---

# Policy

| File | What it is |
|---|---|
| [schema.cedarschema](schema.cedarschema) | Entity and action schema every policy validates against |
| [platform/base.cedar](platform/base.cedar) | Platform base policies: tenant scope, separation of duties |

Tenant policies live next to each tenant's manifest and can only **narrow** these ([sb-policy(5)](../manual/sb-policy.5.md)). Every policy file and every `cedar` block in the docs is validated against the schema in CI (SB-112).
