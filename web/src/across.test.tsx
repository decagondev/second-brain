import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { CatchUp, ServiceCard, Trail, WaitsCard } from "./Across";
import { CitationChip } from "./Answer";
import type { Fetch } from "./api";

afterEach(cleanup);

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

// SB-4504 to SB-4507
describe("work across departments", () => {
  it("lists what a space waits on and what waits on it, counts the rest, and clears from either side", async () => {
    const calls: { url: string; method?: string }[] = [];
    let cleared = false;
    const fetcher: Fetch = async (u, init) => {
      calls.push({ url: u, method: init?.method });
      if (init?.method === "DELETE") {
        cleared = true;
        return json({ cleared: true });
      }
      return json({
        space: "it",
        waiting: cleared ? [] : [{ kind: "request", id: "qs_1", title: "Van tracker access", space: "it", on: "fleet", reason: "fleet must name the vans", until: "2026-12-01", can_clear: true }],
        waited_on: [{ kind: "request", id: "qs_2", title: "Request: Access", space: "returns", on: "it", reason: "a login", until: "2026-11-20", overdue: true }],
        hidden: 2,
      });
    };
    render(<WaitsCard space="it" fetcher={fetcher} />);
    await act(async () => undefined);
    expect(screen.getByText("Van tracker access")).toBeTruthy();
    expect(screen.getByText("Overdue")).toBeTruthy();
    expect(screen.getByText("2 more you can't see.")).toBeTruthy();
    expect(screen.getAllByRole("button", { name: /Clear the wait/ })).toHaveLength(1); // only what the reader may clear
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Clear the wait: Van tracker access" }));
    });
    expect(calls.some((c) => c.method === "DELETE" && c.url === "/v1/spaces/it/waits?kind=request&id=qs_1")).toBe(true);
    expect(screen.queryByText("Van tracker access")).toBeNull();
  });

  it("shows service levels only to stewards, a small row masked", async () => {
    const steward: Fetch = async () =>
      json({ space: "it", weeks: [], rows: [{ category: "Access", week: "2026-W48", count: 5, first_reply_hours: 1, close_hours: 60, late_share: 0.5 }, { category: "question", week: "2026-W48", fewer_than_5: true }], teams: [{ space: "ap", fewer_than_5: true }] });
    render(<ServiceCard space="it" fetcher={steward} />);
    await act(async () => undefined);
    expect(screen.getByText("50%")).toBeTruthy();
    expect(screen.getByText("fewer than 5")).toBeTruthy();
    expect(screen.getByLabelText("Teams").textContent).toBe("ap: fewer than 5");
    cleanup();
    render(<ServiceCard space="it" fetcher={async () => json({ error: { code: "EPERM", message: "no" } }, 403)} />);
    await act(async () => undefined);
    expect(screen.queryByLabelText("Service")).toBeNull();
  });

  it("catches up since the day asked, and shows the trail with what replaced what", async () => {
    const calls: { url: string; body?: unknown }[] = [];
    const fetcher: Fetch = async (u, init) => {
      calls.push({ url: u, body: init?.body ? JSON.parse(String(init.body)) : undefined });
      if (u.startsWith("/v1/catch-up"))
        return json({ trace: "tr_1", answer: { claims: [{ text: "The dock opens at six.", cites: [{ cite: "docs:dock@2", title: "Dock hours", span: [0, 1], flags: [] }] }] }, catch_up: { since: "2026-11-20", documents: 3, waiting: 1 } });
      return json({ space: "ops", topic: "dock", decisions: [{ cite: "docs:dock@1", title: "Dock hours", at: "2026-01-02T00:00:00Z", replaced_by: { cite: "docs:dock2@1", title: "Dock hours from December" } }], hidden: 1 });
    };
    render(<CatchUp space="ops" branch since="2026-11-20" fetcher={fetcher} />);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Catch me up" }));
    });
    expect(calls[0]).toEqual({ url: "/v1/catch-up?space=ops&branch=true", body: { since: "2026-11-20" } });
    expect(screen.getByText("3 documents changed since 2026-11-20.", { exact: false })).toBeTruthy();
    cleanup();
    render(<Trail space="ops" fetcher={fetcher} />);
    fireEvent.change(screen.getByLabelText("Decisions about"), { target: { value: "dock" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Show the trail" }));
    });
    expect(screen.getByText("Replaced by Dock hours from December")).toBeTruthy();
    expect(screen.getByText("1 decision is not shown because you can't open it.")).toBeTruthy();
    cleanup();
    render(<CitationChip trace="tr_1" cite={{ cite: "docs:dock@1", title: "Dock hours", span: [0, 0], flags: [], replaced_by: { cite: "docs:dock2@1", title: "Dock hours from December" } }} />);
    expect(screen.getByText("Replaced by Dock hours from December")).toBeTruthy();
  });
});
