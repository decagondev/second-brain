# CLAUDE.md

Read `AGENTS.md` first: those rules apply to every change. This file says where the project is and what to do next.

## State (2026-11-19)

- **W0 (Manual and Laws):** 10 Done, 6 In review. `make check` is green on a clean clone.
  - SB-004, SB-005, SB-014: need two human reviewers (sign-off tables in `docs/adr/README.md` and `docs/07-threat-model.md`).
  - SB-011, SB-012, SB-013: models and all 13 mutations pass, locally and in CI (2026-11-19). Left: two reviewers.
- **W1 (First Cell):** 8 Done; SB-104 In review (30-minute cell needs a cloud adapter, ADR-0018); SB-109 to SB-112 In review (second reviewer for policy and audit). User reviewed on 2026-11-19; CI green.
- **W2 (Connectors and ingestion):** built (2026-11-19). Done: SB-201, SB-202, SB-205, SB-206. In review: SB-203 (Drive), SB-204 (Slack), SB-211 (Linear) need real sandboxes; SB-207 needs the tool bus for its eval; SB-208 to SB-210 need staging and the cell runtime. Log only through `internal/obs`. 
- **W3 (Cited answers):** S3.1 to S3.3 done (SB-301, SB-302, SB-304, SB-305 Done; SB-303 In review for staging p95; SB-306 In review for the audit-chain record, SB-401; SB-307 web UI In review for a human WCAG audit). LAW-01 to LAW-03, LAW-07 (prompts) and LAW-14 green. Next: **SB-308 (Slack app entry point)**.
- Status of every ticket: `epics/README.md` (Progress) and each ticket's `Status` row and Progress section.

## Decisions for W1 (confirm with the user before acting)

1. **Cloud and IaC tool** for the cell template (SB-103, SB-104). **Decided 2026-11-19 (ADR-0018):** cells are cloud-neutral behind the `Placement` and `Keys` ports. Build the `local` adapter now; AWS, GCP and Azure adapters come later, and each picks its IaC tool. Real-cloud acceptance items stay In review until then.
2. **First third-party Go modules:** **Decided 2026-11-19 (ADR-0017):** `cedar-go` v1.8.0 and `pgx/v5` v5.8.0, the newest that build with Go 1.24. Every allowed `go.sum` line is in `tools/deps/allowed.txt`, and `make deps` enforces it. Vendor each module in the change that first imports it, and only adapters may import them.
3. **Manifest YAML (SB-101):** the default plan is a strict in-repo parser for the manifest's subset, keeping line and column for errors. Unknown fields are errors.
4. **Planning fix:** add a W1-level LAW-02 law test (requests and sessions never cross tenants) and a LAW-12 test (deprovisioning within 5 minutes, using a fake clock). Both go through the red ratchet (`laws/expected_red.txt`, which now takes per-test lines). The retrieval-level LAW-02 test stays red until SB-301. Done so far: the cell-level (SB-103) and request-level (SB-105) LAW-02 tests. LAW-12 (SB-107) and the session/scope one (SB-108) are done too.

## Decisions for W2 to W7 (the user, 2026-11-19)

1. **LLM:** an abstract model gateway (SB-305) with adapters for OpenAI-compatible APIs, Anthropic, Groq and OpenRouter (ADR-0019). Tests use fakes.
2. **Embeddings:** the free local hashing embedder for now.
3. **Vector store:** Postgres with pgvector (`sudo apt install -y postgresql-17-pgvector` in WSL).
4. **Vendors:** Google Drive (docs), Slack (chat and the chat-app entry point), Linear (tickets), HubSpot (CRM). Every vendor is built against a recorded fake; real-sandbox runs stay In review.
5. **Web UI:** TypeScript, React and Vite, served by the cell.
6. **Go modules:** standing approval for vetted modules, each with an ADR and allow-list entry; tell the user. Go stays 1.24.
7. **Release and ops:** cosign keyless signing in CI with an in-repo SPDX SBOM; local adapters plus Postgres for audit anchors, kill switches and backups; logs via `internal/obs`, a Prometheus text endpoint, and Sev-1 alerts through a webhook adapter.
8. **First real cloud (W7):** GCP (and Firebase where it fits).
9. **Process:** one commit per slice, pushed to `main` after `make check` passes; no PRs, no attribution. Human-only items (pilot, onboarding, pen test, game days, second reviewers) get tooling and runbooks and stay In review. Stop only for new decisions, installs, or a failing check that can't be fixed.

## How to work a ticket

1. Update the man pages listed on the ticket **first** (CI enforces this).
2. Write or extend the law tests; they must fail for the right reason.
3. Implement behind `internal/core/ports`; keep vendor code in `internal/adapters/`.
4. `make check`; plus `make proofs` and `make crosscheck` if approvals, rollout or permissions changed.
5. Update the ticket (Status, checkboxes, Progress), add a devlog entry (What / Why this way / Learned / Laws), and update the ledger and changelog.
6. Ticket files are edited by hand now. Don't re-run the plan generator over them.

## Toolchain

Go 1.24 (`GOTOOLCHAIN=local`) · Postgres 17 for `make integration` (`SB_TEST_POSTGRES_DSN`, e.g. `postgres:///sb_test?host=/var/run/postgresql` in WSL) · Python 3.12+ (`pip install pyyaml markdown cedarpy`) · `golangci-lint` v2 · Java 21 for proofs · Node 22 plus `npm install --prefix tools/check` and Playwright for `make diagrams`; `make web` runs `npm ci` in `web/`.

**On Windows, run every command inside WSL** (`wsl.exe -e bash -lc '…'` from `/mnt/c/...`), never PowerShell or Git Bash. The Makefile and `tools/ci/*.sh` assume a POSIX shell, and Windows tools mangle paths (TLC rejects `proofs/tla/X.tla`; the Java launcher expands `*` arguments). Install the toolchain inside the WSL distro, not on Windows.

## Don'ts

- Don't treat `Unknown`, a timeout or a missing source as allow.
- Don't hard-code a cloud account id or assume the factory holds a cell's keys (ADR-0015).
- Don't put real customer names anywhere. Use `acme-logistics`, `beta-freight`, `gamma-parts`.
- Don't delete regression or smoke tests.
