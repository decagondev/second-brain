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

The bundle manifest, `bundle.json` (SB-503):

| Field | Holds |
|---|---|
| `code` | The SHA-256 of each built artefact, by name (`sb`, `cell`, `web`) |
| `prompts` | Each bundled prompt's name and SHA-256 |
| `tools` | Each registered tool: name, connector, consequential, workflow, and the SHA-256 of its payload schema |
| `policies` | Each platform policy file and the schema, with SHA-256 |
| `model` | `provider`, `model` and parameters (`max_tokens`) |
| `gate` | The SHA-256 of `evals/gate.yaml` the release was gated with |
| `created` | The build date, `YYYY-MM-DD` |

The release id is `rel_<created>_<first 6 hex of SHA-256 of the canonical manifest without the id>`, for example `rel_2027-03-02_9b1c4e`. The canonical form is RFC 8785 JSON, as for payload hashes. Changing **any** field, a prompt byte, a policy, the model, a tool's schema or one artefact, gives a different id. Every audit row carries the running release id (sb-audit(5)), so any action can be traced to the exact prompts and policies it ran under.

### Builds

Builds are hermetic and reproducible: each release is built twice independently and the digests must match. Lockfiles are regenerated every release. Artefacts are signed and carry an SPDX SBOM (ADR-0005).

Exactly (SB-504):

- **Hermetic.** `make release-build` builds with `CGO_ENABLED=0`, `-trimpath`, `-buildvcs=false` and an empty build id, with the commit passed in by `-ldflags`, so the output depends only on the source and the Go toolchain version. `tools/ci/repro.sh` builds twice, in two fresh directories with separate build caches, and fails if any digest differs.
- **SBOM.** `tools/release/sbom` writes SPDX 2.3 JSON: one package for each Go module in the binary (from its embedded build info, with its `go.sum` hash) and one for each npm package in `web/package-lock.json` (with its integrity hash). The SBOM is an artefact of the bundle, so its digest is in `bundle.json` and covered by the release signature.
- **Signed.** The release signs the bundle's digest. In CI, signing is **cosign keyless**: GitHub's OIDC identity for this repository's release workflow, with a Rekor transparency-log entry. Development bundles are signed with a local Ed25519 key. Either way, a signature covers the bundle digest, and through it every artefact, prompt, policy, tool schema, the model and the SBOM.
- **Gated.** The eval gate's verdict for a release is a signed statement too, `gate.json` `{ "release", "passed", "results_sha256" }`, signed the same way.

### What a cell admits (SB-506)

A cell **pulls** bundles from the release store over HTTPS. The factory never connects into a cell. Before it starts anything, the cell checks, in order, and refuses on the first failure (`EUNSIGNED`, and a Sev-1 alert):

1. `bundle.json`'s id matches its contents (sb-release(7), Bundles);
2. the bundle signature verifies for the release identity the cell trusts;
3. every artefact it will run, the SBOM included, has the digest listed in the bundle;
4. `gate.json` is for this release id, says `passed`, and its signature verifies.

A cell runs only what passed all four. There is no flag to skip one (LAW-06).

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
