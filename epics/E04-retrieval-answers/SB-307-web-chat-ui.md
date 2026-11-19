---
title: "SB-307: Web chat UI"
summary: "The default entry point for staff."
---

# SB-307: Web chat UI

| Field | Value |
|---|---|
| Epic | [E04 Retrieval and Cited Answers](README.md) |
| Wave / slice | W3 / S3.2 Answers |
| Stories | US-04.4 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | [LAW-14](../../manual/sb-laws.7.md#law-14) |
| Depends on | [SB-306](../E04-retrieval-answers/SB-306-ask-agent-and-post-v1-ask.md) |
| Status | In review |

## Why

The default entry point for staff.

## Acceptance criteria

- [ ] Citations open the source at the cited version (the UI uses `cites[].url` and falls back to the trace; connectors fill `url` with the cell runtime)
- [x] `uncited` claims are visually distinct
- [ ] Meets WCAG 2.2 AA (axe checks pass in CI; colour contrast and a screen-reader pass need a human audit)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] UI shell
- [x] Citation chips
- [x] Accessibility checks
- [x] Write or extend the law tests: LAW-14
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S3.4): `web/` is the UI (React 18.3.1, Vite 5.4.11, TypeScript 5.6.3, exact versions with a lockfile). Claims render as cited, with one chip per citation linking to `url` or to the trace, or as uncited, with the word "Uncited" and a dashed border. Notices, the refused count, and errors with their remedy and trace are shown, and conversations carry `cv_…`. Vitest covers a LAW-14 rendering property over 200 random answers, including inconsistent ones (a mutation that ignores `uncited` is caught), and runs axe-core checks on the answer and the whole app. `internal/api.Web` serves `web/dist` at `/` with the sb-ask(3) CSP and never shadows `/v1`. The agent fills `cites[].url` through a `Link` function, and `answer.Cite` gained `url`. `make web` and a CI `web` job run typecheck, tests and the build.

**Left:** connector links at a version (with the cell runtime), and a human WCAG 2.2 AA audit (contrast in both themes, screen reader, keyboard).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
