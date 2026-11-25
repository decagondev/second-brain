import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import type { Fetch } from "./api";

afterEach(cleanup);

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

const plan = { id: "rs_1", kind: "research", question: "How did the close go?", space: "company", state: "planned", steps: ["What moved?", "What is open?"], at: "2026-11-20T09:00:00Z", cost_usd: 0, summary: [], sections: [], dropped: 0 };

// SB-4201, SB-4203
describe("the legwork tab", () => {
  it("shows a plan before anything runs, runs it only on Run, and marks an action item done", async () => {
    const calls: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      calls.push(`${init?.method ?? "GET"} ${u}`);
      if (u === "/v1/research" && init?.method === "POST") return json(plan);
      if (u === "/v1/research/rs_1/run") return json({ ...plan, state: "queued" });
      if (u === "/v1/research") return json({ items: [] });
      if (u === "/v1/me/actions") return json({ actions: [{ id: "ac_1", space: "finance", text: "send the report", owner: "sam", due: "2026-12-01" }] });
      if (u === "/v1/me/actions/ac_1/done") return json({ id: "ac_1", space: "finance", text: "send the report", owner: "sam", done_at: "2026-11-21T09:00:00Z" });
      return json({});
    };
    const { container } = render(<App fetcher={fetcher} />);
    await act(async () => {
      fireEvent.click(screen.getByRole("tab", { name: "Legwork" }));
    });
    fireEvent.change(screen.getByLabelText("A question that needs more than one search"), { target: { value: "How did the close go?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Plan" }));
    });
    expect(screen.getByText("What moved?")).toBeTruthy();
    expect(calls.some((c) => c.includes("/run"))).toBe(false);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Run" }));
    });
    expect(calls).toContain("POST /v1/research/rs_1/run");
    expect(screen.getByText("Queued. You're told when it's ready.")).toBeTruthy();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Mark done" }));
    });
    expect(screen.queryByRole("button", { name: "Mark done" })).toBeNull();
    const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
    expect(r.violations.map((v) => v.id)).toEqual([]);
  });
});
