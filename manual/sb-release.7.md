---
title: "sb-release(7)"
man: "7"
summary: Bundles, channels, rings, rollback and attestation.
---

# sb-release(7)

## NAME

sb-release — how changes reach tenants, and how we prove what runs

## SYNOPSIS

```text
bundle = images + prompts + tools + policies + model + SBOM + eval results  → rel_DATE_DIGEST
rings:   0 internal → 1 friendly (testing) → 2 all (stable)
```

## DESCRIPTION

### Bundles

A release is a signed **bundle**. Anything that changes behaviour is inside it and hashed into the release id, including prompts and the model version (ADR-0013). A prompt change is a release.

### Builds

Builds are hermetic and reproducible: each release is built twice independently and the digests must match. Lockfiles are regenerated every release. Artefacts are signed and carry an SPDX SBOM (ADR-0005).

### Channels

Tenants choose `stable` or `testing`, never a version (ADR-0004).

### Rings and promotion

A gated release goes to ring 0 (the internal tenant), then ring 1 (friendly tenants on `testing`), soaks for at least 48 hours, then can be promoted to `stable`. **Promote guard:** promotion is refused unless every stable tenant runs the current stable. That is what keeps everyone at most one behind (LAW-08).

### Rollback

`sb release rollback` makes the previous stable current, revokes the bad release and redeploys affected cells, in under 10 minutes.

### Attestation

Cells verify signatures before starting a container, and report running digests hourly. Any mismatch is Sev-1 (LAW-06).

### CVEs

A CVE fix is a patch release through the normal path, within 24 hours on `testing`. Because no one is more than one behind, everyone has it after one promotion.

## LAWS

[LAW-06](sb-laws.7.md#law-06), [LAW-08](sb-laws.7.md#law-08).

## EXAMPLES

```sh
sb release status
sb release promote rel_2027-03-09_4f7a20 --to stable
sb release rollback --reason "citation regressions in ring 2"
```

## SEE ALSO

[sb-release(1)](sb-release.1.md), [13 Release and supply chain](../docs/architecture/13-release-and-supply-chain.md)
