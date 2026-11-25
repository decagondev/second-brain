import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { AgentValue, PauseHere } from "./Accountable";
import type { Fetch } from "./api";

afterEach(cleanup);

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

const report = {
  weeks: 12,
  from: "2026-09-02",
  rows: [{ agent: "case-brief", space: "support", drafts: 4, published: 2, published_edited: 1, discarded: 1, rounds: 2, median_hours: 3.5, rated: 2, helpful_rate: 0.5, cost_usd: 0.4, cost_per_published_usd: 0.1333 }],
  spend: [{ agent: "case-brief", month: "2026-11", spent_usd: 20, monthly_usd: 20, capped: true }],
};

// SB-4701 to SB-4703
describe("every agent accountable", () => {
  it("shows an owner what their agents' drafts came to, with Pause here and the cap", async () => {
    const calls: { u: string; body?: string }[] = [];
    const fetcher: Fetch = async (u, init) => {
      calls.push({ u, body: init?.body as string | undefined });
      return u === "/v1/agents/value" ? json(report) : json({ paused: true });
    };
    await act(async () => {
      render(<AgentValue fetcher={fetcher} />);
    });
    expect(screen.getByText("What your agents' drafts came to")).toBeTruthy();
    expect(screen.getByText("50% (2)")).toBeTruthy();
    expect(screen.getByText("$0.13")).toBeTruthy();
    expect(screen.getByText("$20.00: at its cap until next month")).toBeTruthy();
    fireEvent.click(screen.getByText("Pause here"));
    fireEvent.change(screen.getByLabelText("Why pause case-brief here?"), { target: { value: "wrong account" } });
    await act(async () => {
      fireEvent.click(screen.getByText("Pause"));
    });
    expect(calls[1]).toEqual({ u: "/v1/spaces/support/agents/case-brief/pause", body: JSON.stringify({ reason: "wrong account" }) });
    expect(screen.getByRole("status").textContent).toContain("case-brief is paused in support");
    expect(screen.getByText("Resume")).toBeTruthy();
  });

  it("shows nothing to someone who owns no agent", async () => {
    const fetcher: Fetch = async () => json({ weeks: 12, from: "2026-09-02", rows: [], spend: [] });
    let c: HTMLElement;
    await act(async () => {
      c = render(<AgentValue fetcher={fetcher} />).container;
    });
    expect(c!.textContent).toBe("");
  });

  it("resumes, and says why when a pause is refused", async () => {
    const fetcher: Fetch = async (u) => (u.endsWith("/resume") ? json({ paused: false }) : json({ error: { message: "Only the space's stewards or the agent's owner can pause it here." } }, 403));
    render(<PauseHere space="support" agent="case-brief" paused fetcher={fetcher} />);
    await act(async () => {
      fireEvent.click(screen.getByText("Resume"));
    });
    expect(screen.getByRole("status").textContent).toContain("case-brief runs in support again.");
    fireEvent.click(screen.getByText("Pause here"));
    fireEvent.change(screen.getByLabelText("Why pause case-brief here?"), { target: { value: "x" } });
    await act(async () => {
      fireEvent.click(screen.getByText("Pause"));
    });
    expect(screen.getByRole("status").textContent).toContain("Only the space's stewards");
  });
});
