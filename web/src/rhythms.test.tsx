import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { ProjectsCard, RhythmsCard } from "./Rhythms";
import type { Fetch } from "./api";

afterEach(cleanup);

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

// SB-4801, SB-4803, SB-4804
describe("ways of working", () => {
  it("shows a proposed rhythm, approves it only on Approve, and starts and closes a project", async () => {
    const calls: { url: string; method?: string; body?: unknown }[] = [];
    let state = "proposed";
    let projects: unknown[] = [];
    const fetcher: Fetch = async (u, init) => {
      calls.push({ url: u, method: init?.method, body: init?.body ? JSON.parse(String(init.body)) : undefined });
      if (u.endsWith("/approve")) {
        state = "running";
        return json({ id: "rh_1", state });
      }
      if (u === "/v1/spaces/support/rhythms")
        return json({ items: [{ id: "rh_1", rhythm: "onboarding", space: "support", subject: "usr_pat", subject_kind: "person", state,
          steps: [{ day: 0, kind: "draft", ask: "Where to start", approver: "subject", state: "waiting", due: "2026-11-25" }] }] });
      if (u === "/v1/spaces/support/projects" && init?.method === "POST") {
        projects = [{ id: "pj_1", name: "Payments revamp", lead: true, state: "open", decisions: [], hidden: 0, collection: { id: "co_1" } }];
        return json(projects[0]);
      }
      if (u === "/v1/spaces/support/projects") return json({ items: projects });
      if (u.endsWith("/close")) {
        projects = [{ id: "pj_1", name: "Payments revamp", lead: true, state: "closed", decisions: [], hidden: 0, collection: { id: "co_1", closed: "2026-11-25" } }];
        return json(projects[0]);
      }
      return json({});
    };
    render(<><RhythmsCard space="support" fetcher={fetcher} /><ProjectsCard space="support" fetcher={fetcher} /></>);
    await act(async () => undefined);
    expect(screen.getByText("onboarding")).toBeTruthy();
    expect(screen.getByText(/waiting for its plan to be approved/)).toBeTruthy();
    expect(calls.some((c) => c.method === "POST")).toBe(false);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Approve" }));
    });
    expect(calls.some((c) => c.url === "/v1/rhythms/rh_1/approve")).toBe(true);
    expect(screen.getByText(/next: Where to start, 2026-11-25/)).toBeTruthy();

    fireEvent.change(screen.getByLabelText("Project name"), { target: { value: "Payments revamp" } });
    fireEvent.change(screen.getByLabelText("Members (emails)"), { target: { value: "kim@acme.example, lee@acme.example" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Start a project" }));
    });
    const started = calls.find((c) => c.url === "/v1/spaces/support/projects" && c.method === "POST")!;
    expect((started.body as { members: string[] }).members).toEqual(["kim@acme.example", "lee@acme.example"]);
    expect(screen.getByText("Payments revamp")).toBeTruthy();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Close project" }));
    });
    expect(calls.some((c) => c.url === "/v1/projects/pj_1/close")).toBe(true);
    expect(screen.getByText(/closed/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Close project" })).toBeNull();
  });
});
