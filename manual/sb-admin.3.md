---
title: "sb-admin(3)"
man: "3"
summary: Admin API used by the console and by automation.
---

# sb-admin(3)

## NAME

sb-admin — manifest diffs, kill switches, audit and usage

## SYNOPSIS

```http
GET  /v1/admin/health
GET  /v1/admin/feedback?rating=down[&reviewed=true]
POST /v1/admin/feedback/TRACE/reviewed
PUT  /v1/admin/notice
DELETE /v1/admin/notice
GET  /v1/admin/manifest
POST /v1/admin/manifest/diff
POST /v1/admin/kill
POST /v1/admin/unkill
GET  /v1/admin/kill
GET  /v1/admin/audit?since=…&kind=…&space=…&actor=…&from=…&to=…
GET  /v1/admin/usage?week=2027-W10
GET  /v1/admin/refusals?since=…
POST /v1/admin/subjects/erase
POST /v1/admin/subjects/export
GET  /v1/admin/retention/plan?days=N
GET  /v1/admin/index/scopes
GET  /v1/admin/index/leftovers
POST /v1/admin/index/refresh
POST /v1/admin/index/prune
POST /v1/admin/sessions/revoke
GET  /v1/admin/agents
GET  /v1/admin/agents/value
GET  /v1/admin/agents/NAME/record
GET  /v1/agents/shadow                          (an agent's owners)
GET  /v1/admin/tokens
DELETE /v1/admin/tokens/ID
POST /v1/admin/notifications/test
POST /v1/spaces/NAME/proposals                 (stewards)
POST /v1/spaces/NAME/proposals/term            (stewards)
GET  /v1/spaces/NAME/agents                    (stewards)
POST /v1/spaces/NAME/proposals/agents          (stewards)
POST /v1/spaces/NAME/agents/AGENT/pause        (stewards, the agent's owner)
POST /v1/spaces/NAME/agents/AGENT/resume       (stewards, the agent's owner)
GET  /v1/spaces/NAME/service                   (stewards)
GET  /v1/admin/proposals
POST /v1/admin/proposals/ID/approve
POST /v1/admin/proposals/ID/reject
GET  /v1/admin/costs?month=2027-03[&format=csv]
GET  /v1/admin/costs/spaces?month=2027-03
GET  /v1/admin/guardrails
GET  /v1/admin/tree/plan?op=move|merge&space=NAME&to=NAME
GET  /v1/spaces/NAME/reviews                   (stewards)
POST /v1/spaces/NAME/reviews/ID/decide         (stewards)
GET  /v1/admin/reviews
GET  /v1/admin/reviews/ID/evidence
POST /v1/holds                                (Legal stewards)
POST /v1/holds/ID/release                     (Legal stewards)
GET  /v1/holds                                (Legal stewards, operators)
```

## DESCRIPTION

The console (`/?admin`) is a client of this API and has no other powers. It opens on the cell's health, overdue jobs and stale connectors first (SB-1607), with the machine's checks on a self-hosted cell (SB-1802), then the queue of answers people marked unhelpful (SB-1609). Beside the kill switches and the applied manifest it has the open steward proposals, each with its diff and its hash, approved only by sending back the hash shown or rejected with a reason, and every access review with its state, `overdue` and a link to its evidence (SB-1509; the stewards' own screen is in sb-spaces(7)). Every endpoint needs the `operator` role (Cedar policy) except `usage` and `costs`, which `approver` may also read.

**`GET /v1/admin/health`** is the cell's health from its own records (SB-1607): `{ "jobs": [{ "name", "law", "last_finish", "overdue", "unreadable" }], "connectors": [{ "kind", "last_success", "last_failure", "failures", "stale" }], "at" }`. A job is overdue when it hasn't finished within its grace (sb-runtime(7), Schedules), or has never finished, or its record can't be read (`unreadable`), exactly as the watchdog decides. A connector is stale when its last successful sync is older than the 15-minute freshness target or it has none; `failures` counts the syncs that failed since its last success. A connector whose last sync failed adds `failure` (SB-3201), the class of that failure: `auth` (the source answered 401 or 403, or said the credentials are revoked), `rate_limited` (429), `unreachable` (no answer: DNS, TLS, a refused connection or a timeout) or `source_error` (a 5xx, or any failure the cell can't place, which is never treated as fine), and `fix`, the runbook section for it (`docs/runbooks/connectors.md#auth` and so on). The class is all the record keeps: never a URL, a body, a token or a message from the source (LAW-07). A success clears it. Overdue jobs and stale connectors come first. It reads only finish times and counts, never content, and a store it can't read answers `EUNAVAIL`, never a healthy report. On a self-hosted cell it adds **`box`** (SB-1802): `{ "facts_at", "verdict", "checks": [{ "check", "verdict", "detail", "fix" }] }`, the machine's checks judged exactly as `install.sh --doctor` judges them (sb-selfhost(7), Doctor), worst first, from the facts the hourly host step leaves in `state/host/facts.json`. A verdict is `pass`, `warn`, `fail` or `unknown`; no facts, facts that don't parse, or facts older than two hours are `unknown`, never green. A managed cell has no `box`, and the console says so. A cell with a channel to notify through adds **`notify`** (SB-2607): `{ "last_run", "via", "sent", "skipped", "failed", "warning", "reason" }`, the notify job's last run, counts only, never who; `warning` is true when that run failed, or no run finished in the last hour, and `reason` says which. The console shows it under Health, and the weekly report has the same as one line.

**`GET /v1/admin/feedback?rating=down`** is the queue of answers people marked unhelpful (SB-1609), newest first, the last 90 days: `{ "items": [{ "trace", "at", "user", "release", "rating", "wrong_citation", "comment", "claims", "uncited", "refused", "decisions": [{ "verdict", "reason", "count" }], "reviewed": { "by", "at" } }] }`. Each item carries the trace's counts, its recorded retrieval decisions counted by verdict and reason (`allow`/`allowed`, `refuse`/`source_denied` and so on), and the user's own comment, which they wrote to be read; it never carries the question, the answer text or the cites, so it shows nothing of a document the operator couldn't open. A trace that has expired keeps its counts and has no decisions. Reviewed items are left out unless `reviewed=true`. `rating` is `down` or `up`; anything else is `EINVAL`.

**`POST /v1/admin/feedback/TRACE/reviewed`** marks an item reviewed by the operator who sends it, and records that in the audit log (`config` row, subject `feedback:TRACE`). Marking an item twice keeps the first review. A trace with no feedback is `EINVAL`.

**`PUT /v1/admin/notice`** `{ "text": "…", "until": "2027-03-12T18:00:00Z" }` (SB-3207) sets the cell's one notice to everyone: one line of plain text, at most 200 characters, with no link or markup (no `<`, `>`, `[`, `]`, `` ` ``, or `://`), ending at `until`, which must be in the future and at most 7 days away; anything else is `EINVAL`. It replaces any notice there was. **`DELETE /v1/admin/notice`** clears it. Each writes a `kind: config` audit row (subject `notice`, state `set` or `cleared`, with the SHA-256 of the text, never the text). The notice is one record in the cell's store and ends by itself at `until`: nothing needs to clear it. **`GET /v1/notice`**, for anyone signed in, answers `{ "text", "until" }` while it stands and `{}` after. The web UI shows it at the top of every screen, and the chat apps put it first in every answer (sb-ask(3)). Admins only, like the rest of this page.

**`GET /v1/admin/manifest`** returns the applied manifest and its git commit.

**`POST /v1/admin/manifest/diff`** takes a JSON patch, e.g. `[{ "op": "add", "path": "/agents/-", "value": "draft-reply" }]`, validates it as `sb tenant check` would, and returns the YAML diff. Depending on tenant setup, the diff is either opened as a pull request against the tenant's manifest repo or signed and applied. Either way, the result is a manifest change (LAW-16).

**`POST /v1/admin/kill`** takes `{ "scope": "agent"|"connector"|"tenant", "name": "…", "reason": "…" }` and has the same semantics as [sb-kill(1)](sb-kill.1.md).

**`GET /v1/admin/audit`** pages through [sb-audit(5)](sb-audit.5.md) rows. `space=NAME` keeps the rows written in that space and the `kill` rows about it (SB-1008). `actor=` keeps one person's rows: a user id, or an email the directory gives exactly one active user, whose id is then matched; an email it can't place is `EINVAL`, never every row (SB-3204). `from=` and `to=` (dates, `YYYY-MM-DD`, UTC, both inclusive) keep the rows written on those days. Every filter only narrows the 500 rows read from `since`, so paging works as before: `next` is the row to read from next whether or not any row on this page matched.

**`GET /v1/admin/usage?week=2027-W10`** returns the weekly report's numbers (sb-observability(7), Weekly leadership report): weekly active users, seats, asks, cited-answer rate, gaps by kind, drafts and their used-to-discarded ratio, actions by outcome and model spend.

**`GET /v1/admin/refusals`** lists refusal counts by connector and reason (`deny`, `unknown`, `timeout`, `missing`), with no document titles, so operators can spot a misbehaving connector.

**`POST /v1/admin/subjects/erase`** takes `{ "user": "usr_…", "reason": "…" }` and erases the user's derived data as sb-tenant(5), Retention and deletion, describes. The response is `{ "erased": { "traces": 3, "conversations": 1, "feedback": 1, "logs": 12 }, "audit": ["aud_…", "aud_…"] }`. A user id that isn't `usr_` followed by `[a-z0-9_]` is `EINVAL`, and a missing reason is `EINVAL`. Erasing a user with nothing stored succeeds with zero counts.

**`POST /v1/admin/subjects/export`** (SB-1903) takes the same `{ "user": "usr_…", "reason": "…" }` and returns a copy of everything erase would remove for that person, by the same selection in each store: `{ "user", "records": { "traces": [ … ], "conversations": [ … ], "feedback": [ … ], "saved questions": [ … ], "shares": [ … ], "steward questions": [ … ], "logs": [ … ] }, "sha256", "audit": ["aud_…", "aud_…"] }`. Each record is the store's own JSON (a log line that isn't JSON is a JSON string); steward questions carry their replies. It is an operator action like erase: it writes a `kind: retention` audit row (subject `export:USER`, state `requested`, the reason) before reading anything, and a second row after whose reason holds the count per store and `sha256=` the SHA-256 of `records` as returned, never the text (LAW-07). If the first row can't be written nothing is read; if a store can't be read the second row says `failed` and the answer is `EUNAVAIL` with no records, because a partial copy is never handed over as the whole. An export after an erase is empty. The same `EINVAL`s as erase.

**`GET /v1/admin/retention/plan?days=N`** (SB-1905) is what the nightly retention job (sb-tenant(5), Retention and deletion) would delete if it ran `N` days from now (default 0, at most 3650), deleting nothing: `{ "days", "cutoff", "would_delete": { "traces": 120, "conversations": 40, … }, "held": 7 }`. It runs the job's own selection in each store, the tenant's cutoff and then each shorter space's, so the counts are what that run deletes if nothing new arrives; records a legal hold keeps are counted apart in `held`, not in `would_delete`. A store that can't be read is `EUNAVAIL` and names it. `days` outside 0 to 3650 is `EINVAL`. It writes no audit row: it changes nothing and returns only counts. `sb tenant retention --plan` prints it (sb-tenant(1)).

**`GET /v1/admin/index/scopes`** (SB-2903) counts, for each space in the cell and each of its scope patterns, the documents in the index whose location the pattern matches now (each document once, at its newest version): `{ "patterns": [{ "space": "ops", "kind": "docs", "pattern": "/Ops/**", "matches": 120, "of": 900, "problem": "" }] }`, where `of` is how many documents that kind has and `problem` is `none` for a pattern matching nothing or `too_wide` for one matching more than half of `of`. Only locations are read, never text; it changes nothing and writes no audit row. An index that can't be read is `EUNAVAIL`. `sb space scope-report` prints it (sb-space(1)).

**`GET /v1/admin/index/leftovers`** (SB-2905) lists the index's **leftovers**, each once by cite: `{ "gone": ["docs:…@3"], "unheld": ["docs:…@1"] }`. `gone` are the chunks of documents their connector no longer lists (a full listing from the connector, as a rebuild reads it), and `unheld`, in a cell with spaces, the chunks no space holds. A connector that can't be listed is `EUNAVAIL`, never an empty list, so nothing a source still lists is ever called gone. Cites only, never text. **`POST /v1/admin/index/prune`** `{ "reason": "…" }` lists them again and removes the `gone` documents from the index, with a `kind: config` audit row before (subject `index:prune`, state `requested`, the reason) and one after (`pruned`, `gone=N`), and answers `{ "pruned": N }`; a missing reason is `EINVAL`, and if the first row can't be written nothing is removed. Nothing a source still lists is touched, so `unheld` chunks stay: they are a scope to fix in the manifest (`sb space scope-report`), and a company-wide ask may still use one. A rebuild removes leftovers too (LAW-13: the index is derived, so removing one loses nothing); prune only saves waiting for it. `sb index prune [--dry-run] --reason TEXT` calls them (sb(1)).

**`POST /v1/admin/index/refresh`** `{ "cite": "docs:…", "reason": "…" }` (SB-3202) re-reads one document from its source now, with the same pipeline and checks a sync uses, instead of waiting for the next change. The cite's version, if any, is ignored: the source's current version is what is read. A document the source says is gone (404, 410, or a missing file) is taken out of the index as a sync would take it out; a fetch that fails any other way changes nothing and is `EUNAVAIL`. It writes a `kind: config` audit row before (subject the document, state `requested`, the reason) and one after (`refreshed` or `removed`), and answers `{ "document": "docs:…", "version": N, "changed": true|false, "removed": false }`. Admins only; a missing reason is `EINVAL`, a kind the cell has no connector for is `EINVAL`. `sb index refresh CITE --reason TEXT` calls it (sb(1)).

The nightly `retention` job counts the leftovers and keeps the counts, and **`GET /v1/admin/health`** carries `"index": { "at", "gone", "unheld", "warning" }`, `warning` true above zero or when the count is missing or older than two days. It also carries `"prompt": { "p95", "p95_before", "warning" }` (SB-2906): the 95th percentile of the input tokens of each model call in the last seven days and the seven before, from the usage ledger, with `warning` true when this week's is over 1.5 times the week before's. Counts only. It also carries `"answer_time": { "p95_seconds", "p95_seconds_before", "warning" }` (SB-3205), the same for the seconds each answer took, from the duration the usage ledger records for each answer; `warning` is true when this week's is over 1.5 times the week before's, and absent when the ledger has no durations.

**Spaces with no steward left (SB-3102).** Health also carries `"spaces_unstewarded": { "spaces": ["finance"], "unknown": ["legal"], "warning": true }`: the spaces whose steward groups have no active member in the directory now (`spaces`), and those the directory couldn't answer for (`unknown`), read at the moment health is asked. Either list warns: a directory that can't answer is never "has a steward" (LAW-03). The console shows it under Health. A deprovisioned steward stops counting at once, as everywhere (LAW-12). Questions sent to such a space go to the company-wide answerers (sb-spaces(7), Stewards).

**`GET /v1/admin/agents`** (SB-4004) is every agent in one place: `{ "agents": [{ "name", "department", "spaces": [], "runs", "rated", "unhelpful", "eval": "passed" | "held", "suite", "killed", "owner", "owner_inactive"? }], "days": 30 }`, in name order. It lists each agent the manifest's `agents` enables, and `handover`, which every cell has. `department` is the template a department agent belongs to (sb-departments(7)), left out for an agent that runs in every space, and `spaces` are the spaces that can run it. `runs` counts its answers in the last 30 days from the traces, and `rated` and `unhelpful` the feedback on those answers and how much of it was a thumbs down; searches, eval runs and refused rate limits are no agent's run. `eval` is `passed` because a release runs only after its eval gate passed, or `held` while a tenant eval case holds the release (sb-evals(7)); `suite` is the agent's own suite, `agent-NAME`, or `gate` for the whole gate. A hold that can't be read is `EUNAVAIL`, never `passed`. `killed` is the agent's kill switch now (sb-kill(8)). `owner` is who answers for it, from its definition (sb-agents(7), Definitions), and `owner_inactive: true` flags an owner with no active person now (SB-4402). Counts only: never a question, an answer or who asked. The admin console's **Agents** shows the table.

**`GET /v1/admin/agents/value`** (SB-4701) is each agent's drafts over 12 weeks, per agent and space, published, edited, discarded, rounds of changes, hours to publish, helpful rate and cost, with each agent's spend this month against its cap (sb-agents(7), Value). An agent's owner gets the same for the agents they own at `GET /v1/agents/value`. Counts only. The admin console's **Agents** shows it under the registry. Each registry row also carries `paused`, the spaces the agent is paused in (SB-4702).

**`GET /v1/admin/agents/NAME/record`** (SB-4707) is the agent's signed record, `{ "record", "sha256", "signed_at", "sig", "key" }`, with `?format=md` its Markdown (sb-agents(7), An agent's record); an agent the bundle doesn't define is `EINVAL`. The registry's **Download record** saves both. **`GET /v1/agents/shadow`** (SB-4704) is the open shadow windows of the agents the caller owns (sb-agents(7), Shadow runs).

**`GET /v1/spaces/NAME/agents`** (SB-4004) is the same for a space's stewards: the rows of the agents that space can run, counted in that space only, with `listed`, the department agents its `agents` lists, and `can_list`, the ones its template could list. A listed department agent with no run there in 60 days is `"unused": true`, and `propose_off` is the `agents` list without them, for **Propose turning off**; with `retention_days` under 60 neither is shown (SB-4402). **`POST /v1/spaces/NAME/proposals/agents`** `{ "agents": ["incident-summary"] }` is a steward proposal of the space as applied with that `agents` list, checked, recorded and audited as any proposal, so an operator approves the exact manifest diff. The steward screen's **Agents** has the table and **Propose these agents**.

**`GET /v1/admin/tokens`** (SB-3803) lists every live read-only API token of the tenant (sb-api(3), Read-only API tokens): `{ "items": [{ "id": "tok_…", "owner": "usr_…", "name", "created", "last_used": "YYYY-MM-DD", "expires" }] }`, oldest unused first: the ones never used, then by the day they were last used. `last_used` is the day the token was last checked against the directory on a use, which happens at most once a minute; it is left out for a token never used. Never the token, and never its hash. **`DELETE /v1/admin/tokens/ID`** `{ "reason": "…" }` revokes one at once: a `kind: access` audit row (subject `token:tok_…`, state `token_revoked`, the reason, the admin as actor) is written first, and if it can't be, nothing is revoked (`EUNAVAIL`). A missing reason, or no such live token, is `EINVAL`. The admin console's **API tokens** lists them with **Revoke**.

**`POST /v1/admin/sessions/revoke`** (SB-1906) takes `{ "user": "usr_…", "reason": "…" }` and ends every session of that person: `{ "user", "revoked": 2, "audit": ["aud_…", "aud_…"] }`. It writes a `kind: access` audit row (subject `sessions:USER`, state `revoke_requested`, the reason) before any session ends, and a second (`revoked`, `sessions=N`) after; if the first can't be written nothing is revoked (`EUNAVAIL`). A person with no session succeeds with `revoked: 0`. A missing reason, or a user that isn't `usr_…`, is `EINVAL`. It doesn't deactivate the person: they can sign in again (sb-identity(7), Signing out everywhere).

**`POST /v1/admin/notifications/test`** (SB-2602) takes `{ "to": "ann@…" }` and sends that person the fixed test notification through the cell's real channel, as `POST /v1/me/notifications/test` does for oneself (sb-ask(3), Notifications): `{ "sent": true, "via": "mattermost" }`, with `sent: false` when the channel doesn't know them. The address must be one active directory user's (`EINVAL` otherwise), and an address can be tested once a minute (`ERATE`). `sb selfhost check-notify --to EMAIL` calls it (sb-selfhost(7), Checking the settings).

**`GET /v1/admin/costs?month=2027-03`** returns the month's cost statement (sb-tenant(5), Model billing and cost statements): `{ "tenant", "month", "partial", "lines": [{ "provider", "model", "tokens_in", "tokens_out", "price_in_per_m", "price_out_per_m", "cost_usd", "tier"? }], "total_usd", "budget_usd_month", "forecast_usd", "forecast"? }`. For the month still running, `forecast_usd` is the month's spend so far plus the last 7 days' daily rate (fewer when the month is younger) times the days left, to the cent; with fewer than 3 days of the month gone it is `null` with `"forecast": "not enough data yet"`, and for a past month it is `null` (SB-3505). The model's budget guard checks the same forecast at most once an hour, and a forecast over `budget_usd_month` sends the month's one `budget_warning` early (sb-observability(7)). With `&format=csv` it is `text/csv` with the same columns, one row per line. A month that isn't `YYYY-MM` is `EINVAL`. Operators and approvers may read it.

**`GET /v1/admin/costs/spaces?month=2027-03`** (SB-4105) is the month's spend by space, each with its own spend, its branch's (with its teams'), its `budget_usd_month` and, for the running month, a forecast (sb-observability(7), Spend by space). Readable by the same people as `costs`; amounts only. The console shows it as **Spend by space**.

**`GET /v1/spaces/NAME/service`** (SB-4507), for a space's stewards, is 12 weeks of the space's service levels by category and week: time to first reply, time to close and the share closed late, and each team's counts, every row under 5 shown only as `fewer_than_5` (sb-ask(3), Department service levels). Counts and medians only: never a question, a request's text or who sent it. The space's home shows it as **Service**.

**`GET /v1/admin/tree/plan?op=move&space=support&to=customer`** (SB-4806) is the reorg dry run of `sb tree move` and `sb tree merge` (sb-space(1), tree) against the cell's applied manifest, for admins: `{ "plan": { "op", "space", "to", "spaces": [{ "space", "parent_before", "parent_after", "escalate_now", "escalate_after", "rolls_up_now", "rolls_up_after", "settings": [{ "setting", "gains", "loses" }], "gone"? }] }, "manifest": "…", "in_flight": { "SPACE": { "rhythms", "open", "waiting", "projects", "routines", "triggered": [] } } }`. `op=merge` folds `space` into `to`. `manifest` is the manifest the change would leave, to propose through the usual review; nothing is applied. `in_flight` counts what is under way in each affected space now: running rhythms, open queue items and those waiting on another space, open projects, routines asked in the space, and the agents with a schedule or trigger there. Counts only. A change that would widen access (a label that loosens, a retention that grows, a merge whose member groups differ) is `EPERM` with the setting named (LAW-24); an unknown space, or a result `sb tenant check` refuses, is `EINVAL`.

**`GET /v1/admin/guardrails`** (SB-4406), for operators, returns 12 weeks of counts, newest first (sb-observability(7), Guardrail insights): `{ "weeks": [{ "week": "2027-W10", "flags": [{ "source", "space", "count" }], "flagged_uploads", "refused_tools", "denied_tools", "dropped_uncited" }], "doubled": ["SOURCE"…] }`. Each count is a number, or `"fewer than 5"` when it is 1 to 4. No chunk text, title, question or person appears.

### Roles, exactly (SB-603)

| Role | Who | May |
|---|---|---|
| `operator` | members of the tenant's IdP group `sb-operators` | every admin endpoint |
| `approver` | members of any workflow's approval group in the manifest | `usage`, and approving actions (sb-approvals(7)) |
| everyone else | | none of these endpoints |

Roles are Cedar policies, not code: the base policy permits `admin` on the tenant's own `Cell` only for a principal in `Group::"sb-operators"`, with the tenant condition every permit carries (sb-policy(5)). A tenant policy can narrow it but not widen it. Groups come from the cell's own directory, so an operator of one tenant has no role in another (LAW-02).

### The manifest diff, exactly (SB-602)

`POST /v1/admin/manifest/diff` accepts JSON Patch operations (`add`, `remove`, `replace`) on these paths only: `/agents`, `/workflows`, `/connectors/<kind>`, `/approval/<workflow>`, `/exclude/<kind>`, `/retention_days`, `/budget_usd_month`, `/pilot_group`, `/channel`. `tenant`, `region` and `idp` can't be patched here. The patched manifest is checked exactly as `sb tenant check` checks a file. The response is `{ "manifest": "<the whole new manifest as YAML>", "diff": ["- agents: [ask]", "+ agents: [ask, draft-reply]"], "plan": "<the sb tenant apply --plan output>" }`. It **changes nothing**: every proposal writes a `kind: config` audit row with the operator and the patch's hash, and the change takes effect only when the new manifest is applied. Kill switches offer **make permanent**, which is this same endpoint with a patch that removes the agent or connector.

### Proposals (SB-1201)

**`POST /v1/spaces/NAME/proposals`** is for the space's stewards (Cedar `propose_space`; sb-spaces(7), Delegated changes). The body is the space's whole new definition, `{ "space": { "members": [...], "scope": {...}, ... } }`, in the fields of sb-space(5). The response is `{ "id": "prop_…", "space": "NAME", "base": "<sha256 of the applied manifest>", "hash": "<sha256 of the proposed manifest>", "diff": [...], "plan": "…" }`, and nothing changes yet. Every diff in this page writes a nested line with its full path, such as `- spaces.finance.label: restricted`, so a change in one space can't hide behind the same text in another (SB-1401).

**`GET /v1/admin/proposals`** lists the open proposals. **`POST /v1/admin/proposals/ID/approve`** takes `{ "hash": "…" }` and needs the `operator` role (Cedar `approve_space`, forbidden for the proposal's own steward). It returns `{ "id", "manifest": "<the YAML to apply>", "approved": true }`. A hash that isn't the proposal's is `EINVAL`; an applied manifest that changed since the proposal is `EDRIFT`, and the proposal is closed as `stale` with its own audit row. **`…/reject`** takes `{ "reason": "…" }`. Every proposal, approval, rejection and closing is a `kind: config` audit row with the space, written before it is stored, and a proposal takes one decision only: a second decision made at the same time finds it no longer open (SB-1401).

### Access reviews (SB-1202)

**`GET /v1/spaces/NAME/reviews`** lists the space's reviews for its stewards (Cedar `review_space`; sb-spaces(7), Access reviews). **`POST /v1/spaces/NAME/reviews/ID/decide`** takes `{ "user": "usr_…", "decision": "certify" | "remove", "reason": "…" }`; a removal needs a reason, and deciding on your own access is `EPERM`. The response is the review; when the decision completes it, the response's `requests` hold the change requests, and a manifest one carries its proposal id for `…/proposals/ID/approve`.

An **agent review** (SB-4705; sb-agents(7), Agent access reviews) is a review with `kind: "agents"`: each item's `user` is an agent or trigger (`case-brief`, `brief:case-brief`, `updates`, `leader_week`), with `via`, `last_use`, `unused` and `tools`, and the review carries `propose_off`; it is decided on the same route, and its removals become one manifest proposal.

**`GET /v1/admin/reviews`** lists every review with `overdue` for the operator. **`GET /v1/admin/reviews/ID/evidence`** returns `{ "review": {…}, "sha256": "…" }`, the audit-ready evidence.

### Legal holds (SB-1207)

**`POST /v1/holds`** takes `{ "matter": "…", "space": "NAME", "users": ["usr_…"], "reason": "…" }` with a space, users, or both, and needs a Legal steward (Cedar `place_hold`; sb-spaces(7), Legal holds). It returns the hold, `{ "id": "hold_…", "matter", "space", "users", "placed_by", "placed_at" }`. **`POST /v1/holds/ID/release`** takes `{ "reason": "…" }`. **`GET /v1/holds`** lists every hold, active and released. A subject erasure or a space removal that a hold covers is `EHOLD`.

## ERRORS

`EPERM` without the role, or for approving your own proposal · `EMANIFEST` for an invalid patch or a steward proposal the rules refuse · `EDRIFT` if the live config has drifted and a diff can't be based on it · `EHOLD` for an erasure or a space removal a legal hold covers.

## LAWS

[LAW-10](sb-laws.7.md#law-10), [LAW-16](sb-laws.7.md#law-16).

## SEE ALSO

[sb-tenant(5)](sb-tenant.5.md), [sb-kill-switches(7)](sb-kill-switches.7.md), [sb-observability(7)](sb-observability.7.md)
