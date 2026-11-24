---
title: Ask Better, Capture Knowledge, Trust and Control
summary: Concept for W32 to W34 (accepted 2026-11-24): twenty small, high-value features for askers, stewards and owners, and for people and admins. No new service, store or Go module.
---

# Ask better, capture knowledge, trust and control

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.1 | 2026-11-24 | Platform lead | Proposed, waiting on the user's decisions | [Principles](01-principles.md) · [14-evidence-gaps-surprises.md](14-evidence-gaps-surprises.md) · [13-fresh-tidy-focused.md](13-fresh-tidy-focused.md) · [PRD waves](03-prd.md#waves) |
| 0.2 | 2026-11-24 | Platform lead | Plan accepted: every recommended decision (2026-11-24) | [Principles](01-principles.md) · [14-evidence-gaps-surprises.md](14-evidence-gaps-surprises.md) · [13-fresh-tidy-focused.md](13-fresh-tidy-focused.md) · [PRD waves](03-prd.md#waves) |

## Summary

W29 to W31 are built and pushed (`428f20d`). An answer now shows its passages and sections, can be narrowed to one document, warns when it rests on one source, and is written in the asker's language. Stewards see what people couldn't find. Operators hear about failing syncs, slow answers and a filling disk before users do. Reading the code at `428f20d`, the easy gaps left fall into three groups:

- **Asking is good but could be easier.** An answer shows only what it cites, so the near-misses the search found are lost. Glossary terms are listed at the foot of the answer, not where they're used. Every answer is the same length. When two sources disagree, the answer picks one quietly. A conversation can be copied one answer at a time, but not as a whole. "Who to ask" names an owner, but there's no way to see what else they own. A new person facing an empty ask box doesn't know what to ask in a space.
- **Knowledge is answered once and then lost.** A steward's good reply to a question goes back to one person and never reaches the index. Stewards answer the same question twice, because nothing spots a repeat. An open question can wait forever. When an owner leaves, their documents show "owner gone" but nobody takes them over. Owners hear about old documents but never that their documents helped. Terms that documents define themselves, like "purchase order (PO)", never reach the glossary.
- **People can't see or control what the brain keeps about them.** Only an admin can export or erase a person's data. A person can sign out everywhere, but can't see where they're signed in. Nobody checks that a restricted space's members still need it. The cost statement shows the month so far, not where it's heading. The weekly report doesn't break anything down by space.

Three waves, twenty tickets. Each feature reuses an endpoint, job, adapter, screen or record kind the cell already has.

## Keeping it small

The bloat rules from W16 to W31 still apply:

1. **No new service the cell must run.** Everything runs in `sb-api`, `sb-jobs` or the existing scripts.
2. **No new store.** New records are `cell_docs` kinds or small files under `state/`.
3. **No new third-party Go module.**
4. **At most one setting per feature.**
5. **No new page.** Features go in as chips, lines, list sections and buttons.
6. **Nothing widens what anyone sees.** Every title, count and list is checked live as the person.

One more rule for these waves: **a person's own data is theirs to see and remove.** Self-service export and erasure use the same code the admin's do (SB-611, SB-1903). They're scoped to the person's own records, audited without content, and never override a legal hold.

## The twenty features

| # | Ticket | Wave | Feature | For | In one line |
|---|---|---|---|---|---|
| 1 | SB-3301 | W32 | See also | Asker | Up to 3 documents the search found and the asker can open, but the answer didn't cite, listed by title under the answer |
| 2 | SB-3302 | W32 | Glossary where it's used | Asker | A glossary term in a claim is underlined, and its meaning shows on hover and focus |
| 3 | SB-3303 | W32 | Short or full answer | Asker | A **Short** toggle asks for at most 3 claims; the default stays as it is |
| 4 | SB-3304 | W32 | Sources disagree | Asker | A claim whose cited documents disagree is marked "Sources disagree", with both cites |
| 5 | SB-3305 | W32 | Download the conversation | Asker | The whole conversation as Markdown, made in the browser, under the same rules as Copy |
| 6 | SB-3306 | W32 | Their other documents | Asker | Under "Who to ask", **Their documents** lists what that owner owns that the asker can open |
| 7 | SB-3307 | W32 | Suggested questions | Steward, asker | Stewards set up to 8 starter questions for a space; an empty ask box in that space shows them as buttons |
| 8 | SB-3401 | W33 | Reply to note | Steward | **Publish as a note** on a steward's reply makes it a note in the space, so the next asker gets it from the index |
| 9 | SB-3402 | W33 | Spot a repeat | Steward | A new question to the stewards that matches an open or answered one in the space says "Looks like q_12"; **Answer both** replies once |
| 10 | SB-3403 | W33 | Notes come due | Steward | A published note gets a review date (180 days) and joins Documents to review and Coming due |
| 11 | SB-3404 | W33 | Nudge old questions | Steward | A question open 5 working days goes into the stewards' digest as overdue, then to the company-wide answerers |
| 12 | SB-3405 | W33 | Take over a document | Steward | A steward names a caretaker for a document whose owner is gone; "Who to ask" shows the caretaker |
| 13 | SB-3406 | W33 | Your documents helped | Owner | The monthly owner note says how many helpful answers cited their documents last month; counts only |
| 14 | SB-3407 | W33 | Terms the documents define | Steward | The review list suggests terms the space's documents define, like "purchase order (PO)"; **Suggest** makes the proposal |
| 15 | SB-3501 | W34 | What the brain keeps about me | Everyone | **Download my data** gives a person their own questions, saved questions, steward questions and settings, audited |
| 16 | SB-3502 | W34 | Forget my questions | Everyone | A person erases their own question history and saved questions; a legal hold still keeps them |
| 17 | SB-3503 | W34 | Where I'm signed in | Everyone | A list of the person's own sessions and API tokens (kind, when it started, last used), with **Sign out** on each |
| 18 | SB-3504 | W34 | Access review | Steward, admin | Every 90 days a restricted space's stewards confirm each member still needs it; **Remove** asks the admin, since the IdP stays the truth |
| 19 | SB-3505 | W34 | Month-end forecast | Admin | The cost statement adds where the month is heading at this rate, and warns before the budget passes |
| 20 | SB-3506 | W34 | Usage by space | Admin, steward | The weekly report breaks asks, found-nothing and helpful rate down by space; counts only |

## W32 Ask better (epic E33, SB-3301 to SB-3307)

The goal is that an asker gets more out of each answer and knows what to ask.

### S32.1 More from each answer

**SB-3301 See also.** The response gains `see_also`: at most 3 documents among the chunks retrieval kept for this asker that no claim cites, by title, cite and age. They're never shown as text. The UI lists them under the answer as "See also", each with **Show passage** and **Ask about this document**. The chat apps leave them out. They're the retrieval's own results after the live check, so they show nothing the asker couldn't find in Documents.
*Laws:* LAW-01 (only checked chunks), LAW-14 (never shown as claims). *Acceptance:* an answer citing 1 of 3 retrieved documents lists the other 2. A document the asker can't open never appears.

**SB-3302 Glossary where it's used.** The response already lists the glossary `terms` used (SB-2203). The UI underlines each one where a claim uses it, as a whole word with the same case rule. Its meaning shows on hover and keyboard focus as a tooltip, as plain text. Nothing changes on the server.
*Laws:* LAW-09 (a meaning is text, never markup), LAW-14. *Acceptance:* "PO" in a claim shows "purchase order". axe passes. A term's meaning containing HTML shows as text.

**SB-3303 Short or full answer.** `POST /v1/ask` takes `"length": "short"`. The prompt then adds one line: at most 3 claims, the most important first. The answer still goes through the same citation checks. The UI gets a **Short answer** toggle next to Ask, kept in the browser. Anything other than `short` changes nothing.
*Laws:* LAW-14. *Acceptance:* a short ask gets at most 3 claims, each still cited or marked uncited. An unknown value is ignored.

**SB-3304 Sources disagree.** The answer schema lets the model mark a claim `"disagree": true` when its sources say different things. The cell keeps the mark only if the claim cites at least two different documents the asker can open, and drops it otherwise. The UI shows "Sources disagree" before the claim, with both cites, and the chat apps say it in words. The citation eval suite gets two cases: one where documents disagree and one where they agree.
*Laws:* LAW-14, LAW-01 (a hidden document never counts). *Acceptance:* the disagreement case is marked with both cites. A mark on a claim with one document is dropped. The new eval cases pass.

### S32.2 Know where to look

**SB-3305 Download the conversation.** **Download conversation** saves the whole conversation as one Markdown file, made in the browser. Each turn follows SB-2401's Copy rules: claims with numbered cites, "Uncited" where the screen has it, and hidden sources only as the "N sources were not shown" line. Nothing goes to the server.
*Laws:* LAW-07 (nothing new is kept), LAW-14. *Acceptance:* a three-turn conversation downloads with three questions and their cites. An uncited claim keeps its label.

**SB-3306 Their other documents.** In "Who to ask", each person gets **Their documents**. It opens the Documents tab with `owner=EMAIL` on `GET /v1/search`, which keeps only results owned by that person, after the live check. The filter only narrows, and an owner who isn't exactly one active user returns nothing.
*Laws:* LAW-01, LAW-12. *Acceptance:* the list shows only that owner's documents the asker can open. A deprovisioned owner lists nothing.

**SB-3307 Suggested questions.** A space's stewards keep up to 8 starter questions, at most 120 characters each, on the steward screen. They're one `cell_docs` record per space, like reply templates (SB-2503), with an audit row. When the ask box is empty and the asker has picked that space, the questions show as buttons that fill the box. Company-wide asks show none.
*Laws:* LAW-19 (only the space's own stewards), LAW-11. *Acceptance:* a member of the space sees the questions. A non-member never gets them. A ninth question gives `EINVAL`.

## W33 Capture knowledge (epic E34, SB-3401 to SB-3407)

The goal is that a good answer, once given, is kept for the next person, and every document keeps someone looking after it.

### S33.1 Keep the good replies

**SB-3401 Reply to note.** A steward's reply to a question gets **Publish as a note**. It runs through the note flow that is already there (SB-1705, `note-publish`): it becomes a note in the space, with the reply's text and its cites as sources, and the question's text only if the steward keeps it. The publishing steward's approval is the click, and it's audited. The next sync puts it in the index.
*Laws:* LAW-16, LAW-11, LAW-07 (the asker's name is never in the note). *Acceptance:* a published reply is found by the next ask in the space. The note names no asker.

**SB-3402 Spot a repeat.** When someone asks the stewards, the cell compares the question with the space's open and answered questions from the last 90 days, using the embedder it already has. A close match is shown **to the stewards only** as "Looks like q_12". **Answer both** sends one reply to each asker, each checked as its asker. The asker never sees anyone else's question.
*Laws:* LAW-01, LAW-07. *Acceptance:* two near-identical questions are linked for the steward. Neither asker sees the other's.

**SB-3403 Notes come due.** A published note gets `review_by` 180 days ahead, the same field documents get from their front matter (SB-2705). It then shows in Documents to review and Coming due like any other document. Republishing the note resets the date.
*Laws:* LAW-13. *Acceptance:* a note published 175 days ago is coming due. One just republished isn't.

**SB-3404 Nudge old questions.** The notify job marks a question to the stewards that's been open 5 working days as overdue. It's listed first in the stewards' digest. After 5 more working days it goes to the company-wide answerers too (SB-1704), with "Finance hasn't answered this in 10 working days". A claimed question (SB-2502) waits for its claim to lapse first.
*Laws:* LAW-07 (the digest shows counts and titles only, as today). *Acceptance:* a question at day 5 is overdue. At day 10 it's rerouted. A claimed one isn't.

### S33.2 Keep someone on every document

**SB-3405 Take over a document.** On a document whose owner is gone (SB-2201), a steward of its space can **Name a caretaker**: an active member of the space. It's one `cell_docs` record, audited. "Who to ask" and the owners' monthly note then use the caretaker until the source names a new owner. The owner at the source stays the truth whenever it's someone active.
*Laws:* LAW-12 (a deprovisioned caretaker is dropped), LAW-19, LAW-11. *Acceptance:* an owner-gone document names its caretaker. A new owner at the source takes over again.

**SB-3406 Your documents helped.** The monthly owner note (SB-2806) adds one line: "Your documents were cited in N answers people found helpful last month." The number comes from the feedback counts already kept. It names no asker and no question, and it's left out when N is 0.
*Laws:* LAW-07. *Acceptance:* two helpful answers citing an owner's document make N=2. Unhelpful ones don't count.

**SB-3407 Terms the documents define.** The weekly job scans the space's chunks for a definition shaped like "purchase order (PO)" or "PO (purchase order)". Each term that's in at least 2 documents and not already in the glossary is suggested on the review list, with the number of documents. **Suggest** fills SB-3107's form for a proposal. Text that reads like an instruction is never suggested (sb-ingest(7)).
*Laws:* LAW-16 (only a proposal), LAW-09. *Acceptance:* a term in 3 documents is suggested. One already in the glossary isn't. An instruction-shaped term is skipped.

## W34 Trust and control (epic E35, SB-3501 to SB-3506)

The goal is that people can see and remove what the brain keeps about them, and admins see access and spending coming.

### S34.1 Your own data

**SB-3501 What the brain keeps about me.** `GET /v1/me/export` gives the person a JSON file of their own records. These are their traces (questions, answers and cites, with each cite checked again as them now), saved questions, questions to stewards, notification settings and token names. It's the admin export (SB-1903) scoped to the caller, with an audit row that holds no content. The Questions tab gets **Download my data**.
*Laws:* LAW-01 (cites checked again), LAW-11. *Acceptance:* the file holds the person's records and no one else's. A cite they've lost access to is left out.

**SB-3502 Forget my questions.** `POST /v1/me/forget` erases the person's own traces, conversations and saved questions through the retention job's erase (SB-611), after a typed confirmation. A legal hold still keeps what it holds, and the answer says how many were kept. The audit row has counts, never text.
*Laws:* LAW-07, LAW-11, holds win (SB-1207). *Acceptance:* after forgetting, their history is empty. A held conversation stays and is counted.

**SB-3503 Where I'm signed in.** `GET /v1/me/sessions` lists the person's own sessions and API tokens: kind (web, chat, token), when each started, last used day and expiry. It never shows an IP address or a token. **Sign out** ends one (`DELETE /v1/me/sessions/{id}`). The current one is marked "This browser".
*Laws:* LAW-12, LAW-11. *Acceptance:* signing out one session leaves the others. Nobody else's session is ever listed.

### S34.2 See it before it goes wrong

**SB-3504 Access review.** Every 90 days, the notify job asks a restricted space's stewards to review its members. The steward screen lists the members from the directory, with **Still needs it** and **Remove**. **Remove** sends the admin a request to take the person out of the group at the IdP, since SCIM stays the truth and the cell changes no group. Each answer is audited. Health warns when a review is 14 days overdue.
*Laws:* LAW-12, LAW-11, LAW-03 (an unread directory is a warning). *Acceptance:* a review records every answer. **Remove** reaches the admin and changes no group. Overdue warns.

**SB-3505 Month-end forecast.** `GET /v1/admin/costs` adds `forecast_usd`: the month so far, scaled by the days left at the last 7 days' rate. With fewer than 3 days of data it says "not enough data yet". When the forecast passes the budget, the existing `budget_warning` alert fires early, once a month.
*Laws:* LAW-03 (no forecast from too little data). *Acceptance:* a steady rate forecasts correctly. Two days of data gives no number. Passing the budget warns once.

**SB-3506 Usage by space.** The weekly report adds one row per space: asks, the share that found nothing (SB-3101), and the share rated helpful. Spaces with under 5 asks show "fewer than 5", so a small space can't be read as one person. The steward screen shows its own space's row.
*Laws:* LAW-07, LAW-19. *Acceptance:* the rows add up to the totals. A space with 3 asks shows "fewer than 5".

## Decisions for the user

Decided by the user on 2026-11-24, with every recommended option:

1. **Scope and order.** *Recommended:* all three waves, all twenty tickets, in the order W32, W33, W34. *Other:* W34 first, so the privacy and access features land first.
2. **Sources disagree (SB-3304).** *Recommended:* the model marks it, and the cell keeps the mark only when two different documents the asker can open are cited. *Other:* leave it out, since the model can be wrong about disagreement.
3. **Forget my questions (SB-3502).** *Recommended:* anyone can erase their own history; legal holds still keep theirs. *Other:* admins only, as today.
4. **When an old question is nudged (SB-3404).** *Recommended:* overdue at 5 working days, rerouted at 10. *Other:* 3 and 5, or 10 and 20.
5. **Access review (SB-3504).** *Recommended:* every 90 days for restricted spaces, and **Remove** asks the admin. *Other:* a report only, with no steward sign-off.

## Still open from earlier waves

- **SB-1605** and **SB-2601** need a run on a real Linux VM.
- **SB-903** and **SB-908** need a real GCP project.
- **Mattermost (SB-2005, SB-2105, SB-2603)** has been tested against a fake server only.
