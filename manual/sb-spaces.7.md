---
title: "sb-spaces(7)"
man: "7"
summary: Department spaces inside a company's cell, and how they narrow what the source allows (W8 draft; built in W10 to W12).
---

# sb-spaces(7)

## NAME

sb-spaces — department brains as spaces inside one company's cell, which can only narrow access

## SYNOPSIS

```text
cell (one company)  ⊃  spaces  (engineering, people, finance, …)
ask in space S      →  chunks from S's scope that pass every check below
ask company-wide    →  chunks from every space the asker is a member of,
                       minus restricted content from spaces they aren't in
use a chunk         ⇔  member ∧ label allowed ∧ no barrier ∧ source ACL = Allow (live)
```

## DESCRIPTION

This page was the W8 draft that W10 to W12 build against ([ADR-0020](../docs/adr/ADR-0020-department-spaces-inside-the-cell.md), [08 Department brains](../docs/08-department-brains.md)); W10 built spaces in the cell (SB-1001 to SB-1008), and `sb tenant apply` applies them since SB-1008. Until then it refused a manifest that declared any, so a company never believed its People brain was restricted before the whole cell enforced it.

### What a space is

A **space** is a department brain inside a company's cell: an Engineering brain, a People brain, a Finance brain. It has:

| Part | What it is |
|---|---|
| **Scope** | which parts of the company's connectors it covers: folders, channels, projects, repositories, record types |
| **Members** | the IdP groups whose people may ask in it |
| **Stewards** | the IdP groups who may propose changes to it (W12) |
| **Label default** | the label its documents get when the source carries none (below) |
| **Agents** | which agents run in it, from the manifest's `agents` |
| **Retention** | its own `retention_days`, at most the tenant's |
| **Data key** | its own data key, wrapped by the cell's root key, so a space can be crypto-shredded alone (LAW-18 still holds) |

Every space shares the cell's IdP, release, audit chain and index. **The cell is still the tenant boundary** (ADR-0001), so nothing about LAW-02 changes. A department that needs harder separation than a space gives, such as a legally separate subsidiary, is a separate tenant.

A document can be in several spaces (a shared spec in Engineering and Product), and a person can be a member of several spaces. A manifest with no `spaces:` block behaves exactly as a cell does today: one undivided brain, where the source's live ACL is the only rule.

### Spaces only narrow

A space can **hide** what someone could open at the source. It can never **show** what the source denies (LAW-19). The source's live ACL stays last and stays the authority (ADR-0002). The space checks run before it because they are local and cheap, so they save ACL calls without ever replacing one.

### Labels

Every document carries one label from a four-step ladder:

| Label | Meaning | Who may read it, inside its space |
|---|---|---|
| `public` | may leave the company | any member, and any company-wide asker |
| `internal` | anyone in the company | any member, and any company-wide asker |
| `confidential` | the space's people and named readers | members of a space that holds it |
| `restricted` | need-to-know | members of a space that holds it, and never in a company-wide answer for anyone else (LAW-20) |

A label from the source wins where the source has one (for example a Drive label). Otherwise the document gets the most restrictive default among the spaces that hold it. A document with no label and no space default is `internal`. An unreadable or unknown label is `restricted`: unknown never widens.

### Information barriers

A **barrier** separates two groups, so that neither side's people can use content held only by the other side, in either direction (LAW-22). Barriers are for ethical walls: a deal team and everyone else, or two competing client teams. A barrier holds even where membership and labels would allow a read. It is checked on the asker's groups against the groups of every space that holds the document, and against the groups of the document's owner at the source, so a deal team's file shared with everyone still stays on the deal team's side (SB-1203). It holds for drafts, actions and approvals as well as answers (sb-permissions(7), Barriers beyond reads). A steward's note is placed before it is written, by its path (SB-2001): `notes/SPACE/qs_ID.md` belongs to the spaces whose scope holds it and is owned by the steward, so a steward publishes notes on their own side of a barrier and never into a space across it.

### Company-wide asks

Built in SB-1006: retrieval runs the space checks before the live ACL batch (sb-ask(3), How the `ask` agent runs).


An ask with no space is **company-wide**. Its candidates come from every space the asker is a member of, plus `public` and `internal` content from spaces they aren't in. Content labelled `confidential` or `restricted` comes only from the asker's own spaces. Each chunk still passes the barrier check and the source's live ACL.

### The access decision, in order

Every chunk passes all of these before it reaches a model's context, in this order, and the first that doesn't say yes refuses it. Any `Unknown` refuses (LAW-03).

1. **Membership:** for an ask in space S, is the asker a member of S? For a company-wide ask, which spaces hold the chunk, and is the asker a member of any of them?
2. **Label:** is the chunk's label allowed for this asker, as the table above says?
3. **Barrier:** is there a barrier between the asker's groups and the groups of any space that holds the document?
4. **Source ACL:** does the source allow the asker to open the document now, asked live (sb-permissions(7))?

The exact checks, the Cedar entities and the error codes are in [sb-permissions(7), Spaces, labels and barriers](sb-permissions.7.md#spaces-labels-and-barriers-exactly-sb-802) and [sb-policy(5), Spaces](sb-policy.5.md#spaces-labels-and-barriers-sb-802-built-in-sb-1004).

### The space registry and the request context (SB-1002)

The cell keeps its spaces in its own database, one row per space, with the id `spc_<name>` (for example `spc_finance`) and every field of sb-space(5) after the template's defaults are filled in. Only `sb tenant apply` writes the registry: it makes the rows match the manifest, so a space the manifest drops is deleted, and nothing else in the cell can create, change or delete a space (LAW-16).

Every request carries the space it is in, the same way it carries its tenant (sb-tenancy(7), Scope in code). The auth middleware sets it, and nothing else can:

- a request names its space with `?space=NAME` (sb-api(3), Spaces). No `space`, or `space=company`, is a **company-wide** request;
- a space the registry doesn't hold is `ESPACE`, never a company-wide request instead;
- a space the asker isn't a member of is `ESPACE` too, with the same message, so the error never tells a non-member whether a space exists;
- a registry that can't answer is `EUNAVAIL`. Unknown never means company-wide.

Membership is read from the asker's directory groups on every request, so a move takes effect at the next request (LAW-21). A space's members are matched by group name: `group:finance` is the directory group `finance`.

### Stewards

Stewards don't approve their own changes. They propose a space change as a manifest diff, and a tenant operator approves it (LAW-16). Finance spaces add segregation of duties: whoever requests a change can't approve it.

**Delegated changes, exactly (SB-1201).** A steward of a space, a member of one of its `stewards` groups, proposes a new definition of **that space only**, through `POST /v1/spaces/NAME/proposals` (sb-admin(3), Proposals). The proposal is checked when it's made, and refused with `EMANIFEST` if it:

- names another space, or changes anything outside this space (the barriers, the connectors, other spaces);
- changes the space's `template`;
- lowers the space's `label` below what it is now, on the ladder `public` < `internal` < `confidential` < `restricted`;
- leaves the space with no stewards, or fails any rule of `sb tenant check` (sb-space(5));
- names a label that isn't one of the four, or has any value that reads back differently once written as a manifest (a YAML comment or quote inside it), so what the operator approves is what was proposed (SB-1401).

A proposal records the steward, the hash of the manifest it was based on and the hash of the manifest it would make, and writes a `kind: config` audit row (`state` `proposed`, the space in `space`). A tenant operator approves it by sending that exact hash back (LAW-15's hash binding, applied to manifests): a different hash, or a manifest that changed since the proposal was made, is refused, and the steward proposes again. Nobody approves their own proposal, even an operator who is also a steward (`EPERM`). An approval writes a `kind: config` row (`state` `approved`) and returns the manifest to apply; the space changes only when it's applied (LAW-16). Stewards widen nothing alone: every change, adding members included, needs an operator.

**Start here (SB-1706).** A space's `start_here` (sb-space(5)) is the short reading list its stewards keep for new members. `GET /v1/start?space=NAME` returns it to a member, item by item in order: each document is checked as a search checks it, now, as that member (the space, label and barrier checks and the live ACL), and is shown with its title, cite, link, owner and age. A document they can't open, or whose check is Unknown or late, isn't named: it is counted in `hidden`. A folder item lists the documents under it they can open, by location, up to ten items in all. Someone who isn't a member gets `ESPACE`, as for any request in the space; company-wide there is no list. The web UI shows "Start here" above the conversation when a space is chosen. `sb tenant verify` reports an item **no member** of its space can open (sb-tenant(1)), so a broken list is found before a new member finds it.

**Stewards' tools (SB-2501 to SB-2503).** A steward marks a document reviewed at its version, so it leaves the review list until it changes or comes due again; claims a question for an hour so two stewards don't answer it at once; and keeps up to 20 reply templates for the space (sb-ask(3), Questions to the stewards). Claims and templates belong to the stewards there are now (LAW-21) and to that space only (LAW-19).

**Retire from answers (SB-2801).** A steward can take a superseded document out of their own space's answers, with a reason and, optionally, the document that replaces it; it stays at the source and in every other space, and **Bring back** undoes it (sb-ask(3), Documents to review). A retirement only narrows its own space (LAW-19).

**Start here gone stale (SB-2907).** The review list also names the space's `start_here` items that are gone, stale, or that more than a quarter of its members can't open, each with why, so a new member's reading list doesn't rot unseen. An item the steward can't open is only counted (sb-ask(3), Documents to review).

**Unused glossary terms (SB-2805). The never-cited section also names the space's glossary terms no question asked in the space used in 180 days, so a steward can propose removing them in the manifest. Counts from the traces only; nothing is removed by itself.

**Suggest a glossary term (SB-3107).** `POST /v1/spaces/NAME/proposals/term` `{ "term": "COD", "meaning": "cash on delivery", "spellings": ["C.O.D."] }` is a steward proposal (Delegated changes, above) that adds one term to the space's glossary (sb-space(5)), with up to 10 other spellings, each a term with the same meaning. The cell builds the space's new definition from the applied manifest, so the steward sends only the term, and it is checked, recorded and audited exactly as any proposal; a term or spelling the glossary already has is `EINVAL`, and so is one that breaks the glossary's limits. The answer is the proposal's, as for `POST /v1/spaces/NAME/proposals`. Nothing changes until an operator approves it by hash and the manifest is applied (LAW-16); then the term is in the glossary. The steward screen's glossary has **Suggest a term**.

**Freshness (SB-2807).** The steward's screen shows, for the space, the share of this month's cited answers that rested mostly on stale cites, and last month's, so a steward sees whether answers are getting staler. The weekly report has the same share for the whole cell (sb-observability(7)).

**Documents to review (SB-1709).** The steward's screen lists the documents of the space that answers rely on but nobody has changed in `review_after_days` (sb-space(5)), and any a reader marked as a wrong citation, each checked live as the steward (sb-ask(3), Documents to review). One the steward can't open is counted with its owner, so they can ask the owner to review it. The list also holds the space's **notes read wider than the space** (SB-2004): a note's readers are the members when it was published, so when someone leaves the space the note doesn't follow. Each note whose readers include someone who is no longer a member is listed, checked live as the steward, with the number of readers to remove, never who. **Republish to members only** publishes the same note again to the readers it had who are still members (never anyone it didn't have, and never someone who joined since, who can be given a new note), after the same confirmation as any note; once the share is next read the note leaves the list. **Download CSV** (SB-2505) saves exactly the rows on screen, made in the browser from the list already loaded: one row per document the steward can open (title, cite, times cited, age, times marked wrong, owner gone), and documents they can't open only as a count with their owners, never a title; every cell is quoted, and one that starts with `=`, `+`, `-`, `@`, a tab or a return is prefixed with `'` so a spreadsheet never runs it. It also holds the space's **documents whose owner has gone** (SB-2201), marked "owner gone" so a steward finds them a new owner, and lists **copies** of the same text once, with where each copy the steward can open lives (SB-2202).

**Questions and notes (SB-1704, SB-1705).** A space's stewards also answer the questions its members send when nothing answered them (sb-ask(3), Questions to the stewards), from a queue on the steward's screen. A reply is the steward's own text with cites they can open; the asker sees each cite only if they can open it too. A steward can publish a reply as a **note**, a Markdown document in the space's notes folder that the next ask finds and cites (sb-connector(3), `docs: folder`, Notes). A note's readers are the space's members when it is published, or fewer, never more; before publishing the steward is told how many of the reply's cites some of those readers can't open, and confirms that number, because a note is new text the source's ACL doesn't follow. The steward's publish is the note's approval, bound to its hash; no second person approves a note (the user's decision, 2026-11-22). Company-wide questions go to the manifest's `questions_to` group, which works the `company` queue the same way. **No steward left** (SB-3102): a question sent to a space whose steward groups have no active member in the directory when it is sent goes to the `company` queue instead, if the manifest has `questions_to`, carrying `"from_space": "finance"`, and the company-wide answerers see "Finance has no active steward" on it. The asker is told where it went. When the directory can't answer, the question stays in the space's queue, which never widens who reads it on a guess, and health warns (sb-admin(3)).

**In the web UI (SB-1509).** `/?steward=NAME` is the steward's screen for space `NAME`. It takes the space's new definition as JSON in the fields of sb-space(5), sends it as a proposal, and shows the diff, the plan and the hash the operator will approve; a refused proposal shows each reason. Below it are the space's questions (SB-1704), each with a reply box and, once replied, **Publish as note** (SB-1705), then the space's access reviews, one person per row with their `via`, last use and decision. An undecided person has **Still needs it** (certify) and **Remove**, and Remove asks for the reason first. A decision on your own access is refused by the cell, and the screen says so. When the review completes, the screen lists its change requests and the proposal id of a manifest one. The admin console (`/?admin`, sb-admin(3)) shows the open proposals with their diff, and approves one only by sending back the hash it displays, the same binding as an approval of an action; it also lists every review, overdue ones first, with a link to its evidence.

### Access reviews, exactly (SB-1202)

Each quarter the stewards of every `restricted` space re-certify who can use it, for SOX and SOC 2. The `review` job (sb-runtime(7), Schedules) opens one review per restricted space at 05:00 UTC on the 1st of January, April, July and October, for the quarter just begun (`2027-Q1`). A space that already has a review for that quarter gets no second one.

- **Who is listed.** Every person in every directory group named in the space's `members`, once, with the groups that give them access (`via`) and their **last use**: the time of the latest audit row written in the space with them as actor, or `never`. A group the directory can't read stops the review from opening, and the job fails loudly, because a list with someone missing would certify less than it claims.
- **Deciding.** A steward marks each person `certify` or `remove`; a removal needs a reason. Nobody decides on their own access: another steward, or an operator, does (an operator may decide through the same endpoint, so a review whose only steward is its last undecided member still completes). A decision is final, and each is a `kind: access` audit row with the space, whose `seq` the review keeps. Two decisions at once never overwrite each other: each is recorded and stored as one step (SB-1401).
- **Removals ask; they don't act.** A decision changes nothing by itself. When the last person is decided the review is complete, and each removal becomes a change request for the group that gives the access. For a group listed in the manifest's `groups:` (Identity Platform without SCIM, sb-identity(7)), the requests together become one manifest proposal that takes those people out of those groups, which an operator approves by hash like any steward proposal (LAW-16); for a group from the IdP, an **IdP change request** names the person and the group, for the operator to make in the IdP, after which SCIM removes the access (LAW-21). If taking the people out would leave the manifest failing `sb tenant check`, those requests become IdP-style requests for the operator to resolve by hand, never dropped. Removing someone from a group removes everything that group gives them, which is why the review shows `via`.
- **Overdue.** A review still open 14 days after it opened is overdue: `sb_access_review_overdue{space}` is 1 and a Sev-2 `review_overdue` alert goes to the tenant operator (sb-observability(7)). It stays overdue until it is complete.
- **Telling the stewards (SB-3504).** The notify job tells each active steward of the space, through the `questions` kind (sb-ask(3), Notifications), when a review opens ("The finance access review is open: 12 people, due 2027-01-15") and once more when it becomes overdue ("The finance access review is overdue"), with a link to the steward screen; counts and dates only. The steward screen's buttons are **Still needs it** (`certify`) and **Remove** (`remove`, with a reason); **Remove** only ever asks, as above, and the operator sees the IdP change requests on `GET /v1/admin/reviews` (sb-admin(3)). A directory that can't be read stops the review from opening and fails the job, as above, so a review never starts from a guess (LAW-03).
- **Evidence.** `GET /v1/admin/reviews/ID/evidence` (sb-admin(3)) returns the review as one JSON document: the space, quarter, stewards, opening, due and completion times, every person with their `via`, last use, decision, who decided, when, why, and the audit `seq` of the decision, the change requests with their proposal ids, and `sha256`, the SHA-256 of the canonical `review` object. An auditor checks the hash and finds each decision's row in `sb audit export` by its `seq`.

### Legal holds, exactly (SB-1207)

A legal hold stops deletion in the scope of a matter, and only there. A **Legal steward** (a steward of any space whose template is `legal`) places a hold with `POST /v1/holds` (sb-admin(3)), naming the matter and its scope:

- a **space** (`"space": "finance"`): everything the cell keeps in that space;
- **custodians** (`"users": ["usr_…"]`): everything the cell keeps about those people, in every space;
- or both: those people's records in that space.

While a hold is active:

- **Retention skips held records.** The nightly sweep (sb-tenant(5), Retention and deletion) deletes everything else older than `retention_days`, and keeps every record a hold covers, however old. A record whose store doesn't know its space or its user (log files, conversations) is kept if any hold could cover it: unknown never deletes.
- **Subject erasure is refused** (`EHOLD`), and nothing is erased, for a custodian, and for anyone while a hold on a whole space is active (their records there are held): an erasure would destroy what the hold preserves. It can be made again once the hold is released.
- **A held space can't be removed.** `sb tenant apply` of a manifest without a held space fails with `EHOLD` before anything changes, so the space's row stays and its key is never crypto-shredded. A hold on custodians alone covers every space, so no space can be removed while one is active.

Placing and releasing a hold each write a `kind: retention` audit row (subject `hold:ID`, state `placed` or `released`, the matter and scope in the reason) **before** the hold takes or loses effect; if the row can't be written, nothing changes (LAW-11). Releasing needs a Legal steward and a reason. A released hold stops protecting at once, and the next sweep deletes what it had kept.

### The tree (SB-3901 to SB-3903)

Spaces follow the company's shape: the company, its departments, and their teams. A space names its department with `parent` (sb-space(5)); a space with no parent is a department, and a company-wide ask is the top of the tree. There are at most three levels and no cycles, and `sb tenant check` refuses anything else with its line. Reorg preview shows a change of `parent` like any change to the manifest.

**The tree only narrows** (LAW-24). A team is at least as strict as its department: its label is at least the department's, its retention at most the department's, and the department's barriers hold in it as everywhere. **Members are never inherited.** A team's members are its own groups, since IdPs already nest groups, and a person in a team who isn't in the department's groups isn't a member of the department.

**Settings flow down (SB-3902).** A team with no glossary, start-here list, review interval or agents of its own takes its department's. Glossaries merge, the team's own term winning; start-here lists join, the department's first, at most ten, each item checked as the reader as always. The department's stewards are stewards of each of its teams for questions that come up the tree and for roll-ups, not for publishing notes in a team unless the team names them.

**Ask across a branch (SB-3903).** In a department, `?space=NAME&branch=true` (sb-ask(3)) asks in the department and every team under it **that the asker is a member of**. Retrieval and every check run once per space, as the asker in that space, and the chunks each run allowed are joined: nothing one space's checks refused enters through another's unless that run allowed it on its own. A team the asker isn't in adds nothing and isn't named. Cites name the spaces that hold each document, as always. The web UI's switcher offers **Finance and its teams** only when the asker is in at least one of its teams; `GET /v1/spaces` says so with `"teams": true` on the department, and names a space's `parent` only to a member of the parent too.

**Departments work together (SB-4501 to SB-4503).** A branch ask can also **consult** the department agents of the teams the asker is a member of, at most 6, each one level deep and as the asker, and returns a section per team, saying which teams weren't consulted and why; teams the asker isn't in are only counted (sb-ask(3), Ask each department; LAW-27). An answer whose question names a request category of another space the asker is in offers **Hand off**: the asker sends the drafted request, with the cites they choose to share, and its stewards see only the cites they can open, the rest counted. Each Monday, a department's stewards get the leader's week, a draft of the branch with every line cited or, for a team, counted (sb-agents(7), The leader's week).
**Work across departments (SB-4504 to SB-4507).** A request, action item or goal can be marked **waiting on** another space, with a reason and a date: both spaces' homes list it, each item named only to someone who can see it and counted for anyone else, a department's roll-up counts its teams' waits, and one still waiting a working day past its date goes once to the stewards of the two spaces' nearest common parent. A decision can name the one it **replaces**, and a topic shows its **trail**. **Catch me up** summarises a space, branch or collection since a day, as the person. And a space's stewards see its **service levels**, 12 weeks of first reply, time to close and late share by category, with a department seeing its teams' as counts, masked under 5 (sb-ask(3), Work across departments).

### Run the company (SB-4101 to SB-4103)

Each space can hold its **goals** (documents with `type: goal`, sb-ingest(7)), which a department sees with its teams' goals nested under its own; its weekly **status updates**, which the `update` agent drafts and a steward publishes, and which the department above lists team by team; and its **policies to read**, which its stewards mark and its members, and the members of its teams, are asked to read once per version. Stewards see by name who hasn't read a policy, and a count of who can't open it. Everything is checked as the person who looks, so a goal, an update or a policy they can't open is only counted, never named (sb-ask(3), Goals, updates and policies).

**Requests, budgets and a home (SB-4104 to SB-4106).** A space can take **requests** in up to 10 categories, each due in a number of working days; a request waits in the steward queue like a question and climbs the tree once it is past due. A space can have its own **monthly budget**, which its teams spend from: at 80% its stewards are told, and at 100% its scheduled briefs and updates pause until the month ends while asks carry on. And the Answer tab is each space's **home**: policies to read, goals, the latest updates, open requests and what changed this week, each checked as the reader and left out when empty (sb-ask(3), The web UI).

### Shape the work (SB-4601 to SB-4603)

A space's stewards shape what its agents write without building agents. **Skills** are named ways of writing ("weekly-update"), kept in the space's manifest with their eval cases and proposed like any change (Delegated changes, above), loaded by `/NAME` and inherited by its teams (sb-space(5), `skills`; sb-agents(7), Skills). **Ask for changes** sends an agent's draft back with a note, up to 3 rounds, from the same sources (sb-agents(7)). A **shared routine** is a steward's own routine shared with the space: each member who opts in gets their own copy, run as them; the stewards see only how many opted in, and a member who leaves the space has theirs stopped (sb-ask(3), Shared routines). None of them widens what anyone sees (LAW-19).

### Reorg preview (SB-1206)

Reorgs are planned. Before the IdP changes, HR and IT run `sb space diff` (sb-space(1)) on the manifest and a snapshot of the people being moved, with their groups before and after, and see per person the spaces they'd gain and lose. It flags anyone who would lose every space, gain a restricted one, or end up on both sides of a barrier. It works offline and changes nothing; the reorg itself is the IdP's SCIM moves (Moves, below) and, if spaces change, a manifest diff.

### Moves

When the IdP removes someone from a space's groups, they lose that space within 5 minutes. A new membership grants nothing until the cell has confirmed it from the directory (LAW-21, W12).

## LAWS

LAW-19 Spaces only narrow · LAW-20 Restricted stays home · LAW-21 Moves revoke fast · LAW-22 Barriers hold ([sb-laws(7)](sb-laws.7.md)). LAW-02 and LAW-03 are unchanged.

## EXAMPLES

A People brain that answers everyone's handbook questions but keeps case files home:

```yaml run
spaces:
  people:
    template: people
    members: [group:people-team]
    stewards: [group:hr-leads]
    label: restricted
    scope:
      docs: ["/People/**"]
```

Anyone may ask "how many days of parental leave do we get?" company-wide and get a cited answer, because the handbook is labelled `internal` at the source. A case file in `/People/Cases/` is `restricted`, so no company-wide answer for anyone outside `people-team` uses it, even if a Drive sharing mistake made it openable.

## SEE ALSO

[sb-space(5)](sb-space.5.md), [sb-departments(7)](sb-departments.7.md), [sb-permissions(7)](sb-permissions.7.md), [sb-policy(5)](sb-policy.5.md), [sb-tenant(5)](sb-tenant.5.md), [sb-tenancy(7)](sb-tenancy.7.md)
