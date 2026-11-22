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
| `catalogue` | SHA-256 of the department template catalogue, `internal/core/manifest/catalogue.json` (sb-departments(7); SB-1101) |
| `model` | `provider`, `model` and parameters (`max_tokens`); for a model run on the cell's own machine (SB-1710), `weights_sha256`, the SHA-256 of the weights file, and `embedder`, the embedding model's id with its weights' SHA-256 (`MODEL@sha256:HEX`), so the release names the exact bytes it answers with |
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

### Cells in a tenant's account (SB-707)

A cell in `tenant` custody (sb-factory(7)) gets releases exactly as a factory cell does: it **pulls**. Its only link to the factory's release store is a read token held in the tenant's own vault, `vault://tenants/TENANT/release-pull`, scoped to reading bundles. The factory's `publish-releases` grant lets the factory publish to its own store, and gives it no way into the tenant's account. The cell admits a bundle only after all four checks above, against the release identity it already trusts. A tampered object, an unknown release, a missing gate record or a wrong token all fail admission with `EUNSIGNED`, and the cell keeps running what it runs.

The release controller holds every cell on the train with its custody. **Tenant-custody cells count in the promote guard** like any other: a stable tenant in its own account that hasn't pulled the current stable blocks promotion, so LAW-08 holds for it too. The controller learns what such a cell runs only from its signed attestations (Attestation, below), never by reaching in.

A tenant may **pause pulls** by withholding the pull token. While paused, the cell falls behind, and once it is more than one behind stable, the promote guard holds every promotion until it pulls again. A pause that should last becomes a **detach** (sb-factory(7), Detach; the runbook `docs/runbooks/pause-pulls.md`). A detached cell leaves the train: the controller drops it from the guard and never deploys to it again, and LAW-06 and LAW-08 stop applying to it, as sb-factory(7) says.

### Channels

Tenants choose `stable` or `testing`, never a version (ADR-0004). The release controller points each channel at one release in the store, `channels/stable` and `channels/testing`, each holding a release id; a cell's updater reads its channel's pointer with its pull token and admits the release it names like any other (SB-1505). A pointer to a release that fails admission changes nothing in the cell. A bundle names each service's image by digest as `image/api`, `image/ingest`, `image/jobs`, `image/updater` (and `image/web`); `make images` builds them and `tools/release/bundle --image NAME=sha256:…` records them (sb-selfhost(7)).

### Rings and promotion

A gated release goes to ring 0 (the internal tenant), then ring 1 (friendly tenants on `testing`), soaks for at least 48 hours, then can be promoted to `stable`. **Promote guard:** promotion is refused unless every stable tenant runs the current stable. That is what keeps everyone at most one behind (LAW-08).

### Rollback

`sb release rollback` makes the previous stable current, revokes the bad release and redeploys affected cells, in under 10 minutes.

### Attestation

Cells verify signatures before starting a container, and report running digests hourly. Any mismatch is Sev-1 (LAW-06).

Exactly (SB-509): every cell **pushes** a signed attestation on start, after each deploy, and every hour. The controller never reaches into a cell, so this works for cells in any account. An attestation is `{ "tenant", "release", "digests": { artefact: sha256 }, "seq", "at" }`, signed with the cell's attestation key (Ed25519, its seed wrapped under the cell's root key, like the audit anchors). The controller checks it against its own records:

| Finding | When | Alert |
|---|---|---|
| `attestation_mismatch` | a digest isn't the bundle's digest for that artefact, an artefact isn't in the bundle, the release isn't the one the controller deployed to that tenant, or the signature doesn't verify for that cell's key | Sev-1 |
| `attestation_replay` | `seq` isn't greater than the last accepted one | Sev-1 |
| `attestation_missing` | no accepted attestation for 2 hours | Sev-1 |

An accepted attestation updates the tenant's attested release and digests, which `sb release status` shows.

### CVEs

A CVE fix is a patch release through the normal path, within 24 hours on `testing`. Because no one is more than one behind, everyone has it after one promotion.

Exactly (SB-505):

- **Lockfiles every release.** `make lockfiles` regenerates `go.sum` and `vendor/` (`go mod tidy`, `go mod vendor`) and `web/package-lock.json` (`npm install --package-lock-only`). The release workflow runs it, and the release fails if any lockfile then differs from the commit. A release is always built from lockfiles that were current when it was cut, and the change is reviewed like code. A new `go.sum` line still needs its ADR and allow-list entry (ADR-0017), because `make deps` enforces the list.
- **Advisory watcher.** A scheduled job runs every day at 06:00 UTC: `govulncheck` over the Go code (only vulnerabilities in code that is actually called), and `npm audit` over the web app's production packages at `high` and above. A finding fails the job, which notifies the maintainers. The job's output names the advisory, the package and the fixed version.
- **The 24-hour path.** The [patch-release playbook](../docs/patch-release-playbook.md) takes a finding to a patch release on `testing`: bump the one dependency, regenerate lockfiles, run `make check`, cut the release through the normal gate and signing, and record the advisory id in the changelog. Nothing skips the gate.

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
