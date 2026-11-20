---
title: "SB-1103: Code connector (read-only)"
summary: "Engineering knowledge lives in repositories, pull requests and issues."
---

# SB-1103: Code connector (read-only)

| Field | Value |
|---|---|
| Epic | [E11 Department Templates and Connectors](README.md) |
| Wave / slice | W11 / S11.2 Engineering sources |
| Stories | US-11.2 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-03](../../manual/sb-laws.7.md#law-03), [LAW-07](../../manual/sb-laws.7.md#law-07) |
| Depends on | [SB-201](../E03-connectors-ingestion/SB-201-connector-contract-sdk-and-contract-kit.md) |
| Status | In review |

## Why

Engineering knowledge lives in repositories, pull requests and issues.

## Acceptance criteria

- [x] GitHub read-only: READMEs, docs folders, issues and pull-request descriptions; source code only if the space opts in
- [x] ACL from repository permissions and teams; a private repo with unknown access is `Unknown`
- [x] Secret scanner runs on every chunk; a hit is dropped and counted, never stored (LAW-07)
- [x] Passes the contract kit against a recorded fake; a real org stays In review (the real organisation run is left)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Connector
- [x] ACL mapping
- [x] Secret drop
- [x] Contract kit
- [x] Write or extend the law tests: LAW-01, LAW-03, LAW-07
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S11.2): sb-connector(3) gained `code: github`, sb-ingest(7) "Secrets never stored", and sb-tenant(5) and sb-ask(3) the `code` kind; `code` is a kind of the manifest, kill switches, vault references and asks. `internal/adapters/connectors/github` reads one organisation over REST and GraphQL with a token from the vault: every README, `docs/` file and Markdown file, issues and pull requests (description only), and other files only for repositories a space opts in with `REPO/**`. Locations are `REPO/PATH`, `REPO/issues/N` and `REPO/pulls/N`. Access is asked live per repository: the asker by SSO email, members for public repositories, collaborator permission (read or higher) for private and internal ones, `Unknown` for anything else; the prefilter is the members token or the collaborators' SSO emails. Changes come from issues, commits (removals are deletes) and `MemberEvent`/`PublicEvent` (permission-only changes for the whole repository), strictly after the cursor. `githubtest` records the API; the connector passes the contract kit. Every chunk of every connector now passes `ingest.SecretIn` (private keys, cloud, GitHub, Slack, Stripe and Google keys, and long secret assignments) and a match is dropped and counted, never stored. Law tests: `TestLAW03_GitHubUnknownIsNeverAllow`, `TestLAW01_GitHubAccessIsLive` (a check that keeps its first answer is caught) and `TestLAW07_IngestNeverStoresASecret` (an ingest without the scanner is caught).

**Left:** a run against a real organisation, and the runtime wiring of `SourceRepos` from the spaces' `code` scopes.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
