---
title: Show the Evidence, Fill the Gaps, Run Without Surprises
summary: Concept for W29 to W31 (accepted 2026-11-24): twenty small, high-value features for askers, stewards and operators. No new service, store or Go module.
---

# Show the evidence, fill the gaps, run without surprises

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.1 | 2026-11-24 | Platform lead | Proposed, waiting on the user's decisions | [Principles](01-principles.md) · [13-fresh-tidy-focused.md](13-fresh-tidy-focused.md) · [12-ask-steward-operate.md](12-ask-steward-operate.md) · [PRD waves](03-prd.md#waves) |
| 0.2 | 2026-11-24 | Platform lead | Plan accepted: every recommended decision (2026-11-24) | [Principles](01-principles.md) · [13-fresh-tidy-focused.md](13-fresh-tidy-focused.md) · [12-ask-steward-operate.md](12-ask-steward-operate.md) · [PRD waves](03-prd.md#waves) |

## Summary

After W28 an answer has cites, warns when its sources are old, and keeps a short and focused context. Stewards can retire, review and tidy documents, and operators can watch the index and the prompts. Reading the code at `83c8451`, the easy gaps that remain fall into three groups:

- **The asker can't see the evidence without leaving the page.** A cite chip gives a title and an age, but not the sentences it rests on, though the answer already holds the span (`cites[].span`). Chunks don't record the heading they sit under, so a cite of a 40-page handbook says only "Handbook". The UI speaks German and French, but "answers stay in whatever language the model writes" (sb-ask(3)). There is no way to ask about one particular document. An answer resting on a single document looks the same as one backed by five. A GitHub cite opens the file at the top, not at the cited lines.
- **Stewards only see the gaps people report.** Nothing counts the questions that found nothing, per space, so the stewards don't know what to write next. A space whose steward group has emptied out still takes questions that nobody gets. Marks and retirements go one document at a time. Feedback counts are admin-only (`GET /v1/admin/feedback`). The review list shows what is overdue but not what falls due soon. Owners hear about their old documents (SB-2806) but can't mark one reviewed. A new glossary term needs a manifest edit.
- **Operators find out late.** Connector health says a sync failed, but not why (expired token, rate limit, network). The only way to re-read one changed document is to wait for a sync or rebuild. Health is JSON only, so the site's Prometheus alerts can't see most of it. The audit API filters by kind and space but not by who did it or when. Nobody watches how long answers take, or how fast the disk is filling. And there is no way to tell everyone "the brain is down for an update at 18:00".

Three waves, twenty tickets. Each feature reuses an endpoint, job, adapter, screen or record kind the cell already has.

## Keeping it small

The bloat rules from W16 to W28 still apply:

1. **No new service the cell must run.** Everything runs in `sb-api`, `sb-jobs` or the existing scripts.
2. **No new store.** New records are `cell_docs` kinds or small files under `state/`, like `state/index/leftovers.json`.
3. **No new third-party Go module.**
4. **At most one setting per feature.**
5. **No new page.** Features go in as chips, lines, list sections and buttons.
6. **Nothing widens what anyone sees.** Every passage, count and list is checked live as the person.

One more rule for these waves: **evidence is shown, never paraphrased.** A passage is the source's own text at the cited version, shown as a quote. It is fetched and checked again when it is opened, so a share or an old trace can't show text the person has since lost access to (LAW-01, LAW-09).

## The twenty features

| # | Ticket | Wave | Feature | For | In one line |
|---|---|---|---|---|---|
| 1 | SB-3001 | W29 | Show the passage | Asker | **Show passage** on a cite opens the cited sentences as a quote, checked live as the asker |
| 2 | SB-3002 | W29 | Section on the cite | Asker | A cite names the heading it sits under: "Staff handbook › Carry-over" |
| 3 | SB-3003 | W29 | Answers in the asker's language | Asker | The answer is written in the person's UI language (English, German or French); the cites stay as they are |
| 4 | SB-3004 | W29 | Ask about this document | Asker | From a cite or a search result, the next question searches only that document (up to 5 documents) |
| 5 | SB-3005 | W29 | One-source warning | Asker | An answer whose every cite comes from one document says so in one line |
| 6 | SB-3006 | W29 | Open at the cited lines | Asker | A GitHub cite opens the file at the cited lines (`#L12-L20`) |
| 7 | SB-3101 | W30 | Questions that found nothing | Steward | Per space, the weekly count of questions that found nothing, the glossary terms they used, and the trend; counts only |
| 8 | SB-3102 | W30 | Spaces with no steward left | Admin, steward | A space whose steward groups have no active member shows in health, and its questions go to the company-wide answerers |
| 9 | SB-3103 | W30 | Review several at once | Steward | Pick up to 50 documents on the review list and mark them reviewed or retire them in one step, with one audit row each |
| 10 | SB-3104 | W30 | Feedback for the stewards | Steward | The steward screen shows the space's Helpful and Not helpful counts this month, and the documents cited most in unhelpful answers |
| 11 | SB-3105 | W30 | Coming due | Steward | The review list adds the documents that fall due in the next 14 days, so stewards can get ahead |
| 12 | SB-3106 | W30 | Owners mark their own reviewed | Owner | From the monthly owner note, an owner marks their own document "Reviewed, still right" |
| 13 | SB-3107 | W30 | Suggest a glossary term | Steward | A steward adds a term on the steward screen; it becomes a manifest proposal for an admin to approve |
| 14 | SB-3201 | W31 | Why a sync fails | Operator | Connector health says why the last sync failed (sign-in, rate limit, unreachable, source error) and names the fix |
| 15 | SB-3202 | W31 | Refresh one document | Operator | `sb index refresh CITE` re-reads one document from its source now, audited |
| 16 | SB-3203 | W31 | Health in Prometheus | Operator | Every health check is also a gauge on `/metrics`, so the site's existing alerts cover them |
| 17 | SB-3204 | W31 | Audit by person and date | Admin | `actor=`, `from=` and `to=` on the audit API, in the console and on `sb audit tail` |
| 18 | SB-3205 | W31 | Answer time watch | Operator | Health and the weekly report show the 95th percentile of seconds per answer, and warn when it grows by half week on week |
| 19 | SB-3206 | W31 | Disk runway | Operator | The doctor and health estimate the days until the data disk is full, and warn under 30 |
| 20 | SB-3207 | W31 | Notice to everyone | Admin | An admin sets one line, such as "Updating at 18:00, back by 18:30", that ends by itself; shown in the web UI and in chat replies |

## W29 Show the evidence (epic E30, SB-3001 to SB-3006)

The goal is that an asker can check an answer without leaving it, and the answer reads in their language.

### S29.1 See what it rests on

**SB-3001 Show the passage.** Each citation chip gets **Show passage**. It calls `GET /v1/traces/{id}/passage?cite=…`, which checks the asker's access to that document again, now and as them, and returns the cited span's text at that version (at most 600 characters, cut at a sentence). The UI shows it as a block quote with the cite. It is never rendered as markup (LAW-09). If the asker can't open the document any more, it says "You can't open this source now" and never shows the text. If that version is gone from the index, it says "This version is no longer indexed".
*Laws:* LAW-01 (a live check on every open), LAW-07 (nothing new is stored), LAW-09. *Acceptance:* the passage matches the span. Once access is revoked after the answer, it is refused. HTML in a source shows as text. axe passes.

**SB-3002 Section on the cite.** At ingest, each chunk records the nearest heading above it: a Markdown `#` to `###`, Word and OpenDocument headings, and HTML `h1` to `h3`, at most 120 characters. The chip and the copied and printed cite show "Title › Section". A rebuild fills it for every document (LAW-13). A heading is data. It can't move a document or change its rank.
*Laws:* LAW-09, LAW-13. *Acceptance:* a Markdown chunk under "## Carry-over" shows it. A document with no headings shows the title only. A rebuild fills it in.

**SB-3003 Answers in the asker's language.** `POST /v1/ask` takes `"language": "de"` (`en`, `de` or `fr`; anything else is ignored), and the web UI sends the person's chosen language. Slack and Mattermost send the manifest's `language`. The answer prompt adds one line: write the claims in that language and keep the labels. Cites, titles and passages stay in the source's own language. The eval suite gains a German case and a French case, and the gate runs them.
*Laws:* LAW-14 (an uncited claim is still marked), LAW-07. *Acceptance:* a German ask gets German claims with the same cites. An unknown code changes nothing. The new eval cases pass.

### S29.2 Narrow it and say how thin it is

**SB-3004 Ask about this document.** A citation chip and a Documents result get **Ask about this document**. The next question carries `"documents": ["docs:…"]` (at most 5), and retrieval keeps only chunks of those documents. The documents still go through the live checks first. A chip above the ask box shows the filter, and **Clear** removes it. A document the asker can't open gives `no_access` like any other.
*Laws:* LAW-01 (the filter runs after the check and can only narrow). *Acceptance:* an answer cites only the chosen document. A hidden one gives `no_access` and nothing else.

**SB-3005 One-source warning.** When every cite of an answer with at least two claims comes from one document, the response carries `"notice_single": true`. The UI then shows one line above the answer: "This answer rests on one document. Check it if it matters." Slack and Mattermost put it first. Hidden cites don't count.
*Laws:* LAW-01, LAW-14. *Acceptance:* three claims from one document warn. Two documents don't. A one-claim answer doesn't.

**SB-3006 Open at the cited lines.** For a GitHub cite, the connector turns the span into line numbers at that commit and adds `#L12-L20` to `url`. The folder connector's `file:` links and other kinds are unchanged.
*Laws:* none new. *Acceptance:* a span in lines 12 to 20 links to `#L12-L20` at the cited commit.

## W30 Fill the gaps (epic E31, SB-3101 to SB-3107)

The goal is that stewards see what people couldn't find, and the upkeep is spread across stewards and owners.

### S30.1 See what's missing

**SB-3101 Questions that found nothing.** The weekly job counts, per space, the asks that ended `no_sources` or `most_uncited`, from the traces. The steward screen and the stewards' digest line (SB-2507) show the count, last week's count, and the glossary terms those questions used (SB-2203), most used first. They never show question text.
*Laws:* LAW-07 (counts and terms the space already defined). *Acceptance:* two no-source asks count 2. A question's text appears nowhere.

**SB-3102 Spaces with no steward left.** The nightly job checks each space's steward groups against the directory. A space with no active steward shows in `GET /v1/admin/health` as `spaces_unstewarded` and warns. Its new questions go to the company-wide answerers (SB-1704) with "Finance has no active steward", instead of waiting for nobody. An unknown directory answer counts as a warning, never as fine.
*Laws:* LAW-12, LAW-03 (Unknown isn't fine). *Acceptance:* deprovisioning the last steward warns and reroutes. A directory error warns.

**SB-3103 Review several at once.** The review list gets a checkbox on each document, and **Mark reviewed** and **Retire** act on the selection. `POST …/review/marks` and `POST …/retired` take `"cites": […]` (at most 50). Each document is checked and audited on its own. The response lists any that failed and why.
*Laws:* LAW-11 (one audit row each), LAW-19 (only the steward's own space). *Acceptance:* marking 3 writes 3 rows. A document outside the space fails alone.

**SB-3104 Feedback for the stewards.** The steward screen shows the space's month of feedback: Helpful and Not helpful counts, last month's, and at most 10 documents most cited in Not helpful answers. It shows titles only for documents the steward can open and counts for the rest. Comments stay admin-only.
*Laws:* LAW-01, LAW-07. *Acceptance:* the counts match `GET /v1/admin/feedback` for the space. A hidden document is a count.

### S30.2 Share the upkeep

**SB-3105 Coming due.** `GET /v1/spaces/NAME/review?section=due` lists the documents that answers cite and that turn stale in the next 14 days, by `review_after_days` or their own review-by date (SB-2705), soonest first. **Reviewed, still right** works on them as on stale ones.
*Laws:* LAW-01. *Acceptance:* a document due in 10 days is listed. One due in 20 days isn't.

**SB-3106 Owners mark their own reviewed.** The monthly owner note (SB-2806) links to the Documents tab filtered to their documents that are past review. Each one there has **Reviewed, still right** when the person is its current owner at the source. The mark is the same record stewards make, and the audit row names the owner.
*Laws:* LAW-01, LAW-11, LAW-12 (a deprovisioned owner can't mark). *Acceptance:* the owner can mark their own document. They can't mark someone else's, which is refused with `EPERM`.

**SB-3107 Suggest a glossary term.** The glossary section of the steward screen gets **Suggest a term** (term, meaning, at most 10 expansions). It sends a manifest proposal through `POST /v1/spaces/NAME/proposals`, which an admin approves as today. Nothing changes until the proposal is applied.
*Laws:* LAW-16 (the manifest stays the truth), LAW-11. *Acceptance:* a suggestion appears in `GET /v1/admin/proposals`. Approving and applying it adds the term.

## W31 Run without surprises (epic E32, SB-3201 to SB-3207)

The goal is that the operator learns about trouble from the cell before users do, and can fix the small things in one command.

### S31.1 Know why

**SB-3201 Why a sync fails.** A connector's sync record keeps the class of its last failure: `auth` (401 or 403), `rate_limited` (429), `unreachable` (DNS, TLS, timeout), or `source_error` (5xx or anything else). It never keeps a URL, a body or a token. Health shows it with the runbook section that fixes it, as the doctor does (SB-2605).
*Laws:* LAW-07, LAW-03 (an unclassified failure is `source_error`, never fine). *Acceptance:* a fake 401 shows `auth` and its fix. The record carries no URL.

**SB-3202 Refresh one document.** `sb index refresh CITE --reason TEXT`, or `POST /v1/admin/index/refresh`, re-reads one document from its source now with the normal pipeline and checks, and writes an audit row. A document the source no longer has is tombstoned like a sync would. Admins only.
*Laws:* LAW-11, LAW-13. *Acceptance:* a changed document has a new version after the refresh. A gone one is tombstoned.

**SB-3203 Health in Prometheus.** `/metrics` gains `sb_health_warning{check="job:notify"}` and so on, at 0 or 1, for every line of `GET /v1/admin/health`, and `sb_health_unknown` for lines it can't read. The runbook gets example alert rules.
*Laws:* LAW-03 (an unreadable line is 1 on `sb_health_unknown`, never 0 on both). *Acceptance:* each warning in health is 1 on its gauge.

**SB-3204 Audit by person and date.** `GET /v1/admin/audit` takes `actor=` (a user id or email), `from=` and `to=` (dates), beside `kind` and `space`. The console's Audit section gets the same three fields, and `sb audit tail` gets `--actor`, `--from` and `--to`.
*Laws:* LAW-11 (filtering reads, never changes). *Acceptance:* the filters narrow as expected, and paging still works.

### S31.2 See it coming

**SB-3205 Answer time watch.** The usage ledger row each model call already writes gains its duration in milliseconds. Health and the weekly report add this week's 95th percentile of seconds per answer, last week's, and a warning when it is over 1.5 times last week's, as for prompt size (SB-2906).
*Laws:* LAW-07 (counts only). *Acceptance:* a fake week with doubled times warns. A flat one doesn't.

**SB-3206 Disk runway.** The host step keeps the data disk's used bytes for the last 14 days in `state/box/disk.json`. The doctor and the console's Box section estimate the days until the disk is full from that growth, and warn under 30 days. Under 7 days the warning is an alert. With fewer than 3 days of history it shows "not enough history yet", never a number.
*Laws:* LAW-03. *Acceptance:* steady growth gives the right estimate. A shrinking disk shows no runway. Two days of history shows "not enough history".

**SB-3207 Notice to everyone.** `PUT /v1/admin/notice` sets one line, at most 200 characters, with an end time no more than 7 days away. The web UI shows it at the top of every screen, and chat replies put it first, until it ends. `DELETE` clears it. It is one `cell_docs` record with an audit row. It is plain text and can't hold a link or markup.
*Laws:* LAW-09, LAW-11. *Acceptance:* the notice shows and ends by itself. More than 200 characters, or an end time more than 7 days away, gets `EINVAL`.

## Decisions for the user

Decided by the user on 2026-11-24, with every recommended option:

1. **Scope and order.** *Recommended:* all three waves, all twenty tickets, in the order W29, W30, W31. *Other:* W31 first, so the operator side lands before the asker features.
2. **Answer language (SB-3003).** *Recommended:* the language the person picked in the UI, or the manifest's language in chat. *Other:* the language the question is written in, which the model has to guess; or leave answers as they are.
3. **Passage length (SB-3001).** *Recommended:* up to 600 characters of the cited span, cut at a sentence. *Other:* the whole chunk, which is longer and shows more of the source.
4. **Owners marking their own documents (SB-3106).** *Recommended:* yes, for their own documents only. *Other:* stewards only, as today.
5. **Who sets the notice (SB-3207).** *Recommended:* admins only, ending by itself within 7 days. *Other:* stewards too, for their own space.

## Still open from earlier waves

- **SB-1605** and **SB-2601** need a run on a real Linux VM.
- **SB-903** and **SB-908** need a real GCP project.
- **Mattermost (SB-2005, SB-2105, SB-2603)** has been tested against a fake server only.
