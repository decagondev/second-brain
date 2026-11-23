---
title: Fresh Answers, a Tidy Corpus, a Focused Context
summary: Concept for W26 to W28 (accepted 2026-11-23): twenty small features against stale data, document rot and context creep, for askers, stewards and operators. No new service, store or Go module.
---

# Fresh answers, a tidy corpus, a focused context

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.1 | 2026-11-23 | Platform lead | Proposed, waiting on the user's decisions | [Principles](01-principles.md) · [12-ask-steward-operate.md](12-ask-steward-operate.md) · [11-loop-own-fit.md](11-loop-own-fit.md) · [PRD waves](03-prd.md#waves) |
| 0.2 | 2026-11-23 | Platform lead | Plan accepted: every recommended decision (2026-11-23) | [Principles](01-principles.md) · [12-ask-steward-operate.md](12-ask-steward-operate.md) · [11-loop-own-fit.md](11-loop-own-fit.md) · [PRD waves](03-prd.md#waves) |

## Summary

After W25 the brain answers with cites, tells people when something needs them, and is easy to ask, steward and run. Reading the code at `568dc00`, staleness is handled in five places, and they all sit with the steward or the operator:

- the 15-minute sync freshness target per connector (sb-ingest(7));
- the live permission check on every cite (LAW-01);
- the stewards' review list: documents answers keep citing that are older than `review_after_days`, documents whose owner left, wrong citations and copies (SB-1709, SB-2201, SB-2202);
- the **Reviewed, still right** mark (SB-2501);
- wrong citations turned into eval cases (SB-2204).

What is missing falls into three groups:

- **The asker isn't warned.** A cite chip says "400 days old", but an answer that rests mostly on old or overdue documents looks just like a fresh one. A document that says "review by March" is treated like any other. A connector that stopped syncing an hour ago leaves the answer silent about missing changes. A reader can report a wrong citation, but not an outdated one.
- **The corpus only grows.** Nothing takes a superseded document out of answers short of deleting it at the source. Nobody sees which documents are never cited, which notes rest on sources that have since changed, which glossary terms nobody uses, or two documents with the same name and different text. Owners never hear about their own old documents.
- **The context can creep.** A long conversation sends every earlier question to the model, however far the topic has moved. One long document can fill most of what the model reads. Nobody sees a scope pattern that matches nothing or far too much, chunks the index still holds after the source lost them, eval cases whose expected document has moved on, or prompts that grow week by week.

Three waves, twenty tickets. Every feature reuses an endpoint, job, adapter or screen that already exists. None of them changes what someone can see, and most of them make answers narrower.

## Keeping it small

The same bloat rules as W16 to W25:

1. **No new service the cell must run.** Everything runs in `sb-api`, `sb-jobs` or the existing scripts.
2. **No new store.** New records are `cell_docs` kinds (a retirement, a nudge's last run), or counts the cell already keeps.
3. **No new third-party Go module.**
4. **At most one setting per feature.**
5. **No new page.** Features go in as notices, badges, list sections and lines, not new screens.
6. **Nothing widens what anyone sees.** Every list and count shows only what the person could open, checked live as them.

One more rule for these waves: **staleness only ever narrows or warns.** A date inside a document can make it stale sooner, never later (LAW-09: data is not instructions). A retirement takes a document out of a space's answers, never puts one in. No feature here hides a cite's age or a warning.

## The twenty features

| # | Ticket | Wave | Feature | For | In one line |
|---|---|---|---|---|---|
| 1 | SB-2701 | W26 | Old-source warning | Asker | An answer that rests mostly on stale cites says so above the answer, on the web and in Slack and Mattermost |
| 2 | SB-2702 | W26 | Reviewed shows on the cite | Asker | A cite a steward marked "Reviewed, still right" at this version shows "Reviewed DATE" and isn't counted as stale |
| 3 | SB-2703 | W26 | Report as outdated | Asker | Feedback gains "outdated" for a cite, next to "wrong citation", and it puts the document on the review list |
| 4 | SB-2704 | W26 | Sync lag notice | Asker | When a cite's connector is past its freshness target, the answer says "Documents last synced at 14:05; newer changes may be missing" |
| 5 | SB-2705 | W26 | Review-by dates in the document | Steward | A `review-by` or `expires` date in a document's front matter or custom properties makes it stale on that date, if that's sooner |
| 6 | SB-2706 | W26 | Newer wins a tie | Asker | Among chunks that rank the same, the newer version comes first |
| 7 | SB-2801 | W27 | Retire from answers | Steward | A steward takes a superseded document out of their space's answers, with a reason and an optional replacement; undoable and audited |
| 8 | SB-2802 | W27 | Same name, different text | Steward | The review list shows documents in the space with the same title and different text, so the stewards can pick one |
| 9 | SB-2803 | W27 | Never cited | Steward | A list of the space's documents no answer cited in 180 days, as candidates to retire; titles only for those the steward can open |
| 10 | SB-2804 | W27 | Notes whose sources moved | Steward | A note comes back on the review list when a document it cites has a new version or is gone |
| 11 | SB-2805 | W27 | Unused glossary terms | Steward | The glossary section marks terms no question used in 180 days |
| 12 | SB-2806 | W27 | Owners hear about their own old documents | Owner | Once a month, an owner is told how many of their documents answers keep citing are past review, with a link to them |
| 13 | SB-2807 | W27 | Freshness in the report | Steward, admin | The steward screen and the weekly report show the share of cited answers resting on stale documents, and the trend; counts only |
| 14 | SB-2901 | W28 | Conversations stay short | Asker | A conversation sends at most the last 8 questions to the model, and the UI offers **Start fresh** once earlier ones are dropped |
| 15 | SB-2902 | W28 | One document can't crowd out the rest | Asker | At most 3 chunks of one document go into what the model reads |
| 16 | SB-2903 | W28 | Scope report | Admin | `sb space scope-report` shows how many documents each scope pattern matches, and names the ones that match none or more than half the connector |
| 17 | SB-2904 | W28 | Eval cases that went stale | Steward | A space's eval case whose expected document has a new version or is gone is marked "check this case" |
| 18 | SB-2905 | W28 | Index leftovers | Operator | The nightly check counts chunks of documents the source no longer has, or no space holds; health warns, and `sb index prune --dry-run` lists them |
| 19 | SB-2906 | W28 | Prompt growth watch | Operator | Health and the weekly report show the 95th percentile of input tokens per answer, with a warning when it grows by half week on week |
| 20 | SB-2907 | W28 | Start here goes stale | Steward | A `start_here` item that is gone, stale or unreadable to the space's members shows on the review list |

## W26 Fresh answers (epic E27, SB-2701 to SB-2706)

The goal: an asker can tell at a glance when an answer rests on old sources, and the brain prefers the newer of two equal ones.

### S26.1 Say when it's old

**SB-2701 Old-source warning.** When more than half of an answer's cites are `stale` (sb-ask(3); SB-1709), the response gains `"notice_stale": { "stale": N, "of": M, "oldest_days": D }`, and the UI shows one line above the answer: "Most of the sources for this answer haven't been reviewed in over a year. Check with the owner before relying on it." Slack and Mattermost put the same line first. The count uses only cites the asker can see, so hidden sources change nothing.
*Laws:* LAW-01 (hidden cites don't count), LAW-14 (the warning never replaces a cite's own label). *Acceptance:* the warning shows at 2 of 3 stale cites and not at 1 of 3; a hidden stale document doesn't change it; axe.

**SB-2702 Reviewed shows on the cite.** A cite whose document has a **Reviewed, still right** mark (SB-2501) at this version, in a space that holds it, carries `"reviewed": "2026-11-20"` and isn't `stale` until `review_after_days` passes from the mark. The chip says "Reviewed 20 Nov" in words. The steward's name isn't shown.
*Laws:* LAW-13 (the mark is on the document's version, so a rebuild keeps it). *Acceptance:* mark, then the cite is no longer stale; a new version is stale again by its own age.

**SB-2703 Report as outdated.** `POST /v1/feedback` gains `"outdated_citation": "docs:…@17"`, beside `wrong_citation`. The web UI offers both under a thumbs-down. An outdated report puts the document on the space's review list with `"outdated": N`, and a **Reviewed** mark clears it as it does a wrong citation. It is never turned into an eval case, because the cite wasn't wrong when it was made.
*Laws:* LAW-01 (only a cite the reporter was shown). *Acceptance:* a report puts it on the list; a mark takes it off; a cite the reporter wasn't shown is refused.

**SB-2704 Sync lag notice.** When a cite's connector is past its freshness target (the health check's `stale`, sb-observability(7)), the answer gains `"notice_sync": [{ "kind": "docs", "last_success": "…" }]`, and the UI shows "Documents last synced at 14:05; newer changes may be missing." It names the connector kind only, never a document.
*Laws:* LAW-03 (an unknown sync time is shown as unknown, never as fresh). *Acceptance:* a fake clock past the target shows the notice; an unreadable sync record shows "unknown", not nothing.

### S26.2 Trust the dates

**SB-2705 Review-by dates in the document.** Ingest reads one date from a document's own metadata: `review-by:` or `expires:` in Markdown front matter, or a custom `review-by` property in an Office file. The index keeps it on the chunk as `review_by`. A cite is `stale` from that date, or from `review_after_days`, whichever comes first. A date can only make a document stale sooner, never fresher, so a document can't keep itself off the review list.
*Laws:* LAW-09 (the date is data: it only narrows), LAW-13 (it is read again on a rebuild). *Acceptance:* a past date makes a new document stale; a date in 2099 changes nothing; a malformed date is ignored and counted in the ingest report.

**SB-2706 Newer wins a tie.** When two chunks have the same fused rank score, the one from the newer version (the one the index saw last) comes first. Nothing else about ranking changes, so eval scores stay where they are.
*Laws:* LAW-01 (ordering runs after the check). *Acceptance:* a tie goes to the newer document; the eval suite's scores don't move.

## W27 A tidy corpus (epic E28, SB-2801 to SB-2807)

The goal: stewards and owners can see what has gone stale or unused, and take it out of answers without waiting on the source.

### S27.1 Take it out of answers

**SB-2801 Retire from answers.** On the review list and on a cite, a steward can **Retire from this space's answers** with a reason (at most 200 characters) and, optionally, the document that replaces it. Answers, searches and drafts in that space then skip the document, and a cite of the replacement shows "Replaces: OLD TITLE" if the asker can open the old one. The document stays at the source and in other spaces. **Bring back** undoes it. Each retirement is a `cell_docs` record and an audit row.
*Laws:* LAW-19 (a retirement only narrows its own space), LAW-11, LAW-13 (a rebuild keeps it: it is held on the document's id, not the index). *Acceptance:* a retired document is never cited in that space and still is in another; a rebuild keeps it out; undo brings it back.

**SB-2802 Same name, different text.** The review list gains a section of documents the space holds with the same title (case and spacing ignored) and different text, grouped, newest first. Copies with the same text are already shown once (SB-2202). Documents the steward can't open are counted with their owner, as elsewhere on the list.
*Laws:* LAW-01. *Acceptance:* two "Leave policy" files with different text are grouped; a hidden one is a count only.

**SB-2803 Never cited.** `GET /v1/spaces/NAME/review?section=uncited` lists the space's documents that no answer cited in the last 180 days and that are older than 90 days, at most 100, oldest first, from the traces the cell already keeps. These are candidates to retire, nothing more.
*Laws:* LAW-01, LAW-07 (counts and titles the steward can open, never question text). *Acceptance:* a document cited once in the window isn't listed; a hidden one is a count only.

**SB-2804 Notes whose sources moved.** A note (SB-1705) records the versions of the documents it cites when it is published. When one of them has a new version or is gone, the note comes back on the review list as "its sources changed", until a steward edits it or marks it reviewed.
*Laws:* LAW-01 (the changed source is named only if the steward can open it). *Acceptance:* a new version of a cited document lists the note; a mark clears it.

### S27.2 Keep upkeep shared

**SB-2805 Unused glossary terms.** The glossary section on the steward screen marks each term that no question in the space used in the last 180 days, from the `terms` the traces already record (SB-2203). Stewards remove a term in the manifest as today; the screen only points.
*Laws:* LAW-07 (counts only). *Acceptance:* a term used once in the window isn't marked.

**SB-2806 Owners hear about their own old documents.** On the first Monday of the month, the notify job tells each active owner how many of their own documents answers cited at least 5 times in 30 days are past review: "4 of your documents that answers rely on are past their review date." The link opens the Documents tab filtered to them. It is a new kind `owned` they can turn off. Only the owner's own documents are counted.
*Laws:* LAW-01 (only documents the owner can open), LAW-07 (counts only), LAW-12 (a deprovisioned owner is told nothing). *Acceptance:* told once a month (fake clock); a document the owner can no longer open isn't counted.

**SB-2807 Freshness in the report.** The steward screen shows, per space, the share of the month's cited answers that rested mostly on stale cites (as in SB-2701), and the same share last month. The weekly report (sb-observability(7)) adds one line for the whole cell. These are counts only.
*Laws:* LAW-07. *Acceptance:* the share matches the traces; the report line carries no title or question.

## W28 A focused context (epic E29, SB-2901 to SB-2907)

The goal: what the model reads stays short and on topic, and the operator can see when the index, the scopes, the eval cases or the prompts start to creep.

### S28.1 Keep the model's context short

**SB-2901 Conversations stay short.** A later turn sends at most the last 8 earlier questions to the model (sb-ask(3), Conversations), not every one. The response says `"earlier_dropped": N` when some were left out (`dropped` already counts a department agent's uncited claims), and the UI then offers **Start fresh** under the answer. The stored conversation isn't changed.
*Laws:* LAW-07 (no more text is sent than before, only less). *Acceptance:* the ninth earlier question isn't in the prompt; **Start fresh** starts a new `cv_…`.

**SB-2902 One document can't crowd out the rest.** At most 3 chunks of one document go into what the model reads for one answer, after the live checks and the one-copy rule (SB-2202). The rest of the context goes to other documents. Search results are unchanged.
*Laws:* LAW-01 (it runs after the check). *Acceptance:* a long document matching every chunk gives 3, and the rest come from others; the eval suite doesn't regress.

**SB-2903 Scope report.** `sb space scope-report` reads the manifest and the index and prints, for each space and each scope pattern, how many documents it matches now. It names patterns that match none (a typo or a moved folder) and patterns that match more than half of a connector's documents (likely too wide). It reads counts only and changes nothing.
*Laws:* LAW-16 (the manifest stays the truth; the report only reads it). *Acceptance:* a pattern with a typo is named; `/**` is named as too wide.

### S28.2 Watch for creep

**SB-2904 Eval cases that went stale.** A space's eval case from a wrong citation (SB-2204) records the version it expects. When that document has a new version, or is gone, the case shows "check this case" on the steward screen, and the nightly eval run reports it apart from failures. Nothing is deleted on its own.
*Laws:* none new. *Acceptance:* a new version marks the case; it is reported apart, not as a regression.

**SB-2905 Index leftovers.** The nightly job counts chunks of documents the source no longer lists, and chunks no space holds in a cell with spaces. Health shows the counts with a warning above zero. `sb index prune --dry-run` lists them by cite, and `sb index prune` removes them, with an audit row. A rebuild already removes them, so this only saves waiting for one.
*Laws:* LAW-13 (the index is derived: removing a leftover loses nothing), LAW-11. *Acceptance:* a missed tombstone is counted and pruned; nothing a source still lists is touched.

**SB-2906 Prompt growth watch.** The usage ledger already records input tokens for every model call. Health and the weekly report add the 95th percentile for the week, and warn when it is over 1.5 times the week before. Glossary expansion, longer conversations and notes are where creep shows first.
*Laws:* LAW-07 (counts only). *Acceptance:* a fake week with doubled tokens warns; a flat one doesn't.

**SB-2907 Start here goes stale.** The review list gains the space's `start_here` items (SB-1706) that are gone at the source, stale, or that more than a quarter of the space's members can't open. Each one says why. The list is reviewed like any other item.
*Laws:* LAW-01 (an item the steward can't open is a count only). *Acceptance:* a deleted item is listed; one that is fine isn't.

## Decisions for the user

Decided by the user on 2026-11-23, with every recommended option:

1. **Scope and order.** *Recommended:* all three waves, all twenty tickets, in the order W26, W27, W28. *Other:* W28 first, so the context is kept short before more warnings land.
2. **When to warn the asker (SB-2701).** *Recommended:* when more than half of an answer's cites are stale. *Other:* when any cite is stale (more warnings, more noise); or no warning, chips only (as today).
3. **Retiring a document (SB-2801).** *Recommended:* stewards retire a document from their own space's answers, with a reason, undoable and audited. *Other:* leave it out, so only deleting at the source removes a document.
4. **Owner nudges (SB-2806).** *Recommended:* monthly, on by default, each person can turn it off. *Other:* off by default, so owners opt in.
5. **Conversation length (SB-2901).** *Recommended:* the last 8 questions. *Other:* the last 5 (shorter, loses more context), or the last 12.

## Still open from earlier waves

- **SB-1605** and **SB-2601** need their run on a real Linux VM. The same VM would check W17's `--doctor`, restore and offline upgrade, and the S3 restore drill against a company bucket (SB-2303).
- **SB-903 and SB-908** need a real GCP project; the human reviews of W0 to W7 are unchanged.
- **Mattermost (SB-2005, SB-2105, SB-2603)** has been tested against a fake server only. `sb selfhost check-chat` is the command for the real run.
