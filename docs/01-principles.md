---
title: Principles
summary: The opinions this platform is built on, and what each one means in practice.
---

# Principles

These are opinions, on purpose. When two good ideas conflict, the principle with the lower number wins.

| # | Principle | In one line |
|---|---|---|
| P1 | Soundness over convenience | If the platform says it, it holds |
| P2 | The manual comes first | Every feature is a man page before it's code |
| P3 | Three lines beat ten | One obvious way per task |
| P4 | Data is not instructions | Ingested content is untrusted input |
| P5 | The model is userspace | Policy is the kernel; the model proposes, code disposes |
| P6 | SOLID and modular | The core depends on ports, never on a vendor or a company |
| P7 | Boring at scale | One trunk, one template, one release train, drills for everything |
| P8 | Evidence by default | If it isn't in the audit log, it didn't happen through us |

## P1: Soundness over convenience

The platform makes four kinds of claim: *you may see this*, *this is from that document version*, *this stayed inside your tenant*, and *this action ran*. Each one is either true or not made.

- An unknown permission result is a refusal (`EPERM`), never a pass. That includes timeouts, missing sources and connector errors.
- An answer with nothing to cite is labelled `uncited` **by code**, not by asking the model to say so.
- An action is "ran" only with a receipt from the target. The model's claim that it did something counts for nothing.
- When soundness costs latency, we pay the latency and then optimise. We never cache an *allow* across requests (ADR-0002).

## P2: The manual comes first

This is the UNIX way: write the man page, then make the program match it.

- Every feature starts as a change to `manual/`. Review the page, then write the code.
- A PR that changes behaviour without changing `manual/` is incomplete, and CI rejects it (SB-007). A labelled override exists for pure refactors.
- Manual examples tagged `run` are extracted and executed in CI, so the manual can't lie (SB-009).

## P3: Three lines beat ten

If three lines of config, CLI or code do the job well, ten lines is a defect.

- **One** manifest per tenant. **Three** onboarding verbs: `check`, `apply`, `verify`. **Four** connector calls: `fetch`, `acl`, `changes`, `act`.
- Convention over configuration: connector credentials live at a fixed vault path, so the manifest never mentions secrets.
- No per-company code paths. If a company needs something, it becomes a manifest field for everyone, or it doesn't happen.
- Names are greppable and say what they do: `sb tenant verify`, `EPERM`, `LAW-03`, `doc@version`.
- Formats are friendly to humans, LLMs and scripts alike: YAML for files, JSON for APIs, NDJSON for logs, stable error codes everywhere.

## P4: Data is not instructions

Every document, message and ticket is written by someone who isn't our user, and some of it will be hostile.

- Ingested text is sanitised, flagged and wrapped as quoted data before it reaches a prompt (`sb-ingest(7)`).
- Content can never cause a tool call. Only the user's request can, and only through the tool bus (LAW-09).
- Provenance travels with every chunk, so a citation can always show where text came from.

## P5: The model is userspace

Think of the policy engine as the kernel and the model as a user process.

- Every tool call goes through one path: kill-switch check, `authorise`, audit row, approval check, execute (`sb-agents(7)`).
- The model sees only chunks that passed live authorisation. It can't widen its own context.
- Prompts, tool definitions and the model version are pinned in the release bundle. A prompt change is a release (SB-503).

## P6: SOLID and modular

- **Single responsibility.** Each package has one job, named for it: `policy`, `audit`, `retrieve`, `compose`.
- **Open/closed.** New sources, IdPs and models are new adapters. The core doesn't change.
- **Liskov.** Every connector passes the same contract kit. If it passes, it's substitutable.
- **Interface segregation.** Ports are small. `Connector` has four methods, not forty.
- **Dependency inversion.** `internal/core` depends only on `internal/core/ports`. Adapters depend on the core, never the reverse. CI enforces this (SB-008).

## P7: Boring at scale

- One trunk, one cell template, one release train. Tenants choose a **channel** (`stable` or `testing`), never a version.
- No tenant is more than one release behind stable (LAW-08).
- Reproducible, signed builds with an SBOM. A CVE fix is a rebuild that every tenant takes.
- Managed services over self-run ones. No Kubernetes in v1 (ADR-0011).
- Drills, not hopes: kill switches every release, rollback every release, restore monthly, index rebuild on a schedule.

## P8: Evidence by default

- Every tool call writes an audit row **before** it executes (LAW-04). The log is append-only and hash-chained (LAW-11).
- Every decision records its reason: which policy, which release, which model.
- Every bug gets a root cause and the name of the law or test that now catches it (`living/BUGS_AND_MITIGATIONS.md`).

## Consequences we accept

- Some answers will be refused because a source was slow. That is correct behaviour, and the fix is a better connector.
- Onboarding a company with an unusual system waits for a connector that passes the contract kit. We won't add a special case.
- The first agent can't send anything by itself. That's a feature.

## See also

[Problem statement](00-problem-statement.md) · [Engineering standards](06-engineering-standards.md) · [ADRs](adr/README.md)
