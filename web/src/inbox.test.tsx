import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import { SavedView } from "./Saved";
import type { Fetch } from "./api";

afterEach(cleanup);

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

const inbox = {
  needs_you: [
    { kind: "request", id: "qs_1", space: "finance", title: "New laptop for Lee", at: "2026-11-20T09:00:00Z", due: "2026-11-22", overdue: true, link: "/?steward=finance" },
    { kind: "policy", id: "docs:Finance/expenses.md", space: "finance", title: "Expense policy", link: "/?space=finance" },
  ],
  failed: [{ kind: "routine", id: "rs_2", title: "What changed in the close?", at: "2026-11-24T08:00:00Z", why: "paused: the budget for this month is spent", link: "/?tab=legwork" }],
  done: [],
  counts: { needs_you: 2, failed: 1, done: 0 },
};

// SB-4301, SB-4303
describe("the inbox", () => {
  it("lists what waits on me with links to where it's acted on, and keeps lines about me", async () => {
    const calls: string[] = [];
    let lines: { id: string; text: string; at: string }[] = [];
    const fetcher: Fetch = async (u, init) => {
      calls.push(`${init?.method ?? "GET"} ${u}`);
      if (u === "/v1/me/inbox") return json(inbox);
      if (u === "/v1/me/memory" && init?.method === "POST") {
        const text = JSON.parse(String(init.body)).text as string;
        if (text.includes("Refunds take")) return json({ error: { code: "EINVAL", message: "That reads like a document, not something about you." } }, 400);
        lines = [...lines, { id: "mem_1", text, at: "2026-11-25T09:00:00Z" }];
        return json(lines[lines.length - 1]);
      }
      if (u === "/v1/me/memory/mem_1") {
        lines = [];
        return json({ removed: 1 });
      }
      if (u === "/v1/me/memory") return json({ on: true, lines });
      if (u === "/v1/questions") return json({ items: [] });
      return json({});
    };
    const { container } = render(<App fetcher={fetcher} />);
    await act(async () => {
      fireEvent.click(screen.getByRole("tab", { name: "Inbox" }));
    });
    expect(screen.getByText("Needs you (2)")).toBeTruthy();
    expect(screen.getByText("Overdue")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Open: New laptop for Lee" }).getAttribute("href")).toBe("/?steward=finance");
    expect(screen.getByRole("link", { name: "Open: Expense policy" }).getAttribute("href")).toBe("/?space=finance");
    expect(screen.getByText(/paused: the budget/)).toBeTruthy();
    expect(calls.filter((c) => c.startsWith("POST") && c.includes("inbox"))).toEqual([]); // the inbox never acts
    fireEvent.change(screen.getByLabelText("Something about you"), { target: { value: "I lead the finance team." } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Add" }));
    });
    expect(screen.getByText("I lead the finance team.")).toBeTruthy();
    fireEvent.change(screen.getByLabelText("Something about you"), { target: { value: "Refunds take 10 days to reach the customer." } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Add" }));
    });
    expect(screen.getByText("That reads like a document, not something about you.")).toBeTruthy();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Remove: I lead the finance team." }));
    });
    expect(screen.queryByText("I lead the finance team.")).toBeNull();
    const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
    expect(r.violations.map((v) => v.id)).toEqual([]);
  });
});

// SB-4302
describe("routines", () => {
  it("makes a saved question a routine on a chosen day, and stops it", async () => {
    const bodies: string[] = [];
    let routine: unknown;
    const fetcher: Fetch = async (u, init) => {
      if (u === "/v1/saved/sq_1/routine") {
        bodies.push(String(init?.body));
        const b = JSON.parse(String(init?.body));
        routine = b.every === "off" ? undefined : b;
        return json({ id: "sq_1", question: "What changed in the close?", space: "spc_finance", at: "2026-11-20T09:00:00Z", routine });
      }
      if (u === "/v1/saved") return json({ items: [{ id: "sq_1", question: "What changed in the close?", space: "spc_finance", at: "2026-11-20T09:00:00Z", routine }] });
      return json({});
    };
    render(<SavedView fetcher={fetcher} onAsk={() => {}} />);
    const select = await screen.findByLabelText("Routine: What changed in the close?");
    await act(async () => {
      fireEvent.change(select, { target: { value: "tuesday" } });
    });
    expect(bodies[0]).toBe(JSON.stringify({ every: "weekly", day: "tuesday" }));
    expect((screen.getByLabelText("Routine: What changed in the close?") as HTMLSelectElement).value).toBe("tuesday");
    await act(async () => {
      fireEvent.change(screen.getByLabelText("Routine: What changed in the close?"), { target: { value: "off" } });
    });
    expect(bodies[1]).toBe(JSON.stringify({ every: "off" }));
  });
});
