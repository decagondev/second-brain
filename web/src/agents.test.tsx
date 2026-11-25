import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { AdminAgents, SpaceAgents } from "./Agents";
import { StewardQuestions } from "./Questions";
import { App } from "./App";
import type { Fetch } from "./api";

afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const rows = [
  { name: "ask", spaces: ["eng", "finance"], runs: 12, rated: 4, unhelpful: 1, eval: "passed", suite: "gate", killed: false },
  { name: "incident-summary", department: "engineering", spaces: ["eng"], runs: 3, rated: 0, unhelpful: 0, eval: "held", suite: "agent-incident-summary", killed: true },
];

// SB-4004: every agent in one place, and a steward's proposal of its space's agents.
describe("Agents", () => {
  it("lists every agent for an operator", async () => {
    const fetcher: Fetch = async () => new Response(JSON.stringify({ agents: rows, days: 30 }), { status: 200 });
    const { container } = render(<AdminAgents fetcher={fetcher} />);
    await act(async () => {});
    expect(screen.getByRole("rowheader", { name: "incident-summary" })).toBeTruthy();
    expect(container.textContent).toContain("25%");
    expect(container.textContent).toContain("held (agent-incident-summary)");
    expect(container.textContent).toContain("killed");
    expect(await violations(container)).toEqual([]);
  });

  it("proposes a space's agents as a manifest diff", async () => {
    const calls: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      calls.push(`${init?.method ?? "GET"} ${u} ${init?.body ?? ""}`.trim());
      if (u.endsWith("/proposals/agents"))
        return new Response(JSON.stringify({ id: "prop_1", hash: "h", diff: ["~ spaces.eng.agents"], plan: "" }), { status: 200 });
      return new Response(JSON.stringify({ agents: rows, listed: [], can_list: ["incident-summary"], days: 30 }), { status: 200 });
    };
    const { container } = render(<SpaceAgents base="/v1/spaces/eng" fetcher={fetcher} />);
    await act(async () => {});
    fireEvent.click(screen.getByRole("checkbox", { name: "incident-summary" }));
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Propose these agents" }));
    });
    expect(calls).toContain(`POST /v1/spaces/eng/proposals/agents {"agents":["incident-summary"]}`);
    expect(container.textContent).toContain("~ spaces.eng.agents");
    expect(screen.getByRole("status").textContent).toContain("prop_1");
    expect(await violations(container)).toEqual([]);
  });

  // SB-4005: a weekly brief waits as a draft; another steward sees that it's there, not its text.
  it("shows an agent's draft, and hides its text from another steward", async () => {
    const item = (hidden: boolean) => ({ id: "qs_b", question: "Weekly brief: incident-summary", at: "2027-03-01T00:00:00Z", state: "open",
      thread: { by: "sam@acme-logistics.example", link: "", text: hidden ? "" : "- Outage on Monday. [Postmortem]", agent: "incident-summary", kind: "brief", hidden } });
    for (const hidden of [false, true]) {
      const fetcher: Fetch = async (u) => new Response(JSON.stringify({ items: u.endsWith("/questions") ? [item(hidden)] : [] }), { status: 200 });
      const { container } = render(<StewardQuestions space="eng" fetcher={fetcher} />);
      await act(async () => {});
      const g = screen.getByRole("group", { name: "Drafted by an agent" });
      expect(g.textContent).toContain("Weekly brief by incident-summary");
      expect(screen.queryByRole("group", { name: "Saved from chat" })).toBeNull();
      expect(g.textContent?.includes("Only sam@acme-logistics.example sees its text")).toBe(hidden);
      if (!hidden) expect((screen.getByLabelText("Reply to: Weekly brief: incident-summary") as HTMLTextAreaElement).value).toContain("Outage on Monday");
      expect(await violations(container)).toEqual([]);
      cleanup();
    }
  });

  // SB-4007: a company-wide answer suggests the asker's department agent; the button asks it there.
  it("asks the suggested department's agent in its space", async () => {
    const calls: string[] = [];
    const answer = { trace: "tr_1", release: "rel_x", refused: 0, notice: null, conversation: "c1", answer: { claims: [{ text: "Done.", cites: [] }] } };
    const fetcher: Fetch = async (u, init) => {
      if (u === "/v1/notice") return new Response("{}", { status: 200 });
      if (u === "/v1/spaces") return new Response(JSON.stringify({ spaces: [{ name: "finance", label: "confidential" }] }), { status: 200 });
      if (u.startsWith("/v1/search")) return new Response(JSON.stringify({ trace: "tr_s", release: "rel_x", refused: 0, results: [] }), { status: 200 });
      if (u.startsWith("/v1/ask")) {
        calls.push(`${u} ${init?.body ?? ""}`);
        const first = calls.length === 1;
        return new Response(JSON.stringify(first ? { ...answer, suggest: { space: "finance", agent: "close-checklist", terms: ["AP"] } } : answer), { status: 200 });
      }
      return new Response(JSON.stringify({ items: [] }), { status: 200 });
    };
    const { container } = render(<App fetcher={fetcher} />);
    await act(async () => undefined);
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "Is the AP close done?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    expect(container.textContent).toContain("finance has an agent for this: close-checklist.");
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask close-checklist in finance" }));
    });
    expect(calls[1]).toBe(`/v1/ask?space=finance {"question":"Is the AP close done?","clarify":true,"agent":"close-checklist","language":"en"}`);
  });
});
