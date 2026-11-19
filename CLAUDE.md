# CLAUDE.md

Read `AGENTS.md` first: those rules apply to every change. This file says where the project is and what to do next.

## State (2026-11-19)

- **W0 (Manual and Laws):** 10 Done, 6 In review. `make check` is green on a clean clone.
  - SB-004, SB-005, SB-014: need two human reviewers (sign-off tables in `docs/adr/README.md` and `docs/07-threat-model.md`).
  - SB-011, SB-012, SB-013: models and all 13 mutations pass, locally and in CI (2026-11-19). Left: two reviewers.
- **W1 (First Cell):** 8 Done; SB-104 In review (30-minute cell needs a cloud adapter, ADR-0018); SB-109 to SB-112 In review (second reviewer for policy and audit). User reviewed on 2026-11-19; CI green.
- **W2 (Connectors and ingestion):** SB-201, SB-202, SB-205 and SB-206 Done (embeddings: local hashing embedder until Q-04's provider). SB-203 (Google Drive, the user's choice) In review: needs a real Workspace service account for a sandbox run. Linear suggested for tickets (SB-211), not yet confirmed. Log only through `internal/obs` (redacting).
- Status of every ticket: `epics/README.md` (Progress) and each ticket's `Status` row and Progress section.

## Decisions for W1 (confirm with the user before acting)

1. **Cloud and IaC tool** for the cell template (SB-103, SB-104). **Decided 2026-11-19 (ADR-0018):** cells are cloud-neutral behind the `Placement` and `Keys` ports. Build the `local` adapter now; AWS, GCP and Azure adapters come later, and each picks its IaC tool. Real-cloud acceptance items stay In review until then.
2. **First third-party Go modules:** **Decided 2026-11-19 (ADR-0017):** `cedar-go` v1.8.0 and `pgx/v5` v5.8.0, the newest that build with Go 1.24. Every allowed `go.sum` line is in `tools/deps/allowed.txt`, and `make deps` enforces it. Vendor each module in the change that first imports it, and only adapters may import them.
3. **Manifest YAML (SB-101):** the default plan is a strict in-repo parser for the manifest's subset, keeping line and column for errors. Unknown fields are errors.
4. **Planning fix:** add a W1-level LAW-02 law test (requests and sessions never cross tenants) and a LAW-12 test (deprovisioning within 5 minutes, using a fake clock). Both go through the red ratchet (`laws/expected_red.txt`, which now takes per-test lines). The retrieval-level LAW-02 test stays red until SB-301. Done so far: the cell-level (SB-103) and request-level (SB-105) LAW-02 tests. LAW-12 (SB-107) and the session/scope one (SB-108) are done too.

## How to work a ticket

1. Update the man pages listed on the ticket **first** (CI enforces this).
2. Write or extend the law tests; they must fail for the right reason.
3. Implement behind `internal/core/ports`; keep vendor code in `internal/adapters/`.
4. `make check`; plus `make proofs` and `make crosscheck` if approvals, rollout or permissions changed.
5. Update the ticket (Status, checkboxes, Progress), add a devlog entry (What / Why this way / Learned / Laws), and update the ledger and changelog.
6. Ticket files are edited by hand now. Don't re-run the plan generator over them.

## Toolchain

Go 1.24 (`GOTOOLCHAIN=local`) · Postgres 17 for `make integration` (`SB_TEST_POSTGRES_DSN`, e.g. `postgres:///sb_test?host=/var/run/postgresql` in WSL) · Python 3.12+ (`pip install pyyaml markdown cedarpy`) · `golangci-lint` v2 · Java 21 for proofs · Node 22 plus `npm install --prefix tools/check` and Playwright for `make diagrams`.

**On Windows, run every command inside WSL** (`wsl.exe -e bash -lc '…'` from `/mnt/c/...`), never PowerShell or Git Bash. The Makefile and `tools/ci/*.sh` assume a POSIX shell, and Windows tools mangle paths (TLC rejects `proofs/tla/X.tla`; the Java launcher expands `*` arguments). Install the toolchain inside the WSL distro, not on Windows.

## Don'ts

- Don't treat `Unknown`, a timeout or a missing source as allow.
- Don't hard-code a cloud account id or assume the factory holds a cell's keys (ADR-0015).
- Don't put real customer names anywhere. Use `acme-logistics`, `beta-freight`, `gamma-parts`.
- Don't delete regression or smoke tests.
