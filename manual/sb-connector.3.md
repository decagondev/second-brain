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

### Scans, images and recordings (SB-6101, SB-6102)

`Fetch` may return **`Images`** and **`Media`**, the parts of the document it couldn't read as text, each `{ name, mime, page, data }`: an image file itself, a PDF's pictures when it has no text layer (as pages 1, 2 …), the pictures under a .docx's `word/media/` or a .pptx's `ppt/media/` (each with the first slide that shows it), and an audio or video file. Ingest has them read by the gated readers and marks what they wrote (sb-ingest(7), What a model read). `docs: folder` and `docs: sharepoint` return them: a picture of at most 10 MB, a recording of at most 200 MB, at most 50 pictures of a PDF.

### Location and label (SB-1005)

`Fetch` also returns where the document lives, its **location**, and the source's own sensitivity **label** if it has one, in the four-step vocabulary of sb-spaces(7). Ingest routes documents to spaces by location (sb-ingest(7), Spaces and labels). A connector whose source has no labels returns none, and never invents one; ingest also reads a Markdown document's front matter `sensitivity:`, which only ever raises the label (SB-5405, sb-ingest(7)). It also returns the document's **owner** at the source, as an email, where the source has one (Drive: the file's owner; a shared drive's files have none), for information barriers (sb-permissions(7), SB-1203). A source without owners returns none. A connector may also return **pages**, where each page of a PDF or slide of a deck starts in the text (SB-5205; sb-ingest(7), Pages and slides); the folder connector does for `.pdf` and `.pptx`.

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
| Fetch | `files.get` as the service account. Google Docs are exported as `text/plain`; `text/*` files are downloaded; other types are indexed by name only. A Google Meet transcript, the Doc Meet names `MEETING (DATE) - Transcript`, keeps each `Speaker: words` line and drops the timestamp lines between them (SB-5306), and is a transcript for search's `type=transcript`, the decisions draft (SB-3702) and its action items (SB-4203), as a `.vtt` file is. Its access is the Doc's permissions, so an attendee without access never sees it. `version` is Drive's file `version`; the prefilter is the file's permissions (users' emails, `group:EMAIL`, `domain:DOMAIN`, `anyone`) |
| Location and label | Location is the file's folder path from the top of its drive, then its name: `/Finance/Q3/Forecast`. Folder names come from `files.get` on each parent, cached for the sync. The label comes from one Drive label the tenant names as its classification, at `vault://tenants/TENANT/connectors/docs-classification` as JSON `{"label": ID, "field": ID, "choices": {"CHOICE_ID": "restricted", …}}`, with no secrets in it; `files.get` asks for it with `includeLabels`. A file without it has no label; a choice missing from `choices` is `restricted` |
| ACL | Asks Drive **as the asker** (domain-wide delegation, `sub` = the user's email): `files.get` returns 200 → `Allow`; 404 or 403 → `Deny`, but only if the service account itself can still see the file; anything else, or a deleted file → `Unknown`. At most 8 calls run at once, within the 800 ms deadline |
| Rate limits | `429` and `403 rateLimitExceeded` are retried with backoff while the deadline allows; then `Unknown` |
| Changes | Drive's `changes.list` page token is the cursor. A removed or trashed file is a delete. A change whose `time` differs from the file's `modifiedTime` by more than a second is reported as permission-only |
| Act | `ENOTSUP` (read-only) |

The connector is tested against `drivetest`, a recorded fake of the Drive API, and passes the contract kit there. A run against a real Google Workspace sandbox needs a service account the user provides.

### `docs: folder` (SB-1606)

Read-only. A directory the cell can read, such as a file share mounted read-only on a self-hosted cell (sb-selfhost(7)), or the example share a demo ships with. Name it as `connectors: { docs: folder }` and give its directory in sb-cell(5) as `connectors.docs.base: "file:///srv/share"`. It needs no credential.

| Concern | Behaviour |
|---|---|
| Documents | Every `.txt`, `.md`, `.pdf`, `.docx`, `.pptx`, `.odt`, `.html`, `.htm`, `.xlsx`, `.ods`, `.csv`, `.vtt` and `.srt` file under the directory, by its path from the top, e.g. `finance/q3-forecast.md`. Names starting with `.`, `.acl` files and other types are never documents. The directory is opened as a root the connector can't leave, so a link that points outside it is never read |
| Fetch | Text and Markdown as they are (a file that isn't UTF-8 is indexed by name only); a PDF's text from its text operators, uncompressed or Flate-compressed, which covers the PDFs office tools write with standard fonts; a PDF with no text it can read, or any file over 8 MiB, is indexed by name only. Office documents and web pages are read as text only, with the standard library (SB-2003): a `.docx` from `word/document.xml`, a `.pptx` from its slides in order (`ppt/slides/slideN.xml`, not the speaker notes), an `.odt` from `content.xml`'s paragraphs and headings, and a page's visible text. Nothing else in an archive is opened, so macros (`vbaProject.bin`, `Basic/`), embedded objects, media and comments are never read; inside the parts that are read, embedded objects, deleted text and an `.odt`'s scripts, forms and annotations are skipped, and in a page, scripts, styles, `noscript`, frames, objects, SVG, comments and the title. An archive listing more than 4096 entries, or whose text parts would decompress past 16 MiB, is skipped whole, never half-read, and so is one that doesn't open or a part that doesn't parse: the file is indexed by name only. A page that isn't UTF-8 is indexed by name only. Spreadsheets are read as their values (SB-2206): each sheet, in the workbook's order, is a line `Sheet: NAME` and then one line a row that holds a value, its cells tab-separated in their columns, with a cell's own tabs and line breaks read as spaces; sheets are separated by a blank line. An `.xlsx` is read from `xl/workbook.xml`, its relationships, `xl/sharedStrings.xml` and its worksheets only; an `.ods` from `content.xml`'s tables, with repeated rows and columns expanded; a `.csv` is one sheet named after the file, UTF-8 (a byte-order mark is dropped), its delimiter the one of `,`, `;` and tab its first line uses most. A formula is never evaluated: a cell shows the value the spreadsheet last saved for it, and a formula's text is never read. Macros, comments, charts, drawings, pivot caches, external links and an `.ods`'s scripts, forms, annotations and embedded objects are never read. A sheet with more than 5,000 rows holding a value is skipped whole, the other sheets kept; a spreadsheet whose text would pass 16 MiB, or one that doesn't open or parse, is indexed by name only. Meeting transcripts (SB-3701), WebVTT (`.vtt`) and SubRip (`.srt`) files in UTF-8, are read cue by cue: each cue is a heading line with its start time, `### 00:12:40`, then a line with its speaker and text, `Ann: We ship on Friday.` (WebVTT's `<v Ann>` voice tag, or a leading `Name:`; no speaker, the text alone). Tags, styles, notes and regions are dropped. A cue with no timing line, a timing that doesn't parse, or no text is skipped, and the transcript ends with the line `N cues couldn't be read.` when any was. The start time is therefore the `section` of every chunk that starts after the first cue (sb-ingest(7), Sections), so a cite of a transcript says where in the meeting it is, and each quoted passage shows its cues' times. Speaker names stay in the text and personal data masking applies as to any document; the brain records no meeting, and reads only the transcripts people put in the share. `version` is the modification time in nanoseconds. The title is the file's name without its extension; the prefilter is the ACL file's readers |
| ACL file | Who may read a file is written beside it in `FILE.acl` (`q3-forecast.md.acl`), or in `.acl` in the same folder for every file there without its own. One rule a line: `read EMAIL`, `read domain:DOMAIN`, `owner EMAIL` (the owner also reads), `label NAME` (the sensitivity label, e.g. `confidential`); `#` starts a comment. A line it doesn't understand makes the whole file unreadable |
| Location and label | Location is the file's folder from the top, then its name: `/finance/q3-forecast`. The label is the ACL file's `label`, none without one |
| ACL | Read live, per document, from the ACL file: a reader listed by email or domain → `Allow`; a readable ACL file that doesn't list the user → `Deny`. A file with **no ACL file** is `Unknown`, never allowed, and so is an ACL file that can't be read or parsed, a file that's gone, or an answer later than the 800 ms deadline |
| Changes | Each sync lists the directory and compares it with the listing the cursor names: a new or modified file is a change, a file whose ACL file alone changed is permission-only, and a file that's gone is a delete. Listings are kept, named by their content, under the cell's state (`state/connectors/docs/`), so a sync restarted from an old cursor reports the same changes; the eight newest are kept. A cursor whose listing is gone is an error, never a fresh start that would miss deletes. An empty cursor lists everything |
| Act | `ENOTSUP`, except notes when the manifest sets `notes: true` (below) |

**Notes (SB-1705).** With `notes: true`, `act` writes one kind of thing, a steward's note, through the tool bus as the tool `note-publish` (sb-agents(7)). The target is `notes/SPACE/qs_ID.md`, where `SPACE` is the space's name or `company`, or `notes/SPACE/updates/qs_ID.md` for a weekly status update (SB-4102; sb-agents(7), The update agent); the payload is `{ "markdown": "…", "readers": [EMAIL…], "owner": EMAIL, "label": LABEL }`. `act` opens `notes/` alone as a root it can't leave, writes the `.md` file and then its ACL file (`read` per reader, the `owner`, the `label`), each created new (a note is never overwritten: the same note again returns the same receipt, a different one at the same path is an error), except that the same note with the same owner and label and **fewer** readers rewrites its ACL file to those readers (SB-2004: a note republished to the space's members now); any other change to the ACL file, a reader added or swapped, is an error, and reads both back. The receipt is `notes/SPACE/qs_ID.md@sha256:HEX`, the SHA-256 of the Markdown as read back, so a note is `Ran` only once it is on the share as written (LAW-05). Nothing else under the share is ever written, and only `.md` and `.acl` files under `notes/`. The next sync finds the note like any other file, so it is indexed with its ACL and cited like any document; a space's `scope` (sb-space(5)) holds its notes when it includes `/notes/SPACE/**`. Because a note's path names its space, a cell with information barriers places the note before it is written, by that path and the spaces whose scope holds it, with the steward as its owner (SB-2001, sb-spaces(7)). On a self-hosted cell `notes/` is the one folder of the share mounted writable, and only into `api` (sb-selfhost(7)).

Tested on a temporary directory, where it passes the contract kit; a fault-injecting view of the directory stands in for an ACL file that can't be read, a slow share and a missing file.

### `chat: slack` (SB-204)

Read-only. Name it as `connectors: { chat: slack }`; `exclude: { chat: ["#hr-*"] }` patterns match channel names.

| Concern | Behaviour |
|---|---|
| Credentials | A Slack app's bot token and signing secret at `vault://tenants/TENANT/connectors/chat` (JSON `{"bot_token", "signing_secret"}`). Scopes: `channels:history`, `groups:history`, `channels:read`, `groups:read`, `users:read`, `users:read.email`. The connector reads no direct messages: no `im:history` or `mpim:` scopes. The chat app (`/ask`, sb-ask(3)) adds `commands`, and notifications (sb-ask(3), Notifications; SB-2101) add `chat:write` and `im:write` to send direct messages, which never carry an answer or a document |
| Documents | One document per message: id `CHANNEL/TS`, title `#channel`, text the message. Thread replies are their own documents. Version is the edit time, or else the post time, in microseconds, so edits always raise it |
| Direct messages | Never fetched: DMs and group DMs are excluded by default, and the app has no scope to read them |
| Location and label | Location is the channel, `#name`. A message in a **private** channel is labelled `confidential`; a public channel's messages carry no label |
| ACL | Live, per document: a **public** channel → `Allow` for any active, non-guest member of the workspace, and for guests who are in the channel; a **private** channel → `Allow` only for its members. The user is found by email; Slack answering "no such user" → `Deny`. Anything else, or a channel or message that's gone → `Unknown` |
| Changes | Slack has no change feed, so the cell receives the Events API (`message`, `message_changed`, `message_deleted`, `member_joined_channel`, `member_left_channel`, `channel_archive`) at `POST /connectors/slack/events` (served by `sb-api`, sb-cell(5)), verifies Slack's request signature (v0, at most 5 minutes old), and appends each event to the connector's inbox. The cursor is a position in that inbox. Joins and leaves become permission-only changes for that channel's messages. An empty cursor (first sync, rebuild, and the nightly reconcile) lists every message in every channel the app is in |
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
| Identity and paging (SB-1401) | The asker is found by the SSO identity whose name is their email; none, several, or only other members' identities is `Unknown`, never someone else's access. Every listing is read page by page, 100 at a time, issues oldest update first |
| ACL | Live, per repository. The asker is found by email through the organisation's SSO identities (GraphQL `externalIdentities`); none → `Deny`. A **public** repository → `Allow` for any member of the organisation. A **private** or **internal** one → `Allow` when `collaborators/LOGIN/permission` is `read` or higher, through a team or directly; `none` → `Deny`. A repository that's gone, or any other answer → `Unknown`. One permission call per repository per batch, at most 8 at once, within the 800 ms deadline |
| Changes | The cursor is the time of the newest change seen. Issues and pull requests come from `issues?since=`; files from the commits since the cursor and the files each one touched (a removed file is a delete); `MemberEvent` and `PublicEvent` in a repository's events become permission-only changes for all its documents. Only changes strictly after the cursor count, so a restarted sync repeats exactly what it read. An empty cursor lists every document |
| Links (SB-3006) | A cite of a file links to it on the repository's default branch, `https://github.com/ORG/REPO/blob/HEAD/PATH#L12-L20`, at the cited lines: the chunk's first and last line in the file, which ingest records for every chunk (sb-ingest(7), Chunks). The default branch, because the version is a time, not a commit, so a line can drift after a later commit; the cite's age says so. With a GitHub Enterprise base URL (`https://HOST/api/v3`), the link is on `https://HOST`. An issue or pull request links to it, with no lines. The link is made from the indexed location and the organisation in the credentials; nothing is fetched to make it |
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
| Documents | One document per issue: id is Linear's issue id, title `KEY-123 title`, text a first line `Labels: a, b` when the issue has labels (SB-4204), a line `Status: NAME`, with ` (closed)` after the name when its workflow state is completed or canceled (SB-4604), then the description and comments. Version is `updatedAt` in microseconds |
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

### `docs: sharepoint` (SB-5901, SB-5902, SB-5906)

SharePoint document libraries and OneDrives through Microsoft Graph, read-only (ADR-0023), with the standard library only: no Microsoft SDK. One Graph client per cell signs in with OAuth client credentials against Entra; its app's `{ "tenant_id", "client_id", "client_secret" }` are at `vault://tenants/TENANT/msgraph`, read at call time and registered with the log redactor with each token. The application permissions it needs are read scopes only: `Sites.Read.All` (or `Sites.Selected`), `Files.Read.All`, `Group.Read.All`, `User.Read.All` and `OnlineMeetingTranscript.Read.All`. sb-cell(5) `microsoft` names the drives, the organizers whose Teams transcripts are read, and the company's email domain.

| Concern | Behaviour |
|---|---|
| Changes | `GET /drives/ID/root/delta` per drive with `Prefer: deltashowsharingchanges`, from the delta link the cursor keeps (one per drive and organizer, as JSON); an empty cursor lists everything. A deleted item is a delete; an item whose sharing changed is read again whole (`ACLOnly`, which re-ingests). A delta link Graph no longer takes (410) starts that drive over |
| Fetch | The item, its content (at most 25 MB) through the Office readers by extension (sb-ingest(7): `.docx`, `.pptx`, `.xlsx`, `.pdf`, `.vtt`, text), with each page or slide's start (SB-5205) and an Office file's review-by property. The version is the number after the last comma of its `cTag`; the location is its folder under the drive's root, then its name (`/Finance/close.docx`); the owner is `createdBy.user.email` |
| ACL copy | `GET …/items/ID/permissions`: each granted user's email, each group as `group:MAIL` (or `group:ID`), an `anonymous` link as `anyone` and an `organization` link as `domain:DOMAIN`, both of which are wide (sb-permissions(7), Shared too widely) |
| Live check | The item's permissions read again now and the person's groups from Entra with nested membership (sb-identity(7), Entra ID), at most 8 at a time within 800 ms: a grant to their email, to one of their groups, an `anyone` link, or an `organization` link for an address in the domain allows; anything else denies; an item that's gone, an error or a late answer is Unknown (LAW-03) |
| Purview labels (SB-5902) | Each item's `sensitivityLabel.labelId` is mapped by sb-cell(5) `labels.microsoft` to public, internal, confidential or restricted. An id it doesn't map counts as **restricted** until an admin maps it, and the health report's `unmapped_labels` lists such ids with how many items carried each, never which. A new mapping relabels at the next sync |
| Teams transcripts (SB-5906) | For each organizer, `GET /users/ID/onlineMeetings/getAllTranscripts(meetingOrganizerUserId='ID')/delta`; each transcript is a document titled `SUBJECT (DATE) - Transcript`, of type `transcript` (SB-3701), its WebVTT read with speakers and times (`folder.TranscriptText`), under `/Teams meetings/`, open to the meeting's organizer and attendees, who could open its recording; the live check reads the meeting's participants again. A meeting without a transcript is never listed. Its text is data (LAW-09) |
| Act | `ENOTSUP`: read-only |

Tested against `graphtest`, a recorded fake of the Graph endpoints and the token endpoint, with the contract kit and the connector laws (`TestLAW01_SharePointAccessIsLive`, `TestLAW03_SharePointUnknownIsNeverAllow`). A run against a real tenant is In review until one is provided.

### `docs: confluence` (SB-6201)

Confluence Cloud, read-only, over its REST API with the standard library (ADR-0024). It reads the pages and blog posts of the spaces sb-cell(5) `atlassian.spaces` names. **Changes** lists every item of each space with its version (cursor: each item's version), so a new or newer version is a change and an item no longer listed is gone; the order is fixed, so a first sync resumes (SB-6004). **Fetch** reads the page in its storage format through the HTML reader, puts its labels on a `Labels:` line first, and sets its location to `/SPACEKEY/TITLE`. Its **ACL copy** is the page's read restrictions: the people (by email) and `group:NAME` for each group; a page with none is `confluence:users` (every Confluence user, the live check decides), or `anyone` in a space open to anonymous users, which makes it wide (SB-5401). **ACL** asks Confluence's own permission check (`POST /wiki/rest/api/content/ID/permission/check`) for the person's account, found by email (sb-identity(7), Atlassian groups); no account is Deny, anything else that fails, or a late answer, is Unknown (LAW-03). A restriction changed without a new version is in the live check at once and in the ACL copy at the page's next version. **Act** is `ENOTSUP`. Refs are `page:ID` and `blog:ID`.

### `tickets: jira` (SB-6204)

Jira Cloud, read-only (ADR-0024). It reads the issues of the projects sb-cell(5) `atlassian.projects` names. **Changes** lists every issue with its updated time, in key order; a changed one is a change and one no longer listed is gone. **Fetch** reads the summary, description and comments (Atlassian Document Format as text), with a `Labels:` line and a `Status:` line first, `(closed)` when its status is in the done category, as Linear's are (SB-4204, SB-4604), so incident briefs and reviews start on Jira as on Linear. The title is `KEY SUMMARY`, the location the project key, and the version the updated time. Its **ACL copy** is the people Jira says may browse it (`GET /rest/api/3/user/permission/search`, which follows permission schemes and issue security levels), or `jira:users` when that is more than 1,000 and it has no security level. **ACL** asks the same search for the person's email, issue by issue; a deleted issue is Deny, anything else that fails or is late Unknown. **Act** is `ENOTSUP`: the brain files nothing (SB-6207 drafts an issue the person files).

A verified Jira webhook (`POST /hooks/jira`, signed with `webhook_secret` as `X-Hub-Signature: sha256=…`) starts a sync of `tickets` at once, so `ticket-filed`, incident briefs and reviews don't wait for the next minute (SB-6205); an unsigned or wrongly signed delivery is refused, and its body is never read as anything but a signal.

### Calendars (SB-4202)

A calendar isn't a source: nothing in it is indexed, answered from or kept. It is read only for meeting prep (sb-ask(3), Meeting prep), as the person whose calendar it is, and only when that person asks for a brief or has turned on `prep`. It is set in sb-cell(5) `calendar`, not in the tenant's `connectors`.

| Concern | Behaviour |
|---|---|
| Credentials | A Google service account key at `vault://tenants/TENANT/calendar`, with domain-wide delegation for `https://www.googleapis.com/auth/calendar.events.readonly` only. Each call takes a token for the person, by their directory email, as its subject, so it reads exactly what that person's own calendar shows them |
| Events | `GET /calendar/v3/calendars/primary/events` with `singleEvents=true`, `orderBy=startTime` and the window asked, at most 50: the event's id, summary as its title, description as its agenda, start time, and the attendees' emails. Cancelled events, and events the person declined, are left out. The description is data (LAW-09) and is never logged |
| Errors | A person the delegation doesn't cover, or a 401 or 403, is no events and a count (`sb_calendar_failed_total`), never another person's calendar. HTTP 429 is retried with backoff while the deadline allows |
| Act | None: the calendar has no write path |

Tested against `caltest`, a recorded fake of the events endpoint and the token endpoint.

**Outlook** (SB-5905). With sb-cell(5) `calendar.kind: outlook`, the calendar is read through Microsoft Graph with the cell's Graph client (`docs: sharepoint`, above; `Calendars.Read`): `GET /users/EMAIL/calendarView` for the window asked, at most 50, the person's own address only, never anyone else's. Each event's subject is its title, its body preview its agenda, and its attendees' addresses are matched to people through the directory (sb-identity(7), Entra ID). Cancelled events and those the person declined are left out, and a person Graph doesn't know reads no events. Tested against `graphtest`.

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
