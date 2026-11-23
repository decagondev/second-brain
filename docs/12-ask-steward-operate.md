---
title: Easier to Ask, Easier to Steward, Easier to Operate
summary: Concept for W23 to W25 (accepted 2026-11-23): twenty small quality-of-life features on what is already built, for askers, stewards and the people who run the box. No new service, store or Go module.
---

# Easier to ask, easier to steward, easier to operate

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.1 | 2026-11-23 | Platform lead | Proposed, waiting on the user's decisions | [Principles](01-principles.md) · [11-loop-own-fit.md](11-loop-own-fit.md) · [10-run-protect-close.md](10-run-protect-close.md) · [selfhost runbook](runbooks/selfhost.md) · [PRD waves](03-prd.md#waves) |
| 0.2 | 2026-11-23 | Platform lead | Plan accepted: every recommended decision (2026-11-23) | [Principles](01-principles.md) · [11-loop-own-fit.md](11-loop-own-fit.md) · [10-run-protect-close.md](10-run-protect-close.md) · [selfhost runbook](runbooks/selfhost.md) · [PRD waves](03-prd.md#waves) |

## Summary

After W22 the brain answers with cites, tells people when something needs them, keeps documents owned, and fits the site's languages, logs and storage. Reading the code at `c7be7dd`, the gaps that are left are small, and they are the ones people trip over every day:

- **Asking has rough edges.** An answer can't be copied with its cites, printed, or reached from the keyboard alone. Search can't be narrowed by file type or age. A search with no results, or a refused answer, is a dead end with no next step. A person retypes the question they asked an hour ago.
- **Stewarding is all one by one.** A stale document that is still right stays on the review list for good, because nothing marks it as reviewed. Two stewards can answer the same question at once. Stewards type the same replies again and again, can't see how much is waiting without opening each tab, and can't take the review list to a meeting. Someone on leave gets every message all the same.
- **Running the box needs too much reading.** Email notifications (SB-2307) are set in the cell's config file, not in `install.sh`. Nobody can check that a person actually receives a notification, or that the Slack or Mattermost bot has the rights it needs, until something silently doesn't arrive. A typo in the cell's config shows up only when a service fails to start. API tokens expire without warning. `sb selfhost doctor` says what is wrong, but not how to fix it.

Three waves, twenty tickets. Every feature reuses an endpoint, job, adapter or screen that already exists. None of them changes what someone can see.

## Keeping it small

The same five bloat rules as W16 to W22:

1. **No new service the cell must run.** Everything runs in `sb-api`, `sb-jobs` or the existing scripts.
2. **No new store.** New records are `cell_docs` kinds (a review mark, a claim, a reply template, a pause).
3. **No new third-party Go module.**
4. **At most one setting per feature.**
5. **No new page.** Features go in as buttons, sections, filters and badges, not new screens.

One more rule for these waves: **nothing widens what anyone sees.** Every list, count, copy and print shows the same things the screen already shows, checked live as the person.

## The twenty features

| # | Ticket | Wave | Feature | For | In one line |
|---|---|---|---|---|---|
| 1 | SB-2401 | W23 | Copy an answer with its cites | Asker | One button copies the answer as plain text or Markdown, each claim with its cite link, and uncited claims still marked |
| 2 | SB-2402 | W23 | Print an answer | Asker | A print stylesheet: the answer with its cites as numbered footnotes, so the browser's Save as PDF just works |
| 3 | SB-2403 | W23 | Keyboard first | Everyone | `/` focuses the ask box, Ctrl or Cmd+Enter asks, Esc closes a panel, and a skip link; tested with axe and the keyboard alone |
| 4 | SB-2404 | W23 | Search filters | Asker | Filter search by file type and age (`type=`, `since=`) on `GET /v1/search` and the Documents tab |
| 5 | SB-2405 | W23 | Recent questions | Asker | The person's last 20 questions offered in the ask box, kept only in their browser and cleared at sign-out |
| 6 | SB-2406 | W23 | No dead ends | Asker | No results or a refusal suggests a next step: the whole company, fewer filters, the space's glossary, or asking the stewards |
| 7 | SB-2501 | W24 | Reviewed, still right | Steward | Mark a stale document reviewed at its version; it leaves the review list until it changes or comes due again |
| 8 | SB-2502 | W24 | Claim a question | Steward | "Ann is answering this" shown to the other stewards, lapsing after an hour, so two people don't reply at once |
| 9 | SB-2503 | W24 | Reply templates | Steward | Up to 20 saved replies per space that a steward picks from and edits before sending |
| 10 | SB-2504 | W24 | Counts on the tabs | Steward | The Questions and review tabs show how many are waiting, in words for screen readers |
| 11 | SB-2505 | W24 | Review list as CSV | Steward | Download the review list as it is on screen; documents the steward can't open stay counts only |
| 12 | SB-2506 | W24 | Pause notifications | Everyone | "Pause for N days" (up to 30) for leave; new questions still reach the space's other stewards |
| 13 | SB-2507 | W24 | Stewards' line in the digest | Steward | The weekly digest adds the steward's spaces: questions answered, still open, and how old the oldest is; counts only |
| 14 | SB-2601 | W25 | Email in the installer | IT | `SB_EMAIL_SERVER`, `SB_EMAIL_FROM` and `SB_EMAIL_LOGIN_FILE` in `install.env`, written to the cell's config and vault |
| 15 | SB-2602 | W25 | Send me a test | Everyone, IT | A button in notification settings, and `sb selfhost check-notify --to EMAIL`, sends one fixed test message through the cell's real channel |
| 16 | SB-2603 | W25 | Check the chat bot | IT | `sb selfhost check-chat` asks Slack or Mattermost what the bot's token can do, and names any missing scope or permission |
| 17 | SB-2604 | W25 | Check the config first | Operator | `sb cell check-config FILE` runs the services' own config check and confirms each vault secret exists; `update.sh` runs it before any restart |
| 18 | SB-2605 | W25 | Doctor says how to fix it | Operator | Each finding that isn't OK in `sb selfhost doctor` and the console's Box section names the one command or runbook section that fixes it |
| 19 | SB-2606 | W25 | Token expiry warning | Integrator | A notification 7 days before an API token expires, and days left in the Tokens list |
| 20 | SB-2607 | W25 | Notifications in health | Operator | `GET /v1/admin/health` and the weekly report show the notify job's last run and what it sent and skipped per channel; counts only |

## W23 Easier to ask (epic E24, SB-2401 to SB-2406)

The goal: an answer is easy to take away, and no screen leaves the asker without a next step.

### S23.1 Take the answer with you

**SB-2401 Copy an answer with its cites.** A **Copy** button under each answer puts the answer on the clipboard in two forms: plain text and Markdown. Each claim is followed by its cite's title and link. An uncited claim is copied with the same "not cited" mark the screen shows. Hidden sources are copied only as their count, as on screen.
*Laws:* LAW-14 (an uncited claim stays marked in the copy), LAW-01 (only cites the asker could open). *Acceptance:* the copied text matches the screen; a test fails if an uncited claim loses its mark.

**SB-2402 Print an answer.** A print stylesheet hides the tabs and buttons and prints the question, the answer, and its cites as numbered footnotes with their links and ages. Saving as PDF is just the browser's print dialog, so there is no server-side rendering.
*Laws:* LAW-14. *Acceptance:* a print-media snapshot test; the uncited mark prints.

**SB-2403 Keyboard first.** `/` focuses the ask box, Ctrl or Cmd+Enter asks, Esc closes an open panel and returns focus to where it was, and a "Skip to the answer" link comes first. Every shortcut is listed under a "Keyboard" help line. None of them fires while someone is typing in a field.
*Laws:* LAW-14 (accessibility). *Acceptance:* a test drives ask, follow-up, save and the tabs with the keyboard only; axe in each language.

### S23.2 Find it faster

**SB-2404 Search filters.** `GET /v1/search` gains `type=pdf,docx,sheet,note,…` (from the extension on the document's source id or title, which the index already holds; one with none is `other`) and `since=Nd` (the version's first-seen time, as in `GET /v1/changes`). The Documents tab shows them as two drop-downs. Filters only narrow: they run after the live ACL check, so they never show anything more.
*Laws:* LAW-01. *Acceptance:* each filter narrows; a document the person can't open never appears, whatever the filters.

**SB-2405 Recent questions.** The ask box offers the person's last 20 questions as they type. They are kept in that browser only (never sent to or stored by the cell for this), cleared at sign-out and on **Clear**. A shared question (SB-1703) isn't added.
*Laws:* LAW-07 (the cell keeps no more question text than before). *Acceptance:* sign-out clears them; a test confirms no request carries them.

**SB-2406 No dead ends.** A search with no results, or an answer the agent refused for lack of sources, shows next steps: **Ask the whole company** (when asked in a space), **Clear filters**, any of the space's glossary terms (SB-2203) that match words in the question, and **Ask the stewards** (SB-1704). Suggestions come from the person's own question, the filters they set and the space's public glossary, never from documents.
*Laws:* LAW-01 (a suggestion reveals nothing about documents). *Acceptance:* a test with a hidden matching document shows the same suggestions as with no document.

## W24 Easier to steward (epic E25, SB-2501 to SB-2507)

The goal: a steward clears their list in minutes and doesn't repeat work, and nobody is pinged while they are away.

### S24.1 Clear the list

**SB-2501 Reviewed, still right.** On the review list (SB-1709), **Reviewed** marks a document as reviewed at its current version. It leaves the list until a new version arrives or `review_after_days` passes again. A document that is flagged as a wrong citation after the mark comes back at once. The mark is a `cell_docs` record with the steward, the version and the time, and an audit row.
*Laws:* LAW-13 (a rebuild doesn't clear or reset a mark: the version is the document's, not the index's), LAW-11. *Acceptance:* mark, rebuild, still off; a new version, back on.

**SB-2502 Claim a question.** In a space's question queue, **I'll answer this** shows "ann@… is answering" to the other stewards for an hour, or until she replies or lets go. Anyone can still reply; the claim only warns. Someone who is no longer a steward loses their claims at once.
*Laws:* LAW-21 (claims belong to the stewards there are now). *Acceptance:* a claim lapses after an hour (fake clock); a removed steward's claim disappears.

**SB-2503 Reply templates.** Each space keeps up to 20 replies (a name and up to 2,000 characters of text) that its stewards write and edit on the Questions screen. **Use a template** fills the reply box, which stays editable. Templates are the stewards' own text, like a reply, and are never sent to the model.
*Laws:* LAW-19 (a space's templates are only offered in that space). *Acceptance:* a steward of another space never sees them.

### S24.2 See what waits

**SB-2504 Counts on the tabs.** The Questions tab shows how many open questions wait in the spaces the person stewards. The review section shows how many documents wait for review. Both come from the existing endpoints, are read when the tab bar loads and every five minutes, and are said in words ("3 waiting") for screen readers.
*Laws:* LAW-01 (counts from the same checked lists). *Acceptance:* a count matches its list; axe.

**SB-2505 Review list as CSV.** **Download CSV** on the review list writes exactly the rows on screen: title, link, owner, age, cites, flags, copies, owner gone. Documents the steward can't open appear only as counts with their owner, as on screen. The file is made in the browser from the list already loaded, so there is no new endpoint.
*Laws:* LAW-01. *Acceptance:* the CSV's rows equal the list's; a hidden document's title is never in it; cells are quoted against formula injection.

**SB-2506 Pause notifications.** Notification settings gain **Pause for N days** (1 to 30). While paused, nothing is sent to that person, and the weekly digest skips them. New questions still go to the space's other stewards, so no queue goes unheard. The pause ends by itself.
*Laws:* none new. *Acceptance:* a paused steward gets nothing; the other stewards are still told; the pause ends on its day (fake clock).

**SB-2507 Stewards' line in the digest.** A steward's weekly digest (SB-2103) gains one line per space they steward: "Support: 6 questions answered, 2 still open, the oldest 4 days." These are counts only, from the queue as it is when the digest is sent.
*Laws:* LAW-07 (no question text), LAW-21 (only spaces they steward now). *Acceptance:* someone removed as a steward gets no line for that space.

## W25 Easier to operate (epic E26, SB-2601 to SB-2607)

The goal: whoever runs the box sets everything from one file, can check each channel before people depend on it, and is told how to fix what is wrong.

### S25.1 Set it up once

**SB-2601 Email in the installer.** `install.env` gains `SB_EMAIL_SERVER`, `SB_EMAIL_FROM` and an optional `SB_EMAIL_LOGIN_FILE` (`{"username", "password"}`, root-only). `install.sh` writes the first two to sb-cell(5) `email` and puts the login in the vault at `vault://tenants/TENANT/email`. `--doctor` checks that the server answers and offers STARTTLS.
*Laws:* LAW-07 (the login never reaches a log, the support bundle or `compose.env`). *Acceptance:* the install test writes the config; `sb secrets scan` finds no password outside the vault.

**SB-2602 Send me a test.** Notification settings gain **Send me a test**, and `sb selfhost check-notify --to EMAIL` does the same for an admin. Each sends one fixed message ("This is a test from Second Brain") through the cell's real channel, at most once a minute per person, and reports whether the channel accepted it.
*Laws:* LAW-07. *Acceptance:* the fake Slack, Mattermost and SMTP servers each receive exactly the fixed text.

**SB-2603 Check the chat bot.** `sb selfhost check-chat` reads the bot token from the vault and asks the chat what it may do: Slack's `auth.test` and the scopes it returns, Mattermost's `users/me` and the bot's team membership. It names each missing scope or permission (`chat:write`, `im:write`, `users:read.email`, creating direct channels, reading emails) and the screen in the chat's admin settings where it is granted. This is also the run that closes out the real-Mattermost check left open from SB-2005.
*Laws:* LAW-07 (the token never printed). *Acceptance:* fakes with each scope missing; a run against a real Mattermost is logged on the ticket.

**SB-2604 Check the config first.** `sb cell check-config FILE` runs the same check every service runs at start (`wire.Parse`), then confirms every vault reference it names exists with owner-only permissions, without reading or printing the secrets. `update.sh` runs it before restarting anything, and stops with the line at fault.
*Laws:* none new. *Acceptance:* a typo, a missing secret and a world-readable secret each stop the update before a restart.

### S25.2 Know it's working

**SB-2605 Doctor says how to fix it.** Each verdict that isn't OK in `internal/core/box` gains a `fix`: one command (for example `sudo tools/selfhost/install.sh --recovery-kit`) or a runbook section. `sb selfhost doctor` prints it under the finding, and the console's Box section shows it.
*Laws:* none new. *Acceptance:* a test fails if a non-OK verdict has no `fix`, or a `fix` names a runbook section that doesn't exist.

**SB-2606 Token expiry warning.** The notify job tells a token's owner 7 days before it expires ("Your API token ci-reports expires in 7 days"), once, as a new kind `tokens` they can turn off. The Tokens list shows days left.
*Laws:* LAW-07 (the token's name only, never the token), LAW-12 (a deprovisioned owner's tokens are already gone, and they are told nothing). *Acceptance:* told once (fake clock); never for a revoked token.

**SB-2607 Notifications in health.** The notify job keeps its last run's counts: sent, skipped (nobody by that email, or the person paused), and failed, per channel. `GET /v1/admin/health` shows them, with a warning when the last run failed or hasn't run for an hour, and the weekly report adds one line. These are counts only, never who.
*Laws:* LAW-07. *Acceptance:* a failing fake channel turns health to warning; the counts name no one.

## Decisions for the user

Decided by the user on 2026-11-23, with every recommended option:

1. **Scope and order.** *Recommended:* all three waves, all twenty tickets, in the order W23, W24, W25. *Other:* W25 first, so whoever runs the box can check notifications before the asker and steward features land.
2. **Recent questions (SB-2405).** *Recommended:* kept in the person's browser until they sign out. *Other:* this browser tab only, gone when it closes (safest on shared computers, less useful); or leave the feature out.
3. **Claiming a question (SB-2502).** *Recommended:* a soft claim that warns the other stewards for an hour and never blocks a reply. *Other:* leave claiming out.
4. **Pausing notifications (SB-2506).** *Recommended:* pause for up to 30 days, with new questions still reaching the other stewards. *Other:* pause plus daily quiet hours (a second setting, so it breaks the one-setting rule).

## Still open from earlier waves

- **SB-1605** still needs its run on a real Linux VM. The same VM would check W17's `--doctor`, restore and offline upgrade, and the S3 restore drill against a company bucket (SB-2303).
- **SB-903 and SB-908** need a real GCP project; the human reviews of W0 to W7 are unchanged.
- **Mattermost (SB-2005, SB-2105)** has been tested against a fake server only. SB-2603 gives a ready command for the real run.
