---
title: Tenants
summary: Each tenant's own repo folder, starting with its golden questions.
---

# Tenants

One folder per tenant. Today each holds the tenant's golden questions in `evals/`, with the world they run against ([sb-eval(5)](../manual/sb-eval.5.md), Tenant golden questions). They run as `tenant/<name>`:

```sh
python3 -m evals run --tenant-dir tenants/acme-logistics --model extractive
```

| Tenant | Golden questions |
|---|---|
| [acme-logistics](acme-logistics/evals/golden.yaml) | Example set for the internal tenant |
