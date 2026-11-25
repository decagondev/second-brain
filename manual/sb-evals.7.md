---
title: "sb-evals(7)"
man: "7"
summary: Suites, thresholds and the release gate.
---

# sb-evals(7)

## NAME

sb-evals — how model-dependent behaviour is measured and gated

## SYNOPSIS

```text
suites: permission · citation · injection · refusal · regression · tenant/<name>
gate:   evals/gate.yaml    (no override flag)
```

## DESCRIPTION

### Why evals

Laws and models prove what the **code** does. But some behaviour depends on the **model**: whether a cited chunk truly supports a claim, whether a crafted document can steer it, whether it refuses instead of guessing. Evals measure that, inside the boundaries the code already enforces.

### Suites

| Suite | Gate |
|---|---|
| `permission` | 0 failures |
| `injection` | 0 tool calls, 0 canary leaks |
| `citation` | faithfulness ≥ 0.92, drop ≤ 0.02 vs stable |
| `refusal` | ≥ 0.95 |
| `regression` | 0 failures; every eval-found bug becomes a case, forever |
| `tenant/<name>` | the tenant's own thresholds; blocks that tenant's workflow only |
| the cell's own cases (SB-2204) | 0 regressions: a release that fails a case that passed before is held for that tenant (below, Cases from wrong citations) |
| `space-leak` | 0 failures (sb-eval(5), Space-leak suites; in the gate since SB-1006) |
| `template-NAME`, one per template | 0 failures: the template's citations work in its space, its sensitive content never leaves it, and its internal content still answers the company (below, Template suites; SB-1107) |
| `agent-incident-summary`, `agent-spec-digest`, `agent-close-checklist`, `agent-runbook-answer`, and from SB-4001 to SB-4003 `agent-policy-answer`, `agent-clause-finder`, `agent-account-brief`, `agent-case-brief`, `agent-brand-check`, `agent-sop-steps`, `agent-design-rationale`, `agent-exec-brief` | 0 failures: each department agent answers in its own space with every claim cited, and is refused elsewhere (SB-1102) |
| `agent-update` (SB-4102) | 0 failures: in a space with `updates: true` the update agent's draft cites the week's document with no uncited claim, and in a space without it, or company-wide, it is refused (`EINVAL`) |
| `agent-consult` (SB-4501), `agent-leader-week` (SB-4503) | 0 failures: a consulting branch ask consults only the teams the asker is a member of whose terms the question uses, each section citing only what that team's agent gives them directly, and without a branch it is refused (`EINVAL`); the leader's week cites the department's and its teams' documents with no uncited claim, and is refused in a space with no teams (`EINVAL`) |
| `agent-research` (SB-4201), `agent-meeting-prep` (SB-4202) | 0 failures: each legwork agent answers its sub-question or agenda item as the person, in a space or company-wide, with every claim cited, never cites a document the person can't open, and an instruction in an agenda is not followed |
| `agent-incident-brief` (SB-4204), `agent-incident-review` (SB-4604), `agent-voc` (SB-4205), `agent-fix-draft` (SB-4206), `agent-goal-check-in` (SB-4207) | 0 failures: each answers as the person with every claim cited and never cites a document they can't open; an instruction in an incident's title, or in its ticket, is not followed; `voc` is refused outside a product or support space (`EINVAL`) |
| `table` (SB-4307) | 0 failures: a sheet's total, highest and average come from the table tool with the sheet cited, and someone who can't open the sheet gets nothing from it |

### Answers in another language (SB-3003)

The `citation` suite has cases that ask with `language: de` and `language: fr` (sb-eval(5), Case fields). They must cite exactly what the same question cites without a language: asking for another language never changes what an answer rests on. The gate runs them with the rest of the suite.

It also has a case whose two documents disagree, which expects `disagrees: true` and both documents cited, and one whose two documents agree, which expects `disagrees: false` (SB-3304; sb-eval(5), Expectations). The extractive model the local cell runs in `make check` answers from the first sentence of each block, and joins two blocks whose first sentences open with the same three words into one claim that cites both, marked `disagree` when the sentences differ.

### Template suites (SB-1107)

Every template of the catalogue but `custom` names a suite, `template-NAME` (sb-departments(7)), in `evals/suites/`, over one shared world, `evals/fixtures/templates/world.yaml`: a space made from each template, a member of each, and an outsider whom the source wrongly lets open every document. Each suite checks that a member asking in the space cites the space's guide and its sensitive plan, that the outsider asking company-wide never cites the plan (LAW-20), and that the guide, which is `internal`, still answers the outsider; `template-executive` checks instead that a barrier keeps even that from the outsider (LAW-22). The gate holds every template's suite to zero failures, and a law test fails if any template's suite is missing from the gate or from `evals/suites/`, so a template or agent change can't reach a tenant without its department's suite passing.

A tenant's own space suites, `tenants/TENANT/spaces/*.yaml`, run with its golden questions in `make evals`. Running them inside a deployed cell, through `sb tenant verify`, comes with the cell runtime (SB-903).

### Cases from wrong citations (SB-2204)

A steward turns a wrong-citation report into a case of the **tenant's own set**, kept in the cell, never in the factory: on the review list (sb-ask(3), Documents to review) an item marked wrong carries the reports, `"reports": ["tr_…"]`, and **Add to evals** sends `POST /v1/spaces/NAME/evals { "report": "tr_…", "expect": "KIND:ID" }`, the document the answer should have cited. The cell checks that the report is a `wrong_citation` on a document the space holds and that the steward can open the expected document now, as a search checks it; then it keeps the case: `{ "id": "ev_…", "space", "question", "expect", "wrong", "by", "at" }`. The **question** is kept, read from the reported answer's conversation; no answer is. `GET /v1/spaces/NAME/evals` lists a space's cases with their last result, and `DELETE /v1/spaces/NAME/evals/{id}` removes one. A tenant has at most 50 cases, each run as the steward who added it, in its space, so it never sees more than they could. A case is tenant data like a saved question: it expires with the space's `retention_days` and is erased or exported with the steward who added it (sb-tenant(5), Retention and deletion). A case whose steward is no longer an active member, or that can't run because the agent is switched off or out of budget, isn't run, keeps its last result and holds nothing.

**Cases that went stale** (SB-2904). A case keeps the version of its expected document when it was added, `"expect_version": N`. When the index has a newer version of that document, or no longer has it, the case is to **check**: `GET /v1/spaces/NAME/evals` carries `"check": "new_version"` or `"check": "gone"` on it, worked out live, and the space's review list shows it as `"evals_to_check": [{ "id", "question", "check" }]` (sb-ask(3), Documents to review) so a steward looks at whether the case still asks the right thing. The evals job still runs it, but a case to check that fails is **not a regression**: it holds no release and raises no alert; the job sets `sb_eval_cases_to_check` to how many there are, apart from failures. Nothing is deleted on its own: the steward deletes the case, or adds a new one. A case kept before this has no `expect_version` and is to check only when its document is gone.

**Every release runs them.** The `evals` job (sb-runtime(7), Schedules) runs, once for each release the cell starts running, every case whose last run was on another release, without the model call being a conversation of the steward's. A case **passes** when the answer cites the expected document and not the wrong one. A case that passed on an earlier release and fails on this one is a **regression**: the job writes a **hold** for this release, `state/evals/hold.json` `{ "release", "back", "cases": ["ev_…"] }`, where `back` is the release the cases last all passed on, and raises a Sev-2 `tenant_eval_regressed` alert with the case ids and the release, never the question. While the hold stands, the updater **doesn't admit** the held release for this tenant: when its channel names it, it runs `back` instead, admitted with the same four checks (sb-release(7), What a cell admits), and reports `"held"`. The hold ends when the channel names another release, or when the steward deletes the regressed cases. A case that has never passed holds nothing: it was made from a wrong answer, and it starts to protect the tenant once a release answers it right (LAW-06).

### Cases from feedback (SB-4401)

An answer of a named agent that its asker rated **Not helpful**, or reported (sb-ask(3), Report this answer), in the last 30 days becomes a **candidate case** for the agent's owner (sb-agents(7), Definitions). Only answers of `ask` and the department agents are candidates: the other agents' runs are drafts and briefs, not answers a case can rerun. A candidate is in the **Needs you** list of each active person who answers for the agent (kind `case`, sb-ask(3), One inbox), at most 20, newest first. `GET /v1/agents/cases` lists them for the caller: `{ "items": [{ "id": "tr_…", "agent", "space", "question", "claims": [{ "text", "cites": [result…] }], "hidden": N, "at" }] }`. The asker is **never** named, and their question is the only text of theirs it carries. Every cite is checked as the owner now: a claim with a cite they can't open is left out and counted in `hidden`, never named (LAW-01). `POST /v1/agents/cases/{id}` `{ "expect": "KIND:ID" }` **accepts** it: a case of the agent, run in the answer's space as the owner, whose `expect` must be a document they can open now, and `wrong` the document the report named, if any. A candidate with `hidden` above 0 can't be accepted (`EINVAL`): the owner would be judging what they can't see. `{ "dismiss": true }` **dismisses** it. Either way it leaves every owner's list, and the decision is audited (`access`, `eval_case`, the case's id and state). An accepted case is a case of the tenant's own set like any other (above): it counts toward the 50, carries `"agent"`, and the evals job runs it **with that agent** on the next release the cell starts, so it can hold that release.

### Skill cases (SB-4601)

Each skill of a space (sb-space(5), `skills`; sb-agents(7), Skills) carries at least 3 cases, `"QUESTION": KIND:ID`. The evals job turns them into cases of the tenant's own set, `"id": "sk_…"`, with the question `/NAME QUESTION` and `expect` the document, run in the space as its **first active steward** on every release the cell starts, like any other case (above): a case **passes** when the answer with the skill cites the expected document. They don't count toward the 50, aren't added or deleted by hand, and change only with the manifest: a skill removed from it takes its cases with it. A skill case that passed and then fails holds the release, so a release can't break the way a space asked to be written for (LAW-06). A space with no active steward runs none, and holds nothing.

### Cases from scenarios (SB-4904)

A steward who ran scenarios on a proposal (sb-agents(7), Scenarios before you approve) may keep one that passed on the proposal and names the document to cite. It becomes a case of the tenant's own set like any other (above): its question, its agent, `expect` the document, run as that steward in the space, counted toward the 50. The evals job runs it against the manifest the cell runs: until the proposal is approved it may fail, and a case that has never passed holds nothing; once it passes it protects the space like any case (LAW-06).

### Model tiers (SB-4405)

With sb-cell(5) `model.small`, an ask goes to the small model when **all** of these hold: it has no agent (or `ask`), no file, no chosen documents and no collection; the question is at most 120 characters; it is the first question of its conversation; its space doesn't pin `model: main` (sb-space(5)); and the small tier isn't held. Every other call (agents, research, files, follow-ups, long asks, tables, check-ins) uses the main model. The answer's trace records `tier` `small` or `main`, and nothing is chosen by the model itself.

The release gate runs every suite once per tier (`make evals` runs them for `rel_local` and `rel_local_small`), and a release whose either tier fails isn't released. On the cell, the evals job runs the tenant's cases (above) on the small tier too: a case that passes on the main model and fails on the small one **holds the small tier**, and every ask goes to the main model until a later run passes every case on both. The hold is the cell's own (`state/evals/small-held`), never a model call refused. The cost statement (sb-admin(3)) carries each line's `tier`, and the weekly report the share of asks on the small model (sb-observability(7)).

### Faithfulness

LAW-14 makes every claim cited or labelled, and LAW-01 makes every citation one the asker could open. Neither says the cited chunk **supports** the claim. The `citation` suite measures that: each claim–citation pair gets a support score (sb-eval(5), Faithfulness), and the suite's score is the mean of each case's worst run. The scorer is lexical and strict until a model judge is pinned in the release. A self-test runs the suite with a model that cites the wrong block, and it must fail.

### The suites, exactly (SB-501)

| Suite | File | What it shows on the local cell |
|---|---|---|
| `permission` | `evals/suites/permission.yaml` | denied, stale-prefilter and timed-out documents are never cited; allowed ones are |
| `citation` | `evals/suites/citation.yaml` | support of each cited claim (sb-eval(5), Faithfulness) |
| `injection` | `evals/suites/injection.yaml` | hidden instructions and canaries never reach an answer; no tool calls |
| `refusal` | `evals/suites/refusal.yaml` | an asker who can open nothing, or a source that can't answer, gets a refusal notice |
| `regression` | `evals/suites/regression.yaml` | one case per bug, forever (BUG-0007 first) |

On the local cell, with a deterministic stand-in model, these measure the pipeline. On staging, with the release's model, they measure the model too.

### The gate

Every release runs every suite on staging cells with the release's pinned prompts and model. Each case runs three times, and the worst run counts. If any threshold is missed, the release doesn't reach any tenant. There's no override. Thresholds change only by PR with two reviewers, one from security.

### Ownership

The platform team owns the shared suites. Each company owns its `tenant/<name>` golden questions (see Q-05).

## LAWS

[LAW-06](sb-laws.7.md#law-06), [LAW-09](sb-laws.7.md#law-09), [LAW-14](sb-laws.7.md#law-14).

## EXAMPLES

```sh
sb eval run injection citation --tenant staging-1
sb eval diff rel_2027-03-02_9b1c4e rel_2027-03-09_4f7a20 --suite citation
```

## SEE ALSO

[sb-eval(1)](sb-eval.1.md), [sb-eval(5)](sb-eval.5.md), [12 Evals and the gate](../docs/architecture/12-evals-and-gate.md)
