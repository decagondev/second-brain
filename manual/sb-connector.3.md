---
title: "sb-connector(3)"
man: "3"
summary: The four-call connector contract every source adapter implements.
---

# sb-connector(3)

## NAME

sb-connector — `Fetch`, `ACL`, `Changes`, `Act`: the connector contract, API version 1

## SYNOPSIS

```go
package ports

const ConnectorAPIVersion = 1

type Connector interface {
    APIVersion() int
    Fetch(ctx context.Context, ref SourceRef) (Document, error)
    ACL(ctx context.Context, user UserRef, refs []SourceRef) ([]Verdict, error)
    Changes(ctx context.Context, cursor Cursor) (ChangeSet, Cursor, error)
    Act(ctx context.Context, a Action, key IdempotencyKey) (Receipt, error)
}
```

## DESCRIPTION

A connector is an adapter between one kind of company system and the platform. It holds no state except a sync cursor and does nothing but these four calls.

### Version negotiation

`APIVersion` returns the contract version the connector implements. The platform loads a connector only if `ports.Negotiate` accepts its version: today exactly `1`. A connector built for a newer or older contract is refused at load time with `ENOTSUP`, never run in a degraded mode.

### Fetch

Returns the document's text and structure, its **version** (monotonic per document, as reported by the source), and an ACL snapshot used only as a prefilter. The same `(ref, version)` must always return the same content hash.

### Location and label (SB-1005)

`Fetch` also returns where the document lives, its **location**, and the source's own sensitivity **label** if it has one, in the four-step vocabulary of sb-spaces(7). Ingest routes documents to spaces by location (sb-ingest(7), Spaces and labels). A connector whose source has no labels returns none, and never invents one.

### Fields (SB-1106)

A connector to a **structured** system (an HRIS, an accounting system) returns a document's data as named **fields** rather than as text, and **declares** every field it can return: its name and whether it's **sensitive** (pay, performance, cases, personal identifiers, bank details, tax ids). It never declares a field it returns under another name. Ingest decides which fields become text (sb-ingest(7), Fields); the connector never does. A connector that returns fields but can't declare them returns nothing that is ingested.

### ACL

The **authority** on who may see what. Called live, at answer time, for a batch of up to 200 refs. Returns one `Verdict` per ref, in order:

| Verdict | When |
|---|---|
| `VerdictAllow` | The source says this user can open this document now |
| `VerdictDeny` | The source says they can't |
| `VerdictUnknown` | **Anything else**: timeout, error, 404, rate limit, unmapped permission model, partial response |

A connector must never return `VerdictAllow` because it couldn't tell. The caller's deadline is 800 ms. A late answer counts as `Unknown`.

### Changes

Returns everything that changed since `cursor`: creates, updates, deletes **and permission changes**. The cursor must be resumable, and replaying a change must be harmless.

### Act

Performs one write for a gated workflow. Must be idempotent per `key` (the approval id) and must return a `Receipt` with the **target-issued** id of the change. If the target gives no id, `Act` reads the change back to obtain one or returns an error. Connectors that don't support writes return `ENOTSUP`.

### Credentials

Resolved at call time from `vault://tenants/<tenant>/connectors/<kind>` through the `Secrets` port. Never stored in a struct field, never logged.

## ERRORS

| Error | Meaning |
|---|---|
| `ENOTSUP` | Call not supported (usually `Act`) |
| `EUNAVAIL` | Source unreachable; on `ACL` this becomes `VerdictUnknown` |
| any other | Treated as `VerdictUnknown` on `ACL`, as `DidNotRun` on `Act` |

## CONTRACT KIT

Run `go test ./internal/adapters/connectors/<name>/... -run Kit` against a sandbox. All cases must pass before a connector can be named in any manifest. The cases are listed in [03 Connectors](../docs/architecture/03-connectors.md#the-contract-kit).

The kit is `connectorkit.Run(t, sandbox)` (package `internal/core/connectorkit`). A connector's test supplies a `Sandbox`: the connector under test, an existing document, a user who can open it and one who can't, a way to inject a fault into the next `ACL` call (`500`, timeout, `404`), a way to edit a document and to change its permissions, 200 existing refs for the batch case, and, for writable connectors, a valid `Action`. The kit also checks `APIVersion` against `ports.Negotiate`.

## CONNECTORS

### `docs: google-drive` (SB-203)

Read-only. Name it in a manifest as `connectors: { docs: google-drive }`.

| Concern | Behaviour |
|---|---|
| Credentials | A Google service account key (JSON) at `vault://tenants/TENANT/connectors/docs`, with domain-wide delegation for the scope `https://www.googleapis.com/auth/drive.readonly` and nothing else |
| Fetch | `files.get` as the service account. Google Docs are exported as `text/plain`; `text/*` files are downloaded; other types are indexed by name only. `version` is Drive's file `version`; the prefilter is the file's permissions (users' emails, `group:EMAIL`, `domain:DOMAIN`, `anyone`) |
| Location and label | Location is the file's folder path from the top of its drive, then its name: `/Finance/Q3/Forecast`. Folder names come from `files.get` on each parent, cached for the sync. The label comes from one Drive label the tenant names as its classification, at `vault://tenants/TENANT/connectors/docs-classification` as JSON `{"label": ID, "field": ID, "choices": {"CHOICE_ID": "restricted", …}}`, with no secrets in it; `files.get` asks for it with `includeLabels`. A file without it has no label; a choice missing from `choices` is `restricted` |
| ACL | Asks Drive **as the asker** (domain-wide delegation, `sub` = the user's email): `files.get` returns 200 → `Allow`; 404 or 403 → `Deny`, but only if the service account itself can still see the file; anything else, or a deleted file → `Unknown`. At most 8 calls run at once, within the 800 ms deadline |
| Rate limits | `429` and `403 rateLimitExceeded` are retried with backoff while the deadline allows; then `Unknown` |
| Changes | Drive's `changes.list` page token is the cursor. A removed or trashed file is a delete. A change whose `time` differs from the file's `modifiedTime` by more than a second is reported as permission-only |
| Act | `ENOTSUP` (read-only) |

The connector is tested against `drivetest`, a recorded fake of the Drive API, and passes the contract kit there. A run against a real Google Workspace sandbox needs a service account the user provides.

### `chat: slack` (SB-204)

Read-only. Name it as `connectors: { chat: slack }`; `exclude: { chat: ["#hr-*"] }` patterns match channel names.

| Concern | Behaviour |
|---|---|
| Credentials | A Slack app's bot token and signing secret at `vault://tenants/TENANT/connectors/chat` (JSON `{"bot_token", "signing_secret"}`). Scopes: `channels:history`, `groups:history`, `channels:read`, `groups:read`, `users:read`, `users:read.email`. No write scopes, and no `im:` or `mpim:` scopes. The chat app (`/ask`, sb-ask(3)) adds only `commands` |
| Documents | One document per message: id `CHANNEL/TS`, title `#channel`, text the message. Thread replies are their own documents. Version is the edit time, or else the post time, in microseconds, so edits always raise it |
| Direct messages | Never fetched: DMs and group DMs are excluded by default, and the app has no scope to read them |
| Location and label | Location is the channel, `#name`. A message in a **private** channel is labelled `confidential`; a public channel's messages carry no label |
| ACL | Live, per document: a **public** channel → `Allow` for any active, non-guest member of the workspace, and for guests who are in the channel; a **private** channel → `Allow` only for its members. The user is found by email; Slack answering "no such user" → `Deny`. Anything else, or a channel or message that's gone → `Unknown` |
| Changes | Slack has no change feed, so the cell receives the Events API (`message`, `message_changed`, `message_deleted`, `member_joined_channel`, `member_left_channel`, `channel_archive`) at `/connectors/slack/events`, verifies Slack's request signature (v0, at most 5 minutes old), and appends each event to the connector's inbox. The cursor is a position in that inbox. Joins and leaves become permission-only changes for that channel's messages. An empty cursor (first sync, rebuild, and the nightly reconcile) lists every message in every channel the app is in |
| Act | `ENOTSUP` (read-only) |

Tested against `slacktest`, a recorded fake of the Web API and the Events API's signed delivery.

### `code: github` (SB-1103)

Read-only. Name it as `connectors: { code: github }`. It indexes what engineers write about their systems, not their systems: source code only where a space opts in.

| Concern | Behaviour |
|---|---|
| Credentials | A GitHub token with read-only access to the organisation's repositories, metadata, issues and pull requests, and the organisation's name, at `vault://tenants/TENANT/connectors/code` as JSON `{"token": "…", "org": "…"}`. The token is registered with the log redactor before first use |
| Documents | One document per file and per issue or pull request, on each repository's default branch. Files: every `README*`, every file under `docs/`, and every `*.md`; with source opted in, every other text file under 1 MiB too. A file's id is `REPO/blob/PATH`, its title the path, its text the file, and its version the commit time of the last commit that touched it, in microseconds. An issue is `REPO/issues/N` and a pull request `REPO/pulls/N`, titled `#N title`, with the description as text (never the diff), versioned by `updated_at` in microseconds |
| Source code | Off by default. A repository's other files are indexed only when a space's `code` scope names the whole repository, `REPO/**` (sb-space(5)); a narrower scope, such as `REPO/docs/**`, opts nothing in |
| Location and label | Location is `REPO/PATH` for a file, `REPO/issues/N` or `REPO/pulls/N` otherwise, so scope patterns such as `payments/**` hold a repository. GitHub has no sensitivity labels, so none |
| ACL | Live, per repository. The asker is found by email through the organisation's SSO identities (GraphQL `externalIdentities`); none → `Deny`. A **public** repository → `Allow` for any member of the organisation. A **private** or **internal** one → `Allow` when `collaborators/LOGIN/permission` is `read` or higher, through a team or directly; `none` → `Deny`. A repository that's gone, or any other answer → `Unknown`. One permission call per repository per batch, at most 8 at once, within the 800 ms deadline |
| Changes | The cursor is the time of the newest change seen. Issues and pull requests come from `issues?since=`; files from the commits since the cursor and the files each one touched (a removed file is a delete); `MemberEvent` and `PublicEvent` in a repository's events become permission-only changes for all its documents. Only changes strictly after the cursor count, so a restarted sync repeats exactly what it read. An empty cursor lists every document |
| Act | `ENOTSUP` (read-only) |

Tested against `githubtest`, a recorded fake of the REST and GraphQL APIs it uses. A run against a real organisation stays In review.

### `hris: bamboohr` (SB-1104)

Read-only and structured. Name it as `connectors: { hris: bamboohr }`. It is meant for a `people` space, which is `restricted` (sb-departments(7)).

| Concern | Behaviour |
|---|---|
| Credentials | A BambooHR API key of a read-only user and the company's subdomain, at `vault://tenants/TENANT/connectors/hris` as JSON `{"key": "…", "subdomain": "…"}`. The key is registered with the log redactor before first use |
| Documents | One structured document per employee, `employees/ID`, titled with the display name, and one text document per company file shared with all employees (policies, the handbook), `files/ID`, titled with the file's name. An employee's version is BambooHR's `lastChanged`, a file's its `dateCreated`, both in microseconds (a new version of a file is a new file) |
| Fields | Declared and not sensitive: `displayName`, `firstName`, `lastName`, `jobTitle`, `department`, `division`, `location`, `workEmail`, `supervisor`. Declared and sensitive: `payRate`, `payType`, `ssn`, `dateOfBirth`, `address1`, `homePhone`, `mobilePhone`, `maritalStatus`, `gender`, `ethnicity`, `terminationReason`. The connector asks BambooHR only for the non-sensitive fields and the sensitive ones a space's `fields` allow-list names (sb-space(5)), so other sensitive values are never read, let alone ingested (sb-ingest(7), Fields). Anything BambooHR returns that isn't declared is dropped |
| Location and label | Location is `employees/DEPARTMENT/ID` or `files/CATEGORY/NAME`. No source label; an employee record with an allow-listed sensitive field is `restricted` (sb-ingest(7)) |
| ACL | Live: the asker is found by work email in the directory. An **active** employee → `Allow` for employee records and shared files; someone BambooHR doesn't list, or lists as inactive → `Deny`. A file no longer shared with all employees, or a record or file that's gone → `Deny`. Anything else → `Unknown`. Spaces, labels and the allow-list decide who sees what beyond that (sb-spaces(7)) |
| Changes | The cursor holds the newest `lastChanged` seen and the ids of the shared files. `employees/changed?since=` gives inserted and updated employees (changes) and deleted ones (deletes); a file that appears is a change, and one that's gone or no longer shared is a delete. An empty cursor lists every employee and shared file |
| Act | `ENOTSUP` (read-only) |

Tested against `bamboohrtest`, a recorded fake of the API it uses. A run against a real BambooHR account stays In review.

### `accounting: quickbooks-online` (SB-1105)

Read-only and structured. Name it as `connectors: { accounting: quickbooks-online }`. It is meant for a `finance` space, which is `restricted` (sb-departments(7)).

| Concern | Behaviour |
|---|---|
| Credentials | A QuickBooks Online OAuth access token with the `com.intuit.quickbooks.accounting` scope, the company's realm id, and the company's **readers**, at `vault://tenants/TENANT/connectors/accounting` as JSON `{"access_token": "…", "realm_id": "…", "readers": ["…"]}`. The token is registered with the log redactor. Access tokens last an hour; refreshing them from the refresh token is the cell runtime's job, not the connector's (SB-903), and a token that has expired makes every call fail, so every verdict is `Unknown` |
| Documents | The chart of accounts, one structured document per account (`accounts/ID`); one text document per budget (`budgets/ID`), its name, period and one line per account and month; and the profit and loss and the balance sheet for each of the last 12 calendar months (`reports/ProfitAndLoss/YYYY-MM`, `reports/BalanceSheet/YYYY-MM`), one line per row. An account's or budget's version is its `LastUpdatedTime`; a report's is the newest `LastUpdatedTime` of any account, all in microseconds |
| Fields | An account declares `Name`, `AccountType`, `AccountSubType`, `Classification`, `CurrentBalance` and `Description`, none of them sensitive. **Never read:** account and routing numbers (`AcctNum`, `BankNum`), card details, the company's tax ids, and anything from payroll. The connector doesn't ask for them, doesn't declare them, and drops them if QuickBooks sends them, so no allow-list can let them in |
| Location and label | Location is `accounts/CLASSIFICATION/NAME`, `budgets/NAME` or `reports/REPORT/YYYY-MM`. No source label; the finance space's `restricted` default applies |
| ACL | QuickBooks Online doesn't expose who may see its reports, so the tenant lists them: `readers` are the work emails of the company's QuickBooks users with report access, kept in step with QuickBooks by the finance stewards and reviewed with the space (W12). A reader → `Allow`; anyone else → `Deny`; credentials that can't be read, or no `readers` key → `Unknown` |
| Changes | The cursor holds the newest `LastUpdatedTime` seen and a hash of `readers`. Change data capture (`cdc?entities=Account,Budget&changedSince=`) gives changed and deleted accounts and budgets; any account change also changes every report document. A change to `readers` is a permission-only change for every document. An empty cursor lists every account, budget and report |
| Act | `ENOTSUP` (read-only) |

Tested against `qbotest`, a recorded fake of the API it uses. A run against a real QuickBooks Online sandbox company stays In review.

### `tickets: linear` (SB-211)

Read-only in W2. Name it as `connectors: { tickets: linear }`. Writes for the `ticket-update` workflow arrive with SB-410, through the tool bus.

| Concern | Behaviour |
|---|---|
| Credentials | A Linear API key with read access at `vault://tenants/TENANT/connectors/tickets`. With the `ticket-update` workflow enabled, the key also needs `comments:create` |
| Documents | One document per issue: id is Linear's issue id, title `KEY-123 title`, text the description and comments. Version is `updatedAt` in microseconds |
| Act (`ticket-update`) | Target `tickets:<KEY-123 or issue id>`, payload `{"comment": "…"}`. Posts one comment whose id is a UUID derived from the idempotency key, so a retry finds the first comment instead of posting again. The receipt is `linear:comment/<id>`. An answer without an id is read back by that id, and if it isn't there, `Act` fails. `Find` (for the reconciler) looks the comment up by the same id |
| Location and label | Location is the issue's team key, `ENG`. Linear has no labels for sensitivity, so none |
| ACL | Live: the user is found by email (none, or inactive → `Deny`). A **private** team's issues → `Allow` only for team members; a public team's issues → `Allow` for full members, and for guests only if they're in the team. An archived, trashed or missing issue → `Unknown` |
| Changes | Polled from the GraphQL API, so no public endpoint is needed. The cursor holds the last `updatedAt` seen for issues and for team changes. Updated issues are content changes; archived or trashed issues are deletes; a team made private or public, or a membership added or removed, is a permission-only change for every issue in that team. An empty cursor lists every issue |
| Rate limits | HTTP 429 or a `RATELIMITED` error is retried with backoff while the deadline allows; then `Unknown` |
| Act | `ENOTSUP` until SB-410 |

Tested against `lineartest`, a recorded fake of the GraphQL operations the connector sends (by `operationName`).

### `crm: hubspot` (SB-610)

Read-only. Name it as `connectors: { crm: hubspot }`. Contacts are personal data and are never indexed; only companies and deals are.

| Concern | Behaviour |
|---|---|
| Credentials | A HubSpot private app's access token with `crm.objects.companies.read`, `crm.objects.deals.read`, `crm.objects.owners.read` and `settings.users.read`, and the tenant's role map, at `vault://tenants/TENANT/connectors/crm` as JSON `{"token": "…", "roles": {"ROLE_ID": "all" \| "team" \| "owned"}}`. The role map says what each HubSpot role may view for companies and deals, as set in HubSpot's permission sets, because HubSpot's API doesn't expose permission sets |
| Documents | One document per company (`companies/ID`, title the company name) and per deal (`deals/ID`, title the deal name). Text is the record's fixed properties: name, domain, industry, stage, amount, close date and description, one per line. Version is `hs_lastmodifieddate` in microseconds |
| Prefilter | The owner's email and the broad token `hubspot:users`, so every HubSpot user's search includes CRM records and the live check decides. Because the prefilter doesn't depend on roles or teams, role and team changes need no permission-only changes |
| Location and label | Location is the record type, `companies` or `deals`. HubSpot has no sensitivity labels, so none |
| ACL | Live, per record (LAW-01). The user is found by email in the users and owners APIs: none, or a deactivated owner → `Deny`. A super admin, or a role mapped to `all` → `Allow`; `team` → `Allow` if the user owns the record or is in the record's team (`hubspot_team_id`); `owned` → `Allow` only if the user owns it; otherwise `Deny`. A role missing from the role map, or an archived or missing record → `Unknown` |
| Changes | Polled with the CRM search API, so no public endpoint is needed: records with `hs_lastmodifieddate` at or after the cursor, oldest first. Archived records (listed with `archived=true`, by `archivedAt`) are deletes. A record whose `hubspot_owner_assigneddate` is within a second of its `hs_lastmodifieddate`, and whose `createdate` isn't, was just reassigned, and is reported as a permission-only change. An empty cursor lists every record |
| Rate limits | HTTP 429 is retried with backoff while the deadline allows; then `Unknown` |
| Act | `ENOTSUP` (read-only) |

Tested against `hubspottest`, a recorded fake of the CRM v3 objects, search, owners and settings users endpoints the connector calls.

## MOCK CONNECTOR

`internal/adapters/connectors/mock` is a scriptable source system and connector for laws, evals and tests. Nothing in it is random; every behaviour is asked for:

| Ask for | How |
|---|---|
| Allow or Deny | the document's viewers (`Put`, `SetViewers`) |
| Unknown for one user and document | `ScriptVerdict(email, id, VerdictUnknown)` (any verdict can be forced this way) |
| A failing ACL call | `QueueFaults(500, timeout, 404, …)`: one fault per coming call, in order |
| A deleted document | `Delete` (Fetch fails, ACL is Unknown) |
| A target with no receipt | `IssueIDs(false)`: `Act` returns `ENORECEIPT` and changes nothing |
| Read-only | `New(src, false)`: `Act` returns `ENOTSUP` |

It passes the contract kit in both modes. The connector-level LAW-03 and LAW-05 tests run on it.

## LAWS

[LAW-01](sb-laws.7.md#law-01), [LAW-03](sb-laws.7.md#law-03), [LAW-05](sb-laws.7.md#law-05), [LAW-07](sb-laws.7.md#law-07).

## EXAMPLES

A minimal connector skeleton is in [examples/connector/](../examples/connector/README.md).

## SEE ALSO

[sb-permissions(7)](sb-permissions.7.md), [sb-ingest(7)](sb-ingest.7.md), [sb-approvals(7)](sb-approvals.7.md)
