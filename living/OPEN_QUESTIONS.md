---
title: Open Questions
summary: Questions only the parent organisation can answer, who decides, and the default if unanswered.
---

# Open questions

| ID | Question | Decides | Needed by | Default if unanswered |
|---|---|---|---|---|
| Q-01 | Does the parent organisation ever read portfolio-company data? | Parent-org leadership + legal | Before W6 | No. No cross-company features; ADR-0014 stays Proposed |
| Q-02 | Does any company ship software or connected devices into the EU (Cyber Resilience Act)? | Parent-org legal | W5 | Assume yes for the vulnerability-handling process; SB-505 covers it |
| Q-03 | Which IdPs and source systems do the first companies use, and how many companies in year one? | Parent-org IT + company IT | W1 (IdPs), W2 (sources) | One OIDC IdP; docs: Google Drive, chat: Slack, tickets: Linear (all chosen by the user, 2026-11-19) |
| Q-04 | Data-residency, retention and compliance baselines (SOC 2, ISO 27001, HIPAA, GDPR, others)? Model-provider data terms? | Parent-org security + legal | W1 (regions), W3 (provider) | EU and US regions (confirmed for now by the user, 2026-11-19; `eu`, `us` in sb-tenancy(7)); zero-retention model endpoint; SOC 2 controls as baseline. Cloud: cloud-neutral, adapters per cloud (ADR-0018). Embeddings: a deterministic local embedder behind the `Embedder` port until the provider is chosen (SB-206) |
| Q-05 | Who owns the eval suite and the approval rules inside each company? | Each company's leadership | W4 | Company IT lead owns both; platform team owns shared suites |
| Q-06 | Will the parent sign off the proposed metric targets? | Parent-org leadership | W3 | Targets in the PRD stand as working targets |
| Q-07 | Is the chat-app entry point required for go-live, or web-only acceptable? | Product owner | W3 | Both in v1 (SB-307, SB-308) |
| Q-08 | Who is on call for Sev-1 law violations: the platform team, company IT, or both? | Parent-org platform lead | W6 | Platform team primary, company IT informed |
| Q-09 | Which hand-off modes are needed (own cloud account, divestiture detach, both), and how likely is a divestiture in two years? | Parent-org leadership | Before W7 | Build both; rehearse detach on a copy only |
| Q-10 | After hand-off, who holds the model-provider contract and support terms? Does the factory keep any operational duty? | Parent-org legal + procurement | Before W7 | Tenant holds its own model credential; factory keeps only release publishing |
| Q-11 | Which HRIS and accounting systems do the first companies use, and which departments do they actually have? | Parent-org IT + company IT | Before W11 | BambooHR and QuickBooks Online on recorded fakes; the twelve templates in sb-departments(7) (the user, 2026-11-19) |
| Q-12 | For HR data in the EU, does a company need works-council or DPIA sign-off before a People space ingests more than policies? | Parent-org legal + company HR | Before W11 | Yes: People spaces ingest policies and the org chart only until sign-off is recorded |
