---
title: One Org, Many Levels; An Agent for Every Department; Run the Company
summary: Concept for W38 to W40 (proposed 2026-11-24): twenty features that turn the second brain into a company OS, with spaces arranged as company, department and team, a read-only agent for every department, and goals, updates, policies and requests on top. No new service, store or Go module.
---

# One org, many levels; an agent for every department; run the company

| Version | Date | Owner | Status | Related |
|---|---|---|---|---|
| 0.1 | 2026-11-24 | Platform lead | Proposed, waiting on the user's decisions | [Principles](01-principles.md) · [08-department-brains.md](08-department-brains.md) · [16-trust-capture-reach.md](16-trust-capture-reach.md) · [PRD waves](03-prd.md#waves) |

## Summary

W35 to W37 are built and pushed (`60d298c`). The user asked for a deeper dive this time: what would most turn the product into a **company OS** built on the second brain, with **agentic pieces for each department, in a hierarchy**.

The department brains plan of 2026-11-19 ([08-department-brains.md](08-department-brains.md), W8 to W12) is built. It gave us **spaces** (one per department, from twelve templates), labels, information barriers, stewards, access reviews, legal holds, reorg preview and four read-only department agents. Three things are still missing, and they are what this plan adds.

1. **Spaces are flat.** A real company is a tree: the company, then departments, then teams (Finance, then Accounts Payable and FP&A). Today a team is just another space beside its department. Nothing inherits, nothing rolls up, and nothing escalates up the tree.
2. **Only four departments have an agent.** Engineering, Product, Finance and IT/Security do. People, Legal, Sales, Support, Marketing, Operations, Design and Executive have only `ask`. Every agent runs only when someone asks. None runs on a schedule or when something happens, and nobody can see every agent in one place.
3. **The brain knows things but doesn't help run anything.** Goals, weekly updates, policies people must read, and requests to a department (IT access, HR questions) still live in other tools. Those are the parts of a company OS that sit closest to knowledge.

Three waves, twenty tickets: **W38 One Org, Many Levels** (the tree), **W39 An Agent for Every Department** (agents), and **W40 Run the Company** (goals, updates, policies, requests, budgets and a home for each space).

## What others do, 2026

| Product | What it does that matters here |
|---|---|
| **Glean** | An agent library filtered by department (Engineering, Sales, HR, Support, IT), with templates that list the connectors each needs. Sub-agents, scheduled and content-change triggers, verification badges, departmental categories, access policies that keep some groups from writing to systems of record, and an insights dashboard (adoption, hours saved, feedback). |
| **Microsoft Agent 365** | A registry of every agent in the tenant: publisher, owner, state, connectors and permissions. Admins can block an agent or give it a new owner, and get cards for agents without owners or at risk. Usage, errors and run time per agent. Least privilege per agent. |
| **Notion custom agents** | Agents made from a job description and triggers (Q&A, task routing, status updates). Logged runs, reversible changes, a usage dashboard, and an agent that pauses itself at its credit limit. |
| **Atlassian Rovo and the Teamwork Graph** | One list of every agent, with control of who can build and who can run them. Triggers such as a new hire joining or a Sev-1 opening. A plan that a person approves before it runs. A graph of people, goals and work that knows who really owns a project. |
| **YC's "AI operating system for companies" brief (Summer 2026)** | An agent registry with discovery, scheduled and event triggers, a token budget and cost allocation per department, company goals given to agents as context, output review queues and human approval for high-stakes work, an audit log of every agent action, and adoption per department. |

**What they agree on:** a catalogue of agents by department, one registry with owners and usage, scheduled and event triggers, a person approving what agents produce, cost per department, and goals as shared context.

**Where we differ, on purpose:** none of them has permissions that follow the org tree. Each agent has its own permissions, set by an admin. Here an agent runs **as the person**, in a space, with the source's live ACL as the ceiling (ADR-0002), so a team-level agent can never see more than its team. We also keep agents as reviewed definitions in the release bundle (P3), not something any employee builds in a browser. A company that needs a new agent asks for it, as Glean's "Request an agent" does.

## Keeping it small

The bloat rules from earlier waves still apply:

1. **No new service the cell must run.** Scheduled agents run in the existing `sb-jobs`.
2. **No new store.** Goals, updates and policies are documents in the source with front matter. Acknowledgements, requests and agent schedules are `cell_docs` kinds.
3. **No new third-party Go module.**
4. **At most one setting per feature.**
5. **No new page.** Each space's **Start** tab becomes its home.
6. **Nothing widens what anyone sees.** Every title, count and list is checked live as the person.
7. **A person approves every write.** Anything an agent writes is a draft until a person publishes it, as notes already are (SB-1705, LAW-16).

Two more rules for these waves:

8. **The tree only narrows.** A team can be stricter than its department, and never looser. A department's view of its teams is what its members could open anyway (a new law, LAW-24).
9. **Agents are definitions in the release bundle.** Each has a prompt, no tools unless it is a workflow agent, and an eval suite in the gate. Stewards choose which ones their space runs; nobody writes a prompt in production.

## The twenty features

| # | Ticket | Wave | Feature | For | In one line | From |
|---|---|---|---|---|---|---|
| 1 | SB-3901 | W38 | Parent spaces | Admin | A space names its `parent`. The tree is at most three levels: company, department, team | Teamwork Graph, org charts |
| 2 | SB-3902 | W38 | Settings flow down | Steward | Glossary, start-here, review interval and agents pass to child spaces. Label and retention can only get stricter down the tree | Org policy inheritance |
| 3 | SB-3903 | W38 | Ask across a branch | Asker | In a department, **This department and its teams** asks across its teams' spaces that the asker is a member of | Glean, Rovo |
| 4 | SB-3904 | W38 | Escalate up the tree | Asker, steward | A question unanswered in a team for 10 working days goes to its department, then to the company-wide answerers | Service desks |
| 5 | SB-3905 | W38 | Roll-ups for leaders | Department steward | The review list and usage show each team's counts under the department. Rows with fewer than 5 are hidden | Rovo, Agent 365 |
| 6 | SB-3906 | W38 | Who's who in a branch | Everyone | A space's people: its stewards, caretakers and the owners of its most-cited documents. **Who to ask** climbs to the department when a team has nobody | Teamwork Graph |
| 7 | SB-3907 | W38 | The tree never widens | Everyone | LAW-24 and its law test: a branch ask, a roll-up or an inherited setting never shows more than each member could open | — |
| 8 | SB-4001 | W39 | People and Legal agents | People, Legal | `policy-answer` gives the policy that applies, with its version. `clause-finder` finds the clauses in contracts on a topic | Glean, Lyzr |
| 9 | SB-4002 | W39 | Sales, Support and Marketing agents | Sales, Support, Marketing | `account-brief` covers an account's history. `case-brief` covers a support case and what solved similar ones. `brand-check` checks a draft against the brand guide | Glean |
| 10 | SB-4003 | W39 | Operations, Design and Executive agents | Ops, Design, Exec | `sop-steps` gives a procedure's steps in order. `design-rationale` says why a design is as it is. `exec-brief` covers what the leadership team decided and what's open | Glean, YC brief |
| 11 | SB-4004 | W39 | Every agent in one place | Admin, steward | `GET /v1/admin/agents` lists each agent with its department, the spaces it runs in, runs, the share rated unhelpful, eval status and kill state | Agent 365, Rovo |
| 12 | SB-4005 | W39 | Agents on a schedule | Steward | A space can run one agent weekly, such as `exec-brief` or a week-in-the-department brief. Its output waits in the steward queue as a draft | Notion, Glean, Rovo |
| 13 | SB-4006 | W39 | Agents when something happens | Joiner, steward | A joiner gets a welcome brief from the start-here lists of their spaces, up the tree. A new transcript drafts its decisions for the steward | Rovo (new hire), Glean |
| 14 | SB-4007 | W39 | The right department | Asker | A company-wide ask that matches a department suggests that department's agent, if the asker is a member | Glean sub-agents |
| 15 | SB-4101 | W40 | Goals | Everyone | Documents with `type: goal` front matter (owner, quarter, parent goal, status) become goals. The tree shows them by space, and answers cite them | YC brief, Lattice |
| 16 | SB-4102 | W40 | Status updates | Team steward, leader | A weekly update is drafted from what changed, its decisions and its goals. The team's steward edits and publishes it, and the department sees its teams' updates | Notion status agents, Rovo |
| 17 | SB-4103 | W40 | Policies to read | Steward, member | A steward marks a document as a policy for a branch. Members are told, and press **I've read it** for each version. Stewards see who hasn't | Compliance tools |
| 18 | SB-4104 | W40 | Requests to a department | Everyone, steward | A request with a category ("IT: access") goes to a space's queue, with a due date in working days. Overdue requests go up the tree | Service desks, YC brief |
| 19 | SB-4105 | W40 | Department budgets | Admin, steward | A monthly token budget for a branch with its forecast. Warns at 80%, and scheduled agents pause at 100% | Notion auto-pause, YC brief |
| 20 | SB-4106 | W40 | A home for each space | Everyone | The Start tab shows the space's goals, its latest update, policies to read, open requests and what changed | Notion, Rovo |

## W38 One org, many levels (epic E39, SB-3901 to SB-3907)

The goal is that spaces follow the company's shape (company, department, team), and access only narrows down the tree.

### S38.1 The tree

**SB-3901 Parent spaces.** sb-space(5) gains `parent: NAME`. A space with no parent is a department, or a team of the company itself. The tree is at most three levels deep (company, department, team), with no cycles. `sb tenant check` refuses a deeper tree, a missing parent or a cycle, with the line. A child space must be at least as strict as its parent: its label is at least the parent's, its retention is at most the parent's, and it inherits its parent's barriers. Members aren't inherited. A team's members are its own groups, since IdPs already nest groups, and a person in a team space who isn't in the department space doesn't become a department member. Reorg preview (SB-1206) shows a change of `parent` like any change to the manifest.
*Laws:* LAW-24 (new), LAW-21. *Acceptance:* a four-level tree, a cycle and a looser child are refused with their lines. The registry knows each space's parent and depth.

**SB-3902 Settings flow down.** A child space without its own `glossary`, `start_here`, `review_after_days` or `agents` takes its parent's. Glossaries merge, and the child's own term wins. Start-here lists join, the department's first, at most ten. A child can only tighten `label` and `retention_days`, and anything looser is refused (SB-3901). Stewards of the parent are also stewards of each child, for questions that come up the tree (SB-3904) and for roll-ups (SB-3905), but not for publishing notes in the child unless they're named there.
*Laws:* LAW-24. *Acceptance:* a team with no glossary uses its department's. A team's own term wins. A looser label is refused.

**SB-3903 Ask across a branch.** In a space that has children, the switcher offers **This department and its teams** (`?space=finance&branch=true`). Retrieval then runs the space checks once per space in the branch the asker is a member of, and unions the chunks each check allowed. A team the asker isn't in adds nothing, and isn't named. Cites say which space each came from. Without `branch`, nothing changes.
*Laws:* LAW-01, LAW-24. *Acceptance:* a department member in one team gets that team's documents and not another's. A non-member gets the department only.

### S38.2 Leaders and escalation

**SB-3904 Escalate up the tree.** The reroute of SB-3404 climbs the tree. A question still open after 10 working days in a team goes to its department's stewards. After 10 more it goes to the company-wide answerers. It stays in each earlier queue, with "waited in ap" on it, and is audited at each step (`rerouted`, `to`). A claimed question waits until the claim lapses, as now.
*Laws:* LAW-07, LAW-11. *Acceptance:* a team question reaches the department on day 10 and the company on day 20. A claim holds it. Counts only in notifications.

**SB-3905 Roll-ups for leaders.** A department's review list gains **Teams**: per child space, the counts it already shows its own stewards (to review, overdue, questions open and found-nothing). The usage row (SB-3506) does the same. Rows under 5 asks show "fewer than 5", as now. The weekly report's usage by space nests teams under their department. Titles and questions never roll up, only counts.
*Laws:* LAW-07, LAW-24. *Acceptance:* a department steward sees each team's counts and no team document's title. A team with 3 asks shows "fewer than 5".

**SB-3906 Who's who in a branch.** `GET /v1/spaces/NAME/people` lists the space's stewards, caretakers and the owners of its ten most-cited documents this month, each by name and email, checked live (an inactive person is left out, LAW-12). It counts members, without listing them. **Who to ask** on an answer climbs to the department's stewards when no owner of a cited document is active. The web UI shows it on the space's Start tab (SB-4106).
*Laws:* LAW-01, LAW-12. *Acceptance:* owners appear only for documents the asker can open. A deprovisioned steward disappears at once.

**SB-3907 The tree never widens.** A new law, **LAW-24**: *a space's place in the tree never lets anyone see, count or be told about a document they couldn't see from the space that holds it.* Its law test runs generated trees with random members, labels and barriers through branch asks, roll-ups, inherited start-here lists and escalations, and checks every result against the leaf space's own decision. The selftest breaks it by unioning members up the tree, and it must go red.
*Laws:* LAW-24. *Acceptance:* the law test is green for 100 generated trees, and red for the broken system.

## W39 An agent for every department (epic E40, SB-4001 to SB-4007)

The goal is that every department template has an agent worth asking, agents can run on a schedule or when something happens, and admins see every agent in one place.

### S39.1 The catalogue

All six new agents follow the department agent rules (sb-agents(7)): a prompt in the release bundle, run on the `ask` path as the asker, only in their template's spaces, every claim cited (an uncited claim is dropped), no tools, and an eval suite in the gate.

**SB-4001 People and Legal agents.** `policy-answer` (People): the policy that applies to a question, quoted, with its version and its review date, and the person to ask. It never answers about a named employee's case. `clause-finder` (Legal): the clauses on a topic across the contracts the asker can open, one cited line each, grouped by contract. It never gives legal advice beyond the text.
*Laws:* LAW-01, LAW-09, LAW-14. *Acceptance:* each agent's eval suite passes the gate. Each is refused outside its template's spaces.

**SB-4002 Sales, Support and Marketing agents.** `account-brief` (Sales): an account's history from the CRM fields and documents the asker can open (open deals, last contact, the people), every line cited. `case-brief` (Support): a ticket's history and how similar tickets were solved, cited. `brand-check` (Marketing): a pasted draft checked against the brand guide, each finding citing the rule. It suggests, and never rewrites.
*Laws:* LAW-01, LAW-09, LAW-14. *Acceptance:* each agent's eval suite passes the gate. `brand-check` treats the draft as data (LAW-09).

**SB-4003 Operations, Design and Executive agents.** `sop-steps` (Operations): a procedure's steps in order, each cited, and which version. `design-rationale` (Design): why a design is as it is, from specs, decisions and reviews. `exec-brief` (Executive): what the leadership team decided this month and what's open, from decisions and minutes.
*Laws:* LAW-01, LAW-09, LAW-14. *Acceptance:* each agent's eval suite passes the gate. `exec-brief` runs only in a restricted space made from `executive`.

### S39.2 Agents that run themselves

**SB-4004 Every agent in one place.** `GET /v1/admin/agents` lists every agent in the release: its department, the spaces it runs in (from the manifest), runs and the share rated unhelpful in the last 30 days, the last eval result for this release, and whether it's killed. The admin console shows it. A space's stewards see the same for the agents in their space on the review list, and propose turning one on or off as a manifest diff (W12). Counts only.
*Laws:* LAW-07, LAW-10. *Acceptance:* the list matches the manifest and the traces. A killed agent shows killed.

**SB-4005 Agents on a schedule.** A space can set `brief: { agent: NAME, weekly: monday }`. The `notify` job then runs that agent once a week **as a steward of the space** (the first named, checked active), with a fixed question for that agent. The answer becomes a draft in the space's steward queue, as a saved thread does (SB-3703). It reaches anyone else only when a steward publishes it as a note. If the space's budget is spent (SB-4105), the run is skipped and the stewards are told. One schedule per space.
*Laws:* LAW-16, LAW-12, LAW-07. *Acceptance:* the brief waits for a steward and nothing is published by itself. A space with no active steward runs nothing.

**SB-4006 Agents when something happens.** Two fixed triggers, no others. When SCIM adds someone to a space's group, they get a **welcome brief**: the start-here lists of that space and each space above it, run as them, each item checked as them, sent as one notification (`welcome`, off-able). When a new transcript lands in a space (SB-3701), a draft of its decisions goes to the steward queue. Every trigger writes a trace and an audit row.
*Laws:* LAW-01, LAW-12, LAW-16. *Acceptance:* a joiner gets one brief with only what they can open. A mover gets the new space's. A transcript's decisions wait for a steward.

**SB-4007 The right department.** A company-wide ask whose terms match a department's glossary well (the same term counting as trends, SB-3704) suggests "Finance's `close-checklist` may answer this", with a button that asks it there. Only spaces the asker is a member of are suggested, at most one. It's a suggestion only.
*Laws:* LAW-01. *Acceptance:* a member sees the suggestion. A non-member never sees that space named.

## W40 Run the company (epic E41, SB-4101 to SB-4106)

The goal is that the work around knowledge (goals, updates, policies, requests and budgets) lives where the knowledge is, without becoming a second project tool.

### S40.1 Goals and updates

**SB-4101 Goals.** A document with front matter `type: goal`, `owner`, `quarter`, `status` (on track, at risk, off track, done) and optionally `parent_goal` is typed `goal` (`ask.TypeOf`). `GET /v1/spaces/NAME/goals` lists the branch's goals by quarter, each checked as the asker, nested by `parent_goal`. A goal whose parent the asker can't open shows without it. Answers cite goals like any document. Goals stay in the source; the brain only reads them.
*Laws:* LAW-01, LAW-13. *Acceptance:* a goal file appears under its space and quarter. A goal the asker can't open is left out.

**SB-4102 Status updates.** Each Monday, a space with `updates: true` gets a draft update in its steward queue, from the `update` agent: what changed in the space last week, its new decisions and its goals' status, each line cited. The steward edits it and publishes it as a note (SB-1705) under `notes/SPACE/updates/`. The parent space's Start tab lists its children's latest published updates, each checked as the reader. Nothing is published without the steward.
*Laws:* LAW-16, LAW-01. *Acceptance:* the draft cites its lines. An unpublished draft is seen by nobody but the stewards.

**SB-4103 Policies to read.** A steward marks a document as a **policy** for their space's branch (`POST /v1/spaces/NAME/policies`). Members who can open it are told once (`policy`, a notification kind) and see it on their Start tab until they press **I've read it**, recorded per person and version as a `cell_docs` record and audited. A new version asks again. The space's stewards see who hasn't read it, by name, which is the point of the record. Members who can't open the document are counted, not named, so the steward can fix its access at the source.
*Laws:* LAW-01, LAW-11, LAW-07. *Acceptance:* a new version asks again. The steward's list matches the records. A member who can't open it is only counted.

### S40.2 Requests, budgets and a home

**SB-4104 Requests to a department.** A space can set `requests: [ "Access", "Laptop", … ]`, at most 10 categories, with `due_days`. **Ask the department** on its Start tab sends a request with a category and text to its steward queue. It's a steward question with a category and a due date, so claims, replies, notifications and escalation up the tree (SB-3904) work unchanged. The requester sees its state on the Questions tab. The brain doesn't do the work. A request needing a change goes to a workflow agent, through approvals, as now.
*Laws:* LAW-07, LAW-11. *Acceptance:* a request gets its category and due date, and goes up the tree when overdue. Its notifications carry counts only.

**SB-4105 Department budgets.** A space can set `budget_usd_month`. The cell meters each ask and each scheduled run by the space it ran in, rolled up the tree, with the forecast of SB-3505. The space's stewards are told at 80% of its budget. At 100%, scheduled briefs and updates (SB-4005, SB-4102) pause until the month ends. Asks carry on until the tenant's own budget, since stopping a person mid-question helps nobody. The admin console shows each branch's spend.
*Laws:* LAW-07. *Acceptance:* a space over budget pauses its brief and still answers. Spend rolls up to the department.

**SB-4106 A home for each space.** The Start tab becomes the space's home, in this order: policies to read, the space's goals this quarter, its latest published update (and its teams', for a department), open requests, and what changed this week. Each part is one existing call, checked as the reader, and is hidden when empty. No new page.
*Laws:* LAW-01. *Acceptance:* a member sees each part with only what they can open. An empty part is hidden.

## What we leave out

- **Agents people build themselves in the browser** (Notion, Glean, Copilot Studio). Every agent here is reviewed and evaluated in the release (P3). A company asks for one, and it ships in a release.
- **Agents that act on their own across tools** (Rovo Max, Glean autonomous agents). Actions still go through approvals (sb-approvals(7)).
- **A project tracker or HR system.** Goals, updates and requests stay light: read from the source or kept as steward questions. Jira, Linear and the HRIS stay the systems of record.
- **Parent-company portfolio grants** (E13). Still parked while Q-01 is no.

## Decisions for the user

Waiting on the user. Each has a recommended option:

1. **Scope and order.** *Recommended:* all three waves, all twenty tickets, in the order W38, W39, W40. The tree comes first because agents, roll-ups and budgets follow it. *Other:* W39 first, so the new agents land sooner.
2. **The tree (SB-3901 to SB-3903).** *Recommended:* at most three levels set by `parent`. A child can only be stricter, members aren't inherited, and a branch ask covers only the teams the asker is in. *Other:* any depth.
3. **Who writes agents (SB-4001 to SB-4003).** *Recommended:* agents stay definitions in the release bundle, reviewed and evaluated. Stewards choose which ones their space runs. *Other:* stewards may write a space's own prompt, with its own eval cases.
4. **Agents that run themselves (SB-4005, SB-4006).** *Recommended:* one weekly brief a space and two fixed triggers (a joiner, a new transcript). Output is a draft a steward publishes, or a message to the person themselves, and these agents never call tools. *Other:* briefs post straight to the space's chat channel.
5. **Policies to read (SB-4103).** *Recommended:* stewards see by name who hasn't read a policy, since that is what the record is for. *Other:* counts only.
6. **Budgets (SB-4105).** *Recommended:* warn at 80%, pause scheduled agents at 100%, and keep answering asks up to the tenant's budget. *Other:* refuse asks in a space over its budget.

## Still open from earlier waves

- **SB-1605** and **SB-2601** need a run on a real Linux VM.
- **SB-903** and **SB-908** need a real GCP project.
- **Mattermost (SB-2005, SB-2105, SB-2603)** has been tested against a fake server only.

## Sources

- [Glean: Agent Library](https://www.glean.com/ai-agents/agent-library)
- [Glean: agent templates](https://docs.glean.com/agents/templates)
- [Glean: the enterprise agent development lifecycle](https://www.glean.com/press/glean-introduces-the-enterprise-agent-development-lifecycle-codifying-how-enterprises-build-govern-and-measure-ai-agents)
- [Microsoft Agent 365: agent registry](https://learn.microsoft.com/en-us/microsoft-agent-365/builder/agent-registry)
- [Microsoft Agent 365: managing agents](https://learn.microsoft.com/en-us/microsoft-agent-365/admin/manage-agents)
- [Notion: custom agents](https://www.notion.com/blog/introducing-custom-agents)
- [Atlassian: Rovo at Team '26](https://www.atlassian.com/blog/company-news/rovo-team-26)
- [Atlassian: Teamwork Graph at Team '26](https://www.atlassian.com/blog/company-news/teamwork-graph-team-26)
- [The AI operating system for companies (YC RFS, Summer 2026)](https://modelence.com/yc-rfs-summer-2026/the-ai-operating-system-for-companies)
- [Lyzr: agentic OS platforms](https://www.lyzr.ai/blog/best-agentic-os-platforms/)
