---
title: Devlog
summary: The running story of the project, newest first.
---

# Devlog

Every PR adds an entry, and so does every significant design step. Newest first. Entries are never rewritten; corrections go in a new entry. Format: [05-living-documents.md](../docs/05-living-documents.md#devlog-entry-format).

## 2026-11-24 S37.1: From other tools

**Wave/slice:** W37 / S37.1 (SB-3801 to SB-3803) · **Status:** Done

**What:** With mcp: true in the cell's settings, other AI tools can use the brain over MCP with a person's own read-only API token: search, ask and show a cited passage, each checked live as that person. Ask takes one file of up to 2 MB, cited as your file and kept nowhere. Admins see every live API token in one list, oldest unused first, and can revoke one with a reason.

**Why this way:** MCP is how AI assistants now reach company knowledge, so a read-only endpoint on the existing token reuses every check the paths already make, and it stays off until a cell opts in. A file is read the way ingest reads a document, so the same hidden-text, injection and secret rules apply, but none of it is indexed, and the trace records only that a file was there. Tokens were only visible to their owners, so an admin couldn't find a stale one after someone left. The list shows the token's id and never its hash.

**Learned:** A token's last-used day comes free from the minute-by-minute directory re-check. An MCP tool error belongs in the result with isError, not as a JSON-RPC error, so the calling model can read the refusal. jsdom's File has no arrayBuffer, so the web UI reads a file with FileReader.

**Laws:** LAW-01, LAW-07, LAW-12, LAW-14. Every law test is still green.

## 2026-11-24 S36.2: Fill it in once

**Wave/slice:** W36 / S36.2 (SB-3705 to SB-3707) · **Status:** Done

**What:** Each term that found nothing on the review list has Draft a note, which puts a draft in the space's questions with the term as its title. In Who to ask, Ask them sends the question to that document's owner, who answers it from their own queue on the Questions tab. Before a note is published, the preview lists up to three documents in the space that already say the same.

**Why this way:** A found-nothing term tells the steward what's missing, and a draft in the queue is the shortest path to the existing note flow, so nothing new publishes. Asking the owner reuses the steward queue as a one-person queue that only that person can open; the owner sees only the question, and a deprovisioned owner can't be asked (LAW-12). Already written? checks each document as the steward, so it never names one they can't open, and it's a warning only.

**Learned:** A one-person queue fits the steward flow once the space key can name a person: replies, claims and notifications work unchanged, but rerouting, the digest's space lines and notes had to leave it out. A company-wide note's similar documents can't be filtered by space, since company-wide chunks carry none.

**Laws:** LAW-01, LAW-07, LAW-11, LAW-12, LAW-16. Every law test is still green.

## 2026-11-24 S36.1: From meetings and chat

**Wave/slice:** W36 / S36.1 (SB-3701 to SB-3704) · **Status:** Done

**What:** Meeting transcripts in the share (.vtt and .srt) are read cue by cue, each with its start time and speaker, so answers can cite them. Decisions get their own type: a Decisions filter, and in a space a list of its decisions with no query. A Slack shortcut or a Mattermost command saves a chat thread to the space's stewards, who publish it as a note or discard it. The stewards' digest says which terms were asked most.

**Why this way:** Much of what a company decides is said in meetings and chat and never written down, so the brain reads the transcripts people keep and lets anyone hand a thread to the stewards. A saved thread is never published by the save: only a steward's reply and note publish it, through the existing note flow. The thread is read only where the person pressed the shortcut or typed the command, so nobody saves a thread from a channel they weren't in. Trends show counts and the space's own glossary terms, never a question (LAW-07).

**Learned:** A cue's start time as a Markdown heading makes it the chunk's section with no change to ingest, but a short transcript's first chunk starts at the title, so its time shows in the passage rather than the section. A decision heading is often inside a chunk rather than at its start, so a chunk holding one counts too. Mattermost slash commands don't say which thread they were typed in, so the command takes the message's link and checks its channel.

**Laws:** LAW-01, LAW-07, LAW-09, LAW-11, LAW-12, LAW-16. Every law test is still green.

## 2026-11-24 S35.2: When an answer goes wrong

**Wave/slice:** W35 / S35.2 (SB-3605 to SB-3607) · **Status:** Done

**What:** The review list names the pairs of documents that answers marked as disagreeing this month. An asker who rates an answer Not helpful can send it to the stewards with a note, and the stewards see it in their queue. Each answer says how many of its sources someone has checked, on screen and in chat.

**Why this way:** Sources disagree was shown to askers one answer at a time, so nobody fixed the documents; counting pairs over a month points the steward at the two to reconcile. A report carries the answer, so it shows the steward only the claims and documents they can open themselves (LAW-01), and it names the asker only because they chose to send it. The verified count uses the same live check as Verified only, so a mark at an old version or past its interval never counts.

**Learned:** The extractive model already flags disagreement whenever two documents open with the same words, so conflicts showed up in the cell test without a planted trace. A report's uncited claims are counted, not shown, since nothing the steward can open stands behind them.

**Laws:** LAW-01, LAW-07, LAW-11, LAW-14. Every law test is still green.

## 2026-11-24 S35.1: Answers people can rely on

**Wave/slice:** W35 / S35.1 (SB-3601 to SB-3604) · **Status:** Done

**What:** A steward pins a cited reply as the space's verified answer, and a close ask in the space shows it first with the date it was verified. Verified only on ask and search answers from reviewed documents and steward notes alone. A review mark takes its own interval, and three unhelpful ratings since a mark put the document back on the review list.

**Why this way:** People asked the same questions and got slightly different answers; a pin gives the stewards' answer once and keeps the model's answer under it. A pin is checked again as each asker, and one cite they can't open hides the whole pin (LAW-01). Verified only is a filter after the live checks, so it only ever narrows. A mark that never expires would hide a document that has gone wrong, so ratings since the mark undo it.

**Learned:** A pin's question is shown to everyone in the space, so the steward sees and can edit it before pinning. The man page's response example must carry every field, which caught the new pinned field.

**Laws:** LAW-01, LAW-03. Every law test is still green.

## 2026-11-24 S34.2: See it before it goes wrong

**Wave/slice:** W34 / S34.2 (SB-3504 to SB-3506) · **Status:** Done

**What:** Stewards of a restricted space hear once when its access review opens and once when it becomes overdue, and the screen's buttons read Still needs it and Remove. The cost statement for the running month carries a month-end forecast, and a forecast over the budget sends the one budget warning early. The weekly report has a usage line per space, and the steward's review list shows their own space's week.

**Why this way:** The reviews already existed (SB-1202); what was missing was telling the people who must act, so the notify job reads the reviews and marks each notice in its clock, never twice. The forecast is the month so far plus the last week's rate, with no number before 3 days, so a quiet first day never alarms anyone; the guard reads it at most once an hour. Usage shows no shares below 5 asks, so a small space can't be read person by person (LAW-07), and a steward sees only their own space's row.

**Learned:** The notify clock is files named by part, so a review's notice part avoids colons. A label the screen shows must match its accessible name, so the aria label changed with the button.

**Laws:** LAW-03, LAW-07, LAW-21. Every law test is still green.

## 2026-11-24 S34.1: Your own data

**Wave/slice:** W34 / S34.1 (SB-3501 to SB-3503) · **Status:** Done

**What:** Download my data gives a person a JSON file of their own records: answers with the cites they can still open, conversations, feedback, saved questions and shares, questions to stewards, settings and token names. Forget my questions erases their own history after they type the phrase, keeping and counting what a legal hold covers. Where I'm signed in lists their sessions and API tokens with Sign out on each.

**Why this way:** The export is the operator's export run for the caller, so it reads exactly what an erasure would remove, and then every cite is checked again as them now: a document they lost isn't handed back through an old answer (LAW-01). Forget uses each store's own retention selection, so a hold keeps its records one at a time rather than refusing everything (SB-1207), and the audit holds counts only (LAW-07, LAW-11). A session's id is a hash of its key, never the key, so the list can't be used to take a session (LAW-12).

**Learned:** The trace records keep no question text; the questions live in the conversations. The person's Forget had to differ from the operator's Erase, which refuses a held person outright: here a hold keeps only what it covers.

**Laws:** LAW-01, LAW-07, LAW-11, LAW-12. Every law test is still green.

## 2026-11-24 S33.2: Keep someone on every document

**Wave/slice:** W33 / S33.2 (SB-3405 to SB-3407) · **Status:** Done

**What:** A steward names a caretaker for a document whose owner at the source is gone: an active member of the space, who then shows in Who to ask and gets the owners' monthly note for it. That note now also says how many answers people found helpful last month cited the person's documents. The review list suggests terms the space's documents define, like purchase order (PO), and Suggest fills the Glossary form.

**Why this way:** The owner at the source stays the truth: a caretaker shows only while no active owner does, and a deprovisioned one is nobody (LAW-12). One cell_docs record, audited by a hash (LAW-11, LAW-19). The helpful count is counts only, of documents the person can open now (LAW-01, LAW-07). Suggested terms come from documents the steward can open, never from flagged chunks, and are only ever a proposal (LAW-09, LAW-16).

**Learned:** The defined terms didn't need the weekly job: the review list already reads the space's chunks, so computing them on demand keeps nothing and is always current. An access request had to skip caretakers, since only the real owner can share at the source.

**Laws:** LAW-01, LAW-07, LAW-09, LAW-11, LAW-12, LAW-16, LAW-19. Every law test is still green.

## 2026-11-24 S33.1: Keep the good replies

**Wave/slice:** W33 / S33.1 (SB-3401 to SB-3404) · **Status:** Done

**What:** Publish as note can keep the question's text in the note, off by default. Every note now carries review_by 180 days ahead, so it comes up for review like any document, and a republish dates it again. A question sent to the stewards that looks like another of the queue from the last 90 days shows Looks like qs_N to the stewards, and Answer both replies to each. An open question is overdue from 5 working days and counted first in the stewards' digest; from 10, with no live claim, the company-wide answerers see it too.

**Why this way:** The note names no asker, and the question goes in only when the steward chooses (LAW-07, LAW-11). The review date is front matter that ingest already read, so no new field or index column (LAW-13). Repeats use the embedder the cell already has and never reach an asker; no embedder links nothing (LAW-01). An old question stays in its own queue and is shown company-wide only after a definite 10 working days with no live claim, audited by id and counts only (LAW-07).

**Learned:** Notes are never overwritten, which would have stopped a republish from moving its date. Comparing the note after its front matter keeps that rule for the text while letting the date move.

**Laws:** LAW-01, LAW-07, LAW-11, LAW-13. Every law test is still green.

## 2026-11-24 S32.2: Know where to look

**Wave/slice:** W32 / S32.2 (SB-3305 to SB-3307) · **Status:** Done

**What:** Download conversation saves every answered turn as one Markdown file, made in the browser. Who to ask has Their documents, which lists that owner's documents the asker can open, through GET /v1/search?owner=EMAIL with or without a query. A space's stewards keep up to 8 suggested questions, which the space's members see as buttons under an empty ask box.

**Why this way:** The download is Copy as Markdown turn by turn, so it shows nothing the screen doesn't and sends nothing to the cell (LAW-07). The owner filter runs after every check, on the owner as shown, so it only narrows and an owner who isn't one active user matches nothing (LAW-01, LAW-12). Suggested questions are the stewards' own text, read only by the space's members and audited by digest (LAW-11, LAW-19).

**Learned:** Listing an owner's documents needed no new index query: DocQuery already took owners for handover briefs. The owner filter had to check the owner as shown, not the chunk's raw owner field, or a deprovisioned owner's documents would still match.

**Laws:** LAW-01, LAW-07, LAW-11, LAW-12, LAW-19. Every law test is still green.

## 2026-11-24 S32.1: More from each answer

**Wave/slice:** W32 / S32.1 (SB-3301 to SB-3304) · **Status:** Done

**What:** An answer lists, under See also, up to three documents the search found and the asker can open but no claim cites, by title and age, each with Show passage and Ask about this document. Glossary terms are underlined where a claim uses them, with their meaning on hover and focus. A Short answer checkbox asks for at most three claims. A claim whose sources say different things is marked Sources disagree, in the UI and the chat apps, with both cites.

**Why this way:** See also comes only from the chunks the live check already allowed, and never shows their text, so it shows nothing Documents wouldn't (LAW-01, LAW-14). The disagreement mark is the model's word, kept by code only when two different documents survived the check for this asker, so a hidden document can never be hinted at. A short answer goes through the same composer, so every claim is still cited or marked uncited.

**Learned:** The extractive model the evals run had to learn to join two blocks on one subject, or a disagreement case could never be tested in make check; chunk text starts with the title line, so it compares the sentence after it. An eval expectation over the whole answer caught a disagreement from other retrieved documents, so the agreeing case asks as a person who can't open them.

**Laws:** LAW-01, LAW-09, LAW-14. Every law test is still green.

## 2026-11-24 S31.2: See it coming

**Wave/slice:** W31 / S31.2 (SB-3205 to SB-3207) · **Status:** Done

**What:** Health and the weekly report show seconds per answer at the 95th percentile, this week against last, and warn once it is over 1.5 times as long. The box check has a disk runway line: from at most one used-bytes sample a day it says how many days are left at the current rate, warns under 30 and fails under 7, and says not enough history yet until it has some. An admin can set one notice to everyone with PUT /v1/admin/notice: the web UI shows it at the top of every screen and the Slack and Mattermost apps put it first in every answer, until it ends by itself within 7 days or DELETE clears it.

**Why this way:** Each feature reuses what is there: the usage ledger for answer times, the box facts for disk history and the traces store for the notice, so nothing new runs or keeps data. The notice is plain text with no link or markup, so it can't be used to send people somewhere, and its audit rows carry only the text's SHA-256 (LAW-07). A runway is never guessed from too little history (LAW-03).

**Learned:** The web tests' fetchers answer every unknown URL as an answer, so a new GET on load had to be answered in each of them. An aside can't take role status under axe; a div can.

**Laws:** LAW-03, LAW-07. Every law test is still green.

## 2026-11-24 S31.1: Know why

**Wave/slice:** W31 / S31.1 (SB-3201 to SB-3204) · **Status:** Done

**What:** A connector's sync record keeps the class of its last failure, from the status the source answered with or the network error that kept it from answering: auth, rate_limited, unreachable or source_error. Health shows it with the section of the new connectors runbook that fixes it, and the console says it in words. POST /v1/admin/index/refresh and sb index refresh CITE --reason re-read one document through the same pipeline a sync uses, audited before and after; one the source says is gone leaves the index. /metrics now carries every health line as sb_health_warning and sb_health_unknown, read on each scrape. The audit API takes actor, from and to, and sb audit tail takes --since and --until.

**Why this way:** The class is all a record keeps, so nothing from the source reaches a log (LAW-07), and anything the cell can't place is source_error, never fine (LAW-03). A refresh only removes a document on a definite 404, 410 or missing file; any other fetch failure changes nothing. A health line that can't be read is 1 on the unknown gauge, never 0 on both.

**Learned:** Connectors each had their own private error type; a small StatusError interface in ports lets core read the status without importing an adapter. sb audit tail already used --from for the row number, so the day filters are --since and --until there. The console has no audit list, so the filters weren't added to one.

**Laws:** LAW-03, LAW-07, LAW-11, LAW-13. Every law test is still green.

## 2026-11-24 S30.2: Share the upkeep

**Wave/slice:** W30 / S30.2 (SB-3105 to SB-3107) · **Status:** Done

**What:** GET /v1/spaces/NAME/review?section=due lists the documents answers lean on that turn stale in the next 14 days, by review_after_days from their version or a mark, or by their own review-by date, soonest first with due_in_days, and the steward screen shows them with Reviewed, still right. POST /v1/documents/reviewed lets a document's current owner at the source mark it reviewed in each space that holds it, the stewards' own record with an audit row naming the owner; search results carry mine, and the Documents tab offers the mark on them. POST /v1/spaces/NAME/proposals/term turns a term, its meaning and up to 10 other spellings into an ordinary steward proposal built from the applied manifest, and the steward screen has Suggest a term.

**Why this way:** Coming due is the same staleness rule run forward, so a mark moves it on and a document's own date doesn't. An owner is recognised only by their own addresses in the directory now, plus the live check, so a deprovisioned or unknown owner gets EPERM, never a guess. A term is a manifest change, so it goes through the proposal and approval by hash (LAW-16); the cell builds the definition so the steward can't overwrite the rest of the space by mistake.

**Learned:** The proposal handler split into a steward check, a body, and a submit, so the term proposal reuses every check. The owner note keeps linking to the Documents tab unfiltered: a filtered list would be a new page.

**Laws:** LAW-01, LAW-11, LAW-12, LAW-16. Every law test is still green.

## 2026-11-24 S30.1: What the stewards don't see

**Wave/slice:** W30 / S30.1 (SB-3101 to SB-3104) · **Status:** Done

**What:** A space's review now shows how many questions found nothing this week and the week before, with the glossary terms they used, and the stewards' weekly digest line adds that count. A question sent from a space whose steward groups have no active member goes to the company stewards and says which space it came from; health's spaces_unstewarded lists those spaces and any the directory couldn't answer for. Review marks and retirements take up to 50 cites at once, each decided on its own with its own result, and the review list has checkboxes for them. The review also shows this month's and last month's Helpful and Not helpful ratings for answers asked in the space, and the documents the unhelpful ones cited most, checked as the steward.

**Why this way:** Counts and glossary terms tell a steward where the gaps are without showing anyone's question. Only a definite no-steward answer reroutes: a directory that can't answer keeps the question where it was, so nothing is widened on a guess (LAW-03), and health warns instead. A bulk action is the single action repeated, so it gets every check the single one has, and one failure stops nothing.

**Learned:** The running-cell steward test looked for the reported cite anywhere in the review body, which the new feedback section now also names; it now reads the items list only.

**Laws:** LAW-03, LAW-14. Every law test is still green.

## 2026-11-24 S29.2: Narrow it and say how thin it is

**Wave/slice:** W29 / S29.2 (SB-3004 to SB-3006) · **Status:** Done

**What:** POST /v1/ask takes documents (at most 5, KIND:ID): retrieval keeps only those documents' chunks, the ranked ones first and then the rest of each document in order, all through the same checks, and refused counts only the chosen documents; the web UI's Ask about this document on a chip or a search result fills a filter shown above the ask box until it is cleared or the conversation starts fresh. An answer of two or more cited claims that all cite one document carries notice_single, shown first on the web and in Slack and Mattermost. Ingest records each chunk's first and last line in the document's own text (line_start, line_end), and the GitHub connector links a file cite on the default branch at those lines, an issue or a pull request without them; the cell now wires the ask agent's Link to any connector that can link.

**Why this way:** A filter after the checks can only narrow (LAW-01), and the rest of a chosen document is looked up so a question worded differently still finds it; the three-chunk cap is lifted there because every chunk is one the asker chose. The one-source line counts only the cites in the answer, so a hidden document changes nothing. A GitHub version is a commit time, not a commit, so the link is to the default branch, and the cite's age says how old the lines are.

**Learned:** The ask agent's Link had never been wired in the running cell, so no cite carried a url there; it is now wired through each connector's own Link, and only GitHub has one. The plan's "at that commit" was not possible without a commit id, so the ticket says default branch.

**Laws:** LAW-01, LAW-14. Every law test is still green.

## 2026-11-24 S29.1: See what it rests on

**Wave/slice:** W29 / S29.1 (SB-3001 to SB-3003) · **Status:** Done

**What:** GET /v1/traces/{id}/passage returns one cite's own text, at most 600 characters cut at a sentence, only to whoever may read the trace, only for a cite in its answer, and checked live as them each time; the web UI's Show passage shows it as plain text in a quote. Each chunk records the nearest heading at or above it (Markdown levels 1 to 3, which the folder connector now writes for Word, OpenDocument and HTML headings) in a new section column, and cites, copies and printed footnotes read Title › Section. POST /v1/ask takes language (en, de or fr), which adds one line to the answer prompt; the web UI sends the chosen language and the chat apps send the manifest's.

**Why this way:** The answer already holds each cite's span, so the passage is a lookup, not a new store, and the live check runs on every open so an old trace can't show text the person has lost (LAW-01; Unknown refuses, LAW-03). A heading is only shown, never ranked or routed (LAW-09), and it is computed from the text so a rebuild fills it in (LAW-13). The language line goes in the system prompt so cites and labels are untouched, and the eval cases prove a language never changes what an answer cites.

**Learned:** There is no not-found error code, and a gone version isn't an error the person made: a passage of a version the index lost is a 200 with gone set, which the UI words as no longer indexed.

**Laws:** LAW-01, LAW-03, LAW-07, LAW-09, LAW-13, LAW-14. Every law test is still green.

## 2026-11-23 S28.2: Watch for creep

**Wave/slice:** W28 / S28.2 (SB-2904 to SB-2907) · **Status:** Done

**What:** An eval case keeps the version of the document it expects; when that document has a newer version or is gone the case is to check, on the case list and the review list, and a failing case to check holds no release. The nightly retention job counts the index's leftovers (documents their source no longer lists, chunks no space holds) for health, GET /v1/admin/index/leftovers lists them by cite, and `sb index prune` removes the gone ones with a reason and two audit rows. Health and the weekly report show the 95th percentile of input tokens per model call this week and last, warning when it grows by half. The review list names start_here items that are gone, stale or that more than a quarter of the space's members can't open, each with why.

**Why this way:** Creep is slow: an eval case quietly tests last year's document, a missed tombstone keeps a deleted file answerable, prompts grow a little each week, and a new member's reading list rots. Each check only counts or narrows: prune touches nothing a source still lists (LAW-13: the index is derived), a source that can't be listed is an error rather than everything gone, member access is the live ACL where Unknown is can't (LAW-01), and every number is a count (LAW-07).

**Learned:** The plan said prune would remove chunks no space holds too, but its own acceptance says nothing a source still lists is touched, and a company-wide ask can still use such a chunk; prune removes only documents their source no longer lists, and lists the rest as a scope to fix.

**Laws:** LAW-01, LAW-07, LAW-11 (prune is audited), LAW-13. Every law test is still green.

## 2026-11-23 S28.1: Keep the context short

**Wave/slice:** W28 / S28.1 (SB-2901 to SB-2903) · **Status:** Done

**What:** A later turn of a conversation sends only the last 8 earlier questions to the model, and the response says how many it left out in earlier_dropped; the web UI then offers Start fresh. At most 3 chunks of one document go into what the model reads for an answer or a draft, after the checks and the one-copy rule; search is unchanged. `sb space scope-report` prints, from GET /v1/admin/index/scopes, how many documents each space's scope patterns match now, naming patterns that match nothing and patterns that match more than half of their connector's documents.

**Why this way:** Long conversations and one long document crowded out what the question was about, and a scope typo or a `/**` went unnoticed until someone asked why an answer was missing or too broad. The caps only narrow what the model reads (LAW-01), the stored conversation keeps every question, and the scope report reads locations and counts only and changes nothing: a fix is still a manifest change (LAW-16).

**Learned:** Response already had `dropped` for a department agent's uncited claims, so the conversation cap's count is a separate field, earlier_dropped, and the plan's wording was corrected to match.

**Laws:** LAW-01 (the caps only narrow), LAW-07 (counts only), LAW-16 (the report changes nothing). Every law test is still green.

## 2026-11-23 S27.2: Keep upkeep shared

**Wave/slice:** W27 / S27.2 (SB-2805 to SB-2807) · **Status:** Done

**What:** The never-cited section names the space's glossary terms no question used in 180 days. On the month's first weekly run the notify job tells each active owner how many of their own often-cited documents are past review, a new kind `owned` that is on by default and can be turned off. The steward's screen shows the share of this month's and last month's cited answers that rested mostly on stale cites, and the weekly report adds the same share for the whole cell.

**Why this way:** Upkeep fell on stewards alone, and nobody could see whether answers were getting staler. Every new number is a count from the traces the cell already keeps: no question, title or term leaves the steward's screen, and an owner is told only of documents they can open now, checked as them (LAW-01, LAW-07, LAW-12).

**Learned:** The first Monday of a month is the week whose Monday falls on days 1 to 7, so a month that starts on a Tuesday waits for the following Monday; the test pins that rather than treating the 1st as special. Extracting the cite freshness rule into one helper let the owner note and the answer use the same judgement.

**Laws:** LAW-01 (owners told only of what they can open), LAW-07 (counts only), LAW-12 (a deprovisioned owner is told nothing). Every law test is still green.

## 2026-11-23 S27.1: Take it out of answers

**Wave/slice:** W27 / S27.1 (SB-2801 to SB-2804) · **Status:** Done

**What:** A steward can retire a document from their own space's answers with a reason and, optionally, the document that replaces it; answers, searches and drafts in that space skip it, and a company-wide ask skips it only when every space holding it did. Retired lists them with Bring back, both audited. A replacement's cite says Replaces OLD TITLE when the asker can open the old one. The review list now groups documents with the same title and different text, counts a note's sources that changed or are gone, and shows documents no answer cited in 180 days on request.

**Why this way:** Old documents that are still at the source keep coming back in answers, and stewards had no way to say so short of asking the owner to delete them. A retirement only narrows what one space's answers use (LAW-09), it is on the document's id so a rebuild keeps it out (LAW-13), and every list checks each document live as the steward and only counts the rest.

**Learned:** The web test fakes answered every URL with the review list, so a new list read on the same screen needed its items checked for shape rather than trusted; the request-count tests now count only the requests they are about.

**Laws:** LAW-09 (a retirement only narrows, and only in its space), LAW-13 (kept across a rebuild). Every law test is still green.

## 2026-11-23 S26.2: Trust the dates

**Wave/slice:** W26 / S26.2 (SB-2705, SB-2706) · **Status:** Done

**What:** Ingest reads a document's own review-by or expires date from Markdown front matter, or from a .docx or .pptx file's custom review-by property, and keeps it on every chunk (review_by in the index). A cite is stale from that date or from its age, whichever comes first, and a review mark doesn't lift it; the review list holds such a document once answers cite it. Of two candidates with the same fused rank score, the newer version now comes first.

**Why this way:** A date written in a document is data, so it can only make the document stale sooner, never fresher (LAW-09): a document can't keep itself off the review list with a date in 2099. A value that isn't a date is ignored and counted by connector only. The tie-break changes order and nothing else, so what passes the checks, and the eval scores, stay as they were.

**Learned:** Rounding the fused score to find ties would have lumped neighbouring ranks together (1/61 and 1/62 agree to three places), so the tie is the exact score; the plan's wording was corrected to match.

**Laws:** LAW-09 (a document's date only narrows), LAW-13 (the date is read from the source again on a rebuild). Every law test is still green.

## 2026-11-23 S26.1: Say when it's old

**Wave/slice:** W26 / S26.1 (SB-2701, SB-2702, SB-2703, SB-2704) · **Status:** Done

**What:** An answer whose cited documents are mostly stale carries notice_stale, and the web UI, Slack and Mattermost put one warning line before the answer. A cite a steward marked Reviewed, still right at its version carries reviewed, says so on its chip, and is stale only once review_after_days pass from the mark. POST /v1/feedback takes outdated_citation, which puts the document on the review list with an outdated count until a mark clears it, and never becomes an eval case. When a cite's connector is past the 15-minute freshness target, or the cell can't tell, the answer carries notice_sync and says when that kind last synced, or that it is unknown.

**Why this way:** The warning counts only the cites in the answer, which the asker could open, so a hidden document changes nothing. An unreadable sync record is shown as unknown, never as fresh (LAW-03), and an unreadable mark store only means no reviewed chip, so a failure never hides staleness. Both notices name a connector kind or a count, never a document the asker didn't see.

**Learned:** The mark already lived per space and version, so the cite reads it per answer with one lookup per space; an outdated report reuses the wrong-citation path on the review list, including the rule that a report after a mark brings the document back.

**Laws:** LAW-01 (only the answer's own cites count), LAW-03 (unknown sync is never fresh), LAW-13 (a mark is on the document's version), LAW-14 (warnings never replace a cite's own label). Every law test is still green.

## 2026-11-23 S25.2: Know it's working

**Wave/slice:** W25 / S25.2 (SB-2605, SB-2606, SB-2607) · **Status:** Done

**What:** Every doctor line that isn't a pass names its fix, and the fixes that need more than one command point to the section of docs/runbooks/selfhost.md that explains it. The web UI shows each API token's days left, and the notify job tells a token's owner once, seven days before it expires, by the token's name (a new tokens kind they can turn off). The notify job keeps its last run, when, the channel and counts only, and GET /v1/admin/health, the console's Notifications line and the weekly report show it, with a warning when the last run failed or none ran in the last hour.

**Why this way:** Operators learn of a stopped notify job from the health page, not from people who weren't told. The run record holds counts, never who was told or what about (LAW-07), and stays in the cell's state owner-only, so it adds no store. A token warning names only the person's own token, to that person.

**Learned:** The fixes already covered every non-pass line; what was missing was a test that would keep it so, and a check that a runbook section a fix names exists, so a renamed heading fails the build instead of sending an operator nowhere.

**Laws:** LAW-07 (the run record and the token warning carry counts and the owner's own token name only), LAW-12 (tokens are still the owner's own, read-only). Every law test is still green.

## 2026-11-23 S25.1: Set it up right the first time

**Wave/slice:** W25 / S25.1 (SB-2601, SB-2602, SB-2603, SB-2604) · **Status:** Done

**What:** A box sends notifications by email from install.env (SB_EMAIL_SERVER, SB_EMAIL_FROM, a root-only SB_EMAIL_LOGIN_FILE), checked by sb selfhost check-email at install, with --check and by the doctor. People press Send me a test, and operators run sb selfhost check-notify, to send one fixed message through the cell's real channel. sb selfhost check-chat asks Slack or Mattermost what the bot may do and names each missing scope or permission with where it is granted. sb cell check-config checks a configuration and that its secrets are in the vault owner-only, and update.sh runs it before any restart.

**Why this way:** Each check names the fault and where it is fixed, and none prints a secret: check-config reads a secret file's mode, never its content, and check-chat never prints the token. The test message is a fixed text, so a test can't carry anything else (LAW-07), at most once a minute per person. The sign-in for the mail server goes to the two services that send and nowhere else, as the other secrets do.

**Learned:** The install wrote vault files with the default umask, so the host copies were readable by others even though the volumes weren't; the install now writes them owner-only, and update.sh fixes an older install's files once before its first check, so the new check doesn't stop every box that upgrades. The proxy's file isn't a vault reference, so check-config leaves it to check-proxy.

**Laws:** LAW-07 (the test is a fixed text), no secret in any check's output (tests fail on the token, the password or a secret file's content). Every law test is still green.

## 2026-11-23 S24.2: Know what waits

**Wave/slice:** W24 / S24.2 (SB-2504, SB-2505, SB-2506, SB-2507) · **Status:** Done

**What:** The steward screen's Questions and Documents to review headings say how many wait, read again every five minutes. Download CSV saves the review list as shown, made in the browser. A person can pause every notification for 1 to 30 days, and the pause ends by itself. A steward's weekly digest adds a line per space they steward: answered this week, still open, and the oldest one's age.

**Why this way:** Counts come from the lists the screen already loads, so they can't say more than the lists do. The CSV is built from the loaded list, so it can't hold a row the screen doesn't, and a document the steward can't open stays a count with its owners (LAW-01); every cell is quoted and one that starts like a formula is prefixed so a spreadsheet never runs it. A pause is per person, so the job still tells the space's other stewards, and it is set only as days from now, so a body can't set an end date. The digest line is counts and a space's name only (LAW-07), to the stewards there are when it is sent (LAW-21).

**Learned:** A steward with nothing changed in their spaces still needs the queue line, so the digest now goes to a steward with lines even when no document changed; it links to the Questions tab rather than one space, since a line can cover several.

**Laws:** LAW-01 (the CSV has no hidden title), LAW-07 (the digest line is counts only; `TestDigestStewardLines` fails on any question or reply text), LAW-21 (a removed steward gets no line; a paused one is skipped while the others are told). Every law test is still green.

## 2026-11-23 S24.1: Clear the list

**Wave/slice:** W24 / S24.1 (SB-2501, SB-2502, SB-2503) · **Status:** Done

**What:** A steward can mark a document on Documents to review as reviewed, still right, at its current version; it leaves the list until a new version, the mark's age or a new wrong-citation report. A steward can claim an open question for an hour, so the others see who is answering; the claim never blocks a reply. Each space keeps up to 20 reply templates its stewards add, edit, delete and use to fill a reply.

**Why this way:** The list only helps if a steward can clear it without changing the document. The mark is kept against the document's own version, so an index rebuild neither clears nor renews it (LAW-13), and it is written to the audit log. A claim is shown only while its holder is still a steward, checked against the directory each time, so a person who leaves keeps no visible hold on the queue. Templates are the space's, so another space's template looks like none (EPERM), the same as one that doesn't exist.

**Learned:** The audit log already has a config kind for steward actions, so marking reviewed uses it (state reviewed) rather than a new kind. Notes read wider than the space, documents with no owner and copies stay listed whatever the mark, since a mark says the text is right, not that its readers or owner are.

**Laws:** LAW-01 (a mark or a template of another space looks like none; a claim shows only an active steward's email), LAW-13 (marks key on the document version). Every law test is still green.

## 2026-11-23 S23.2: Find it faster

**Wave/slice:** W23 / S23.2 (SB-2404, SB-2405, SB-2406) · **Status:** Done

**What:** Search filters by file type and age, on the API and the Documents tab. The ask box offers the person's last 20 questions from their own browser. A search with no results, or an answer with nothing in a space, offers what to try next: the whole company, clearing filters, asking it as a question.

**Why this way:** Filters run after the live checks, on documents the person may already open, so they can only narrow and the hidden count stays the same. Recent questions never reach the cell, so the cell keeps no more question text than it did (LAW-07); they go when the person signs out or the session ends, which matters on a shared computer. Next steps depend only on what the person typed and chose, so a document they can't open can't be inferred from which buttons appear (LAW-01).

**Learned:** The glossary suggestion in the plan would have meant matching the space's terms against the question a second time; the terms the question used are already shown under results and answers, so no second path was added.

**Laws:** LAW-01 (`TestSearchFilters`: a denied document never appears under any filter and `refused` is unchanged; next steps the same with or without hidden documents), LAW-07 (recent questions stay in the browser). Every law test is still green.

## 2026-11-23 S23.1: Take the answer with you

**Wave/slice:** W23 / S23.1 (SB-2401, SB-2402, SB-2403) · **Status:** Done

**What:** Each answer can be copied as plain text or Markdown, or printed (and so saved as PDF), with its cites numbered and every uncited claim still marked. The page can be used from the keyboard alone: a skip link, `/` to the ask box, Ctrl+Enter to ask, Esc to close a panel.

**Why this way:** The copy is built from the answer the screen shows and nothing else, so it can't carry more than the asker saw; uncited claims keep their word, as on screen (LAW-14). Printing is a stylesheet, not a server-side PDF, so there's nothing new to run or to leak. Shortcuts never fire while typing, so they can't steal a character.

**Learned:** Vitest leaves CSS empty unless `css: true`; the print test needs the stylesheet's text. Markdown output needs escaping, or a claim's own brackets could turn into a link in whatever the person pastes it into.

**Laws:** LAW-14 (the uncited mark survives copy and print; axe with the new controls in each language), LAW-01 (only what the answer shows). Every law test is still green.

## 2026-11-23 S22.3: Tokens and email

**Wave/slice:** W22 / S22.3 (SB-2306, SB-2307) · **Status:** Done

**What:** A person can make a read-only API token (`sbt_…`, at most 90 days) that asks and searches as them and nothing else, audited at each use. A cell with neither Mattermost nor Slack sends the W20 notifications by email through the company's mail server (sb-cell(5) `email`), over STARTTLS only.

**Why this way:** A token is a session of its owner, stored as a hash, so every path that ends sessions (deprovisioning, sign out everywhere) ends tokens with no new code, and the token can do no more than its owner. A use that can't be audited is refused, never let through. Email carries exactly the chat's text and link, so a mailbox, a forward or a mail server's archive holds no more than a chat would; the subject names the kind only.

**Learned:** Go's `net/smtp` refuses PLAIN sign-in outside TLS by itself, but nothing stops a client from carrying on in the clear when a server doesn't offer STARTTLS; the adapter checks the extension and stops. Header injection was the other risk: the message's text and the recipient never reach a header unchecked.

**Laws:** LAW-12 (tokens: the generator signs in with tokens every other time; a deprovisioned owner's token stops within the bound), LAW-07 (`TestLAW07_EmailNotificationsSayNothingOfWhatWasAsked`, the real notify job through the SMTP adapter, the whole email checked, with a self-test). Every law test is still green.

## 2026-11-23 S22.2: Fast enough on a small box

**Wave/slice:** W22 / S22.2 (SB-2304, SB-2305) · **Status:** Done

**What:** Asking in the web UI shows the documents the asker can open (title, owner, age) at once, while the answer is written, then replaces them with the cited answer. Each person may ask `asks_per_minute` questions in any minute (default 6); one more gets `ERATE` with the seconds to wait, and the weekly report counts these refusals.

**Why this way:** On a CPU box an answer takes 7 to 32 seconds, and search needs no model, so its results can come first; they are search's own results, checked live as the asker, and shown without their text so nothing there reads as a claim. One person asking twenty questions shouldn't hold up everyone on a one-CPU box; the count lives in the cell's memory, so a restart can only let a few more through, never refuse more, and a refusal is traced with no question so the report can count it.

**Learned:** `sb tenant verify` asks through the cell's own agent as sampled users, so a per-person limit would have failed verification on any real tenant; its agent is a copy with no limit.

**Laws:** LAW-14 (the sources never render as claims, axe before and after the answer), LAW-01 (the sources are search's live-checked results), LAW-16 (`TestLAW16_TheManifestIsTheTruth` now generates `asks_per_minute`). Every law test is still green.

## 2026-11-23 S22.1: Language and logs

**Wave/slice:** W22 / S22.1 (SB-2301, SB-2302, SB-2303) · **Status:** Done

**What:** The web UI comes in English, German and French: every string it shows is in one catalog, the manifest's `language` sets the default and a person can switch, kept in their browser, and error codes are translated. A cell can stream its audit rows to the company's SIEM as RFC 5424 over TLS, each row exactly as the export has it. A self-hosted box can send its off-box copies to the company's S3 bucket instead of a mounted NAS: `SB_OFFSITE=s3://BUCKET/PREFIX` with a key pair file.

**Why this way:** The catalog is keyed by the English text, and a test parses every component, so a string added without a translation fails the build rather than showing up in English. The SIEM stream uses the audit log itself as its queue and keeps only a cursor, so a SIEM that is down never blocks a write; rows older than a day are named in one gap message for the SIEM to read from the export. The bucket copies are sealed and written once, and `sb backup push` refuses anything that is neither sealed nor a small record, so the bucket needs no object lock and still never holds anything readable.

**Learned:** A TLS write to a peer that has hung up succeeds, and the row is lost; the sender now checks the connection with a 1 ms read before each write. In the bucket, the box can't rotate what it wrote once, so the fourteen nights are the bucket's own lifecycle rule.

**Laws:** LAW-14 (axe and the uncited-claim mark in each language), LAW-11 (`TestLAW11_TheSIEMGetsEveryRowInOrder`, with a self-test for a sender that trusts every write), LAW-07 (`TestLAW07_NothingUnsealedReachesTheBucket`, with a self-test for a push that sends whatever it is given). Every law test is still green.

## 2026-11-23 S21.4: Spreadsheets and national ID numbers

**Wave/slice:** W21 / S21.4 (SB-2206, SB-2207) · **Status:** Done

**What:** The folder connector reads `.xlsx`, `.ods` and `.csv` files: each sheet becomes `Sheet: NAME` and one tab-separated line a row, up to 5,000 rows a sheet and 16 MiB a file. The personal-data scanner also finds UK National Insurance numbers, US Social Security numbers, Dutch BSNs and German tax ids, flags them `personal_data` and masks them but the last four.

**Why this way:** A spreadsheet is read as its saved values, never evaluated, and only the parts that hold them are opened, so a formula, a macro or a comment can't make ingest compute or read anything else. Each national number has to keep its format's rules and check digits, and SSNs need separators, so phone and order numbers are mostly left alone; a nine-digit number that passes the BSN 11-check is masked, since the scanner errs towards masking.

**Learned:** LibreOffice ends every sheet with a row repeated about a million times and every row with a cell repeated a thousand times; expanding repeats only for rows and cells that hold a value keeps an `.ods` cheap to read without a separate bound.

**Laws:** LAW-09 and LAW-13 (`TestLAW09_OfficeFilesAreData`, `TestLAW13_OfficeIndexIsDerived`, now over spreadsheets too, with a canary in formulas, comments, macros and scripts), LAW-07 (`TestLAW07_PersonalDataIsMasked` over national ids; a self-test catches a masker that misses them). Every law test is still green.

## 2026-11-23 S21.3: Ask for access

**Wave/slice:** W21 / S21.3 (SB-2205) · **Status:** Done

**What:** Under "N sources were not shown" on an answer or search, *Ask the owners for access* (`POST /v1/access-requests`) sends each owner of a document the source refused one `access` message naming their document and who asked. The asker sees only counts: sent, already asked this week, and with no owner to ask.

**Why this way:** Only `source_denied` documents are asked about: a space, label or barrier refusal isn't fixed by sharing at the source, and a request across a barrier would tell the other side what was asked. Requests are remembered under a hash of asker, document and week, so the ledger names neither, and an owner who turned `access` off is skipped without the asker learning it.

**Learned:** A document the asker can't open is never a search candidate unless the index's prefilter is stale, so the law test and the cell test both change the source's ACL after the sync, as happens between syncs in a real cell.

**Laws:** LAW-01 (`TestLAW01_AskingForAccessTellsTheAskerOnlyCounts`), LAW-07 (`TestLAW07_AnAccessRequestNeverCarriesTheQuestion`), self-test catches a store that forgets requests. Every law test is still green.

## 2026-11-23 S21.2: Space glossary and wrong citations as eval cases

**Wave/slice:** W21 / S21.2 (SB-2203, SB-2204) · **Status:** Done

**What:** A space keeps a `glossary` in the manifest (up to 200 terms, proposed like any space change). Asks and searches in that space match each term the question uses and also search for its meaning, send the terms to the model in one data block, and list them in `terms`; entries that read like instructions are never used and the stewards' review list names them. A wrong-citation report on the review list becomes an eval case (`/v1/spaces/NAME/evals`): the question and the document it should cite, never an answer. The new `evals` job runs the cases once per release, and a release that fails a case an earlier release passed is held for the tenant: the updater runs the release the cases last passed on instead.

**Why this way:** The glossary lives in the manifest so it changes through the steward-proposal and operator-approval path that already exists, with no new store. Meanings are matched as extra phrases (`Query.Also`), not added to the query text, because Postgres ANDs plain query words. A tenant's questions can't leave the cell, so the factory gate can't run its cases: the cell runs them on the release it runs and holds a regressed one through the updater, which admits the fallback with the same four checks.

**Learned:** The law test caught an entry the sanitiser missed: `system:` only flags at the start of a line, so a term and its meaning are now checked on their own as well as together.

**Laws:** LAW-19 (`TestLAW19_ASpacesGlossaryIsUsedOnlyInThatSpace`, self-test catches a glossary used in every space), LAW-09 (`TestLAW09_GlossaryEntriesAreDataNeverInstructions`), LAW-06 (`TestLAW06_AReleaseThatRegressesATenantCaseIsHeld`, self-test catches an updater that ignores the hold). Every law test is still green.

## 2026-11-23 S21.1: Orphaned and duplicate documents

**Wave/slice:** W21 / S21.1 (SB-2201, SB-2202) · **Status:** Done

**What:** The stewards' review list gains documents whose owner at the source is no active directory user, marked `owner_gone` with no owner shown, and lists copies of the same text once, as the newest copy, with the others under `copies`. Ingest keeps a hash of each document's normalised text (`text_sha256`, a new index column), and answers, searches and drafts cite only the newest copy the asker can open.

**Why this way:** The owner check reads the directory when the list is asked for, so a deprovisioned owner's documents are on it at once with no new job; a directory that can't answer is `EUNAVAIL`, never a guess. Copies are dropped after the live check, so the copy kept is one the asker can open, and the hash leaves out the title, so a renamed copy still matches. The hash comes from the source text alone, so a rebuild gives it again.

**Learned:** Dropping copies before trimming to the budget keeps a full context: the live check now runs over every candidate, as it already did in one batch.

**Laws:** LAW-12 (`TestLAW12_ADeprovisionedOwnersDocumentsAreOnTheReviewList`, self-test catches a check that never finds an owner gone), LAW-01 (`TestLAW01_OwnerGoneNamesOnlyWhatTheStewardCanOpen`), LAW-13 (`TestLAW13_AnswersCiteTheNewestCopyBeforeAndAfterARebuild`, self-test catches an index with no text hash). Every law test is still green.

## 2026-11-23 S20.3: Share in Mattermost and handover by name

**Wave/slice:** W20 / S20.3 (SB-2105, SB-2106) · **Status:** Done

**What:** A Mattermost answer can be shared to its channel with a *Share in channel* button (`POST /chat/mattermost/actions`), which posts the citation links only, as Slack's does. The Handover tab finds the person to brief by name or email (`GET /v1/handover/people`), listing only people the caller may brief. **W20 built.**

**Why this way:** Mattermost sends action presses unsigned, so the button's context is signed by the cell with the command token and checked at the press; the asker is mapped again and the trace read with their scope, so no one else can post their answer's cites. The people search runs the brief's own policy decision on each match, so its list can't be wider than what `POST /v1/handover` would allow; the name filter comes first, so only matches are decided and audited.

**Learned:** Mattermost's action requests carry no response URL; the slash command's own, kept in the signed context, posts in the channel without the bot joining it.

**Laws:** LAW-01 (`TestLAW01_MattermostSharesOnlyTheAskersCites`, self-test catches a share carrying claims; `TestLAW01_HandoverSearchListsOnlyWhomTheAskerMayBrief`, self-test catches a search that skips the policy). Every law test is still green.

## 2026-11-23 S20.2: what changed, for you

**Wave/slice:** W20 / S20.2 (SB-2103, SB-2104) · **Status:** Done

**What:** Once a week (the first notify run after Monday 08:00 in the manifest's new `digest` time zone, or never with `digest: off`) each person who asked anything in 90 days gets the titles of what changed in their spaces, as `GET /v1/changes` shows them at that moment. A saved question can be watched (`PUT /v1/saved/{id}/watch`, a checkbox in the Saved tab): the same weekly run searches it again as its owner and says when its sources changed.

**Why this way:** Both run as the person at send time through `wire.Cell.ScopeAs`, which reads their groups from the directory then, so the digest is exactly what they'd see if they opened the page (LAW-01), and the change list already ignores rebuilds (LAW-13). Watching uses search rather than the model: what changed is the sources, and a weekly ask per watched question would load a CPU box; only a hash of the cites is kept. The time zone is the `digest` field itself, so the feature stays one setting.

**Learned:** A fresh cell would otherwise send its first digest on whatever day it's installed; the first run now only marks the week.

**Laws:** LAW-01 (`TestLAW01_TheDigestNamesOnlyWhatThePersonCanOpenNow`) and LAW-13 (`TestLAW13_ARebuildIsNeverInTheDigest`) over the real job, agent, pipeline and index, with documents that change, then drop the person, then a rebuild; the same law checks the watch message names no source and only a hash is kept (LAW-07). The self-test catches a digest that doesn't check as the person and one that lists everything. Every law test is still green.

## 2026-11-23 S20.1: stewards and askers are told

**Wave/slice:** W20 / S20.1 (SB-2101, SB-2102) · **Status:** Done

**What:** A `notify` job in `sb-jobs` (every five minutes) tells each space's stewards how many new questions it has, and each asker that their question has a reply or became a note, as a direct message from the Mattermost or Slack bot the cell already has (`internal/core/notify`, `slackapp.DM`, `mmapp.DM`). Messages are a count, a space's name and a link to the cell's page. `GET|PUT /v1/me/notifications` and checkboxes in the Questions tab turn kinds off; `/?tab=questions` opens the app where the link points.

**Why this way:** Who is told is read from the directory when the job sends, not from the question, so a steward who left the group or was deprovisioned hears nothing (LAW-21, LAW-12), and the message never carries text, so the person reads the question where the cell checks again what they may see (LAW-07). The job decides every message before sending any and keeps only its last run time, so a failed send is counted and alerted but nothing is told twice. Settings are a `pref` cell record, erased with the person and never expired, so an opt-out doesn't quietly lapse after the retention period.

**Learned:** Space labels in `SpaceFacts` are sensitivity labels, not display names, so messages name the space by its manifest name.

**Laws:** LAW-21 (`TestLAW21_NotificationsGoToTheStewardsThereAreNow`) and LAW-07 (`TestLAW07_NotificationsSayNothingOfWhatWasAsked`) over the real job, store, directory and Slack DM adapter against a fake Slack; the self-test catches stewards read once, the deprovisioned told, and a message that previews the question. Every law test is still green.

## 2026-11-23 S19.3: ask from Mattermost

**Wave/slice:** W19 / S19.3 (SB-2005) · **Status:** Done

**What:** A Mattermost custom slash command reaches the cell at `POST /chat/mattermost` (`internal/adapters/chatapp/mmapp`): it checks the command's token from `vault://tenants/TENANT/mattermost` and the tenant's team, maps the Mattermost user by email to one active directory user with the Slack app's own rules, and answers only the asker through the server's response URL, in escaped Markdown. sb-cell(5) `mattermost.base` and `mattermost.team_id` turn it on; the self-hosted runbook says how to set it up. **W19 built.**

**Why this way:** The person mapping and the space-by-channel rule are the Slack app's own methods, exported rather than copied, so the two chat entry points can't drift apart on who gets an answer. Mattermost sends no email, so the cell reads it with a bot token at every request, never caching it, as the Slack path does. The response URL must be the configured server's own `/hooks/commands/`, so a forged command can't make the cell post an answer elsewhere.

**Learned:** The vault accepts only known paths, so a new secret is a change to `secrets.ParseRef`'s list, not just a new file; the running-cell test found it.

**Laws:** LAW-01 (`TestLAW01_MattermostAnswersCiteOnlyWhatTheAskerCanOpen`: the chat law over the real Mattermost app and ask agent), LAW-12 (`TestLAW12_MattermostRefusesTheDeprovisioned`, with a self-test that catches a mapping that answers the deprovisioned). LAW-07's masking is the ask path's, unchanged. Every law test is still green.

## 2026-11-23 S19.2: office files, and notes that stay as narrow as their space

**Wave/slice:** W19 / S19.2 (SB-2003, SB-2004) · **Status:** Done

**What:** The folder connector reads `.docx`, `.pptx`, `.odt` and `.html` files as text, with the standard library only (`folder.OfficeText`): only the parts that hold the document's text are opened, an archive too large or too long is skipped whole, and a fuzz target guards the reader. The stewards' review list now shows notes whose readers include people who left the space, as a count, and **Republish to members only** narrows the note's ACL file to the readers who are still members; the connector rewrites a note's ACL only to fewer readers.

**Why this way:** Opening only `word/document.xml`, the slides and `content.xml` means macros, embedded objects and media are never read rather than read and filtered, which is the only way to be sure. Skipping an oversized archive whole, rather than reading its first 16 MiB, keeps the index from holding half a document that answers would treat as all of it. Pages are scanned, not parsed as XML: `encoding/xml` refused unquoted attributes, which real pages are full of. A note can't follow its space by itself because its ACL is a file; republishing to the intersection of its readers and the members now can only remove people, so the connector can enforce "never wider" without knowing the directory.

**Learned:** The index doesn't keep a document's label when the cell has no spaces routing, so a republish takes the label from the space as the first publish did, and the connector refuses it if the label changed. Share directories made per law run must be unique; reusing one let one run's file leak into the next.

**Laws:** LAW-09 (`TestLAW09_OfficeFilesAreData`: the ingest law over office files, with a canary in every hidden part never reaching the index; a self-test catches a reader that indexes them), LAW-13 (`TestLAW13_OfficeIndexIsDerived`), LAW-19 and LAW-21 (`TestLAW19_NotesStayAsNarrowAsTheirSpace`: the count names no one, republishing takes the note off the list and leaves it readable by exactly the members who could read it and still can; self-tests catch stale members and a republish that writes nothing). Every law test is still green.

## 2026-11-23 S19.1: notes across barriers, and the handover screen

**Wave/slice:** W19 / S19.1 (SB-2001, SB-2002) · **Status:** Done

**What:** In a cell with information barriers, a steward's note is placed by its path before it is written (`barrier.Cell.Target`, `barrier.NoteTarget`): `notes/SPACE/qs_ID.md` is held by the spaces whose scope holds it and owned by the steward, so a steward publishes on their own side of a barrier and a note into a space across it is refused. The web UI gains a **Handover** tab (`web/src/Handover.tsx`): a person's user id in, the cited brief out, grouped by location (the response's new `locations`), with the documents the asker can't open as a count and the advice to have access moved before the person leaves; a refusal shows the policy's reason.

**Why this way:** A note's place is known before it exists, because its path names its space, so it can take the same barrier check as any target instead of being refused in every barrier cell. Placement by path, not by the steward's say-so, means a steward who names another space's notes folder is checked against that space. The brief's grouping comes from the cell, which already sorts by location, rather than from parsing the model's text; `locations` covers only documents the asker can open, so it names nothing the brief couldn't already cite.

**Learned:** The W16 refusal wasn't a special case anywhere: it fell out of `Locate` failing for a file that didn't exist yet, so the fix was a placement for notes, and no refusal had to be removed.

**Laws:** LAW-22 and LAW-04 (`TestLAW22_NotesPublishOnTheirSideOfABarrier`: the real bus, Cedar, barrier placement and folder connector; a note in the steward's own space runs with an `Executing` row, one across the barrier is refused and not written, an unreadable registry refuses; self-tests catch a cell that refuses every note, places a note in no space, or ignores an unreadable registry). LAW-14 on the handover screen through the shared `ClaimView`. Every law test is still green.

## 2026-11-23 S18.3: retire a box with proof

**Wave/slice:** W18 / S18.3 (SB-1907) · **Status:** Done

**What:** `install.sh --retire [CONFIG] --yes-destroy` (`tools/selfhost/retire.sh`) destroys a self-hosted box for good: the systemd units, the containers, every volume, and every file under `build/selfhost`, each overwritten then deleted. It leaves a statement, `sb-retire/v1`, that names what was destroyed and kept, when, and the cell's own signed anchor on its last audit row, signed with a key made for it and printed. New: `internal/core/box/retire.go` (`LastAnchor`, `Retirement.Sign`, `VerifyRetirement`), `sb selfhost retire-statement` and `sb selfhost verify-retirement`. **W18 built.**

**Why this way:** The audit export is taken and checked under the cell's own anchor key before anything is destroyed, so a box whose chain doesn't verify is never retired into a statement that can't be checked; its copy goes off the box sealed to the recovery key, because audit outlives the box (seven years). The statement is signed by a fresh key, not the cell's audit key, which the retirement destroys, and the private half lives only in the process that signs; a statement therefore proves it was made once, by whoever ran the retirement, and the printed key is what the company files with it. The anchor in it is the cell's own signed anchor, so anyone holding the off-box export can check the statement names where the chain really ended.

**Learned:** The demo's notes volume is made outside compose and carries no compose label, so the first live run left it behind; volumes are now found by label and by name, and the script fails if any is left. The demo's `SB_OFFSITE` sits under `build/selfhost`, so the script moves an off-box directory found there out of the way instead of shredding the copies it promises to keep.

**Laws:** LAW-18 and LAW-11 (`TestLAW18_ARetiredBoxLeavesASignedStatement`: the statement names the last row of a chain that verifies under the cell's key and refuses a changed or foreign one; it verifies under its own key only, every statement has a fresh key that isn't the cell's, and any changed field is refused; self-tests catch a check that ignores the signature, an anchor taken from an unpinned export, and a key that is kept). Every law test is still green.

## 2026-11-23 S18.2: a person's data, the retention plan, sign out everywhere, personal data flags

**Wave/slice:** W18 / S18.2 (SB-1903 to SB-1906) · **Status:** Done

**What:** `POST /v1/admin/subjects/export` hands a data-protection officer everything erase would remove for one person, audited with its hash. `GET /v1/admin/retention/plan?days=N` and `sb tenant retention --plan` show what the nightly sweep would delete, by store, with what legal holds keep counted apart. `POST /v1/sessions/revoke` (the web UI's **Sign out everywhere**) and `POST /v1/admin/sessions/revoke` end every session of a person, audited first. Ingest flags `personal_data` on chunks with a valid IBAN or card number; their snippets, the model's context and every answer show only the last four characters, and stewards see how many documents in their space carry the flag. `retention.Store` gained `Plan` and `Export`; `session.Revoker`, `ingest.PIISpans`, `ingest.MaskPII` and `ports.DocQuery.Flag` are new.

**Why this way:** The plan and the export are written beside `Expire` and `Erase` in each store, over the same selection function (the memory store runs one `drop` with a dry flag), so they can't drift from what the job deletes; the law test runs plan then sweep and requires equal counts. The plan's space pass counts only the records between the space's cutoff and the tenant's, because the sweep's first pass has already deleted the older ones. Masking happens where text leaves the index (snippets, the model's context, answer text), not in the index, so the flag stays derived and a rebuild gives the same flags; masking the model's context means an answer can't quote what the model never saw. A card number needs an issuer prefix as well as Luhn, so order and phone numbers are mostly left alone.

**Learned:** The card scan first took the leftmost 19 digits of a run, which missed a card after a short reference number (`Ref 12 4111 …`); it now grows each candidate group by group, at most 19 digits, so a long run of short groups stays linear. Revoking sessions needed no new store: `ports.Sessions.RevokeUser` already existed for SCIM.

**Laws:** LAW-16 (`TestLAW16_RetentionPlanIsWhatTheSweepDeletes`: the plan's counts equal the sweep's per store, holds apart, and the plan deletes nothing; self-tests for a plan that ignores holds, deletes, or uses its own cutoff), LAW-07 and LAW-04 (`TestLAW07_SubjectExportIsWhatEraseRemoves`: audited first, hash not text, nobody else's records, export equals erase, empty after; `TestLAW07_PersonalDataIsMasked`: no reader is shown more than the last four characters, and a rebuild gives the same flags, LAW-13), LAW-12 (`TestLAW12_SignOutEverywhereEndsEverySession`: through the real middleware, every session of the person refused, no one else's, nothing revoked when the audit log is down).

## 2026-11-23 S18.1: off-box copies sealed, and the keys in a 3-of-5 kit

**Wave/slice:** W18 / S18.1 (SB-1901, SB-1902) · **Status:** Done

**What:** Everything a self-hosted box copies to `SB_OFFSITE` is sealed to a recovery key: the nightly dump (`sb backup seal`) and the backup job's off-box copy (`backups[].seal`, `offsite.Target`). `install.sh --recovery-kit` seals an archive of the box's keys under a kit key, splits that key into five share pages (any three rebuild it), copies the sealed archive off the box and shreds the recovery key's private half; `restore.sh --from-kit` rebuilds the keys on a new machine and `restore.sh --dump X.sealed` opens a sealed dump. New: `internal/core/offsite`, `internal/core/shamir`, `sb backup keygen|seal|open`, `sb recovery split|join`, and two doctor checks.

**Why this way:** The shares split a 32-byte kit key, not the archive, so each is one page a person can type back in; the archive is sealed with that key and kept off the box, which is no weaker, since without three shares the archive is noise. Sealing uses an ephemeral X25519 key and HKDF per file and authenticated 64 KiB chunks, so a multi-gigabyte dump streams and any change, truncation or reordering fails; all of it is the Go 1.24 standard library (`crypto/ecdh`, `crypto/hkdf`), so no module was added. Only the off-box copy is sealed: the copies on the box and in the write-once store are what the drill and a same-machine restore read, and they never leave the box.

**Learned:** An install that writes a configuration field the admitted release doesn't know stops the cell, and the old updater can't upgrade past it (BUG-0029): the install now starts its own images when that happens and keeps the old pins for one rollback. `internal/core/seal` already existed (per-space data keys), so the new package is `offsite`. Shamir with k-1 shares returns bytes, not an error, so the kit's record carries a 4-byte key check to say "wrong shares" rather than "changed archive".

**Laws:** LAW-11 (what sits off the box reveals nothing, opens only with the recovery key, and not at all after any change: `TestLAW11_OffBoxCopiesAreSealed`, with a self-test that catches an unauthenticated cipher), LAW-18 (any three of five shares rebuild the kit key, two don't, and no share holds the key as it is: `TestLAW18_AnyThreeSharesAndNoTwo`, with a self-test that catches a 2-of-5 kit); LAW-03's box test now covers the drill and kit facts.

## 2026-11-23 S17.3: a proxy, the company's Postgres, and a support bundle with no secret

**Wave/slice:** W17 / S17.3 (SB-1806 to SB-1808) · **Status:** Done

**What:** The four services with a route out take the company's proxy (`SB_HTTPS_PROXY`, or `SB_HTTPS_PROXY_FILE` for one with credentials, read by each service's `main` from its own vault before any HTTP client); `SB_DATABASE=external` moves the cell onto the company's Postgres over `sslmode=verify-full` and takes the bundled one out of the layout; `install.sh --support` writes a tar of versions, settings, the doctor, `ps` and logs that `sb secrets scan` has checked against every stored secret. `sb selfhost check-proxy` and `check-dsn` check the settings in Go. `dbdump.sh` and `restore.sh` share `pg.sh`, and the drill now restores into a container with no network for both layouts. W17 is built.

**Why this way:** Credentials never go into `install.env` or a container's environment, because both are read by more people and tools than the vault is: the proxy's URL goes from a file into the vault, and the database password reaches `pg_dump` through a `pg_service.conf`, not its arguments, which any local user can read in `/proc`. The bundled database is a compose profile the wrapper adds by default, so a box installed before this change keeps its Postgres without being reinstalled. The support bundle is scanned for the values the box really stores, not only for patterns, because a stored secret has no shape; it learns a JSON credential's secret field on its own, since a log line would carry the secret, not the object.

**Learned:** A vault change reaches a running container only when it starts again, so an install that changes one now recreates the services; before, moving back from an external database left the services on the old string. libpq's channel binding can't use an Ed25519 server certificate (`could not find digest for NID UNDEF`), so the runbook's database certificate should be RSA or ECDSA. The smoke test had two bugs that only Docker Desktop hid (BUG-0028).

**Laws:** LAW-07 (a support bundle carries no stored secret, whole or in part: `TestLAW07_SupportBundlesCarryNoSecret`; a proxy's credentials never sit in a setting that isn't secret nor in the error that refuses them: `TestLAW07_ProxyCredentialsStayInAFile`), LAW-02 (a database on another machine only over TLS that checks its name: `TestLAW02_ExternalDatabaseIsVerified`), each with a self-test.

## 2026-11-23 S17.2 Upgrade and recover: restore, a monthly drill, verified offline upgrades and rollback (SB-1803 to SB-1805)

**Wave/slice:** W17 / S17.2 · **Status:** Done.

**What:** `tools/selfhost/restore.sh` (one-command restore, refusing a dump made under other keys, and `--drill` with `sb-selfhost-drill.timer`); a record beside each dump; `update.sh --bundle FILE` with `sb selfhost verify-bundle` and `release.VerifyImageTar`; `update.sh --rollback` and `--resume` with `previous.env` and a hold; the doctor's `restore drill` and `upgrades` checks; `SB_BUNDLE_RELEASES` for `make selfhost-bundle`; BUG-0027 fixed.

**Why this way:** Each step that used to be a page of the runbook is now one command that refuses before it changes anything. An offline bundle is checked by the sb already on the box, never the bundle's own, against the key in the cell's own configuration, and an image archive is accepted only if every blob hashes to its name and the index names exactly the release's image, which is the same content-addressing Docker relies on, checked before Docker sees it. A rollback can only go to digests the cell attested before, one step back, and holds the timer so it stays put.

**Learned:** This environment can run Docker, so every one of these ran against a live demo box: an upgrade from a bundle, the same bundle with ten bytes of a layer changed (refused, nothing loaded), a rollback, a held timer run, resume, a drill and a real restore. The restore found BUG-0027: a new cell's audit export is a start anchor with no rows, so the tenant couldn't be read from it. With Docker's containerd image store an image's id is its manifest digest, which is what a release records, so the archive's own index can be checked against the release directly.

**Laws:** LAW-06 (an offline bundle loads only the admitted release's exact images: `TestLAW06_OfflineBundlesLoadOnlyTheReleasesImages`, with a self-test), LAW-08 (one step back, never two), LAW-11 (a restore verifies the chain), LAW-13 (the index rebuilds, never restored).

## 2026-11-23 S17.1 Know it's healthy: doctor and the box's health (SB-1801, SB-1802)

**Wave/slice:** W17 / S17.1 · **Status:** Done.

**What:** `tools/selfhost/facts.sh` gathers a self-hosted box's facts (services, timers, disks, certificate expiry, the off-box directory, backup times, `/readyz`, disk encryption) as one JSON document; `internal/core/box` judges them; `sb selfhost doctor` and `install.sh --doctor` print each check with its fix; the hourly host step leaves the facts in `state/host/facts.json`; `GET /v1/admin/health` adds `box`; the drift job raises a `box_check` Sev-2 once a day per failing check; the console's health panel gains a Box section; the bundle carries `bin/sb`.

**Why this way:** The script gathers and Go judges, so every threshold has a test that makes it fail and the doctor and the console can never disagree. Facts are a file in the cell's own state, written by the host step that already runs hourly, so the cell gets the machine's view without a socket, an agent or a new service. A missing or old fact is Unknown, and the doctor exits 1 on it, because a check that couldn't run is the one most likely to be hiding the problem.

**Learned:** Run against the demo box in this environment, the first draft failed `/readyz` because it answers 204, not 200, and correctly flagged the demo's stand-in off-box directory as on the root disk. A once-key in the usage ledger must be lower-case letters, digits and dashes, so the check's name is hashed into it.

**Laws:** LAW-03 (a fact the host step couldn't gather, or facts over two hours old, are never healthy: `TestLAW03_BoxNeverShowsAGapAsHealthy`, with a self-test), LAW-07 (the facts hold no secret).

## 2026-11-22 W17 to W19 plan accepted: run it, protect it, close the loops

**Wave/slice:** W17 to W19 (planning) · **Status:** Accepted by the user, with every recommended decision.

**What:** `docs/10-run-protect-close.md` and epics E18 to E20: twenty small features in eight slices (SB-1801 to SB-1808, SB-1901 to SB-1907, SB-2001 to SB-2005). W17 makes the self-hosted box easy to run (doctor, host health, one-command restore and drill, verified offline upgrades, rollback, an outbound proxy, the company's own Postgres, a support bundle). W18 protects the data (sealed off-box copies, a 3-of-5 recovery kit, subject export, personal data flags, a retention preview, sign out everywhere, retiring a box with proof). W19 closes W16's gaps (notes with barriers, the handover brief in the UI, Word files on shares, notes that stay narrow, Mattermost).

**Why this way:** Reading the selfhost runbook and the scripts at `b6ed548`, most of what's left is the operator's day: an offline upgrade loads images before their signature is checked, the nightly dump goes off the box in plain text, and the keys exist only on the box. Each fix is a script flag or a few endpoints on records the cell keeps, under W16's bloat rules.

**Learned:** The committed `site/` lags the sources because `make docs` rebuilds it locally; commits keep to their own paths, as CLAUDE.md says.

**Laws:** none changed; the tickets name the laws each will test.

## 2026-11-22 SB-1710 run for real: a 7B model and an embedder on the box, gated

**Wave/slice:** W16 / S16.5 · **Status:** Done. **W16 built**, every ticket Done.

**What:** with the user's approval, llama.cpp's server image pinned by digest, Qwen2.5-7B-Instruct Q4_K_M and nomic-embed-text-v1.5 Q8_0 fetched and checked against Hugging Face's published SHA-256s, `make selfhost-model` run until the gate passed, and `make selfhost-smoke` green with the model profile. Fixes the run found: the Python eval CLI takes `openaicompat=BASE#MODEL`, `model.sh` runs the template suites the gate requires, `update.sh` keeps the model image pin, the models volume carries Compose's labels, and the handover prompt requires one claim per document. The smoke run now prints the answer latency it measures.

**Why this way:** The gate did its job before anything was configured: it refused the model when a handover brief skipped a document the person owned, and refused it again when the script hadn't run every suite the gate names. The fix for the first was the prompt, which ships in the release, not the test; the fix for the second was the script.

**Learned:** 32 minutes for the gate and a median of 15 seconds an answer on 16 CPU cores: a local 7B model is right for sovereignty and demos, and search stays the fast path. A volume one script creates and Compose then uses should carry Compose's labels, or every compose call warns.

**Laws:** LAW-06 (the exact weights are hashed into the release and gated), LAW-07 (the model servers hold no secret and have no route out), LAW-13 (the index was rebuilt from the share for the new embedder).

## 2026-11-22 S16.5 On the box: a local model and embedder, pinned and gated (SB-1710)

**Wave/slice:** W16 / S16.5 · **Status:** In review (the run needs model weights the user fetches). **W16 built**: SB-1701 to SB-1709 Done.

**What:** `embed` in sb-cell(5) with an `openaicompat` embedder, `model.local`, an embedder change that rebuilds the index, the bundle's `weights_sha256` and `embedder` pins, the compose profile `model` (two llama.cpp servers from one pinned image), `make selfhost-model`, model weights in the offline bundle, and smoke checks for the profile.

**Why this way:** The cell already spoke the OpenAI API, so the whole feature is configuration, an embedder adapter and a script, with no new service the cell must run. llama.cpp's server serves chat or embeddings well, not both from one process, so the profile runs the same pinned image twice, one model each; that keeps one binary and one digest to trust. The gate runs on the exact weights before the cell is ever pointed at them, and the switch is written only after a pass, so "a model that fails the gate doesn't run" is the order of the script, not a hope. An embedder change is the one change the index can't absorb in place, so the ingest service treats it as a rebuild from the sources, the law's own remedy (LAW-13).

**Learned:** A record of what built the index has to live outside the index, or dropping the index forgets why it was dropped; it lives under the cell's state. A cell that predates the record is treated as built by the hashing embedder, which it was.

**Laws:** LAW-02 (one model server per cell, on its own network), LAW-06 (the model's weights are part of the release), LAW-07 (the model servers hold no secret), LAW-13 (a new embedder rebuilds from the sources).

## 2026-11-22 S16.4 Keep it current: what changed, and documents to review (SB-1708, SB-1709)

**Wave/slice:** W16 / S16.4 · **Status:** Done.

**What:** first-seen times carried across rebuilds and a `rebuilt` mark for the ones that can't be, `DocQuery.SeenSince`, `GET /v1/changes`, What changed this week in the UI and a count per space in the weekly report; `review_after_days`, `GET /v1/spaces/NAME/review`, `age_days` and `stale` on cites, and the steward screen's Documents to review.

**Why this way:** The index is derived (LAW-13), so first-seen times are a convenience it may lose. A rebuild keeps them where it can and marks where it can't, and the one reader that would be fooled by a lost time, "what changed", never reads a marked version: a rebuild can make the list shorter, never longer. The review list counts from the traces and feedback the cell already keeps, so it adds no store and no tracking of people; it checks each document as the steward, and a document the steward can't open is reduced to its owner, which is exactly what they need to ask someone to review it.

**Learned:** A rebuild that only drops and replays looks like a fresh index to everything downstream; carrying the times is one map read before the drop. A space field whose default is written out would break the manifest's read-back property (FuzzCheck), so the default is stored as 0 and never printed.

**Laws:** LAW-01 (changes and the review list only what the reader can open), LAW-13 (first-seen times are derived; a rebuild marks, never invents, changes), LAW-14 (age shown on cites, never hidden), LAW-16 (the threshold is in the manifest).

## 2026-11-22 S16.3 Onboard and hand over: start here and a handover brief (SB-1706, SB-1707)

**Wave/slice:** W16 / S16.3 · **Status:** Done.

**What:** `start_here` in a space, `GET /v1/start`, a Start here panel, and verify's `START` check; `handover` with `POST /v1/handover`, the `brief_handover` policy action and an eval suite in the gate. Underneath: chunks carry their location, and `ports.DocLookup` finds documents by source, folder or owner in both indexes.

**Why this way:** Both features are lists of documents chosen by something other than a query, so both end in `retrieve.Allowed`, the same checks a search ends with; neither can show what an ask couldn't. A start list names its items in the manifest, so changing it is the steward-proposal-and-operator-approval path every space change already takes (LAW-16), and verify finds a list nobody can read before a new joiner does. The handover brief's "who may ask" is a policy decision over relations the cell derives from the directory at the request, so a tenant policy can narrow it and the default deny refuses everyone else. It works on the person's documents with the asker's access, never the person's, which is what keeps it on the right side of deprovisioning (LAW-12).

**Learned:** A brief grouped by location needed the location on the chunk; it was already on the document for routing, so it costs one column and keeps the index derived (LAW-13). The verify target needed a member's scope in the space, which the issuer's `IssueIn` already gives.

**Laws:** LAW-01 (only what the member or asker can open), LAW-03 (Unknown or late counted, never shown), LAW-09 (no tools), LAW-12 (the leaver's documents, never their access), LAW-14 (every line of a brief cited), LAW-16 (the list is in the manifest).

## 2026-11-22 S16.2 Ask and capture: questions to stewards, replies become notes (SB-1704, SB-1705)

**Wave/slice:** W16 / S16.2 · **Status:** Done.

**What:** `ask.Stewards` (send, the asker's list, the steward's queue, reply, note), `questions_to` and `notes` in the manifest, the Cedar action `answer_questions` and a forbid on `note-publish` for non-stewards, `toolbus.Call.Context`, `retrieve.Allowed`, `ports.CiteLookup` in both indexes, the folder connector's note writer, the self-hosted `notes` mount, and the UI's Ask-the-stewards, Questions tab and steward queue.

**Why this way:** A reply's cites are checked for the asker every time they look, through the same function retrieval ends with, so a steward's quote can never widen what the asker reads and a later loss of access shows at once. The steward rule is a policy decision on the caller's groups at that request, so moving out of the stewards group needs no cleanup. The note is the steward's own text, written by the one tool a person calls; the user's decision makes the steward's publish its approval, so the tool runs at once like an ordinary tool call, with the payload hash in the audit row before the write. The connector opens only `notes/` for writing and creates files exclusively, so it can't touch the share or overwrite a note, and the receipt is the hash of what it read back. Readers are resolved to emails because a folder ACL can't evaluate groups; that's why they are a snapshot of the members at publish time and can only narrow.

**Learned:** In a cell with information barriers the bus places a call's target before it runs, and a note that doesn't exist yet can't be placed, so notes are refused there for now (documented, safe). Law tests that need an unanswered ask are steadier with a trace written directly than with a query a keyword index might still match.

**Laws:** LAW-01 (cites re-checked for the asker), LAW-04 (audit before store), LAW-05 (Ran only with the note's hash as read back), LAW-07 (hashes, never text, in audit), LAW-09 (no model writes a note), LAW-13 (notes are source documents the next sync indexes), LAW-15 (the publish binds the note's hash), LAW-21 (a moved steward loses the queue).

## 2026-11-22 S16.1 Find: search without the model, who to ask, saved and shared questions (SB-1701 to SB-1703)

**Wave/slice:** W16 / S16.1 · **Status:** Done.

**What:** `GET /v1/search` (the ask path's retrieval and live check, stopped before the composer), owners and `people` on answers from the directory's active users, and saved and shared questions (`/v1/saved`, `/v1/shares`, `/q/sh_…`). Chunks carry their title and when the index first saw their version. The web UI became one page with Answer, Documents and Saved tabs; the Slack reply names who to ask.

**Why this way:** Search is a second exit from the one retrieval function, `retrieve.ContextUpTo`, so it can't drift from what an answer is allowed to use; only the budget of chunks differs. Owners are shown only on cites, which are only ever allowed chunks, so the rule "never name the owner of a document you can't open" holds by construction rather than by a filter. A share holds the question and nothing else, and the recipient asks it: the Slack Share button's rule, now for the web. Share ids come from crypto/rand because the cell's other ids are time-ordered and guessable. Saved questions and shares live in the trace stores as two more record kinds, so retention, holds and erasure cover them with no new store.

**Learned:** In the running cell a refused document usually doesn't reach the live check at all: the prefilter drops it, so `refused` stays 0 for someone with no access. Tests assert on what is shown, not on the refused count. A core test may not import an adapter even for a convenient store; the store contract is tested in the adapter.

**Laws:** LAW-01 (search and owners only from allowed documents), LAW-02 (shares stay in their tenant), LAW-03 (Unknown or late never shown), LAW-07 (the query and the shared question are hashed in traces and audit), LAW-12 (a deprovisioned owner is never suggested), LAW-14 (a snippet is a quote with its cite).

## 2026-11-22 W16 accepted

**Wave/slice:** W16 (plan) · **Status:** Planned

**What:** the user accepted all of W16 and the recommended option for each decision: a steward's publish is a note's approval, bound to its hash (SB-1705); llama.cpp's server for the local model (SB-1710); a handover brief for the person's manager, a steward of one of their spaces, or the person (SB-1707). Tickets SB-1701 to SB-1710 are Open.

**Why this way:** each recommendation was the smaller of the two options that keeps the laws as they are.

**Learned:** nothing new.

**Laws:** no change.

## 2026-11-22 W16 proposed: team knowledge

**Wave/slice:** W16 (plan) · **Status:** Proposed, for the user's review

**What:** [09-team-knowledge.md](../docs/09-team-knowledge.md) and epic E17: ten tickets in five slices (SB-1701 to SB-1710). Search without the model, owners on citations, saved and shared questions; questions to stewards and replies published as notes; a start list per space and a handover brief; what changed and documents to review; and an optional local model and embedder on the box. The PRD's wave list and the plan index show W16 as proposed.

**Why this way:** After W15 the cell answers, but an answer ends at the answer: nothing routes an unanswered question to the person who knows, keeps a good answer, or hands knowledge on when people join or leave. Each feature is a second use of something the cell already has (the retrieval path, the owner on every chunk, traces, the folder connector, stewards, the tool bus), so none needs a new service, store or module, and each keeps the source ACL as the authority by checking every item live for the person who sees it.

**Learned:** Ownership at the source is enough to answer "who knows"; an expertise profile built from activity would be monitoring and a works-council question (Q-12).

**Laws:** no change; each ticket names the laws it must keep green.

## 2026-11-22 S15.4 Cell health, a budget warning and the unhelpful-answer queue (SB-1607 to SB-1609)

**Wave/slice:** W15 / S15.4 · **Status:** Done. **W15 built**; SB-1605's real-VM run is in review.

**What:** `internal/core/health` and `GET /v1/admin/health`, from the job records and a new per-connector sync record the ingest service writes; the budget guard's `budget_warning` at 80%, once a month; the weekly report posted to a webhook; `ask.Queue`, `ask.MarkReviewed`, `GET /v1/admin/feedback` and `POST /v1/admin/feedback/TRACE/reviewed`; the console's health panel and queue, first on the page. The demo's user is now its operator, so the console can be tried.

**Why this way:** Health reads the same records the watchdog and the ingest service already keep, so the panel can't disagree with the alerts, and it refuses to draw a report from a record it can't read rather than show green. The warning's once-a-month guarantee lives in the usage ledger as an exclusive create, because the api, ingest and jobs services each meter calls and any of them can be first. The queue shows what an operator needs to triage, counts and the asker's own words, and counts the retrieval decisions instead of listing their cites, because a refused cite names a document the operator may not be allowed to see.

**Learned:** Every admin call writes its own policy decision row, so a test that counts audit rows to find "one new row" counts the guard's decision too; tests now look for the row by its subject. A console section that one older cell can't answer shouldn't blank the others, so health and the queue load on their own.

**Laws:** LAW-03 (health never shows an unreadable record as healthy), LAW-13 (stale connectors are visible), LAW-14 (the feedback loop on uncited and refused answers reaches someone), LAW-16 (the report and the warning come from the cell's own records).

## 2026-11-21 S15.3 A sovereign install and a folder connector (SB-1605, SB-1606)

**Wave/slice:** W15 / S15.3 · **Status:** SB-1606 Done, SB-1605 In review (a real VM run)

**What:** the `docs: folder` connector with ACL files and PDF text, and an example share for the demo; one setup path for every self-hosted cell (`selfhost.sh` with or without `SB_CONFIG`), `install.sh` for a Linux VM, an offline install bundle, a nightly database dump, and the systemd units; the bundled IdP, sb-idp(1); the monthly access statement; the self-hosted runbook. The demo now answers from documents and signs in with a password.

**Why this way:** A file share has no permission API, so the permission lives beside the file in a plain file the company's own admins can read and review, and a file without one is never shown: Unknown, the same as any connector that can't answer. Change detection keeps its listings by content hash so a sync restarted from any recent cursor sees the same changes, which is what the contract kit asks of every connector. The bundled IdP exists because the cell's sign-in rules (a `tenant` claim, `sub` the directory id) can't be met by an off-the-shelf IdP without per-site setup, and an air-gapped site has nothing else; it is small, does one tenant and one client, and provisions users over the same SCIM a company IdP would, so the directory stays the one source of who exists. On one machine there is no IAM to enumerate, so LAW-17 becomes a list the cell keeps of the factory's paths, checked monthly, with each check's statement anchored into the write-once store.

**Learned:** Every service builds the whole cell, sign-in included, so ingest and jobs must reach the cell's own public name too; the proxy now answers that name on both inner networks. A sign-in limit counted per client address counts the proxy, and so locks everyone out at once; behind the proxy the IdP counts the address the proxy saw. Files a container writes into a Windows folder can't be read or removed from WSL, so the demo's clean removes its stand-in off-box copy through a container, and the smoke checks read it through one.

**Laws:** LAW-03 (no ACL file, a slow share or a bad ACL is never Allow; an unknown or disabled user never signs in), LAW-13 (the folder index rebuilds to the same entries), LAW-01 (verify green on the demo), LAW-12 (disabling a bundled user deprovisions them), LAW-17 (the signed access statement), LAW-07 (password hashes only; every key made and kept on the box).

## 2026-11-21 S15.2 Write-once storage and upgrades on the box (SB-1603, SB-1604)

**Wave/slice:** W15 / S15.2 · **Status:** Done

**What:** `internal/adapters/backup/s3`, a write-once target over the S3 API with SigV4 in the standard library, and its fake `s3test`; `worm` in sb-cell(5), used by the `anchor` and `backup` jobs; versitygw and a one-shot `wormsetup` in the compose layout; the `updater` as a one-shot container, the host step `tools/selfhost/update.sh` with `make selfhost-update` and systemd units, a demo release signed with the machine's own key (`tools/selfhost/devrelease`), and the `local` deployer reporting what the host started. Nine new smoke checks (24 in all) and the law test `TestLAW11_AnchorsAreWriteOnceOnTheBox`.

**Why this way:** Object lock in compliance mode is the one thing an S3 store offers that nobody, root included, can lift, so it is the box's equivalent of a locked retention policy; the cell checks the lock before every write and proves at install, with a probe it can't delete, that the store really enforces it, because a configuration can claim a lock a store doesn't keep. Upgrades are split so that the part that reads releases can't start anything and the part that starts containers decides nothing: the updater only admits and records, and a small host script starts exactly what was recorded, after checking it is a digest the cell attested.

**Learned:** MinIO's published images can no longer be pulled, so the store is versitygw, which keeps object versions and enforces compliance retention on a plain directory. Volumes used only by a compose profile aren't made by `compose create`, and one made by hand without compose's labels draws a warning on every run, so `make selfhost` makes them with those labels.

**Laws:** LAW-11 (anchors and the write-once copy can't be overwritten or deleted for seven years, and a weak or unenforced lock is refused), LAW-06 (only admitted, attested digests run; an unadmitted release changes nothing and raises a Sev-1).

## 2026-11-21 S15.1 Container images and a cell in containers (SB-1601, SB-1602)

**Wave/slice:** W15 / S15.1 · **Status:** Done

**What:** one distroless, non-root, read-only image per service with reproducible digests; `deploy/selfhost/compose.yaml`, which lays a cell out the way GCP does (a TLS proxy as the only door, each service with its own vault, Postgres on a network with no way out); `make selfhost`, `make selfhost-smoke`, and a CI job running both; the database DSN read from the vault.

**Why this way:** The same images serve Cloud Run and the box, so a cell tested on a laptop is the cell that ships. Splitting secrets by volume does on one machine what separate service accounts do on GCP: `ingest` can't read the sign-in secret because it doesn't have it. The smoke checks look from outside, as an auditor would, rather than trusting the compose file.

**Learned:** The first real cell with its UI configured couldn't start at all (BUG-0026): no test had ever set `web`. Image digests differed between two builds of one commit until BuildKit's provenance attestation, which stamps the build's own time, was turned off; the files' and image's dates now come from the commit. Docker Desktop's engine is reached from WSL through its Windows `docker.exe`, so the scripts take `DOCKER` and `CURL`.

**Laws:** LAW-07 (each service holds only its own secrets, closed to everyone else), LAW-02 (one machine, one tenant), LAW-03 (BUG-0026: a part that can't be built stops the cell, and now the cell can be built).

## 2026-11-21 W15 planned: the self-hosted cell

**Wave/slice:** W15 (plan) · **Status:** Planned

**What:** ADR-0022 (proposed) and epic E16: nine tickets in four slices (SB-1601 to SB-1609). Container images for every service and a cell in containers that mirror how GCP runs one; write-once audit storage on the box and upgrades without the Docker socket; a sovereign install on a VM or an on-prem server, and a folder connector; and four easy wins (cell health, a budget warning with report delivery, and a queue of unhelpful answers). The PRD's wave list now shows W13 Hardening, W14 Cell Runtime and W15, with the portfolio grants parked.

**Why this way:** W14 left a whole cell running on one machine with Postgres, so the shortest path to a company that keeps its data on its own hardware, and to a cell for demos and testing, is the same cell in containers that mirror GCP rather than a second runtime.

**Learned:** The site's wave list is built from the PRD's wave table, so a wave planned only in the plan index doesn't show on the home page; the PRD has to change with the plan.

**Laws:** no change; each ticket names the laws it must keep green.

## 2026-11-21 S14.5 Console screens for stewards and access reviews (SB-1509)

**Wave/slice:** W14 / S14.5 · **Status:** Done

**What:** the steward's screen for a space (propose a change, decide access reviews) and two new sections of the admin console (open proposals approved by their hash, every review overdue first).

**Why this way:** Both screens are clients of APIs that already enforce the rules, so the screens add no power of their own: an approval sends back the hash on screen, exactly as the approver view does for actions, and a decision the cell refuses (your own access) is shown, not hidden. A removal asks for its reason before it is sent, because the audit row needs one.

**Learned:** Two buttons with the same accessible name ("Remove usr_bo") made the test, and a screen reader, ambiguous; the confirming button now says what it confirms.

**Laws:** LAW-15 (an operator approves a proposal only by its hash; nobody decides on their own access), LAW-16 (nothing changes until the approved manifest is applied).

## 2026-11-21 S14.4 A whole cell on Postgres, end to end (SB-1508)

**Wave/slice:** W14 / S14.4 · **Status:** In review (the GCP path waits for a project)

**What:** Postgres adapters for every store that was in memory, the cell wired to them with `database`, the verifier's checks run against the cell itself, `make cell` with a development IdP, and an end-to-end test of a durable cell that also restarts it.

**Why this way:** The records are JSON documents keyed by tenant, kind and id, with the time, user and space the queries need beside them, so one small helper serves nine stores, and the decisions that must not race (an action's state, a proposal or review decided) lock their row in a transaction. The verifier asks through a copy of the agent, so LAW-03's injected timeout can't touch anyone's real answers.

**Learned:** The first test to provision its users through SCIM found that legal holds and erasure refused every SCIM id (BUG-0025, PT-0015): every earlier test had made its users by hand. It also found that the services, started together, raced to create the same tables, and that a test helper shared one cookie jar between three users. The local vault refuses secrets on a Windows drive mounted in WSL, where every file is 0777, so `make cell` keeps its files in `~/.sb-cell`.

**Laws:** LAW-01, LAW-03 and LAW-14 through `sb tenant verify --cell` on a durable cell, LAW-02 with the control plane's foreign ids, LAW-13 (a hold placed by a legal steward only, BUG-0025), LAW-15 (the requester can't approve), LAW-11 (the audit log survives a restart).

## 2026-11-21 S14.4 The CLI against a running cell (SB-1507)

**Wave/slice:** W14 / S14.4 · **Status:** Done

**What:** the last `ENOTIMPL` commands: `sb audit verify --tenant`, `tail` and `export` against a running cell; `sb eval diff`; `sb factory handoff` and `detach` on the local control plane, rehearsed end to end against a running local cell; three control-plane routes in the cell (audit rows, the signed export, the detach package).

**Why this way:** Each command reuses what was already proven offline: verify reads the cell's export with the `--export` code, detach writes the package with `export.Write` and refuses to detach until `export.Verify` passes, and the hand-off runs the same controller the game day checked against `Handoff.tla`. Only the target is new, and locally it is honest about what it is: a second local account and a moved copy of the chain, with the running cell serving throughout.

**Learned:** The first time a hand-off state went through a real store, the first approval crashed: JSON drops an empty map, and every earlier test had used the in-memory store (BUG-0024). `factory status` also reported the tenant's own wrap as broken after a hand-off, because it only knew the factory's keys.

**Laws:** LAW-11 (exports verify before they are written), LAW-17 (only retained paths remain after a hand-off, none after detach), LAW-18 (a cell's key always has a working wrap, the tenant's after hand-off).

## 2026-11-21 S14.3 The updater, and apply against a running cell (SB-1505, SB-1506)

**Wave/slice:** W14 / S14.3 · **Status:** Done

**What:** `cmd/sb-updater` with channel pointers in both release stores, a local deployer that refuses unattested images, the Cloud Run deployer with a metadata-server token, and a Sev-1 on any release that isn't admitted; the control plane's identity in the cell, guarding the verifier, `POST /v1/cell/manifest` and `GET /v1/cell/live`; `wire.Cell.Apply`, which swaps every manifest-derived part in whole; `sb tenant apply --cell`; the hourly `drift` job.

**Why this way:** Most of a cell's parts were built from the manifest by value, so changing the manifest in place would leave some parts on the old one. Splitting the composition root into what is built once (stores, sign-in) and what comes from the manifest (`Parts`) lets an apply build the new set completely, refuse before anything changes, and swap it in with one atomic store. The updater needs only files, so unlike ingest and jobs it runs on its own today.

**Learned:** Wiring the verifier found two nil functions that would have panicked once a caller had the right identity (attestation and the checks), now `EUNAVAIL`; an Identity Platform cell's `groups` were never applied at start; and three more vault paths the manual named weren't allowed by the reference parser.

**Laws:** LAW-06 (an unsigned or ungated release changes nothing, with a Sev-1), LAW-13 (apply never removes a held space), LAW-16 (the running cell runs exactly its applied manifest, audited; drift is reported, never fixed silently).

## 2026-11-21 S14.2 The ingest and jobs services (SB-1503, SB-1504)

**Wave/slice:** W14 / S14.2 · **Status:** Done

**What:** `internal/cell/ingestsvc` (syncs from stored cursors, re-route first, per-connector backoff, the `ingest_stale` alert) and `internal/cell/jobsvc` (every scheduled job, finishes recorded, the watchdog, a local clock and `sb-jobs run JOB`); `cmd/sb-ingest`, `cmd/sb-jobs` and `sb-api --with ingest,jobs`; `schedule.Due`; file adapters for job finishes and a usage ledger; the budget guard, token metering, the weekly report and the cost statement wired; legal holds and reporting lines wired once for every service (PT-0011 fixed, BUG-0023).

**Why this way:** The services share the cell's stores, and most stores are still in memory, so a separate ingest process would fill an index the api never reads. Rather than pretend, the separate binaries refuse to start until the stores are durable (SB-1508), and the api runs ingest and jobs in its own process meanwhile. A job records a finish only when it succeeds, so the watchdog treats a failing job exactly like a missed one; there is no third state to forget about.

**Learned:** Wiring found three more "nil means none" gaps beside PT-0011: the gateway had no budget guard and metered nothing, draft outcomes were counted but not kept, and the report had no source. All three are now one usage ledger. Two vault paths the manual already used (`connectors/docs-classification`) or needed (`audit/anchor-seed`) weren't allowed by the reference parser.

**Laws:** LAW-13 (`TestLAW13_IngestThroughARunningCell`, `TestRetentionKeepsHeldRecords`), LAW-01 (a scope change at the source stops answers through a running cell), LAW-11 (anchors write-once, backups 3-2-1), LAW-16 and LAW-17 (the review and access jobs run and their misses are Sev-1).

## 2026-11-21 S14.1 One configuration, and the api service (SB-1501, SB-1502)

**Wave/slice:** W14 / S14.1 · **Status:** Done

**What:** sb-cell(5), a cell's own configuration, parsed strictly by `internal/cell/wire`; `wire.Build`, the one composition root that builds every part of a cell; `cmd/sb-api`, which serves every endpoint from it with health, readiness and graceful shutdown; `tests/cell`, which signs in through a test IdP and sends one request of each kind.

**Why this way:** Every part was already built and tested on its own, and the eval cell and the law tests each wired their own subset. One composition root means a running cell is built the same way in every service, and a part that can't be built stops the cell rather than leaving it running without that part. The in-memory stores are named in the manual and `durable: true` refuses them, so nobody runs a cell they think is durable when it isn't. The Postgres stores are wired with `make cell` (SB-1508).

**Learned:** Wiring found what unit tests couldn't: four routes the manual didn't name (`/login`, `/callback`, `/readyz`, the Slack events endpoint), a chat app that needs a workspace id no file held, and the issuer rule (SB-108), which needed `auth.WithIssuer` so the composition root never makes an Issuer itself. The report endpoints had no usage source in a running cell, so they answer `EUNAVAIL` until the jobs service keeps one, rather than zeros.

**Laws:** LAW-02 (one configuration, one tenant: another tenant's manifest refuses), LAW-03 (nothing that can't be built is skipped: `TestRefusesToStart`), LAW-16 (the cell runs its checked manifest), LAW-01 and LAW-12 through the real sign-in and middleware.

## 2026-11-20 W14 planned: the cell runtime

**Wave/slice:** W14 (plan) · **Status:** Planned

**What:** epic E15, nine tickets in five slices (SB-1501 to SB-1509): one cell configuration and composition root, the `api`, `ingest`, `jobs` and `updater` services, `sb tenant apply` against a running cell, the CLI commands that need one, a whole cell running locally end to end, and the console screens for stewards and access reviews.

**Why this way:** after W13 every part of a cell was built and tested on its own; the last piece between "built and tested" and "deployable" is running them together as real services from one configuration.

**Learned:** (recorded after the fact, 2026-11-21) this entry was missing from the devlog when the wave was planned; the plan itself is in the commit "W14 plan: the cell runtime".

**Laws:** no change; each ticket names the laws it must keep green.

## 2026-11-20 S13.6 Quick fixes after hardening

**Wave/slice:** W13 / S13.6 · **Status:** Done

**What:** Same-second change cursors for GitHub, BambooHR and QuickBooks; GitHub source opt-in backfill; hourly open-month QuickBooks reports; conversations tagged with their space; `make coverage` as its own step and CI job. PT-0014 fixed (BUG-0022).

**Why this way:** The Linear connector already solved the same-second problem by remembering what it reported at the cursor's exact time; copying that one pattern beats three new ideas. Coverage moved out of `make check` because it doubled the gate's time without changing its verdict on a change.

**Learned:** Freezing a fake's clock is enough to reproduce a race with wall time; three fakes gained that one switch.

**Laws:** LAW-13 holds in the cases it missed.

**Next:** plan W14, the cell runtime.

## 2026-11-20 S13.5 Coverage gaps (SB-1405)

**Wave/slice:** W13 / S13.5 · **Status:** Done. **W13 built.**

**What:** Coverage profiles from `make test`, `make laws` and `make smoke`, merged by `tools/check/coverage.py` with a 75% floor per core package; tests for `proposals` and the `barrier` cell facts.

**Why this way:** Most core code is tested through its laws, so a per-package unit-test figure would say `runtime` is 14% covered when its laws drive four fifths of it. Merging every suite measures what the tests really exercise, and the floor stops a new package arriving untested.

**Learned:** The one package below the floor was the one the review's concurrency fix had just changed: new code with its test in another package's law.

**Laws:** none changed; every core package now has a figure and a floor.

**Next:** W13 is built. What's left is outside the code (see the project notes' State).

## 2026-11-20 S13.4 The manual and the code agree (SB-1404)

**Wave/slice:** W13 / S13.4 · **Status:** Done

**What:** `tools/check/manual_code.py` in `make lint`; six manual drifts fixed (sb-api(3), sb-ask(3), sb(1)); CLAUDE.md's state section cut to a summary.

**Why this way:** The manual is the spec (P2), so the check runs both ways: code the manual doesn't describe is unspecified behaviour, and a manual promise the code doesn't keep is a broken spec. Not-built endpoints have a named place to wait, with their ticket, so a gap is a decision rather than an accident.

**Learned:** Every drift found was in a place no law watches: an example request, a UI route, a chat webhook, a synopsis line.

**Laws:** LAW-16 (the manual as the spec of the manifest-driven cell) gains a static check.

**Next:** S13.5, coverage gaps (SB-1405).

## 2026-11-20 S13.3 Stricter lint and tidier files (SB-1403)

**Wave/slice:** W13 / S13.3 · **Status:** Done

**What:** `errorlint`, `nilerr`, `unparam`, `wastedassign`, `gocritic`, `bodyclose`, `misspell`, `unconvert`, `nakedret`, `dupl` and `gocyclo`, with every finding fixed or annotated; the tool bus split into `toolbus.go`, `decide.go` and `view.go`; `verify.Run` split per law; shared API helpers; barrier wrappers removed.

**Why this way:** Linters that can be satisfied mechanically are worth turning on only if every finding gets a real answer; the deliberate exceptions (`nilerr` on ACLs that answer `Unknown`) now say why in the code, so the next reader doesn't "fix" them into errors.

**Learned:** `errorlint`'s `%v`-versus-`%w` check would have meant 87 edits that change what callers can unwrap, so it is off: wrapping is a decision, not a style. Its comparison and assertion checks found three real misses.

**Laws:** none changed; every law test unchanged and green.

**Next:** S13.4, the manual and the code agree (SB-1404).

## 2026-11-20 S13.2 Fuzzing and the race detector (SB-1402)

**Wave/slice:** W13 / S13.2 · **Status:** Done

**What:** Fuzz targets for the manifest, payload hashing, the reorg snapshot, chunking and secrets, scope patterns and the SCIM endpoints; `make fuzz`; `-race` in `make test`; `tests/spaces` and `tests/groupsync` added to `make check`. BUG-0021.

**Why this way:** Each target checks a property, not just "no panic": a manifest reads back as it was checked, a hash depends only on the canonical form, a secret in a chunk is a secret in the document. That is what turned a fuzzer into a bug finder within a minute.

**Learned:** Two test suites (`tests/spaces`, `tests/groupsync`) were never run by `make check`; the regression tests from S13.1d lived in one of them. Adding a suite now means adding it to the Makefile in the same change.

**Laws:** LAW-07, LAW-15, LAW-16 fuzzed.

**Next:** S13.3, stricter lint and tidier big files (SB-1403).

## 2026-11-20 S13.1d Ingest, connector and runtime findings (SB-1401)

**Wave/slice:** W13 / S13.1d · **Status:** Done (SB-1401)

**What:** `Reroute` drops what it can't fetch and goes on; field allow-lists in the routing fingerprint; holders narrowed for sensitive fields; GitHub identity matching and paging (and a paging GitHub fake); QuickBooks query paging; a per-sync Drive folder cache; KMS key rings checked by name; whole-resource Cloud Run updates; shred before drop. PT-0012, PT-0013 (BUG-0019, BUG-0020); PT-0014 accepted.

**Why this way:** Where the right answer is unknown (a document that can't be fetched, an identity that isn't the asker's), the fix takes the side the laws take: out of the index, Unknown.

**Learned:** Half of these lived in the fakes, not the code: a fake that ignores paging, filters perfectly, and labels key rings lets the code be wrong in exactly those ways. The fakes now behave more like the services.

**Laws:** LAW-03, LAW-13 and LAW-20 hold where they didn't. SB-1401 is done.

**Next:** S13.2, fuzzing and the race detector (SB-1402).

## 2026-11-20 S13.1c Governance findings (SB-1401)

**Wave/slice:** W13 / S13.1c · **Status:** In progress (one more fix slice)

**What:** `Update` on the proposal and review stores, used for every decision; stale proposals audited; operators may decide reviews; `retention.Job.SpaceDays`; IdP requests for removals a manifest group doesn't list; `EHOLD` for held erasure; feedback swept before traces; both SCIM manager shapes and `remove`. PT-0010 (BUG-0018), PT-0011 accepted.

**Why this way:** A decision is audited and stored as one step under the store's lock, so the audit trail and the stored state can't disagree, whichever of two concurrent requests wins.

**Learned:** The race test was the only way to see the lost decision: every single-threaded test passed.

**Laws:** LAW-11, LAW-13 and LAW-16 hold where they didn't; a regression test for each.

**Next:** S13.1d, the ingest and connector findings.

## 2026-11-20 S13.1b Authorisation and approval findings (SB-1401)

**Wave/slice:** W13 / S13.1b · **Status:** In progress (two more fix slices)

**What:** In the tool bus: `Edit` re-decides the duty and keeps the expiry, the duty counts the amount's size, `mayDecide` is shared by approve and reject, a duty without reporting lines refuses, and `Get` hides an action across a barrier. Owners by email (`barrier.Directory`), barriers need spaces (sb-space(5) rule 14), chat scopes carry attributes, and routing version 2. PT-0008, PT-0009; BUG-0016, BUG-0017.

**Why this way:** Each fix makes one rule hold in every path it applies to, instead of the first path it was written for: approval checks in one function for both decisions, attributes in every scope, and the barrier for reading an action as well as deciding on it.

**Learned:** Most of these were "the second path": reject beside approve, chat beside web, edit beside propose. A law test that drives only the main path proves only the main path.

**Laws:** LAW-15 and LAW-22 hold in the paths they missed; a regression test for each.

**Next:** S13.1c, the governance findings.

## 2026-11-20 S13.1a The review's critical findings (SB-1401)

**Wave/slice:** W13 / S13.1a · **Status:** In progress (more fixes follow)

**What:** Three independent reviews of W8 to W12. This slice fixes the two critical findings and the diff that could hide a change: whole-document secret spans in `internal/core/ingest` with more patterns, `StewardChange` that refuses values which read back differently, and a path-aware, multiset `manifest.Diff`. Findings PT-0005 to PT-0007; BUG-0014 and BUG-0015.

**Why this way:** Scanning the whole document before chunking makes the chunk size irrelevant to what counts as a secret. Comparing what was checked with what was proposed catches every way a value can change on the way through YAML, not only the comment trick that was found.

**Learned:** Both critical bugs lived between two correct pieces: a scanner and a chunker that were each right alone, a rank check and a formatter that were each right alone. The review found them by following a value across the boundary.

**Laws:** LAW-07 and LAW-16 hold again where they didn't; regression tests for each.

**Next:** S13.1b, the authorisation and approval findings.

## 2026-11-20 W13 planned: hardening instead of portfolio grants

**Wave/slice:** W13 (plan) · **Status:** Planned

**What:** Q-01 stays no, so the portfolio grants epic (E13) is parked and LAW-23 stays in the red ratchet. W13 becomes a hardening wave (E14, SB-1401 to SB-1405): a fresh-eyes review of W8 to W12, fuzzing and the race detector, stricter lint and tidier files, the manual and the code in agreement, and the coverage gaps in the core.

**Why this way:** before calling the platform feature complete, the code should be reviewed, fuzzed, tidied and matched to its manual, so that what is handed to companies is as solid as its laws say.

**Learned:** (recorded after the fact, 2026-11-21) this entry was missing from the devlog when the wave was planned; the plan itself is in the commit "W13 plan: hardening instead of portfolio grants".

**Laws:** LAW-23 stays red while E13 is parked.

## 2026-11-20 S12.4b Cross-space red team (SB-1208)

**Wave/slice:** W12 / S12.4b · **Status:** In review (the game day is human)

**What:** `docs/red-team-spaces.md`, `docs/runbooks/space-red-team-game-day.md`, the `space-redteam` suite and world (with a document `owner` in the eval cell), sb-space(5) rule 13 and `reachChecks`, findings PT-0003 and PT-0004, BUG-0013, and `laws/law20reach.go`. The manual changed first: sb-laws(7), sb-eval(5), sb-space(5).

**Why this way:** Attacks that can be automated are a gating eval, so they are tried on every release and not only on the day; the game day is for what a person thinks of that the corpus doesn't.

**Learned:** The attack that worked was configuration, not code: two spaces' scopes overlapping. Every space check looked at one space at a time, and the law generators gave each space its own folder, so nothing ever generated the overlap. The new rule compares spaces with each other.

**Laws:** LAW-19 to LAW-22 extended (one law test, one mutation; one gating suite). Only LAW-23 stays red.

**Next:** W13, only if Q-01 is yes.

## 2026-11-20 S12.4a Legal hold per space (SB-1207)

**Wave/slice:** W12 / S12.4a · **Status:** Done

**What:** `internal/core/holds`, holds in `retention.Store.Expire` and `Job.Erase`, a space on traces, `spaces.SyncHeld`, `api.Holds` with Cedar `place_hold` and `release_hold`, `EHOLD`, and `laws/law13holds.go`. The manual changed first: sb-spaces(7), sb-tenant(5), sb-admin(3), sb-errors(7).

**Why this way:** A hold is a filter every deleter asks, not a flag on records, so it takes effect the moment it is placed and stops the moment it is released, and a store that can't tell whether a record is in scope keeps it. Space removal checks holds before it touches anything, so a held space can never be half-removed with its key shredded.

**Learned:** Conversations and log files don't know their space, so a hold on one space keeps every conversation while it is active. That keeps more than the matter needs, which is the safe direction; tagging conversations with their space would narrow it.

**Laws:** LAW-13 and LAW-11 extended (two mutations caught). Only LAW-23 stays red.

**Next:** S12.4b, the cross-space red team and game day (SB-1208).

## 2026-11-20 S12.3b Reorg preview (SB-1206)

**Wave/slice:** W12 / S12.3b · **Status:** Done

**What:** `internal/core/reorg` (snapshot parse and diff), `sb space diff`, and `laws/law16reorg.go`. The manual changed first: sb-space(1), sb-spaces(7), sb(1).

**Why this way:** The snapshot is the reorg as HR plans it, each moved person's groups before and after, rather than two full directory dumps: it is what HR already has, it is small enough to review, and the preview needs no cell and no credentials.

**Learned:** The useful flag beyond the ticket's two is `barrier`: a reorg that puts someone on both sides of a wall is a mistake to catch before the IdP changes, not after.

**Laws:** LAW-16 extended (three mutations caught). Only LAW-23 stays red.

**Next:** S12.4, legal hold per space (SB-1207).

## 2026-11-20 S12.3a Directory attributes and five-minute moves (SB-1205)

**Wave/slice:** W12 / S12.3a · **Status:** Done

**What:** SCIM `title` and enterprise `department`, `division`, `costCenter` (with `manager` from SB-1204), `ports.User.Attributes`, `auth.Cell.Attributes` and `auth.DirectoryAttributes`, `Scope.WithAttrs`, principal attributes in the kernel, the Cedar adapter and the schema, and `laws/law21moves.go`. The manual changed first: sb-identity(7), sb-permissions(7), sb-policy(5).

**Why this way:** The asker is read from the directory at every request, groups and attributes alike, so a move needs nothing to expire and nothing to invalidate: the five-minute bound holds with room to spare, and the law test checks the request path rather than a cache.

**Learned:** The first trusted-claims mutation went uncaught because the test's IdP view was keyed by user name while the cell sees directory ids; keyed by id, it is caught. A mutation that isn't caught is a question about the test before it is a verdict on the code.

**Laws:** LAW-21 green and out of the ratchet (two mutations caught). Only LAW-23 stays red.

**Next:** S12.3b, the reorg preview (SB-1206).

## 2026-11-20 S12.2b Segregation of duties (SB-1204)

**Wave/slice:** W12 / S12.2b · **Status:** Done

**What:** `duties` in the manifest and its schema, `Tool.Duty` and two-stage approval in the tool bus, `approvals.ReportingLine` and `Amount`, the SCIM `manager`, a platform forbid on approvals outside the named group, `Approvals.tla` and `crosscheck.py` with the second approver and the reporting line, and `laws/law15duties.go`. The manual changed first: sb-approvals(7), sb-tenant(5), sb-identity(7).

**Why this way:** The first approval is a recorded step on the same action, not a second action, so one hash, one expiry and one edit-voids-everything rule cover both. An amount the cell can't read asks for both approvals, never one, and a reporting line it can't read refuses.

**Learned:** The law test caught a real gap on its first run: Cedar permits add up, so the example tenant's "IT leads approve ticket updates" permit let an IT lead give the finance approval too. A platform forbid now ties every approval to the group named for it.

**Laws:** LAW-15 modelled and tested further (five mutations caught across model and law test). LAW-21 and LAW-23 stay red.

**Next:** S12.3, SCIM attributes and five-minute moves (SB-1205).

## 2026-11-20 S12.2a Information barriers (SB-1203)

**Wave/slice:** W12 / S12.2a · **Status:** Done

**What:** Owner on documents and chunks (Drive's file owner; an `owner` index column), owner groups at decision time in retrieval, `Authoriser.ItemAllowed` for draft items, `internal/core/barrier` and `Bus.Barriers` for tool calls, proposals and approvals, new forbids in `policy/platform/spaces.cedar`, and `laws/law22barriers.go`. The manual changed first: sb-permissions(7), sb-policy(5), sb-space(5), sb-spaces(7), sb-connector(3).

**Why this way:** The barrier stays one set of forbids in the kernel, fed facts from the manifest and the directory, rather than a generated rule per barrier, so `make policy` can still prove it only narrows. Ownership is resolved to groups at decision time, not at ingest, so a move in the directory changes it at the next request, and an owner that can't be read refuses like any `Unknown`.

**Learned:** A shared drive file has no owner, so ownership can only ever add refusals; a missing owner never opens anything that holding spaces would refuse.

**Laws:** LAW-22 green and out of the ratchet (three mutations caught); LAW-03's space-facts test covers the two new facts. LAW-21 and LAW-23 stay red.

**Next:** S12.2b, segregation of duties (SB-1204).

## 2026-11-20 S12.1b Quarterly access reviews (SB-1202)

**Wave/slice:** W12 / S12.1b · **Status:** In review (console screens)

**What:** `internal/core/reviews` (open, decide, requests, overdue, watch, evidence, the job), `internal/api/reviews.go`, the `review` schedule, `review_space` in Cedar, and `laws/law16reviews.go`. The manual changed first: sb-spaces(7), sb-admin(3), sb-runtime(7), sb-observability(7).

**Why this way:** A review is evidence, so a decision changes nothing by itself: removals become the same hash-approved manifest proposals stewards already use, or IdP requests, and the live change keeps one path (LAW-16). The member list is built from the directory groups, not from who happened to ask, and a group that can't be read stops the review rather than certifying a shorter list.

**Learned:** Last use comes cheaply from the audit trail, because every row written in a space already carries the space.

**Laws:** LAW-16 extended (one law test, three mutations caught). LAW-21 to LAW-23 stay red.

**Next:** S12.2, information barriers (SB-1203).

## 2026-11-20 S12.1a Stewards and delegated space changes (SB-1201)

**Wave/slice:** W12 / S12.1a · **Author:** platform lead + Claude · **Status:** In review (console screens)

**What:** `propose_space` and `approve_space` in the schema and base policy, `manifest.StewardChange` and `WithSpace`, `internal/core/proposals`, `internal/api/stewards.go`, and `laws/law15stewards.go`. The manual changed first: sb-spaces(7), sb-admin(3).

**Why this way:** A steward edits one thing, their space's definition, so the proposal is that definition and nothing else; anything outside it can't even be expressed. Approval binds the exact manifest by hash and the manifest it was made from, so what an operator reviewed is what applies, and a change that lands in between makes the proposal stale rather than silently merged.

**Learned:** The first version of the law test passed while checking nothing: every proposal failed the manifest check because unset fields were written as zeros. The test now fails if no proposal is ever approved, and `WithSpace` fills defaults as the checker does.

**Laws:** LAW-15, LAW-16 and LAW-19 extended (one law test, two mutations caught). LAW-21 to LAW-23 stay red.

**Next:** S12.1b, quarterly access reviews (SB-1202).

## 2026-11-20 S11.4 Per-template eval suites in the gate (SB-1107)

**Wave/slice:** W11 / S11.4 · **Author:** platform lead + Claude · **Status:** In review (tenant suites in a deployed cell)

**What:** twelve `evals/suites/template-*.yaml`, `evals/fixtures/templates/world.yaml`, the gate entries and `make evals`, the barrier fix in `spaces.ReadContext`, and `laws/law06templates.go`. The manual changed first: sb-evals(7).

**Why this way:** A template is a promise about one department, so each gets its own suite and its own line in the gate; a shared world keeps the twelve comparable and cheap. The law test reads the catalogue, so adding a template without a suite fails the build rather than waiting for a reviewer to notice.

**Learned:** The executive barrier case failed on its first run, and the bug was real: barriers compared `group:NAME` strings with bare group names, so they held only when every side was written the same way. An eval world written the natural way found it.

**Laws:** LAW-06 extended (one law test, one mutation caught); LAW-20 and LAW-22 covered by the suites. W11 is built; LAW-21 to LAW-23 stay red for W12 and W13.

**Next:** W12, space governance (E12).

## 2026-11-20 S11.3c The QuickBooks Online connector (SB-1105)

**Wave/slice:** W11 / S11.3c · **Author:** platform lead + Claude · **Status:** In review (a real sandbox company)

**What:** `internal/adapters/connectors/quickbooks` and `qbotest`, the `accounting` kind, and `laws/lawquickbooks.go`. The manual changed first: sb-connector(3), sb-tenant(5) and sb-ask(3).

**Why this way:** Finance questions are about totals, budgets and the shape of the ledger, never about account numbers, so the connector reads only what answers them and leaves the rest undecoded; nothing that wasn't decoded can be logged or indexed. QuickBooks keeps report access inside its own app, so the readers are a list the finance stewards keep in the vault: refusing everyone when the list is missing is the only safe default.

**Learned:** A report has no version of its own; the newest change to any account is the version that makes a rebuild give the same text.

**Laws:** LAW-01, LAW-03, LAW-07 and LAW-20 extended (three law tests, one mutation caught). LAW-21 to LAW-23 stay red.

**Next:** S11.4, per-template eval suites in the gate (SB-1107).

## 2026-11-20 S11.3b The BambooHR connector (SB-1104)

**Wave/slice:** W11 / S11.3b · **Author:** platform lead + Claude · **Status:** In review (a real account)

**What:** `internal/adapters/connectors/bamboohr` and `bamboohrtest`, the `hris` kind, and `laws/lawbamboohr.go`. The manual changed first: sb-connector(3), sb-tenant(5) and sb-ask(3).

**Why this way:** Ingest already drops sensitive fields, but a value that is never read can never leak through a log, a crash dump or a cache, so the connector asks BambooHR for the sensitive fields a space allow-lists and nothing more. Employee records are open to every active employee at the source because the directory is; who sees pay or personal details is decided by the restricted label and the People space, which the policy kernel enforces.

**Learned:** The shared connector law expects a ref the source can't place to be Unknown, and an id BambooHR can't parse is exactly that; a record that's gone is a plain deny.

**Laws:** LAW-03, LAW-07 and LAW-20 extended (two law tests, one mutation caught). LAW-21 to LAW-23 stay red.

**Next:** S11.3c, the QuickBooks Online connector (SB-1105).

## 2026-11-20 S11.3a Field-level allow-lists for structured sources (SB-1106)

**Wave/slice:** W11 / S11.3a · **Author:** platform lead + Claude · **Status:** Done

**What:** `ports.Field`, `ports.FieldDeclarer` and `Document.Fields`; `internal/core/ingest/fields.go`; `fields` on a space through the manifest, plan and registry; structured records in the mock source; `laws/law07fields.go`. The manual changed first: sb-connector(3), sb-ingest(7) and sb-space(5).

**Why this way:** For an HR or ledger record the dangerous thing isn't the document but one field of it, so the decision moves from "which documents" to "which fields", and it is made once, in ingest, from the connector's declaration and the space's allow-list, never by each connector writing its own text. A sensitive field a steward allows makes the whole record restricted, because a salary next to a name is only as safe as its most sensitive line.

**Learned:** "Can't declare its fields" has to mean "feeds nothing", not "feeds everything as text": the second would turn the safest-looking connector into the leakiest.

**Laws:** LAW-07 and LAW-20 extended (two law tests, one mutation caught). LAW-21 to LAW-23 stay red.

**Next:** S11.3b, the BambooHR and QuickBooks Online connectors (SB-1104, SB-1105).

## 2026-11-20 S11.2 The GitHub code connector (SB-1103)

**Wave/slice:** W11 / S11.2 · **Author:** platform lead + Claude · **Status:** In review (a real organisation)

**What:** `internal/adapters/connectors/github` and `githubtest`, the `code` kind, `ingest.SecretIn` and the drop in `ingest.Pipeline`, and `laws/lawgithub.go`. The manual changed first: sb-connector(3), sb-ingest(7), sb-tenant(5) and sb-ask(3).

**Why this way:** Engineers write their knowledge in READMEs, docs and issues; source code is a much larger and much riskier corpus, so it is opted in per repository by a space's scope rather than on by default. Access is per repository because that is GitHub's own unit of permission, and asking once per repository keeps a 200-document batch to a handful of calls. The secret scanner sits in ingest, not in this connector, because a key pasted into a Slack message or a Drive doc is just as much a secret.

**Learned:** Fault injection has to match the shape of the calls: with refs in two repositories the connector makes two permission calls, and a single injected fault only covers one of them.

**Laws:** LAW-01, LAW-03 and LAW-07 extended (three law tests, two mutations caught). LAW-21 to LAW-23 stay red.

**Next:** S11.3, BambooHR, QuickBooks Online and field-level allow-lists (SB-1104 to SB-1106).

## 2026-11-20 S11.1b Department agents (SB-1102)

**Wave/slice:** W11 / S11.1b · **Author:** platform lead + Claude · **Status:** Done

**What:** four prompts in the bundled set, `internal/core/ask/departments.go`, the agent path in `ask.Agent`, a base `use_agent` permit, manifest rule 11, agents in the local eval cell, four suites, and `laws/law14departments.go`. The manual changed first: sb-agents(7), sb-ask(3), sb-space(5), sb-tenant(5), sb-eval(5) and sb-evals(7).

**Why this way:** A department agent is the ask agent with a different prompt, so it inherits every check the ask path already proves (spaces, the live ACL, the gateway, the composer) instead of re-earning them. Binding it to its template rather than to a space's list alone stops a space from borrowing Finance's close checklist and pointing it at someone else's documents. Dropping uncited claims is stricter than labelling them, which is what a summary meant to be pasted somewhere needs.

**Learned:** A mutation that rewrites the system's input must copy it, or it rewrites the test's oracle too and passes for the wrong reason.

**Laws:** LAW-14 and LAW-09 extended (two law tests, three mutations caught). LAW-21 to LAW-23 stay red.

**Next:** S11.2, the GitHub code connector (SB-1103).

## 2026-11-20 S11.1a The template catalogue as data, and sb space init (SB-1101)

**Wave/slice:** W11 / S11.1a · **Author:** platform lead + Claude · **Status:** Done

**What:** `internal/core/manifest/catalogue.json` and `ParseCatalogue`, the bundle's `catalogue` hash, `internal/cli/space.go`, and `laws/law16catalogue.go`. The manual changed first: sb-space(1) (new), sb-departments(7), sb-release(7) and sb(1).

**Why this way:** A template decides what a department's brain shows by default, so it belongs with the other things a release pins (prompts, policies, the gate), not in code that only a build changes. `sb space init` edits text rather than re-rendering the manifest, because a manifest is a reviewed file and a tool that drops its comments would make every review harder.

**Learned:** The manual's catalogue table and the data file are two copies of one fact; the test that compares them is what lets the manual stay the authority.

**Laws:** LAW-16 extended (one law test, one mutation caught). LAW-21 to LAW-23 stay red.

**Next:** S11.1b, department agents (SB-1102).

## 2026-11-20 S10.3c Audit and kill switch per space (SB-1008)

**Wave/slice:** W10 / S10.3c · **Author:** platform lead + Claude · **Status:** Done

**What:** the `space:NAME` kill scope through the checker, the ask and draft agents, the tool bus and retrieval; `space` on audit rows and actions; `audit.InSpace` and the admin filter; `kill.DrillScopes`; `sb kill space`; apply accepting spaces; `laws/law10spaces.go`. The manual changed first: sb-kill-switches(7), sb-kill(1), sb-audit(5), sb-audit(7), sb-audit(1) and sb-admin(3).

**Why this way:** A killed space has to disappear from company-wide answers too, or killing Finance would only stop people asking Finance directly. A document another space also holds is dropped as well, because a kill that leaves part of a space readable isn't one. The audit field is omitted for company-wide rows, so every row written before spaces still hashes as it did and old exports keep verifying.

**Learned:** An unkill has the same 10-second bound as a kill; the law test first expected recovery at once and was wrong about the system, not the system about the law.

**Laws:** LAW-10 and LAW-11 extended (two law tests, two mutations caught). W10 is built; LAW-21 to LAW-23 stay red for W12 and W13.

**Next:** W11, department templates (E11).

## 2026-11-20 S10.3b Spaces in the web UI and the chat app (SB-1007)

**Wave/slice:** W10 / S10.3b · **Author:** platform lead + Claude · **Status:** In review (WCAG audit, real Slack)

**What:** `internal/api/spaces.go`, spaces and label on `answer.Cite`, `channels` on a space, the Slack app's channel step, the web UI's switcher and badges, and `laws/law20chat.go`. The manual changed first: sb-ask(3) and sb-space(5).

**Why this way:** The switcher lists only what `GET /v1/spaces` returns, and that list is the asker's memberships, so the UI can't offer a space that would be refused, nor reveal one the asker isn't in. In Slack a channel is the only signal of intent, so a mapped channel for a non-member gets no answer: a company-wide answer there would look like the space's answer and teach people to trust it.

**Learned:** The app now calls the API when it starts, and the older UI tests counted every call as an ask; their fakes answer the spaces call with none.

**Laws:** LAW-20 extended at the chat app (one law test, one mutation caught); LAW-14's UI property covers chips with places. LAW-21 to LAW-23 stay red.

**Next:** S10.3c, audit and kill switch per space (SB-1008).

## 2026-11-20 S10.3a Ask in a space or company-wide (SB-1006)

**Wave/slice:** W10 / S10.3a · **Author:** platform lead + Claude · **Status:** Done

**What:** the space step in `retrieve.Context`, spaces in `tools/evalcell` and `evals/targets.py`, `laws/law20ask.go`, `space-leak` in the gate. The manual changed first: sb-ask(3), sb-citations(7), sb-spaces(7), sb-eval(5), sb-evals(7) and sb-laws(7).

**Why this way:** The space checks run before the ACL batch because they are local and cheap, and a chunk they refuse never needs a source call; the batch still decides every chunk they allow, so the source stays the authority (LAW-19). The trace reason is worked out after the kernel's decision, so explaining a refusal can never change it.

**Learned:** The eval world needed a document's path and label in its first version: placing it afterwards made a second version, and the controls looked for the first.

**Laws:** LAW-20 green (out of the ratchet, one mutation caught); LAW-19 and LAW-01 extended on the ask path. LAW-21 to LAW-23 stay red.

**Next:** S10.3b, spaces in the web UI and the chat app (SB-1007).

## 2026-11-20 S10.2b Ingest routes and labels every document (SB-1005)

**Wave/slice:** W10 / S10.2b · **Author:** platform lead + Claude · **Status:** In review (a real Drive with labels)

**What:** `internal/core/route`, routing in `ingest.Pipeline` (with `Reroute`), location and label in `ports.Document` and every connector, label, holders and routing on `ports.Chunk` in both indexes, and `laws/law13spaces.go`. The manual changed first: sb-ingest(7) and sb-connector(3).

**Why this way:** Routing is a pure function of a document's location and label and the manifest, so the index stays derived (LAW-13) and a rebuild needs nothing but the sources and the manifest. A fingerprint of the space configuration on every chunk turns "a scope changed" into a comparison the next sync can make, rather than a job someone has to remember to run. The connector reports where a document lives and never decides its space; only the manifest does.

**Learned:** A Drive folder that can't be read has to fail the fetch. Routing a file without its full path would put it in no space, which for a company-wide ask is the wider answer.

**Laws:** LAW-13 and LAW-20 extended (two law tests, two mutations caught). LAW-20's planned ask-path test, and LAW-21 to LAW-23, stay red.

**Next:** S10.3a, ask in a space or company-wide (SB-1006).

## 2026-11-20 S10.2a Cedar schema v2: spaces and labels, narrowing only (SB-1004)

**Wave/slice:** W10 / S10.2a · **Author:** platform lead + Claude · **Status:** Done

**What:** schema v2 in `policy/schema.cedarschema`, `policy/platform/spaces.cedar`, string sets in the Cedar adapter, `spaces.ReadContext` and `ChunkAttrs`, rule 5 in `tools/check/policy.py`, and `laws/law19policy.go`. The manual changed first: sb-policy(5), sb-permissions(7) and sb-laws(7).

**Why this way:** Spaces enter Cedar as facts on the request rather than as entities with relations, so the policy can't reach a space the kernel didn't hand it, and a missing fact is visible to the policy as a missing attribute that refuses. Every space rule is a forbid, which Cedar can't turn into an allow, and the static check makes that a property of the files rather than of review.

**Learned:** A permit with an unconstrained action reads like it is about something else and still covers `read`; the narrowing check now treats it as a read permit.

**Laws:** LAW-19 green (out of the ratchet, one mutation caught); LAW-03 extended (one law test, one mutation caught). LAW-20 to LAW-23 stay red, as planned.

**Next:** S10.2b, ingest routes and labels every document (SB-1005).

## 2026-11-20 S10.1c Per-space data keys (SB-1003)

**Wave/slice:** W10 / S10.1c · **Author:** platform lead + Claude · **Status:** In review (the stores seal from SB-1005)

**What:** `keywrap.PurposeSpace`, `Shred` and `DataKey`; `internal/core/seal`; keys in `spaces.Sync`; `laws/law18spaces.go`. The manual changed first: sb-tenancy(7) and sb-factory(7).

**Why this way:** A space key is one more purpose in the wrap inventory, so every rule SB-704 proved for hand-off covers it with no new code. Shredding is the only way a purpose leaves the inventory, and it is refused for anything but a space, so LAW-18's "never remove the last working wrap" stays true for the keys that keep the cell alive. The tenant and space go into the cipher's additional data so a sealed record can't be moved to a space that would open it.

**Learned:** "Not found" and "doesn't unwrap" have to stay different errors: one means the space was shredded on purpose, the other is a Sev-1.

**Laws:** LAW-18 extended (one law test, two mutations caught). LAW-19 to LAW-23's planned tests stay red.

**Next:** S10.2a, Cedar schema v2 for spaces and labels (SB-1004).

## 2026-11-20 S10.1b The space registry and the request context (SB-1002)

**Wave/slice:** W10 / S10.1b · **Author:** platform lead + Claude · **Status:** Done

**What:** `ports.SpaceStore`, `internal/adapters/spaces/memory`, `internal/adapters/postgres/spaces`, `internal/core/spaces` (Sync, Resolve), `internal/core/spacestest`, the scope's space in `tenancy` and `auth`, and `laws/law19context.go`. The manual changed first: sb-spaces(7), sb-api(3) and sb-tenancy(7).

**Why this way:** The space joins the scope because the scope is already the one thing every handler must carry and nothing can forge. Resolving it in the middleware means no handler ever sees a space name it would have to check. Every way a space can fail to resolve is a refusal, because the tempting default, company-wide, is the wider answer.

**Learned:** A non-member and an unknown space must get the same error; otherwise `?space=` becomes a way to list another department's spaces.

**Laws:** LAW-19 gains a green law test at the request context (one mutation caught); LAW-02's cross-tenant case is in it. LAW-19 to LAW-23's planned tests stay red.

**Next:** S10.1c, per-space data keys (SB-1003).

## 2026-11-20 S10.1a Spaces in the manifest (SB-1001)

**Wave/slice:** W10 / S10.1a · **Author:** platform lead + Claude · **Status:** Done

**What:** `internal/core/manifest/spaces.go` (the parser, rules, template defaults and plan attributes), spaces and barriers in `Format`, the JSON Schema and `plan`, the apply refusal, `examples/tenants/spaces.yaml`, and `TestLAW16_SpacesAreExactlyAsDeclared`. The manual changed first: sb-space(5), sb-tenant(5), sb-tenant(1), sb-spaces(7) and sb-onboarding(7).

**Why this way:** Checking and planning spaces now lets a company write and review its spaces while the enforcement is built, but applying them before SB-1006 would recreate the risk SB-801 refused: a People space that looks restricted and isn't. So the line moved from `check` to `apply`, which still refuses, before any write, with the ticket named. Template defaults are filled in by the checker, so the plan shows every field a space will have and nothing is left for the cell to guess.

**Learned:** A barrier between two groups that share a space is easy to write and impossible to honour; the checker reports it against the barrier, naming the space.

**Laws:** LAW-16 extended (one law test replaced and widened, its mutation still caught). LAW-19 to LAW-23 are still red, as planned.

**Next:** S10.1b, the space registry and request context (SB-1002) and per-space keys (SB-1003).

## 2026-11-20 S9.3b Identity Platform sign-in (SB-907)

**Wave/slice:** W9 / S9.3b · **Author:** platform lead + Claude · **Status:** In review (a real project, SB-903)

**What:** the `identity-platform://` idp and `groups:` in `internal/core/manifest`, `Provider.IdentityPlatform` in the `oidc` adapter, `internal/core/groupsync`, `laws/law02idp.go` and `laws/law12groups.go`. The manual changed first: sb-identity(7) and sb-tenant(5).

**Why this way:** Every company's token from one Identity Platform project has the same issuer, signing keys and audience, so the per-company Identity Platform tenant is the only thing that tells them apart; the verifier checks it on every token. The directory is keyed by verified email because Identity Platform's user ids are its own and mean nothing to the cell. For a company without SCIM the manifest is the directory, and an apply that removes someone revokes their sessions; the 60-second re-validation keeps the 5-minute bound even when that revocation is lost.

**Learned:** The simplest wrong configuration is a cell that trusts the project, and every other check still passes for it. The mutation test is exactly that cell.

**Laws:** LAW-02 and LAW-12 extended (two law tests, two mutations caught). LAW-19 to LAW-23 are still red, as planned.

**Next:** W10, spaces in a cell (SB-1001 onward). SB-903 and SB-908 wait on a real GCP project.

## 2026-11-20 S9.3a Firebase Hosting for the web UI (SB-906)

**Wave/slice:** W9 / S9.3a · **Author:** platform lead + Claude · **Status:** In review (a real site, SB-903)

**What:** `internal/core/hosting`, `gcp.Hosting`, Firebase Hosting in `gcptest`, and `laws/law02hosting.go`. The manual changed first: sb-runtime(7) and sb-ask(3).

**Why this way:** The site lives in the cell's own project, so Firebase Hosting's own rule (it rewrites only to Cloud Run services in its project) already keeps it on its cell. The generator and its check make that a property of the code too, with one allowed shape and everything else refused. The law test tries the realistic mistakes: a site config copied from another tenant, a rewrite to another tenant's `web.app`, a scheme-less URL, and publishing the other tenant's site name.

**Learned:** The first draft of the generated CSP was looser than sb-ask(3)'s, which also forbids framing. The generator now uses the page's policy, word for word.

**Laws:** LAW-02 extended (one law test, one mutation caught). LAW-19 to LAW-23 are still red, as planned.

**Next:** S9.3b, the Identity Platform adapter (SB-907).

## 2026-11-20 S9.2b Metrics and the verifier endpoint (SB-905)

**Wave/slice:** W9 / S9.2b · **Author:** platform lead + Claude · **Status:** In review (a real cell, SB-903 and SB-908)

**What:** `internal/api/verifier.go`, `runtime.Attestor`, LAW-17 in `internal/core/verify`, `alerts.Dispatch`, `sb tenant verify --cell`, and two law tests. The manual changed first: sb-runtime(7), sb-tenant(1), sb-observability(7) and sb(1)'s examples.

**Why this way:** An attestation is only evidence if it reports what runs, so the attestor reads Cloud Run at the moment it is asked, never the updater's intent. A half-applied update then shows up as the mismatch it is. The verifier runs inside the cell and returns only the report, so `sb tenant verify` against a deployed cell sees no more than the local one does: counts, never content.

**Learned:** A per-process counter for the attestation's sequence number would replay after every restart, and the controller would raise a Sev-1 replay for nothing. Milliseconds since the epoch, kept increasing, survive restarts.

**Laws:** LAW-06 and LAW-17 extended (two law tests, two mutations caught). LAW-19 to LAW-23 are still red, as planned.

**Next:** S9.3: Firebase Hosting (SB-906) and the Identity Platform adapter (SB-907).

## 2026-11-20 S9.2a Schedules, the watchdog and the WORM bucket (SB-904)

**Wave/slice:** W9 / S9.2a · **Author:** platform lead + Claude · **Status:** In review (the jobs runner in a running cell, SB-903)

**What:** `internal/core/schedule`, `gcp.Scheduler` and `gcp.Bucket` (`schedules.go`), Cloud Scheduler and Cloud Storage in `gcptest`, and `laws/law11worm.go`. The manual changed first: sb-runtime(7), sb-ingest(7) and sb-observability(7).

**Why this way:** Several laws depend on something happening on time: anchors hourly, the access check monthly, sync every minute. A job that silently stops is a law that silently stops being checked. So the watchdog treats "no record" and "can't read the record" as missed, and a missed law job is a Sev-1. The WORM bucket is locked at bootstrap by the project owner, because a lock can't be undone. The cell gets no delete permission at all, so even full cell credentials can't remove evidence.

**Learned:** sb-ingest(7) already said sync runs every 60 seconds. The first draft of the schedule said 5 minutes, and the manual check caught it, so sync is every minute: Cloud Scheduler's finest, with the jitter in the job.

**Laws:** LAW-11, LAW-13 and LAW-17 extended (four law tests, two mutations caught). LAW-19 to LAW-23 are still red, as planned.

**Next:** S9.2b, metrics and the verifier endpoint (SB-905).

## 2026-11-20 S9.1b Private Cloud SQL with CMEK and a perimeter (SB-902)

**Wave/slice:** W9 / S9.1b · **Author:** platform lead + Claude · **Status:** In review (a real project and organisation, SB-903)

**What:** private-IP CMEK instances in `internal/adapters/placement/gcp` (with `ensureKey`, `sqlKey` and the project number), two new deploy permissions, `deploy/gcp/bootstrap-perimeter.sh`, CMEK and VPC Service Controls in `gcptest`, and `laws/law02perimeter.go`. The manual changed first: sb-runtime(7) and sb-tenancy(7).

**Why this way:** CMEK puts the database's key in the same custody as the cell's root key, so off-boarding is one key destruction and hand-off moves one key ring. The perimeter is a second fence, so it is tested exactly where the first one has failed: with an IAM grant from one cell to another made by mistake. The adapter refuses an existing public or foreign-keyed instance, like every other identity check, rather than adopting it.

**Learned:** The key ring was labelled with the kind of whichever key made it first, so a second key in the ring would have failed the identity check. The ring now has its own kind, `key-ring`. A destroyed KMS key keeps its resource and loses its material, and the fake now models that, so policy calls still work while encryption fails.

**Laws:** LAW-02 and LAW-18 extended (one mutation caught). All law tests green except LAW-19 to LAW-23, red as planned.

**Next:** S9.2, schedules (SB-904) and metrics and the verifier endpoint (SB-905).

## 2026-11-20 S9.1a Cloud Run services and the in-cell updater (SB-901)

**Wave/slice:** W9 / S9.1a · **Author:** platform lead + Claude · **Status:** In review (a real project, SB-903)

**What:** `manual/sb-runtime.7.md`, `internal/core/runtime`, `gcp.CloudRun` and its roles, Cloud Run and Binary Authorization in `gcptest`, `laws/law06runtime.go`, and the fix for BUG-0013. The manual changed first.

**Why this way:** The factory never deploys into a cell, so the cell's own updater pulls, admits and deploys. It does that through the same admission code a cell has always used, so LAW-06 has one implementation, not two. Binary Authorization makes the platform itself refuse anything the cell didn't attest, so even someone holding the updater's rights can't sideload an image. Deploying by digest only removes the tag-moved-under-you case entirely.

**Learned:** The law tests had been leaking a temporary release store per LAW-06 run, and today they finally filled `/tmp`'s inodes (BUG-0013). Bytes were fine; inodes weren't.

**Laws:** LAW-06 and LAW-02 extended over the runtime (one mutation caught). LAW-19 to LAW-23 stay red, as planned.

**Next:** S9.1b, private Cloud SQL with CMEK and a perimeter (SB-902).

## 2026-11-20 S8.2 Space laws, model, threats and leak evals (SB-804 to SB-807)

**Wave/slice:** W8 / S8.2 · **Author:** platform lead + Claude · **Status:** Done (SB-806 In review for two reviewers)

**What:** LAW-19 to LAW-23 in sb-laws(7). Red law tests (`laws/law19.go`, `law21.go`, `law23.go`) and reference systems with mutations (`laws/selftest_spaces_test.go`). `proofs/alloy/Spaces.als` with three mutations in `tools/ci/mutations.py` and a mirror in `proofs/crosscheck.py`. The inside-the-company threats in `docs/07-threat-model.md`. The `space-leak` eval suite and `tools/check/evals_red.py`. The manual changed first: sb-laws(7), sb-eval(5) and sb-evals(7).

**Why this way:** Space features are born against their laws, as W1 to W7 were. A red test alone only proves that something is red, so each law also has a reference system that passes it and broken ones that fail it. The Alloy model is the same decision as sb-permissions(7), checked over every small company. The eval suite is red for the reason that matters: today's cell really does cite a sharing-mistake HR file to a company-wide asker. The new check makes sure it stays red for that reason and no other.

**Learned:** Alloy prints SAT results with an instance count column (`1/1  SAT`), so the mutation check now matches the result at the end of the line rather than a fixed column layout. The eval runner's table runs a long case id straight into its result (`leak-people-001FAIL`), and the expected-red check parses that too.

**Laws:** LAW-19 to LAW-23 Stated and Tested (red), and LAW-19, LAW-20 and LAW-22 Modelled. Every other law test is green. **W8 built.**

**Next:** W9, cells on GCP (SB-901 to SB-908).

## 2026-11-19 S8.1 Department spaces, specified (SB-801 to SB-803)

**Wave/slice:** W8 / S8.1 · **Author:** platform lead + Claude · **Status:** Done

**What:** `manual/sb-spaces.7.md`, `manual/sb-space.5.md` and `manual/sb-departments.7.md`; spaces in sb-permissions(7), sb-policy(5), sb-errors(7), sb-tenant(5), sb-tenancy(7) and sb-onboarding(7); three generated error codes; `manifest.Planned`; and `law16Planned` with its self-test.

**Why this way:** The pages are drafts that W10 to W12 build against, written from ADR-0020 and docs/08-department-brains.md. The order of checks puts the source's ACL last on purpose: the space checks are cheap and local, so they save ACL calls, but nothing they say can replace one. Refusing `spaces:` by name until it is built is the important behaviour change. The old "unknown field" message was already a refusal, but a named one tells an operator why, and the law test pins it, so no future parser change can quietly start dropping the block.

**Learned:** Manual examples tagged `yaml run` go through `sb tenant check` when they contain `tenant:`. The space examples are therefore fragments without it, and only parse until SB-1001, when full manifests with spaces can be checked.

**Laws:** LAW-16 extended (`TestLAW16_UnbuiltFieldsAreRefused`, with its mutation caught). LAW-19 to LAW-23 come in S8.2 as red tests. All law tests are green.

**Next:** S8.2: the five new laws as red tests (SB-804), `Spaces.als` (SB-805), threat model v1 (SB-806) and the space-leak eval scaffold (SB-807).

## 2026-11-19 W8 to W13 planned: department brains and the GCP runtime

**Wave/slice:** W8 to W13 (plan) · **Author:** platform lead + Claude · **Status:** Planned

**What:** [08-department-brains.md](../docs/08-department-brains.md), ADR-0020 (department spaces inside the cell) and ADR-0021 (the GCP cell runtime), five new epics (E09 to E13), seven W8 tickets in E00, and 42 tickets in all across six waves. The PRD, the plan index, open questions Q-11 and Q-12, risks R-13 to R-15 and the glossary follow.

**Why this way:** Companies organise by function and control access by role, attribute, label and barrier, so a department brain is a space with exactly those four layers, inside the company's cell. A cell per department would treat colleagues like strangers. A space can only narrow what the source allows, so ADR-0002 and every existing law still hold, and the new laws (LAW-19 to LAW-23) are about the inside of a company. The GCP runtime comes first (W9) because spaces need a cell that really runs, and it closes most of what W1 to W7 left In review.

**Learned:** The hard part isn't splitting the index, it's the company-wide ask. It has to union what someone may see across spaces without pulling restricted content along, so LAW-20 is the law the design turns on, and the Alloy model checks it before any code.

**Laws:** LAW-19 to LAW-23 proposed, to be written as red tests in W8 (SB-804).

## 2026-11-19 S7.3c The hand-off game day (SB-710)

**Wave/slice:** W7 / S7.3c · **Author:** platform lead + Claude · **Status:** In review (the day with a friendly tenant)

**What:** `tests/handoff` (in `make check`), `docs/runbooks/handoff-game-day.md`, the day's checks in sb-factory(7), and the fix for BUG-0012 in `internal/core/handoff` and its tests and law world.

**Why this way:** A game day is a rehearsal, so the code can rehearse it too. The test wires the real controller to the same adapters a real day uses, including the GCP fake with IAM bindings shaped like the bootstrap script's. Every check on the day is then also a check in CI.

**Learned:** That wiring found BUG-0012 at once. The controller and the access check had two different ideas of what a path is, and every earlier test used fakes shaped to the controller's idea. Fakes that mirror the code under test confirm it rather than test it. The rehearsal is the first test where the paths come from somewhere else.

**Laws:** LAW-11, LAW-17 and LAW-18 checked end to end. All law tests are green. **W7 is built.**

**Next:** what's left is human (reviews, the pilot, game days, the pen test), real cloud and sandbox runs that need the user's accounts, and the runtime that runs cells and the control plane for real.

## 2026-11-19 S7.3b Model billing and cost statements (SB-709)

**Wave/slice:** W7 / S7.3b · **Author:** platform lead + Claude · **Status:** In review (usage meter in a running cell)

**What:** `internal/core/costs`, `GET /v1/admin/costs` with a shared `readers` guard, and a rotation test in `openaicompat`. The manual changed first: sb-tenant(5), sb-admin(3) and sb-factory(7).

**Why this way:** Taking over the model contract needed no new mechanism. The key was already per cell and read per call (ADR-0019), so the option is the vault path the tenant controls, and the test proves the "next call" promise. The statement is built from metering, with list prices from the release. It refuses to print a cost it can't price, because a finance document with a guessed line is worse than an error.

**Learned:** `usage` had its operator-or-approver check inline. A second endpoint with the same audience made it a guard, so the two can't drift apart.

**Laws:** none on this ticket; all law tests are green.

**Next:** S7.3c, the hand-off game day (SB-710), the last W7 ticket.

## 2026-11-19 S7.3a Detach and the export package (SB-708)

**Wave/slice:** W7 / S7.3a · **Author:** platform lead + Claude · **Status:** In review (detach on a running cell)

**What:** `internal/core/export`, custody history, `SetCustody` and `Tombstone` in the local registry, `sb factory verify-export`, and `laws/law11export.go`. The manual changed first: sb-factory(7) gained "Detach, exactly", and sb-factory(1) and sb-audit(5) follow it.

**Why this way:** A company that leaves must be able to prove, without us, that what it took is complete and unaltered. So the package carries its own trust roots (the anchor key and the release key), and the verifier is made of the checks the platform already relies on: the audit chain verifier and a cell's admission. Detach writes and verifies the package before it revokes anything, because revoking first could strand a company with a bad package and no way back in.

**Learned:** File digests alone prove nothing about an audit export cut short with its manifest updated, which is what a careful forger would do. The law test models that forger: every tampering except the byte edit and the extra file also rewrites `MANIFEST.json`.

**Laws:** LAW-11 extended (`TestLAW11_ExportPackageVerifiesOffline`, with its mutation caught). LAW-17 after detach is covered by the controller's Detach in the hand-off law test. All law tests are green.

**Next:** S7.3b, cost statements and a tenant-held model credential (SB-709).

## 2026-11-19 S7.2d Releases after hand-off (SB-707)

**Wave/slice:** W7 / S7.2d · **Author:** platform lead + Claude · **Status:** In review (a cell in a real tenant account)

**What:** `internal/adapters/release/httpstore`, custody, `SetCustody` and `Detach` in `internal/core/rollout`, `laws/law06pull.go` and the tenant-custody rollout world, and `docs/runbooks/pause-pulls.md`. The manual changed first: sb-release(7) gained "Cells in a tenant's account", and sb-factory(7) links to it.

**Why this way:** The factory already never reaches into a cell (cells pull, cells push attestations), so a cell in a tenant's account needs no new trust, only a read token it keeps itself. The risk is in the controller. A guard that quietly skipped cells it can't deploy to would let LAW-08 fail for exactly the tenants who took custody. So custody is explicit on each tenant, only `Detach` takes a cell off the train, and a law test with a mutation pins that down.

**Learned:** Pausing pulls is the tenant's right, but it holds up everyone's promotions once the cell is more than one behind. The runbook says so plainly, and makes "a long pause becomes a detach" the rule instead of an argument.

**Laws:** LAW-06 and LAW-08 extended (two law tests, one mutation caught). All law tests are green.

**Next:** S7.3: detach and the export package (SB-708), billing and the model provider (SB-709), and the hand-off game day (SB-710).

## 2026-11-19 S7.2c Access paths (SB-706)

**Wave/slice:** W7 / S7.2c · **Author:** platform lead + Claude · **Status:** In review (deployed cell, monthly schedule)

**What:** `internal/core/access`, `gcp.IAM` and `gcp.VerifierPermissions`, `getIamPolicy` in `gcptest`, `internal/adapters/access/localfs`, `sb factory access`, and `laws/law17access.go`. The manual changed first: sb-factory(7) gained "Access paths, exactly", and sb-factory(1) and sb-tenant(1) follow it.

**Why this way:** LAW-17 is only as good as the enumeration, so paths are read from where they live (IAM policies and the wrap inventory), never from the plan's list of what should be there. The reader is a separate verifier identity that the tenant grants and can use themselves, so the factory can't make a path invisible by withdrawing its own read rights. And a read that fails counts as a failure, so an outage never passes as "no paths".

**Learned:** The grant a binding serves depends on where it is. `monitoring.viewer` on the project serves `support-metrics`, but the same role on the KMS key serves nothing. The source therefore maps roles to grants only for project bindings, and the law test's model says the same.

**Laws:** LAW-17 extended (`TestLAW17_AccessCheckFindsEveryUnlistedPath`, with its mutation caught). All law tests are green.

**Next:** S7.2d, releases after hand-off (SB-707).

## 2026-11-19 S7.2b The hand-off controller (SB-705)

**Wave/slice:** W7 / S7.2b · **Author:** platform lead + Claude · **Status:** In review (CLI and a cloud Target)

**What:** `internal/core/handoff` (Controller, MemoryStore, Transitions), the tenant-account parser in `cell/custody.go`, and `laws/law17world.go`, which replaces the pending world with the real controller. The manual changed first: sb-factory(7) names the controller, the Target port and the cutover re-check.

**Why this way:** The model is the spec, so the controller's phase table is checked against `Handoff.tla` by reading the file, and a drift between the two fails a test instead of a review. Infrastructure work sits behind one `Target` port, and the key steps go through the verified wrap inventory (SB-704), so the controller's safety doesn't depend on any cloud adapter being right. Cutover re-enumerates the paths after revoking them: if a revoke silently did nothing, the step fails rather than reporting LAW-17 as kept.

**Learned:** A green law test over random runs can still be vacuous if the runs never reach the phases it checks. A self-test now shows the runs reach all nine phases, including `TenantOwned` and `Detached`.

**Laws:** LAW-17 and LAW-18 hand-off tests are green on the real controller, and `laws/expected_red.txt` is empty again. All law tests are green.

**Next:** S7.2c, access-path enumeration (SB-706) and releases after hand-off (SB-707).

## 2026-11-19 S7.2a Key wraps and re-wrap (SB-704)

**Wave/slice:** W7 / S7.2a · **Author:** platform lead + Claude · **Status:** Done

**What:** `internal/core/keywrap`, `internal/adapters/keywrap/localfs`, the audit-signing data key made in `apply`, `key_wraps` in `sb factory status`, and `laws/law18wrap.go`. The manual changed first: sb-factory(7) gained "Key wraps, exactly", and sb-factory(1), sb-laws(7) and the status example follow it.

**Why this way:** A hand-off moves custody of data keys, not data, so the inventory is the thing to get right. Every check that guards a wrap's removal unwraps now, through the kept custodian's own key store, rather than trusting a record that says it worked earlier: KMS access can disappear between two steps.

**Learned:** The law test went red on its first run. Abort removed the tenant's wraps whenever some other wrap existed, even one that no longer unwrapped. An abort after the factory's KMS lost a key would have orphaned the data key, which is exactly LAW-18's failure. Abort is now the destroy guard with the roles swapped, so there is one rule, not two.

**Laws:** LAW-18 extended (`TestLAW18_DestroyNeedsAWorkingWrap`, red first, two mutations caught). The LAW-17 and LAW-18 hand-off tests are still red, as planned, until SB-705.

**Next:** S7.2b, the hand-off controller (SB-705).

## 2026-11-19 S7.1c GCP placement (SB-703)

**Wave/slice:** W7 / S7.1c · **Author:** platform lead + Claude · **Status:** In review (a real GCP project)

**What:** `internal/adapters/placement/gcp` and `gcptest`, `deploy/gcp/bootstrap-deploy-role.sh`, and `laws/law02gcp.go`. The manual changed first: sb-tenancy(7) gained "Placement on GCP, exactly".

**Why this way:** The first real cloud is GCP (the user's decision). Plain REST keeps the adapter within the standard library, so the module allow-list doesn't grow for four small API surfaces. The deploy role is defined once in code, `DeployPermissions`. The script must match it, the fake enforces it, and a test shows every entry is needed, so least privilege is measured rather than claimed. The fake also refuses calls across projects, which is what makes the two-account test mean something.

**Learned:** The first draft of the permission test passed with `useToDecrypt` removed, because stamping never decrypts. The test now wraps and unwraps a data key too. Cloud SQL keeps its labels under `settings.userLabels`, not `labels`, and both the adapter and the fake now read them there.

**Laws:** LAW-02 extended (`TestLAW02_GCPCellsShareNothing`). LAW-17 and LAW-18 are still red, as planned, until SB-705. Everything else is green.

**Next:** S7.2: key re-wrap (SB-704) and the hand-off controller (SB-705).

## 2026-11-19 S7.1b Factory roles (SB-702)

**Wave/slice:** W7 / S7.1b · **Author:** platform lead + Claude · **Status:** Done

**What:** the schema gains `CellCustody` and six custody actions, `base.cedar` the two roles and three forbids, and the policy kernel an explicit-tenant rule for custody records. Tests are in `cedar_test.go` and the LAW-02 policy law. The manual changed first: sb-policy(5) gained "Factory roles, exactly", and sb-factory(7) points to it.

**Why this way:** Every permit in the platform carries `principal.tenant == resource.tenant`, and CI enforces it. The factory roles keep that rule. A custody record belongs to the factory's own tenant and names the cell it is about in a separate attribute, so a factory operator's permit is still tenant-scoped, and no new kind of permit is needed.

**Learned:** The kernel's convenience of filling an empty resource tenant from the caller is a hole for any entity owned by someone other than the caller. A tenant could name a group `sb-factory-operators` and build a custody record with no tenant. Such entities are now listed, and an empty tenant on them is `Unknown`.

**Laws:** LAW-02 extended. LAW-17 and LAW-18 are still red, as planned, until SB-705. Everything else is green.

**Next:** S7.1c, the GCP placement adapter (SB-703).

## 2026-11-19 S7.1a The hand-off, specified (SB-701)

**Wave/slice:** W7 / S7.1a · **Author:** platform lead + Claude · **Status:** In review (security and tenant review)

**What:** `manual/sb-factory.1.md`, a final sb-factory(7) with "The hand-off, exactly", `docs/runbooks/handoff.md`, `laws/law17.go` (the hand-off world, LAW-17 and LAW-18 checks), and a reference controller with four mutants in the self-tests.

**Why this way:** Manual first, and the manual follows the model. The table of phases is `Handoff.tla` written out for people, and the law tests check its invariants after every random operation, so the controller (SB-704, SB-705) has a precise target before a line of it exists. Writing a reference controller from the model shows the checks pass on something correct, and the mutants show they are not vacuous. Without that, a red test proves only that it is red.

**Learned:** The model's dual approval says nothing about who approves, so a planner approving their own plan would satisfy it. The manual now requires a second factory operator, and the law checks it. The model may follow (W7 follow-up).

**Laws:** LAW-17 and LAW-18 move to Tested (red) through the ratchet, with two lines in `expected_red.txt`, both SB-705. Everything else is green. LAW-17 now has a **Watch** line, so the alert test required a rule: `sb_unlisted_access_paths` above 0 is Sev-1, with `docs/runbooks/LAW-17.md`. It fires once SB-706 emits the metric.

**Next:** S7.1b, the factory roles (SB-702).

## 2026-11-19 S6.4b Pen-test readiness (SB-612)

**Wave/slice:** W6 / S6.4b · **Author:** platform lead + Claude · **Status:** In review (the external test is human)

**What:** `security/` (scope, rules of engagement, `findings.yaml`, `attacks/injection.txt`), `tools/check/findings.py` in `make check`, three law tests (`law02attack.go`, `law07encoded.go`, `law09corpus.go`), and fixes in `internal/obs` (encoded secrets, credential-named attributes) and `internal/core/ingest` (Markdown role labels). The manual changed first: sb-laws(7) gained External review, and sb-observability(7) the redactor's encodings. The threat model was refreshed.

**Why this way:** A pen test is worth most when its findings can't be forgotten, so the register is checked by the build: a critical or high finding that isn't fixed fails `make check`. Before the outside team starts, we attacked the three laws in the ticket ourselves the way a tester would. Two of the three tests went red, which makes them the register's first two entries, recorded and fixed the same way an external finding would be.

**Learned:** The LAW-07 and LAW-09 tests were too polite. One only logged secrets made of letters, digits and dashes, exactly as written. The other drew from five fixed attacks, none of them in Markdown. Both now generate hostile inputs, and the corpus is a file a tester can add to without writing Go.

**Laws:** LAW-02, LAW-07 and LAW-09 extended. LAW-07 and LAW-09 went red first and are green after the fixes, and LAW-02 was green first time. All law tests are green.

**Next:** W7.

## 2026-11-19 S6.4a DR game day tooling (SB-613)

**Wave/slice:** W6 / S6.4a · **Author:** platform lead + Claude · **Status:** In review (the game day itself is human)

**What:** `internal/core/gameday` (Restore, Score, SameIndex), `tests/gameday` (the rehearsal, now in `make check`), `laws/law11restore.go`, and `docs/runbooks/dr-game-day.md`. The manual changed first: sb-observability(7) gained the game day's scenario, targets and scoring.

**Why this way:** A game day is only worth something if it's scored the same way every time, so the score is code and a day passes only when every line passes. The restore reuses the drill's per-copy checks instead of adding new ones. It writes only into empty stores and checks each row's `seq` and `prev` as it goes, so a restore can't splice two chains together. When two good copies disagree there's no safe automatic choice, so it stops and a human chooses.

**Learned:** The architecture page promises an RPO of 15 minutes from continuous log archiving, but the local adapters only back up nightly. The rehearsal now says so plainly by failing the RPO when the backup is an hour old, instead of hiding it. The real RPO comes with W7's cloud adapter.

**Laws:** LAW-11 extended (`TestLAW11_ChainSurvivesRestore`, with its mutation caught). LAW-13 is covered by the rebuild drill and checked again in the rehearsal. All green.

**Next:** S6.4b, the pen test and threat model refresh (SB-612).

## 2026-11-19 S6.3c Retention and deletion (SB-611)

**Wave/slice:** W6 / S6.3c · **Author:** platform lead + Claude · **Status:** In review (nightly schedule, W7)

**What:** `internal/core/retention` (Sweep, Erase), retention views on the trace store, `internal/adapters/logs/localfs` (daily log files that can expire and erase), `POST /v1/admin/subjects/erase`, the `retention` audit kind, and `laws/law13retention.go`. The manual changed first: sb-tenant(5) gained Retention and deletion, sb-admin(3) the endpoint, and sb-audit(5) the kind.

**Why this way:** Retention is a list of stores that the job is given, and the index and audit are deliberately not on it. The index is derived and ages with its sources (LAW-13). Audit has its own seven-year WORM policy. The law test makes that a checked property instead of a convention. Erasure audits first, the same way the tool bus does (LAW-04): an erasure nobody can prove was asked for is worse than one that waits for the audit log to come back.

**Learned:** Conversations had no timestamp, so they couldn't be aged; they now carry the time of their last question. Log erasure matches the user id as a whole JSON value, never as a substring, so erasing `usr_ann` keeps `usr_anna`'s lines.

**Laws:** LAW-13 extended (`TestLAW13_RetentionLeavesTheIndexDerived`, with two mutations caught). LAW-11 accepts the new kind. All green.

**Next:** S6.4, the pen test and threat model (SB-612) and the DR game day (SB-613). Both are human-led, so this slice builds their tooling and runbooks.

## 2026-11-19 S6.3b HubSpot CRM connector (SB-610)

**Wave/slice:** W6 / S6.3b · **Author:** platform lead + Claude · **Status:** In review (real sandbox)

**What:** `internal/adapters/connectors/hubspot` and its recorded fake `hubspottest`, plus `laws/law01crm.go` (the CRM LAW-01 world, the reference model, and the LAW-03 world). The manual changed first: sb-connector(3) gained `crm: hubspot`, and sb-permissions(7) the `hubspot:users` broad token.

**Why this way:** HubSpot's API doesn't expose permission sets, so the tenant states each role's view level (`all`, `team` or `owned`) in the vault next to the token. A role that isn't stated is `Unknown`, not a guess. The prefilter is the owner's email plus a broad token, so role and team edits can't leave a stale prefilter that hides a record from someone who can now see it, and the live check does all the deciding. Contacts are personal data, and we don't index them.

**Learned:** The contract kit wants reassignments reported as permission-only changes. HubSpot's search results carry `hubspot_owner_assigneddate`, and comparing it with `hs_lastmodifieddate` and `createdate` tells a reassignment apart from an edit or a new record without keeping any per-record state in the cursor.

**Laws:** LAW-01 extended (`TestLAW01_CRMRecordPermissionsAreLive`, with its mutation caught), and LAW-03 over the new connector. All green.

**Next:** S6.3c, retention and deletion (SB-611).

## 2026-11-19 S6.3a Alerts, telemetry and budgets (SB-607 to SB-609)

**Wave/slice:** W6 / S6.3a · **Author:** platform lead + Claude · **Status:** In review (running cell, staging)

**What:** `internal/core/alerts` with ten runbooks, `obs.Catalog`, histograms, the label rule, ask and ACL metrics, a tenant dashboard, and `internal/core/budget` in the gateway. The manual changed first: sb-observability(7) gained "Law alerts, exactly", "Telemetry labels" and "Budgets and rate limits", and sb-tenant(5) the budget's meaning.

**Why this way:** Law alerts are driven by the manual. A test reads sb-laws(7), so a law that gains a production check can't ship without an alert and a runbook. The label rule fails closed: a bad value becomes `invalid` rather than being trusted, and it is counted so the bug is visible. The budget check uses the call's worst case before calling and real usage after, so a call can't overshoot because tokens are only known later. A meter that can't be read refuses, because budgets that fail open are not budgets.

**Learned:** `ModelRequest` had no idea who a call was for, so the rate limit needed a `User` field. Adapters never serialise it, and their tests show the request bodies, so it can't leak to a provider.

**Laws:** unchanged and all green; LAW-07 is reinforced by the label rule.

**Next:** S6.3b, the HubSpot CRM connector (SB-610) and retention and deletion (SB-611).

## 2026-11-19 S6.2 Onboarding and the weekly report (SB-604 to SB-606)

**Wave/slice:** W6 / S6.2 · **Author:** platform lead + Claude · **Status:** In review (dry run, friendly tenant, delivery)

**What:** The runbook's steps and gates tightened and checked by a lint, `internal/core/report`, and the usage endpoint. The manual changed first: sb-onboarding(7) now gives an expected result for every step and a checklist for every gate, and sb-observability(7) defines every line of the weekly report.

**Why this way:** A runbook a company's IT team follows unaided can't have a step whose success they have to guess, so the lint makes "every step says what to expect" a build rule rather than a style hope. The report counts gaps by kind, not by question, because a "top unanswered questions" list would be content. It is built from the same records the rest of the platform keeps, so there is no second analytics store to secure.

**Learned:** The first lint run found three steps without expectations, two of them on day 0, the first day a company meets the platform. The tenant-eval step had also pointed at `--model mock`, which needs canned responses a tenant doesn't have; it now uses the local cell.

**Laws:** unchanged, all green.

**Next:** S6.3: per-tenant observability (SB-607), a Sev-1 for every law violation (SB-608), cost controls (SB-609), the HubSpot CRM connector (SB-610), and retention and deletion (SB-611).

## 2026-11-19 S6.1b Admin console and roles (SB-602, SB-603)

**Wave/slice:** W6 / S6.1b · **Author:** platform lead + Claude · **Status:** SB-603 Done, SB-602 In review (manifest repo integration)

**What:** The operator role in base policy, `manifest.Format`, `Patch` and `Diff`, the admin API, and the web console. The manual changed first: sb-admin(3) gained "Roles, exactly" and "The manifest diff, exactly".

**Why this way:** The console proposes manifest changes and never applies config directly. The manifest stays the only way the cell's configuration changes (LAW-16), and every proposal is audited even if it's never applied. The patch goes through the exact checker `sb tenant check` uses, by formatting the patched manifest and checking that text. Validation has one implementation, and `Format`'s round-trip test keeps it honest. Kill switches are the one thing that acts at once, as the manual says, and "make permanent" is just a proposed diff.

**Learned:** Writing `Format` needed no new parser knowledge: defaults the checker applies come back out explicitly, so a formatted manifest is a fully spelled-out one. That's exactly what a reviewer of a console proposal wants to see.

**Laws:** all green. LAW-02 at the policy level holds with the new operator permit, and LAW-16 now covers console changes too.

**Next:** S6.2, onboarding: the runbook (SB-604), a friendly tenant unaided (SB-605, human), and the weekly leadership report (SB-606).

## 2026-11-19 S6.1a Tenant verify (SB-601)

**Wave/slice:** W6 / S6.1a · **Author:** platform lead + Claude · **Status:** In review (a deployed cell's endpoint, W7)

**What:** `internal/core/verify` and a local-cell target in `tools/evalcell --verify`. The manual changed first: sb-tenant(1) gained "How verify decides, exactly".

**Why this way:** The laws suite proves the code over generated worlds. Verify asks the same questions of one tenant's real data, so a misconfigured connector or a source whose permission model we misread shows up before staff do. LAW-01 re-asks the source live instead of trusting the answer's own record, and also requires that record, because "the source said yes later" isn't the same as "the source was asked". The target interface is the seam between local and deployed cells: the runner never changes, only the target does.

**Learned:** The local target needed the eval cell's construction as a reusable function. That refactor (`build`) now serves both the eval harness and verify, so the two can't disagree about what a cell is. The issuer rule kept the target inside `tools/evalcell`, the one harness allowed to mint scopes, rather than in a new production package.

**Laws:** LAW-01, LAW-02, LAW-03 and LAW-14 are checked by verify as well as by the laws suite. All green.

**Next:** S6.1b, the admin console and API, where every change is a manifest diff and an audit row (SB-602), and console roles (SB-603).

## 2026-11-19 S5.3b Attestation and drift (SB-509, SB-510), W5 built

**Wave/slice:** W5 / S5.3b · **Author:** platform lead + Claude · **Status:** In review (cell loops, W7)

**What:** `internal/core/attest` (signed cell reports and the controller's check) and `internal/core/drift` (live config against the applied manifest). The manual changed first: sb-release(7) gained the attestation rules and findings, and sb-tenant(5) gained "DRIFT".

**Why this way:** Cells push and the controller only listens, so attestation works for a cell in any account, including after hand-off (W7). The report is signed with the cell's own key, so a compromised control plane can't fake a healthy cell and a compromised cell can't impersonate another. `seq` makes a replayed good report useless. Silence is a finding in its own right, because a cell that stops reporting is the one to worry about. Drift reports and never repairs: repairing would make the live state the source of truth, which LAW-16 forbids.

**Learned:** The drift law's first draft compared only attributes, and its mutation (missing whole resources) showed why the property must also require resources present on one side only. W5 is built. Releases are bundles with derived ids; they are reproducible, SBOM'd and signed (keyless in CI), admitted only when gated, rolled out by a controller that is the TLA+ model in code, attested while running, and checked for drift.

**Laws:** all green, LAW-06 and LAW-16 extended.

**Next:** W6, starting with `sb tenant verify` (SB-601).

## 2026-11-19 S5.3a The release controller and rollback (SB-507, SB-508)

**Wave/slice:** W5 / S5.3a · **Author:** platform lead + Claude · **Status:** In review (real cells, W7)

**What:** `internal/core/rollout`, a timed rollback drill, and `sb release status|promote|rollback`. The manual changed first: sb-release(1) gained "How the controller decides, exactly".

**Why this way:** The controller is a transcription of `Rollout.tla`, with the same moves, guards and names, and it checks the model's invariants after every move. A bug that slips past a guard can therefore never reach a state the model forbids; the move is refused instead. That's the same defence in depth as the policy kernel: the guard is the rule, the invariant check is the proof. Soak time is measured per ring from the first deploy. Draining comes straight from BUG-0001, and a failed redeploy leaves a cell draining, never back on the revoked build.

**Learned:** The first cut of the LAW-08 mutation (promoting without the guard) was caught twice over: by the law, and by the controller's own invariant check on the next promotion. That is the point of having both.

**Laws:** LAW-08 moves from Stated to Tested. Every law test is green.

**Next:** S5.3b, running-build attestation per cell (SB-509) and manifest drift detection (SB-510).

## 2026-11-19 S5.2b Lockfiles and the CVE path (SB-505)

**Wave/slice:** W5 / S5.2b · **Author:** platform lead + Claude · **Status:** In review (first scheduled run, testing channel)

**What:** `make lockfiles`, a lockfile-currency step in the release workflow, a daily advisory workflow, and the patch-release playbook. The manual changed first: sb-release(7) "CVEs" now says exactly what is regenerated, what the watcher runs, and the 24-hour path.

**Why this way:** "Regenerated every release" is enforced by failing the release when regenerating would change a lockfile, not by regenerating silently in CI. That way every dependency change is a reviewed commit, and the build stays reproducible from the repository alone. `govulncheck` reports only reachable vulnerabilities, so the watcher's failures are ones that matter, and the playbook doesn't train people to ignore it. The playbook skips nothing: a CVE fix goes through the same gate and signature as any release.

**Learned:** Nothing surprising. Lockfiles were current, and `npm audit` was clean.

**Laws:** unchanged, all green. LAW-08 comes with the promote guard (SB-507).

**Next:** S5.3, the release controller with channels and rings and the promote guard (SB-507, LAW-08), one-step rollback (SB-508), running-build attestation (SB-509), and manifest drift detection (SB-510).

## 2026-11-19 S5.2a Reproducible builds, SBOM, signing and admission (SB-504, SB-506)

**Wave/slice:** W5 / S5.2a · **Author:** platform lead + Claude · **Status:** In review (first CI signature, cell pull loop)

**What:** A reproducible-build check, an SPDX SBOM generator, `make release-build`, a release workflow with cosign keyless signing, `release.Admit` with Ed25519 and cosign verifiers, a local release store, and LAW-06 green. The manual changed first: sb-release(7) gained the exact build, SBOM, signing and gate rules, and "What a cell admits".

**Why this way:** The signature covers one digest, the bundle's, and the bundle lists every artefact's digest, the SBOM's included. So one signature check plus digest checks covers everything a cell runs. The gate's verdict is a signed record of its own, bound to the release id, so passing the gate can't be claimed by editing a file. Both verifiers refuse a signature whose key or identity isn't the one the cell trusts. The cosign adapter refuses when cosign is missing: no verifier means no start. Cells pull, so the factory never needs a way in.

**Learned:** The first LAW-06 world rebuilt identical bundles and found them "already signed", which is correct: same contents, same release. The law now asks for a fresh bundle when it wants an unsigned one. A safety check refused a repro script that cleaned up with `rm -rf` on a variable, so the script now cleans its caches with `go clean -cache` and never removes a directory. Vendored builds carry no module hashes in their build info, so the SBOM takes them from `go.sum`.

**Laws:** LAW-06 moves from Tested (red) to Tested. Every law test is green.

**Next:** S5.2b, lockfile regeneration and the CVE patch path (SB-505).

## 2026-11-19 S5.1b Eval suites and the gate (SB-501, SB-502)

**Wave/slice:** W5 / S5.1b · **Author:** platform lead + Claude · **Status:** In review (staging runs, branch protection)

**What:** Four new shared suites with their worlds, fault injection and stale prefilters in the local cell, results stored per release, and `evals gate`. The manual changed first: sb-eval(1) gained `--results`, `--release` and `gate`, and sb-evals(7) "The suites, exactly".

**Why this way:** The gate reads stored results instead of running suites itself, so the same results can be checked again, compared with stable, and archived with the release. A missing result fails: "we didn't run it" must never look like "it passed". Scores mean different things per suite (support for citation, pass rate for the rest), so the gate file stays the same shape as in the manual.

**Learned:** The first injection world hid a canary between zero-width characters, and the scanner-minded test "failed": the sanitiser stripped the invisible characters and left the canary, which was visible text all along. Real hidden text lives in comments and hidden styles, so that's where the world now hides it. The local cell's refusal case also had to change: a stand-in model answers from whatever it is given, so "refuse" on the local cell means "nothing allowed came back". Judging a model's own refusals needs the real model on staging.

**Laws:** all green except LAW-06 (red until SB-506).

**Next:** S5.2, reproducible builds with an SBOM and signing (SB-504), lockfiles and the CVE path (SB-505), and deploy-time verification (SB-506).

## 2026-11-19 S5.1a The release bundle (SB-503)

**Wave/slice:** W5 / S5.1a · **Author:** platform lead + Claude · **Status:** In review (signing and admission next)

**What:** `internal/core/release` (bundle, canonical form, id, check), `internal/core/canonjson` (shared with approvals), `tools/release/bundle`, `release/model.json`, and the LAW-06 law test, red. The manual changed first: sb-release(7) now gives the bundle's fields and the id rule.

**Why this way:** The id is derived from the contents, so "any change is a new release" is arithmetic, not a process rule. Tool schemas are hashed as text, pinned with the release, so a change to what an approver can be shown is a release too. The model lives in a file in the repository, so changing it is a reviewed commit and a new id, never a runtime setting (ADR-0013). Canonical JSON had one home in approvals. A second user meant moving it to a package of its own, rather than having release import approvals.

**Learned:** The bundler's own test runs against the real repository. That way, a new prompt file or policy is included automatically, and a missing input fails the build rather than producing a bundle that quietly omits it.

**Laws:** LAW-06 moves from Stated to Tested (red), for the right reason, until SB-504 and SB-506.

**Next:** S5.1b, the eval suites with results stored per release, and the gate (SB-501, SB-502).

## 2026-11-19 S4.4b Backups and the restore drill (SB-412), W4 built

**Wave/slice:** W4 / S4.4b · **Author:** platform lead + Claude · **Status:** In review (cloud WORM bucket, monthly schedule)

**What:** `internal/core/backup` (the 3-2-1 check, the nightly job, the restore drill and its report), a write-once local target, and an end-to-end test. The manual changed first: sb-audit(7) "Retention and backup" now says exactly what is copied, where, for how long, and what the drill checks.

**Why this way:** The backup reuses the signed export, so a restored copy verifies with the same `audit.Verify` and pinned key as any export, with no second verifier to maintain. The 3-2-1 rule is checked by the job itself, not assumed from configuration, and a write-once target means a compromised job can add a night but never rewrite one. The drill checks every copy, not just one, because the copy you'd restore from after an incident is the one nobody looked at.

**Learned:** The first version of the drill test lived in the local target's package and imported the memory stores, which the dependency rule forbids for sibling adapters. Moving it to `tests/backup` is the honest place anyway: it is a system test of three packages working together.

**Laws:** LAW-11 now also holds over backups. All laws green. W4 is built: every tool call goes through the bus, every approval is bound to its payload, every outcome needs a receipt, and the audit chain is verified, backed up and drilled.

**Next:** W5, evals against real cells and the release gate (SB-501 onward).

## 2026-11-19 S4.4a Canaries and the secret scanner (SB-411)

**Wave/slice:** W4 / S4.4a · **Author:** platform lead + Claude · **Status:** In review (running-cell wiring, source sandboxes)

**What:** Canaries, a scanner, Sev-1 alerts through a webhook, redaction in the audit log, the gateway hook, and a CI scan. The manual changed first: sb-observability(7) gained "Canaries and the secret scanner", sb-audit(5) the redaction rule, sb-tenant(5) the `canary` vault path, and sb-laws(7) the checks.

**Why this way:** A canary is a secret nothing legitimate ever uses, so any sighting is a leak, with no false positives to tune. The scanner learns every resolved secret too, through the same registrar as the redactor, so the two can't drift apart. Findings and alerts carry ids and line numbers, never values: an alert that quoted the leak would be a second leak. The audit log scrubs before hashing, so the chain verifies over what was actually stored.

**Learned:** The first CI run found the model API key "leaking" to the model provider. It was in the `Authorization` header, which is exactly where it should go. The scan was wrong, not the code: a prompt is the request body. That's the scanner doing its job, and it sharpened the manual's definition of where prompts are scanned.

**Laws:** LAW-07 now covers audit rows, with a red-first law test and a mutation. All laws green.

**Next:** S4.4b, backups of the audit chain and approvals (3-2-1, WORM) and the monthly restore drill (SB-412).

## 2026-11-19 S4.3c draft-reply (SB-409)

**Wave/slice:** W4 / S4.3c · **Author:** platform lead + Claude · **Status:** Done

**What:** The draft-reply agent, its API and outcome metric, a draft panel in the web UI, a `draft` prompt, and a base permit for an enabled `draft-reply`. The manual changed first: sb-actions(3) now says exactly how a draft is made, and adds the outcome endpoint.

**Why this way:** A draft reuses the ask pipeline (live check, composer, trace) with one addition: the ticket or thread being answered is live-checked as an item in its own right before its text is used. Drafting on a ticket you can't open would otherwise leak it through the draft. The agent has no tools at all, so "a human always sends" is structural, not a rule someone has to remember. The base permit needs an `enabled` attribute, optional in the schema so older entities still evaluate cleanly, and tenant policies can then narrow by group or by ticket project, as the example policy already does for `FIN`.

**Learned:** Making `enabled` a required attribute would have broken every older fixture with evaluation errors, which cedar turns into Unknown. That would be safe, but every existing test would change meaning. Optional plus `has` keeps old entities at Deny and new ones explicit.

**Laws:** all green. LAW-14 holds through the same composer, and drafts make no tool calls (LAW-04).

**Next:** S4.4, secret canaries and the scanner (SB-411), and audit and approval backups with a restore drill (SB-412).

## 2026-11-19 S4.3b The ticket-update workflow (SB-410)

**Wave/slice:** W4 / S4.3b · **Author:** platform lead + Claude · **Status:** In review (real Linear sandbox)

**What:** The workflow template and registry, Linear writes with `Find`, the eval gate in apply, `--record` in the eval harness, and an end-to-end test from proposal to Linear receipt. The manual changed first: sb-approvals(7) gained "The ticket-update workflow", sb-tenant(1) "Workflows wait for their evals", sb-connector(3) Linear's write scope and `Act` row, and sb-eval(1) `--record`.

**Why this way:** The comment's id comes from the idempotency key, so Linear itself enforces "once per approval". A retry, a lost answer or the reconciler all land on the same comment. The connector never returns success without a receipt: an answer with no id is read back by that id, or it is an error. That keeps LAW-05 true at the connector, not just in the bus. The eval gate refuses the whole plan before any step runs, so an apply never half-enables a workflow. The template's payload is strict: an extra field is refused, because an approver shown `{comment}` must not approve a hidden `assignee` change.

**Learned:** Building the end-to-end test from the example manifest and the real base policies caught nothing new, and that is the point: it proves the pieces from four slices fit, with the audit trail in the order the model says. The Linear law world first shared one fake across runs, and effects leaked from one seed into the next. Connector-level law worlds need a fresh target per run.

**Laws:** all green. LAW-04, LAW-05 and LAW-15 are extended end to end, and LAW-05 also holds at the Linear connector.

**Next:** S4.3c, the draft-reply agent (SB-409).

## 2026-11-19 S4.3a Idempotency store and reconciler (SB-408)

**Wave/slice:** W4 / S4.3a · **Author:** platform lead + Claude · **Status:** In review (Postgres stores, real `Find`)

**What:** `toolbus.Receipts` and its memory store, `unconfirmed` outcomes, and `Bus.Reconcile` with `RunReconciler`. The manual changed first: sb-approvals(7) now states the idempotency store, what "unconfirmed" means, and the reconciler's schedule and limits.

**Why this way:** The receipt store is keyed by the approval id, the same key the target sees. So the bus's own state, the receipt store and the target all agree on "this ran once", and losing one of them can't cause a second run. Only errors and timeouts are unconfirmed: a target that answered without a receipt has told us what happened. The reconciled action stays `DidNotRun`, because that state is terminal in `Approvals.tla`. The receipt rides alongside it, and the audit log gets the `Ran` row that LAW-05 needs. That keeps the model and the code in step without a new transition.

**Learned:** It was tempting to let the reconciler move the action to `Ran`. That would be a transition the model doesn't have, and proving it safe would mean changing the TLA+ spec. Recording the receipt without the state change keeps every proof as it is.

**Laws:** all green. LAW-05 also holds across reconciliation, because only a found receipt makes a `Ran` row.

**Next:** S4.3b, the `ticket-update` workflow end to end: Linear writes, the manifest's eval gate, and draft to receipt (SB-410).

## 2026-11-19 S4.2b The actions API and the approver view (SB-407)

**Wave/slice:** W4 / S4.2b · **Author:** platform lead + Claude · **Status:** In review (notification adapters, human WCAG pass)

**What:** HTTP handlers for actions over the tool bus, `Bus.Get` with access rules and a field diff, policy reasons and id-only notifications from the bus, and the web approver view. The manual changed first: sb-actions(3) gained the propose body, the read rules and fields, "The approver view", and the notification rule.

**Why this way:** The approver view sends back the hash it displayed, not one it fetches again, so the server's hash check (LAW-15) also covers "the page went stale". A test proves it by changing the payload after the page loads. The diff compares canonical values field by field, so a key-order change isn't shown as a change. Reading an action is limited to the requester, the approver group and auditors, and anyone else gets the same answer as for an unknown id, so ids can't be probed.

**Learned:** Go's JSON gives the shallower of two same-named fields precedence. That's handy for one response shape where `diff` must be `null` rather than missing, but it is easy to misread, so the handler builds that shape explicitly instead of relying on `omitempty`.

**Laws:** LAW-15 now also covers the UI. Every law test is green.

**Next:** S4.3, the reconciler for unknown outcomes (SB-408), draft-reply (SB-409) and the first gated workflow template (SB-410).

## 2026-11-19 S4.2a The tool bus, approvals and the executor (SB-405, SB-406, SB-408)

**Wave/slice:** W4 / S4.2a · **Author:** platform lead + Claude · **Status:** In review (bundle registry, reconciler, Postgres store)

**What:** `internal/core/toolbus`, `internal/core/approvals`, a memory action store, a lint that only the bus calls `Act`, and three tenant-scoped base permits. The manual changed first: sb-agents(7) gained the bus's two paths and the lint, and sb-approvals(7) gained "The payload hash, exactly" and "Approving, exactly".

**Why this way:** One package owns the whole order (kill, authorise, audit, approve, execute), so there's no second path to get wrong, and the lint keeps it that way. The hash covers the target and the workflow as well as the payload, so an approved comment can't be replayed on another ticket. The approval checks run in a fixed order, and each failure names its reason. The approval row is written before the state moves, and moves are compare-and-set, so two approvers racing can't both run it. Payload numbers are integers within ±2^53, because RFC 8785 float formatting is where implementations disagree.

**Learned:** With the bus in, `laws/expected_red.txt` is empty for the first time: every law test runs against real code. One trap in the law harness: the bus writes the `Executing` row inside `Invoke`, before it returns the action id, so the world must learn the id when it is minted, not when `Invoke` returns. A second fix along the way: `decision` rows had the action in `reason` twice.

**Laws:** LAW-04 and LAW-05 move from Tested (red) to Tested. LAW-15 moves from Stated to Tested. `make proofs` and `make crosscheck` pass with the new permits.

**Next:** S4.2b, the actions API (`POST /v1/actions`, approve, reject) and the approver view (SB-407).

## 2026-11-19 S4.1b Kill switches and the kill drill (SB-403, SB-404)

**Wave/slice:** W4 / S4.1b · **Author:** platform lead + Claude · **Status:** In review (tool-bus check points, push channel, release pipeline)

**What:** `internal/core/kill`: an audited `Control`, the per-cell `Checker` and the `Drill`. There are memory and local-control-plane stores, and `sb kill`, `sb unkill` and `sb kill list`. The manual changed first: sb-kill-switches(7) gained "How it works, exactly" and the drill's procedure, and sb-kill(1) the local control plane, the actor and the list output.

**Why this way:** A 2-second poll leaves 8 seconds of margin under the 10-second bound even if every push is lost, and push only makes it faster. The checker keeps its last good list when a reload fails, because dropping known switches on an error would be the wrong way to fail. After 30 seconds it fails safe for consequential calls only, so reads keep working through a control-plane outage. Kill and unkill write their audit row first, so a switch never changes untraced (LAW-04 in spirit). The drill goes through the same `Control`, so drills are audited like real kills.

**Learned:** The first version of the timed law test was wrong: it expected a kill set during an outage to take effect within 10 seconds, which no cell can do if it can't see the store. The manual's rule is the fail-safe. The law now counts a kill from when the cell could first see it, and checks the 30-second fail-safe separately. Putting the law in front of the manual sharpened both. The smoke test that expected `sb kill` to be `ENOTIMPL` now expects `EUSAGE` without `--reason`, plus a new end-to-end kill, list and unkill test. Nothing was deleted.

**Laws:** LAW-10 is Tested. LAW-04 and LAW-05 are still red until the tool bus and the executor.

**Next:** S4.2, the tool bus (SB-405, which turns LAW-04 green), approvals bound to content (SB-406, LAW-15) and the approver view (SB-407).

## 2026-11-19 S4.1a The audit chain (SB-401, SB-402)

**Wave/slice:** W4 / S4.1a · **Author:** platform lead + Claude · **Status:** In review (tool-bus hook, cell runtime)

**What:** `internal/core/audit` has rows, canonical form, chain hashing, signed anchors, export and an offline verifier. There is an append-only Postgres store and a memory store, and `sb audit verify --export`. The manual changed first: sb-audit(5) gained "Canonical form and hash, exactly", "Storage" and "Anchors", and the export now starts with an anchor. sb-audit(1) documents the offline verify and `--key`.

**Why this way:** Only the store holds the write lock, and it hands the last seq and hash to the core's `build`. So the hashing rules live in one place, and Postgres can serialise with an advisory lock without knowing them. Rows are append-only in two independent ways: triggers refuse UPDATE, DELETE and TRUNCATE whatever the role, and the writer role holds only INSERT and SELECT. The anchor key is the cell's own: its seed is a data key under the cell's root key, never a factory-wide key (ADR-0015). Verify pins a public key, because a file that carries its own key proves only that someone signed it.

**Learned:** A tamper the law test found by design: an export that starts mid-chain can't tell a dropped first row from a later start. The fix is a signed start anchor on the row before the first exported row. RFC 8785 and Go's JSON disagree on HTML escaping and on `U+2028`, so the canonical encoder turns off HTML escaping, and rows refuse those characters outright. Shell heredocs mangled `\n` in Go string literals twice, so Go code with escapes now goes through the Edit tool only.

**Laws:** LAW-11 moves from Stated to Tested. LAW-04 and LAW-05 are still red until the tool bus (SB-405) and the executor (SB-408).

**Next:** S4.1b, the kill switches (SB-403) and the kill drill (SB-404), with a LAW-10 timed test.

## 2026-11-19 S3.8 Pilot tooling (SB-311), W3 built

**Wave/slice:** W3 / S3.8 · **Author:** platform lead + Claude · **Status:** In review (the pilot itself is human work)

**What:** The pilot review runbook in sb-onboarding(7), a Gate 1 report from the weekly feedback export, and a retro template. This ends W3. Done: SB-301, SB-302, SB-304 and SB-305. In review: SB-303, SB-306 to SB-311, each for a named human or staging item.

**Why this way:** The pilot's numbers come from the export, and the export holds ids and counts only, so the report can be shared with leadership without leaking content. Every wrong-citation report becomes a tenant eval case, so pilot findings turn into regressions that stay tested forever.

**Learned:** "No feedback" has to fail the gate. Dividing by zero would otherwise read as a pass at 0%, or as a crash.

**Laws:** W3 ends with LAW-01, 02, 03, 07 and 14 green on the real pipeline. LAW-04 and LAW-05 (the tool bus) are red until W4.

**Next:** W4, the tool bus (SB-405) and receipts (SB-408) first, to turn LAW-04 and LAW-05 green.

## 2026-11-19 S3.7 Feedback (SB-310)

**Wave/slice:** W3 / S3.7 · **Author:** platform lead + Claude · **Status:** In review (weekly schedule with the cell runtime)

**What:** `POST /v1/feedback`, the feedback store, a weekly export and the web UI controls. The manual changed first: sb-ask(3) now gives the endpoint's rules and the export's contents.

**Why this way:** Feedback goes through `Agent.Trace`, so the trace's ownership check is the same code that guards `GET /v1/traces`, with no second copy to drift. Only the asker rates: an auditor reading a trace isn't the person the answer was for. The export carries ids, counts and the user's own comment, never the question or the answer (AGENTS.md: no content in exports, logs or golden files). A reviewer who needs the text opens the trace with auditor rights, and that access is audited.

**Learned:** It is tempting to offer every citation in the wrong-citation picker. Limiting it to cited claims keeps the UI consistent with the rule that uncited claims have no citations (LAW-14).

**Laws:** none new. LAW-04 and LAW-05 are still the only red ones.

**Next:** SB-311, the pilot. It is human-only, so it gets a runbook, and then W4.

## 2026-11-19 S3.6 Citation-faithfulness eval (SB-309)

**Wave/slice:** W3 / S3.6 · **Author:** platform lead + Claude · **Status:** In review (`sb tenant verify`, staging model)

**What:** A strict lexical support scorer. `tools/evalcell`, a local cell that runs the real ask path over a fixture world. The shared `citation` suite, and per-tenant golden questions in `tenants/<name>/evals/`. All of it runs in `make evals`, including a self-test that a misattributing model fails. The manual changed first: sb-eval(5) gained Faithfulness, Worlds for the local cell and Tenant golden questions, sb-eval(1) the new models and `--tenant-dir`, and sb-evals(7) Faithfulness.

**Why this way:** The scorer is lexical, and it zeroes any claim with a number the source doesn't say. It can't be fooled into a high score, which matters more for a gate than catching every good paraphrase. A model judge can replace it once one is pinned in a release. The eval cell is a separate tool, not a CLI command, so the product CLI stays as specified. It reuses every production package and swaps only the model for a deterministic one. A tenant's gate may only be stricter than the shared gate.

**Learned:** Building the local cell found BUG-0007: the agent never gave connectors the asker's email, so real connectors would have refused everything. Our fakes keyed on ids, real connectors key on emails, and no test put the two together. The eval cell now does, on every `make check`. Also, chunk spans index into the normalised "title, blank line, text", not the text alone. That's easy to get wrong when quoting a span back.

**Laws:** LAW-14 is now measured as well as enforced. LAW-04 and LAW-05 are still the only red ones.

**Next:** SB-310, feedback capture (`POST /v1/feedback`).

## 2026-11-19 S3.5 Slack `/ask` (SB-308)

**Wave/slice:** W3 / S3.5 · **Author:** platform lead + Claude · **Status:** In review (real Slack workspace)

**What:** The Slack app as a second entry point to the same ask agent: a signed slash command and a Share button. The manual changed first: sb-ask(3) gained "The chat app (Slack)", sb-identity(7) gained "Chat users", and sb-connector(3) notes the one added scope, `commands`.

**Why this way:** Replies go through Slack's `response_url`, so the bot needs no Slack write scope at all, and every reply is ephemeral. The identity mapping needs exactly one active directory user for the Slack email, checked on every request. Two matches, an inactive user or no match means no answer, never a guess. Share posts citation links only, so a reader who can't open a source learns nothing from the share. That keeps LAW-01 true for everyone in the channel, not just the asker. `response_url` must be on `hooks.slack.com`, so a forged request can't make the cell POST anywhere else.

**Learned:** The law-test harness could reuse the ask agent by pulling its construction into `askSystem.agent`, so the chat law runs the same cell as the web law. The red-first step worked as intended here: a pending `NewChatAsk`, red for the right reason, then the real app. The mutation that answers as the tenant's first user is caught because the live ACL is asked about the wrong person.

**Laws:** LAW-01 extended to the chat path. LAW-04 and LAW-05 are still the only red ones.

**Next:** SB-309, the citation-faithfulness eval.

## 2026-11-19 S3.4 Web chat UI (SB-307)

**Wave/slice:** W3 / S3.4 · **Author:** platform lead + Claude · **Status:** In review (human WCAG audit, connector links)

**What:** A small React, Vite and TypeScript app in `web/`, served by the cell from the release's built assets with a self-only CSP. It asks, shows cited claims with chips and uncited claims with an "Uncited" label, and shows notices, refused counts and errors with their trace. The manual changed first: sb-ask(3) gained "The web UI", which covers what each element shows, the CSP, and the LAW-14 rendering rule.

**Why this way:** The UI decides "cited" defensively. A claim with `uncited` set, or with no citations, shows as uncited even if the server sent something inconsistent, so the server's label always wins and nothing renders as plain text. A missing `url` falls back to the trace, so every chip leads somewhere honest. Accessibility is tested with axe-core in jsdom. That can't measure colour contrast, so the palette was chosen for 4.5:1 and a human audit stays on the ticket. Five runtime-free dev tools and React itself, all pinned exactly with a lockfile, keep the supply chain small (ADR-0005).

**Learned:** My first LAW-14 rendering property passed against a mutation that ignored `uncited`: it only checked "chips xor label", and the mutated UI still showed chips. The property now also says the server's uncited label must show. Same lesson as S3.1: every property needs a mutation to prove it can fail. Go's mux cleans `..` paths with a redirect before any handler runs, so the static handler never sees one.

**Laws:** LAW-14 extended to the UI. LAW-04 and LAW-05 are still the only red ones.

**Next:** SB-308, the Slack app as an entry point to `POST /v1/ask`.

## 2026-11-19 S3.3 The ask agent and `POST /v1/ask` (SB-306)

**Wave/slice:** W3 / S3.3 · **Author:** platform lead + Claude · **Status:** In review (audit-chain record waits for SB-401)

**What:** `internal/core/ask` runs one question end to end: validate, retrieve and check live, compose through the gateway, keep a trace. `internal/api` serves it as `POST /v1/ask` and `GET /v1/traces/{id}`. The manual changed first: sb-ask(3) gained "How the ask agent runs, exactly", the conversation rule, the trace reasons, the `conversation` field and `EINVAL`.

**Why this way:** When no chunk is allowed, the model isn't called at all. That's cheaper, and it means there's nothing for the model to make up. A conversation sends only the user's earlier questions, never earlier answers, so a turn can't carry content the user has lost access to since; every turn is checked live again. A missing kill-switch reader counts as killed, and a trace that can't be stored means no answer: missing infrastructure fails closed. The request and response types are checked against the man page's tables and example by a test, rather than generated; the man page stays the source and CI catches drift.

**Learned:** The laws' ask system now goes through the real agent, so LAW-01 to LAW-03 test the whole path, model included. The stand-in model cites everything it sees and tries a label that was never there. The end-to-end LAW-14 test is an extension of a law that was already Tested, so it didn't go through the red ratchet; its mutation (citing refused candidates) is caught.

**Laws:** LAW-01 and LAW-14 extended end to end. LAW-04 and LAW-05 are still the only red ones.

**Next:** SB-307, the web UI (React, Vite, TypeScript), served by the cell.

## 2026-11-19 S3.2 Answer composer and model gateway (SB-304, SB-305)

**Wave/slice:** W3 / S3.2 · **Author:** platform lead + Claude · **Status:** Done

**What:** The composer (`internal/core/answer`) turns the model's JSON claims into cited claims, deciding every citation in code. The gateway (`internal/core/gateway`) sits in front of every model adapter: it refuses secrets, pins the release's model, retries rate limits within the deadline and meters tokens. There are four standard-library adapters (OpenAI-compatible, Anthropic, Groq, OpenRouter) with one contract test against a recorded fake, and an embedded prompt set with SHA-256s. The manual changed first: sb-citations(7) "How the composer decides" and sb-agents(7) "MODEL GATEWAY".

**Why this way:** The model only ever names labels (C1, C2, ...), never citations, so it can't cite anything outside the allowed context, even by accident. The secret check reuses the log redactor's registry: every resolved secret is already there, so nothing new has to be kept in sync. Groq and OpenRouter are thin wrappers over the OpenAI-compatible client; Anthropic gets its own body shape.

**Learned:** The secret scan has to check the joined request too, since a secret split between the system prompt and a message is one string by the time it reaches the provider. A shell heredoc mangled a backslash in a Go string literal, so Go source with escapes now goes through the Write or Edit tool.

**Laws:** LAW-14 moves from Stated to Tested. LAW-07 now also covers prompts. Only LAW-04 and LAW-05 (the tool bus, W4) stay red.

**Next:** S3.3, the ask agent and `POST /v1/ask` (SB-306).

## 2026-11-19 S3.1 Retrieval and live authorisation (SB-301 to SB-303)

**Wave/slice:** W3 / S3.1 · **Author:** platform lead + Claude · **Status:** SB-301 and SB-302 Done, SB-303 In review

**What:** Hybrid retrieval over a tenant-bound pgvector index, with the prefilter inside the query, followed by one batched live ACL call per connector and a policy `read` check, then a 12-chunk budget. The laws harness's `NewSystem` now runs this real pipeline. The manual changed first: sb-permissions(7) gained "Retrieval and the live check, exactly", and sb-ask(3) links to it.

**Why this way:** Three independent barriers: the index can only serve its own tenant, the prefilter narrows inside the SQL, and the live check plus kernel decide. The laws are written against the outcome, so each barrier can fail without a leak. The pgvector adapter ships the vector as text cast to `vector`, which avoids a pgvector Go module.

**Learned:** My first "real pipeline" mutation (a cell indexing other tenants' documents) wasn't caught, and for a good reason: the prefilter alone stopped it. To prove LAW-02 itself would see a leak, the mutation also has to break the prefilter. That's the honest version of "defence in depth": each layer is shown to matter by removing the others. A sibling-adapter import in a test tripped the dependency rule, so the pgvector test now has its own tiny embedder.

**Laws:** LAW-01, LAW-02 and LAW-03 move from Tested (red) to Tested. Only the tool-call tests (LAW-04, LAW-05) are still red, until W4.

**Next:** S3.2, the answer composer (SB-304) and the model gateway (SB-305, ADR-0019).

## 2026-11-19 SB-211 Linear connector (W2 built)

**Wave/slice:** W2 / S2.4 · **Author:** platform lead + Claude · **Status:** In review

**What:** A read-only Linear connector over GraphQL, with team-based ACL and polled changes. The manual changed first: sb-connector(3) gained a `tickets: linear` section. This completes Wave 2's build. Every W2 ticket is Done or In review, and each one's Progress says what a human, a staging cell or a later ticket must add.

**Why this way:** Linear's API can filter by `updatedAt`, so polling gives a complete change feed without a public endpoint. Polling team and membership changes turns permission changes into permission-only changes for that team's issues, which is the same shape the ingest pipeline already handles for Drive and Slack.

**Learned:** The contract kit caught a real bug on the first run: a poll timestamp moved during pagination. My first fix, a one-second overlap, failed the kit's "nothing after the latest cursor" case. That was right, because overlap pushes the duplicate problem onto every consumer. An exact tie-break on (timestamp, ids seen) is the correct cursor.

**Laws:** connector-level LAW-03 (with LAW-01's truth) now runs on four connectors: mock, Drive, Slack and Linear.

**Next:** W3 at SB-301 (tenant-scoped hybrid retrieval with the ACL prefilter, on pgvector).

## 2026-11-19 SB-204 Slack connector

**Wave/slice:** W2 / S2.2 · **Author:** platform lead + Claude · **Status:** In review

**What:** A read-only Slack connector with live channel-membership ACL, an Events API receiver with signature checks, and an inbox-backed change feed. The manual changed first: sb-connector(3) gained a `chat: slack` section, and sb-tenant(5) gained the model secret reference for ADR-0019.

**Why this way:** One document per message makes edits and deletes exact: an edit raises that message's version, and a delete removes it. Per-thread documents would make a deleted reply look like an *older* version. Events are verified before they're stored, so the inbox only ever holds Slack's word.

**Learned:** Law worlds can be wrong too. The Slack world's fault injection didn't match what "the source can't answer" means for a batch spanning channels, and the law caught it. Each time, the right fix was to the world, not the connector, and the devlog says so.

**Laws:** connector-level LAW-03 (with LAW-01's truth check) also runs on Slack.

**Next:** SB-211, the Linear connector.

## 2026-11-19 SB-210 Rebuild drill

**Wave/slice:** W2 / S2.3 · **Author:** platform lead + Claude · **Status:** In review

**What:** `ingest.Drill`, a weekly schedule constant, and a law test running the drill. The manual changed first: sb-ingest(7) Rebuild describes the drill.

**Why this way:** The drill compares the set LAW-13 talks about, not row counts, and reports differences by id so a failure can be investigated without exposing content.

**Learned:** I had been gating commits on `tail`'s exit code, not on `make check`'s. The command now exits with make's own status.

**Laws:** LAW-13 stays Tested, with the production drill now under test.

**Next:** SB-204 (Slack) and SB-211 (Linear), the last W2 tickets.

## 2026-11-19 SB-209 Deletes and permission changes reach the prefilter

**Wave/slice:** W2 / S2.3 · **Author:** platform lead + Claude · **Status:** In review

**What:** A LAW-01 law test for the ingest gap, plus manual wording. The behaviour itself came from SB-206.

**Why this way:** A separate tombstone table would be a second source of truth about deletions. The source's change feed already is one, and replaying it during a rebuild reproduces the deletes.

**Learned:** The law went green on its first non-pending run, because SB-206 had already handled permission-only changes. The self-test proves the law would notice if that handling went away.

**Laws:** LAW-01 stays Tested (red); the ingest-gap level is now green.

**Next:** SB-210, the rebuild drill.

## 2026-11-19 SB-208 Durable cursor, scheduler and freshness

**Wave/slice:** W2 / S2.3 · **Author:** platform lead + Claude · **Status:** In review

**What:** A cursor store port and file adapter, a syncer that advances the cursor only after a successful sync, a 60-second scheduler with jitter, and freshness metrics through a small Prometheus text registry in `internal/obs`. The manual changed first: sb-ingest(7) gained a Sync section.

**Why this way:** "Save the cursor last" plus idempotent ingest means a crash anywhere repeats work rather than losing it, with no transaction across the source, the index and the cursor. Freshness is the age of the last success, not a queue depth, because it is the one number that says whether the 15-minute promise holds.

**Learned:** The metrics registry is small enough to own (about 100 lines), which keeps OpenTelemetry out until there's a reason for it.

**Laws:** none.

**Next:** SB-209 (deletes and ACL changes within the freshness SLA) and SB-210 (rebuild drill), then the Slack and Linear connectors.

## 2026-11-19 SB-207 Sanitiser and injection flags

**Wave/slice:** W2 / S2.3 · **Author:** platform lead + Claude · **Status:** In review

**What:** Ingest now strips hidden text and flags text addressed to an AI, per chunk, and `Wrap` puts chunks in data blocks they can't close. The manual changed first: sb-ingest(7) gained the rule table.

**Why this way:** The sanitiser doesn't try to make hostile text safe. It records that text is suspicious and keeps it out of the instruction channel. Flagged chunks are still indexed and citable, because hiding them would make the assistant silently wrong about what a document says. Invisible characters are matched by Unicode category, not a hand list, so new zero-width tricks are covered.

**Learned:** While writing the law, the Write tool turned my `\u200b` escapes into real zero-width characters inside the Go source, which is exactly the attack this ticket defends against. I caught it by reading the file back. The repo now has a regression test that fails on any invisible or control character in source (mutation-checked with a planted file), and code spells such characters as code points. The law itself needed one correction: zero-width obfuscation reveals text when stripped, so the right check is "flagged", not "removed".

**Laws:** LAW-09 moves from Stated to Tested (ingest level). The agent-level check waits for SB-405/SB-501.

**Next:** SB-208 (durable cursor and freshness), then the Slack and Linear connectors.

## 2026-11-19 SB-203 Google Drive connector

**Wave/slice:** W2 / S2.2 · **Author:** platform lead + Claude · **Status:** In review

**What:** The user chose Google Drive as the first docs vendor. The connector is standard library only (OAuth JWT bearer and the Drive v3 REST calls), read-only, and passes the contract kit against `drivetest`, a recorded fake. The manual changed first: sb-connector(3) gained a CONNECTORS section, and 03-connectors names the vendor.

**Why this way:** Drive has no "can user X see file Y" endpoint, so the live check asks Drive *as* the user, through domain-wide delegation. A 404 is ambiguous (no access, or gone), so it's a Deny only when the service account still sees the file; otherwise it's Unknown. A Google SDK would have meant several new modules under ADR-0017. Two endpoints and a JWT are small enough to own.

**Learned:** The kit wants permission-only changes flagged, but Drive's change feed doesn't say what changed. Comparing the change time with the file's `modifiedTime` gives a good enough signal, and a wrong guess only costs a re-ingest. Building the fake surfaced a race of my own: a shared fault queue between user and service-account calls. Faults now have separate queues. These law tests had no red phase, because I wrote the connector first; that's noted on the ticket.

**Laws:** connector-level LAW-03 (with LAW-01's truth check) and LAW-13 now also run on Drive. No status change.

**Next:** a real Workspace sandbox run when the user can provide a service account. Then SB-204 (chat, vendor needed), SB-207 or SB-208.

## 2026-11-19 SB-206 Ingest pipeline

**Wave/slice:** W2 / S2.3 · **Author:** platform lead + Claude · **Status:** Done

**What:** Source changes now become provenance-tagged, embedded chunks in an index. The pure parts (normalise, chunk, cite, chunk id) are in `internal/core/ingest`, and Fetch, Embed and the index write go through ports. The manual changed first: sb-ingest(7) gained a Chunks section.

**Why this way:** LAW-13 needs the index to be a function of the sources. So everything that ends up in the index is derived: chunk boundaries from the text, ids from tenant, cite and span, and the prefilter sorted. There's no clock or randomness in a chunk. A rebuild is simply "drop, then replay the change feed from the start", which uses the same code path as normal sync. The embedder follows the cloud decision's pattern: a port with a deterministic local adapter until the provider is chosen.

**Learned:** Sync must collapse multiple changes per document (the last one wins) and carry a "permission changed" flag through, so a permission-only change re-ingests the same version. Otherwise the version check would skip it, and the prefilter would go stale until the next content edit.

**Laws:** LAW-13 moves from Stated to Tested.

**Next:** SB-207 (sanitiser and injection flags), then SB-208 (durable cursor and freshness).

## 2026-11-19 SB-205 Secrets by reference, and a logger that can't leak them

**Wave/slice:** W2 / S2.1 · **Author:** platform lead + Claude · **Status:** Done

**What:** Validated vault references, a resolver that registers every value with the log redactor before returning it, a local vault, and a slog-based logger that scrubs those values everywhere. The manual changed first: sb-tenant(5) gained a Secrets section.

**Why this way:** The usual secret leak isn't a secret field; it's a URL, header or error message that happens to contain the secret. So redaction works by value (what was resolved), on the rendered text of every attribute, as well as by key name. The resolver refuses to hand out a secret if there's no redactor to tell, which makes forgetting to wire redaction an error instead of a leak.

**Learned:** The law test needs a "you must have logged something" clause. Otherwise a logger that drops every line passes trivially, and the self-test proves it.

**Laws:** LAW-07 moves from Stated to Tested.

**Next:** SB-203 once the user picks the docs vendor; otherwise the next W2 tickets that don't depend on it.

## 2026-11-19 SB-202 Scriptable mock connector

**Wave/slice:** W2 / S2.1 · **Author:** platform lead + Claude · **Status:** Done

**What:** The SB-201 mock gained a fault queue and forced verdicts. A fault-matrix test shows each behaviour on demand, and the connector-level LAW-03 test now mixes a scripted Unknown into otherwise truthful batches. The manual changed first: sb-connector(3) gained a MOCK CONNECTOR section.

**Why this way:** Laws and evals need to name the bad thing that happens ("the second call times out"), not hope a random fault lands. A queue is deterministic and reproducible with `SB_SEED`.

**Learned:** A no-receipt target is ambiguous in real life (did it apply?). The mock models the safe reading: no id, no effect, and an error. The real "applied but unconfirmed" case belongs to the tool bus (SB-408), which must report `DidNotRun` and reconcile.

**Laws:** no status change.

**Next:** SB-205 (credentials via vault references), then SB-203 once the docs vendor is chosen.

## 2026-11-19 SB-201 Connector contract, kit and mock

**Wave/slice:** W2 / S2.1 · **Author:** platform lead + Claude · **Status:** Done

**What:** The connector contract gained a version and negotiation. The contract kit turns 03-connectors' case table into `connectorkit.Run`. A controllable mock source and connector passes it in read-only and writable modes. The manual changed first: sb-connector(3) gained version negotiation and how to run the kit, and 03-connectors gained two kit cases.

**Why this way:** A kit is only worth something if a connector can't pass it while being wrong. So the kit drives faults through the sandbox, not through flags on the connector. It checks idempotency by counting effects at the target, not by trusting the receipt. Version negotiation refuses rather than degrades, because a degraded connector is an unknown permission model.

**Learned:** The ratchet earned its keep again. My first LAW-05 connector test swallowed the "not implemented" error and passed against nothing, and the ratchet reported it as green before any code existed. Law tests must surface `ErrPending`, not treat it as an ordinary error.

**Laws:** LAW-03 and LAW-05 stay Tested (red), with the connector level now green.

**Next:** SB-202, the fault-injecting mock for laws and evals, then SB-203, which needs the user to pick the docs vendor.

## 2026-11-19 First push and first CI run

**Wave/slice:** W0/W1 · **Author:** platform lead + Claude · **Status:** done

**What:** The user created the git repository and its remote. The first commit went to `main`. CI's first run failed in `check` and `proofs` with "permission denied", because the scripts `make` runs directly had been committed from Windows without the executable bit. `c0a12be` set the bit, and the second run was green in every job. A `.gitattributes` keeps LF line endings, so WSL never sees CRLF scripts.

**Learned:** A repository created on Windows needs its executable bits set in git explicitly (`git update-index --chmod=+x`). Local WSL runs on `/mnt/c` can't show the problem, because drvfs reports every file as executable.

## 2026-11-19 SB-114 Factory seams (design note) and W1 status

**Wave/slice:** W1 / S1.2 · **Author:** platform lead + Claude · **Status:** In review

**What:** The last W1 ticket. It adds the custody constants (one file), a custody lint, read-only `sb factory status`, and the first LAW-18 law test. The manual changed first: sb-factory(7) now covers status output and the lint.

**Design note, every hand-off seam and its test:**

| Seam | Where | Test |
|---|---|---|
| Custody record (`custody`, `placement`, `key_custodian`) | registry (`cells` table, `localfs`), `ports.CellRecord` | `registrytest.Contract` (Postgres and local), `TestTenantApply` smoke |
| Custody values in one place | `internal/core/cell/custody.go` | `tools/check/custody.py` with self-test |
| No hard-coded account | `Placement` port; local adapter stamps into `factory-account` | `celltest.Contract` (two mock accounts), custody lint |
| Envelope key hierarchy | `Keys` port, `cell.NewDataKey` | `celltest.Contract` envelope test, `TestLAW18_RootKeyAlwaysHasACustodian` |
| Custody isn't a manifest field | `manifest` forbidden names | golden `invalid.expected`, `TestEveryErrorMessage` |
| Cell can be re-pointed | `cell.LoadConfig` (`SB_CONTROL_PLANE`, `SB_IDENTITY_CERT`) | `TestConfig` |
| Read-only view | `sb factory status` | smoke |

**Why this way:** Hand-off (W7) only works if no code in W1 to W6 assumes the factory holds a cell. With each assumption behind a seam that has a test, W7 becomes a matter of new adapters and a workflow, not a refactor.

**Learned:** This law test had no red phase, because the code it tests landed in earlier tickets. The self-test's mutations stand in for it. That's worth stating rather than hiding.

**Laws:** LAW-18 moves from Modelled to Tested.

**W1 status:** all 13 tickets are In review, `make check`, `make integration` and `make proofs` are green in WSL, and laws stand at 32 tests with 5 expected red (retrieval and tool-call levels, W3 and W4). What's left for every W1 ticket: the user's review and the first CI run (the folder isn't a git repository yet). Real-cloud items (30-minute cell, staging timing, IaC module) wait for a cloud adapter (ADR-0018).

## 2026-11-19 SB-112 Policy analysis in CI

**Wave/slice:** W1 / S1.4 · **Author:** platform lead + Claude · **Status:** In review

**What:** One script, `tools/check/policy.py`, runs schema validation, the tenant-condition rule and the tenant-policy narrowing rule on every `make check`. It also produces a permission diff for pull requests. The manual changed first: sb-policy(5) now says how CI enforces its rules.

**Why this way:** A permission diff computed by evaluating a fixed matrix shows what reviewers care about ("IT leads gained approve"), not a text diff of Cedar. The static tenant check complements SB-110's dynamic law test: the static check catches the rule being broken in the source, and the dynamic test catches it being broken in effect.

**Learned:** After BUG-0005, every new checker in this repo has a self-test that runs before it scans. That cost three lines here and would have caught that bug immediately.

**Laws:** LAW-02 is guarded statically and dynamically at the policy level. Status is unchanged (Tested, red, until SB-301).

**Next:** SB-114, factory seams (custody record, placement, key hierarchy), the last W1 ticket.

## 2026-11-19 SB-111 Policy decisions are audited

**Wave/slice:** W1 / S1.4 · **Author:** platform lead + Claude · **Status:** In review

**What:** The kernel became a value, `policy.Kernel` (engine plus recorder), and the free `Authorise` function is gone. Every decision that must be audited is written before it is returned, and a failed write turns it into `Unknown`. Read allows are sampled. The manual changed first: sb-audit(7) gained "Policy decisions".

**Why this way:** As long as the free function existed, code could ask without being audited. Removing it makes "every decision is audited" true by construction rather than by review. Treating a failed audit write as `Unknown` applies LAW-04's audit-before-act at the decision itself: if we can't prove we decided, we didn't. Only read allows are sampled, because they're the high-volume, low-risk path, and refusals always matter.

**Learned:** The existing `LAW-04 SB-405` ratchet line was law-wide, so a green decision-level test would have been reported as "LAW-04 is green". The per-test lines from SB-103 made the change one line.

**Laws:** LAW-04 stays Tested (red), with the decision level now green. The tool-call level stays red until SB-405.

**Next:** SB-112, policy analysis in CI: schema validation, the tenant-condition check, and permission diffs.

## 2026-11-19 SB-110 Cedar schema and base policies

**Wave/slice:** W1 / S1.4 · **Author:** platform lead + Claude · **Status:** In review

**What:** The schema now covers documents and gives every entity a tenant. Base policies scope every permit. A new law test evaluates the real platform and example tenant policies with the kernel's tenant guard turned off. The manual changed first: the sb-policy(5) entity table and actions.

**Why this way:** The kernel guard (SB-109) and the policies are two independent layers. Testing the policies with the guard off shows each layer holds alone, so removing either one can't silently open a cross-tenant path.

**Learned:** The test was red on its first run for a real reason. The shipped `use_agent ask` permit had no tenant condition (BUG-0006), and the rule against that existed only in prose. That's the second guardrail this session that existed only on paper (after BUG-0005). The pattern holds: write the test that would fail, run it, and only then believe the rule.

**Laws:** LAW-02 stays Tested (red), with the policy level now green and a self-test for a tenant-less permit. Only retrieval (SB-301) is still red.

**Next:** SB-111, policy decisions are audited.

## 2026-11-19 SB-109 Policy kernel: three answers, Unknown refuses

**Wave/slice:** W1 / S1.4 · **Author:** platform lead + Claude · **Status:** In review

**What:** `policy.Authorise` is the single entry point and `policy.Require` the caller helper. The Cedar adapter runs on cedar-go v1.8.0, now vendored. The kernel owns everything that can go wrong around the engine: deadline, panic, error, invalid value, missing scope and another tenant's resource. Each of these is `Unknown` or `Deny`, never `Allow`. A new lint, `decisions.py`, plus `exhaustive`, stops callers from writing "not denied" or hiding Unknown behind `default:`. The manual changed first: sb-permissions(7) gained "The policy kernel".

**Why this way:** Cedar itself is default-deny, but LAW-03 is about everything around Cedar: timeouts, crashes and erroring policies, which Cedar skips. Putting those rules in the kernel, not the adapter, means a future engine can't weaken them. The principal comes only from the Scope, so a caller can't authorise as someone else.

**Learned:** cedar-go reports a policy that errors during evaluation in `Diagnostic.Errors` and otherwise treats it as not applying. That can turn a broken `forbid` into an Allow. The adapter maps any evaluation error to `Unknown`, and a test pins it. Mindful of BUG-0005, the new lint runs a self-test before it scans, so it can't silently stop matching. Vendoring cedar-go pulled in 6 MB of its test-corpus archives, because `go mod vendor` copies whole package directories. That's accepted for now and noted for the SBOM work (SB-504).

**Laws:** LAW-03 stays Tested (red), with the policy level now green. `TestLAW03_PolicyUnknownRefuses` allows only when a clean engine said Allow for an own-tenant resource with a valid scope. A caller using `!= Deny` is caught.

**Next:** SB-110, the Cedar schema and base policies validated in CI with cedar-go.

## 2026-11-19 SB-108 Tenant scope as a type

**Wave/slice:** W1 / S1.3 · **Author:** platform lead + Claude · **Status:** In review

**What:** `tenancy.Scope` can only be made by the cell's single `Issuer`, which only the auth middleware holds. The middleware issues the scope after authentication and a directory read of groups, and puts it in the request context. The first cell-data port (`People`) takes a scope on every method and refuses the zero scope. The manual changed first: sb-tenancy(7) "Scope in code" gained the rules, and the architecture dependency table now lets ports import `core/tenancy`.

**Why this way:** Go has no "friend" access, so "only auth can make a Scope" is two things: a type with unexported fields and a single constructor on a capability (`Issuer`), plus a CI rule on who may create the capability. The zero scope is invalid rather than "all tenants", so forgetting a scope fails loudly.

**Learned:** The CI rule was useless at first. A heredoc turned its `\b` into a backspace character, so it matched nothing (BUG-0005). It was found only because I mutation-checked the rule by adding a violating file. A guardrail without a test isn't a guardrail, and the rule now has a regression test. I then scanned the repository for control characters: none are left.

**Laws:** LAW-02 stays Tested (red). It now has green tests at the cell, routing, sign-in and scope levels; only retrieval (SB-301) is red. `TestLAW02_ScopeIsTheOnlyWayIn` seeds users of three tenants into one shared store, as the worst case. It requires that only the cell's own tenant's sessions get a scope, that lookups find exactly own-tenant users, that groups are the user's own, and that a scope-less lookup is an error.

**Next:** SB-109, the policy kernel on cedar-go (ADR-0017).

## 2026-11-19 SB-107 SCIM and fast deprovisioning

**Wave/slice:** W1 / S1.3 · **Author:** platform lead + Claude · **Status:** In review

**What:** Each cell has a SCIM 2.0 endpoint, which is the only way users and groups enter its directory. A deactivate, whether by `PATCH`, `PUT` or `DELETE`, revokes the user's sessions before the response. Groups are read at decision time. The manual changed first: sb-identity(7) gained the SCIM table and the order of deactivation.

**Why this way:** There are two independent mechanisms: immediate revocation, and re-validation every 60 seconds. Each alone meets the 5-minute bound, so losing one doesn't break LAW-12. A revocation failure fails the SCIM request, because a `200` that didn't revoke would tell the IdP the leaver is gone when they aren't. `DELETE` keeps the user, inactive, so the audit trail can still name them.

**Learned:** The law test had to check collateral damage as well. A "revoke everything" implementation passes the leaver check, so the test also requires active users to keep working. Real IdPs differ in small ways, for example `op: "Replace"` and `"value": "False"`. The endpoint accepts those, and a test pins them.

**Laws:** LAW-12 moves from Stated to Tested. `TestLAW12_DeprovisionRevokesFast` provisions 2 to 4 users through SCIM, deactivates one, then replays requests over 20 minutes of fake time. Half the runs drop the revocation fan-out. No request from the leaver may succeed at 5 minutes or later, and every other user must keep being served.

**Next:** SB-108, the tenant-scoped request context (`tenancy.Scope`) and the session-level LAW-02 test.

## 2026-11-19 SB-106 OIDC sign-in per tenant

**Wave/slice:** W1 / S1.3 · **Author:** platform lead + Claude · **Status:** In review

**What:** Each cell signs users in with its own tenant's IdP, using the code flow with PKCE, a state and a nonce. It verifies the ID token itself, with RS256 over the issuer's key set, and checks issuer, audience, time, nonce, tenant and subject, all standard library. Sessions follow sb-identity(7): 12 hours with no extension, re-validated every 60 seconds against the directory. A user the directory doesn't know never gets a session. The manual changed first: sb-identity(7) gained the check table and session wording.

**Why this way:** JWT verification is small enough to own, and owning it lets us refuse everything except RS256 with a known `kid`. That closes the classic `alg: none` and algorithm-confusion holes by construction. The tenant claim is checked twice: by the cell's provider and by the edge, so either alone stops a cross-tenant sign-in. Sessions are per cell, so a stolen session id is worthless in another cell.

**Learned:** Refreshing the key set on every unknown `kid` would let anyone with a forged token make the cell hammer the company's IdP. Refreshes are now limited to one a minute, with a test. The cost is up to a minute's delay after a key rotation, and that's now in the manual. httptest TLS servers share one certificate, which made the two-IdP browser fixture simpler than expected.

**Laws:** LAW-02 stays Tested (red). `TestLAW02_TokensStayInTheirTenant` mints tokens with a random issuer (including an unknown IdP), signer, tenant claim and audience, and presents each at a random cell. It requires exactly the well-formed, own-tenant tokens to sign in. A cell that ignores the tenant claim is caught. The test was red (`not implemented`) before the adapter existed.

**Next:** SB-107, SCIM provisioning and fast deprovisioning, with the LAW-12 law test on a fake clock.

## 2026-11-19 SB-105 Tenant registry in Postgres, and the edge router

**Wave/slice:** W1 / S1.2 · **Author:** platform lead + Claude · **Status:** In review

**What:** The registry now holds cell records (endpoint, hosts and custody) next to applied resources and the journal. There are two adapters, Postgres (pgx v5.8.0, the first vendored module, 7.9 MB) and the local directory, and one contract test both must pass. The edge router (`internal/edge`) refuses with coded JSON errors and forwards only to the host's cell. Apply writes the cell record when it creates a cell. Cells read their control plane from required configuration. The user installed Postgres 17 in WSL, so the Postgres adapter is tested against a real database, and CI gets a `postgres` job. The manual changed first: sb-tenancy(7) "Registry and routing".

**Why this way:** Routing is data, so no code names a company. The host decides the cell and the token can only confirm it, never redirect it. That's why a router that "helpfully" routes by the token's tenant is the mutation the self-test must catch. A shared host is impossible by schema (primary key), not by a check that could be skipped. The edge sits outside `internal/core`, because it speaks HTTP and the error registry.

**Learned:** The request-level law found a hole on its first green run: a valid token with an empty tenant claim. The router said `EAUTH` and the law said `ETENANT`. `EAUTH` is right (that token isn't sign-in for any cell), and the manual now says so. A contract test pays off at once: the local adapter needed a `tenants/` directory to count as an empty registry, which is the same "unknown is not absent" rule written down for a second adapter.

**Laws:** LAW-02 stays Tested (red) overall. `TestLAW02_RequestsNeverCrossTenants` drives the real router over HTTP with 2 or 3 cells, extra hosts, unknown hosts, bad and foreign tokens. It requires exactly `EPERM`/`EAUTH`/`ETENANT`, or forwarding to the host's own cell. It went red (`not implemented`) before the router existed.

**Next:** SB-106, OIDC sign-in per tenant, which fills the token-verifier port.

## 2026-11-19 ADR-0017 First third-party modules, allow-listed

**Wave/slice:** W1 / S1.2–S1.4 · **Author:** platform lead + Claude · **Status:** Accepted

**What:** The user approved cedar-go and pgx/v5. ADR-0017 pins cedar-go v1.8.0 and pgx v5.8.0, plus six transitive modules. `tools/deps/allowed.txt` holds every `go.sum` line we accept, and `make deps` now fails on any other `go.sum` line or `go.mod` requirement. Mutation-checked: adding `require github.com/evil/mod v1.0.0` fails `make deps`, and removing it passes again. `go.mod` has no `require` yet; each module is vendored when SB-105 or SB-109 first imports it.

**Why this way:** `go.sum` proves the bytes, not the intent. With an allow-list, a new module, or a version bump that pulls in a new indirect module, is a visible line in review.

**Learned:** The newest cedar-go needs Go 1.26 and the newest pgx needs Go 1.25. `go get` silently upgraded the probe module's toolchain to 1.26 until `GOTOOLCHAIN=local` stopped it. Without that pin, CI would build with a different Go than `tools/versions.env` says.

**Laws:** none directly. This is supply-chain groundwork for LAW-06 (only signed, gated builds run).

**Next:** SB-105 (registry and router), then SB-106 to SB-109.

## 2026-11-19 SB-104 Idempotent, resumable `sb tenant apply`

**Wave/slice:** W1 / S1.2 · **Author:** platform lead + Claude · **Status:** In review

**What:** `internal/core/apply` enacts the SB-102 plan. It stamps a new cell first, through the SB-103 template, then records each change in the registry, with a write-ahead journal entry before and after each step. The new ports are `RegistryWriter` and `Journal`. The local control plane (`file:///DIR`) implements them in `localfs`, and stamps into `DIR/accounts/factory-account`, the v1 placement from sb-factory(7). The manual changed first: sb-tenant(1) "How apply works", and sb(1) describes the local control plane layout.

**Why this way:** Resumability comes from idempotent steps, not from replaying a journal. Stamping finds what exists, and each registry write is the whole state after one change. A rerun therefore just plans what is left. The journal is for people and for audit (SB-401), not for recovery, so a lost journal can't corrupt a cell. The cell is recorded only after every resource exists, so a recorded cell is always a complete one. A no-change apply returns before taking an apply id, so it really does write nothing.

**Learned:** The crash test counts writes and fails each one in turn. It found no bug, but writing it showed that "finding" a resource has to cost nothing: a no-change re-stamp that counted as a write would have broken "touches nothing". The fakes now count only creations.

**Laws:** LAW-16 stays Tested. `TestLAW16_ApplyMakesTheCellMatch` applies two random manifests in turn on the real local control plane. It requires the recorded state to equal the second manifest's resources, and a repeat apply to change nothing. An applier that skips removals is caught.

**Next:** SB-105, the control-plane tenant registry. Ask about ADR-0017 (cedar-go, pgx) before SB-109.

## 2026-11-19 SB-103 Cell template behind Placement and Keys

**Wave/slice:** W1 / S1.2 · **Author:** platform lead + Claude · **Status:** In review

**What:** The user decided that cells are cloud-neutral, with a local adapter now and AWS, GCP and Azure later (ADR-0018). The template is Go in `internal/core/cell`: five resources, each named from the tenant and pinned to the region. `Stamp` applies it through two new small ports. `Placement` finds or creates resources in one account, and `Keys` holds the root key and wraps data keys. `celltest.Contract` is the test every adapter must pass. The `local` adapter keeps one directory per mock account and wraps keys with AES-256-GCM. The manual changed first: sb-tenancy(7) now has a Cell anatomy section.

**Why this way:** "Isolation is a property of the template" only holds if the template is code we can test. IaC text per cloud can't be unit-tested the same way, so the template lives in the core and each cloud's IaC sits behind an adapter. The ports are split by job (interface segregation), so a key store and a resource placer can come from different vendors. The core depends only on the ports (dependency inversion). Adding a cloud means adding an adapter folder that passes the contract (open for extension, closed for modification). Envelope keys now mean hand-off (LAW-18) re-wraps one root key instead of re-encrypting data.

**Learned:** The ratchet keyed expected-red by law, so a green cell-level LAW-02 test would have been reported as "LAW-02 is green" while retrieval is still red. `laws/expected_red.txt` now takes an optional test name. The cell test was listed red, failed with `not implemented`, then went green, and its line was removed. One `make check` run hit a transient missing file in `site/` while building on `/mnt/c`, and the rerun was clean. Keep an eye on it; building `site/` inside the WSL filesystem would avoid it.

**Laws:** LAW-02 stays Tested (red) overall. The new `TestLAW02_CellsShareNothing` stamps 2 or 3 tenants into one or two mock accounts, twice each. It requires unique ids, the right tenant and region on every resource, idempotent re-stamps and all five kinds. Three broken templates are caught by the self-test: one shares the database, one drops the region and one forgets the queue.

**Next:** SB-104, idempotent and resumable `sb tenant apply` on the local adapter. Ask about ADR-0017 before SB-109.

## 2026-11-19 SB-102 `sb tenant apply --plan`

**Wave/slice:** W1 / S1.1 · **Author:** platform lead + Claude · **Status:** In review

**What:** The plan is a pure function in `internal/core/plan`: the manifest becomes a sorted list of resources, which is diffed against the applied resources from a new `ports.Registry`. The human output follows sb-tenant(1) (`+`, `~` with `was`, `-`, then a summary), and `--json` is the same plan as data. Until SB-105 there is a local registry adapter, `internal/adapters/registry/localfs`, chosen with `--endpoint file:///DIR`. The manual changed first: sb-tenant(1) gained the resource table, plan format and failure modes, and its example now shows settings. sb(1) gained `file://` endpoints.

**Why this way:** The plan is the manifest diff made concrete, so it lives in core and has no I/O. SB-104's apply engine can then enact exactly what was shown. `Registry.Applied` returns `found` separately from `err`, and the adapter returns "no cell" only when it positively knows. A missing directory or a bad record is `EUNAVAIL`, never an empty state, because an empty state would plan a second cell. That's the same rule as "unknown is refusal". The region is checked against the applied cell, which is the part of sb-tenant(5) rule 1 that `check` can't do offline.

**Learned:** Go map order would have made the plan unstable. Sorting by kind order and then name in one place, plus a test that plans 50 times and compares bytes, keeps it deterministic. Settings shown as resources make the first plan longer than the old man page example, which listed 4 lines. I updated the example rather than hide the settings, because a dry run that hides changes is worse than a long one.

**Laws:** LAW-16 stays Tested, now with a plan-side law test. For random pairs of manifests, enacting the plan on the first cell yields the second, a matching cell plans nothing, and a new cell plans only additions. Self-tests catch planners that drop removals or changes.

**Next:** SB-103 (cell template). It needs the cloud and IaC choice (Q-04, CLAUDE.md decision 1) before it starts.

## 2026-11-19 SB-101 Manifest parser and `sb tenant check`

**Wave/slice:** W1 / S1.1 · **Author:** platform lead + Claude · **Status:** In review

**What:** `internal/core/manifest` adds a strict YAML-subset parser that keeps the line and column of every node, a checker for every rule in sb-tenant(5), and a JSON Schema built from the same field table and vocabulary. `sb tenant check` prints compiler-format errors and exits 1 with `EMANIFEST`. The manual's `yaml run` manifests and `examples/tenants/*.yaml` now go through the real binary in `make examples`, and `invalid.yaml` must print `invalid.expected` exactly. The manual changed first: sb-tenant(5) SYNTAX and exact rules, sb-tenant(1) output format, sb-tenancy(7) regions.

**Why this way:** The parser is our own, not a YAML library, as decided for W1 (CLAUDE.md decision 3). A general YAML parser accepts anchors, tags, merge keys and type coercion (the "Norway problem"), and most lose positions after decoding. The manifest needs few constructs, and every one of them gets a position, so the parser and checker come to about 1,000 lines of standard-library Go and stay inside ADR-0016. Errors about a field point at its key; errors about a value point at the value. The first syntax error stops the check, because positions after it can't be trusted.

**Learned:** The golden file fixed the rule order. With two errors at the same spot (a workflow missing both its approval and its connector), the order is the emit order, so the sort has to be stable. Reporting "approval names an unknown workflow" when `workflows` is itself malformed doubled the noise, so cross-field rules now apply only when the fields they relate are well-formed. Writing the property test found no checker bug, but the generator had to indent nested block lists to be valid YAML at all.

**Laws:** LAW-16 moves from Stated to Tested (manifest side). `TestLAW16_TheManifestIsTheTruth` renders 300 random valid manifests in mixed flow, block, quote and comment styles. It requires each one to parse to exactly the tenant it describes, and requires one injected forbidden or unknown field to be rejected at its key with nothing applied. Three broken checkers (one ignores unknown fields, one drops `exclude`, one invents defaults) are caught by the self-test. The drift side of LAW-16 waits for SB-510.

**Next:** SB-102 (`sb tenant apply --plan`). Then SB-103 and SB-104, which need the cloud/IaC decision (Q-04) first.

## 2026-11-19 SB-011 to SB-013 First model-checker run

**Wave/slice:** W0 / S0.4 · **Author:** platform lead + Claude · **Status:** see tickets (In review)

**What:** Ran `make proofs` for the first time, locally on Windows (Temurin 25; `tla2tools.jar` from the pinned v1.8.0 release, which reports TLC2 2026.11.15; Alloy 6.2.0; both checksums match), then again in WSL Debian on OpenJDK 21 with the same results. All four models pass. TLC state counts are identical to `crosscheck.py`: Approvals 1,068, Rollout 6,290, Handoff 23. All five `Permissions.als` checks are UNSAT at scope 6, and the sanity `run` is SAT. Added `tools/ci/mutations.py`, run by `make proofs`. It applies the 13 mutations from `proofs/README.md`, and all 13 give the expected counterexample:

| # | Mutation | Counterexample |
|---|---|---|
| 1 | Approvals: no `u # requester` | `NoSelfApproval` |
| 2 | Approvals: no `payload = approvedHash` | `AuditBeforeExec` (through `Tamper`) |
| 3 | Approvals: no `~killed` | `KillWins` |
| 4 | Approvals: no `Log` in `Execute` | `AuditBeforeExec` |
| 5 | Rollout: no promote guard | `OneBehindMax` |
| 6 | Rollout: `Promote` drops `r \in gated` | `OnlyGatedRuns` |
| 7 | Rollout: testing tenant deploys a revoked release | `RevokedOnlyWhileDraining` |
| 8 | Rollout: rollback without draining | `RevokedOnlyWhileDraining` |
| 9 | Handoff: provision without both approvals | `DualApproval` |
| 10 | Handoff: destroy factory wrap before tenant wrap | `KeyNeverOrphaned` |
| 11 | Handoff: cutover keeps factory paths | `NoUnlistedAccess` |
| 12 | Handoff: cutover before tenant wrap | `ServingCanUnwrap` |
| 13 | Permissions: `context = candidates - live.Deny` | `UnknownNeverUsed`, `MissingNeverUsed`, `CitedImpliesLiveAllow` SAT |

**Why this way:** We automated the mutations instead of only recording them once. That way a later model edit that makes a guard redundant fails `make proofs`, instead of quietly weakening the evidence behind *Modelled*.

**Learned:** The first run found two bugs in `proofs.sh` and none in the models. (1) TLC rejects `proofs/tla/X.tla` on Windows ("File name ... does not match the name ... of the top level module"), so the script now runs TLC from the model's directory, with `-metadir` in a temp dir so no `states/` is left behind. (2) Alloy 6.2 `exec` runs only the default command, so the old `grep -c unsat` check could never pass. The script now runs each `check` and `run` by name and needs `UNSAT` or `SAT` for each. `-c '*'` is not an option: the Windows Java launcher expands it as a file glob.

**Laws:** LAW-06, LAW-08, LAW-10, LAW-15, LAW-17 and LAW-18 move from Stated to Modelled. LAW-01 to LAW-05 now have model evidence but stay Tested (red), because the ladder only moves up.

**Next:** Get two reviewers and one green CI `proofs` job, then mark SB-011 to SB-013 Done. W1 starts at SB-101.

## 2026-11-18 W0 Exit gate status

**Wave/slice:** W0 / all · **Author:** platform lead + Claude · **Status:** see ticket

**What:** 10 of 16 W0 tickets Done, 6 In review. `make check` is green on a clean copy. What's left needs people or CI: two-reviewer sign-off of the first man pages, the ADRs and the threat model (SB-004, SB-005, SB-014), and the first CI run of TLC and Alloy (SB-011 to SB-013).

**Why this way:** W0's exit gate asks for reviews and model-checker runs, and neither can be faked. Marking those tickets *In review* with an explicit Progress section keeps the ledger honest.

**Learned:** The order of work matters. Writing the manual change (`ENOTIMPL`) first, then the error generator, then the CLI meant that every later check had a stable vocabulary to test against.

**Laws:** LAW-01 to LAW-05 are now Tested (red). No law is Modelled yet, because none has had a TLC or Alloy run.

**Next:** Push to a repo with CI, watch the `proofs` job, then collect reviews. W1 can start on SB-101 in parallel.

## 2026-11-18 SB-012 Rollout model: non-atomic rollback

**Wave/slice:** W0 / S0.4 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** Fixed BUG-0001 in `Rollout.tla` and the cross-check. Rollback now revokes at once and redeploys cell by cell; cells on a revoked build are *draining*. `NoRevokedRuns` became `RevokedOnlyWhileDraining` and `DrainingOnlyRevoked`. `OneBehindMax` and `StableNotAhead` now apply to serving (non-draining) cells.

**Why this way:** The atomic model was simpler but promised something rollback can't do in 10 minutes across many cells.

**Learned:** The promote guard also protects rollback: a draining stable cell isn't on the current stable, so no promotion can happen until every cell has redeployed. The cross-check grew from 2,974 to 6,290 states. The two new mutations (deploying a revoked build, rolling back without draining) are both caught.

**Laws:** LAW-06 and LAW-08 stay Stated until TLC runs.

## 2026-11-18 SB-011 Proofs wired into CI

**Wave/slice:** W0 / S0.4 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** Pinned `tla2tools.jar` 1.8.0 and `alloy.jar` 6.2.0 by SHA-256 in `tools/versions.env`. `tools/ci/fetch_tools.sh` refuses a jar whose checksum doesn't match. `tools/ci/proofs.sh` passes a model only when the checker positively reports success. SB-013 and SB-705 use the same script.

**Why this way:** Unknown means no (P1), applied to our own tooling: output we don't recognise is a failure.

**Learned:** The jars could be downloaded and hashed here but not run, so the checksums are real and the first run will happen in CI. The Alloy CLI flags still need confirming on that run.

**Laws:** No change until the first run.

## 2026-11-18 SB-007 Manual-diff rule tested

**Wave/slice:** W0 / S0.3 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** `tools/ci/test_manual_diff.sh` builds a throwaway git repo and proves four cases: code-only fails, code plus manual passes, the override label passes, docs-only passes. It runs in `make check` and in CI before the real rule.

**Why this way:** A one-off test PR proves the rule once. A scripted test proves it on every build.

**Learned:** POSIX `sh` has no `$RANDOM`. The script now uses `date +%s%N`.

**Laws:** No change.

## 2026-11-18 SB-009 The manual can't lie

**Wave/slice:** W0 / S0.3 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** Added `tools/check/examples.py` (man-page section order, `run` blocks, example YAML) and moved the diagram renderer into the repo as `tools/check/diagrams.py`, with Mermaid pinned in `tools/check/package.json`. Results: 51 diagrams render, all 32 man pages have their sections in order, and 0 links are broken.

**Why this way:** Checks that report *pending by ticket*, such as manifest validation until SB-101, keep the gap visible without failing the build for work that isn't due yet.

**Learned:** Mutation-checked with a planted page whose sections were out of order and whose `run` block didn't parse. Both were caught.

**Laws:** No change.

## 2026-11-18 SB-016 Smoke and regression conventions

**Wave/slice:** W0 / S0.1 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** `tests/smoke/` builds the real `sb` binary and checks the exit-status contract. `tests/regress/` has the BUG-0003 Cedar test and a README mapping every bug to its test.

**Why this way:** Smoke tests are black-box on purpose: they test the binary users run, not the package.

**Learned:** A missing validator in a regression test is a hard failure, not a skip. The BUG-0003 test exits if `cedarpy` isn't installed.

**Laws:** No change.

## 2026-11-18 SB-015 Eval harness scaffold

**Wave/slice:** W0 / S0.4 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** `evals/`: loader, the seven checks from `sb-eval(5)`, worst-of-three runs, a mock target and a smoke suite. `sb eval run` delegates to it.

**Why this way:** Evals become the main gate in W5. Building the harness now means suites can accumulate from W2 on.

**Learned:** Two rules got sharper by being written as code: an unknown expectation fails rather than being ignored, and `faithful_min` fails when there are no cited claims at all, because silence can't demonstrate faithfulness.

**Laws:** LAW-09 and LAW-14 have their first harness; their suites arrive with SB-207 and SB-309.

## 2026-11-18 SB-010 Law harness: five laws red, checks proven

**Wave/slice:** W0 / S0.4 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** Added `laws/`: a seeded `Check` helper, a world generator, LAW-01 to LAW-05 checks, the red ratchet (`expected_red.txt` plus `tools/check/laws.py`), and self-tests against a reference system and six broken systems.

**Why this way:** A red law that fails for the wrong reason proves nothing, and neither does a check that would never fail. The ratchet handles the first; the self-tests handle the second.

**Learned:** The first self-test run failed: the cross-tenant mutation wasn't caught, because the live permission check masked it. Users in another tenant always got Unknown. Real sources can say Allow to outsiders, for example a doc shared with "anyone with the link". The generator now creates public docs, and LAW-02 is proven independent of the live check. Logged as BUG-0004.

**Laws:** LAW-01 to LAW-05: Stated → Tested (red, ratcheted).

## 2026-11-18 SB-008 Dependency rule enforced

**Wave/slice:** W0 / S0.3 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** `depguard` rules in `.golangci.yml`, plus `tools/check/deps.py` reading `go list`. A planted import of `internal/errs` from `internal/core/ports` was caught by both.

**Why this way:** Two independent checks, so a lint config change can't silently drop the rule.

**Learned:** `golangci-lint` v2 needs `version: "2"` and the new `linters.settings` layout.

**Laws:** No change.

## 2026-11-18 SB-006 Error registry generated from the manual

**Wave/slice:** W0 / S0.2 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** `tools/gen/errs.py` turns the registry table in `sb-errors(7)` into `internal/errs/codes.go`, formatted with `gofmt`. `make gen-check` fails on drift. `ENOTIMPL` (501, exit 2) was added to the manual first.

**Why this way:** The manual is the spec (P2), so the code is derived from it, never the reverse.

**Learned:** An uncoded error must exit 1, not 0. That's now a unit test.

**Laws:** No change.

## 2026-11-18 SB-001 Repo skeleton and toolchain

**Wave/slice:** W0 / S0.1 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** `go.mod`, `cmd/sb`, `internal/cli`, `internal/core/ports`, `internal/version`, the Makefile, `tools/versions.env` and CI. Every command in `sb(1)` exists; unbuilt ones exit `ENOTIMPL` naming their ticket. `make check` passes on a clean copy.

**Why this way:** One module rather than a `go.work` workspace (P3). Standard library only, because the Go module proxy refused the request and every dependency is supply-chain surface (ADR-0016).

**Learned:** A CLI that honestly says *not built yet (SB-101)* is better than a missing command. Scripts and agents get a stable exit code (2) and a ticket to look at.

**Laws:** No change.

## 2026-11-18 SB-002, SB-003, SB-004, SB-005, SB-014 Closing the design-baseline tickets

**Wave/slice:** W0 / S0.1 to S0.4 · **Author:** platform lead + Claude · **Status:** see ticket

**What:** The design baseline already delivered the template, the site, the living documents, first drafts of the man pages, the threat model and the ADRs. SB-002 and SB-003 are Done. SB-004, SB-005 and SB-014 are In review, with sign-off tables added for the reviews.

**Why this way:** Reviews are people's work. Leaving them visibly open is better than ticking boxes.

**Learned:** None beyond the sign-off tables.

**Laws:** No change.

## 2026-11-18 SB-000 Factory and hand-off planned, not built

**Wave/slice:** pre-W0 · **Author:** platform lead + Claude · **PR:** design baseline

**What:** Named the control plane, cell template and release train together as the **factory**. Added custody records (`factory`, `tenant`, `detached`), a dual-approved hand-off workflow, ADR-0015, architecture doc 16, `sb-factory(7)`, LAW-17 and LAW-18, and `Handoff.tla`. The hand-off itself is epic E08 in a new wave W7, after v1. v1 gains one ticket (SB-114) and extra acceptance criteria on five others; no v1 wave changes length.

**Why this way:** Building hand-off now would derail v1. Ignoring it would let cheap v1 shortcuts (hard-coded account ids, one factory signing key, push deploys) turn into a migration across every cell. The seams cost almost nothing now.

**Learned:**

- **Pull beats push for more than security.** Cells that pull signed bundles and push attestations work identically in the factory's account and in a company's own account. Choosing pull in v1 is what makes LAW-06 and LAW-08 survive hand-off with no new mechanism.
- **Key order is the whole game.** The cross-check's `destroyfirst` mutation orphans the root key in 16 states. Re-wrap first, then destroy, and LAW-18 forbids *zero* custodians, not two.
- **Binding approvals to a plan hash makes a forgotten "void approvals on re-plan" harmless.** The `keepapprovals` probe found no violation, because an approval of an old hash can never satisfy the new one. Same lesson as LAW-15, one level up.
- **Custody must not be a manifest field.** If it were, anyone with manifest write access could start a hand-off. So `sb tenant check` rejects it.

**Laws:** LAW-17 and LAW-18 Stated. `Handoff.tla` written, cross-checked in Python (23 states, 4 of 4 mutations caught), and not yet run in TLC (SB-705).

**Next:** SB-114 lands in W1 alongside the cell template.

## 2026-11-18 SB-000 Design logic cross-checked in Python

**Wave/slice:** pre-W0 · **Author:** platform lead + Claude · **PR:** design baseline

**What:** Wrote `proofs/crosscheck.py`, a breadth-first search over every reachable state of hand-written Python mirrors of `Approvals.tla` and `Rollout.tla`. All invariants hold: 1,068 states for approvals and 2,974 for rollout. Each of the seven mutations in `proofs/README.md` produces a counterexample.

**Why this way:** TLC couldn't be run in the design environment. A second, independent encoding of the same rules catches design mistakes now, and gives SB-011 and SB-012 an expected result to compare against.

**Learned:**

- Removing the hash re-check from `Execute` trips `AuditBeforeExec` before `ExecutedMatchesApproved`, because the audit row carries the payload hash. So the audit log is itself evidence of LAW-15: an auditor can see that the executed hash differs from the approved one.
- The promote-guard counterexample needs only two promotions and one slow tenant: 208 states. It's a small, likely mistake, which is exactly why it's worth a law.
- This is not a proof of the TLA+ text. The ledger keeps these laws at *Stated* until TLC runs clean in CI.

**Laws:** no status change.

## 2026-11-18 SB-000 Cedar policies validated against the schema

**Wave/slice:** pre-W0 · **Author:** platform lead + Claude · **PR:** design baseline

**What:** Wrote `policy/schema.cedarschema` and `policy/platform/base.cedar`, then validated every Cedar snippet in the repo (platform policy, example tenant policy, `sb-policy(5)`, architecture doc 02) against the schema with the Cedar validator. All pass.

**Why this way:** Policy snippets in docs are code that people copy. If they don't validate, the manual is lying (P2).

**Learned:**

- The first draft named the proposed-action resource `Action`, which collides with Cedar's built-in action type, so `resource is Action` could never match. It's now `ProposedAction`. Logged as [BUG-0003](BUGS_AND_MITIGATIONS.md#bug-0003-policy-examples-used-a-reserved-type-and-an-unguarded-optional-attribute).
- An optional context attribute (`target_project`) must be guarded with `has`; the validator refuses unguarded access. Good: that's the soundness we want from the policy language (ADR-0008).
- Mutation check: three deliberately broken policies (typo'd attribute, wrong resource type, unguarded optional) were all rejected. The validator is a real gate, so SB-112 uses it.

**Laws:** no status change. LAW-02 and LAW-15 now have platform policies backing them.

## 2026-11-18 SB-000 Formal models written (not yet run)

**Wave/slice:** pre-W0 · **Author:** platform lead + Claude · **PR:** design baseline

**What:** Wrote `Approvals.tla`, `Rollout.tla` and `Permissions.als` alongside the man pages, using the same state names as `sb-approvals(7)` and `sb-release(7)`. They were reviewed by hand but **not model-checked**: the tool jars couldn't be run in the design environment, so SB-011 to SB-013 start by running them in CI.

**Why this way:** Writing the models while the man pages were still soft meant the models could change the design cheaply, and twice they did (below).

**Learned:**

- **LAW-08 needs a promote guard.** Writing `OneBehindMax` made it obvious that "promote when the new release is good" lets a slow tenant fall two behind. The fix is a guard: promote only when every stable tenant runs the current stable. That rule is now in `sb-release(1)` and SB-507.
- **Edits must void approvals, and so must tampering.** Modelling an approval as a flag on the action was not enough; it has to bind to the payload hash, and `Execute` has to re-check the hash. The `Tamper` action exists to prove the re-check matters (LAW-15).
- **The rollout model is more optimistic than reality.** It treats rollback as atomic across cells; in reality rollback takes up to 10 minutes. Logged as [BUG-0001](BUGS_AND_MITIGATIONS.md#bug-0001-rollout-model-assumes-atomic-rollback).

**Laws:** 16 Stated. LAW-01, 02, 03, 04, 05, 06, 08, 10 and 15 have models written; none is *Modelled* until a clean run plus a mutation check.

**Next:** SB-011, SB-012, SB-013.

## 2026-11-18 SB-000 Plan generated from one data file

**Wave/slice:** pre-W0 · **Author:** platform lead + Claude · **PR:** design baseline

**What:** 8 epics, 37 user stories, 26 slices and 85 tickets across 7 waves, all generated from a single plan file into `epics/**` and the wave section of the PRD. The generator refuses to run if a ticket depends on a later wave or names a man page that doesn't exist.

**Why this way:** The plan, the PRD tables and the ticket files used to drift in every project we've seen. One source makes drift impossible at generation time; after that, tickets are edited by hand as the source of truth.

**Learned:** The policy engine deserved its own epic (E02). In the preliminary PRD it was one ticket inside tenancy, but it's the kernel that every later epic depends on, and it has its own laws.

**Laws:** no change.

## 2026-11-18 SB-000 Design baseline from the preliminary PRD

**Wave/slice:** pre-W0 · **Author:** platform lead + Claude · **PR:** design baseline

**What:** Turned the preliminary PRD (v0.1) into the full design set: problem statement, principles, domain model, 16 architecture documents, 14 ADRs, threat model, 31 man pages, examples, living documents and the docs site.

**Why this way:** Manual first (ADR-0009). Every later ticket links to a man page that already exists, so implementation follows a spec instead of inventing one.

**Learned:**

- **The preliminary PRD's eight laws weren't enough.** Kill switches, audit tamper-evidence, deprovisioning, index derivation, uncited labelling, approval binding and manifest drift were promises without laws. There are now 16 laws, each with a check.
- **`EPERM` should cover unknown as well as deny.** Telling a user "unknown" invites retrying until it flips; telling them "deny" leaks that a document exists. Operators get the breakdown instead.
- **Kill switches need a fail-safe direction.** If a cell can't reach the kill-switch store, it should behave as if killed for consequential tools. Losing the ability to be stopped is treated as being stopped.
- **Never cache an allow across requests.** Even 60 seconds breaks LAW-01's "at answer time". Denies can be cached; allows can't (ADR-0002).

**Laws:** LAW-01 to LAW-16 Stated in `sb-laws(7)`.

**Next:** W0 starts with SB-001.
