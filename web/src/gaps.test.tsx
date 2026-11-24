import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { ReviewList, StewardQuestions } from "./Questions";
import type { Fetch } from "./api";

afterEach(cleanup);

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

// SB-3101, SB-3103, SB-3104: found nothing, feedback, several at once.
describe("steward review: gaps and feedback", () => {
  function cell() {
    const calls: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      const m = init?.method ?? "GET";
      calls.push(`${m} ${u} ${init?.body ?? ""}`);
      if (u.endsWith("/review/marks")) return json({ items: [{ cite: "docs:a@1", ok: true }, { cite: "docs:b@1", error: { code: "EDENIED", message: "You can't mark this one" } }] });
      if (u.includes("/review")) {
        return json({
          items: [
            { cite: "docs:a@1", title: "Rota", snippet: "", flags: [], cited: 3, flagged: 0 },
            { cite: "docs:b@1", title: "Cutoffs", snippet: "", flags: [], cited: 1, flagged: 1 },
          ],
          hidden: [],
          found_nothing: { this_week: 4, last_week: 1, terms: ["cutoff", "rota"] },
          feedback: {
            this_month: { helpful: 5, not_helpful: 2 },
            last_month: { helpful: 3, not_helpful: 1 },
            documents: [{ cite: "docs:b@1", title: "Cutoffs" }],
            documents_hidden: 1,
          },
        });
      }
      return json({ items: [] });
    };
    return { calls, fetcher };
  }

  it("shows counts and terms, never a question, and marks several at once", async () => {
    const { calls, fetcher } = cell();
    const { container } = render(<ReviewList space="ops" fetcher={fetcher} />);
    expect(await screen.findByText(/Questions that found nothing: 4 this week, 1 the week before\. Glossary terms they used: cutoff, rota\./)).toBeTruthy();
    expect(screen.getByText("This month: 5 helpful, 2 not helpful. Last month: 3 helpful, 1 not helpful.")).toBeTruthy();
    expect(screen.getByText("and 1 document you can't open")).toBeTruthy();
    fireEvent.click(screen.getByRole("checkbox", { name: "Select Rota" }));
    fireEvent.click(screen.getByRole("checkbox", { name: "Select Cutoffs" }));
    expect(screen.getByText("2 selected")).toBeTruthy();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Mark reviewed" }));
    });
    expect(calls.some((c) => c === `POST /v1/spaces/ops/review/marks {"cites":["docs:a@1","docs:b@1"]}`)).toBe(true);
    expect(screen.getByText("docs:b@1: You can't mark this one")).toBeTruthy();
    expect(screen.getByText("1 selected")).toBeTruthy();
    const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
    expect(r.violations.map((v) => v.id)).toEqual([]);
  });

  it("retires only with a reason", async () => {
    const { fetcher } = cell();
    render(<ReviewList space="ops" fetcher={fetcher} />);
    fireEvent.click(await screen.findByRole("checkbox", { name: "Select Rota" }));
    expect((screen.getByRole("button", { name: "Retire" }) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.change(screen.getByLabelText("Reason"), { target: { value: "Moved." } });
    expect((screen.getByRole("button", { name: "Retire" }) as HTMLButtonElement).disabled).toBe(false);
  });
});

// SB-3102: a question rerouted from a space with no steward says where it came from.
describe("rerouted question", () => {
  it("names the space it came from", async () => {
    const fetcher: Fetch = async () =>
      json({ items: [{ id: "q1", question: "Who covers nights?", at: "2026-11-23T10:00:00Z", state: "open", from_space: "night-shift" }] });
    render(<StewardQuestions space="company" fetcher={fetcher} />);
    expect(await screen.findByText("night-shift has no active steward")).toBeTruthy();
  });
});
