import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { Goals, PoliciesToRead, StewardPolicies, TeamUpdates } from "./Company";
import type { Fetch } from "./api";

afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const json = (x: unknown) => new Response(JSON.stringify(x), { status: 200 });

// SB-4101 to SB-4103 in the web UI.
describe("run the company", () => {
  it("shows goals nested by quarter and team updates", async () => {
    const fetcher: Fetch = async (u) =>
      u.endsWith("/goals")
        ? json({ quarters: [{ quarter: "2027-Q1", goals: [{ cite: "docs:a@1", title: "Faster finance", space: "finance", status: "on track", goals: [{ cite: "docs:b@1", title: "Close faster", space: "ap", status: "at risk" }] }] }], hidden: 1 })
        : json({ updates: [{ space: "ap", title: "AP update", cite: "docs:notes/ap/updates/qs_1.md@1", at: "2027-03-01T09:00:00Z" }], hidden: 0 });
    const { container } = render(
      <>
        <Goals space="finance" fetcher={fetcher} />
        <TeamUpdates space="finance" fetcher={fetcher} />
      </>,
    );
    await act(async () => {});
    const g = screen.getByRole("region", { name: "Goals" });
    expect(g.textContent).toContain("2027-Q1");
    expect(g.querySelector("ul ul")?.textContent).toContain("Close faster");
    expect(g.textContent).toContain("1 goal is not shown because you can't open it.");
    expect(screen.getByRole("region", { name: "Team updates" }).textContent).toContain("AP update");
    expect(await violations(container)).toEqual([]);
  });

  it("records I've read it for the version shown", async () => {
    const calls: string[] = [];
    let read = false;
    const fetcher: Fetch = async (u, init) => {
      calls.push(`${init?.method ?? "GET"} ${u} ${init?.body ?? ""}`.trim());
      if (init?.method === "POST") {
        read = true;
        return json({ read: true });
      }
      return json({ policies: [{ space: "finance", doc: "docs:leave.md", cite: "docs:leave.md@7", title: "Leave policy", version: 7, read }] });
    };
    const { container } = render(<PoliciesToRead space="finance" fetcher={fetcher} />);
    await act(async () => {});
    expect(await violations(container)).toEqual([]);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "I've read Leave policy" }));
    });
    expect(calls).toContain(`POST /v1/spaces/finance/policies/read {"doc":"docs:leave.md","version":7}`);
    expect(screen.getByRole("status").textContent).toBe("Recorded: you've read Leave policy.");
    expect(screen.queryByRole("button", { name: "I've read Leave policy" })).toBeNull();
  });

  it("shows stewards who hasn't read a policy, and marks one", async () => {
    const calls: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      calls.push(`${init?.method ?? "GET"} ${u} ${init?.body ?? ""}`.trim());
      if (init?.method === "POST") return json({ doc: "docs:x.md", title: "Expenses policy", version: 1 });
      return json({ policies: [{ space: "finance", doc: "docs:leave.md", cite: "docs:leave.md@7", title: "Leave policy", version: 7, not_read: ["lee@acme.example"], read: 3, cant_open: 1 }] });
    };
    const { container } = render(<StewardPolicies base="/v1/spaces/finance" fetcher={fetcher} />);
    await act(async () => {});
    expect(container.textContent).toContain("Not read yet: lee@acme.example");
    expect(container.textContent).toContain("3 have read it");
    expect(container.textContent).toContain("1 member can't open it");
    fireEvent.change(screen.getByLabelText("Document (KIND:ID)"), { target: { value: "docs:x.md" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Mark as a policy" }));
    });
    expect(calls).toContain(`POST /v1/spaces/finance/policies {"doc":"docs:x.md"}`);
    expect(await violations(container)).toEqual([]);
  });
});
